// Package services
package services

// Last modified at 2026/02/11 星期三 22:20:29
// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/databases"
	"b0gus/misc_utils"
	"b0gus/terminal"
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
)

// getRandomSSHVersion will return a random
// SSH-proto_version-software_version SP comments CR LF
func getRandomSSHVersion() string {
	buf := make([]byte, 5)
	_, err := rand.Read(buf)
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"services.SSHversionWarn", nil,
		)
		configs.Logger.Warn(payload)
		return "SSH-2.0-OpenSSH_10.0p2_3.5.4 Debian-7"
	}
	for i := range 3 {
		buf[i] %= 10
	}
	buf[3] %= uint8(len(sshSoftwareArr))
	buf[4] %= uint8(len(sshOSCommentArr))
	var (
		header     = "SSH-2.0-%s_%d.%d.%d %s"
		sshVersion = fmt.Sprintf(
			header,
			sshSoftwareArr[buf[3]],
			buf[0], buf[1], buf[2],
			sshOSCommentArr[buf[4]],
		)
	)
	return sshVersion
}

type SSHServConf struct {
	clientLimitor     atomic.Uint32 // connection limitor
	configGuard       sync.RWMutex
	tcpListenerGuard  sync.RWMutex
	hostSigner        ssh.Signer
	DbFd              *configs.RuntimeDB // database handler for writing data.
	ConfOptions       *configs.SSHconfig
	serverListenerPtr *net.Listener // current listener on Addr:Port
}

// a reading try from a close channel in go1.26 will finally get empty string.
//	package main
//	import "fmt"
//	func main() {
//		ch := make(chan string)
//		go func() { ch <- "hello" } ()
//		val, ok := <- ch
//		fmt.Println(val, len(val), ok)
//		close(ch)
//		val, ok = <- ch
//		fmt.Println(val, len(val), ok)
//		val, ok = <- ch
//		fmt.Println(val, len(val), ok)
//	}
//	// hello 5 true
//	//  0 false
//	//  0 false
//

// cmdForwarding will create a mock shell for interaction
func (s *SSHServConf) cmdForwarding(sshChan ssh.Channel) { // api_id uint64,
	term := terminal.NewShell(sshChan, sshChan)
	var shouldCease atomic.Bool
	shouldCease.Store(false)

	defer func() {
		_ = sshChan.Close()
	}()
	go func() {
		err := term.Run()
		defer shouldCease.Store(true)
		if err == nil {
			return
		}
		logInfo := configs.GetLocalizedMsg(
			"services.SSHwriteResponseError",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger.Info(logInfo)
	}()
	s.configGuard.RLock()
	respType := s.ConfOptions.ResponseType
	s.configGuard.RUnlock()

	for !shouldCease.Load() {
		currPayload := term.GetCurrCmd()
		if currPayload == nil || len(currPayload.Payload) == 0 {
			break
		}
		/*
			TODO: Add hook for specific commands output like `uname -a`
				0. check if the configuration needs such modification
				1. inspect command, determine whether it matches the request or not
				2. once match, modify the return pattern
		*/
		switch strings.ToLower(respType) {
		case "llm":
			// TODO: switch to easier mode if all tokens run up or utilized local LLM if possible...

		case "empty":
			go term.SetCurrResp(&terminal.ShellSyncObj{Ctx: currPayload.Ctx, Payload: ""})
		case "repeat":
			fallthrough
		default:
			// ? what the hack
			go term.SetCurrResp(currPayload)
		}
		cmdText := databases.CommandInfo{Cmd: currPayload.Payload}
		_ = s.DbFd.CreateOrUpdateItem(&cmdText, &cmdText)
		// cmd_record := databases.RemoteCommandRelation{
		// 	Rid: int64(apiId),
		// 	Cid: cmd_text_record.CommandId, // [TODO] ?
		// }
		// s.DbFd.CreateOrUpdateItem(cmd_record, &cmd_record)
	}
}

func (s *SSHServConf) mockShellForRemote( // api_id uint64,
	banner string,
	sshConn *ssh.ServerConn,
	sshChan ssh.Channel,
	reqs <-chan *ssh.Request,
) {
	defer func() { _ = sshChan.Close() }()
	for req := range reqs {
		switch req.Type {
		case "shell":
			_ = req.Reply(true, nil)
			welcomeMsg := fmt.Sprintf(
				strings.ReplaceAll(banner, "\n", "\r\n")+"Last login: %s from %s\r\n",
				time.Now().Format(time.ANSIC),
				sshConn.RemoteAddr().String(),
			)
			_, _ = sshChan.Write([]byte(welcomeMsg))
			go s.cmdForwarding(sshChan) // api_id,
		default:
			/*
				just accept 'pty-req' and 'window-change' without any action

				payload format of 'window-change':
					`uint32(rows)#uint32(cols)#uint32(width)#uint32(height)`
				here `#` means concatenate the information

				reject/abort all other requests like "exec"

				TODO: scp will send 'subsystem' as its pre-executed request.
					so it is worth wondering what kind of attacking payload does the attacker send.
					To achieve this goal, the requirements are container and privilege deprivation
			*/
			_ = req.Reply(
				req.Type == "pty-req" || req.Type == "window-change",
				nil,
			)
		}
	}
}

func (s *SSHServConf) handleNewSSHchan(
	// api_id uint64,
	banner string,
	sshConn *ssh.ServerConn,
	newChan ssh.NewChannel,
) {
	if newChan.ChannelType() != "session" {
		_ = newChan.Reject(
			ssh.UnknownChannelType,
			"Unsupported channel type",
		)
		return
	}
	sshChan, reqs, err := newChan.Accept()
	if err != nil {
		payload := configs.GetLocalizedMsg(
			"services.SSHchannelAcceptanceError",
			map[string]any{
				"RemoteAddr": sshConn.RemoteAddr().String(),
				"ErrInfo":    err,
			},
		)
		configs.Logger.Error(payload)
		return
	}
	// TODO: use configuration to determine which mode do we need.
	//		1. talk with an LLM with tailored prompt
	s.mockShellForRemote(banner, sshConn, sshChan, reqs)
}

// TODO: Notice that the listener/connection setup phases of various
// 	protocol are pretty similar, it will be much better to extract the commonness
// 	from these functions and organize them as a generic function/interface

func (s *SSHServConf) clientConnHandler(
	conn net.Conn,
) {
	var (
		maxNum            uint32
		clientConnTimeout time.Duration
	)
	s.configGuard.RLock()
	maxNum = s.ConfOptions.MaxClientNum
	clientConnTimeout = time.Duration(s.ConfOptions.ClientConnTimeout) * time.Second
	maxTryTimes := int(s.ConfOptions.MaxAuthTries)
	s.configGuard.RUnlock()

	if s.clientLimitor.Load() >= maxNum {
		_ = conn.Close() // out of threshold, abort this connection
		return
	}
	s.clientLimitor.Add(1)
	defer func() {
		s.clientLimitor.Add(^uint32(0))
		_ = conn.Close()
	}()

	err := conn.SetDeadline(time.Now().Add(clientConnTimeout))
	if err != nil {
		errInfo := configs.GetLocalizedMsg(
			"services.SSHsetTimeoutError",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger.Error(errInfo)
		return
	}
	ip, port := misc_utils.IPAddrSplit(conn.RemoteAddr().String())
	s.DbFd.SetupContext(
		&databases.AddrInfo{Ip: ip},
		&databases.PortInfo{Port: int64(port)},
	)

	passwordFn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		// Record password in this function
		passwordRecord := databases.PasswordInfo{Password: string(password)}
		usernameRecord := databases.UsernameInfo{Name: conn.User()}
		clientSshVersion := databases.SshVersionInfo{
			Version: string(conn.ClientVersion()),
		}
		// I still don't think relation really matters here
		// because the values are more important
		_ = s.DbFd.CreateOrUpdateItemsInSeq(
			&passwordRecord, &usernameRecord,
			&clientSshVersion,
		)
		return &ssh.Permissions{}, nil
	}
	pubkeyFunc := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		// Record public key in this function
		pubkeyRecord := databases.PublickeyInfo{
			PubFp: crypto_aux.Base64Deserialize(key.Marshal()),
		}
		clientSshVersion := databases.SshVersionInfo{
			Version: string(conn.ClientVersion()),
		}
		// TODO: look like we can not bypass this relation if relation has to be well managed.
		// databases.RemoteInfo{}
		_ = s.DbFd.CreateOrUpdateItemsInSeq(&pubkeyRecord, &clientSshVersion)
		return nil, fmt.Errorf("public key authentication is not allowed")
	}

	sshConfig := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  passwordFn,
		PublicKeyCallback: pubkeyFunc,
		NoClientAuth:      false, // request basic authentication
		MaxAuthTries:      maxTryTimes,
	}

	sshConfig.AddHostKey(s.hostSigner)

	sshConn, chans, relayReqs, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		errInfo := configs.GetLocalizedMsg(
			"services.SSHEstablishConnectionFailure",
			map[string]any{
				"RemoteAddr": conn.RemoteAddr().String(),
				"CurrSSHver": sshConfig.ServerVersion,
				"ErrInfo":    err,
			},
		)
		configs.Logger.Error(errInfo)
		return
	}
	defer func() { _ = sshConn.Close() }()
	/* reject all relay requests since all clients are untrusted */
	go ssh.DiscardRequests(relayReqs)
	for newChan := range chans {
		s.configGuard.RLock()
		permitLogin := s.ConfOptions.PermitLogin
		loginBanner := s.ConfOptions.LoginBanner
		s.configGuard.RUnlock()
		if !permitLogin {
			_ = newChan.Reject(ssh.Prohibited, "Access Denied")
			continue
		}
		go s.handleNewSSHchan(
			// uint64(attacker_port_query_cond.Port),
			loginBanner, sshConn, newChan,
		)
	}
}


func (s *SSHServConf) UpdateConfig(src *configs.SSHconfig) {
	if !configs.CheckSSHconfig(src) {
		warnInfo := configs.GetLocalizedMsg(
			"services.SSHinvalidConfigurationWarn",
			nil,
		)
		configs.Logger.Warn(warnInfo)
		return
	}
	s.configGuard.Lock()
	if s.ConfOptions != src {
		s.ConfOptions.MaxClientNum = src.MaxClientNum
		s.ConfOptions.PermitLogin = src.PermitLogin
		s.ConfOptions.LoginBanner = src.LoginBanner
		s.ConfOptions.ClientConnTimeout = src.ClientConnTimeout
		s.ConfOptions.ResponseType = src.ResponseType
	}
	s.configGuard.Unlock()
	// we have to free the previous listener
	s.AlterListener(src.ListenPort)
}

func (s *SSHServConf) AlterListener(port uint16) {
	s.tcpListenerGuard.Lock()
	defer func() { s.tcpListenerGuard.Unlock() }()
	if port == s.ConfOptions.ListenPort {
		return /* no need to update */
	}

	/* apply new configuration here */
	tmp, err := net.Listen(
		"tcp",
		fmt.Sprintf("%d", port),
	)
	if err != nil {
		errInfo := configs.GetLocalizedMsg(
			"services.SSHSwitchListenerError",
			map[string]any{
				"ListenAddr": fmt.Sprintf(":%d", port),
			},
		)
		configs.Logger.Info(errInfo)
		return
	}
	if s.serverListenerPtr != nil {
		_ = (*s.serverListenerPtr).Close()
	}
	s.serverListenerPtr = &tmp
	s.ConfOptions.ListenPort = port
}

func (s *SSHServConf) SSHMaliciousClientHandler(
	scc *configs.ServConcurrentCtrl,
) {
	var (
		port    uint16
		endSign atomic.Bool
	)
	endSign.Store(false)
	s.tcpListenerGuard.RLock()
	port = s.ConfOptions.ListenPort
	s.tcpListenerGuard.RUnlock()

	// only accept tcp stream
	listener, err := net.Listen(
		"tcp", fmt.Sprintf(":%d", port),
	)
	if err != nil {
		errInfo := configs.GetLocalizedMsg(
			"services.SSHSetupListenerError",
			map[string]any{
				"ListenAddr": fmt.Sprintf(":%d", port),
			},
		)
		configs.Logger.Error(errInfo)
		return
	}
	defer func() { _ = listener.Close() }()

	s.tcpListenerGuard.Lock()
	s.serverListenerPtr = &listener
	s.tcpListenerGuard.Unlock()

	var closeListener = func() {
		s.tcpListenerGuard.Lock()
		_ = (*s.serverListenerPtr).Close()
		s.tcpListenerGuard.Unlock()
	}

	s.clientLimitor.Store(0)
	go func() {
		/* signal to this channels will use as termination determinant */
		for {
			select {
			case <-scc.Ctx.Done():
				// terminated notification
			case castedDatum := <-scc.DataCh:
				switch tmp := castedDatum.(type) {
				case nil:
					// cease services
				case configs.SSHconfig:
					s.UpdateConfig(&tmp)
					continue
				case *configs.SSHconfig:
					s.UpdateConfig(tmp)
					continue
				default:
					// receive this signal for updating configuration
					continue
				}
			}
			break
		}
		// otherwise cease listening
		endSign.Store(true)
		closeListener()
	}()

	var currListener net.Listener

	/* effect before each return */
keepSpinning:
	if endSign.Load() {
		closeListener()
		return
	}

	s.tcpListenerGuard.RLock()
	// when we have to hot-plug with new configuration,
	// we need a blocking mechanism to stop accepting new connection
	// until the listener is ready to be put in use again
	currListener = *s.serverListenerPtr
	s.tcpListenerGuard.RUnlock()

	inConn, err := currListener.Accept()
	if err != nil {
		warnInfo := configs.GetLocalizedMsg(
			"services.SSHAcceptConnWarn",
			map[string]any{
				"ErrInfo": err,
			},
		)
		configs.Logger.Warn(warnInfo)
		goto keepSpinning
	}
	if endSign.Load() {
		_ = inConn.Close()
		_ = listener.Close()
		closeListener()
		return
	}
	go s.clientConnHandler(inConn)
	goto keepSpinning
}

func (s *SSHServConf) Run(
	sshConfObj *configs.SSHconfig,
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB,
	args ...any,
) {
	defer func() {
		payload := configs.GetLocalizedMsg(
			"services.SSHQuitInfo",
			nil,
		)
		configs.Logger.Info(payload)
	}()

	if len(args) != 1 {
		// TODO: add an description for this.
		return
	}
	hostKey, ok := args[0].(ssh.Signer)
	if !ok {
		payload := configs.GetLocalizedMsg(
			"services.SSHCanNotUseSigner",
			map[string]any{
				"Signer": hostKey,
			},
		)
		configs.Logger.Warn(payload)
		return
	}
	s.hostSigner = hostKey
	err := db.CreateTable(
		&databases.AddrInfo{}, &databases.PortInfo{},
		&databases.UsernameInfo{}, &databases.PasswordInfo{},
		&databases.PublickeyInfo{},
		&databases.SshVersionInfo{},
		&databases.CommandInfo{},
		// extended relations
		&databases.RemoteInfo{},
		&databases.RemoteUsernameRelation{},
		&databases.RemotePasswordRelation{},
		&databases.RemotePublickeyRelation{},
		&databases.RemoteSshverRelation{},
		&databases.RemoteCommandRelation{},
	)
	if err != nil {
		configs.Logger.Error(err.Error())
		return
	}
	s.ConfOptions = sshConfObj
	s.DbFd = db
	s.SSHMaliciousClientHandler(scc)
}
