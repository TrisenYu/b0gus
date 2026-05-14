package diff_tests

import (
	"b0gus/terminal"
	"bytes"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

var CmdSeeds = []string{
	// command line payload
	"",
	"/usr/bin/whoami",
	"ls -liah",
	`
wget https://download.nvidia.com/XFree86/Linux-x86_64/580.105.08/NVIDIA-Linux-x86_64-580.105.08.run
`,
	"./NVIDIA-Linux-x86_64-580.105.08.run",
	"\x01\x02\x03\x04",
	"\uf800",
	"\u1234\u2345\u6789\u789a",
	"ls -liha",
	"ls -liha \n echo 'helo'",
	"gcc a.cc -lmath -O2 \\\n\t-o a.exe",
	"[ -z \"$PWD\"] && echo 'hell yeah'",
	"[ -z \"$PWD\"] && echo 'hell yeah",
	"[ -z \"$PWD\"] && echo `hell yeah",
	"[ -z \"$PWD\"] && echo `hell yeah'",
	"[ -z \"$PWD\"] && echo \x00`hell yeah'",
	"abc \\\\\\\a\b\f\n\r\t\v\\",
	"صَبَاحُ الْخَيْرِ",
	"早安！",
	"ehlo",
	"help",
	"exit",
	"\x1b[",
	"\x1b[1",
	"\x1b[1;",
	"\x1b[1;bm",
	"\x1b\x12",
	"\u001Bb\u001Bb\u001Bb\u001Bb\u001Bb\u001Bb\u001Bf\u001Bf\u001Bf\u001Bf\u001Bf\u001Bf",
	// NTP payload
	"l\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
		"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
}

func TestShell(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for _, payload := range CmdSeeds {
		buf.WriteString(payload)
		buf.WriteByte('\n')
	}
	term := terminal.NewShell(
		buf, &bytes.Buffer{},
		terminal.ShellRules{
			DefaultPrompts: "$ ",
			PendingPrompts: "> ",
			NeedHijackCmd:  true,
			LFisCRLF:       false,
			FullCRLF:       false,
		},
	)
	go func() {
		_ = term.Run()
	}()
	for range CmdSeeds {
		curr := term.GetCurrCmd()
		if curr == nil {
			break
		}
		t.Log(curr.Payload)
		term.SetCurrResp(curr)
	}
}

func FuzzShell(f *testing.F) {
	for _, seed := range CmdSeeds {
		f.Add(seed)
	}
	var sb strings.Builder
	for range 512 {
		sb.WriteRune('\x1b')
		sb.WriteRune('[')
		sb.WriteRune('C')
	}
	f.Add(sb.String())
	sb.Reset()
	for i := range 0x10000 {
		sb.WriteRune(rune(i))
	}
	f.Add(sb.String())
	sb.Reset()

	f.Fuzz(func(t *testing.T, input string) {
		for i := range 3 {
			localFuzz(t, input, i&1 != 0, i&2 != 0)
		}
		for i := range 3 {
			respPassing(t, input, i&1 != 0, i&2 != 0)
		}
	})
}

func localFuzz(t *testing.T, input string, LFasCRLF, fullCRLF bool) {
	inBuf := bytes.NewBufferString(input)
	term := terminal.NewShell(
		inBuf, &bytes.Buffer{},
		terminal.ShellRules{
			DefaultPrompts: "$ ",
			PendingPrompts: "> ",
			NeedHijackCmd:  false,
			LFisCRLF:       false,
			FullCRLF:       false,
		},
	)
	err := term.Run()
	if err != nil && !errors.Is(err, io.EOF) {
		t.Error(err)
	}
}

func respPassing(t *testing.T, input string, LFasCRLF, fullCRLF bool) {
	inBuf := bytes.NewBufferString(input)
	term := terminal.NewShell(
		inBuf, &bytes.Buffer{},
		terminal.ShellRules{
			DefaultPrompts: "$ ",
			PendingPrompts: "> ",
			NeedHijackCmd:  true,
			LFisCRLF:       LFasCRLF,
			FullCRLF:       fullCRLF,
		},
	)
	var quitFlag atomic.Bool
	quitFlag.Store(false)

	go func() {
		defer quitFlag.Store(true)
		err := term.Run()
		if err != nil && !errors.Is(err, io.EOF) {
			t.Error(err)
		}
	}()
	for !quitFlag.Load() {
		resp := term.GetCurrCmd()
		term.SetCurrResp(resp) // directly return back
	}
}
