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
	"strings"

	//"net/smtp"
	//"net/textproto"
	"sync"
	"sync/atomic"
)

type SMTPServConf struct {
	clientLimitor                 atomic.Uint32
	tcpListenerGuard, configGuard sync.RWMutex
	DbFd                          *configs.RuntimeDB // database handler for writing data.
	ConfOptions                   *configs.SMTPconfig
}

type SMTPLoginStatus int
type SMTPStatusCode int

const (
	SMTPLoginInvalid SMTPLoginStatus = iota
	SMTPLoginName
	SMTPLoginPassword
)

const (
	SMTPIndirect SMTPStatusCode = iota
)

func (s *SMTPServConf) upgradeToTLS(conn net.Conn) (*tls.Conn, error) {
	cert, err := tls.LoadX509KeyPair(
		s.ConfOptions.LocalTLSCertPath,
		s.ConfOptions.LocalTLSKeyPath,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to load cert due to err: %v", err,
		)
	}
	// [TODO]: TLS here requires real CA-signed certificates
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("handshake failure: %v", err)
	}
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

func (s *SMTPClientCtx) LoginHandler(input string) bool {
	// [TODO]: do we really need base64?
	switch s.LoginStatus {
	case SMTPLoginName:
		username, err := crypto_aux.Base64Serialize(input)
		if err != nil {
			return false
		}
		s.username = string(username) // TODO: login to backend
		s.LoginStatus = SMTPLoginPassword
		_ = s.SendResp(334, crypto_aux.Base64Deserialize([]byte("password"))+"\r\n")
		return false
	case SMTPLoginPassword:
		password, err := crypto_aux.Base64Serialize(input)
		if err != nil {
			return false
		}
		s.Authorized = true
		// TODO
		fmt.Println(password)
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

func (s *SMTPServConf) clientConnHandler(conn net.Conn) {
	if s.clientLimitor.Load() > 1 {
		// abort this connection due to excess of threshold
		_ = conn.Close()
		return
	}
	s.clientLimitor.Add(1)
	defer func() {
		s.clientLimitor.Add(^uint32(0))
		_ = conn.Close()
	}()
	// TODO: set deadline for every smtp client.
	// err := conn.SetDeadline(time.Now().Add())
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
		if s.ConfOptions.AuthRequired && !currCtx.Authorized {
			success := currCtx.LoginHandler(line)
			if success {
				_ = currCtx.SendResp(235, "Authentication successful")
				currCtx.Authorized = true
			} else {
				_ = currCtx.SendResp(535, "Authentication failed")
			}
			continue
		}

		// TODO: standard status code and response
		//  strategy for remote email sending and attachment splitting isolation
		assembleCmd := strings.SplitN(line, " ", 2)
		cmd := strings.ToLower(assembleCmd[0])
		args := ""
		if len(assembleCmd) > 1 {
			args = assembleCmd[1]
		}
		switch cmd {
		case "helo":
			_ = currCtx.SendResp(250, fmt.Sprintf("Hello %s", args))
		case "ehlo":
		case "auth":
			if currCtx.Authorized {
				_ = currCtx.SendResp(503, "Already authenticated (disconnect to switch user)")
				continue
			}
		case "starttls":
			err := currCtx.SendResp(220, "Ready to start TLS")
			if err != nil {

			}
			tlsConn, err := s.upgradeToTLS(conn)
			if err != nil {
				continue
			}
			// TODO: maintain tlsConn
			currCtx.Writer = bufio.NewWriter(tlsConn)
			currCtx.Reader = textproto.NewReader(bufio.NewReader(tlsConn))
			_ = tlsConn.Close()
		case "plain":
		case "login":
		case "mail":
		case "rcpt":
		case "data":
		case "quit":
			return
		default:
			_ = currCtx.SendResp(500, fmt.Sprintf("Unrecognized command: %s", cmd))
		}
	}
}

func (s *SMTPServConf) SMTPMaliciousClientHandler(scc *configs.ServConcurrentCtrl) {
	var endSign atomic.Value
	endSign.Store(false)
	listener, err := net.Listen(
		"tcp", fmt.Sprintf(":%d", s.ConfOptions.ListenPort),
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
	}
	// TODO: db.CreateTable()
	s.ConfOptions = smtpConfObj
	s.DbFd = db
	s.SMTPMaliciousClientHandler(scc)
}
