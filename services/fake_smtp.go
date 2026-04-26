package services

/*
	SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

an SMTP server runs at port 25, SMTPS server is at 465
references:
	https://github.com/phin3has/mailoney
	https://github.com/python/cpython/blob/3.14/Lib/smtplib.py
	https://github.com/mhale/smtpd/blob/master/smtpd.go
	https://github.com/masa23/mmauth

[TODO]: https://github.com/stophobia/claude-code-guide2/blob/main/skills/smtp-penetration-testing/SKILL.md
*/

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/databases"
	"b0gus/terminal"
	"bytes"
	"context"
	//"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	//"net/textproto"
	"strings"
	"sync/atomic"
	"time"
)

type (
	SMTPLoginStatus int
	SMTPCmdStatus   int
	SMTPRespStatus  int
)

const (
	SMTPLoginInvalid SMTPLoginStatus = iota
	SMTPLoginNameRequired
	SMTPLoginPasswordRequired
)

const (
	SMTPOtherRequired SMTPCmdStatus = iota
	SMTPMailRequired
	SMTPRcptRequired
	SMTPDataRequired
)

const (
	SMTPsysStatus             SMTPRespStatus = 211
	SMTPHelpMsg               SMTPRespStatus = 214
	SMTPServReady             SMTPRespStatus = 220
	SMTPServClosing           SMTPRespStatus = 221
	SMTPOk                    SMTPRespStatus = 250
	SMTPWillForward           SMTPRespStatus = 251
	SMTPCantVerify            SMTPRespStatus = 252
	SMTPOngoingAuth           SMTPRespStatus = 334
	SMTPStartMailInp          SMTPRespStatus = 354
	SMTPServUnavail           SMTPRespStatus = 421
	SMTPMailboxTempUnavail    SMTPRespStatus = 450
	SMTPLocalErr              SMTPRespStatus = 451
	SMTPInsufficientStorage   SMTPRespStatus = 452
	SMTPApplyParamFail        SMTPRespStatus = 455
	SMTPCmdSyntaxErr          SMTPRespStatus = 500
	SMTPParamsSyntaxErr       SMTPRespStatus = 501
	SMTPCmdNotImpl            SMTPRespStatus = 502
	SMTPBadSequence           SMTPRespStatus = 503
	SMTPParamsNotImpl         SMTPRespStatus = 504
	SMTPAuthErr               SMTPRespStatus = 535
	SMTPMailUnavail           SMTPRespStatus = 550
	SMTPTryForwarding         SMTPRespStatus = 551
	SMTPExceedStorage         SMTPRespStatus = 552
	SMTPNotAllowedMailboxName SMTPRespStatus = 553
	SMTPTransactionFail       SMTPRespStatus = 554
	SMTPMailRcptParmNotParsed SMTPRespStatus = 555
)

type SMTPClientCtx struct {
	ctx         context.Context
	term        *terminal.Shell
	LoginStatus SMTPLoginStatus
	CmdStatus   SMTPCmdStatus
	Authorized  bool
	AlreadyTLS  bool
}

// SendResp will send response of one command as input.
// Note that When requiring sending multiple response for one command, the context should set normal for the
// first one, while the others have to be kept as nil. Otherwise, the whole service will be stuck by the full channel.
func (s *SMTPClientCtx) SendResp(
	ctx context.Context,
	code SMTPRespStatus, resp string,
) {
	payload := fmt.Sprintf("%d %s", code, resp)
	// unexpected output behavior
	if ctx == nil {
		_, _ = s.term.GetWriter().Write(append([]byte(payload), []byte("\r\n")...))
	} else {
		s.term.SetCurrResp(&terminal.ShellSyncObj{Ctx: ctx, Payload: payload})
	}
}

func (s *SMTPClientCtx) EHLOhandler(ctx context.Context) {
	s.SendResp(ctx, SMTPOk, "Hello")
	supported := []string{"STARTTLS", "DSN", "ETRN", "8BITMIME", "AUTH PLAIN LOGIN", "SMTPUTF8"}
	rand.Shuffle(len(supported), func(i, j int) {
		supported[i], supported[j] = supported[j], supported[i]
	})
	for _, t := range supported {
		// the design is only one command mapped to one response
		// since outside of loop, the quota is run up
		// here the logic requires using nil to represent current context.
		s.SendResp(nil, SMTPOk, t)
	}
}

func (s *SMTPClientCtx) mailHandler() {
	// from, exist
	// opt<to, cc/bcc>
	// opt<subj>
	// opt<date>
	// ext<mal-info>/<trust-info>
	// new-break-line
	// [content]
	// [.]
}

func (s *SMTPClientCtx) LoginHandler(
	ctx context.Context,
	db *configs.RuntimeDB, input string,
) bool {
	switch s.LoginStatus {
	case SMTPLoginNameRequired:
		_ = crypto_aux.Base64Convert([]byte(input))
		s.LoginStatus = SMTPLoginPasswordRequired
		payload, err := crypto_aux.Base64Recover("password")
		if err != nil {
			s.SendResp(ctx, SMTPCantVerify, "Authentication failed")
			return false
		}
		cast, err := crypto_aux.Base64Recover(input)
		if err != nil {
			// unable to cast from base64 string
			s.SendResp(ctx, SMTPCantVerify, "Authentication failed")
			return false
		}
		t := &databases.UsernameInfo{Name: string(cast)}
		_ = db.CreateOrUpdateItem(t, t)
		s.SendResp(ctx, SMTPOngoingAuth, string(payload))
		return false
	case SMTPLoginPasswordRequired:
		original, err := crypto_aux.Base64Recover(input)
		if err != nil {
			s.SendResp(ctx, SMTPCantVerify, "Authentication failed")
			return false
		}
		s.Authorized = true
		t := &databases.PasswordInfo{Password: string(original)}
		_ = db.CreateOrUpdateItem(t, t)
		s.LoginStatus = SMTPLoginInvalid
		return true
	default:
		s.SendResp(ctx, SMTPParamsNotImpl, "Authentication failed")
		return false
	}
}

func (s *SMTPClientCtx) PlainHandler(
	ctx context.Context,
	db *configs.RuntimeDB, input string,
) bool {
	original, err := crypto_aux.Base64Recover(input)
	if err != nil {
		s.SendResp(ctx, SMTPCantVerify, "Authentication failed")
		return false
	}
	parts := bytes.Split(original, []byte{0})
	if len(parts) < 3 {
		s.SendResp(ctx, SMTPCantVerify, "Authentication failed")
		return false
	}
	s.Authorized = true
	_ = db.CreateOrUpdateItemsInSeq([]configs.DBstruct{
		&databases.UsernameInfo{Name: string(parts[1])},
		&databases.PasswordInfo{Password: string(parts[2])},
	}...)
	s.LoginStatus = SMTPLoginInvalid
	return true
}

func (s *SMTPClientCtx) upgradeToTLS(
	currSMTPconf configs.SMTPconfig,
	conn net.Conn,
) (*tls.Conn, error) {

	cert, err := tls.LoadX509KeyPair(
		currSMTPconf.LocalTLSCertPath,
		currSMTPconf.LocalTLSKeyPath,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"unable to load cert due to err: %v", err,
		)
	}
	tlsConfig := &tls.Config{ // TLS here requires real CA-signed certificates
		Certificates:           []tls.Certificate{cert},
		MinVersion:             tls.VersionTLS12,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		SessionTicketsDisabled: true,
	}
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("handshake failure: %v", err)
	}
	// [TODO]: shall we record? And how shall we record?
	// state := tlsConn.ConnectionState()
	// tls.CipherSuiteName(state.CipherSuite)
	// tls.VersionName(state.Version)
	return tlsConn, nil
}

//var (
//	MailCmdToBeMatched = regexp.MustCompile(`^(?i)mail from\s*:\s*<([^>]+)>`)
//	RcptCmdToBeMatched = regexp.MustCompile(`^(?i)rcpt to\s*:\s*<([^>]+)>`)
//)

func (s *SMTPClientCtx) CommandDispatcher(
	ctx context.Context,
	confObj *atomic.Pointer[configs.LocalConfig],
	db *configs.RuntimeDB, line string,
) any {
	assembleCmd := strings.SplitN(line, " ", 3)
	if len(assembleCmd) < 1 {
		s.SendResp(ctx, SMTPCmdNotImpl, "invalid command")
		return SMTPCmdNotImpl
	}
	// todo: encode command sequence and possible arguments for each command
	cmdPref := strings.ToLower(assembleCmd[0])
	currSMTPconf, ok := confObj.Load().SelectTerm(configs.SMTPEnum).(configs.SMTPconfig)
	if !ok {
		s.SendResp(ctx, SMTPServUnavail, "SMTP service unavailable")
		return SMTPServUnavail
	}
	switch cmdPref {
	case "helo":
		s.SendResp(ctx, SMTPOk, "Hello")
	case "ehlo":
		// The EHLO command operates and can be used in the same way as the HELO command.
		// However, it additionally requests that the returned reply should identify specific
		// SMTP service extensions that are supported by the SMTP server.
		s.EHLOhandler(ctx)
	case "etrn":
		// handle for the clients' email queue?
		if currSMTPconf.AuthRequired && !s.Authorized {
			s.SendResp(ctx, SMTPAuthErr, "yet to be authorized")
			return SMTPAuthErr
		}
		s.SendResp(ctx, SMTPOk, "queueing started")
	case "auth":
		if len(assembleCmd) < 3 {
			s.SendResp(ctx,
				SMTPCmdSyntaxErr,
				"invalid AUTH command. required format: AUTH <METHOD> CORRESPONDING-INPUT",
			)
			return SMTPCmdSyntaxErr
		}
		var calcFn func(context.Context, *configs.RuntimeDB, string) bool
		// [TODO]: buggy implementation
		switch strings.ToLower(assembleCmd[1]) {
		case "plain": // AUTH PLAIN <base64(\0Username\0Password)>
			calcFn = s.PlainHandler
		case "login":
			calcFn = s.LoginHandler
		default:
			s.SendResp(ctx, SMTPCmdNotImpl, "invalid command")
			return SMTPCmdNotImpl
		}
		if s.Authorized {
			// such situation is like resetting current account
			s.Authorized = false
			s.LoginStatus = SMTPLoginNameRequired
			s.CmdStatus = SMTPOtherRequired
		}
		success := calcFn(ctx, db, assembleCmd[2])
		if success {
			s.SendResp(ctx, SMTPOk, "authentication successful")
			s.Authorized = true
			return SMTPOk
		}
		return SMTPCantVerify
	case "starttls":
		if s.AlreadyTLS {
			s.SendResp(ctx, SMTPApplyParamFail, "already TLS enabled")
			return SMTPApplyParamFail
		}
		if len(currSMTPconf.LocalTLSCertPath) == 0 || len(currSMTPconf.LocalTLSKeyPath) == 0 {
			s.SendResp(ctx, SMTPApplyParamFail, "unable to set up TLS connection at present")
			return SMTPApplyParamFail
		}
		s.SendResp(ctx, SMTPServReady, "Ready to start TLS")
		conn, ok := s.term.GetWriter().(net.Conn) // writer is raw without wrapping.
		if !ok {
			return errors.New("unable to cast to net.Conn")
		}
		tlsConn, err := s.upgradeToTLS(currSMTPconf, conn)
		if err != nil || tlsConn == nil {
			s.SendResp(ctx, SMTPServUnavail, "unable to upgrade to TLS")
			return err
		}
		s.term.AlterIOsrc(tlsConn, tlsConn)
		// s.Writer = bufio.NewWriter(tlsConn)
		// s.Reader = textproto.NewReader(bufio.NewReader(tlsConn))
		s.AlreadyTLS = true
	case "rset":
		s.Authorized = false
		s.LoginStatus = SMTPLoginNameRequired
		s.CmdStatus = SMTPOtherRequired
		s.SendResp(ctx, SMTPOk, "reset OK")
	case "noop":
		s.SendResp(ctx, SMTPOk, "OK")
	case "quit":
		s.SendResp(ctx, SMTPOk, "Bye")
		return nil
	case "expn", "help":
		// The EXPN (expand) command expands a mailing list defined on the host where SMTP is running.
		s.SendResp(ctx, SMTPHelpMsg, "Help")
		s.SendResp(nil, SMTPHelpMsg, "noop    - test current connection")
		s.SendResp(nil, SMTPHelpMsg, "rset    - reset current connection")
		s.SendResp(nil, SMTPHelpMsg, "quit    - quit")
	case "mail":
		// if currSMTPconf.AuthRequired && !s.Authorized {
		// 	_ = s.SendResp(SMTPAuthErr, "Yet to be authorized")
		// 	return SMTPAuthErr
		// }
		// if s.CmdStatus != SMTPOtherRequired {
		// 	s.CmdStatus = SMTPOtherRequired
		// 	_ = s.SendResp(SMTPBadSequence, "Invalid operation, reset ")
		// 	return SMTPBadSequence
		// }
		// // look like regex match will be better for such a situation
		// currTest := MailCmdToBeMatched.FindStringSubmatch(line)
		// if len(currTest) != 2 {
		// 	_ = s.SendResp(SMTPCmdSyntaxErr, "Invalid `mail from`")
		// 	return SMTPBadSequence
		// }
		// // currTest[1] // as source email, but have to validate
		// // temporarily drop here.
		// s.CmdStatus = SMTPMailRequired
		// _ = s.SendResp(SMTPOk, "Ok")
		fallthrough
	case "rcpt":
		// if currSMTPconf.AuthRequired && !s.Authorized {
		// 	_ = s.SendResp(SMTPAuthErr, "Yet to be authorized")
		// 	return SMTPAuthErr
		// }
		// if s.CmdStatus != SMTPMailRequired && s.CmdStatus != SMTPRcptRequired {
		// 	_ = s.SendResp(SMTPBadSequence, "Yet not to set `mail from`, abort")
		// 	return SMTPBadSequence
		// }
		// // so does the rcpt command. must match the word `to`
		// currTest := RcptCmdToBeMatched.FindStringSubmatch(line)
		// if len(currTest) != 2 {
		// 	_ = s.SendResp(SMTPCmdSyntaxErr, "Invalid `rcpt to`")
		// 	return SMTPCmdSyntaxErr
		// }
		// s.CmdStatus = SMTPRcptRequired
		// _ = s.SendResp(SMTPOk, "Ok")
		// // currTest[1] // as remote receiver's email
		// // one mail can have multiple receivers
		fallthrough
	case "data":
		// if currSMTPconf.AuthRequired && !s.Authorized {
		// 	_ = s.SendResp(SMTPAuthErr, "Yet to be authorized")
		// 	return SMTPAuthErr
		// }
		// if s.CmdStatus != SMTPRcptRequired {
		// 	_ = s.SendResp(SMTPCmdSyntaxErr, "Invalid DATA before MAIL FROM and RCPT TO")
		// 	return SMTPCmdSyntaxErr
		// }
		// _ = s.SendResp(SMTPStartMailInp, "End data with <CR><LF>.<CR><LF>")
		// // read until "." or network connection becomes broken
		// s.DataHandler()
		// s.CmdStatus = SMTPOtherRequired
		fallthrough
	case "vrfy":
		// VRFY {root, bin, admin, ...}
		// whether email or mail-list exists in current mail-server.
		// treat as username?
		fallthrough
	default:
		// drop argument. Still beyond reproach
		s.SendResp(ctx, SMTPCmdNotImpl, fmt.Sprintf("unrecognized command: %s", cmdPref))
		return SMTPCmdNotImpl
	}
	return SMTPOk
}

type SMTPServConf struct {
	DbFd        *configs.RuntimeDB // database handler for writing data.
	ConfOptions *atomic.Pointer[configs.LocalConfig]
	term        *terminal.Shell
}

func (s *SMTPServConf) ServeHTTP(http.ResponseWriter, *http.Request) {}

// InvokeForTCPtask uses for executing TCP-typed Business
func (s *SMTPServConf) InvokeForTCPtask(conn net.Conn) {
	// somehow, it is acceptable to send the response to AI.
	defer func() { _ = conn.Close() }()
	s.term = terminal.NewShell(
		conn, conn, true,
		"> ", "",
		true, false,
	)
	var shouldCease atomic.Bool
	shouldCease.Store(false)
	go func() {
		err := s.term.Run()
		shouldCease.Store(true)
		if err != nil {
			var sb strings.Builder
			sb.WriteString("<SMTP>: ")
			sb.WriteString(err.Error())
			configs.Logger.Error(sb.String())
		}
	}()

	currCtx := SMTPClientCtx{
		term:        s.term,
		Authorized:  false,
		LoginStatus: SMTPLoginNameRequired,
	}
	_, ok := conn.(*tls.Conn)
	currCtx.AlreadyTLS = ok
	currCtx.SendResp(nil, SMTPServReady, time.Now().Format(time.RFC850))
	currCtx.EHLOhandler(nil)
	// [TODO]: When should the service terminate?
	for !shouldCease.Load() {
		tmp := s.term.GetCurrCmd()
		if tmp == nil {
			break
		}
		line := strings.TrimSpace(tmp.Payload)
		if len(line) == 0 {
			currCtx.SendResp(tmp.Ctx, SMTPCmdNotImpl, "Unrecognized command")
			continue
		}
		res := currCtx.CommandDispatcher(tmp.Ctx, s.ConfOptions, s.DbFd, line)
		if res == nil {
			break
		}
		switch res.(type) {
		case error:
			var sb strings.Builder
			sb.WriteString("<SMTP>: ")
			sb.WriteString(conn.RemoteAddr().String())
			sb.WriteString(" ")
			sb.WriteString(res.(error).Error())
			configs.Logger.Error(sb.String())
		}
	}
}

// InvokeForUDPtask uses for executing UDP-typed Business
func (s *SMTPServConf) InvokeForUDPtask(net.Addr, []byte) []byte { return nil }

// InvokeForICMPtask uses for executing ICMP-typed Business,
// merely on **unix-system**
func (s *SMTPServConf) InvokeForICMPtask(net.Addr, []byte) {}

// Run will execute for SMTP/SMTPS.
// When the `confObj.SMTPconfig.NaturalTLS` is true, the function will run SMTPS for the first time.
func (s *SMTPServConf) Run(
	confObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB,
	args ...any,
) {
	defer func() {
		configs.Logger.Info(configs.GetLocalizedMsg(
			"services.SMTPQuitInfo", nil,
		))
	}()
	if len(args) != 1 {
		payload := configs.GetLocalizedMsg(
			"services.SMTPWrongParamNumErr",
			map[string]any{
				"Expect": 1,
				"Actual": len(args),
			},
		)
		configs.Logger.Info(payload)
		return
	} else if confObj == nil {
		configs.Logger.Error(configs.GetLocalizedMsg(
			"services.SMTPNullConfErr", nil,
		))
		return
	}
	_, ok := confObj.Load().SelectTerm(configs.SMTPEnum).(configs.SMTPconfig)
	if !ok {
		return
	}
	s.ConfOptions, s.DbFd = confObj, db
	_ = db.CreateTable(
		&databases.UsernameInfo{}, &databases.PasswordInfo{},
	)
	var clientAux ReentrantNetType
	clientAux.Init(configs.SMTPEnum, s)
	go clientAux.EventMonitor(confObj, scc)
	clientAux.AlterNetFd(TCPEnum, confObj)
}
