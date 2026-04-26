package terminal

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
/// Last modified at 2026/04/24 星期五 21:33:22

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

var (
	pseudoErrKeepReading              = errors.New("keep reading")
	pseudoErrNewCR                    = errors.New("new cr")
	pseudoErrNewLF                    = errors.New("new lf")
	pseudoErrNewLine                  = errors.New("newline")
	pseudoErrSuspectMultiBytesCtrlSeq = errors.New("suspect control sequence")
	pseudoErrSuspectUtf8              = errors.New("suspect utf8")
)

type LineEditor struct {
	reader    *bufio.Reader
	termState *term.State
	Writer    io.Writer
	debugger  *os.File
	prompt    string

	buf        []rune
	tmpKeep    []byte
	curPos     int
	lastRow    int
	lastTotCnt int

	// configuration options for line editor
	MaxCharCntPerLine int
	maxBufLen         int

	fullCRLF bool
	LFasCRLF bool
	holdCR   bool
}

// SetTermState will turn termState into raw mode, and store the old state value
func (l *LineEditor) SetTermState(r io.Reader) {
	// FIXME: unix-only
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		state, err := term.MakeRaw(int(f.Fd()))
		if err == nil {
			l.termState = state
		}
	}
}

// RestoreTermState recovers the previous terminal states
func (l *LineEditor) RestoreTermState() {
	if l.termState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), l.termState)
	}
}

func (l *LineEditor) ResetBuf() {
	l.buf = []rune{}
	l.curPos = 0
	l.lastRow = 0
	// l.lastTotCnt = 0
}

func (l *LineEditor) Write(args ...any) {
	if len(args) == 0 {
		return
	}
	var sb strings.Builder
	for _, a := range args {
		switch a.(type) {
		case string:
			sb.WriteString(a.(string))
		case rune:
			sb.WriteRune(a.(rune))
		default:
			continue
		}
	}
	_, _ = l.Writer.Write([]byte(sb.String()))
}

// Writeln automatically inserts "\r\n" after [args] and then invokes function Write
func (l *LineEditor) Writeln(args ...any) {
	args = append(args, "\r\n")
	l.Write(args...)
}

// TrimToRange is like a threshold-limitor:
// for any given x, compress them into boundary [0, len(l.buf)]
// when x < 0 or x > len(l.buf)
func (l *LineEditor) TrimToRange(x int) int {
	return min(max(0, x), len(l.buf))
}

func (l *LineEditor) moveCursor(delta int) {
	l.curPos = l.TrimToRange(l.curPos + delta)
}

func (l *LineEditor) InsertRune(c rune) {
	if len(l.buf) > l.maxBufLen {
		return
	}
	t := l.TrimToRange(l.curPos)
	newBuf := make([]rune, 0, len(l.buf)+1)
	newBuf = append(newBuf, l.buf[:t]...)
	newBuf = append(newBuf, c)
	newBuf = append(newBuf, l.buf[t:]...)
	l.buf = newBuf
	l.moveCursor(1)
	l.renderLine()
}

func (l *LineEditor) InsertString(s string) {
	if len(l.buf) > l.maxBufLen {
		return
	}
	t := l.TrimToRange(l.curPos)
	newBuf := make([]rune, 0, len(l.buf)+len(s))
	newBuf = append(newBuf, l.buf[:t]...)
	newBuf = append(newBuf, []rune(s)...)
	newBuf = append(newBuf, l.buf[t:]...)
	l.buf = newBuf
	l.moveCursor(len(s))
	l.renderLine()
}

type curDirection int

const (
	CurLeft curDirection = iota
	CurRight
)

func (l *LineEditor) DeleteCharNearCursor(d curDirection) {
	switch d {
	case CurLeft:
		prev, pos := l.TrimToRange(l.curPos-1), l.TrimToRange(l.curPos)
		bufL, bufR := l.buf[:prev], l.buf[pos:]
		l.buf = append(bufL, bufR...)
		l.moveCursor(-1)
		/*
			if l.debugger != nil {
				_, _ = l.debugger.WriteString("<<<<<<<<<<<<")
			}
		*/
		l.renderLine()
	case CurRight:
		peek := l.TrimToRange(l.curPos + 1)
		bufL, bufR := l.buf[:l.curPos], l.buf[peek:]
		l.buf = append(bufL, bufR...)
		l.curPos = l.TrimToRange(l.curPos)
		/*
			if l.debugger != nil {
				_, _ = l.debugger.WriteString(">>>>>>>>>>>>>")
			}
		*/
		l.renderLine()
	default:
		return
	}
}

func (l *LineEditor) delToPrevWord() {
	if l.curPos == 0 || len(l.buf) == 0 {
		return
	}
	pos := l.TrimToRange(l.curPos - 1)
	for pos >= 0 && unicode.IsSpace(l.buf[pos]) {
		pos--
	}
	for pos >= 0 && !unicode.IsSpace(l.buf[pos]) {
		pos--
	}
	pos = l.TrimToRange(pos + 1)
	l.buf = append(l.buf[:pos], l.buf[l.curPos:]...)
	l.curPos = pos
	/*
		if l.debugger != nil {
			_, _ = l.debugger.WriteString("<--<-<-<-")
		}
	*/
	l.renderLine()
}

func (l *LineEditor) drawContent(promptLen int) ([]any, int) {
	var (
		ret     []any
		headPad []rune
	)
	for range promptLen {
		headPad = append(headPad, '\000')
	}
	totalLen := promptLen + len(l.buf)
	totRow, totCol := totalLen/l.MaxCharCntPerLine, totalLen%l.MaxCharCntPerLine
	st := 0
	l.lastTotCnt = totalLen
	l.buf = append(headPad, l.buf...) // !
	for ; st < totRow; st++ {
		i := st * l.MaxCharCntPerLine
		ret = append(ret, string(l.buf[i:i+l.MaxCharCntPerLine]), "\r\n")
	}
	st *= l.MaxCharCntPerLine
	ret = append(ret, string(l.buf[st:st+totCol]))
	return ret, totRow
}

// renderLine maintains the rendering states of current line
func (l *LineEditor) renderLine() {
	l.resetLineState()
	promptLen := len(l.prompt) - ColorCtrlMinusOfs
	t, totRow := l.drawContent(promptLen)
	payload := append([]any{"\r\033[2K", l.prompt}, t...)
	promptedLen := promptLen + l.curPos
	currRow, currCol := promptedLen/l.MaxCharCntPerLine, promptedLen%l.MaxCharCntPerLine
	for i := 0; i < totRow; i++ {
		payload = append(payload, "\r\033[A")
	}
	for i := 0; i < currRow; i++ {
		payload = append(payload, "\r\033[B")
	}
	l.lastRow = currRow
	currRow *= l.MaxCharCntPerLine
	skip := runewidth.StringWidth(string(l.buf[currRow : currRow+currCol]))
	l.buf = l.buf[promptLen:] // end of !
	if currRow == 0 {
		skip += promptLen
	}
	if skip > 0 {
		payload = append(payload, "\r\033[", strconv.Itoa(skip), "C")
	} else {
		payload = append(payload, "\r")
	}
	l.Write(payload...)
}

// resetLineState resets the terminal line display state and cursor position
// It calculates the row differences between previous and current content,
// generates ANSI control codes to clear obsolete lines, and repositions cursor
func (l *LineEditor) resetLineState() {
	lastRowCnt := l.lastTotCnt / l.MaxCharCntPerLine
	currRow := (len(l.prompt) - ColorCtrlMinusOfs + l.curPos) / l.MaxCharCntPerLine
	payload := make([]any, 0)
	if currRow < l.lastRow {
		for i := l.lastRow; i > currRow; i-- {
			payload = append(payload, "\r\033[2K\033[A")
		}
	}
	i := min(currRow, lastRowCnt)
	switch {
	case currRow <= lastRowCnt:
		for i = min(i, l.lastRow); i <= max(lastRowCnt, currRow); i++ {
			payload = append(payload, "\r\033[2K\033[B")
		}
	default:
		payload = append(payload, "\r\033[2K")
	}
	for ; i > 0; i-- {
		payload = append(payload, "\r\033[A\033[2K")
	}

	/*
		if l.debugger != nil {
			var sb strings.Builder
			sb.WriteString("lR cR lRc|cur|lp|tf: ")
			sb.WriteString(strconv.Itoa(l.lastRow))
			sb.WriteRune('\t')
			sb.WriteString(strconv.Itoa(currRow))
			sb.WriteRune('\t')
			sb.WriteString(strconv.Itoa(lastRowCnt))
			sb.WriteRune('\t')
			sb.WriteString(strconv.Itoa(l.curPos))
			sb.WriteRune('\t')
			sb.WriteString(strconv.Itoa(len(payload)))
			sb.WriteRune('\t')
			for _, b := range bugCh { // bugCh := make([]bool, 0)
				if b {
					sb.WriteRune('^')
				} else {
					sb.WriteRune('v')
				}
			}
			sb.WriteRune(' ')
			sb.WriteString(strconv.Itoa((len(l.prompt) - ColorCtrlMinusOfs + l.curPos) % l.MaxCharCntPerLine))
			sb.WriteRune('\n')
			_, _ = l.debugger.WriteString(sb.String())
		}
	*/

	// return to the zero line
	l.Write(payload...)
}

func (l *LineEditor) renderCursor(
	currRow, currCol int,
	curDirect curDirection,
) {
	payload := make([]any, 0)

	switch curDirect {
	case CurLeft:
		if currRow < l.lastRow {
			payload = append(payload, "\r\033[", strconv.Itoa(l.lastRow), "A")
		}
		if currRow > 0 && len(payload) > 0 {
			payload = append(payload, "\r\033[", strconv.Itoa(currRow), "B")
		}
	case CurRight:
		if currRow > l.lastRow && l.lastRow > 0 {
			payload = append(payload, "\r\033[", strconv.Itoa(l.lastRow), "A")
		}
		if currRow > 0 && (len(payload) > 0 || l.lastRow == 0) {
			payload = append(payload, "\r\033[", strconv.Itoa(currRow), "B")
		}
	}
	var headPad []rune
	promptLen := len(l.prompt) - ColorCtrlMinusOfs
	for range promptLen {
		headPad = append(headPad, '\000')
	}
	l.buf = append(headPad, l.buf...)
	st := l.TrimToRange(l.MaxCharCntPerLine * currRow)
	ed := l.TrimToRange(st + currCol)
	skip := runewidth.StringWidth(string(l.buf[st:ed]))
	if st == 0 {
		skip += promptLen
	}
	if skip > 0 {
		payload = append(payload, "\r\033[", strconv.Itoa(skip), "C")
	} else {
		payload = append(payload, "\r")
	}
	/*
		if l.debugger != nil {
			var sb strings.Builder
			sb.WriteString("lR lT|cR cC|st ed|skip|direc|lP:")
			sb.WriteString(strconv.Itoa(l.lastRow))
			sb.WriteRune(' ')
			sb.WriteString(strconv.Itoa(l.lastTotCnt))
			sb.WriteString("| ")
			sb.WriteString(strconv.Itoa(currRow))
			sb.WriteRune(' ')
			sb.WriteString(strconv.Itoa(currCol))
			sb.WriteString("| ")
			sb.WriteString(strconv.Itoa(st))
			sb.WriteRune(' ')
			sb.WriteString(strconv.Itoa(ed))
			sb.WriteString("| ")
			sb.WriteString(strconv.Itoa(skip))
			if curDirect == CurLeft {
				sb.WriteString(" L")
			} else if curDirect == CurRight {
				sb.WriteString(" R")
			}
			sb.WriteString("| ")
			sb.WriteString(strconv.Itoa(len(payload)))
			sb.WriteRune('\n')
			_, _ = l.debugger.WriteString(sb.String())
		}
	*/
	l.buf = l.buf[promptLen:]
	l.lastRow = currRow
	l.Write(payload...)
}

// jmpToEdOfNextWord moves cursor to the END of the NEXT word
// Similar to shell Alt+Right / word forward jump behavior
func (l *LineEditor) jmpToEdOfNextWord() {
	pos := l.TrimToRange(l.curPos)
	if pos == len(l.buf) {
		pos--
	}
	moveWithAuxFn(
		l.buf, &pos,
		func(x *int) bool { return *x < len(l.buf) && *x >= 0 },
		func(x *int) { *x++ },
	)
	currPos := len(l.prompt) + pos - ColorCtrlMinusOfs
	currRow, currCol := currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine
	l.curPos = pos
	l.renderCursor(currRow, currCol, CurRight)
}

// jmpToStOfPrevWord moves cursor to the START of the PREVIOUS word
// Similar to shell Alt+Left / word backward jump behavior
func (l *LineEditor) jmpToStOfPrevWord() {
	pos := l.TrimToRange(l.curPos)
	if pos == len(l.buf) {
		pos--
	}
	moveWithAuxFn(
		l.buf, &pos,
		func(x *int) bool { return *x >= 0 && *x < len(l.buf) },
		func(x *int) { *x-- },
	)
	currPos := len(l.prompt) + pos - ColorCtrlMinusOfs
	currRow, currCol := currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine
	l.curPos = pos

	l.renderCursor(currRow, currCol, CurLeft)
}

// moveWithAuxFn is a reusable helper for word-boundary cursor movement
// It skips current word, then skips trailing spaces to land at next word boundary
func moveWithAuxFn(
	buf []rune, pos *int,
	check func(*int) bool,
	act func(*int),
) {
	if pos == nil || check == nil || act == nil {
		// won't move at all if such condition is satisfied
		return
	}
	defer func() {
		for check(pos) && unicode.IsSpace(buf[*pos]) {
			act(pos)
		}
		*pos = max(0, min(*pos, len(buf)))
	}()
	if check(pos) && unicode.IsSpace(buf[*pos]) {
		return
	}
	for check(pos) && !unicode.IsSpace(buf[*pos]) {
		act(pos)
	}
}

// AlterIOsrc is not thread-safe currently since the handling task is sequential now
func (l *LineEditor) AlterIOsrc(r io.Reader, w io.Writer) {
	l.reader = bufio.NewReader(r)
	l.Writer, l.curPos = w, 0
	l.lastRow = 0
}

// Readline handles for attempting to convert bytes stream into utf8 characters
// err ~ { pseudoErrNewLine, io.EOF }
func (l *LineEditor) Readline(mulLineRequired bool) (string, error) {
	var rec = make([]any, 0)
	rec = append(rec, '\r')
	if mulLineRequired && l.fullCRLF {
		rec = append(rec, '\n')
	}
	rec = append(rec, l.prompt)
	l.Write(rec...)
	l.ResetBuf()
	for {
		r, err := l.byteSeqToRunes()
		if err == nil { // control + C
			l.Write(rec...)
			l.ResetBuf()
			return "", nil
		} else if errors.Is(err, io.EOF) { // exit
			return string(r), err
		} else if errors.Is(err, pseudoErrNewLine) {
			res := string(l.buf)
			l.ResetBuf()
			return res, err
		} else if errors.Is(err, pseudoErrKeepReading) {
			continue
		} else { // error generate from e.reader.Read
			return "", err
		}
	}
}

func (l *LineEditor) byteSeqToRunes() ([]rune, error) {
	var tmp []byte = nil
	if len(l.tmpKeep) > 0 {
		tmp = append(tmp, l.tmpKeep...)
		l.tmpKeep = make([]byte, 0)
		for st := len(tmp); st < 6; st++ {
			tmp = append(tmp, 0)
		}
	} else {
		tmp = make([]byte, 6)
	}

	n, err := l.reader.Read(tmp)
	if err != nil {
		return nil, err
	}
	for st := 0; st < n; {
		payload, err := l.handleCtrlSeq(tmp[st]) // , prompt
		if err == nil {                          // control + C
			return nil, nil
		} else if errors.Is(err, pseudoErrNewCR) {
			var (
				start     = st + 1
				pseudoErr = pseudoErrKeepReading
			)
			if start < n && tmp[start] == '\n' {
				l.holdCR = false
				pseudoErr = pseudoErrNewLine
				start++
			} else if l.holdCR {
				// hold, but the next char is not LF
				l.holdCR = false
				if !l.fullCRLF { // which means \r as \r\n
					l.Write("\r\n")
					pseudoErr = pseudoErrNewLine
				}
			}
			for ed := start; ed < n; ed++ {
				l.tmpKeep = append(l.tmpKeep, tmp[ed])
			}
			l.holdCR = true
			// only read before st and we stop parsing here.
			return payload, pseudoErr
		} else if errors.Is(err, pseudoErrNewLF) {
			if l.LFasCRLF {
				l.Write("\r\033[K")
			} else if l.holdCR {
				l.holdCR = false
				l.Write("\r")
			} else {
				l.Write("\r\n")
			}
			for ed := st + 1; ed < n; ed++ {
				l.tmpKeep = append(l.tmpKeep, tmp[ed])
			}
			return payload, pseudoErrNewLine
		} else if errors.Is(err, io.EOF) { // command like exit
			return payload, err
		} else if errors.Is(err, pseudoErrKeepReading) { // normal ascii
			st++
			continue
		} else if errors.Is(err, pseudoErrSuspectMultiBytesCtrlSeq) {
			return l.handleMultiBytes4CtrlSeq(n, tmp, l.buf)
		} else if errors.Is(err, pseudoErrSuspectUtf8) {
			r, skip := utf8.DecodeRune(tmp[st:])
			if r != utf8.RuneError {
				l.InsertRune(r)
				st += skip
				continue
			}
			if utf8.RuneStart(tmp[st]) {
				// so save st till the end
				for ed := st; ed < n; ed++ {
					l.tmpKeep = append(l.tmpKeep, tmp[ed])
				}
				// only read before st and we stop parsing here.
				return nil, pseudoErrKeepReading
			}
			// there is a corner case: telnet will send control sequence like "ff fd"
			// which will hang the server... if we jump the wrong position,
			// then the whole encoding will fail since extra bytes have been read.
			l.InsertRune(utf8.RuneError)
			st += 1
			// give a chance to parse further
		}
	}
	return l.buf, pseudoErrKeepReading
}

// handleCtrlSeq uses error to represent the expecting next-state.
//   - nil for indicating ctrl+C signal
//   - pseudoErrNewLine notifies for immediately handling the incoming new line
//   - pseudoErrKeepReading requests for keeping reading the printable characters
//   - pseudoErrSuspectMultiBytesCtrlSeq requests for interpreting the multibyte control sequence
//   - pseudoErrSuspectUtf8 suspects that current character may belong to utf-8 char-set
func (l *LineEditor) handleCtrlSeq(r byte) ([]rune, error) {
	switch {
	case r == 0x01: // ctrl + a
		l.renderCursor(0, len(l.prompt)-ColorCtrlMinusOfs, CurLeft)
	case r == 0x03: // ctrl + c
		l.renderCursor(len(l.buf)/l.MaxCharCntPerLine, (len(l.buf)%l.MaxCharCntPerLine)+1, CurRight)
		if l.fullCRLF {

		} else {
			l.Write("^C\r\n")
		}
		l.ResetBuf()
		return nil, nil
	case r == 0x05: // ctrl + e, jump to the end of line
		// e.curCol = e.tailCol
		currPos := len(l.buf) + len(l.prompt) - ColorCtrlMinusOfs
		l.curPos = len(l.buf)
		l.renderCursor(currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine, CurRight)
	case r == 0x08 || r == 0x7F: // ctrl + h, same as backspace
		l.DeleteCharNearCursor(CurLeft)

	case r == '\t': // ctrl + i, same as \t
		// [TODO]: add autocompletion suggestions to make the the fake shell like a real one

	case r == '\n': // ctrl + j, same as \n
		return l.buf, pseudoErrNewLF
	case r == 0x0b: // ctrl + k, delete from cursor to the end of line
		l.buf = l.buf[:l.curPos]
		l.renderLine()
	case r == '\r': // ctrl + m, same as \r or 4-bytes utf8...
		return l.buf, pseudoErrNewCR
	case r == 0x11: // ctrl + q
		l.ResetBuf()
		l.lastTotCnt = 0
		return []rune("exit"), io.EOF
	case r == 0x15: // ctrl + u
		l.renderCursor(0, 0, CurLeft)
		l.ResetBuf()
		l.renderLine()
		l.lastTotCnt = 0
	case r == 0x17: // ctrl + w
		l.delToPrevWord()
	case r >= 0x20 && r <= 0x7e:
		t := rune(r)
		l.InsertRune(t)

		return []rune{t}, pseudoErrKeepReading
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

func (l *LineEditor) handleMultiBytes4CtrlSeq(
	n int, tmp []byte, res []rune,
) ([]rune, error) {
	if n < 1 || tmp[0] != 0x1B {
		return nil, pseudoErrKeepReading
	}
	// pre-calculate these exclusive flags
	isEsc := rune(tmp[0]) == 0x1B
	flag2bL := isEsc && n > 1 && tmp[1] == 'b' // {esc, b} <=> ctrl + left arrow
	flag2fR := isEsc && n > 1 && tmp[1] == 'f' // {esc, f} <=> ctrl + right arrow
	lPart3 := isEsc && n > 2 && tmp[1] == '['
	flag3A := lPart3 && tmp[2] == 'A'  // up    arrow
	flag3B := lPart3 && tmp[2] == 'B'  // down  arrow
	flag3Cr := lPart3 && tmp[2] == 'C' // right arrow
	flag3Dl := lPart3 && tmp[2] == 'D' // left arrow
	flag3F := lPart3 && tmp[2] == 'F'  // end keystroke
	flag3H := lPart3 && tmp[2] == 'H'  // home keystroke
	flag4 := isEsc && n > 3 &&
		string(tmp[2:4]) == "3~" // delete keystroke
	lPart3 = isEsc && n == 6 && string(tmp[2:4]) == "1;" &&
		(rune(tmp[4]) == '3' || rune(tmp[4]) == '5')
	flag6ctrlR := lPart3 && rune(tmp[5]) == 'C' // esc+1+;+{3,5}+C <=> ctrl + right arrow
	flag6ctrlL := lPart3 && rune(tmp[5]) == 'D' // esc+1+;+{3,5}+D <=> ctrl + left  arrow

	switch {
	case flag2bL || flag6ctrlL:
		l.jmpToStOfPrevWord()
	case flag2fR || flag6ctrlR:
		l.jmpToEdOfNextWord()
	case flag3A || flag3B:
	case flag3Cr:
		l.moveCursor(1)
		currPos := l.curPos + len(l.prompt) - ColorCtrlMinusOfs
		l.renderCursor(currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine, CurRight)
	case flag3Dl:
		l.moveCursor(-1)
		currPos := l.curPos + len(l.prompt) - ColorCtrlMinusOfs
		l.renderCursor(currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine, CurLeft)
	case flag3H: // home keystroke
		currPos := len(l.prompt) - ColorCtrlMinusOfs
		l.curPos = 0
		l.renderCursor(currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine, CurLeft)
	case flag3F: // end keystroke
		currPos := len(l.buf) + len(l.prompt) - ColorCtrlMinusOfs
		l.curPos = len(l.buf)
		l.renderCursor(currPos/l.MaxCharCntPerLine, currPos%l.MaxCharCntPerLine, CurRight)
	case flag4:
		l.DeleteCharNearCursor(CurRight)
	default:
		return nil, pseudoErrKeepReading
	}
	return res, pseudoErrKeepReading
}

func (l *LineEditor) SetDebuggerDstLog(output *os.File) { l.debugger = output }

func NewLineEditor(r io.Reader, w io.Writer, LFasCRLF, fullCRLF bool) *LineEditor {
	e := &LineEditor{
		buf:               make([]rune, 0),
		curPos:            0,
		lastTotCnt:        0,
		reader:            bufio.NewReader(r),
		Writer:            w,
		MaxCharCntPerLine: 64,
		LFasCRLF:          LFasCRLF,
		fullCRLF:          fullCRLF,
		maxBufLen:         256,
	}
	e.SetTermState(r)
	return e
}
