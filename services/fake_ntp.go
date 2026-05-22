package services

/// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/05/22 星期五 10:13:05
// refer:
//    https://github.com/ntp-project/ntp/blob/8a37f9b66d374b164531f0189caba4cbfd68bb61/ntpd/ntp_control.c
//    https://github.com/ntpsec/ntpsec/blob/master/ntpd/ntp_control.c

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"b0gus/configs"
	"b0gus/databases"
)

type (
	ntpCtrlOpcode         int
	ntpCtrlErrcode        uint8
	ntpCtrlRestrictAccess uint16
)

const (
	ntpCtrlOpUnspecified ntpCtrlOpcode = iota
	ntpCtrlOpReadStatus
	ntpCtrlOpReadVariable
	ntpCtrlOpWriteVariable
	ntpCtrlOpReadClock
	ntpCtrlOpWriteClock
	ntpCtrlOpSetTrap
	ntpCtrlOpResponseTrap
	ntpCtrlOpConfigure
	ntpCtrlOpSaveConfig
	ntpCtrlOpReadMRU
	ntpCtrlOpReadOrderedListWithAuth
	ntpCtrlOpRequestNonce
	ntpCtrlOpUnsetTrap = iota + 18
	ntpCtrlOpUnk
)

// Restrict (Access) flags
const (
	ntpCtrlRestrictIgn         ntpCtrlRestrictAccess = 0x0001
	ntpCtrlRestrictWontServ    ntpCtrlRestrictAccess = 0x0002
	ntpCtrlRestrictWontTrust   ntpCtrlRestrictAccess = 0x0004
	ntpCtrlRestrictVersion     ntpCtrlRestrictAccess = 0x0008
	ntpCtrlRestrictNoPeer      ntpCtrlRestrictAccess = 0x0010
	ntpCtrlRestrictNoEphemeral ntpCtrlRestrictAccess = 0x0020
	ntpCtrlRestrictLimited     ntpCtrlRestrictAccess = 0x0040
	ntpCtrlRestrictNoQuery     ntpCtrlRestrictAccess = 0x0080
	ntpCtrlRestrictNoModify    ntpCtrlRestrictAccess = 0x0100
	ntpCtrlRestrictNoTrap      ntpCtrlRestrictAccess = 0x0200
	ntpCtrlRestrictLowTrap     ntpCtrlRestrictAccess = 0x0400

	ntpCtrlRestrictKissOfDeath      ntpCtrlRestrictAccess = 0x0800
	ntpCtrlRestrictEnableMSSNTPauth ntpCtrlRestrictAccess = 0x1000
	ntpCtrlRestrictFlake            ntpCtrlRestrictAccess = 0x2000
	ntpCtrlRestrictNoMRUList        ntpCtrlRestrictAccess = 0x4000
	ntpCtrlRestrictRespFuzz         ntpCtrlRestrictAccess = 0x8000
	ntpCtrlRestrictUnused           ntpCtrlRestrictAccess = 0

	ntpCtrlRestrictAllFlags = ntpCtrlRestrictIgn | ntpCtrlRestrictWontServ |
		ntpCtrlRestrictWontTrust | ntpCtrlRestrictVersion | ntpCtrlRestrictNoPeer |
		ntpCtrlRestrictNoEphemeral | ntpCtrlRestrictLimited | ntpCtrlRestrictNoQuery |
		ntpCtrlRestrictNoModify | ntpCtrlRestrictNoTrap | ntpCtrlRestrictLowTrap |
		ntpCtrlRestrictKissOfDeath | ntpCtrlRestrictEnableMSSNTPauth | ntpCtrlRestrictFlake |
		ntpCtrlRestrictNoMRUList | ntpCtrlRestrictRespFuzz | ntpCtrlRestrictUnused
)

var restrictArr = []ntpCtrlRestrictAccess{
	ntpCtrlRestrictIgn, ntpCtrlRestrictFlake, ntpCtrlRestrictKissOfDeath,
	ntpCtrlRestrictLimited, ntpCtrlRestrictNoPeer, ntpCtrlRestrictNoMRUList,
	ntpCtrlRestrictNoTrap, ntpCtrlRestrictWontTrust, ntpCtrlRestrictWontServ,
	ntpCtrlRestrictAllFlags,
}

// ntpCtrl Error code
const (
	ntpCtrlErrUnspecified ntpCtrlErrcode = iota
	ntpCtrlErrPerm
	ntpCtrlErrBadFmt
	ntpCtrlErrBadOp
	ntpCtrlErrBadAssociatedID
	ntpCtrlErrUnkVar
	ntpCtrlErrBadVal
	ntpCtrlErrRestrict
)

const (
	ctrlResp = 0x80
	ctrlErr  = 0x40
	ctrlMore = 0x20
)

type ntpClockType int

const (
	NTPtsUnspecified ntpClockType = iota
	NTPtsAtomClock
	NTPtsLFRadio
	NTPtsHFRadio
	NTPtsUHFRadio
	NTPtsLocal
	NTPtsNTP
	NTPtsUPDTime
	NTPtsWristWatch
	NTPtsTelephone
)

var clockTypes = []ntpClockType{
	NTPtsUnspecified,
	NTPtsAtomClock,
	NTPtsLFRadio,
	NTPtsHFRadio,
	NTPtsUHFRadio,
	NTPtsLocal,
	NTPtsNTP,
	NTPtsUPDTime,
	NTPtsWristWatch,
	NTPtsTelephone,
}

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

func generateNTPresponse(req []byte) []byte {
	if len(req) < 48 {
		return nil
	}
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

type ntpCtrlPacketTag struct {
	data [504]byte // data contains 480 + 24 bytes
	u    [126]uint32
}

// ntpHeader is used for unmarshalling the first 12 bytes
type ntpHeader struct {
	LiVnMode, RMEOpcode uint8 // leap version mode response more error opcode
	Seq, Status, Ofs    uint16
	associationID, Cnt  uint16
	PaddedData          *ntpCtrlPacketTag
}

func (n ntpHeader) GenerateErrResp(errCode ntpCtrlErrcode) []byte {
	n.LiVnMode = 0
	n.RMEOpcode = uint8(errCode) | ctrlErr | ctrlResp
	n.Status = (uint16(errCode) << 8) & 0xFF00
	n.Seq = 0
	n.associationID = 0
	n.Cnt = 0
	n.Ofs = 0
	return n.Serialize()
}

// Serialize will turn current ntpPacket into 12 bytes' array as the result
func (n ntpHeader) Serialize() []byte {
	var res = make([]byte, 12)
	res[0] = n.LiVnMode
	res[1] = n.RMEOpcode
	res[2] = byte((n.Seq & 0xFF00) >> 8)
	res[3] = byte(n.Seq & 0xFF)
	res[4] = byte((n.Status & 0xFF00) >> 8)
	res[5] = byte(n.Status & 0xFF)
	res[6] = byte((n.associationID & 0xFF00) >> 8)
	res[7] = byte(n.associationID & 0xFF)
	res[8] = byte((n.Ofs & 0xFF00) >> 8)
	res[9] = byte(n.Ofs & 0xFF)
	res[10] = byte((n.Cnt & 0xFF00) >> 8)
	res[11] = byte(n.Cnt & 0xFF)
	return res
}

type ntpCtrlOpInspector struct {
	NeedAuth bool
	fn       func(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte
}

var ntpCtrlOperationLUT = map[ntpCtrlOpcode]ntpCtrlOpInspector{
	ntpCtrlOpUnspecified:             {false, handleNTPCtrlUnspecified},
	ntpCtrlOpReadStatus:              {false, handleNTPCtrlReadStatus},
	ntpCtrlOpReadVariable:            {false, handleNTPCtrlReadVariable},
	ntpCtrlOpWriteVariable:           {true, handleNTPCtrlWriteVariable},
	ntpCtrlOpReadClock:               {false, handleNTPCtrlReadClock},
	ntpCtrlOpWriteClock:              {true, handleNTPCtrlWriteClock},
	ntpCtrlOpSetTrap:                 {true, handleNTPCtrlSetTrap},
	ntpCtrlOpResponseTrap:            {false, nil},
	ntpCtrlOpConfigure:               {true, handleNTPCtrlConfigure},
	ntpCtrlOpSaveConfig:              {true, handleNTPCtrlSaveConfig},
	ntpCtrlOpReadMRU:                 {false, handleNTPCtrlReadMRU},
	ntpCtrlOpReadOrderedListWithAuth: {true, handleNTPCtrlReadOrderedListWithAuth},
	ntpCtrlOpRequestNonce:            {false, handleNTPCtrlRequestNonce},
	ntpCtrlOpUnsetTrap:               {true, handleNTPCtrlUnsetTrap},
	ntpCtrlOpUnk:                     {false, nil},
}

// serveNTP will serve as a fake NTP server for the incoming request streams
// Notice that NTP is a big endian protocol.
func serveNTP(remoteIP net.Addr, req []byte) []byte {
	var data ntpHeader
	if !validLiVnMode(req) { // invalid ntp format
		return data.GenerateErrResp(ntpCtrlErrBadFmt)
	}
	castedAddr, ok := remoteIP.(*net.UDPAddr)
	if !ok { // unable to cast into net.UDPAddr
		return data.GenerateErrResp(ntpCtrlErrBadOp)
	}
	if (req[0] & 0b111) == 3 { // mode: query current time
		return generateNTPresponse(req)
	} else if (req[0] & 0b10) == 6 { // mode: mode control
		return handleNTPCtrlMsg(castedAddr, req)
	}
	return data.GenerateErrResp(ntpCtrlErrBadVal)
}

// validLiVnMode will check for one given NTP packet if setting up the correct versionNumber
// and the querying mode.
func validLiVnMode(req []byte) bool {
	// at least more than one byte
	if len(req) < 1 {
		return false
	}
	/*
		Leap ID
		A leap-second is occasionally added or subtracted
		from Standard Time, which is based on atomic clocks,
		to maintain agreement with Earth rotation.
			00      no warning
			01      +1 second (following minute has 61 seconds)
			10      -1 second (following minute has 59 seconds)
			11      reserved for future use
	*/
	const (
		LiNoWarning = 0
		LiReserved  = 3
	)
	// versionNumber
	const (
		VnFirst = 1
		VnLast  = 4
	)
	// mode to identify different operations
	const (
		ModeQuery = 3
		ModeCtrl  = 6
	)
	/* that is 00_011_011 */
	l := (req[0] >> 6) & 0b11
	v := (req[0] >> 3) & 0b111
	if (l != LiNoWarning) && (l != LiReserved) {
		return false
	}
	return (VnFirst <= v && v <= VnLast) && (req[0]&0b111 == ModeQuery || req[0]&0b111 == ModeCtrl)
}

func handleNTPCtrlMsg(addr *net.UDPAddr, req []byte) []byte {
	var data ntpHeader
	/*
		offsetof(struct ntp_control, u) // strut ctl_pkt_u u defined in struct ntp_control;
		which requires cut-off at the 12th bytes.
	*/
	if len(req) < 12 {
		// errors.New("short ntp control packet format")
		return data.GenerateErrResp(ntpCtrlErrBadFmt)
	}
	err := binary.Read(bytes.NewReader(req[:12]), binary.BigEndian, &data)
	if err != nil {
		return data.GenerateErrResp(ntpCtrlErrBadVal) // , err
	}

	if (data.Ofs != 0) || ((req[1] & (ctrlResp | ctrlErr | ctrlMore)) != 0) {
		// if this packet is a response or a fragment, ignore it.
		// [TODO]: statistic for ctrl modes just like ntpd/ntp_control.c
		// , errors.New("invalid format in ntp control packet")
		return data.GenerateErrResp(ntpCtrlErrBadFmt)
	}

	macLen := len(req) - int((data.Cnt+19)&(^uint16(7))) // round to 8 bytes align
	if len(req)&3 == 0 && len(req) > macLen && 4 <= macLen && macLen <= 24 {
		// aligned at 4 bytes
		var sb strings.Builder
		sb.WriteString("\twants to authenticate with keyID<")
		sb.WriteString(hex.EncodeToString(req[macLen : macLen+4]))
		sb.WriteString(">, mac length is ")
		sb.WriteString(strconv.Itoa(macLen))
		configs.Logger().Info(sb.String())
		// yet to return
		// nil, errors.New("authentication failed")
	}

	for opcode, inpOp := range ntpCtrlOperationLUT {
		if opcode != ntpCtrlOpcode(data.RMEOpcode) {
			continue
		}
		if inpOp.fn == nil {
			// internal error: do not register ntp ctrl operation
			return data.GenerateErrResp(ntpCtrlErrRestrict)
		}
		switch {
		case inpOp.NeedAuth && addr.IP.IsMulticast():
			inpOp.fn(data, req[12:], ntpCtrlRestrictIgn)
		default:
			// actually, IPv4/IPv6 use match->rflags, but do not know which flag
			// will be masked as one in the end
			inpOp.fn(data, req[12:], restrictArr[rand.IntN(len(restrictArr))])
		}
		return data.GenerateErrResp(ntpCtrlErrPerm)
	}
	return data.GenerateErrResp(ntpCtrlErrBadOp)
}

func SetNtpKV(key string, val any, opts ...any) string {
	var sb strings.Builder
	sb.WriteString(key)
	sb.WriteRune('=')
	switch v := val.(type) {
	case string:
		sb.WriteRune('"')
		sb.WriteString(v)
		sb.WriteRune('"')
	case int:
		sb.WriteString(strconv.Itoa(v))
	}
	_ = opts
	return sb.String()
}

// SetNTPkvViaAnyArr will help to set "key1=val1,key2=val2" in the further datagram encapsulation
//nolint:unused
func SetNTPkvViaAnyArr(key []string, val []any) string {
	if len(key) != len(val) {
		return ""
	}
	var sb strings.Builder
	for i := range len(key) {
		sb.WriteString(SetNtpKV(key[i], val[i], nil))
		if i != len(key)-1 {
			sb.WriteRune(',')
		}
	}
	return sb.String()
}

/* [TODO]: handler should record by Time Series database
   which kind of information do we have to record?
*/

// handleNTPCtrlUnspecified
func handleNTPCtrlUnspecified(head ntpHeader, req []byte, mask ntpCtrlRestrictAccess) []byte {
	head.Status = uint16(clockTypes[rand.IntN(len(clockTypes))])
	currChoice := rand.IntN(2)
	if currChoice > 0 {
		return head.Serialize()
	}
	_ = req
	_ = mask
	return head.GenerateErrResp(ntpCtrlErrBadAssociatedID)
}

// handleNTPCtrlReadStatus
func handleNTPCtrlReadStatus(head ntpHeader, req []byte, mask ntpCtrlRestrictAccess) []byte {
	// case 1: query for single peer
	if head.associationID > 0 {
		// we can fake as usual.
		if mask&ntpCtrlRestrictAllFlags == 0 {
			return head.GenerateErrResp(ntpCtrlErrRestrict)
		}
		return head.Serialize()
	}
	// case 2: query for the whole system
	req = make([]byte, 0)
	return append(head.GenerateErrResp(ntpCtrlErrUnspecified), req...)
}

/* [TODO]: implement these functions */

// handleNTPCtrlReadVariable will generate random response for every response
func handleNTPCtrlReadVariable(head ntpHeader, req []byte, mask ntpCtrlRestrictAccess) []byte {

	currChoice := rand.IntN(2)
	if currChoice > 0 {
		return head.Serialize()
	}
	_ = req
	_ = mask
	return head.GenerateErrResp(ntpCtrlErrUnkVar)
}

// handleNTPCtrlWriteVariable will always fail and record the action to the log/database
func handleNTPCtrlWriteVariable(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlReadClock
func handleNTPCtrlReadClock(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlWriteClock
func handleNTPCtrlWriteClock(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlSetTrap
func handleNTPCtrlSetTrap(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlConfigure
func handleNTPCtrlConfigure(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlSaveConfig
func handleNTPCtrlSaveConfig(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlReadMRU
func handleNTPCtrlReadMRU(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlReadOrderedListWithAuth
func handleNTPCtrlReadOrderedListWithAuth(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlRequestNonce
func handleNTPCtrlRequestNonce(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

// handleNTPCtrlUnsetTrap
func handleNTPCtrlUnsetTrap(ntpHeader, []byte, ntpCtrlRestrictAccess) []byte {
	return nil
}

//type ntpPeerInfo struct {
//	addr     [28]byte
//	hostname *string
//
//	version   uint8
//	leap      uint8
//	peerMode  uint8
//	stratum   uint8
//	peerPoll  uint8
//	precision int8
//
//	rootDelay float64
//	rootDisp  float64
//	refID     uint32
//	refTime   uint64
//
//	associatedID uint32
//
//	offset float64
//	delay  float64
//	jitter float64
//	disp   float64
//
//	status uint8
//	reach  uint8
//
//	sent     uint64
//	received uint64
//	badauth  uint64
//
//	rec uint64
//	xmt uint64
//}

// InvokeForTCPtask here will do nothing due to ntp requires udp
func (n *NTPServConf) InvokeForTCPtask(net.Conn)                    {}
func (n *NTPServConf) InvokeForICMPtask(net.Addr, []byte)           {}
func (n *NTPServConf) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (n *NTPServConf) InvokeForUDPtask(remoteIP net.Addr, dataBuf []byte) []byte {
	payload := configs.GetLocalizedMsg(
		"services.NTPReceivePayloadFromRemoteInfo",
		map[string]any{"IP": remoteIP, "Len": len(dataBuf)},
	)
	configs.Logger().Info(payload)
	return serveNTP(remoteIP, dataBuf) // replay
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
		configs.Logger().Info(payload)
	}()
	if len(args) > 1 || args[0] != nil {
		return
	}
	if confObj == nil {
		configs.Logger().Error(configs.GetLocalizedMsg("NTPNullConfErr", nil))
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
