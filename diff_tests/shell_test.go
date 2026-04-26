package diff_tests

// correct program will not crash or be exploited
// good program will not waste

import (
	"b0gus/terminal"
	"bytes"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

var seeds = []string{
	"",
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
	"\x1b[",
	"\x1b[1",
	"\x1b[1;",
	"\x1b[1;bm",
	"\x1b\x12",
	"\u001Bb\u001Bb\u001Bb\u001Bb\u001Bb\u001Bb\u001Bf\u001Bf\u001Bf\u001Bf\u001Bf\u001Bf",
}

func TestShell(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	for _, payload := range seeds {
		buf.WriteString(payload)
		buf.WriteByte('\n')
	}
	term := terminal.NewShell(
		buf, &bytes.Buffer{}, true,
		"$ ", "> ",
		false, false,
	)
	go func() {
		_ = term.Run()
	}()
	for range seeds {
		curr := term.GetCurrCmd()
		if curr == nil {
			break
		}
		t.Log(curr.Payload)
		term.SetCurrResp(curr)
	}
}

func FuzzShell(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}
	var sb strings.Builder
	for range 2048 {
		sb.WriteRune('a')
	}
	f.Add(sb.String())
	sb.Reset()
	for i := range 0x10000 {
		sb.WriteRune(i)
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
		inBuf, &bytes.Buffer{}, false,
		"$ ", "> ",
		LFasCRLF, fullCRLF,
	)
	err := term.Run()
	if err != nil && !errors.Is(err, io.EOF) {
		t.Error(err)
	}
}

func respPassing(t *testing.T, input string, LFasCRLF, fullCRLF bool) {
	inBuf := bytes.NewBufferString(input)
	term := terminal.NewShell(
		inBuf, &bytes.Buffer{}, true,
		"$ ", "> ",
		LFasCRLF, fullCRLF,
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
