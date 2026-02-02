parser grammar RDTparser;

options {
    tokenVocab = RDTlexer;
}

table_name: ID;
field_name: ID;
alias_name: ID;
idx_name: ID;
literal: ID;

field_type: TEXT | INTEGER;
time_unit: MS | US | NS;

field_attrs: field_attr (COMMA field_attr)*;

field_attr: PRIMARY
           | NOTNULL
           | UNIQUE
           | AUTOINCREMENT
           | COUNTER INT_NUMBER
           | DEFAULT (INT_NUMBER | literal)
           | FOREIGN POINT_TO table_name COLON field_name
           | CREATETIME time_unit
           | UPDATETIME time_unit
           | INDEX idx_name
           ;
           
attr_tuples: field_name COLON field_type LPAREN field_attrs? RPAREN SEMICOLON?;
tables: table_name COLON attr_tuples+ (STRICT)?;

source_file: tables+;