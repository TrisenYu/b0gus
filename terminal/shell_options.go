package terminal

// TODO

type ShellOptions struct {
	PromptAlter   func(ty int) string
	CmdHook       func(string) (string, error)
	KeystrokeHook func(...rune)
}
