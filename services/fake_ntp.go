package services

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"b0gus/databases"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"b0gus/configs"
)

/*
	NTP-RFC: www.rfc-editor.org/rfc/rfc958.html
	this is NTP querying packet
	--------------------------------------------------------------------------------------------
	Field Name              Request    Reply                                   bytes
	--------------------------------------------------------------------------------------------
	LI                      0 or 3     0
	VN                      1-4        copied from request
	Mode                    3          4                                         0

	Stratum                 ignore     1                                         1
	Poll                    ignore     copied from request                       2
	Precision               ignore     -log2 server significant bits             3

	Root Delay              ignore     0                                         4
	Root Dispersion         ignore     0                                         8
	Reference Identifier    ignore     source ident                              12
	Reference Timestamp     ignore     time of last radio update				 16
	Originate Timestamp     ignore     copied from transmit timestamp			 24
	Receive Timestamp       ignore     time of day							     32
	Transmit Timestamp      (see text) time of day                               40
	--------------------------------------------------------------------------------------------
	                                                                    total:   48
	--------------------------------------------------------------------------------------------
	For receive timestamp, if no request has ever arrived from the client the
    value is zero
	For transmit timestamp, server need to specify the local time
	at which the reply departed for the client host

	the round-trip delay d and clock offset c is:
         d = (t4 - t1) - (t3 - t2)  and  c = (t2 - t1 + t3 - t4)/2 .

	while the control packet is
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|LI | VN  |Mode |R|M| OpCode      | Sequence                  |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Association ID                                              |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Status                                                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Offset                                                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Count                                                       |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Data (variable length, max 468 octets)                    ...
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Padding (0-3 octets, zero)                                ...
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	| Authenticator (optional, 20 or 24 octets)                 ...
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

func validFormat(req []byte) bool {
	if len(req) == 0 {
		return false
	}
	const (
		/*
			LEAP ID
			A leap-second is occasionally added or subtracted
			from Standard Time, which is based on atomic clocks,
			to maintain agreement with Earth rotation.
				00      no warning
				01      +1 second (following minute has 61 seconds)
				10      -1 second (following minute has 59 seconds)
				11      reserved for future use
		*/
		LiNoWarning = 0
		LiAdd       = 1
		LiSub       = 2
		LiReserved  = 3
		// VERSION NUMBER

		VnFirst = 1
		VnLast  = 4
		// MODE

		ModeQuery = 3
		ModeCtrl  = 6
	)
	var (
		/* 00_011_011 */
		l = (req[0] >> 6) & 0b11
		v = (req[0] >> 3) & 0b111
	)
	if !((l == LiNoWarning) || (l == LiReserved)) {
		return false
	}
	return (VnFirst <= v && v <= VnLast) && (req[0]&0b111 == ModeQuery || req[0]&0b111 == ModeCtrl)
}

const From1900To1970 = 2208988800

// unix time: the number of seconds elapsed since January 1, 1970 UTC
// npt time: the number of seconds elapsed since January 1, 1900 UTC
// 1900~1970
func unix2ntp(u int64) int64 {
	return u + From1900To1970
}

// int2bytes converts int number to four bytes in big endian.
func int2bytes(i int64) []byte {
	var b = make([]byte, 4)
	h1 := i >> 24
	h2 := (i >> 16) - (h1 << 8)
	h3 := (i >> 8) - (h1 << 16) - (h2 << 8)
	b[0] = byte(h1)
	b[1] = byte(h2)
	b[2] = byte(h3)
	b[3] = byte(i)
	return b
}

// TODO: 1. A little bit weird. And need to add time resolution hook
//		 2. what kind of information do we have to record?

func generate(req []byte) []byte {
	currTime := time.Now()
	var second = unix2ntp(currTime.Unix())
	var fraction = unix2ntp(int64(currTime.Nanosecond()))
	var res = make([]byte, 48)
	var vn = req[0] & 0b00_111_000
	res[0] = vn + 4

	res[1] = 1
	res[2] = req[2]
	res[3] = 0xEC

	res[12] = 0x4E
	res[13] = 0x49
	res[14] = 0x43
	res[15] = 0x54

	copy(res[16:20], int2bytes(second)[0:])
	copy(res[24:32], req[40:48])

	copy(res[32:36], int2bytes(second)[0:])
	copy(res[36:40], int2bytes(fraction)[0:])
	copy(res[40:48], res[32:40])
	return res
}

func NTPServe(req []byte) ([]byte, error) {
	if !validFormat(req) {
		return nil, errors.New("invalid ntp format")
	}
	switch req[0] & 0b111 {
	case 3:
		res := generate(req)
		return res, nil
	case 6:
		/*
			TODO: record mode according to RFC 9327
			1: read status
			2: read variable
			3: write variable
			6: Set Trap Address/Port
			7: Trap Response
			9: Save Configuration
			10: Read MRU
			11: Read ordered list
			12: Request Nonce
			31: Unset Trap
			13-31: Reserved
			6/7 ~ [a mechanism for positively notifying events that happened in current NTP server]
		*/
		return nil, errors.New("unsupported NTP operation")
	default:
		return nil, errors.New("invalid ntp mode")
	}
}

// InvokeForTCPtask here will do nothing due to ntp requires udp
func (n *NTPServConf) InvokeForTCPtask(net.Conn)                    {}
func (n *NTPServConf) InvokeForICMPtask(net.Addr, []byte)           {}
func (n *NTPServConf) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (n *NTPServConf) InvokeForUDPtask(remoteIP net.Addr, dataBuf []byte) []byte {
	payload := configs.GetLocalizedMsg(
		"services.NTPReceivePayloadFromRemoteInfo",
		map[string]any{"IP": remoteIP, "Len": len(dataBuf)},
	)
	configs.Logger.Info(payload)
	resp, err := NTPServe(dataBuf)
	if err != nil {
		// invalid ntp packet
		return dataBuf
	}
	return resp
}

type NTPServConf struct {
	/* database handler for writing data */
	DbFd        databases.DBhandler
	ConfOptions *atomic.Pointer[configs.LocalConfig]
}

// Run executes NTP services
func (n *NTPServConf) Run(
	confObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db databases.DBhandler,
	args ...any,
) {
	defer func() {
		payload := configs.GetLocalizedMsg(
			"services.NTPQuitInfo",
			nil,
		)
		configs.Logger.Info(payload)
	}()
	if len(args) > 1 || args[0] != nil {
		return
	}
	if confObj == nil {
		configs.Logger.Error(configs.GetLocalizedMsg("NTPNullConfErr", nil))
		return
	}
	_, ok := confObj.Load().SelectTerm(configs.NTPEnum).(configs.NTPconfig)
	if !ok {
		return
	}
	var clientAux = ReentrantNetType{}
	n.DbFd = db
	clientAux.Init(configs.NTPEnum, n)
	go clientAux.EventMonitor(confObj, scc)
	clientAux.AlterNetFd(UDPEnum, confObj)
}
