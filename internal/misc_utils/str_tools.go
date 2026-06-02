package misc_utils

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clauses

import (
	"errors"
	"io/fs"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
)

func IsIP(x string) bool {
	ip := net.ParseIP(x)
	if ip == nil {
		return false
	}
	_, err := net.ResolveIPAddr("ip", x)
	return err == nil
}

func lazyRegexCompile(str string) func() *regexp.Regexp {
	var (
		regex *regexp.Regexp
		once  sync.Once
	)
	return func() *regexp.Regexp {
		once.Do(func() {
			regex = regexp.MustCompile(str)
		})
		return regex
	}
}

// https://github.com/go-playground/validator/blob/master/baked_in.go
var (
	// accepts hostname starting with a digit https://tools.ietf.org/html/rfc1123
	hostnameRegexStrRFC1123 = `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
	// modified from b0gus/auto_translator.py
	markdownCodeBlockRegexStr = "^(?:[^`]|``?[^`])*`{0,2}$|^```([\\p{L}\\p{N}_]+)\\n([^`]*?)\\n```"
	hostnameRegexRFC1123      = lazyRegexCompile(hostnameRegexStrRFC1123)
	markdownCodeBlockRegex    = lazyRegexCompile(markdownCodeBlockRegexStr)
)

func IsHostnameRFC1123(str string) bool {
	return hostnameRegexRFC1123().MatchString(str)
}

type integer interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | int | uint
}

func IsPort[T integer](x T) bool {
	cast := uint32(x)
	return 0 < cast && cast <= 65535
}

func IsDir(x string) bool {
	fileInfo, err := os.Stat(x)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func IsFile(x string) bool {
	fileInfo, err := os.Stat(x)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

func IsFilePath(x string) bool {
	var (
		err error
		te  *fs.PathError
	)
	if IsDir(x) {
		return false
	}

	// If it exists, it obviously is valid.
	// This is done first to avoid code duplication and unnecessary additional logic.
	if ok := IsFile(x); ok {
		return ok
	}

	if strings.TrimSpace(x) == "" {
		return false
	}
	if strings.HasSuffix(x, string(os.PathSeparator)) {
		return false
	}
	_, err = os.Stat(x)
	switch {
	case errors.As(err, &te):
		if errors.Is(te.Err, syscall.EINVAL) {
			return true
		}
	default:
	}
	return false
}

func IsURL(x string) bool {
	s := strings.ToLower(x)
	if len(s) == 0 {
		return false
	}
	Url, err := url.Parse(s)
	if err != nil || Url.Scheme == "" {
		return false
	}
	isFileScheme := Url.Scheme == "file"

	if (isFileScheme && (len(Url.Path) == 0 || Url.Path == "/")) ||
		(!isFileScheme && len(Url.Host) == 0 && len(Url.Fragment) == 0 && len(Url.Opaque) == 0) {
		return false
	}
	return true
}

// StripMarkdownSignIfAny is redefined from b0gus/auto_translator.py
func StripMarkdownSignIfAny(x string) string {
	res := markdownCodeBlockRegex().FindStringSubmatch(x)
	if res == nil || len(res) == 0 {
		return ""
	}
	if len(res) > 2 && len(res[1]) > 0 {
		return res[2]
	}
	if len(x) > 0 {
		return x
	}
	return ""
}

// dumpers

// DumpBytesInHex will dump hex of byte in bytes array one by one
//nolint
func DumpBytesInHex(xx []byte) string {
	var sb strings.Builder
	digitBrancher := func(x byte) {
		if x >= 10 {
			sb.WriteByte(x - 10 + 'a')
			return
		}
		sb.WriteByte(x + '0')
	}
	sb.WriteString(" [")
	for i, p := range xx {
		digitBrancher(p >> 4) // div
		digitBrancher(p & 15) // mod
		if i < len(xx)-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

// DumpCharsInString will dump Unicode in string one by one
//nolint
func DumpCharsInString(x string) string {
	var sb strings.Builder
	digitBrancher := func(b byte) {
		div, rem := b>>4, b&0xf
		if div >= 10 {
			sb.WriteByte(div - 10 + 'a')
		} else {
			sb.WriteByte(div + '0')
		}
		if rem >= 10 {
			sb.WriteByte(rem - 10 + 'a')
		} else {
			sb.WriteByte(rem + '0')
		}
	}
	t := []rune(x)
	sb.WriteRune('[')
	for i, c := range t {
		sb.Write([]byte("'\\u"))
		digitBrancher(byte(c >> 24))
		digitBrancher(byte(c >> 16))
		digitBrancher(byte(c >> 8))
		digitBrancher(byte(c & 0xFF))
		if i < len(t)-1 {
			sb.Write([]byte("', "))
		}
	}
	sb.WriteRune(']')
	return sb.String()
}
