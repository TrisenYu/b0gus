package main

import (
	"b0gus/terminal"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

//	var mockCommands = []string{
//		"ls", "cd", "pwd", "echo", "cat", "mkdir", "rm", "cp", "mv",
//		"grep", "find", "ps", "kill", "exit", "quit", "logout", "bye",
//	}

func startSocketShell(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen port failed: %w", err)
	}
	defer func() { _ = listener.Close() }()

	_, _ = fmt.Printf(terminal.ColorInfo + "=== Socket Fake Shell (tcp mode) ===\n" + terminal.ColorReset)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf(terminal.ColorError+"accept connection failed: %v\n"+terminal.ColorReset, err)
			continue
		}
		go func(c net.Conn) {
			defer func() { _ = c.Close() }()
			shell := terminal.NewShell(c, c)
			select {
			case <-time.After(5 * time.Minute):
			default:
				if err := shell.Run(); err == nil || err == io.EOF {
					return
				}
				_, _ = fmt.Printf(
					terminal.ColorError+"Connection %s error: %v\n"+terminal.ColorReset,
					c.RemoteAddr(), err,
				)
			}
		}(conn)
	}
}

func startStdioShell() error {
	shell := terminal.NewShell(os.Stdin, os.Stdout)
	return shell.Run()
}

func main() {
	if err := startStdioShell(); err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			terminal.ColorError+"Stdio shell error: %v\r\n"+terminal.ColorReset,
			err,
		)
		os.Exit(1)
	}
}
