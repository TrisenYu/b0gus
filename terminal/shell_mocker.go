package terminal

import (
	"bufio"
	"context"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"fmt"
	"io"
	"os"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"golang.org/x/term"
)

var (
	ColorWarn         = "\033[33m" // yellow
	ColorPrompt       = "\033[32m" // green
	ColorInfo         = "\033[36m" // cray
	ColorReset        = "\033[00m" // white
	ColorError        = "\033[31m" // red
	defaultPrompt     = ColorPrompt + "$ " + ColorReset
	CleanPromptTmpl   = ColorReset + "%s" + ColorReset
	ColorCtrlMinusOfs = 10 // \033,[,x,y,m ... \033,[,0,0,m => 10 bytes

	quitSet = mapset.NewSet("quit", "exit", "bye", "logout")
)

type LineEditor struct {
	// TODO: add history or known commands for autosuggestions?
	writer    io.Writer
	reader    *bufio.Reader
	buf       []rune
	curPos    int
	termState *term.State
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
	_, _ = fmt.Fprintf(e.writer, "\r\033[%dC", pos)
}

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
		_, _ = fmt.Fprintf(e.writer, "\033[%d"+choice, n)
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
	_, _ = fmt.Fprint(e.writer, "\r\033[K")
}

// redraw function will clear current line and set cursor
func (e *LineEditor) redraw(prompt string) {
	e.clearLine()
	_, _ = fmt.Fprint(e.writer, prompt+string(e.buf))
	// currently, default prompt has been colored
	// so there are extra 10 bytes should be dropped
	e.moveCursorTo(len(prompt) + e.curPos - ColorCtrlMinusOfs)
}

/*
	ctrlSeq = map[string]string{
		"\x00": "~", // ctrl+~
		"\x01": "A", // ctrl+a
		"\x02": "B", // ctrl+b
		//  ctrl+C, cease current command
		"\x04": "D", // ctrl+d
		// ctrl+e, use for exit
		"\x06": "F", // ctrl+f
		"\x07": "G", // ctrl+g
		// ctrl+H
		// ctrl+I
		"\x0a": "J", // ctrl+j
		"\x0b": "K", // ctrl+k
		"\x0c": "L", // ctrl+l
		"\x0d": "M", // ctrl+m
		"\x0e": "N", // ctrl+n
		"\x0f": "O", // ctrl+o
		"\x10": "P", // ctrl+p
		// ctrl+q, use for quit
		"\x12": "R", // ctrl+r
		"\x13": "S", // ctrl+s
		"\x14": "T", // ctrl+t
		"\x15": "U", // ctrl+u
		"\x16": "V", // ctrl+v
		"\x17": "W", // ctrl+w
		"\x18": "X", // ctrl+x
		"\x19": "Y", // ctrl+y
		"\x1a": "Z", // ctrl+z
		// [27,32) use for unknown and multiple mappings
	}
*/

// Readline will return the result if any \r\n arrives.
func (e *LineEditor) Readline(prompt string) (string, error) {
	_, _ = fmt.Fprint(e.writer, prompt)
	e.buf = make([]rune, 0)
	e.curPos = 0
	for {
		b, err := e.reader.ReadByte()
		if err != nil {
			return "", err
		}
		// TODO: complete all keystrokes
		switch b {
		case '\r', '\n':
			_, _ = fmt.Fprint(e.writer, "\r\n")
			res := string(e.buf)
			e.buf = make([]rune, 0)
			e.curPos = 0
			return res, nil
		case 0x01: // ctrl + a
			e.curPos = 0
			e.redraw(prompt)
		case 0x03: // ctrl + c
			_, _ = fmt.Fprint(e.writer, "^C\r\n")
			return "", nil
		case 0x04: // ctrl + d
			fallthrough
		case 0x11: // ctrl + q
			return "exit", io.EOF

		case 0x05: // ctrl + e, jump to the end of line
			e.curPos = len(e.buf)
			e.redraw(prompt)
		case 0x0b: // ctrl + k, delete from cursor to the end of line
			e.buf = e.buf[:e.curPos]
			e.redraw(prompt)

		case 0x15: // ctrl + u
			e.clearLine()
			e.buf = make([]rune, 0)
			e.curPos = 0
			e.redraw(prompt)
		case 0x17: // ctrl + w
			e.delToPrevWord()
			e.redraw(prompt)
		case 0x08: // ctrl + H
			fallthrough
		case 0x7f: // backspace
			if e.curPos > 0 {
				e.buf = append(e.buf[:e.curPos-1], e.buf[e.curPos:]...)
				e.curPos--
				e.redraw(prompt)
			}
		case '\t':
			// TODO: add autocompletion suggestions to make the shell more normal
			continue
		case 0x1b: // escape sequence
			nxt1, err := e.reader.ReadByte()
			if err != nil {
				continue
			}
			//peekBytes, err := e.reader.Peek(4)
			//if err != nil {
			//	// I don't know...But it seems fine?
			//}
			// [1{3,5}{DC}
			switch nxt1 {
			case 'b': // ctrl + <-
				e.jumpToStOfPrevWord()
				e.redraw(prompt)
			case 'f': // ctrl + ->
				e.jumpToEdOfNextWord()
				e.redraw(prompt)
			case '[': // extended escape
				nxt2, err := e.reader.ReadByte()
				if err != nil {
					continue
				}
				switch nxt2 {
				case '1':
					nxt3, err := e.reader.ReadByte()
					if err != nil || nxt3 != ';' {
						continue
					}
					nxt4, err := e.reader.ReadByte()
					if err != nil || nxt4 != '3' && nxt4 != '5' {
						continue
					}
					nxt5, err := e.reader.ReadByte()
					if err != nil {
						continue
					}
					if nxt5 == 'D' { // ctrl+left arrow: \ESC[1;5D
						e.jumpToStOfPrevWord()
						e.redraw(prompt)
					} else if nxt5 == 'C' { // ctrl+right arrow: \ESC[1;5C
						e.jumpToEdOfNextWord()
						e.redraw(prompt)
					}
					continue

				case '3': // delete keystroke
					nxt3, err := e.reader.ReadByte()
					if err != nil || nxt3 != '~' {
						continue
					}
					if e.curPos < len(e.buf) {
						e.buf = append(e.buf[:e.curPos], e.buf[e.curPos+1:]...)
						e.redraw(prompt)
					}
				case 'D': // left arrow
					if e.curPos > 0 {
						e.curPos--
						e.moveCursorLR(1, 'l')
					}
				case 'C': // right arrow
					if e.curPos < len(e.buf) {
						e.curPos++
						e.moveCursorLR(1, 'r')
					}
				case 'A': // up arrow
					// temporarily skip this
					fallthrough
				case 'B': // down arrow
					// temporarily skip this
					continue
				}
			}
		default:
			if 0x20 <= b && b <= 0x7e {
				lPart, rPart := e.buf[:e.curPos], e.buf[e.curPos:]
				e.buf = append(lPart, append([]rune{rune(b)}, rPart...)...)
				e.curPos++
				e.redraw(prompt)
			}
		}
	}
}

func NewLineEditor(r io.Reader, w io.Writer) *LineEditor {
	e := &LineEditor{
		buf:    make([]rune, 0),
		curPos: 0,
		reader: bufio.NewReader(r),
		writer: w,
	}
	e.SetTermState(r)
	return e
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

type Shell struct {
	editor         *LineEditor
	writer         io.Writer
	cannotSend     atomic.Bool
	wg             sync.WaitGroup
	opts           ShellOptions
	ctxTimeout     context.Context
	currResp       chan string // NOTE: must use revoke() to close the channel.
	currCmd        chan *ShellSyncObj
	tmpDanglingBuf string
	quoted         string
	prompt         string
	isContinued    bool
}

// ShellSyncObj not only attaches the generated content but also the Ctx for timeout control.
type ShellSyncObj struct {
	Ctx     context.Context
	Payload string
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

func (s *Shell) Run() error {
	s.cannotSend.Store(false)
	defer func() {
		s.editor.RestoreTermState()
		s.revoke()
	}()
	// we handle backslash in the loop...
	// suppose the line editor can handle one line at every time.
	// In order to properly
	for {
		if !s.isContinued && len(s.quoted) == 0 {
			s.prompt = defaultPrompt
		}
		currLine, err := s.editor.Readline(s.prompt)
		if err != nil {
			if err == io.EOF {
				_, _ = fmt.Fprintf(
					s.writer, ColorInfo+"Received EOF, exit."+ColorReset+"\r\n",
				)
				return err
			}
			return fmt.Errorf("read input failed: %v", err)
		}

		// TODO: determine whether to maintain multiple line for business protocol.
		currLine = strings.TrimSpace(currLine)
		var (
			isUnclosed bool
			quoteType  string
			choice     rune = 0
			lenBuf          = len(currLine)
		)
		if len(s.quoted) > 0 {
			choice = rune(s.quoted[0])
		}
		isUnclosed, quoteType = checkQuoteState(choice, currLine)
		if isUnclosed {
			s.quoted = quoteType
			s.prompt = ColorWarn + "quote " + quoteType + "> " + ColorReset
			s.isContinued = true
		} else if len(s.quoted) > 0 && !isUnclosed {
			s.quoted = ""
		}

		if lenBuf > 0 && currLine[lenBuf-1] == '\\' {
			// we have to iterate \ until recognizing it is a broken command
			// or indeed an odd number of backslash
			test := backslashSeqChecker(currLine)
			if test < 0 {
				_, _ = fmt.Fprint(
					s.writer,
					ColorInfo+"sh: invalid command found: "+
						s.tmpDanglingBuf+currLine[:lenBuf-1]+
						ColorReset+"\r\n",
				)
				s.isContinued = false
				s.quoted = ""
				s.tmpDanglingBuf = "" // remember to flush all
				continue
			}
			s.isContinued = test == 1
			currLine = currLine[:lenBuf-1]
		} else {
			if len(s.quoted) == 0 {
				s.isContinued = false
			}
		}

		if s.isContinued { // only odd backslash
			s.tmpDanglingBuf += currLine
			if len(s.quoted) == 0 {
				s.prompt = ColorWarn + "> " + ColorReset
			}
			continue
		} // else {
		if len(s.tmpDanglingBuf) > 0 {
			s.tmpDanglingBuf += currLine
		} else {
			s.tmpDanglingBuf = currLine
		}
		s.prompt = defaultPrompt
		// }

		if len(s.tmpDanglingBuf) <= len("logout") && quitSet.Contains(strings.ToLower(s.tmpDanglingBuf)) {
			_, _ = fmt.Fprint(s.writer, ColorInfo+"bye"+ColorReset+"\r\n")
			return nil
		}
		ctxTimeout, cancel := context.WithCancel(context.Background())
		s.currCmd <- &ShellSyncObj{Ctx: ctxTimeout, Payload: s.tmpDanglingBuf}
		select {
		case resp := <-s.currResp:
			_, _ = fmt.Fprintf(s.writer, "%s\r\n", resp)
			cancel()
		case <-time.After(time.Second * 5):
			cancel()
			// timeout mode
			_, _ = fmt.Fprintf(
				s.writer,
				ColorInfo+"sh: command not found: %s"+ColorReset+"\r\n",
				s.tmpDanglingBuf,
			)
		}
		s.tmpDanglingBuf = "" // remember to flush all
	}
}

// SetCurrResp will check whether the shell is terminated,
// and then use WaitGroup to wait remaining response
func (s *Shell) SetCurrResp(resp *ShellSyncObj) {
	if resp == nil || s.cannotSend.Load() {
		return
	}
	s.wg.Go(func() {
		select {
		case s.currResp <- resp.Payload:
		case <-resp.Ctx.Done():
		}
	})
}

// revoke is used to close the channel.
func (s *Shell) revoke() {
	s.cannotSend.Store(true)
	close(s.currCmd)
	s.wg.Wait()
	close(s.currResp)
}

func (s *Shell) GetCurrCmd() *ShellSyncObj {
	res := <-s.currCmd
	return res
}

// NewShell creates a shell for abstract RW entities.
func NewShell(r io.Reader, w io.Writer) *Shell {
	return &Shell{
		editor:         NewLineEditor(r, w),
		writer:         w,
		prompt:         defaultPrompt,
		tmpDanglingBuf: "",
		isContinued:    false,
		currResp:       make(chan string),
		currCmd:        make(chan *ShellSyncObj),
	}
}
