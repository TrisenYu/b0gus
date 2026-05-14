package terminal

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

// ShellRules defines the field required by shell or its line editor.
type ShellRules struct {
	DefaultPrompts string
	PendingPrompts string

	NeedHijackCmd bool
	// each time the action will send only one keystroke signal as the sign of new line
	LFisCRLF, FullCRLF bool
}

// [TODO]
type shellOptions struct {
	PromptAlter   func(ty int) string
	CmdHook       func(string) (string, error)
	KeystrokeHook func(...rune)
}
