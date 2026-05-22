//go:build tools
// +build tools

// Code generated from RDTparser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package main // RDTparser

import "github.com/antlr4-go/antlr/v4"

// RDTparserListener is a complete listener for a parse tree produced by RDTparser.
type RDTparserListener interface {
	antlr.ParseTreeListener

	// EnterTable_name is called when entering the table_name production.
	EnterTable_name(c *Table_nameContext)

	// EnterField_name is called when entering the field_name production.
	EnterField_name(c *Field_nameContext)

	// EnterAlias_name is called when entering the alias_name production.
	EnterAlias_name(c *Alias_nameContext)

	// EnterIdx_name is called when entering the idx_name production.
	EnterIdx_name(c *Idx_nameContext)

	// EnterLiteral is called when entering the literal production.
	EnterLiteral(c *LiteralContext)

	// EnterField_type is called when entering the field_type production.
	EnterField_type(c *Field_typeContext)

	// EnterTime_unit is called when entering the time_unit production.
	EnterTime_unit(c *Time_unitContext)

	// EnterField_attrs is called when entering the field_attrs production.
	EnterField_attrs(c *Field_attrsContext)

	// EnterField_attr is called when entering the field_attr production.
	EnterField_attr(c *Field_attrContext)

	// EnterAttr_tuples is called when entering the attr_tuples production.
	EnterAttr_tuples(c *Attr_tuplesContext)

	// EnterTables is called when entering the tables production.
	EnterTables(c *TablesContext)

	// EnterSource_file is called when entering the source_file production.
	EnterSource_file(c *Source_fileContext)

	// ExitTable_name is called when exiting the table_name production.
	ExitTable_name(c *Table_nameContext)

	// ExitField_name is called when exiting the field_name production.
	ExitField_name(c *Field_nameContext)

	// ExitAlias_name is called when exiting the alias_name production.
	ExitAlias_name(c *Alias_nameContext)

	// ExitIdx_name is called when exiting the idx_name production.
	ExitIdx_name(c *Idx_nameContext)

	// ExitLiteral is called when exiting the literal production.
	ExitLiteral(c *LiteralContext)

	// ExitField_type is called when exiting the field_type production.
	ExitField_type(c *Field_typeContext)

	// ExitTime_unit is called when exiting the time_unit production.
	ExitTime_unit(c *Time_unitContext)

	// ExitField_attrs is called when exiting the field_attrs production.
	ExitField_attrs(c *Field_attrsContext)

	// ExitField_attr is called when exiting the field_attr production.
	ExitField_attr(c *Field_attrContext)

	// ExitAttr_tuples is called when exiting the attr_tuples production.
	ExitAttr_tuples(c *Attr_tuplesContext)

	// ExitTables is called when exiting the tables production.
	ExitTables(c *TablesContext)

	// ExitSource_file is called when exiting the source_file production.
	ExitSource_file(c *Source_fileContext)
}
