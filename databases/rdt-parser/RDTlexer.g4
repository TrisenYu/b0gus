lexer grammar RDTlexer; // RDT stands for Reduced Database Table

/* column attributes */
NOTNULL:        'notnull';
PRIMARY:        'primary';
AUTOINCREMENT:  'autoincrement';
UNIQUE:         'unique';
FOREIGN:        'foreign';
COUNTER:        'counter';
DEFAULT:        'default';
INDEX:          'index';
CREATETIME:     'createtime';
UPDATETIME:     'updatetime';
MS: 'ms';
US: 'us';
NS: 'ns';

/* columns/values type */
TEXT:       'text';
INTEGER: 'integer';

/* strict mode */
STRICT: 'strict';
INT_NUMBER: '0' | [1-9][0-9]+;

// It is not recommended to use non-ascii characters as column's name
// because regex for non-space unicode codepoint(xid_start) is very long and hard to read or inspect
ID: [a-zA-Z_][a-zA-Z0-9_]+;

COLON:      ':';
POINT_TO:   '->';
COMMA:      ',';
SEMICOLON:  ';';
LPAREN:     '(';
RPAREN:     ')';

WS: [ \t\r\n]+ -> skip;
LINE_COMMENT: '#' ~[\r\n\u0085\u2028\u2029]* -> skip;







