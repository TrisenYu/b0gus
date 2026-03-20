//go:build windows
// +build windows

package terminal

import (
	powershell "github.com/airbus-cert/tree-sitter-powershell"
)



func (at *AbsTerminal) AnalyzeInput() {
	// When typing down the enter key, check if the last char is merely backslash.
	// Otherwise, split any space between input and mark the whole input as a (broken/correct-syntax) command.
	at.lock.Lock()
	defer at.lock.Unlock()

	switch at.ShellType {
	case "powershell":
		at.ResolveShellCmd(powershell.Language())
	case "bash":
	case "Redis":
		fallthrough
	case "SQLite":
		fallthrough
	case "PostgreSQL":
		fallthrough
	case "MySQL":
		at.ResolveSQLCli()
	default:
		// ?
	}
}
