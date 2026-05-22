//go:build tools
// +build tools

// Code generated from RDTparser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package main // RDTparser

import "github.com/antlr4-go/antlr/v4"

// BaseRDTparserListener is a complete listener for a parse tree produced by RDTparser.
type BaseRDTparserListener struct{}

var _ RDTparserListener = &BaseRDTparserListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseRDTparserListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseRDTparserListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseRDTparserListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseRDTparserListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterTable_name is called when production table_name is entered.
func (s *BaseRDTparserListener) EnterTable_name(ctx *Table_nameContext) {}

// ExitTable_name is called when production table_name is exited.
func (s *BaseRDTparserListener) ExitTable_name(ctx *Table_nameContext) {}

// EnterField_name is called when production field_name is entered.
func (s *BaseRDTparserListener) EnterField_name(ctx *Field_nameContext) {}

// ExitField_name is called when production field_name is exited.
func (s *BaseRDTparserListener) ExitField_name(ctx *Field_nameContext) {}

// EnterAlias_name is called when production alias_name is entered.
func (s *BaseRDTparserListener) EnterAlias_name(ctx *Alias_nameContext) {}

// ExitAlias_name is called when production alias_name is exited.
func (s *BaseRDTparserListener) ExitAlias_name(ctx *Alias_nameContext) {}

// EnterIdx_name is called when production idx_name is entered.
func (s *BaseRDTparserListener) EnterIdx_name(ctx *Idx_nameContext) {}

// ExitIdx_name is called when production idx_name is exited.
func (s *BaseRDTparserListener) ExitIdx_name(ctx *Idx_nameContext) {}

// EnterLiteral is called when production literal is entered.
func (s *BaseRDTparserListener) EnterLiteral(ctx *LiteralContext) {}

// ExitLiteral is called when production literal is exited.
func (s *BaseRDTparserListener) ExitLiteral(ctx *LiteralContext) {}

// EnterField_type is called when production field_type is entered.
func (s *BaseRDTparserListener) EnterField_type(ctx *Field_typeContext) {}

// ExitField_type is called when production field_type is exited.
func (s *BaseRDTparserListener) ExitField_type(ctx *Field_typeContext) {}

// EnterTime_unit is called when production time_unit is entered.
func (s *BaseRDTparserListener) EnterTime_unit(ctx *Time_unitContext) {}

// ExitTime_unit is called when production time_unit is exited.
func (s *BaseRDTparserListener) ExitTime_unit(ctx *Time_unitContext) {}

// EnterField_attrs is called when production field_attrs is entered.
func (s *BaseRDTparserListener) EnterField_attrs(ctx *Field_attrsContext) {}

// ExitField_attrs is called when production field_attrs is exited.
func (s *BaseRDTparserListener) ExitField_attrs(ctx *Field_attrsContext) {}

// EnterField_attr is called when production field_attr is entered.
func (s *BaseRDTparserListener) EnterField_attr(ctx *Field_attrContext) {}

// ExitField_attr is called when production field_attr is exited.
func (s *BaseRDTparserListener) ExitField_attr(ctx *Field_attrContext) {}

// EnterAttr_tuples is called when production attr_tuples is entered.
func (s *BaseRDTparserListener) EnterAttr_tuples(ctx *Attr_tuplesContext) {}

// ExitAttr_tuples is called when production attr_tuples is exited.
func (s *BaseRDTparserListener) ExitAttr_tuples(ctx *Attr_tuplesContext) {}

// EnterTables is called when production tables is entered.
func (s *BaseRDTparserListener) EnterTables(ctx *TablesContext) {}

// ExitTables is called when production tables is exited.
func (s *BaseRDTparserListener) ExitTables(ctx *TablesContext) {}

// EnterSource_file is called when production source_file is entered.
func (s *BaseRDTparserListener) EnterSource_file(ctx *Source_fileContext) {}

// ExitSource_file is called when production source_file is exited.
func (s *BaseRDTparserListener) ExitSource_file(ctx *Source_fileContext) {}
