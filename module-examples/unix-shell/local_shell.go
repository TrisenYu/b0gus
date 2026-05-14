package main

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/04/14 星期二 22:24:32

import (
	"b0gus/terminal"
	"fmt"
	"os"
)

// the program here will create a fake shell in your terminal
func main() {
	fd, err := os.OpenFile("../../assets/log/term.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer func() { _ = fd.Close() }()
	shell := terminal.NewShell(
		os.Stdout, os.Stdin,
		terminal.ShellRules{
			DefaultPrompts: "$ ",
			PendingPrompts: "> ",
			NeedHijackCmd:  false,
			LFisCRLF:       false,
			FullCRLF:       false,
		},
	)
	shell.SetDebugger(fd)
	_ = shell.Run()
}
