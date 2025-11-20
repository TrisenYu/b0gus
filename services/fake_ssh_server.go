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
	ssh "golang.org/x/crypto/ssh"
	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	crypto_aux "b0gus/crypto_aux"
	datatypes "b0gus/datatypes"
	misc_utils "b0gus/misc_utils"
)

// https://datatracker.ietf.org/doc/html/rfc4253#section-4.2
// SSH-protoversion-softwareversion SP comments CR LF
var (
	sshSoftwareArr = []string{
		"OpenSSH", "libssh", "libssh2", "billsSSHP",
		"PuTTY", "paramiko", "FlowSSH", "check_ssh",
	}
	// TODO: falsify operating system version in the near future
	sshOSCommentArr = []string{
		"Debian-10", "Debian-11", "Ubuntu-18.04",
		"Ubuntu-20.04", "Ubuntu-22.04", "Fedora-34",
		"Fedora-35", "Fedora-36", "Alpine-3.14",
		"Alpine-3.15", "Alpine-3.16", "FreeBSD-12",
		"FreeBSD-13", "OpenWrt-21.02", "OpenWrt-22.03",
		"Windows-10", "macOS-10.15",
	}
)

func getRandomSSHVersion() string {
	buf := make([]byte, 5)
	_, err := rand.Read(buf)
	if err != nil {
		b0gus_config.Logger.Error(
			"unable to generate 3 random bytes! Will return a default one!",
		)
		return "SSH-2.0-OpenSSH_10.0p2_3.5.4 Debian-7"
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
	Addr               string         // b0gus ssh server addr
	Port               uint16         // b0gus ssh server port number
	MaxClientNum       uint32         // maximum clients number handling in real time
	clientLimitChan    chan struct{}  // channel uses for inflow control
	signalChan         chan os.Signal // OS terminating control signal channel
	shouldTerminate    chan bool      // once being notisfied, push a `true` to the channel
	ClientConnTimeout  time.Duration  // initiated timeout setting
	DB_fd              *gorm.DB       // database for writing data
	InspectCommandHook any            // use for replacing command in repeat mode
}

var (
	term_str = map[string]any{
		"exit": nil,
		"quit": nil,
		"EXIT": nil,
		"QUIT": nil,
	}
	ctrl_seq = map[string]string{
		"\x00": "~", // ctrl+~
		"\x01": "A", // ctrl+a
		"\x02": "B", // ctrl+b
		/* ctrl+C, cease current command */
		"\x04": "D", // ctrl+d
		/* ctrl+e, use for exit */
		"\x06": "F", // ctrl+f
		"\x07": "G", // ctrl+g
		/* ctrl+H */
		/* ctrl+I */
		"\x0a": "J", // ctrl+j
		"\x0b": "K", // ctrl+k
		"\x0c": "L", // ctrl+l
		"\x0e": "N", // ctrl+n
		"\x0f": "O", // ctrl+o
		"\x10": "P", // ctrl+p
		/* ctrl+q, use for quit */
		"\x12": "R", // ctrl+r
		"\x13": "S", // ctrl+s
		"\x14": "T", // ctrl+t
		"\x15": "U", // ctrl+u
		"\x16": "V", // ctrl+v
		"\x17": "W", // ctrl+w
		"\x18": "X", // ctrl+x
		"\x19": "Y", // ctrl+y
		"\x1a": "Z", // ctrl+z
		/* [27,32) use for unknown and multiple mappings */
	}
)

func (s *SSHserverConf) cmdRepeater(ssh_chan ssh.Channel) {
	buf := make([]byte, 1)
	defer ssh_chan.Close()
Rewind:
	payload, last_char, prompt_char := "", "", "$ "
	for {
		if len(payload) > 1536 {
			// 1024 + 512, if longger than this threshold, then abort this
			ssh_chan.Write([]byte("\r\nExceed the maximum input limit\r\n"))
			ssh_chan.CloseWrite()
			return
		}
		lena, err := ssh_chan.Read(buf)
		if err != nil {
			b0gus_config.Logger.WithField("Err", err).
				Info("An error happend during interaction with Client")
			return
		} else if lena == 0 {
			// empty string or a new line, just keep reading
			b0gus_config.Logger.Info("An empty char")
			continue
		}
		// ctrl+c := \x03
		inp := string(buf[:lena])
		if inp == "\r" {
			// "\r" has been observed as the new line sign in linux terminal
			// we need a new dollar sign for fake interaction
			if last_char == "\\" {
				// still in one command, but the prompt should change to `dquote>`
				last_char = ""
				prompt_char = "dquote> "
				ssh_chan.Write([]byte("\r\n" + prompt_char))
				continue
			} else { // still need record command and print
				break
			}
		} else if inp == "\x03" { // ctrl + C
			payload, prompt_char = "", "$ "
			ssh_chan.Write([]byte("\r\n" + prompt_char))
			continue
		} else if inp == "\x08" || inp == "\x7F" {
			// The ascii code of `backspace` and `delete` key, refer to ANSI protocol
			// simplify the handling logic by forbidden the usage of arrow keys
			// and other control key representations used for shifting the cursor
			ssh_chan.Write([]byte("\r\x1b[1001K"))
			payload = payload[:max(0, len(payload)-1)]
			if len(payload) == 0 {
				last_char = ""
			} else {
				last_char = string(payload[len(payload)-1])
			}
			ssh_chan.Write([]byte(prompt_char + payload + " \x1b[1D"))
			continue
		} else if inp == "\x09" {
			// tab key
			payload += " "
			ssh_chan.Write([]byte("	"))
			continue
		} else if inp == "\x05" || inp == "\x11" {
			// ctrl + {E, Q}, treat it as exit/quit signal
			payload = "exit"
			break
		} else if char, ok := ctrl_seq[inp]; ok {
			b0gus_config.Logger.Warn("ctrl+" + char)
			continue
		} else if inp == "\\" {
			if last_char == "\\" {
				payload += "\\"
				last_char = ""
			} else {
				last_char = "\\"
			}
			ssh_chan.Write([]byte("\\"))
			continue
		}
		/* else { */
		last_char = inp
		/* } */
		if buf[0] < 0x20 {
			b0gus_config.Logger.Warn(
				"Current control char was not well handled: ",
				[]byte(last_char),
			)
			continue
		}
		payload += inp
		ssh_chan.Write([]byte(inp))
	}
	_, ok := term_str[payload]
	if ok {
		ssh_chan.Write([]byte("\r\nExit\r\n"))
		ssh_chan.CloseWrite()
		return
	}
	b0gus_config.Logger.Info(payload)
	prompt_char = "$ "
	// TODO: Add hook for specific commands output like `uname -a`
	// 0. check if the configuration needs such modification
	// 1. inspect command, determine whether it matches the request or not
	// 2. once match, modify the return pattern

	_, err := ssh_chan.Write([]byte("\r\n" + payload + "\r\n" + prompt_char))
	if err == nil {
		goto Rewind
	}
	b0gus_config.Logger.
		WithError(err).
		Info("Capture an error when writing payload")
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
		case "shell":
			_ = req.Reply(true, nil)
			welcomeMsg := fmt.Sprintf(
				// TODO: customized welcome banner
				"Last login: %s from %s\r\n$ ",
				time.Now().Format(time.ANSIC),
				ssh_conn.RemoteAddr().String(),
			)
			ssh_chan.Write([]byte(welcomeMsg))
			s.cmdRepeater(ssh_chan)
		default:
			// just accept pty-req and window-change without any action
			// window-change payload format:
			//	`uint32(rows)#uint32(cols)#uint32(width)#uint32(height)`
			// `#` means concatenate the information
			// reject/abort all other requests like "exec"
			// meanwhile, scp will send subsystem as its pre-executed request
			b0gus_config.Logger.Info("client try to " + req.Type)

			_ = req.Reply(
				req.Type == "pty-req" || req.Type == "window-change",
				nil,
			)
		}
	}
}

func (s *SSHserverConf) handle_new_ssh_chan(
	ssh_conn *ssh.ServerConn,
	new_chan ssh.NewChannel,
) {
	if new_chan.ChannelType() != "session" {
		_ = new_chan.Reject(
			ssh.UnknownChannelType,
			"Unsupported channel type",
		)
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
	err := conn.SetDeadline(time.Now().Add(s.ClientConnTimeout))
	if err != nil {
		b0gus_config.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Error("Failed to set timeout for incomming connection")
		return
	}
	b0gus_config.Logger.WithFields(logrus.Fields{
		"timeout": s.ClientConnTimeout,
	}).Info("Set")
	ip, port := misc_utils.IPaddrSplit(conn.RemoteAddr().String())
	var (
		attacker_addr_query_cond = datatypes.AttackerAddrDef{IP: ip}
		attacker_port_query_cond = datatypes.AttackerPortInfoDef{Port: port}
	)

	password_fn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		password_record := datatypes.AttackerPassInfoDef{
			Password: string(password),
		}
		s.DB_fd.Where(password_record).FirstOrCreate(&password_record)
		return &ssh.Permissions{}, nil
	}

	pubkey_func := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		pubkey_record := datatypes.AttackerPubInfoDef{
			PubKeyFingerprint: crypto_aux.PubKeyDeserialize(key.Marshal()),
		}
		s.DB_fd.Where(pubkey_record).FirstOrCreate(&pubkey_record)
		return nil, fmt.Errorf("public key authentication is not allowed")
	}

	ssh_config := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  password_fn,
		PublicKeyCallback: pubkey_func,
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
	// only can we handle so that the table is writable
	s.DB_fd.Where(attacker_addr_query_cond).FirstOrCreate(&attacker_addr_query_cond)
	attacker_port_query_cond.AttackerID = attacker_addr_query_cond.ID
	s.DB_fd.Where(attacker_port_query_cond).FirstOrCreate(&attacker_port_query_cond)

	// reject all relay requests since all clients are untrusted
	go ssh.DiscardRequests(relay_reqs)
	for new_chan := range chans {
		go s.handle_new_ssh_chan(ssh_conn, new_chan)
	}
}

func (s *SSHserverConf) SSHMaliciousClientHandler(host_key ssh.Signer) {
	// only accept tcp stream
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Addr, s.Port))
	if err != nil {
		b0gus_config.Logger.WithField(
			"given addr:",
			fmt.Sprintf("%s:%d", s.Addr, s.Port),
		).Fatal("Failed to listen on ")
	}
	defer listener.Close()

	// make channels for inflow control and termination determinant
	s.clientLimitChan = make(chan struct{}, s.MaxClientNum)
	s.signalChan, s.shouldTerminate = make(chan os.Signal, 1), make(chan bool, 1)
	// signal notification to terminate the ssh server gracefully
	signal.Notify(s.signalChan, syscall.SIGINT, syscall.SIGTERM)

	defer close(s.clientLimitChan)
	defer close(s.signalChan)

	var stop_flag atomic.Bool
	stop_flag.Store(false)

	go func() {
		sig := <-s.signalChan // stuck at this line until receiving any possible signal
		b0gus_config.Logger.WithField("signal", sig.String()).
			Warn("Catch an OS signal for terminating b0gus SSH server.\n")
		stop_flag.Store(true)
		listener.Close()
		s.shouldTerminate <- true
		close(s.shouldTerminate)
		// close only when receiving any termination signal
	}()

keep_spinning:
	select {
	case <-s.shouldTerminate:
		b0gus_config.Logger.Info(
			"Catch an signal requests for shuting down b0gus SSH server.\n",
		)
		return
	default:
		if stop_flag.Load() {
			return
		}
		b0gus_config.Logger.Info("incomming connection...")
		in_conn, err := listener.Accept()
		if err != nil {
			b0gus_config.Logger.Error(
				"Failed to accept incoming connection due to error:\n",
				err.Error(),
			)
			goto keep_spinning
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
	goto keep_spinning
}
