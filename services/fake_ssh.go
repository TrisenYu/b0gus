// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package services

import (
	"crypto/rand"
	// "encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logrus "github.com/sirupsen/logrus"
	ssh "golang.org/x/crypto/ssh"

	// gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	"b0gus/databases"
	b0gus_databases "b0gus/databases"
	b0gus_datatypes "b0gus/generic_datatypes"
	b0gus_misc_utils "b0gus/misc_utils"
)

// default port number: 22
// https://datatracker.ietf.org/doc/html/rfc4253#section-4.2
var (
	sshSoftwareArr = []string{
		"OpenSSH", "libssh", "libssh2", "billsSSHP",
		"PuTTY", "paramiko", "FlowSSH", "check_ssh",
		"dropbear",
	}
	// TODO: falsify operating system and its version in the future
	sshOSCommentArr = []string{
		"Debian-10", "Debian-11", "Ubuntu-18.04",
		"Ubuntu-20.04", "Ubuntu-22.04", "Fedora-34",
		"Fedora-35", "Fedora-36", "Alpine-3.14",
		"Alpine-3.15", "Alpine-3.16", "FreeBSD-12",
		"FreeBSD-13", "OpenWrt-21.02", "OpenWrt-22.03",
		"Windows-11", "Windows-10", "macOS-10.15",
		"Raspbian-5+deb8u4",
	}
)

/* SSH-protoversion-softwareversion SP comments CR LF */
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
	clientLimitor atomic.Uint32
	DB_fd         *databases.RecordDB // database handler for writing data.
	/*
		use for replacing command in repeat mode.
		if not nil, then this field should be `func(string) string`
	*/
	commandHook any
	/* fields below need concurrenct control to follow the configuration */

	TCPListenerSwitchDone sync.Mutex
	ConfigGenericCtrl     b0gus_datatypes.ConcurrentCtrl
	serverListenerPtr     *net.Listener // current listener on Addr:Port
	Addr                  string        // b0gus ssh server addr
	LoginBanner           string        // ssh server login banner
	ClientConnTimeout     time.Duration // initiated timeout setting
	MaxClientNum          uint32        // maximum clients number handling in real time
	Port                  uint16        // b0gus ssh server port number
	PermitLogin           bool          // whether reject or not
	EmptyShell            bool          // no any response
}

var (
	term_str = map[string]any{
		"exit": nil, "quit": nil,
		"EXIT": nil, "QUIT": nil,
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
rewind:
	payload, last_char, prompt_char := "", "", "$ "
inner_loop:
	if len(payload) > 384 {
		// 384=256+128, if longger than this threshold, then abort this
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

	} else if _, ok := ctrl_seq[inp]; ok {
		// b0gus_config.Logger.Warn("ctrl+" + _)
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

	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */
	go func() {
		cmd_text_record := b0gus_databases.CommandTextDef{
			CMD: payload,
		}
		s.DB_fd.Where(cmd_text_record).FirstOrCreate(&cmd_text_record)

		cmd_record := b0gus_databases.PortCmdRelated{
			LoginedID: api_id,
			CMDid:     cmd_text_record.CmdID,
		}
		s.DB_fd.Where(cmd_record).FirstOrCreate(&cmd_record)
	}()
	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */

	prompt_char = "$ "
	var (
		final_payload = "\r\n\r\n$ "
		cmd_hook      any
		empty_shell   bool
	)

	s.ConfigGenericCtrl.Ch <- struct{}{}
	cmd_hook = s.commandHook
	empty_shell = s.EmptyShell
	<-s.ConfigGenericCtrl.Ch

	if cmd_hook != nil {
		/*
			Add hook for specific commands output like `uname -a`
				0. check if the configuration needs such modification
				1. inspect command, determine whether it matches the request or not
				2. once match, modify the return pattern
		*/
		final_payload = "" +
			"\r\n" + cmd_hook.(func(string) string)(payload) +
			"\r\n" + prompt_char
	} else if empty_shell {

	} else {
		final_payload = "\r\n" + payload + "\r\n" + prompt_char
	}
	_, err = ssh_chan.Write([]byte(final_payload))
	if err == nil {
		goto rewind
	}
	b0gus_config.Logger.Infof("Capture an error when writing payload:%v", err)
}

func (s *SSHserverConf) requestsHandler(
	api_id uint64,
	banner string,
	ssh_conn *ssh.ServerConn,
	ssh_chan ssh.Channel,
	reqs <-chan *ssh.Request,
) {
	defer ssh_chan.Close()
	for req := range reqs {
		switch req.Type {
		case "shell":
			_ = req.Reply(true, nil)
			welcomeMsg := fmt.Sprintf(
				strings.ReplaceAll(banner, "\n", "\r\n")+"Last login: %s from %s\r\n$ ",
				time.Now().Format(time.ANSIC),
				ssh_conn.RemoteAddr().String(),
			)
			ssh_chan.Write([]byte(welcomeMsg))
			s.cmdRepeater(api_id, ssh_chan)
		default:
			/*
				just accept 'pty-req' and 'window-change' without any action

				payload format of 'window-change':
					`uint32(rows)#uint32(cols)#uint32(width)#uint32(height)`
				here `#` means concatenate the information

				reject/abort all other requests like "exec"
				meanwhile, scp will send 'subsystem' as its pre-executed request
			*/
			_ = req.Reply(
				req.Type == "pty-req" || req.Type == "window-change",
				nil,
			)
		}
	}
}

func (s *SSHserverConf) handle_new_ssh_chan(
	api_id uint64,
	banner string,
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
	s.requestsHandler(api_id, banner, ssh_conn, ssh_chan, reqs)
}

// TODO: Notice that the listener/connection setup phases of various
// protocol are pretty similar, it will be better to extract the commoness
// from these functions and orignize them as a generic function/interface
func (s *SSHserverConf) clientConnHandler(
	conn net.Conn,
	host_key ssh.Signer,
) {
	var (
		max_num             uint32
		client_conn_timeout time.Duration
	)
	s.ConfigGenericCtrl.Ch <- struct{}{}
	max_num = s.MaxClientNum
	client_conn_timeout = s.ClientConnTimeout
	<-s.ConfigGenericCtrl.Ch

	if s.clientLimitor.Load() >= max_num {
		conn.Close()
		return
	}
	s.clientLimitor.Add(1)
	defer func() {
		s.clientLimitor.Add(^uint32(0))
		conn.Close()
	}()

	err := conn.SetDeadline(time.Now().Add(client_conn_timeout))
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to set timeout for incomming connection due to err:%s",
			err.Error(),
		)
		return
	}
	ip, port := b0gus_misc_utils.IPaddrSplit(conn.RemoteAddr().String())
	// binary.BigEndian.Uint32(net.ParseIP(ip).To4())

	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database types Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */

	var (
		attacker_addr_query_cond = b0gus_databases.AddrInfoDef{IP: ip}
		attacker_port_query_cond = b0gus_databases.PortInfoDef{Port: port}
	)

	s.DB_fd.Where(attacker_addr_query_cond).
		FirstOrCreate(&attacker_addr_query_cond).
		Updates(b0gus_databases.AddrInfoDef{
			ID:       attacker_addr_query_cond.ID,
			IP:       ip,
			TryTimes: attacker_addr_query_cond.TryTimes + 1,
		})
	attacker_port_query_cond.AddrID = attacker_addr_query_cond.ID
	s.DB_fd.Where(attacker_port_query_cond).FirstOrCreate(&attacker_port_query_cond)

	password_fn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		/*
			TODO:
			What if we use a generic metatable as a form and decide them in the backend?

			Though we Only have to push them into the table define in redis/mongodb,
			still have to consider the side-effect impacting on traditional database
		*/
		// Record password in this function
		password_record := b0gus_databases.PassInfoDef{
			Password: string(password),
		}
		s.DB_fd.Where(password_record).
			FirstOrCreate(&password_record).
			Updates(b0gus_databases.PassInfoDef{
				PasswordID: password_record.PasswordID,
				Counter:    password_record.Counter + 1,
			})
		username_record := b0gus_databases.UsernameDef{
			Username: conn.User(),
		}
		s.DB_fd.Where(username_record).
			FirstOrCreate(&username_record).
			Updates(b0gus_databases.UsernameDef{
				UserID:  username_record.UserID,
				Counter: username_record.Counter + 1,
			})
		port_name_related := b0gus_databases.PortNameRelated{
			LoginedID:  attacker_port_query_cond.APIid,
			UsernameID: username_record.UserID,
		}
		s.DB_fd.Where(port_name_related).FirstOrCreate(&port_name_related)
		client_ssh_version := b0gus_databases.SSHClientversionStrDef{
			ClientVersion: string(conn.ClientVersion()),
		}
		s.DB_fd.Where(client_ssh_version).FirstOrCreate(&client_ssh_version)
		port_pass_related := b0gus_databases.PortPassRelated{
			LoginedID: attacker_port_query_cond.APIid,
			PassID:    password_record.PasswordID,
		}
		s.DB_fd.Where(port_pass_related).FirstOrCreate(&port_pass_related)
		port_ver_related := b0gus_databases.PortVerRelated{
			LoginedID:          attacker_port_query_cond.APIid,
			SSHClientVersionID: client_ssh_version.VerID,
		}
		s.DB_fd.Where(port_ver_related).FirstOrCreate(&port_ver_related)

		return &ssh.Permissions{}, nil
	}

	pubkey_func := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		// Record public key in this function
		pubkey_record := b0gus_databases.PubInfoDef{
			PubKeyFingerprint: b0gus_crypto_aux.PubKeyDeserialize(key.Marshal()),
		}
		s.DB_fd.Where(pubkey_record).FirstOrCreate(&pubkey_record)
		client_ssh_version := b0gus_databases.SSHClientversionStrDef{
			ClientVersion: string(conn.ClientVersion()),
		}
		s.DB_fd.Where(client_ssh_version).FirstOrCreate(&client_ssh_version)
		port_ver_related := b0gus_databases.PortVerRelated{
			LoginedID:          attacker_port_query_cond.APIid,
			SSHClientVersionID: client_ssh_version.VerID,
		}
		s.DB_fd.Where(port_ver_related).FirstOrCreate(&port_ver_related)
		return nil, fmt.Errorf("public key authentication is not allowed")
	}
	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database types Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */

	ssh_config := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  password_fn,
		PublicKeyCallback: pubkey_func,
		NoClientAuth:      false, // request basic authentication
		MaxAuthTries:      3,     // TODO: should we configure this and utilize as another strategy?
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
	defer ssh_conn.Close()
	/* reject all relay requests since all clients are untrusted */
	go ssh.DiscardRequests(relay_reqs)
	var (
		permit_login bool
		login_banner string
	)
	for new_chan := range chans {
		s.ConfigGenericCtrl.Ch <- struct{}{}
		permit_login = s.PermitLogin
		login_banner = s.LoginBanner
		<-s.ConfigGenericCtrl.Ch

		if !permit_login {
			new_chan.Reject(ssh.Prohibited, "Access Denied")
			continue
		}
		go s.handle_new_ssh_chan(
			attacker_port_query_cond.APIid,
			login_banner,
			ssh_conn, new_chan,
		)
	}
}

func (s *SSHserverConf) UpdateConfig(src *b0gus_config.SSHconfig) {
	// TODO: Maybe user want to cease the execution immediately by explictly modifying the configuration
	if !b0gus_config.CheckSSHconfig(src) {
		b0gus_config.Logger.Error("Invalid SSH configuration, skip updating")
		return
	}
	s.ConfigGenericCtrl.Ch <- struct{}{}
	s.PermitLogin = src.PermitLogin
	s.EmptyShell = src.EmptyShell
	s.LoginBanner = src.LoginBanner
	s.ClientConnTimeout = time.Duration(src.ClientConnTimeout) * time.Second
	s.MaxClientNum = src.MaxClientNum
	<-s.ConfigGenericCtrl.Ch

	s.NewListener(src.ListenAddr, src.ListenPort)
}

func (s *SSHserverConf) NewListener(addr string, port uint16) {
	s.TCPListenerSwitchDone.Lock()
	defer func() { s.TCPListenerSwitchDone.Unlock() }()
	if addr == s.Addr && port == s.Port {
		return /* no need to update */
	}

	/* apply new configuration here */
	tmp, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%d", addr, port),
	)
	if err != nil {
		b0gus_config.Logger.Infof(
			"Failed to listen on given addr:%s, won't modify the listener of b0gus ssh server",
			fmt.Sprintf("%s:%d", addr, port),
		)
		return
	}
	if s.serverListenerPtr != nil {
		(*s.serverListenerPtr).Close()
	}
	s.serverListenerPtr = &tmp
	s.Addr, s.Port = addr, port
}

func (s *SSHserverConf) SSHMaliciousClientHandler(
	terminator *b0gus_datatypes.ConcurrentCtrl,
	host_key ssh.Signer,
) {
	var (
		addr string
		port uint16
	)
	s.TCPListenerSwitchDone.Lock()
	addr, port = s.Addr, s.Port
	s.TCPListenerSwitchDone.Unlock()
	// only accept tcp stream
	listener, err := net.Listen(
		"tcp", fmt.Sprintf("%s:%d", addr, port),
	)
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to listen on given addr:%s",
			fmt.Sprintf("%s:%d", addr, port),
		)
		return
	}
	defer listener.Close()

	s.TCPListenerSwitchDone.Lock()
	s.serverListenerPtr = &listener
	s.TCPListenerSwitchDone.Unlock()

	var locker = func() {
		s.TCPListenerSwitchDone.Lock()
		(*s.serverListenerPtr).Close()
		s.TCPListenerSwitchDone.Unlock()
	}

	s.clientLimitor.Store(0)
	go func() {
		/* signal to this channels will use as termination determinant */
		<-terminator.Ch // stuck here until receiving termination signal
		b0gus_config.Logger.Warn(
			"Catch a signal requirement for shuting down b0gus SSH server ",
		)
		listener.Close()
		locker()
	}()

	var curr_listener net.Listener

	/* Affect before each return */
keep_spinning:
	if terminator.Flag.Load() {
		locker()
		return
	}

	s.TCPListenerSwitchDone.Lock()
	// when we have to hot-plug with new configuration,
	// we need a block mechanism to stop accept new connection
	// until the listener is ready to be put in use again
	curr_listener = *s.serverListenerPtr
	s.TCPListenerSwitchDone.Unlock()

	in_conn, err := curr_listener.Accept()
	if err != nil {
		b0gus_config.Logger.Warnf(
			"Failed to accept incoming connection due to error:%s",
			err.Error(),
		)
		goto keep_spinning
	}
	if terminator.Flag.Load() {
		in_conn.Close()
		listener.Close()
		locker()
		return
	}
	go s.clientConnHandler(in_conn, host_key)
	goto keep_spinning
}

// *gorm.DB *redis.Client *mongo.Client
func SSHserver(
	need_shutdown *b0gus_datatypes.ConcurrentCtrl,
	ssh_conf_obj *b0gus_config.SSHconfig,
	host_key ssh.Signer,
	db *b0gus_databases.RecordDB,
	wait_group *sync.WaitGroup,
) {
	defer wait_group.Done()
	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */
	// ssh-related tables
	err := db.CreateTable(
		&b0gus_databases.AddrInfoDef{},
		&b0gus_databases.PortInfoDef{},
		&b0gus_databases.UsernameDef{},
		&b0gus_databases.SSHClientversionStrDef{},
		&b0gus_databases.PubInfoDef{},
		&b0gus_databases.PassInfoDef{},
		&b0gus_databases.CommandTextDef{},
		// Relation Tables
		&b0gus_databases.PortNameRelated{},
		&b0gus_databases.PortVerRelated{},
		&b0gus_databases.PortPubKeyRelated{},
		&b0gus_databases.PortPassRelated{},
		&b0gus_databases.PortCmdRelated{},
	)
	if err != nil {
		b0gus_config.Logger.Error(err)
		return
	}
	/* -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= Database Need distinguishing -=-=-=-=-=-=--=-=-=-=-=-=--=-=-=-=-= */

	// Fill SSH Server Configuration with definitions in config file
	ssh_server_conf := SSHserverConf{
		Addr:              ssh_conf_obj.ListenAddr,
		Port:              ssh_conf_obj.ListenPort,
		MaxClientNum:      ssh_conf_obj.MaxClientNum,
		ClientConnTimeout: time.Duration(ssh_conf_obj.ClientConnTimeout) * time.Second,
		PermitLogin:       ssh_conf_obj.PermitLogin,
		EmptyShell:        ssh_conf_obj.EmptyShell,
		LoginBanner:       ssh_conf_obj.LoginBanner,
		DB_fd:             db,
	}
	ssh_server_conf.ConfigGenericCtrl.Ch = make(chan struct{}, 1)
	defer close(ssh_server_conf.ConfigGenericCtrl.Ch)
	ssh_server_conf.ConfigGenericCtrl.Flag.Store(false)
	// callback function for updating when there is any modification in the monitored configuration file
	var ssh_callback = func() {
		b0gus_config.Logger.Info("Renew SSH configuration")
		ssh_server_conf.UpdateConfig(ssh_conf_obj)
	}
	go b0gus_config.GlobConfigMaintainer.Regist(ssh_callback)
	ssh_server_conf.SSHMaliciousClientHandler(need_shutdown, host_key)
}
