// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package services

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	b0gus_config "b0gus/configs"
	b0gus_databases "b0gus/databases"
)

/*
	NTP-RFC: www.rfc-editor.org/rfc/rfc958.html
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

	the roundtrip delay d and clock offset c is:
         d = (t4 - t1) - (t3 - t2)  and  c = (t2 - t1 + t3 - t4)/2 .
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
		LI_NO_WARNING = 0
		LI_ADD        = 1
		LI_SUB        = 2
		LI_RESERVED   = 3
		// VERSION NUMBER
		VN_FIRST = 1
		VN_LAST  = 4
		// MODE
		MODE_CLIENT = 3
	)
	b0gus_config.Logger.Info(req[0])
	var (
		/* 00_011_011 */
		l = (req[0] >> 6) & 0b11
		v = (req[0] >> 3) & 0b111
	)
	if !((l == LI_NO_WARNING) || (l == LI_RESERVED)) {
		return false
	}
	return (VN_FIRST <= v && v <= VN_LAST) && (req[0]&0b111 == MODE_CLIENT)
}

const FROM_1900_TO_1970 = 2208988800

// unix time: the number of seconds elapsed since January 1, 1970 UTC
// npt time: the number of seconds elapsed since January 1, 1900 UTC
// 1900~1970
func unix2ntp(u int64) int64 {
	return u + FROM_1900_TO_1970
}

// int2bytes
// format int number to four bytes.
// big endian.
func int2bytes(i int64) []byte {
	var b = make([]byte, 4)
	h1 := i >> 24
	h2 := (i >> 16) - (h1 << 8)
	h3 := (i >> 8) - (h1 << 16) - (h2 << 8)
	h4 := byte(i)
	b[0] = byte(h1)
	b[1] = byte(h2)
	b[2] = byte(h3)
	b[3] = byte(h4)
	return b
}

// TODO: A little bit weird. And need to add time resolution hook
func generate(req []byte) []byte {
	curr_time := time.Now()
	var second = unix2ntp(curr_time.Unix())
	var fraction = unix2ntp(int64(curr_time.Nanosecond()))
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
	res := generate(req)
	return res, nil
}

type NTPserverConf struct {
	/* database handler for writing data */
	DB_fd *b0gus_databases.RecordDB
	/* fields below need concurrenct control to follow the configuration */
	AlterNTPListener sync.Mutex
	// ConfigGenericCtrl b0gus_datatypes.ConcurrentCtrl
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	Addr              string       // b0gus NTP server addr
	Port              uint16       // b0gus NTP server port number
}

func (n *NTPserverConf) NTPclientHandler(
	link_gadget *serviceReadCtrl,
) {
	var (
		addr    string
		port    uint16
		end_ntp atomic.Bool
	)
	end_ntp.Store(false)

	// n.ConfigGenericCtrl.Ch <- struct{}{}
	addr = n.Addr
	port = n.Port
	// <-n.ConfigGenericCtrl.Ch

	udp_conn, err := net.ListenUDP(
		"udp", &net.UDPAddr{
			IP:   net.ParseIP(addr),
			Port: int(port),
		},
	)
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to listen on given addr:%s",
			fmt.Sprintf("%s:%d", addr, port),
		)
		return
	}

	defer udp_conn.Close()

	go func() {
	stuck:
		select {
		case <-link_gadget.Ctx.Done(): // we are done
		case suspected_datum := <-link_gadget.Data_ch:
			switch suspected_datum.(type) {
			case nil: // terminated signal checked from updated configuration
			case *b0gus_config.NTPconfig:
				b0gus_config.Logger.Infof(
					"yet to have capacity for updating ntp configuration<%v>",
					suspected_datum,
				)
				n.AlterNTPListener.Lock()
				// TODO: we need an updater for connection listener
				n.AlterNTPListener.Unlock()

				goto stuck
			default:
				goto stuck
			}
		}
		b0gus_config.Logger.Warn(
			"Catch a signal requirement for shuting down b0gus NTP server ",
		)
		end_ntp.Store(true)
		udp_conn.Close()
		n.AlterNTPListener.Lock()
		n.serverListenerPtr.Close()
		n.AlterNTPListener.Unlock()
	}()

	n.AlterNTPListener.Lock()
	n.serverListenerPtr = udp_conn
	n.AlterNTPListener.Unlock()

	var curr_listener *net.UDPConn
	for {
		if end_ntp.Load() {
			break
		}
		data_buf := make([]byte, 1024)
		n.AlterNTPListener.Lock()
		curr_listener = n.serverListenerPtr
		n.AlterNTPListener.Unlock()
		_, remote_ip, err := curr_listener.ReadFromUDP(data_buf)
		if err != nil {
			b0gus_config.Logger.Warn(err)
			continue
		}
		b0gus_config.Logger.Infof(
			"Receiving payload<len is %d> from %v",
			len(data_buf), remote_ip,
		)
		resp, err := NTPServe(data_buf)
		if err != nil {
			b0gus_config.Logger.Warn(err)
		}
		_, err = curr_listener.WriteToUDP(resp[:], remote_ip)
		if err != nil {
			b0gus_config.Logger.Warn(err)
			continue
		}
	}
}

// TODO: utilize database pointer
// db *gorm.DB *redis.Client *mongo.Client
func NTPserver(
	link_gadget *serviceReadCtrl,
	ntp_conf_obj *b0gus_config.NTPconfig,
	db *b0gus_databases.RecordDB,
	wait_group *sync.WaitGroup,
	args ...any,
) {
	defer wait_group.Done()
	ntp_server_conf := NTPserverConf{
		Addr:  ntp_conf_obj.ListenAddr,
		Port:  ntp_conf_obj.ListenPort,
		DB_fd: db,
	}
	// ntp_server_conf.ConfigGenericCtrl.Ch = make(chan struct{}, 1)
	// ntp_server_conf.ConfigGenericCtrl.Flag.Store(false)

	ntp_server_conf.NTPclientHandler(link_gadget)
	// close(ntp_server_conf.ConfigGenericCtrl.Ch)

}
