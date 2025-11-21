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
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_datatypes "b0gus/datatypes"
	b0gus_misc_utils "b0gus/misc_utils"
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
	Addr              string         // b0gus ssh server addr
	MaxClientNum      uint32         // maximum clients number handling in real time
	clientLimitChan   chan struct{}  // channel uses for inflow control
	signalChan        chan os.Signal // OS terminating control signal channel
	shouldTerminate   chan bool      // once being notisfied, push a `true` to the channel
	ClientConnTimeout time.Duration  // initiated timeout setting
	DB_fd             *gorm.DB       // database for writing data
	CommandHook       any            // use for replacing command in repeat mode, if not nil, then this field should be `func(string) string`
	Port              uint16         // b0gus ssh server port number
	PermitLogin       bool           // whether reject or not
	EmptyShell        bool           // no any response
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
		"\x0d": "M", // ctrl+m
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

func (s *SSHserverConf) cmdRepeater(
	api_id uint64,
	ssh_chan ssh.Channel,
) {
	buf := make([]byte, 1)
	defer ssh_chan.Close()

	// Do not try to identify what I am doing
	// 2-nested while True loop, but use label and goto for less ident
rewind:
	payload, last_char, prompt_char := "", "", "$ "

inner_loop:
	if len(payload) > 256 {
		// 256, if longger than this threshold, then abort this
		ssh_chan.Write([]byte("\r\nExceed the maximum input limit\r\n"))
		ssh_chan.CloseWrite()
		return
	}
	lena, err := ssh_chan.Read(buf)
	if err != nil {
		b0gus_config.Logger.Error(err)
		return
	} else if lena == 0 {
		// empty string or a new line, just keep reading
		// b0gus_config.Logger.Info("An empty char")
		goto inner_loop
	}

	// ctrl+c := \x03
	inp := string(buf[:lena])
	if inp == "\r" {
		// "\r" has been observed as the new line sign in linux terminal
		// we need a new dollar sign for fake interaction
		if last_char == "\\" {
			// still in one command, but the prompt should change to `dquote>`
			last_char, prompt_char = "", "dquote> "
			ssh_chan.Write([]byte("\r\n" + prompt_char))
			goto inner_loop

		}
		// break and record command
		goto jump_out

	} else if inp == "\x03" { // ctrl + C
		payload, last_char, prompt_char = "", "", "$ "
		ssh_chan.Write([]byte("\r\n" + prompt_char))
		goto inner_loop

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
		goto inner_loop

	} else if inp == "\x09" {
		// tab key
		payload += " "
		ssh_chan.Write([]byte("	"))
		goto inner_loop

	} else if inp == "\x05" || inp == "\x11" {
		// ctrl + {E, Q}, treat it as exit/quit signal
		last_char, payload = "", "exit"
		goto jump_out

	} else if char, ok := ctrl_seq[inp]; ok {
		b0gus_config.Logger.Warn("ctrl+" + char)
		goto inner_loop

	} else if inp == "\\" {
		if last_char == "\\" {
			payload += "\\"
			last_char = ""
		} else {
			last_char = "\\"
		}
		ssh_chan.Write([]byte("\\"))
		goto inner_loop

	}
	/* else { */
	last_char = inp
	/* } */
	if buf[0] < 0x20 {
		b0gus_config.Logger.Warn(
			"Current control char was not well handled: ",
			[]byte(last_char),
		)
		goto inner_loop
	}
	payload += inp
	ssh_chan.Write([]byte(inp))
	goto inner_loop

jump_out:
	_, ok := term_str[payload]
	if ok {
		ssh_chan.Write([]byte("\r\nExit\r\n"))
		ssh_chan.CloseWrite()
		return
	}
	go func() {
		cmd_text_record := b0gus_datatypes.CommandTextDef{
			CMD: payload,
		}
		s.DB_fd.Where(cmd_text_record).FirstOrCreate(&cmd_text_record)

		cmd_record := b0gus_datatypes.PortCmdRelated{
			LoginedID: api_id,
			CMDid:     cmd_text_record.CmdID,
		}
		s.DB_fd.Where(cmd_record).FirstOrCreate(&cmd_record)
	}()

	prompt_char = "$ "
	var final_payload = "\r\n\r\n$ "
	// TODO: is it possible to change the server configuration dynamically ?
	if s.CommandHook != nil {
		// Add hook for specific commands output like `uname -a`
		// 		0. check if the configuration needs such modification
		// 		1. inspect command, determine whether it matches the request or not
		// 		2. once match, modify the return pattern
		final_payload = "" + "\r\n" +
			s.CommandHook.(func(string) string)(payload) +
			"\r\n" + prompt_char
	} else if s.EmptyShell {

	} else {
		final_payload = "\r\n" + payload + "\r\n" + prompt_char
	}
	_, err = ssh_chan.Write([]byte(final_payload))
	if err == nil {
		goto rewind
	}
	b0gus_config.Logger.
		WithError(err).
		Info("Capture an error when writing payload")
}

func (s *SSHserverConf) requestsHandler(
	api_id uint64,
	ssh_conn *ssh.ServerConn,
	ssh_chan ssh.Channel,
	reqs <-chan *ssh.Request,
) {
	client_signal_chan := make(chan string, 1)
	defer close(client_signal_chan)
	defer ssh_chan.Close()

	// TODO: implement different strategies for incomming network flows
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
			s.cmdRepeater(api_id, ssh_chan)
		default:
			// just accept pty-req and window-change without any action
			// window-change payload format:
			//	`uint32(rows)#uint32(cols)#uint32(width)#uint32(height)`
			// `#` means concatenate the information
			// reject/abort all other requests like "exec"
			// meanwhile, scp will send subsystem as its pre-executed request
			// b0gus_config.Logger.Info("client try to " + req.Type)
			_ = req.Reply(
				req.Type == "pty-req" || req.Type == "window-change",
				nil,
			)
		}
	}
}

func (s *SSHserverConf) handle_new_ssh_chan(
	api_id uint64,
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
	s.requestsHandler(api_id, ssh_conn, ssh_chan, reqs)
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
	ip, port := b0gus_misc_utils.IPaddrSplit(conn.RemoteAddr().String())
	var (
		attacker_addr_query_cond = b0gus_datatypes.AddrInfoDef{IP: ip}
		attacker_port_query_cond = b0gus_datatypes.PortInfoDef{Port: port}
	)

	s.DB_fd.Where(attacker_addr_query_cond).
		FirstOrCreate(&attacker_addr_query_cond).
		Updates(b0gus_datatypes.AddrInfoDef{
			ID:       attacker_addr_query_cond.ID,
			IP:       ip,
			TryTimes: attacker_addr_query_cond.TryTimes + 1,
		})
	attacker_port_query_cond.AddrID = attacker_addr_query_cond.ID
	s.DB_fd.Where(attacker_port_query_cond).
		FirstOrCreate(&attacker_port_query_cond)

	password_fn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		// Record password in this function
		password_record := b0gus_datatypes.PassInfoDef{
			Password: string(password),
		}
		s.DB_fd.Where(password_record).FirstOrCreate(&password_record)
		username_record := b0gus_datatypes.UsernameDef{
			Username: conn.User(),
		}
		s.DB_fd.Where(username_record).FirstOrCreate(&username_record)
		port_name_related := b0gus_datatypes.PortNameRelated{
			LoginedID:  attacker_port_query_cond.APIid,
			UsernameID: username_record.UserID,
		}
		s.DB_fd.Where(port_name_related).FirstOrCreate(&port_name_related)
		client_ssh_version := b0gus_datatypes.SSHClientversionStrDef{
			ClientVersion: string(conn.ClientVersion()),
		}
		s.DB_fd.Where(client_ssh_version).FirstOrCreate(&client_ssh_version)
		port_pass_related := b0gus_datatypes.PortPassRelated{
			LoginedID: attacker_port_query_cond.APIid,
			PassID:    password_record.PasswordID,
		}
		s.DB_fd.Where(port_pass_related).FirstOrCreate(&port_pass_related)
		port_ver_related := b0gus_datatypes.PortVerRelated{
			LoginedID:          attacker_port_query_cond.APIid,
			SSHClientVersionID: client_ssh_version.VerID,
		}
		s.DB_fd.Where(port_ver_related).FirstOrCreate(&port_ver_related)

		return &ssh.Permissions{}, nil
	}

	pubkey_func := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		// Record public key in this function
		pubkey_record := b0gus_datatypes.PubInfoDef{
			PubKeyFingerprint: b0gus_crypto_aux.PubKeyDeserialize(key.Marshal()),
		}
		s.DB_fd.Where(pubkey_record).FirstOrCreate(&pubkey_record)
		client_ssh_version := b0gus_datatypes.SSHClientversionStrDef{
			ClientVersion: string(conn.ClientVersion()),
		}
		s.DB_fd.Where(client_ssh_version).FirstOrCreate(&client_ssh_version)
		port_ver_related := b0gus_datatypes.PortVerRelated{
			LoginedID:          attacker_port_query_cond.APIid,
			SSHClientVersionID: client_ssh_version.VerID,
		}
		s.DB_fd.Where(port_ver_related).FirstOrCreate(&port_ver_related)
		return nil, fmt.Errorf("public key authentication is not allowed")
	}

	ssh_config := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  password_fn,
		PublicKeyCallback: pubkey_func,
		NoClientAuth:      false, // request basic authentication
		MaxAuthTries:      3,
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

	// reject all relay requests since all clients are untrusted
	go ssh.DiscardRequests(relay_reqs)
	for new_chan := range chans {
		if !s.PermitLogin {
			new_chan.Reject(ssh.Prohibited, "Access Denied")
			continue
		}
		go s.handle_new_ssh_chan(attacker_port_query_cond.APIid, ssh_conn, new_chan)
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
		b0gus_config.Logger.Warn(
			"Catch an signal requests for shuting down b0gus SSH server.\n",
		)
		return
	default:
		if stop_flag.Load() {
			return
		}
		// b0gus_config.Logger.Info("incomming connection...")
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
			b0gus_config.Logger.Warn(
				"No available slot for new connection at present, shut down connection immediately",
			)
			in_conn.Close()
		}
	}
	goto keep_spinning
}
