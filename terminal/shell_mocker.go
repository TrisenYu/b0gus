package terminal

/// Last modified at 2026/04/07 星期二 23:04:30

import (
	"bufio"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"fmt"
	"io"
	"os"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"golang.org/x/term"
)

const (
	ColorWarn         = "\033[33m" // yellow
	ColorReset        = "\033[00m" // white
	ColorError        = "\033[31m" // red
	ColorCtrlMinusOfs = 10         // \033,[,x,y,m ... \033,[,0,0,m => at least 10 bytes
)

var quitSet = mapset.NewSet("quit", "exit", "bye", "logout")

type LineEditor struct {
	// TODO: add history or known commands for autosuggestions?
	reader    *bufio.Reader
	termState *term.State
	Writer    io.Writer
	buf       []rune
	keep      []byte
	curPos    int
	isNot1By1 bool
}

// SetTermState will turn termState into raw mode, and store the old state value
func (e *LineEditor) SetTermState(r io.Reader) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		state, err := term.MakeRaw(int(f.Fd()))
		if err == nil {
			e.termState = state
		}
	}
}

// RestoreTermState recovers the previous terminal states
func (e *LineEditor) RestoreTermState() {
	if e.termState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), e.termState)
	}
}

// delToPrevWord helps ctrl+W so that the editor looks like a real shell
func (e *LineEditor) delToPrevWord() {
	if e.curPos == 0 {
		return
	}
	pos := e.curPos - 1
	for pos >= 0 && e.buf[pos] == ' ' {
		pos--
	}
	for pos >= 0 && e.buf[pos] != ' ' {
		pos--
	}
	e.buf = append(e.buf[:pos+1], e.buf[e.curPos:]...)
	e.curPos = pos + 1
}

func (e *LineEditor) moveCursorTo(pos int) {
	_, _ = fmt.Fprintf(e.Writer, "\r\033[%dC", pos)
}

// moveCursorLR is buggy when there is multi-bytes character in the buf.
func (e *LineEditor) moveCursorLR(n int, d rune) {
	var choice = "C"
	switch d {
	case 'l':
		choice = "D"
	case 'r':
	default:
		return
	}
	if n > 0 {
		_, _ = fmt.Fprintf(e.Writer, "\033[%d"+choice, n)
	}
}

func (e *LineEditor) jumpToStOfPrevWord() {
	if e.curPos == 0 {
		return
	}
	pos := e.curPos - 1
	for pos >= 0 && e.buf[pos] == ' ' {
		pos--
	}
	for pos >= 0 && e.buf[pos] != ' ' {
		pos--
	}
	e.curPos = pos + 1
}

func (e *LineEditor) jumpToEdOfNextWord() {
	if e.curPos >= len(e.buf) {
		return
	}
	pos := e.curPos
	for pos < len(e.buf) && e.buf[pos] != ' ' {
		pos++
	}
	for pos < len(e.buf) && e.buf[pos] == ' ' {
		pos++
	}
	e.curPos = pos
}

// clearLine clear the line
func (e *LineEditor) clearLine() {
	_, _ = fmt.Fprint(e.Writer, "\r\033[K")
}

// redraw function will clear current line and set cursor
func (e *LineEditor) redraw(prompt string) {
	e.clearLine()
	_, _ = fmt.Fprint(e.Writer, prompt+string(e.buf))
	// currently, default prompt has been colored
	// so there are extra 10 bytes should be dropped
	e.moveCursorTo(len(prompt) + e.curPos - ColorCtrlMinusOfs)
}

var (
	pseudoErrKeepReading              = errors.New("keep reading")
	pseudoErrNewLine                  = errors.New("newline")
	pseudoErrSuspectMultiBytesCtrlSeq = errors.New("suspect control sequence")
	pseudoErrSuspectUtf8              = errors.New("suspect utf8")
)

// Readline handles for attempting to convert bytes stream into utf8 characters
// err ~ { pseudoErrNewLine, io.EOF }
func (e *LineEditor) Readline(prompt string, mulLineRequired bool) (string, error) {
	if e.isNot1By1 {
		_, _ = fmt.Fprint(e.Writer, prompt)
	} else {
		var sb strings.Builder
		sb.WriteRune('\r')
		if mulLineRequired {
			sb.WriteRune('\n')
		}
		_, _ = fmt.Fprint(e.Writer, sb.String(), prompt)
	}
	e.buf = make([]rune, 0)
	e.curPos = 0
	for {
		r, err := e.bytesSequenceToRunes(prompt)
		if err == nil { // control + C
			_, _ = fmt.Fprint(e.Writer, "\r\n", prompt)
			continue
		} else if errors.Is(err, io.EOF) { // exit
			return string(r), err
		} else if errors.Is(err, pseudoErrNewLine) {
			res := string(e.buf)
			e.buf = make([]rune, 0)
			e.curPos = 0
			return res, err
		} else if errors.Is(err, pseudoErrKeepReading) {
			continue
		}
	}
}

// bytesSequenceToRunes is yet a buggy implementation to
// convert byte sequences into utf8 character or identify control sequence.
// Note that returned-err belongs to one of the member of
// { pseudoErrKeepReading, pseudoErrNewLine, io.EOF, errUtf8Invalid, nil }
//
// During execution, protocol like SSH will send keystroke/control-sequence one by one.
// While protocol like SMTP will send the whole command at one time.
func (e *LineEditor) bytesSequenceToRunes(prompt string) ([]rune, error) {
	// utf8 at least have 4 bytes. and control sequence might have 6 bytes
	// if we use peek, then io will have chance to stuck.
	// here we can use a "keep buffer" to store the prefetched bytes.
	var tmp []byte = nil
	if len(e.keep) > 0 {
		tmp = append(tmp, e.keep...)
		e.keep = make([]byte, 0)
		for st := len(tmp); st < 6; st++ {
			tmp = append(tmp, 0)
		}
	} else {
		tmp = make([]byte, 6)
	}
	n, err := e.reader.Read(tmp)
	if err != nil {
		return nil, err
	}
	for st := 0; st < n; {
		// let's handle ctrl-seq or ascii character first ?
		// but it looks like we encode in a wrong way
		// emoji will be interpreted as a single-byte control character
		// and a char under utf8.
		payload, err := e.handleCtrlSeq(tmp[st], prompt)
		if err == nil { // control + C
			return nil, nil
		} else if errors.Is(err, pseudoErrNewLine) { // \r\n
			return payload, err
		} else if errors.Is(err, io.EOF) { // command like exit
			return payload, err
		} else if errors.Is(err, pseudoErrKeepReading) { // normal ascii
			st++
			continue
		} else if errors.Is(err, pseudoErrSuspectMultiBytesCtrlSeq) {
			return e.handleMultiBytes4CtrlSeq(n, tmp, prompt, e.buf)
		} else if errors.Is(err, pseudoErrSuspectUtf8) {
			r, skip := utf8.DecodeRune(tmp[st:])
			if r != utf8.RuneError {
				e.buf = append(e.buf, r)
				e.curPos++
				e.redraw(prompt)
				st += skip
				continue
			}
			if utf8.RuneStart(tmp[st]) {
				// so save st till the end
				for ed := st; ed < n; ed++ {
					e.keep = append(e.keep, tmp[ed])
				}
				// only read before st and we stop parsing here.
				return nil, pseudoErrKeepReading
			}
			// there is a corner case: telnet will send control sequence like "ff fd"
			// which will hang the server... if we jump the wrong position,
			// then the whole encoding will fail since extra bytes have been read.
			e.buf = append(e.buf, utf8.RuneError)
			e.curPos++
			e.redraw(prompt)
			st += 1
			// give a chance to parse further
		}
	}

	return e.buf, pseudoErrKeepReading
}

// handleCtrlSeq uses error to represent the expecting next-state.
//   - nil for indicating ctrl+C signal
//   - pseudoErrNewLine notifies for immediately handling the incoming new line
//   - pseudoErrKeepReading requests for keeping reading the printable characters
//   - pseudoErrSuspectMultiBytesCtrlSeq requests for interpreting the multibyte control sequence
//   - pseudoErrSuspectUtf8 suspects that current character may belong to utf-8 char-set
func (e *LineEditor) handleCtrlSeq(r byte, prompt string) ([]rune, error) {
	switch {
	case r == 0x01: // ctrl + a
		e.curPos = 0
		e.redraw(prompt)
	case r == 0x03: // ctrl + c
		_, _ = fmt.Fprintln(e.Writer, "^C")
		return nil, nil
	case r == 0x05: // ctrl + e, jump to the end of line
		e.curPos = len(e.buf)
		e.redraw(prompt)
	case r == 0x08: // ctrl + h, same as backspace
		if e.curPos > 0 {
			e.buf = append(e.buf[:e.curPos-1], e.buf[e.curPos:]...)
			e.curPos--
			e.redraw(prompt)
		}
	case r == '\t': // ctrl + i, same as \t
		// [TODO]: add autocompletion suggestions to make the the fake shell like a real one
	case r == '\n': // ctrl + j, same as \n
		if e.isNot1By1 {
			e.clearLine()
		} else {
			_, _ = fmt.Fprintln(e.Writer)
		}
		return e.buf, pseudoErrNewLine
	case r == 0x0b: // ctrl + k, delete from cursor to the end of line
		e.buf = e.buf[:e.curPos]
		e.redraw(prompt)
	case r == '\r': // ctrl + m, same as \r or 4-bytes utf8...
		if !e.isNot1By1 {
			return e.buf, pseudoErrNewLine
		}
	case r == 0x11: // ctrl + q
		return []rune("exit"), io.EOF
	case r == 0x15: // ctrl + u
		e.clearLine()
		e.buf = make([]rune, 0)
		e.curPos = 0
		e.redraw(prompt)
	case r == 0x17: // ctrl + w
		e.delToPrevWord()
		e.redraw(prompt)
	case r >= 0x20 && r <= 0x7e:
		t := rune(r)
		e.buf = append(e.buf, t)
		e.curPos++
		e.redraw(prompt)
		return []rune{t}, pseudoErrKeepReading
	case r == 0x7f: // backspace
		if e.curPos > 0 {
			e.buf = append(e.buf[:e.curPos-1], e.buf[e.curPos:]...)
			e.curPos--
			e.redraw(prompt)
		}
	case r == 0x00: // ctrl + ~, or end of current stream.
		return nil, pseudoErrKeepReading
	case r == 0x02: // ctrl + b
		fallthrough
	case r == 0x04: // ctrl + d
		fallthrough
	case r == 0x06: // ctrl + f
		fallthrough
	case r == 0x07: // ctrl + g
		fallthrough
	case r == 0x0c: // ctrl + l, delete from cursor to the end of line
		fallthrough
	case (0x0e <= r && r <= 0x10) || (0x12 <= r && r <= 0x14):
		// ctrl + {n, o, p, r, s, t}
		fallthrough
	case r == 0x16: // ctrl + v
		fallthrough
	case (0x18 <= r && r <= 0x1a) || (0x1c <= r && r <= 0x1f):
		// ctrl + {x, y, z, *, ], ^, _}, temporarily skip
	case r == 0x1b: // 0x1b: escape keystroke
		return nil, pseudoErrSuspectMultiBytesCtrlSeq
	default:
		return nil, pseudoErrSuspectUtf8
	}
	return nil, pseudoErrKeepReading
}

func (e *LineEditor) handleMultiBytes4CtrlSeq(
	n int, tmp []byte, prompt string, res []rune,
) ([]rune, error) {
	flagL := n >= 1 && rune(tmp[0]) == 0x1B
	flag2bL := flagL && n >= 2 && rune(tmp[1]) == 'b' // {esc, b} <=> ctrl + left arrow
	flag2fR := flagL && n >= 2 && rune(tmp[1]) == 'f' // {esc, f} <=> ctrl + right arrow
	lPart3 := flagL && n >= 3 && rune(tmp[1]) == '['
	flag3A := lPart3 && rune(tmp[2]) == 'A' // up    arrow
	flag3B := lPart3 && rune(tmp[2]) == 'B' // down  arrow
	flag3C := lPart3 && rune(tmp[2]) == 'C' // right arrow
	flag3D := lPart3 && rune(tmp[2]) == 'D' // left arrow
	flag4 := flagL && n >= 4 && rune(tmp[2]) == '3' &&
		rune(tmp[3]) == '~' // delete keystroke
	lPart3 = flagL && n == 6 &&
		rune(tmp[2]) == '1' && rune(tmp[3]) == ';' &&
		(rune(tmp[4]) == '3' || rune(tmp[4]) == '5')
	flag6ctrlR := lPart3 && rune(tmp[5]) == 'C' // esc+1+;+{3,5}+D <=> ctrl + right arrow
	flag6ctrlL := lPart3 && rune(tmp[5]) == 'D' // esc+1+;+{3,5}+C <=> ctrl + left  arrow

	if flag2bL || flag6ctrlL {
		e.jumpToStOfPrevWord()
		e.redraw(prompt)
		return res, pseudoErrKeepReading
	} else if flag2fR || flag6ctrlR {
		e.jumpToEdOfNextWord()
		e.redraw(prompt)
		return res, pseudoErrKeepReading
	} else if flag3A || flag3B { // yet to complete up/down arrow keystroke
		return res, pseudoErrKeepReading
	} else if flag3C {
		if e.curPos < len(e.buf) {
			e.curPos++
			e.moveCursorLR(1, 'r')
		}
		return res, pseudoErrKeepReading
	} else if flag3D {
		if e.curPos > 0 {
			e.curPos--
			e.moveCursorLR(1, 'l')
		}
		return res, pseudoErrKeepReading
	} else if flag4 {
		if e.curPos < len(e.buf) {
			e.buf = append(e.buf[:e.curPos], e.buf[e.curPos+1:]...)
			e.redraw(prompt)
		}
		return res, pseudoErrKeepReading
	}
	return nil, pseudoErrKeepReading
}

func (e *LineEditor) AlterIOsrc(r io.Reader, w io.Writer) {
	e.reader = bufio.NewReader(r)
	e.Writer, e.curPos = w, 0
}

func newLineEditor(r io.Reader, w io.Writer, isNotSSHConn bool) *LineEditor {
	e := &LineEditor{
		buf:       make([]rune, 0),
		curPos:    0,
		reader:    bufio.NewReader(r),
		Writer:    w,
		isNot1By1: isNotSSHConn,
	}
	e.SetTermState(r)
	return e
}

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
		if !s.shouldContinued && len(s.quoted) == 0 {
			s.setPrompt(s.DefaultPrompt, ColorReset)
		}
		currLine, err := s.editor.Readline(s.prompt.Load().(string), s.shouldContinued)
		if errors.Is(err, io.EOF) {
			_, _ = fmt.Fprintln(
				s.editor.Writer, ColorWarn+"Received EOF, exit."+ColorReset,
			)
			return err
		} else if errors.Is(err, pseudoErrNewLine) {
			// do nothing here
		}
		currLine = strings.TrimSpace(currLine)
		if len(currLine) == 0 || s.FlipPromptCheck(currLine) {
			continue
		}
		if len(s.tmpDanglingBuf) <= len("logout") &&
			quitSet.Contains(strings.ToLower(s.tmpDanglingBuf)) {
			payload := "bye"
			if !s.editor.isNot1By1 {
				payload = "\r\n" + payload
			}
			_, _ = fmt.Fprintln(s.editor.Writer, payload)
			s.editor.clearLine()
			return nil
		}
		if !s.enableCmdChan {
			// [TODO]: Enhance This. though there is not any better idea...
			_, _ = fmt.Fprintln(
				s.editor.Writer,
				ColorError+"command not found: "+s.tmpDanglingBuf+ColorReset,
			)
			s.tmpDanglingBuf = ""
			continue
		}
		ctxTimeout, cancel := context.WithCancel(context.Background())
		s.currCmd <- &ShellSyncObj{Ctx: ctxTimeout, Payload: s.tmpDanglingBuf}
		select {
		case resp := <-s.currResp: // wait for response
			cancel()
			if !s.editor.isNot1By1 {
				_, _ = fmt.Fprintln(s.editor.Writer)
			}
			s.editor.clearLine() // ?
			_, _ = fmt.Fprintln(s.editor.Writer, resp)
		case <-time.After(time.Second * 5): // otherwise fall into timeout mode?
			cancel()
			_, _ = fmt.Fprintln(
				s.editor.Writer,
				ColorError+"command not found: "+s.tmpDanglingBuf+ColorReset,
			)
		}
		s.tmpDanglingBuf = "" // reset buffer
	}
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
			_, _ = fmt.Fprintln(
				s.editor.Writer,
				ColorError+"invalid command found: "+
					s.tmpDanglingBuf+currLine[:lenBuf-1]+
					ColorReset,
			)
			s.shouldContinued = false
			s.quoted, s.tmpDanglingBuf = "", "" // flush all
			return true
		}
		s.shouldContinued = test == 1
		currLine = currLine[:lenBuf-1]
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
//	return -1 if the backslash are glued with non-space char.
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

var stringSet = mapset.NewSet('"', '`', '\'')

func checkQuoteState(hold rune, s string) (bool, string) {
	if len(s) == 0 {
		return false, ""
	}
	for i, ch := range s {
		flag := i > 0 && s[i-1] == '\\'
		if flag || !stringSet.Contains(ch) {
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
// in their handling context.
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

// a reading try from a closed channel in go1.26 will finally get empty string.
//	package main
//	import "fmt"
//	func main() {
//		ch := make(chan string)
//		go func() { ch <- "hello" } ()
//		val, ok := <- ch
//		fmt.Println(val, len(val), ok)
//		close(ch)
//		val, ok = <- ch
//		fmt.Println(val, len(val), ok)
//		val, ok = <- ch
//		fmt.Println(val, len(val), ok)
//	}
//	> hello 5 true
//	>  0 false
//	>  0 false
//

func (s *Shell) GetCurrCmd() *ShellSyncObj {
	if !s.enableCmdChan {
		return nil
	}
	return <-s.currCmd
}

// NewShell creates a shell for abstract RW entities.
func NewShell(
	r io.Reader, w io.Writer,
	commandHook bool,
	defaultPromptChars, flipPromptChars string,
	isNotSSHchan bool,
) *Shell {
	res := &Shell{
		editor:          newLineEditor(r, w, isNotSSHchan),
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
