// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2026
// Created at 2026/03/10 星期二 10:54:48
// Last modified at 2026/03/14 星期六 23:05:07
package main

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type SMTPServer struct {
}

// A Session is returned after successful login.
type Session struct {
	auth bool
}

// AuthMechanisms returns a slice of available auth mechanisms; only PLAIN is
// supported in this example.
func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

// Auth is the handler for supported authenticators.
func (s *Session) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(
		func(identity, username, password string) error {
			if username != "username" || password != "password" {
				return errors.New("invalid username or password")
			}
			s.auth = true
			return nil
		},
	), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	log.Println("Mail from:", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	log.Println("Rcpt to:", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if !s.auth {
		return smtp.ErrAuthRequired
	}
	if b, err := io.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (s *Session) Reset()        {}
func (s *Session) Logout() error { return nil }

// NewSession is called after client greeting (EHLO, HELO).
func (bkd *SMTPServer) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{}, nil
}

// It can be tested manually with e.g. netcat:
//
//	> netcat -C localhost 1025
//	EHLO localhost
//	AUTH PLAIN
//	AHVzZXJuYW1lAHBhc3N3b3Jk
//	MAIL FROM:<root@nsa.gov>
//	RCPT TO:<root@gchq.gov.uk>
//	DATA
//	Hey <3
//	.
func main() {
	be := &SMTPServer{}
	stmpServ := smtp.NewServer(be)
	stmpServ.Addr = ":2525"
	defer func() { _ = stmpServ.Close() }()
	if err := stmpServ.ListenAndServe(); err != nil {
		fmt.Printf("encounter an error: %v", err)
		return
	}
}
