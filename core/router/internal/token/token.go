package token

import "strconv"

// Type is a specific type of int which describes the symbols.
type Type int

// String returns the name of the token type, e.g. "LBRACE" for the '{' symbol.
// It returns the numeric value of an unknown type as it is.
func (t Type) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}

	return strconv.Itoa(int(t))
}

// Token describes the letter(s) or symbol, is a result of the lexer.
type Token struct {
	Type    Type
	Literal string
	Start   int // including the first char
	End     int // including the last char
}

// /about/{fullname:alphabetical}
// /profile/{anySpecialName:string}
// {id:uint64 range(1,5) else 404}
// /admin/{id:int eq(1) else 402}
// /file/{filepath:file else 405}
const (
	EOF = iota // 0
	ILLEGAL

	// Identifiers + literals
	LBRACE // {
	RBRACE // }
	//	PARAM_IDENTIFIER // id
	COLON  // :
	LPAREN // (
	RPAREN // )
	//	PARAM_FUNC_ARG   // 1
	COMMA
	IDENT // string or keyword
	// Keywords
	// keywords_start
	ELSE // else
	// keywords_end
	INT // 42
)

var typeNames = map[Type]string{
	EOF:     "EOF",
	ILLEGAL: "ILLEGAL",
	LBRACE:  "LBRACE",
	RBRACE:  "RBRACE",
	COLON:   "COLON",
	LPAREN:  "LPAREN",
	RPAREN:  "RPAREN",
	COMMA:   "COMMA",
	IDENT:   "IDENT",
	ELSE:    "ELSE",
	INT:     "INT",
}

var keywords = map[string]Type{
	"else": ELSE,
}

// LookupIdent receives a series of chars
// and tries to resolves the token type.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
