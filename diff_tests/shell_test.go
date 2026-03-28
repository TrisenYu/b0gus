package diff_tests

import (
	"b0gus/terminal"
	"bytes"
	"testing"
)

func TestShell(t *testing.T) {
	var payloads = []string{
		"",
		"\x01\x02\x03\x04",
		"\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x20",
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
	}

	buf := bytes.NewBuffer(nil)
	for _, payload := range payloads {
		buf.WriteString(payload)
		buf.WriteByte('\n')
	}
	term := terminal.NewShell(buf, &bytes.Buffer{}, true)
	go func() {
		_ = term.Run()
	}()
	for range payloads {
		curr := term.GetCurrCmd()
		if curr == nil {
			break
		}
		t.Log(curr.Payload)
		term.SetCurrResp(curr)
	}
}

func FuzzShell(f *testing.F) {
	var seeds = []string{
		"",
		"\x01\x02\x03\x04",
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
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		inBuf := bytes.NewBufferString(input)
		term := terminal.NewShell(inBuf, &bytes.Buffer{}, false)
		_ = term.Run()
	})
}
