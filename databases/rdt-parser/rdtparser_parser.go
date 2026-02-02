// Code generated from RDTparser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package main // RDTparser

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type RDTparser struct {
	*antlr.BaseParser
}

var RDTparserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func rdtparserParserInit() {
	staticData := &RDTparserParserStaticData
	staticData.LiteralNames = []string{
		"", "'notnull'", "'primary'", "'autoincrement'", "'unique'", "'foreign'",
		"'counter'", "'default'", "'index'", "'createtime'", "'updatetime'",
		"'ms'", "'us'", "'ns'", "'text'", "'integer'", "'strict'", "", "", "':'",
		"'->'", "','", "';'", "'('", "')'",
	}
	staticData.SymbolicNames = []string{
		"", "NOTNULL", "PRIMARY", "AUTOINCREMENT", "UNIQUE", "FOREIGN", "COUNTER",
		"DEFAULT", "INDEX", "CREATETIME", "UPDATETIME", "MS", "US", "NS", "TEXT",
		"INTEGER", "STRICT", "INT_NUMBER", "ID", "COLON", "POINT_TO", "COMMA",
		"SEMICOLON", "LPAREN", "RPAREN", "WS", "LINE_COMMENT",
	}
	staticData.RuleNames = []string{
		"table_name", "field_name", "alias_name", "idx_name", "literal", "field_type",
		"time_unit", "field_attrs", "field_attr", "attr_tuples", "tables", "source_file",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 26, 98, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 4,
		1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 5, 7, 42, 8, 7, 10, 7,
		12, 7, 45, 9, 7, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8,
		3, 8, 56, 8, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1, 8, 1,
		8, 1, 8, 1, 8, 3, 8, 70, 8, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 3, 9, 77,
		8, 9, 1, 9, 1, 9, 3, 9, 81, 8, 9, 1, 10, 1, 10, 1, 10, 4, 10, 86, 8, 10,
		11, 10, 12, 10, 87, 1, 10, 3, 10, 91, 8, 10, 1, 11, 4, 11, 94, 8, 11, 11,
		11, 12, 11, 95, 1, 11, 0, 0, 12, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20,
		22, 0, 2, 1, 0, 14, 15, 1, 0, 11, 13, 101, 0, 24, 1, 0, 0, 0, 2, 26, 1,
		0, 0, 0, 4, 28, 1, 0, 0, 0, 6, 30, 1, 0, 0, 0, 8, 32, 1, 0, 0, 0, 10, 34,
		1, 0, 0, 0, 12, 36, 1, 0, 0, 0, 14, 38, 1, 0, 0, 0, 16, 69, 1, 0, 0, 0,
		18, 71, 1, 0, 0, 0, 20, 82, 1, 0, 0, 0, 22, 93, 1, 0, 0, 0, 24, 25, 5,
		18, 0, 0, 25, 1, 1, 0, 0, 0, 26, 27, 5, 18, 0, 0, 27, 3, 1, 0, 0, 0, 28,
		29, 5, 18, 0, 0, 29, 5, 1, 0, 0, 0, 30, 31, 5, 18, 0, 0, 31, 7, 1, 0, 0,
		0, 32, 33, 5, 18, 0, 0, 33, 9, 1, 0, 0, 0, 34, 35, 7, 0, 0, 0, 35, 11,
		1, 0, 0, 0, 36, 37, 7, 1, 0, 0, 37, 13, 1, 0, 0, 0, 38, 43, 3, 16, 8, 0,
		39, 40, 5, 21, 0, 0, 40, 42, 3, 16, 8, 0, 41, 39, 1, 0, 0, 0, 42, 45, 1,
		0, 0, 0, 43, 41, 1, 0, 0, 0, 43, 44, 1, 0, 0, 0, 44, 15, 1, 0, 0, 0, 45,
		43, 1, 0, 0, 0, 46, 70, 5, 2, 0, 0, 47, 70, 5, 1, 0, 0, 48, 70, 5, 4, 0,
		0, 49, 70, 5, 3, 0, 0, 50, 51, 5, 6, 0, 0, 51, 70, 5, 17, 0, 0, 52, 55,
		5, 7, 0, 0, 53, 56, 5, 17, 0, 0, 54, 56, 3, 8, 4, 0, 55, 53, 1, 0, 0, 0,
		55, 54, 1, 0, 0, 0, 56, 70, 1, 0, 0, 0, 57, 58, 5, 5, 0, 0, 58, 59, 5,
		20, 0, 0, 59, 60, 3, 0, 0, 0, 60, 61, 5, 19, 0, 0, 61, 62, 3, 2, 1, 0,
		62, 70, 1, 0, 0, 0, 63, 64, 5, 9, 0, 0, 64, 70, 3, 12, 6, 0, 65, 66, 5,
		10, 0, 0, 66, 70, 3, 12, 6, 0, 67, 68, 5, 8, 0, 0, 68, 70, 3, 6, 3, 0,
		69, 46, 1, 0, 0, 0, 69, 47, 1, 0, 0, 0, 69, 48, 1, 0, 0, 0, 69, 49, 1,
		0, 0, 0, 69, 50, 1, 0, 0, 0, 69, 52, 1, 0, 0, 0, 69, 57, 1, 0, 0, 0, 69,
		63, 1, 0, 0, 0, 69, 65, 1, 0, 0, 0, 69, 67, 1, 0, 0, 0, 70, 17, 1, 0, 0,
		0, 71, 72, 3, 2, 1, 0, 72, 73, 5, 19, 0, 0, 73, 74, 3, 10, 5, 0, 74, 76,
		5, 23, 0, 0, 75, 77, 3, 14, 7, 0, 76, 75, 1, 0, 0, 0, 76, 77, 1, 0, 0,
		0, 77, 78, 1, 0, 0, 0, 78, 80, 5, 24, 0, 0, 79, 81, 5, 22, 0, 0, 80, 79,
		1, 0, 0, 0, 80, 81, 1, 0, 0, 0, 81, 19, 1, 0, 0, 0, 82, 83, 3, 0, 0, 0,
		83, 85, 5, 19, 0, 0, 84, 86, 3, 18, 9, 0, 85, 84, 1, 0, 0, 0, 86, 87, 1,
		0, 0, 0, 87, 85, 1, 0, 0, 0, 87, 88, 1, 0, 0, 0, 88, 90, 1, 0, 0, 0, 89,
		91, 5, 16, 0, 0, 90, 89, 1, 0, 0, 0, 90, 91, 1, 0, 0, 0, 91, 21, 1, 0,
		0, 0, 92, 94, 3, 20, 10, 0, 93, 92, 1, 0, 0, 0, 94, 95, 1, 0, 0, 0, 95,
		93, 1, 0, 0, 0, 95, 96, 1, 0, 0, 0, 96, 23, 1, 0, 0, 0, 8, 43, 55, 69,
		76, 80, 87, 90, 95,
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

// RDTparserInit initializes any static state used to implement RDTparser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewRDTparser(). You can call this function if you wish to initialize the static state ahead
// of time.
func RDTparserInit() {
	staticData := &RDTparserParserStaticData
	staticData.once.Do(rdtparserParserInit)
}

// NewRDTparser produces a new parser instance for the optional input antlr.TokenStream.
func NewRDTparser(input antlr.TokenStream) *RDTparser {
	RDTparserInit()
	this := new(RDTparser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &RDTparserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "RDTparser.g4"

	return this
}

// RDTparser tokens.
const (
	RDTparserEOF           = antlr.TokenEOF
	RDTparserNOTNULL       = 1
	RDTparserPRIMARY       = 2
	RDTparserAUTOINCREMENT = 3
	RDTparserUNIQUE        = 4
	RDTparserFOREIGN       = 5
	RDTparserCOUNTER       = 6
	RDTparserDEFAULT       = 7
	RDTparserINDEX         = 8
	RDTparserCREATETIME    = 9
	RDTparserUPDATETIME    = 10
	RDTparserMS            = 11
	RDTparserUS            = 12
	RDTparserNS            = 13
	RDTparserTEXT          = 14
	RDTparserINTEGER       = 15
	RDTparserSTRICT        = 16
	RDTparserINT_NUMBER    = 17
	RDTparserID            = 18
	RDTparserCOLON         = 19
	RDTparserPOINT_TO      = 20
	RDTparserCOMMA         = 21
	RDTparserSEMICOLON     = 22
	RDTparserLPAREN        = 23
	RDTparserRPAREN        = 24
	RDTparserWS            = 25
	RDTparserLINE_COMMENT  = 26
)

// RDTparser rules.
const (
	RDTparserRULE_table_name  = 0
	RDTparserRULE_field_name  = 1
	RDTparserRULE_alias_name  = 2
	RDTparserRULE_idx_name    = 3
	RDTparserRULE_literal     = 4
	RDTparserRULE_field_type  = 5
	RDTparserRULE_time_unit   = 6
	RDTparserRULE_field_attrs = 7
	RDTparserRULE_field_attr  = 8
	RDTparserRULE_attr_tuples = 9
	RDTparserRULE_tables      = 10
	RDTparserRULE_source_file = 11
)

// ITable_nameContext is an interface to support dynamic dispatch.
type ITable_nameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsTable_nameContext differentiates from other interfaces.
	IsTable_nameContext()
}

type Table_nameContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTable_nameContext() *Table_nameContext {
	var p = new(Table_nameContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_table_name
	return p
}

func InitEmptyTable_nameContext(p *Table_nameContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_table_name
}

func (*Table_nameContext) IsTable_nameContext() {}

func NewTable_nameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Table_nameContext {
	var p = new(Table_nameContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_table_name

	return p
}

func (s *Table_nameContext) GetParser() antlr.Parser { return s.parser }

func (s *Table_nameContext) ID() antlr.TerminalNode {
	return s.GetToken(RDTparserID, 0)
}

func (s *Table_nameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Table_nameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Table_nameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterTable_name(s)
	}
}

func (s *Table_nameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitTable_name(s)
	}
}

func (p *RDTparser) Table_name() (localctx ITable_nameContext) {
	localctx = NewTable_nameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, RDTparserRULE_table_name)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(24)
		p.Match(RDTparserID)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IField_nameContext is an interface to support dynamic dispatch.
type IField_nameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsField_nameContext differentiates from other interfaces.
	IsField_nameContext()
}

type Field_nameContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyField_nameContext() *Field_nameContext {
	var p = new(Field_nameContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_name
	return p
}

func InitEmptyField_nameContext(p *Field_nameContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_name
}

func (*Field_nameContext) IsField_nameContext() {}

func NewField_nameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Field_nameContext {
	var p = new(Field_nameContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_field_name

	return p
}

func (s *Field_nameContext) GetParser() antlr.Parser { return s.parser }

func (s *Field_nameContext) ID() antlr.TerminalNode {
	return s.GetToken(RDTparserID, 0)
}

func (s *Field_nameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Field_nameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Field_nameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterField_name(s)
	}
}

func (s *Field_nameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitField_name(s)
	}
}

func (p *RDTparser) Field_name() (localctx IField_nameContext) {
	localctx = NewField_nameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, RDTparserRULE_field_name)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(26)
		p.Match(RDTparserID)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAlias_nameContext is an interface to support dynamic dispatch.
type IAlias_nameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsAlias_nameContext differentiates from other interfaces.
	IsAlias_nameContext()
}

type Alias_nameContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAlias_nameContext() *Alias_nameContext {
	var p = new(Alias_nameContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_alias_name
	return p
}

func InitEmptyAlias_nameContext(p *Alias_nameContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_alias_name
}

func (*Alias_nameContext) IsAlias_nameContext() {}

func NewAlias_nameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Alias_nameContext {
	var p = new(Alias_nameContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_alias_name

	return p
}

func (s *Alias_nameContext) GetParser() antlr.Parser { return s.parser }

func (s *Alias_nameContext) ID() antlr.TerminalNode {
	return s.GetToken(RDTparserID, 0)
}

func (s *Alias_nameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Alias_nameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Alias_nameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterAlias_name(s)
	}
}

func (s *Alias_nameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitAlias_name(s)
	}
}

func (p *RDTparser) Alias_name() (localctx IAlias_nameContext) {
	localctx = NewAlias_nameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, RDTparserRULE_alias_name)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(28)
		p.Match(RDTparserID)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IIdx_nameContext is an interface to support dynamic dispatch.
type IIdx_nameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsIdx_nameContext differentiates from other interfaces.
	IsIdx_nameContext()
}

type Idx_nameContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIdx_nameContext() *Idx_nameContext {
	var p = new(Idx_nameContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_idx_name
	return p
}

func InitEmptyIdx_nameContext(p *Idx_nameContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_idx_name
}

func (*Idx_nameContext) IsIdx_nameContext() {}

func NewIdx_nameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Idx_nameContext {
	var p = new(Idx_nameContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_idx_name

	return p
}

func (s *Idx_nameContext) GetParser() antlr.Parser { return s.parser }

func (s *Idx_nameContext) ID() antlr.TerminalNode {
	return s.GetToken(RDTparserID, 0)
}

func (s *Idx_nameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Idx_nameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Idx_nameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterIdx_name(s)
	}
}

func (s *Idx_nameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitIdx_name(s)
	}
}

func (p *RDTparser) Idx_name() (localctx IIdx_nameContext) {
	localctx = NewIdx_nameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, RDTparserRULE_idx_name)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(30)
		p.Match(RDTparserID)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILiteralContext is an interface to support dynamic dispatch.
type ILiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ID() antlr.TerminalNode

	// IsLiteralContext differentiates from other interfaces.
	IsLiteralContext()
}

type LiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLiteralContext() *LiteralContext {
	var p = new(LiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_literal
	return p
}

func InitEmptyLiteralContext(p *LiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_literal
}

func (*LiteralContext) IsLiteralContext() {}

func NewLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LiteralContext {
	var p = new(LiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_literal

	return p
}

func (s *LiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *LiteralContext) ID() antlr.TerminalNode {
	return s.GetToken(RDTparserID, 0)
}

func (s *LiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterLiteral(s)
	}
}

func (s *LiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitLiteral(s)
	}
}

func (p *RDTparser) Literal() (localctx ILiteralContext) {
	localctx = NewLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, RDTparserRULE_literal)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(32)
		p.Match(RDTparserID)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IField_typeContext is an interface to support dynamic dispatch.
type IField_typeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	TEXT() antlr.TerminalNode
	INTEGER() antlr.TerminalNode

	// IsField_typeContext differentiates from other interfaces.
	IsField_typeContext()
}

type Field_typeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyField_typeContext() *Field_typeContext {
	var p = new(Field_typeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_type
	return p
}

func InitEmptyField_typeContext(p *Field_typeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_type
}

func (*Field_typeContext) IsField_typeContext() {}

func NewField_typeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Field_typeContext {
	var p = new(Field_typeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_field_type

	return p
}

func (s *Field_typeContext) GetParser() antlr.Parser { return s.parser }

func (s *Field_typeContext) TEXT() antlr.TerminalNode {
	return s.GetToken(RDTparserTEXT, 0)
}

func (s *Field_typeContext) INTEGER() antlr.TerminalNode {
	return s.GetToken(RDTparserINTEGER, 0)
}

func (s *Field_typeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Field_typeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Field_typeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterField_type(s)
	}
}

func (s *Field_typeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitField_type(s)
	}
}

func (p *RDTparser) Field_type() (localctx IField_typeContext) {
	localctx = NewField_typeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, RDTparserRULE_field_type)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(34)
		_la = p.GetTokenStream().LA(1)

		if !(_la == RDTparserTEXT || _la == RDTparserINTEGER) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITime_unitContext is an interface to support dynamic dispatch.
type ITime_unitContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MS() antlr.TerminalNode
	US() antlr.TerminalNode
	NS() antlr.TerminalNode

	// IsTime_unitContext differentiates from other interfaces.
	IsTime_unitContext()
}

type Time_unitContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTime_unitContext() *Time_unitContext {
	var p = new(Time_unitContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_time_unit
	return p
}

func InitEmptyTime_unitContext(p *Time_unitContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_time_unit
}

func (*Time_unitContext) IsTime_unitContext() {}

func NewTime_unitContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Time_unitContext {
	var p = new(Time_unitContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_time_unit

	return p
}

func (s *Time_unitContext) GetParser() antlr.Parser { return s.parser }

func (s *Time_unitContext) MS() antlr.TerminalNode {
	return s.GetToken(RDTparserMS, 0)
}

func (s *Time_unitContext) US() antlr.TerminalNode {
	return s.GetToken(RDTparserUS, 0)
}

func (s *Time_unitContext) NS() antlr.TerminalNode {
	return s.GetToken(RDTparserNS, 0)
}

func (s *Time_unitContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Time_unitContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Time_unitContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterTime_unit(s)
	}
}

func (s *Time_unitContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitTime_unit(s)
	}
}

func (p *RDTparser) Time_unit() (localctx ITime_unitContext) {
	localctx = NewTime_unitContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, RDTparserRULE_time_unit)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(36)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&14336) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IField_attrsContext is an interface to support dynamic dispatch.
type IField_attrsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllField_attr() []IField_attrContext
	Field_attr(i int) IField_attrContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsField_attrsContext differentiates from other interfaces.
	IsField_attrsContext()
}

type Field_attrsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyField_attrsContext() *Field_attrsContext {
	var p = new(Field_attrsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_attrs
	return p
}

func InitEmptyField_attrsContext(p *Field_attrsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_attrs
}

func (*Field_attrsContext) IsField_attrsContext() {}

func NewField_attrsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Field_attrsContext {
	var p = new(Field_attrsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_field_attrs

	return p
}

func (s *Field_attrsContext) GetParser() antlr.Parser { return s.parser }

func (s *Field_attrsContext) AllField_attr() []IField_attrContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IField_attrContext); ok {
			len++
		}
	}

	tst := make([]IField_attrContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IField_attrContext); ok {
			tst[i] = t.(IField_attrContext)
			i++
		}
	}

	return tst
}

func (s *Field_attrsContext) Field_attr(i int) IField_attrContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IField_attrContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IField_attrContext)
}

func (s *Field_attrsContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(RDTparserCOMMA)
}

func (s *Field_attrsContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(RDTparserCOMMA, i)
}

func (s *Field_attrsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Field_attrsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Field_attrsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterField_attrs(s)
	}
}

func (s *Field_attrsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitField_attrs(s)
	}
}

func (p *RDTparser) Field_attrs() (localctx IField_attrsContext) {
	localctx = NewField_attrsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, RDTparserRULE_field_attrs)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(38)
		p.Field_attr()
	}
	p.SetState(43)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == RDTparserCOMMA {
		{
			p.SetState(39)
			p.Match(RDTparserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(40)
			p.Field_attr()
		}

		p.SetState(45)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IField_attrContext is an interface to support dynamic dispatch.
type IField_attrContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PRIMARY() antlr.TerminalNode
	NOTNULL() antlr.TerminalNode
	UNIQUE() antlr.TerminalNode
	AUTOINCREMENT() antlr.TerminalNode
	COUNTER() antlr.TerminalNode
	INT_NUMBER() antlr.TerminalNode
	DEFAULT() antlr.TerminalNode
	Literal() ILiteralContext
	FOREIGN() antlr.TerminalNode
	POINT_TO() antlr.TerminalNode
	Table_name() ITable_nameContext
	COLON() antlr.TerminalNode
	Field_name() IField_nameContext
	CREATETIME() antlr.TerminalNode
	Time_unit() ITime_unitContext
	UPDATETIME() antlr.TerminalNode
	INDEX() antlr.TerminalNode
	Idx_name() IIdx_nameContext

	// IsField_attrContext differentiates from other interfaces.
	IsField_attrContext()
}

type Field_attrContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyField_attrContext() *Field_attrContext {
	var p = new(Field_attrContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_attr
	return p
}

func InitEmptyField_attrContext(p *Field_attrContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_field_attr
}

func (*Field_attrContext) IsField_attrContext() {}

func NewField_attrContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Field_attrContext {
	var p = new(Field_attrContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_field_attr

	return p
}

func (s *Field_attrContext) GetParser() antlr.Parser { return s.parser }

func (s *Field_attrContext) PRIMARY() antlr.TerminalNode {
	return s.GetToken(RDTparserPRIMARY, 0)
}

func (s *Field_attrContext) NOTNULL() antlr.TerminalNode {
	return s.GetToken(RDTparserNOTNULL, 0)
}

func (s *Field_attrContext) UNIQUE() antlr.TerminalNode {
	return s.GetToken(RDTparserUNIQUE, 0)
}

func (s *Field_attrContext) AUTOINCREMENT() antlr.TerminalNode {
	return s.GetToken(RDTparserAUTOINCREMENT, 0)
}

func (s *Field_attrContext) COUNTER() antlr.TerminalNode {
	return s.GetToken(RDTparserCOUNTER, 0)
}

func (s *Field_attrContext) INT_NUMBER() antlr.TerminalNode {
	return s.GetToken(RDTparserINT_NUMBER, 0)
}

func (s *Field_attrContext) DEFAULT() antlr.TerminalNode {
	return s.GetToken(RDTparserDEFAULT, 0)
}

func (s *Field_attrContext) Literal() ILiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILiteralContext)
}

func (s *Field_attrContext) FOREIGN() antlr.TerminalNode {
	return s.GetToken(RDTparserFOREIGN, 0)
}

func (s *Field_attrContext) POINT_TO() antlr.TerminalNode {
	return s.GetToken(RDTparserPOINT_TO, 0)
}

func (s *Field_attrContext) Table_name() ITable_nameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITable_nameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITable_nameContext)
}

func (s *Field_attrContext) COLON() antlr.TerminalNode {
	return s.GetToken(RDTparserCOLON, 0)
}

func (s *Field_attrContext) Field_name() IField_nameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IField_nameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IField_nameContext)
}

func (s *Field_attrContext) CREATETIME() antlr.TerminalNode {
	return s.GetToken(RDTparserCREATETIME, 0)
}

func (s *Field_attrContext) Time_unit() ITime_unitContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITime_unitContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITime_unitContext)
}

func (s *Field_attrContext) UPDATETIME() antlr.TerminalNode {
	return s.GetToken(RDTparserUPDATETIME, 0)
}

func (s *Field_attrContext) INDEX() antlr.TerminalNode {
	return s.GetToken(RDTparserINDEX, 0)
}

func (s *Field_attrContext) Idx_name() IIdx_nameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdx_nameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdx_nameContext)
}

func (s *Field_attrContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Field_attrContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Field_attrContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterField_attr(s)
	}
}

func (s *Field_attrContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitField_attr(s)
	}
}

func (p *RDTparser) Field_attr() (localctx IField_attrContext) {
	localctx = NewField_attrContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, RDTparserRULE_field_attr)
	p.SetState(69)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case RDTparserPRIMARY:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(46)
			p.Match(RDTparserPRIMARY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case RDTparserNOTNULL:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(47)
			p.Match(RDTparserNOTNULL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case RDTparserUNIQUE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(48)
			p.Match(RDTparserUNIQUE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case RDTparserAUTOINCREMENT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(49)
			p.Match(RDTparserAUTOINCREMENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case RDTparserCOUNTER:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(50)
			p.Match(RDTparserCOUNTER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(51)
			p.Match(RDTparserINT_NUMBER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case RDTparserDEFAULT:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(52)
			p.Match(RDTparserDEFAULT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(55)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case RDTparserINT_NUMBER:
			{
				p.SetState(53)
				p.Match(RDTparserINT_NUMBER)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case RDTparserID:
			{
				p.SetState(54)
				p.Literal()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

	case RDTparserFOREIGN:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(57)
			p.Match(RDTparserFOREIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(58)
			p.Match(RDTparserPOINT_TO)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(59)
			p.Table_name()
		}
		{
			p.SetState(60)
			p.Match(RDTparserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(61)
			p.Field_name()
		}

	case RDTparserCREATETIME:
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(63)
			p.Match(RDTparserCREATETIME)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(64)
			p.Time_unit()
		}

	case RDTparserUPDATETIME:
		p.EnterOuterAlt(localctx, 9)
		{
			p.SetState(65)
			p.Match(RDTparserUPDATETIME)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(66)
			p.Time_unit()
		}

	case RDTparserINDEX:
		p.EnterOuterAlt(localctx, 10)
		{
			p.SetState(67)
			p.Match(RDTparserINDEX)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(68)
			p.Idx_name()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAttr_tuplesContext is an interface to support dynamic dispatch.
type IAttr_tuplesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Field_name() IField_nameContext
	COLON() antlr.TerminalNode
	Field_type() IField_typeContext
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	Field_attrs() IField_attrsContext
	SEMICOLON() antlr.TerminalNode

	// IsAttr_tuplesContext differentiates from other interfaces.
	IsAttr_tuplesContext()
}

type Attr_tuplesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAttr_tuplesContext() *Attr_tuplesContext {
	var p = new(Attr_tuplesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_attr_tuples
	return p
}

func InitEmptyAttr_tuplesContext(p *Attr_tuplesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_attr_tuples
}

func (*Attr_tuplesContext) IsAttr_tuplesContext() {}

func NewAttr_tuplesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Attr_tuplesContext {
	var p = new(Attr_tuplesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_attr_tuples

	return p
}

func (s *Attr_tuplesContext) GetParser() antlr.Parser { return s.parser }

func (s *Attr_tuplesContext) Field_name() IField_nameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IField_nameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IField_nameContext)
}

func (s *Attr_tuplesContext) COLON() antlr.TerminalNode {
	return s.GetToken(RDTparserCOLON, 0)
}

func (s *Attr_tuplesContext) Field_type() IField_typeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IField_typeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IField_typeContext)
}

func (s *Attr_tuplesContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(RDTparserLPAREN, 0)
}

func (s *Attr_tuplesContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(RDTparserRPAREN, 0)
}

func (s *Attr_tuplesContext) Field_attrs() IField_attrsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IField_attrsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IField_attrsContext)
}

func (s *Attr_tuplesContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(RDTparserSEMICOLON, 0)
}

func (s *Attr_tuplesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Attr_tuplesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Attr_tuplesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterAttr_tuples(s)
	}
}

func (s *Attr_tuplesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitAttr_tuples(s)
	}
}

func (p *RDTparser) Attr_tuples() (localctx IAttr_tuplesContext) {
	localctx = NewAttr_tuplesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, RDTparserRULE_attr_tuples)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(71)
		p.Field_name()
	}
	{
		p.SetState(72)
		p.Match(RDTparserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(73)
		p.Field_type()
	}
	{
		p.SetState(74)
		p.Match(RDTparserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(76)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&2046) != 0 {
		{
			p.SetState(75)
			p.Field_attrs()
		}

	}
	{
		p.SetState(78)
		p.Match(RDTparserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(80)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == RDTparserSEMICOLON {
		{
			p.SetState(79)
			p.Match(RDTparserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITablesContext is an interface to support dynamic dispatch.
type ITablesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Table_name() ITable_nameContext
	COLON() antlr.TerminalNode
	AllAttr_tuples() []IAttr_tuplesContext
	Attr_tuples(i int) IAttr_tuplesContext
	STRICT() antlr.TerminalNode

	// IsTablesContext differentiates from other interfaces.
	IsTablesContext()
}

type TablesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTablesContext() *TablesContext {
	var p = new(TablesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_tables
	return p
}

func InitEmptyTablesContext(p *TablesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_tables
}

func (*TablesContext) IsTablesContext() {}

func NewTablesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TablesContext {
	var p = new(TablesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_tables

	return p
}

func (s *TablesContext) GetParser() antlr.Parser { return s.parser }

func (s *TablesContext) Table_name() ITable_nameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITable_nameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITable_nameContext)
}

func (s *TablesContext) COLON() antlr.TerminalNode {
	return s.GetToken(RDTparserCOLON, 0)
}

func (s *TablesContext) AllAttr_tuples() []IAttr_tuplesContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IAttr_tuplesContext); ok {
			len++
		}
	}

	tst := make([]IAttr_tuplesContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IAttr_tuplesContext); ok {
			tst[i] = t.(IAttr_tuplesContext)
			i++
		}
	}

	return tst
}

func (s *TablesContext) Attr_tuples(i int) IAttr_tuplesContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAttr_tuplesContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAttr_tuplesContext)
}

func (s *TablesContext) STRICT() antlr.TerminalNode {
	return s.GetToken(RDTparserSTRICT, 0)
}

func (s *TablesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TablesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TablesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterTables(s)
	}
}

func (s *TablesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitTables(s)
	}
}

func (p *RDTparser) Tables() (localctx ITablesContext) {
	localctx = NewTablesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, RDTparserRULE_tables)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(82)
		p.Table_name()
	}
	{
		p.SetState(83)
		p.Match(RDTparserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(85)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = 1
	for ok := true; ok; ok = _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		switch _alt {
		case 1:
			{
				p.SetState(84)
				p.Attr_tuples()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(87)
		p.GetErrorHandler().Sync(p)
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 5, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(90)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == RDTparserSTRICT {
		{
			p.SetState(89)
			p.Match(RDTparserSTRICT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISource_fileContext is an interface to support dynamic dispatch.
type ISource_fileContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllTables() []ITablesContext
	Tables(i int) ITablesContext

	// IsSource_fileContext differentiates from other interfaces.
	IsSource_fileContext()
}

type Source_fileContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySource_fileContext() *Source_fileContext {
	var p = new(Source_fileContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_source_file
	return p
}

func InitEmptySource_fileContext(p *Source_fileContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = RDTparserRULE_source_file
}

func (*Source_fileContext) IsSource_fileContext() {}

func NewSource_fileContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *Source_fileContext {
	var p = new(Source_fileContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = RDTparserRULE_source_file

	return p
}

func (s *Source_fileContext) GetParser() antlr.Parser { return s.parser }

func (s *Source_fileContext) AllTables() []ITablesContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITablesContext); ok {
			len++
		}
	}

	tst := make([]ITablesContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITablesContext); ok {
			tst[i] = t.(ITablesContext)
			i++
		}
	}

	return tst
}

func (s *Source_fileContext) Tables(i int) ITablesContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITablesContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITablesContext)
}

func (s *Source_fileContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *Source_fileContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *Source_fileContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.EnterSource_file(s)
	}
}

func (s *Source_fileContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(RDTparserListener); ok {
		listenerT.ExitSource_file(s)
	}
}

func (p *RDTparser) Source_file() (localctx ISource_fileContext) {
	localctx = NewSource_fileContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, RDTparserRULE_source_file)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(93)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == RDTparserID {
		{
			p.SetState(92)
			p.Tables()
		}

		p.SetState(95)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
