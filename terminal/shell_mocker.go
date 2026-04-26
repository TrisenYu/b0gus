package terminal

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
/// Last modified at 2026/04/19 星期日 21:41:01

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	mapset "github.com/deckarep/golang-set"
)

const (
	ColorWarn  = "\033[33m" // yellow
	ColorReset = "\033[00m" // white
	ColorError = "\033[31m" // red

	ColorCtrlMinusOfs = 10 // \033,[,x,y,m ... \033,[,0,0,m => at least 10 bytes
)

var quitSet = mapset.NewSet("quit", "exit", "bye", "logout")

type Shell struct {
	cannotSend      atomic.Bool
	wg              sync.WaitGroup
	ctxTimeout      context.Context
	editor          *LineEditor
	currResp        chan string // NOTE: must use revoke() to close the channel.
	currCmd         chan *ShellSyncObj
	tmpDanglingBuf  string
	quoted          string
	DefaultPrompt   string
	FlipPrompt      string
	prompt          atomic.Value
	enableCmdChan   bool
	shouldContinued bool
}

// ShellSyncObj not only attaches the generated content but also the Ctx for timeout control.
type ShellSyncObj struct {
	Ctx     context.Context
	Payload string
}

// bufCheck is a threshold limitor for inhibiting the overflow action
func (s *Shell) bufCheck() bool {
	if len(s.tmpDanglingBuf) <= 256 {
		return false
	}
	s.tmpDanglingBuf = ""
	s.shouldContinued = false
	s.quoted = ""
	s.editor.Writeln(ColorError, "command too long", ColorReset)
	return true
}

// Run could be tested by these commands:
//   - rlwrap nc localhost 2025 # installation on Debian: apt install rlwrap
//   - nc localhost 2025 # arrow key will be sent to the shell directly
//   - ssh -P2022 localhost
func (s *Shell) Run() error {
	s.cannotSend.Store(false)
	defer func() {
		s.editor.RestoreTermState()
		s.revoke()
	}()
	for {
		if s.bufCheck() {
			continue
		}
		if !s.shouldContinued && len(s.quoted) == 0 {
			s.setPrompt(s.DefaultPrompt, ColorReset)
		}
		s.editor.prompt = s.prompt.Load().(string)
		currLine, err := s.editor.Readline(s.shouldContinued)

		if errors.Is(err, io.EOF) {
			s.editor.Writeln("bye")
			return nil
		} else if errors.Is(err, pseudoErrNewLine) {
		} else if err == nil { // ctrl + C
			s.tmpDanglingBuf = ""
			s.quoted = ""
			s.shouldContinued = false
			continue
		} else { // like a crash
			return err
		}
		// [TODO]: it would be better if we could check the command is valid or not?
		currLine = strings.TrimSpace(currLine)
		if len(currLine) == 0 || s.FlipPromptCheck(currLine) {
			continue
		}
		// s.editor.GlitchCharCnt = 0
		if len(s.tmpDanglingBuf) <= len("logout") &&
			quitSet.Contains(strings.ToLower(s.tmpDanglingBuf)) {
			s.editor.Writeln("bye")
			return nil
		}
		if !s.enableCmdChan {
			// [TODO]: Enhance This. though there is not any better idea...
			s.editor.Writeln(
				ColorError, "command not found: ", s.tmpDanglingBuf, ColorReset,
			)
			s.tmpDanglingBuf = ""
			continue
		}
		if s.bufCheck() {
			continue
		}
		ctxTimeout, cancel := context.WithCancel(context.Background())
		s.currCmd <- &ShellSyncObj{Ctx: ctxTimeout, Payload: s.tmpDanglingBuf}
		select {
		case resp := <-s.currResp: // wait for response
			cancel()
			s.editor.Writeln(resp)
		case <-time.After(time.Second * 5): // otherwise fall into timeout mode?
			cancel()
			s.editor.Writeln(
				ColorError, "command not found: ", s.tmpDanglingBuf, ColorReset,
			)
		}
		s.tmpDanglingBuf = "" // reset buffer
	}
}

func (s *Shell) SetDebugger(fd *os.File) {
	s.editor.SetDebuggerDstLog(fd)
}

// revoke is used to close the channel.
func (s *Shell) revoke() {
	s.cannotSend.Store(true)
	close(s.currCmd)
	s.wg.Wait()
	close(s.currResp)
}

func (s *Shell) setPrompt(promptChar string, Color string) {
	var sb strings.Builder
	sb.WriteString(Color)
	sb.WriteString(promptChar)
	sb.WriteString(ColorReset)
	s.prompt.Store(sb.String())
}

func (s *Shell) FlipPromptCheck(currLine string) bool {
	if len(s.FlipPrompt) == 0 {
		s.tmpDanglingBuf = currLine
		return false
	}
	var (
		isUnclosed bool
		quoteType  string
		choice     rune = 0
		lenBuf          = len(currLine)
	)

	// TODO: determine whether to maintain multiple line for detailed, business protocol.
	// because the line editor can handle one line at every time,
	// we have to handle backslash in the loop...
	if len(s.quoted) > 0 {
		choice = rune(s.quoted[0])
	}
	isUnclosed, quoteType = checkQuoteState(choice, currLine)
	if isUnclosed {
		s.quoted = quoteType
		s.setPrompt("quote "+quoteType+s.FlipPrompt, ColorWarn)
		s.shouldContinued = true
	} else if len(s.quoted) > 0 {
		s.quoted = ""
	}

	if lenBuf > 0 && currLine[lenBuf-1] == '\\' {
		// we have to iterate \ until recognizing it is a broken command
		// or indeed an odd number of backslash
		test := backslashSeqChecker(currLine)
		if test < 0 {
			s.editor.Writeln(
				ColorError, "invalid command found: ", s.tmpDanglingBuf,
				currLine[:lenBuf-1], ColorReset,
			)
			s.shouldContinued = false
			s.quoted, s.tmpDanglingBuf = "", "" // flush all
			return true
		}
		s.shouldContinued = test == 1
		currLine = currLine[:lenBuf-1] // maybe handle for escape characters
	} else {
		if len(s.quoted) == 0 {
			s.shouldContinued = false
		}
	}

	if s.shouldContinued { // only odd backslash
		s.tmpDanglingBuf += currLine
		if len(s.quoted) == 0 {
			s.setPrompt(s.FlipPrompt, ColorWarn)
		}
		return true
	} // else {
	if len(s.tmpDanglingBuf) > 0 {
		s.tmpDanglingBuf += currLine
	} else {
		s.tmpDanglingBuf = currLine
	}
	s.setPrompt(s.DefaultPrompt, ColorReset)
	// }
	return false
}

// backslashSeqChecker will reversely iterate a string and
// determine whether the tail of given string has odd number of backslash.
//
//	return -1 if the backslash is glued with non-space char.
//	return  0 if other conditions.
//	return  1 if the number of backslash in the string is odd.
func backslashSeqChecker(x string) int {
	cnt := 0
	for i := len(x) - 1; i >= 0; i-- {
		if x[i] == '\\' {
			cnt++
		} else if unicode.IsSpace(rune(x[i])) {
			break
		} else {
			return -1
		}
	}
	return cnt & 1
}

var stringSignSet = mapset.NewSet('"', '`', '\'')

func checkQuoteState(hold rune, s string) (bool, string) {
	if len(s) == 0 {
		return false, ""
	}
	for i, ch := range s {
		flag := i > 0 && s[i-1] == '\\'
		if flag || !stringSignSet.Contains(ch) {
			// 1. no leading backslash before any quote sign
			// 2. in charset
			continue
		}
		if hold == 0 { // initiated set char
			hold = ch
			continue
		} else if hold == ch {
			hold = 0
		}
	}
	return hold != 0, string(hold)
}

func (s *Shell) AlterIOsrc(r io.Reader, w io.Writer) { s.editor.AlterIOsrc(r, w) }

// GetWriter satisfies the demand of some service like SMTP.
// Typically, they require connection to be more secure but lack of states for maintaining connection
// in their handling contexts.
func (s *Shell) GetWriter() io.Writer { return s.editor.Writer }

// SetCurrResp will check whether the shell is terminated,
// and then use WaitGroup to wait remaining response
func (s *Shell) SetCurrResp(resp *ShellSyncObj) {
	if !s.enableCmdChan || resp == nil || s.cannotSend.Load() {
		return
	}
	s.wg.Go(func() {
		select {
		case s.currResp <- resp.Payload:
		case <-resp.Ctx.Done():
			// When canceling the context, then the resp.Payload can't be pushed into the channel s.currResp.
			// Therefore, next round still keeps synchronous.
		}
	})
}

func (s *Shell) GetCurrCmd() *ShellSyncObj {
	if !s.enableCmdChan {
		return nil
	}
	// a reading try from a closed channel in go1.26 will finally get empty string.
	return <-s.currCmd
}

// NewShell creates a shell for abstract RW entities.
func NewShell(
	r io.Reader, w io.Writer,
	commandHook bool,
	defaultPromptChars, flipPromptChars string,
	LFasCRLF, fullCRLF bool, // each time the action will send only one keystroke signal
) *Shell {
	res := &Shell{
		editor:          NewLineEditor(r, w, LFasCRLF, fullCRLF),
		tmpDanglingBuf:  "",
		shouldContinued: false,
		currResp:        make(chan string),
		currCmd:         make(chan *ShellSyncObj),
		enableCmdChan:   commandHook,
	}
	res.DefaultPrompt = defaultPromptChars
	res.FlipPrompt = flipPromptChars
	res.setPrompt(res.DefaultPrompt, ColorReset)
	return res
}
