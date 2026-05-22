//go:build tools
// +build tools

// Code generated from RDTlexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

package main

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type RDTlexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var RDTlexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func rdtlexerLexerInit() {
	staticData := &RDTlexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'notnull'", "'primary'", "'autoincrement'", "'unique'", "'foreign'",
		"'counter'", "'default'", "'index'", "'createtime'", "'updatetime'",
		"'ms'", "'us'", "'ns'", "'text'", "'bytes'", "'integer'", "'strict'",
		"", "", "':'", "'->'", "','", "';'", "'('", "')'",
	}
	staticData.SymbolicNames = []string{
		"", "NOTNULL", "PRIMARY", "AUTOINCREMENT", "UNIQUE", "FOREIGN", "COUNTER",
		"DEFAULT", "INDEX", "CREATETIME", "UPDATETIME", "MS", "US", "NS", "TEXT",
		"BYTES", "INTEGER", "STRICT", "INT_NUMBER", "ID", "COLON", "POINT_TO",
		"COMMA", "SEMICOLON", "LPAREN", "RPAREN", "WS", "LINE_COMMENT",
	}
	staticData.RuleNames = []string{
		"NOTNULL", "PRIMARY", "AUTOINCREMENT", "UNIQUE", "FOREIGN", "COUNTER",
		"DEFAULT", "INDEX", "CREATETIME", "UPDATETIME", "MS", "US", "NS", "TEXT",
		"BYTES", "INTEGER", "STRICT", "INT_NUMBER", "ID", "COLON", "POINT_TO",
		"COMMA", "SEMICOLON", "LPAREN", "RPAREN", "WS", "LINE_COMMENT",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 27, 223, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1,
		2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1,
		3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1,
		5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1,
		6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1,
		8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 1,
		9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 11,
		1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 13, 1, 13, 1,
		14, 1, 14, 1, 14, 1, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 15, 1, 15, 1, 15,
		1, 15, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1, 16, 1, 16, 1, 16, 1, 16, 1,
		17, 1, 17, 1, 17, 4, 17, 183, 8, 17, 11, 17, 12, 17, 184, 3, 17, 187, 8,
		17, 1, 18, 1, 18, 4, 18, 191, 8, 18, 11, 18, 12, 18, 192, 1, 19, 1, 19,
		1, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 22, 1, 22, 1, 23, 1, 23, 1, 24, 1,
		24, 1, 25, 4, 25, 209, 8, 25, 11, 25, 12, 25, 210, 1, 25, 1, 25, 1, 26,
		1, 26, 5, 26, 217, 8, 26, 10, 26, 12, 26, 220, 9, 26, 1, 26, 1, 26, 0,
		0, 27, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10,
		21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33, 17, 35, 18, 37, 19,
		39, 20, 41, 21, 43, 22, 45, 23, 47, 24, 49, 25, 51, 26, 53, 27, 1, 0, 6,
		1, 0, 49, 57, 1, 0, 48, 57, 3, 0, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57,
		65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13, 13, 32, 32, 4, 0, 10, 10, 13,
		13, 133, 133, 8232, 8233, 227, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5,
		1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13,
		1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0,
		21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0,
		0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0,
		0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 43, 1, 0,
		0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1, 0, 0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1,
		0, 0, 0, 0, 53, 1, 0, 0, 0, 1, 55, 1, 0, 0, 0, 3, 63, 1, 0, 0, 0, 5, 71,
		1, 0, 0, 0, 7, 85, 1, 0, 0, 0, 9, 92, 1, 0, 0, 0, 11, 100, 1, 0, 0, 0,
		13, 108, 1, 0, 0, 0, 15, 116, 1, 0, 0, 0, 17, 122, 1, 0, 0, 0, 19, 133,
		1, 0, 0, 0, 21, 144, 1, 0, 0, 0, 23, 147, 1, 0, 0, 0, 25, 150, 1, 0, 0,
		0, 27, 153, 1, 0, 0, 0, 29, 158, 1, 0, 0, 0, 31, 164, 1, 0, 0, 0, 33, 172,
		1, 0, 0, 0, 35, 186, 1, 0, 0, 0, 37, 188, 1, 0, 0, 0, 39, 194, 1, 0, 0,
		0, 41, 196, 1, 0, 0, 0, 43, 199, 1, 0, 0, 0, 45, 201, 1, 0, 0, 0, 47, 203,
		1, 0, 0, 0, 49, 205, 1, 0, 0, 0, 51, 208, 1, 0, 0, 0, 53, 214, 1, 0, 0,
		0, 55, 56, 5, 110, 0, 0, 56, 57, 5, 111, 0, 0, 57, 58, 5, 116, 0, 0, 58,
		59, 5, 110, 0, 0, 59, 60, 5, 117, 0, 0, 60, 61, 5, 108, 0, 0, 61, 62, 5,
		108, 0, 0, 62, 2, 1, 0, 0, 0, 63, 64, 5, 112, 0, 0, 64, 65, 5, 114, 0,
		0, 65, 66, 5, 105, 0, 0, 66, 67, 5, 109, 0, 0, 67, 68, 5, 97, 0, 0, 68,
		69, 5, 114, 0, 0, 69, 70, 5, 121, 0, 0, 70, 4, 1, 0, 0, 0, 71, 72, 5, 97,
		0, 0, 72, 73, 5, 117, 0, 0, 73, 74, 5, 116, 0, 0, 74, 75, 5, 111, 0, 0,
		75, 76, 5, 105, 0, 0, 76, 77, 5, 110, 0, 0, 77, 78, 5, 99, 0, 0, 78, 79,
		5, 114, 0, 0, 79, 80, 5, 101, 0, 0, 80, 81, 5, 109, 0, 0, 81, 82, 5, 101,
		0, 0, 82, 83, 5, 110, 0, 0, 83, 84, 5, 116, 0, 0, 84, 6, 1, 0, 0, 0, 85,
		86, 5, 117, 0, 0, 86, 87, 5, 110, 0, 0, 87, 88, 5, 105, 0, 0, 88, 89, 5,
		113, 0, 0, 89, 90, 5, 117, 0, 0, 90, 91, 5, 101, 0, 0, 91, 8, 1, 0, 0,
		0, 92, 93, 5, 102, 0, 0, 93, 94, 5, 111, 0, 0, 94, 95, 5, 114, 0, 0, 95,
		96, 5, 101, 0, 0, 96, 97, 5, 105, 0, 0, 97, 98, 5, 103, 0, 0, 98, 99, 5,
		110, 0, 0, 99, 10, 1, 0, 0, 0, 100, 101, 5, 99, 0, 0, 101, 102, 5, 111,
		0, 0, 102, 103, 5, 117, 0, 0, 103, 104, 5, 110, 0, 0, 104, 105, 5, 116,
		0, 0, 105, 106, 5, 101, 0, 0, 106, 107, 5, 114, 0, 0, 107, 12, 1, 0, 0,
		0, 108, 109, 5, 100, 0, 0, 109, 110, 5, 101, 0, 0, 110, 111, 5, 102, 0,
		0, 111, 112, 5, 97, 0, 0, 112, 113, 5, 117, 0, 0, 113, 114, 5, 108, 0,
		0, 114, 115, 5, 116, 0, 0, 115, 14, 1, 0, 0, 0, 116, 117, 5, 105, 0, 0,
		117, 118, 5, 110, 0, 0, 118, 119, 5, 100, 0, 0, 119, 120, 5, 101, 0, 0,
		120, 121, 5, 120, 0, 0, 121, 16, 1, 0, 0, 0, 122, 123, 5, 99, 0, 0, 123,
		124, 5, 114, 0, 0, 124, 125, 5, 101, 0, 0, 125, 126, 5, 97, 0, 0, 126,
		127, 5, 116, 0, 0, 127, 128, 5, 101, 0, 0, 128, 129, 5, 116, 0, 0, 129,
		130, 5, 105, 0, 0, 130, 131, 5, 109, 0, 0, 131, 132, 5, 101, 0, 0, 132,
		18, 1, 0, 0, 0, 133, 134, 5, 117, 0, 0, 134, 135, 5, 112, 0, 0, 135, 136,
		5, 100, 0, 0, 136, 137, 5, 97, 0, 0, 137, 138, 5, 116, 0, 0, 138, 139,
		5, 101, 0, 0, 139, 140, 5, 116, 0, 0, 140, 141, 5, 105, 0, 0, 141, 142,
		5, 109, 0, 0, 142, 143, 5, 101, 0, 0, 143, 20, 1, 0, 0, 0, 144, 145, 5,
		109, 0, 0, 145, 146, 5, 115, 0, 0, 146, 22, 1, 0, 0, 0, 147, 148, 5, 117,
		0, 0, 148, 149, 5, 115, 0, 0, 149, 24, 1, 0, 0, 0, 150, 151, 5, 110, 0,
		0, 151, 152, 5, 115, 0, 0, 152, 26, 1, 0, 0, 0, 153, 154, 5, 116, 0, 0,
		154, 155, 5, 101, 0, 0, 155, 156, 5, 120, 0, 0, 156, 157, 5, 116, 0, 0,
		157, 28, 1, 0, 0, 0, 158, 159, 5, 98, 0, 0, 159, 160, 5, 121, 0, 0, 160,
		161, 5, 116, 0, 0, 161, 162, 5, 101, 0, 0, 162, 163, 5, 115, 0, 0, 163,
		30, 1, 0, 0, 0, 164, 165, 5, 105, 0, 0, 165, 166, 5, 110, 0, 0, 166, 167,
		5, 116, 0, 0, 167, 168, 5, 101, 0, 0, 168, 169, 5, 103, 0, 0, 169, 170,
		5, 101, 0, 0, 170, 171, 5, 114, 0, 0, 171, 32, 1, 0, 0, 0, 172, 173, 5,
		115, 0, 0, 173, 174, 5, 116, 0, 0, 174, 175, 5, 114, 0, 0, 175, 176, 5,
		105, 0, 0, 176, 177, 5, 99, 0, 0, 177, 178, 5, 116, 0, 0, 178, 34, 1, 0,
		0, 0, 179, 187, 5, 48, 0, 0, 180, 182, 7, 0, 0, 0, 181, 183, 7, 1, 0, 0,
		182, 181, 1, 0, 0, 0, 183, 184, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184,
		185, 1, 0, 0, 0, 185, 187, 1, 0, 0, 0, 186, 179, 1, 0, 0, 0, 186, 180,
		1, 0, 0, 0, 187, 36, 1, 0, 0, 0, 188, 190, 7, 2, 0, 0, 189, 191, 7, 3,
		0, 0, 190, 189, 1, 0, 0, 0, 191, 192, 1, 0, 0, 0, 192, 190, 1, 0, 0, 0,
		192, 193, 1, 0, 0, 0, 193, 38, 1, 0, 0, 0, 194, 195, 5, 58, 0, 0, 195,
		40, 1, 0, 0, 0, 196, 197, 5, 45, 0, 0, 197, 198, 5, 62, 0, 0, 198, 42,
		1, 0, 0, 0, 199, 200, 5, 44, 0, 0, 200, 44, 1, 0, 0, 0, 201, 202, 5, 59,
		0, 0, 202, 46, 1, 0, 0, 0, 203, 204, 5, 40, 0, 0, 204, 48, 1, 0, 0, 0,
		205, 206, 5, 41, 0, 0, 206, 50, 1, 0, 0, 0, 207, 209, 7, 4, 0, 0, 208,
		207, 1, 0, 0, 0, 209, 210, 1, 0, 0, 0, 210, 208, 1, 0, 0, 0, 210, 211,
		1, 0, 0, 0, 211, 212, 1, 0, 0, 0, 212, 213, 6, 25, 0, 0, 213, 52, 1, 0,
		0, 0, 214, 218, 5, 35, 0, 0, 215, 217, 8, 5, 0, 0, 216, 215, 1, 0, 0, 0,
		217, 220, 1, 0, 0, 0, 218, 216, 1, 0, 0, 0, 218, 219, 1, 0, 0, 0, 219,
		221, 1, 0, 0, 0, 220, 218, 1, 0, 0, 0, 221, 222, 6, 26, 0, 0, 222, 54,
		1, 0, 0, 0, 6, 0, 184, 186, 192, 210, 218, 1, 6, 0, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// RDTlexerInit initializes any static state used to implement RDTlexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewRDTlexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func RDTlexerInit() {
	staticData := &RDTlexerLexerStaticData
	staticData.once.Do(rdtlexerLexerInit)
}

// NewRDTlexer produces a new lexer instance for the optional input antlr.CharStream.
func NewRDTlexer(input antlr.CharStream) *RDTlexer {
	RDTlexerInit()
	l := new(RDTlexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &RDTlexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "RDTlexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// RDTlexer tokens.
const (
	RDTlexerNOTNULL       = 1
	RDTlexerPRIMARY       = 2
	RDTlexerAUTOINCREMENT = 3
	RDTlexerUNIQUE        = 4
	RDTlexerFOREIGN       = 5
	RDTlexerCOUNTER       = 6
	RDTlexerDEFAULT       = 7
	RDTlexerINDEX         = 8
	RDTlexerCREATETIME    = 9
	RDTlexerUPDATETIME    = 10
	RDTlexerMS            = 11
	RDTlexerUS            = 12
	RDTlexerNS            = 13
	RDTlexerTEXT          = 14
	RDTlexerBYTES         = 15
	RDTlexerINTEGER       = 16
	RDTlexerSTRICT        = 17
	RDTlexerINT_NUMBER    = 18
	RDTlexerID            = 19
	RDTlexerCOLON         = 20
	RDTlexerPOINT_TO      = 21
	RDTlexerCOMMA         = 22
	RDTlexerSEMICOLON     = 23
	RDTlexerLPAREN        = 24
	RDTlexerRPAREN        = 25
	RDTlexerWS            = 26
	RDTlexerLINE_COMMENT  = 27
)
