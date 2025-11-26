// SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
package services

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_datatypes "b0gus/datatypes"
)

const (
	FROM_1900_TO_1970 = 2208988800
)

func validFormat(req []byte) bool {
	if len(req) == 0 {
		return false
	}
	const (
		LI_NO_WARNING = 0
		LI_ALARM_COND = 3
		// VERSION NUMBER
		VN_FIRST    = 1
		VN_LAST     = 4
		MODE_CLIENT = 3
	)
	var (
		l = req[0] >> 6
		v = (req[0] << 2) >> 5
		m = (req[0] << 5) >> 5
	)
	if !((l == LI_NO_WARNING) || (l == LI_ALARM_COND)) {
		return false
	}
	return (VN_LAST <= v && v <= VN_FIRST) && m == MODE_CLIENT
}

// unix time: the number of seconds elapsed since January 1, 1970 UTC
// npt time: the number of seconds elapsed since January 1, 1900 UTC
// 1900~1970
func unix2ntp(u int64) int64 {
	return u + FROM_1900_TO_1970
}

// func ntp2unix(n int64) int64 {
// 	return n - FROM_1900_TO_1970
// }

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

// generate
/*
	Field Name              Request    Reply
    ----------------------------------------------------------
    LI                      0 or 3     0
    VN                      1-4        copied from request
    Mode                    3          4
    Stratum                 ignore     1
    Poll                    ignore     copied from request
    Precision               ignore     -log2 server significant bits
    Root Delay              ignore     0
    Root Dispersion         ignore     0
    Reference Identifier    ignore     source ident
    Reference Timestamp     ignore     time of last radio update
	Originate Timestamp     ignore     copied from transmit timestamp
    Receive Timestamp       ignore     time of day
    Transmit Timestamp      (see text) time of day
*/
func generate(req []byte) []byte {
	var second = unix2ntp(time.Now().Unix())
	var fraction = unix2ntp(int64(time.Now().Nanosecond()))
	var res = make([]byte, 48)
	var vn = req[0] & 0x38
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
	DB_fd *gorm.DB // database handler for writing data.
	/*
		use for replacing command in repeat mode.
		if not nil, then this field should be `func(string) string`
	*/
	// fields below need concurrenct control to follow the configuration
	AlterNTPListener  sync.Mutex
	ConfigGenericCtrl b0gus_datatypes.ConcurrentCtrl
	serverListenerPtr *net.UDPConn // current listener on Addr:Port
	Addr              string       // b0gus ssh server addr
	/* NOTE that Port and Addr is hard to update in real time */
	Port uint16 // b0gus ssh server port number
}

func (n *NTPserverConf) NTPclientHandler(
	terminator *b0gus_datatypes.ConcurrentCtrl,
) {
	var (
		addr string
		port uint16
	)
	n.ConfigGenericCtrl.Ch <- struct{}{}
	addr = n.Addr
	port = n.Port
	<-n.ConfigGenericCtrl.Ch

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

	n.AlterNTPListener.Lock()
	n.serverListenerPtr = udp_conn
	n.AlterNTPListener.Unlock()

	go func() {
		/* signal to this channels will use as termination determinant */
		<-terminator.Ch // stuck here until receiving termination signal
		b0gus_config.Logger.Warn(
			"Catch a signal requirement for shuting down b0gus NTP server ",
		)
		udp_conn.Close()
		n.AlterNTPListener.Lock()
		n.serverListenerPtr.Close()
		n.AlterNTPListener.Unlock()
	}()

	var curr_listener *net.UDPConn
	for {
		if terminator.Flag.Load() {
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
			"Receiving payload from %v",
			remote_ip,
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

// TODO: Have not test yet...
func NTPserver(
	terminator *b0gus_datatypes.ConcurrentCtrl,
	ntp_conf_obj *b0gus_config.NTPconfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	defer wait_group.Done()
	ntp_server_conf := NTPserverConf{
		Addr:  ntp_conf_obj.ListenAddr,
		Port:  ntp_conf_obj.ListenPort,
		DB_fd: db,
	}
	ntp_server_conf.ConfigGenericCtrl.Ch = make(chan struct{}, 1)
	ntp_server_conf.ConfigGenericCtrl.Flag.Store(false)

	ntp_server_conf.NTPclientHandler(terminator)
	close(ntp_server_conf.ConfigGenericCtrl.Ch)

}
