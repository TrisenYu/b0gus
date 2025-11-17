package services

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	logrus "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	b0gus_config "b0gus/configs"
)

// https://datatracker.ietf.org/doc/html/rfc4253#section-4.2
// SSH-protoversion-softwareversion SP comments CR LF
var (
	sshSoftwareArr = []string{
		"OpenSSH",
		"libssh",
		"libssh2",
		"billsSSHP",
		"PuTTY",
		"paramiko",
		"FlowSSH",
		"check_ssh",
	}
	// TODO: fake operating system version in the near future
	sshOSCommentArr = []string{
		"Debian-10",
		"Debian-11",
		"Ubuntu-18.04",
		"Ubuntu-20.04",
		"Ubuntu-22.04",
		"Fedora-34",
		"Fedora-35",
		"Fedora-36",
		"Alpine-3.14",
		"Alpine-3.15",
		"Alpine-3.16",
		"FreeBSD-12",
		"FreeBSD-13",
		"OpenWrt-21.02",
		"OpenWrt-22.03",
		"Windows-10",
		"macOS-10.15",
	}
)

func getRandomSSHVersion() string {
	buf := make([]byte, 5)
	_, err := rand.Read(buf)
	if err != nil {
		b0gus_config.Logger.Fatal("unable to generate 3 random bytes!")
	}
	for i := range 3 {
		buf[i] %= 10
	}
	buf[3] %= uint8(len(sshSoftwareArr))
	buf[4] %= uint8(len(sshOSCommentArr))
	var (
		header      = "SSH-2.0-%s_%d.%d.%d %s"
		ssh_version = fmt.Sprintf(
			header,
			sshSoftwareArr[buf[3]],
			buf[0], buf[1], buf[2],
			sshOSCommentArr[buf[4]],
		)
	)
	return ssh_version
}

type SSHserverConf struct {
	Addr              string
	Port              uint16
	MaxClientNum      uint32
	clientLimitChan   chan struct{}
	signalChan        chan os.Signal
	shouldTerminate   chan bool
	ClientConnTimeout time.Duration
}

var term_str = map[string]int8{
	"exit": 1,
	"quit": 1,
	"EXIT": 1,
	"QUIT": 1,
}

func (s *SSHserverConf) cmdRepeater(ssh_chan ssh.Channel, client_sig_chan chan string) {
	// TODO: break until any \n but not one char each time we type
	// TODO: what if user stop inputing and send a SIGINT?
	// there should be a corresponding mechanism for message clean

	payload := ""
	buf := make([]byte, 1)
	defer ssh_chan.Close()

	select {
	case sig_str := <-client_sig_chan:
		// TODO:
		// control + C: cease current command and return -1
		// control + U: clean buf
		// control + D: exit
		// control + A: move cursor to head
		// control + shift + c: copy
		// control + shift + v: paste
		// control + l: clean all, but this program won't do anything
		// switch sig_str {
		// case "C":
		// case "U":
		// case "A":
		// case "L":
		// case "W":
		// default:
		// }
		ssh_chan.Write([]byte(sig_str + "\r\n"))
	default:
	}
	var last_char = ""
	for {
		lena, err := ssh_chan.Read(buf)
		if err != nil {
			b0gus_config.Logger.WithField("Err", err).
				Info("An error happend during interaction with Client")
			break
		}
		inp := string(buf[:lena])
		ssh_chan.Write(buf[:lena])
		// TODO...
		if len(inp) == 0 {
			continue
		}
		b0gus_config.Logger.Info(last_char)
		payload += inp
		last_char = inp
	}
	// _, ok := term_str[string(payload)]
	// if ok {
	// 	ssh_chan.Write([]byte("Exit"))
	// 	ssh_chan.CloseWrite()
	// 	return
	// }
	ssh_chan.CloseWrite()

	// ssh_chan.Write([]byte(payload + "\r\n$"))
}

func (s *SSHserverConf) requestsHandler(
	ssh_conn *ssh.ServerConn,
	ssh_chan ssh.Channel,
	reqs <-chan *ssh.Request,
) {

	client_signal_chan := make(chan string, 1)
	defer close(client_signal_chan)
	defer ssh_chan.Close()

	for req := range reqs {
		switch req.Type {
		case "signal":
			sig := string(req.Payload)
			_ = req.Reply(true, nil)
			select {
			case client_signal_chan <- sig:
			default:
				b0gus_config.Logger.Info("invalid signal: ", sig)
				// invalid signal, won't do anything
				continue
			}
		case "pty-req":
			// just accept without action
			fallthrough
		case "window-change":
			// Payload format: uint32(rows)||uint32(cols)||uint32(width)||uint32(height)
			// here we just accept it and do nothing
			_ = req.Reply(true, nil)
		case "shell":
			_ = req.Reply(true, nil)
			welcomeMsg := fmt.Sprintf(
				// TODO: customized welcome banner
				"Last login: %s from %s\r\n$ ",
				time.Now().Format(time.ANSIC),
				ssh_conn.RemoteAddr().String(),
			)
			_, _ = ssh_chan.Write([]byte(welcomeMsg))
			s.cmdRepeater(ssh_chan, client_signal_chan)
		case "exec":
			// interacation commands
			_ = req.Reply(true, nil)
			_, _ = ssh_chan.Write([]byte(string(req.Payload[4:]) + "\r\n"))
			_ = ssh_chan.CloseWrite()
		default:
			// reject all other requests
			_ = req.Reply(false, nil)
		}
	}
}

func (s *SSHserverConf) checkOSsignal() {
	sig := <-s.signalChan
	// stuck here until receive any signal
	b0gus_config.Logger.WithField("signal", sig.String()).
		Warn("Catch an OS signal for terminating b0gus SSH server.\n")
	s.shouldTerminate <- true
}

func (s *SSHserverConf) handle_new_ssh_chan(ssh_conn *ssh.ServerConn, new_chan ssh.NewChannel) {
	if new_chan.ChannelType() != "session" {
		_ = new_chan.Reject(ssh.UnknownChannelType, "Unsupported channel type")
		return
	}
	ssh_chan, reqs, err := new_chan.Accept()
	if err != nil {
		b0gus_config.Logger.WithFields(logrus.Fields{
			"remote addr": ssh_conn.RemoteAddr().String(),
			"err":         err,
		}).Error("Unable to accept channel for ")
		return
	}
	s.requestsHandler(ssh_conn, ssh_chan, reqs)
}

func (s *SSHserverConf) ClientConnHandler(
	conn net.Conn,
	host_key ssh.Signer,
) {
	defer func() {
		<-s.clientLimitChan
		conn.Close()
	}()
	_ = conn.SetDeadline(time.Now().Add(s.ClientConnTimeout))
	password_fn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		return &ssh.Permissions{}, nil
	}
	privateKey_fn := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		return nil, fmt.Errorf("public key authentication is not allowed")
	}

	ssh_config := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  password_fn,
		PublicKeyCallback: privateKey_fn,
		NoClientAuth:      true, // so-called honeypot
	}

	ssh_config.AddHostKey(host_key)

	ssh_conn, chans, relay_reqs, err := ssh.NewServerConn(conn, ssh_config)
	if err != nil {
		b0gus_config.Logger.WithFields(logrus.Fields{
			"remote addr:":         conn.RemoteAddr(),
			"err":                  err,
			"current-SSH-version:": ssh_config.ServerVersion,
		}).Error("Failed to establish SSH connection")
		return
	}
	go ssh.DiscardRequests(relay_reqs)
	for new_chan := range chans {
		go s.handle_new_ssh_chan(ssh_conn, new_chan)
	}
}

func (s *SSHserverConf) SSHMaliciousClientHandler(host_key ssh.Signer) {
	// only accept tcp stream
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Addr, s.Port))
	if err != nil {
		b0gus_config.Logger.WithField("given addr:", fmt.Sprintf("%s:%d", s.Addr, s.Port)).
			Fatal("Failed to listen on ")
	}
	defer listener.Close()

	// make channels for inflow control and notifying termination
	s.clientLimitChan = make(chan struct{}, s.MaxClientNum)
	s.signalChan = make(chan os.Signal, 1)
	s.shouldTerminate = make(chan bool, 1)
	// signal notification to terminate the ssh server gracefully
	// TODO: BUT not knowing why the program still hang and not free after sending SIGINT or SIGTERM
	signal.Notify(s.signalChan, syscall.SIGINT, syscall.SIGTERM)

	defer close(s.clientLimitChan)
	defer close(s.signalChan)
	defer close(s.shouldTerminate)

	var stop_flag atomic.Bool
	stop_flag.Store(false)

	go s.checkOSsignal()
	// TODO: not quit after ctrl+C, yet to find bug
	for {
		if stop_flag.Load() {
			break
		}
		select {
		case <-s.shouldTerminate:
			stop_flag.Store(true)
			b0gus_config.Logger.Info(
				"catch an signal requests for shuting down b0gus SSH server.\n",
			)
			// close only when receive any termination signal
			// TODO: wait for all client and close all channels
			return
		default:
			in_conn, err := listener.Accept()
			if err != nil {
				b0gus_config.Logger.Error(
					"Failed to accept incoming connection due to error:\n",
					err.Error(),
				)
				continue
			}
			if stop_flag.Load() {
				in_conn.Close()
				return
			}
			select {
			case s.clientLimitChan <- struct{}{}:
				go s.ClientConnHandler(in_conn, host_key)
			default:
				b0gus_config.Logger.Info(
					"No available slot for new connection at present, shut down connection immediately",
				)
				in_conn.Close()
			}
		}
	}
}
