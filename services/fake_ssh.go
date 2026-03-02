// Last modified at 2026/02/11 星期三 22:20:29
// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package services

import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ssh "golang.org/x/crypto/ssh"

	b0gus_assets "b0gus/assets"
	b0gus_config "b0gus/configs"

	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_databases "b0gus/databases"
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
	// TODO: falsify operating system and its version string in the future
	sshOSCommentArr = []string{
		"Debian-10", "Debian-11", "Ubuntu-18.04",
		"Ubuntu-20.04", "Ubuntu-22.04", "Fedora-34",
		"Fedora-35", "Fedora-36", "Alpine-3.14",
		"Alpine-3.15", "Alpine-3.16", "FreeBSD-12",
		"FreeBSD-13", "OpenWrt-21.02", "OpenWrt-22.03",
		"Windows-11", "Windows-10", "macOS-10.15",
		"Raspbian-5+deb8u4",
	}
	termStr = map[string]any{
		"exit": nil, "quit": nil,
		"EXIT": nil, "QUIT": nil,
	}
	ctrlSeq = map[string]string{
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

/* SSH-proto_version-software_version SP comments CR LF */
func getRandomSSHVersion() string {
	buf := make([]byte, 5)
	_, err := rand.Read(buf)
	if err != nil {
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHversionWarn", nil,
		)
		b0gus_config.Logger.Warn(payload)
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
	clientLimitor                 atomic.Uint32
	tcpListenerGuard, configGuard sync.RWMutex
	DB_fd                         *b0gus_config.RuntimeDB // database handler for writing data.
	/*
		use for replacing command in repeat mode.
		if not nil, then this field should be `func(string) string`
	*/
	commandHook any
	/* fields below need concurrenct control to follow the configuration */

	serverListenerPtr *net.Listener // current listener on Addr:Port
	Addr              string        // b0gus ssh server addr
	LoginBanner       string        // ssh server login banner
	ClientConnTimeout time.Duration // initiated timeout setting
	MaxClientNum      uint32        // maximum clients number handling in real time
	Port              uint16        // b0gus ssh server port number
	PermitLogin       bool          // whether reject or not
	EmptyShell        bool          // no any response
}

// TODO: use terminal interaction instead
func (s *SSHserverConf) cmdForwarding(ssh_chan ssh.Channel) { // api_id uint64,
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

	} else if _, ok := ctrlSeq[inp]; ok {
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
	_, ok := termStr[payload]
	if ok {
		ssh_chan.Write([]byte("\r\nExit\r\n"))
		ssh_chan.CloseWrite()
		return
	}

	go func() {
		cmd_text_record := b0gus_databases.CommandInfo{
			Cmd: payload,
		}
		s.DB_fd.CreateOrUpdateItem(&cmd_text_record, &cmd_text_record)
		// cmd_record := b0gus_databases.RemoteCommandRelation{
		// 	Rid: int64(api_id),
		// 	Cid: cmd_text_record.CommandId, // [TODO] ?
		// }
		// s.DB_fd.CreateOrUpdateItem(cmd_record, &cmd_record)
	}()

	prompt_char = "$ "
	var (
		final_payload = "\r\n\r\n$ "
		cmd_hook      any
		empty_shell   bool
	)

	s.configGuard.RLock()
	cmd_hook = s.commandHook
	empty_shell = s.EmptyShell
	s.configGuard.RUnlock()

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
	log_info := b0gus_assets.GetLocalizedMsg(
		b0gus_config.GetLang(),
		"services.SSHWriteResponseError",
		map[string]any{"ErrInfo": err},
	)
	b0gus_config.Logger.Info(log_info)
}

func (s *SSHserverConf) requestsHandler(
// api_id uint64,
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
			s.cmdForwarding(ssh_chan) // api_id,
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

func (s *SSHserverConf) handleNewSSHchan(
// api_id uint64,
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
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHchannelAcceptanceError",
			map[string]any{
				"RemoteAddr": ssh_conn.RemoteAddr().String(),
				"ErrInfo":    err,
			},
		)
		b0gus_config.Logger.Error(payload)
		return
	}
	// api_id,
	s.requestsHandler(banner, ssh_conn, ssh_chan, reqs)
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
	s.configGuard.RLock()
	max_num = s.MaxClientNum
	client_conn_timeout = s.ClientConnTimeout
	s.configGuard.RUnlock()

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
		err_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHsetTimeoutError",
			map[string]any{"ErrInfo": err},
		)
		b0gus_config.Logger.Error(err_info)
		return
	}
	ip, port := b0gus_misc_utils.IPaddrSplit(conn.RemoteAddr().String())
	var (
		// Context created by these data
		attacker_addr_query_cond = b0gus_databases.AddrInfo{Ip: ip}
		attacker_port_query_cond = b0gus_databases.PortInfo{Port: int64(port)}
	)

	// s.DB_fd.Where(attacker_addr_query_cond).
	// 	FirstOrCreate(&attacker_addr_query_cond).
	// 	Updates(b0gus_databases.AddrInfoDef{
	// 		ID:       attacker_addr_query_cond.ID,
	// 		IP:       ip,
	// 		TryTimes: attacker_addr_query_cond.TryTimes + 1,
	// 	})
	// attacker_port_query_cond.AddrID = attacker_addr_query_cond.ID
	// s.DB_fd.Where(attacker_port_query_cond).FirstOrCreate(&attacker_port_query_cond)
	s.DB_fd.SetupContext(&attacker_addr_query_cond, &attacker_port_query_cond)

	password_fn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		/*
			TODO:
			What if we use a generic metatable as a form and decide them in the backend?

			Though we only have to push them into the table define in redis/mongodb,
			still have to consider the side-effect impacting on traditional database
		*/
		// Record password in this function
		password_record := b0gus_databases.PasswordInfo{
			Password: string(password),
		}
		username_record := b0gus_databases.UsernameInfo{
			Name: conn.User(),
		}
		client_ssh_version := b0gus_databases.SshVersionInfo{
			Version: string(conn.ClientVersion()),
		}
		// I still don't think relation really matters here
		// because values are more important
		s.DB_fd.CreateOrUpdateItemsInSeq(
			&password_record, &username_record, &client_ssh_version,
		)
		return &ssh.Permissions{}, nil
	}

	pubkey_func := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		// Record public key in this function
		pubkey_record := b0gus_databases.PublickeyInfo{
			PubFp: b0gus_crypto_aux.PubKeyDeserialize(key.Marshal()),
		}
		client_ssh_version := b0gus_databases.SshVersionInfo{
			Version: string(conn.ClientVersion()),
		}
		// look like we can not bypass this relation if relation has to be managed.
		// b0gus_databases.RemoteInfo{}
		s.DB_fd.CreateOrUpdateItemsInSeq(&pubkey_record, &client_ssh_version)
		return nil, fmt.Errorf("public key authentication is not allowed")
	}

	ssh_config := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  password_fn,
		PublicKeyCallback: pubkey_func,
		NoClientAuth:      false, // request basic authentication
		MaxAuthTries:      3,     // [TODO]: should we configure this and utilize as another strategy?
	}

	ssh_config.AddHostKey(host_key)

	ssh_conn, chans, relay_reqs, err := ssh.NewServerConn(conn, ssh_config)
	if err != nil {
		err_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHEstablishConnectionFailure",
			map[string]any{
				"RemoteAddr": conn.RemoteAddr().String(),
				"CurrSSHver": ssh_config.ServerVersion,
				"ErrInfo":    err,
			},
		)
		b0gus_config.Logger.Error(err_info)
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
		s.configGuard.RLock()
		permit_login = s.PermitLogin
		login_banner = s.LoginBanner
		s.configGuard.RUnlock()

		if !permit_login {
			new_chan.Reject(ssh.Prohibited, "Access Denied")
			continue
		}
		go s.handleNewSSHchan(
			// uint64(attacker_port_query_cond.Port),
			login_banner, ssh_conn, new_chan,
		)
	}
}

func (s *SSHserverConf) UpdateConfig(src *b0gus_config.SSHconfig) {
	if !b0gus_config.CheckSSHconfig(src) {
		warn_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHinvalidConfigurationWarn",
			nil,
		)
		b0gus_config.Logger.Warn(warn_info)
		return
	}
	s.configGuard.Lock()
	s.PermitLogin = src.PermitLogin
	s.EmptyShell = src.EmptyShell
	s.LoginBanner = src.LoginBanner
	s.ClientConnTimeout = time.Duration(src.ClientConnTimeout) * time.Second
	s.MaxClientNum = src.MaxClientNum
	s.configGuard.Unlock()

	s.NewListener(src.ListenAddr, src.ListenPort)
}

func (s *SSHserverConf) NewListener(addr string, port uint16) {
	s.tcpListenerGuard.Lock()
	defer func() { s.tcpListenerGuard.Unlock() }()
	if addr == s.Addr && port == s.Port {
		return /* no need to update */
	}

	/* apply new configuration here */
	tmp, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%d", addr, port),
	)
	if err != nil {
		err_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHSwitchListenerError",
			map[string]any{
				"ListenAddr": fmt.Sprintf("%s:%d", addr, port),
			},
		)
		b0gus_config.Logger.Info(err_info)
		return
	}
	if s.serverListenerPtr != nil {
		(*s.serverListenerPtr).Close()
	}
	s.serverListenerPtr = &tmp
	s.Addr, s.Port = addr, port
}

// type resolveUpdConfig interface {
// 	any | *b0gus_config.SSHconfig
// }

func (s *SSHserverConf) SSHMaliciousClientHandler(
	scc *b0gus_config.ServicesConcurrencyCtrl,
	host_key ssh.Signer,
) {
	var (
		addr     string
		port     uint16
		end_sign atomic.Bool
	)
	end_sign.Store(false)
	s.tcpListenerGuard.RLock()
	addr, port = s.Addr, s.Port
	s.tcpListenerGuard.RUnlock()
	/* only accept tcp stream */
	listener, err := net.Listen(
		"tcp", fmt.Sprintf("%s:%d", addr, port),
	)
	if err != nil {
		err_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHSetupListenerError",
			map[string]any{
				"ListenAddr": fmt.Sprintf("%s:%d", addr, port),
			},
		)
		b0gus_config.Logger.Error(err_info)
		return
	}
	defer listener.Close()

	s.tcpListenerGuard.Lock()
	s.serverListenerPtr = &listener
	s.tcpListenerGuard.Unlock()

	var locker = func() {
		s.tcpListenerGuard.Lock()
		(*s.serverListenerPtr).Close()
		s.tcpListenerGuard.Unlock()
	}

	s.clientLimitor.Store(0)
	go func() {
		/* signal to this channels will use as termination determinant */
	stuck:
		select {
		case <-scc.Ctx.Done(): // terminated notification
		case castedDatum := <-scc.Data_ch:
			switch tmp := castedDatum.(type) {
			case nil: // cease services
			case b0gus_config.SSHconfig:
				s.UpdateConfig(&tmp)
				goto stuck
			case *b0gus_config.SSHconfig:
				s.UpdateConfig(tmp)
				goto stuck
			default:
				// receive this signal for updating configuration
				goto stuck
			}
		}
		// otherwise cease listening
		end_sign.Store(true)
		// listener.Close()
		locker()
	}()

	var curr_listener net.Listener

	/* effect before each return */
keep_spinning:
	if end_sign.Load() {
		locker()
		return
	}

	s.tcpListenerGuard.RLock()
	// when we have to hot-plug with new configuration,
	// we need a blocking mechanism to stop accepting new connection
	// until the listener is ready to be put in use again
	curr_listener = *s.serverListenerPtr
	s.tcpListenerGuard.RUnlock()

	in_conn, err := curr_listener.Accept()
	if err != nil {
		warn_info := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHAcceptConnWarn",
			map[string]any{
				"ErrInfo": err,
			},
		)
		b0gus_config.Logger.Warn(warn_info)
		goto keep_spinning
	}
	if end_sign.Load() {
		in_conn.Close()
		listener.Close()
		locker()
		return
	}
	go s.clientConnHandler(in_conn, host_key)
	goto keep_spinning
}

func (s *SSHserverConf) Run(
	ssh_conf_obj *b0gus_config.SSHconfig,
	scc *b0gus_config.ServicesConcurrencyCtrl,
	db *b0gus_config.RuntimeDB,
	args ...any,
) {
	defer func() {
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHQuitInfo",
			nil,
		)
		b0gus_config.Logger.Info(payload)
	}()

	if len(args) != 1 {
		// TODO: add an description for this.
		return
	}
	host_key, ok := args[0].(ssh.Signer)
	if !ok {
		payload := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"services.SSHCanNotUseSigner",
			map[string]any{
				"Signer": host_key,
			},
		)
		b0gus_config.Logger.Warn(payload)
		return
	}
	err := db.CreateTable(
		&b0gus_databases.AddrInfo{},
		&b0gus_databases.PortInfo{},
		&b0gus_databases.UsernameInfo{},
		&b0gus_databases.PasswordInfo{},
		&b0gus_databases.SshVersionInfo{},
		&b0gus_databases.PublickeyInfo{},
		&b0gus_databases.CommandInfo{},
		// extended relations
		&b0gus_databases.RemoteInfo{},
		&b0gus_databases.RemoteUsernameRelation{},
		&b0gus_databases.RemotePasswordRelation{},
		&b0gus_databases.RemotePublickeyRelation{},
		&b0gus_databases.RemoteSshverRelation{},
		&b0gus_databases.RemoteCommandRelation{},
	)
	if err != nil {
		b0gus_config.Logger.Error(err)
		return
	}
	s.Addr = ssh_conf_obj.ListenAddr
	s.Port = uint16(ssh_conf_obj.ListenPort)
	s.MaxClientNum = ssh_conf_obj.MaxClientNum
	s.ClientConnTimeout = time.Duration(ssh_conf_obj.ClientConnTimeout) * time.Second
	s.PermitLogin = ssh_conf_obj.PermitLogin
	s.EmptyShell = ssh_conf_obj.EmptyShell
	s.LoginBanner = ssh_conf_obj.LoginBanner
	s.DB_fd = db
	s.SSHMaliciousClientHandler(scc, host_key)
}
