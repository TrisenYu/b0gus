package services

// an SMTP server runs at port 25
// reference: https://github.com/phin3has/mailoney

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"

	//"net/smtp"
	//"net/textproto"
	"sync"
	"sync/atomic"
)

type SMTPServConf struct {
	clientLimitor                 atomic.Uint32
	tcpListenerGuard, configGuard sync.RWMutex
	DbFd                          *configs.RuntimeDB // database handler for writing data.
	SMTPConfOptions               *configs.SMTPconfig
	SMTPSconfOptions              *configs.SMTPconfig
}

type (
	SMTPLoginStatus int
	SMTPCmdStatus   int
)

const (
	SMTPLoginInvalid SMTPLoginStatus = iota
	SMTPLoginName
	SMTPLoginPassword
)

const (
	SMTPOtherRequired SMTPCmdStatus = iota
	SMTPMailRequired
	SMTPRcptRequired
	SMTPDataRequired
)

// TODO.
func (s *SMTPServConf) upgradeToTLS(conn net.Conn) (*tls.Conn, error) {
	cert, err := tls.LoadX509KeyPair(
		s.SMTPConfOptions.LocalTLSCertPath,
		s.SMTPConfOptions.LocalTLSKeyPath,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to load cert due to err: %v", err,
		)
	}
	// [TODO]: TLS here requires real CA-signed certificates
	tlsConfig := &tls.Config{
		Certificates:           []tls.Certificate{cert},
		MinVersion:             tls.VersionTLS12,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		SessionTicketsDisabled: true,
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("handshake failure: %v", err)
	}
	// state := tlsConn.ConnectionState()
	// tls.CipherSuiteName(state.CipherSuite)
	// tls.VersionName(state.Version)
	return tlsConn, nil
}

func (s *SMTPServConf) parseMIMEContent(rawContent string) {
	parts := strings.SplitN(rawContent, "\r\n\r\n", 2)
	if len(parts) < 2 {
		return
	}
	headerPart := parts[0]
	bodyPart := parts[1]

	mediaType, params, err := mime.ParseMediaType(headerPart)
	if err != nil {

	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return
	}
	boundary := params["boundary"]
	reader := multipart.NewReader(strings.NewReader(bodyPart), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			// TODO
			continue
		}

		contentType := part.Header.Get("Content-Type")
		filename := part.FileName()
		contentDisposition := part.Header.Get("Content-Disposition")
		content, err := io.ReadAll(part)
		if err != nil {
			_ = part.Close()
			continue
		}
		if filename == "" && !strings.Contains(contentDisposition, "attachment") {
			// text content
			_ = part.Close()
			continue
		}
		fmt.Println(contentType, content, filename)
		// TODO
		// os.WriteFile(filename, content, 0644)
		_ = part.Close()
	}
}

type SMTPClientCtx struct {
	Reader      *textproto.Reader
	Writer      *bufio.Writer
	Authorized  bool
	LoginStatus SMTPLoginStatus
	CmdStatus   SMTPCmdStatus
	username    string
}

func (s *SMTPClientCtx) SendResp(code int, resp string) error {
	payload := fmt.Sprintf("%d %s\r\n", code, resp)
	_, err := s.Writer.WriteString(payload)
	if err != nil {
		return err
	}
	return s.Writer.Flush()
}

func (s *SMTPClientCtx) EHLOhandler() {
	_ = s.SendResp(250, "Hello")
	// TODO: randomized response sequences
	todos := []string{"STARTTLS", "DSN", "ETRN", "SMTPUTF8", "8BITMIME"}
	for _, t := range todos {
		_ = s.SendResp(250, t)
	}
}

func (s *SMTPClientCtx) EHLOHandler() {}

func (s *SMTPClientCtx) LoginHandler(input string) bool {
	// unfortunately, base64 is must.
	switch s.LoginStatus {
	case SMTPLoginName:
		_ = crypto_aux.Base64Deserialize([]byte(input))
		// s.username = string(username) // TODO: login to backend
		s.LoginStatus = SMTPLoginPassword
		payload, err := crypto_aux.Base64Serialize("password")
		if err != nil {

			return false
		}
		_ = s.SendResp(334, string(payload))
		return false
	case SMTPLoginPassword:
		_ = crypto_aux.Base64Deserialize([]byte(input))
		s.Authorized = true
		// TODO password

		s.LoginStatus = SMTPLoginInvalid
		return true
	default:
		return false
	}
}

/*
references:
	https://github.com/python/cpython/blob/3.14/Lib/smtplib.py
	https://github.com/mhale/smtpd/blob/master/smtpd.go
*/

var (
	MailCmdToBeMatched = regexp.MustCompile(`^(?i)mail from\s*:\s*<([^>]+)>`)
	RcptCmdToBeMatched = regexp.MustCompile(`^(?i)rcpt to\s*:\s*<([^>]+)>`)
)

func (s *SMTPServConf) clientConnHandler(conn net.Conn) {
	if s.clientLimitor.Load() > 1 {
		// abort this connection due to excess of threshold
		_ = conn.Close()
		return
	}
	s.clientLimitor.Add(1)
	currConn := conn
	defer func() {
		s.clientLimitor.Add(^uint32(0))
		_ = currConn.Close()
	}()
	// TODO: set deadline for every smtp client.
	err := conn.SetDeadline(
		time.Now().Add(time.Duration(s.SMTPConfOptions.ClientConnTimeout) * time.Second),
	)
	if err != nil {
		return
	}
	// ip, port := misc_utils.IPaddrSplit(conn.RemoteAddr().String())
	currCtx := SMTPClientCtx{
		Reader:      textproto.NewReader(bufio.NewReader(conn)),
		Writer:      bufio.NewWriter(conn),
		Authorized:  false,
		LoginStatus: SMTPLoginName,
	}

	// TODO: find reply code dictionary
	_ = currCtx.SendResp(220, "")
	for {
		line, err := currCtx.Reader.ReadLine()
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// decide whether to authorize
		if s.SMTPConfOptions.AuthRequired && !currCtx.Authorized {
			success := currCtx.LoginHandler(line)
			if success {
				_ = currCtx.SendResp(235, "Authentication successful")
				currCtx.Authorized = true
			} else {
				_ = currCtx.SendResp(535, "Authentication failed")
			}
			continue
		}

		assembleCmd := strings.SplitN(line, " ", 1)
		if len(assembleCmd) < 1 {
			_ = currCtx.SendResp(535, "Invalid command")
			continue
		}
		cmdPref := strings.ToLower(assembleCmd[0])
		switch cmdPref {
		case "helo":
			_ = currCtx.SendResp(250, "Hello")
		case "ehlo":
			currCtx.EHLOhandler()
		case "etrn":
			// handle for the clients' email queue?
			_ = currCtx.SendResp(250, "Queueing started")
		case "auth":
			_ = currCtx.SendResp(503, "Already authenticated")
			continue
		case "starttls":
			if len(s.SMTPConfOptions.LocalTLSCertPath) == 0 || len(s.SMTPConfOptions.LocalTLSKeyPath) == 0 {
				// TODO:
				continue
			}
			err := currCtx.SendResp(220, "Ready to start TLS")
			if err != nil {
				continue
			}
			tlsConn, err := s.upgradeToTLS(conn)
			if err != nil {
				continue
			}
			currCtx.Writer = bufio.NewWriter(tlsConn)
			currCtx.Reader = textproto.NewReader(bufio.NewReader(tlsConn))
			currConn = tlsConn
		case "plain":

		case "login":

		case "mail":
			// peek the next word, `from` or `from:`
			// look like

			// regex match will be better for such situation
			if currCtx.CmdStatus != SMTPOtherRequired {
				currCtx.CmdStatus = SMTPOtherRequired
				_ = currCtx.SendResp(535, "Invalid operation, reset ")
			}
			currTest := MailCmdToBeMatched.FindStringSubmatch(line)
			if len(currTest) != 2 {
				_ = currCtx.SendResp(535, "Invalid `mail from`")
				continue
			}
			// currTest[1] // as source email, but have to inspect the validation

			currCtx.CmdStatus = SMTPMailRequired
		case "rcpt":
			if currCtx.CmdStatus != SMTPMailRequired {
				_ = currCtx.SendResp(535, "Yet not to set `mail from`, abort")
				continue
			}
			currTest := RcptCmdToBeMatched.FindStringSubmatch(line)
			if len(currTest) != 2 {
				_ = currCtx.SendResp(535, "Invalid `rcpt to`")
				continue
			}
			currCtx.CmdStatus = SMTPRcptRequired
			// so does the rcpt command. must match the word `to`
		case "vrfy", "expn":
			// VRFY {root, bin, admin, ...}
			// whether email or mail-list exists in current mail-server.
			// treat as username?
		case "data":
			if currCtx.CmdStatus != SMTPRcptRequired {

			}
		case ".":
			if currCtx.CmdStatus != SMTPDataRequired {
				_ = currCtx.SendResp(500, fmt.Sprintf("Unrecognized command: %s", cmdPref))
				continue
			}
		case "noop":
			_ = currCtx.SendResp(250, "OK")
		case "quit":
			_ = currCtx.SendResp(250, "Bye")
			return
		default:
			_ = currCtx.SendResp(500, fmt.Sprintf("Unrecognized command: %s", cmdPref))
		}
	}
}

func (s *SMTPServConf) SMTPSClientHandler(scc *configs.ServConcurrentCtrl) {
	cert, err := tls.LoadX509KeyPair(
		s.SMTPSconfOptions.LocalTLSCertPath,
		s.SMTPSconfOptions.LocalTLSKeyPath,
	)
	var sb strings.Builder
	if err != nil {
		sb.WriteString("unable to load cert due to err: ")
		sb.WriteString(err.Error())
		configs.Logger.Error(sb.String())
		return
	}
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(int(s.SMTPSconfOptions.ListenPort)))
	listener, err := tls.Listen(
		"tcp", sb.String(),
		&tls.Config{
			Certificates:           []tls.Certificate{cert},
			MinVersion:             tls.VersionTLS12,
			ClientAuth:             tls.RequireAndVerifyClientCert,
			SessionTicketsDisabled: true,
		},
	)
	if err != nil {
		configs.Logger.Error("unable to set up server on port 465!")
		return
	}
	defer func() { _ = listener.Close() }()
	// we won't use STARTTLS in current function

}

func (s *SMTPServConf) SMTPMaliciousClientHandler(scc *configs.ServConcurrentCtrl) {
	var endSign atomic.Value
	endSign.Store(false)
	listener, err := net.Listen(
		"tcp", fmt.Sprintf(":%d", s.SMTPConfOptions.ListenPort),
	)
	if err != nil {
		// TODO
		return
	}
	defer func() { _ = listener.Close() }()
	s.clientLimitor.Store(0)
	go func() {
	stuck:
		select {
		case <-scc.Ctx.Done():
		case castedDatum := <-scc.DataCh:
			switch tmp := castedDatum.(type) {
			case nil:
			case configs.SMTPconfig:
				payload := fmt.Sprintf("%v", tmp)
				configs.Logger.Info(payload)
			case *configs.SMTPconfig:
			default:
				goto stuck
			}
		}
		endSign.Store(true)
		_ = listener.Close()
	}()
keepSpinning:
	if endSign.Load().(bool) {
		// TODO logger
		return
	}
	inConn, err := listener.Accept()
	if err != nil {
		// TODO logger
		goto keepSpinning
	}
	go s.clientConnHandler(inConn)
	goto keepSpinning
}

func (s *SMTPServConf) Run(
	smtpConfObj *configs.SMTPconfig,
	scc *configs.ServConcurrentCtrl,
	db *configs.RuntimeDB,
	args ...any,
) {
	defer func() {
		configs.Logger.Info("SMTP service has quited")
	}()
	// TODO: complete the implementation and set 0 to 1.
	if len(args) != 0 {
		configs.Logger.Info(fmt.Sprintf("too many arguments<%d> for smtp", len(args)))
		return
	} else if smtpConfObj == nil {
		configs.Logger.Error("empty configuration is provided")
		return
	}
	// TODO: db.CreateTable() and SMTPSConfOptions
	s.SMTPConfOptions = smtpConfObj
	s.DbFd = db
	if s.SMTPSconfOptions != nil {
		go s.SMTPSClientHandler(scc)
	}
	s.SMTPMaliciousClientHandler(scc)
}
