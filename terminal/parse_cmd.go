package terminal

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

/// References:
// 	https://github.com/NHAS/reverse_ssh/blob/main/internal/terminal/terminal.go
// 	https://github.com/evilsocket/shellz/blob/master/cmd/shellz/run.go
// 	https://github.com/tj/go-terminput
// 	split shell command as string array
//		https://github.com/mattn/go-shellwords

// exposed port for reading or buffer for editing


type AbsTerminal struct {
	lock      sync.Mutex
	ReadSrc   io.Reader
	WriteSrc  io.Writer
	tmpBuf    []rune
	ShellType string
}

// Some commands are concatenated with the pipe symbol (|) as input to the next command,
// or invoked with backticks (`) or $() as passed parameters.
// Hence, they form an AST (the caller is at the root, and the callee is at the leaf nodes).

func (at *AbsTerminal) ResolveSQLCli() {
	// adjust for different type.
}

// ResolveShellCmd reads from inputSrc and
func (at *AbsTerminal) ResolveShellCmd(shellType unsafe.Pointer) []GenericCmd {
	// ? can memory object be viewed as file?
	defer func() {
		if r := recover(); r != nil {
			// TODO: log and return.
		}
	}()
	parser := sitter.NewParser()
	defer parser.Close()
	err := parser.SetLanguage(sitter.NewLanguage(shellType))
	if err != nil {
		return nil
	}
	cmdWithZeroPadding, err := io.ReadAll(at.ReadSrc)
	if err != nil {
		// can not turn the read src into command slice.
		return nil
	}
	idx := strings.Index(string(cmdWithZeroPadding), "\000")
	var cmd []byte
	if idx != -1 {
		cmd = cmdWithZeroPadding[:idx]
	} else {
		cmd = bytes.TrimSpace(cmdWithZeroPadding)
	}
	ast := parser.Parse(cmd, nil)
	defer ast.Close()
	root := ast.RootNode()
	return travelShellAST(root, cmd)
}
