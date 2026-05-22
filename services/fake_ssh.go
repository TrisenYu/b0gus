// Package services
package services

/// Last modified at 2026/05/16 星期六 12:46:24
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/databases"
	"b0gus/internal/misc_utils"
	"b0gus/terminal"
)

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
		configs.Logger().Warn(payload)
		return "SSH-2.0-OpenSSH_10.0p2_3.5.4 Debian-7"
	}
	for i := range 3 {
		buf[i] %= 10
	}
	buf[3] %= uint8(len(sshSoftwareArr))
	buf[4] %= uint8(len(sshOSCommentArr))
	var sb strings.Builder
	sb.WriteString("SSH-2.0-")
	sb.WriteString(sshSoftwareArr[buf[3]])
	sb.WriteRune('_')
	sb.WriteString(strconv.Itoa(int(buf[0])))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(buf[1])))
	sb.WriteRune('.')
	sb.WriteString(strconv.Itoa(int(buf[2])))
	sb.WriteRune(' ')
	sb.WriteString(sshOSCommentArr[buf[4]])
	// "SSH-2.0-%s_%d.%d.%d %s"
	return sb.String()
}

// SSHServConf is designed for executing the fake ssh service.
//
//	default port number: 22
//	https://datatracker.ietf.org/doc/html/rfc4253#section-4.2
type SSHServConf struct {
	// hostSigner holds the ssh key of host
	hostSigner ssh.Signer
	// DbFd is an object for interacting with database
	DbFd databases.DBhandler
	// ConfOptions is a dangling copy of global configuration
	ConfOptions *atomic.Pointer[configs.LocalConfig]
}

// Run is the function for invoking the fake ssh service
func (s *SSHServConf) Run(
	confObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db databases.DBhandler, args ...any,
) {
	defer func() {
		payload := configs.GetLocalizedMsg(
			"services.SSHQuitInfo",
			nil,
		)
		configs.Logger().Info(payload)
	}()

	if len(args) != 1 {
		// [TODO]: add a description for this.
		payload := configs.GetLocalizedMsg(
			"services.SSHRequireSignerAsOnlyArgErr", nil,
		)
		configs.Logger().Error(payload)
		return
	} else if confObj == nil {
		payload := configs.GetLocalizedMsg(
			"services.SSHEmptyGlobConfigFailure",
			nil,
		)
		configs.Logger().Error(payload)
		return
	}
	_, ok := confObj.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
	if !ok {
		payload := configs.GetLocalizedMsg(
			"services.SSHConfigLoadingFailure",
			nil,
		)
		configs.Logger().Error(payload)
		return
	}
	hostKey, ok := args[0].(ssh.Signer)
	if !ok {
		payload := configs.GetLocalizedMsg(
			"services.SSHCanNotUseSigner",
			map[string]any{"Signer": hostKey},
		)
		configs.Logger().Warn(payload)
		return
	}
	s.hostSigner = hostKey
	err := db.CreateTable(
		&databases.AddrInfo{}, &databases.PortInfo{},
		&databases.UsernameInfo{}, &databases.PasswordInfo{},
		&databases.PublickeyInfo{}, &databases.SshVersionInfo{}, &databases.CommandInfo{},
		// extended relations
		&databases.RemoteUsernameRelation{}, &databases.RemotePasswordRelation{},
		&databases.RemotePublickeyRelation{}, &databases.RemoteSshverRelation{},
		&databases.RemoteCommandRelation{},
	)
	if err != nil {
		configs.Logger().Error(err.Error())
		return
	}
	var clientAux ReentrantNetType
	s.ConfOptions, s.DbFd = confObj, db
	clientAux.Init(configs.SSHEnum, s)
	go clientAux.EventMonitor(confObj, scc)
	clientAux.AlterNetFd(TCPEnum, confObj)
}

// InvokeForUDPtask here will do nothing due to ssh is a TCP protocol
func (s *SSHServConf) InvokeForUDPtask(net.Addr, []byte) []byte { return nil }

// ServeHTTP here will do nothing due to ssh is a TCP protocol
func (s *SSHServConf) ServeHTTP(http.ResponseWriter, *http.Request) {}

// InvokeForICMPtask here will do nothing due to ssh is a TCP protocol
func (s *SSHServConf) InvokeForICMPtask(net.Addr, []byte) {}

// InvokeForTCPtask is implemented for the callback function defined in ReentrantNetType.
func (s *SSHServConf) InvokeForTCPtask(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	snapshot, ok := s.ConfOptions.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
	if !ok {
		// TODO: log
		return
	}
	maxTryTimes := int(snapshot.MaxAuthTries)
	hashAlg := crypto_aux.OnceHashByChoice(snapshot.HashAlgorithm)
	ip, port := misc_utils.IPAddrSplit(conn.RemoteAddr().String())
	sessionID := int64(binary.LittleEndian.Uint64(hashAlg(
		[]byte(conn.RemoteAddr().String()), []byte(time.Now().String()),
	)[:8]))
	_ = s.DbFd.CreateOrUpdateItemsInSeq(
		&databases.AddrInfo{Ip: ip},
		&databases.PortInfo{Port: int64(port)},
	)
	passwordFn := func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		// Record password in this function
		passwordRecord := databases.PasswordInfo{Password: string(password)}
		usernameRecord := databases.UsernameInfo{Name: conn.User()}
		clientSshVersion := databases.SshVersionInfo{Version: string(conn.ClientVersion())}
		_ = s.DbFd.CreateOrUpdateItemsInSeq(
			&passwordRecord, &usernameRecord, &clientSshVersion,
		)
		_ = s.DbFd.CreateOrUpdateItemsInSeq(
			&databases.RemotePasswordRelation{SessionId: sessionID, Pid: passwordRecord.PasswordId},
			&databases.RemoteUsernameRelation{SessionId: sessionID, Uid: usernameRecord.UsernameId},
			&databases.RemoteSshverRelation{SessionId: sessionID, Sid: clientSshVersion.Id},
		)
		return &ssh.Permissions{}, nil
	}
	pubkeyFunc := func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		// Record public key in this function
		keyBytes := key.Marshal()
		pubkeyRecord := databases.PublickeyInfo{PubFp: hashAlg(keyBytes, nil)}
		clientSshVersion := databases.SshVersionInfo{Version: string(conn.ClientVersion())}
		_ = s.DbFd.CreateOrUpdateItemsInSeq(&pubkeyRecord, &clientSshVersion)
		_ = s.DbFd.CreateOrUpdateItemsInSeq(
			&databases.RemotePublickeyRelation{SessionId: sessionID, Pid: pubkeyRecord.PubId},
			&databases.RemoteSshverRelation{SessionId: sessionID, Sid: clientSshVersion.Id},
		)
		return nil, errors.New("public key authentication is not allowed")
	}

	sshConfig := &ssh.ServerConfig{
		ServerVersion:     getRandomSSHVersion(),
		PasswordCallback:  passwordFn,
		PublicKeyCallback: pubkeyFunc,
		NoClientAuth:      false, // request for basic authentication
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
		configs.Logger().Error(errInfo)
		return
	}
	defer func() { _ = sshConn.Close() }()
	/* reject all relay requests since all clients are untrusted */
	go ssh.DiscardRequests(relayReqs)
	for newChan := range chans {
		snapshot, ok := s.ConfOptions.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
		if !ok {
			continue
		} else if !snapshot.PermitLogin {
			_ = newChan.Reject(ssh.Prohibited, "Access Denied")
			continue
		}
		go s.handleNewSSHchan(sessionID, sshConn, newChan)
	}
}

func (s *SSHServConf) handleNewSSHchan(
	sessionId int64,
	sshConn *ssh.ServerConn,
	newChan ssh.NewChannel,
) {
	defer func() { _ = sshConn.Close() }()
	if newChan.ChannelType() != "session" {
		configs.Logger().Debug(newChan.ChannelType())
		_ = newChan.Reject(
			ssh.UnknownChannelType,
			"Unsupported channel type",
		)
		return
	}
	sshChan, reqs, err := newChan.Accept()
	if err == nil {
		// [TODO]: use configuration to determine which kind of mode we really need.
		//		1. talk with an LLM with tailored prompt
		s.mockShellForRemote(sessionId, sshConn, sshChan, reqs)
		return
	}
	payload := configs.GetLocalizedMsg(
		"services.SSHchannelAcceptanceError",
		map[string]any{
			"RemoteAddr": sshConn.RemoteAddr().String(),
			"ErrInfo":    err,
		},
	)
	configs.Logger().Error(payload)
}

func (s *SSHServConf) mockShellForRemote(
	sessionId int64,
	sshConn *ssh.ServerConn,
	sshChan ssh.Channel,
	reqs <-chan *ssh.Request,
) {
	defer func() {
		_, _ = sshChan.SendRequest(
			"exit-status", false,
			ssh.Marshal(&struct{ Status uint32 }{0}),
		)
		_ = sshChan.Close()
	}()
	for req := range reqs {
		switch req.Type {
		case "shell":
			_ = req.Reply(true, nil)
			snapshot, ok := s.ConfOptions.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
			if !ok {
				continue
			}
			var sb strings.Builder
			sb.WriteString(strings.ReplaceAll(snapshot.LoginBanner, "\n", "\r\n"))
			sb.WriteString("Last login: ")
			sb.WriteString(time.Now().Format(time.ANSIC))
			sb.WriteString("from ")
			sb.WriteString(sshConn.RemoteAddr().String())
			sb.WriteString("\r\n")
			_, _ = sshChan.Write([]byte(sb.String()))
			go s.cmdForwarding(sessionId, sshChan)
		case "pty-req":
			fallthrough
		case "env":
			fallthrough
		case "window-change":
			sshWinInfoDebug(req)
			fallthrough
		default:
			/*
				just accept 'pty-req' and 'window-change' without any action

				payload format of 'window-change':
					`uint32(rows)||uint32(cols)||uint32(width)||uint32(height)`
				here `||` means concatenate the information

				reject/abort all other requests like "exec"

				[TODO]: scp will send 'subsystem' as its pre-executed request.
					so it is worth wondering the payload sent by the attacker.
					To achieve this goal, the requirements are container and privilege deprivation
			*/
			_ = req.Reply(
				req.Type == "pty-req" || req.Type == "window-change",
				nil,
			)
		}
	}
}

// sshWinInfoDebug is used for checking the altering windows information during the
// SSH protocol processing.
func sshWinInfoDebug(req *ssh.Request) {
	var sb strings.Builder
	digitBrancher := func(x byte) {
		if x >= 10 {
			sb.WriteByte(x - 10 + 'a')
			return
		}
		sb.WriteByte(x + '0')
	}
	sb.WriteString(req.Type)
	sb.WriteString(" [")
	for i, p := range req.Payload {
		digitBrancher(p >> 4) // div
		digitBrancher(p & 15) // mod
		if i < len(req.Payload)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	configs.Logger().Debug(sb.String())
}

// cmdForwarding will create a mock shell for interaction
func (s *SSHServConf) cmdForwarding(
	sessionId int64, sshChan ssh.Channel,
) {
	term := terminal.NewShell(
		sshChan, sshChan,
		terminal.ShellRules{
			DefaultPrompts: "$ ",
			PendingPrompts: "> ",
			NeedHijackCmd:  true,
			LFisCRLF:       false,
			FullCRLF:       false,
		},
	)
	var shouldCease atomic.Bool
	shouldCease.Store(false)

	defer func() { _ = sshChan.Close() }()
	go func() {
		err := term.Run()
		defer shouldCease.Store(true)
		if err == nil {
			// send exit-status before quiting.
			_, _ = sshChan.SendRequest(
				"exit-status", false,
				ssh.Marshal(&struct{ Status uint32 }{0}),
			)
			_ = sshChan.Close()
			return
		}
		logInfo := configs.GetLocalizedMsg(
			"services.SSHwriteResponseError",
			map[string]any{"ErrInfo": err},
		)
		configs.Logger().Info(logInfo)
	}()

	for !shouldCease.Load() {
		currPayload := term.GetCurrCmd()
		if currPayload == nil || len(currPayload.Payload) == 0 {
			// no more available commands can be extracted from this current session
			break
		}
		/*
			TODO: Add hook for specific commands output like `uname -a`
				0. check if the configuration needs such modification
				1. inspect command, determine whether it matches the request or not
				2. once match, modify the return pattern
		*/
		snapshot, ok := s.ConfOptions.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
		if !ok {
			// use default method
			go term.SetCurrResp(currPayload)
		} else {
			respType := snapshot.ResponseType
			switch strings.ToLower(respType) {
			case "llm":
				// TODO: switch to easier mode if all tokens run up or utilized local LLM if possible...
			case "empty":
				go term.SetCurrResp(&terminal.ShellSyncObj{Ctx: currPayload.Ctx, Payload: ""})
			case "sandbox":
				// TODO container as sandbox
				fallthrough
			case "repeat":
				fallthrough
			default:
				go term.SetCurrResp(currPayload)
			}
		}
		go s.evalAndstoreUntrustCmd(sessionId, currPayload.Payload)
	}
}

func (s *SSHServConf) evalAndstoreUntrustCmd(
	sessionID int64,
	x string,
) {
	// [TODO]: filter the malform, meaningless commands
	//       alleviate the DB I/O pressure.
	//       consider the overhead of TTL, implementation below might be better to
	//       wrap with another go routine.
	//       concurrent control is required.

	// check if we can use LLM as the judger
	// else use default syntax AST parser to check if the command is executable
	// otherwise, the malform commands will stress the database ops

	// if cmdNeedFilterOut(currPayload.Payload) { continue }

	cmdText := &databases.CommandInfo{Cmd: x}
	_ = s.DbFd.CreateOrUpdateItem(cmdText, cmdText)
	tmp := &databases.RemoteCommandRelation{
		SessionId: sessionID,
		Cid:       cmdText.CommandId,
	}
	_ = s.DbFd.CreateOrUpdateItem(tmp, tmp)
}
