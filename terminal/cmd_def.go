package terminal

// https://zsh.sourceforge.io/Doc/Release/Shell-Builtin-Commands.html
// https://www.gnu.org/software/bash/manual/html_node/Bash-Builtins.html
var whiteListCmd = []string{
	"ls", "echo", "uname", "whoami", "cd",
	"exit", "quit", "logout", "source",
	"alias", "unalias", "bye", "export",
	"local", "which", "where",
	// moderation
	"print", "printf", // ? what if their output are leading to a file?
	"curl", "wget", "axel", // ? network activities
}

type GenericCmd struct {
	CmdName             string
	InitArgs            []string
	DependProgRes       []GenericCmd
	AsSrcCmd, IsInvalid bool
}
