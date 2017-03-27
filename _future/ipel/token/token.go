package token

type TokenType int

type Token struct {
	Type    TokenType
	Literal string
	Start   int // excluding, useful for user
	End     int // excluding, useful for user and index
}

func (t Token) StartIndex() int {
	if t.Start > 0 {
		return t.Start + 1
	}
	return t.Start
}

func (t Token) EndIndex() int {
	return t.End
}

// {id:int range(1,5) else 404}
// /admin/{id:int eq(1) else 402}
// /file/{filepath:tail else 405}
const (
	EOF = iota // 0
	ILLEGAL

	// Identifiers + literals
	LBRACE // {
	RBRACE // }
	//	PARAM_IDENTIFIER // id
	COLON // :
	// let's take them in parser
	//	PARAM_TYPE       // int, string, alphabetic, tail
	//	PARAM_FUNC       // range
	LPAREN // (
	RPAREN // )
	//	PARAM_FUNC_ARG   // 1
	COMMA
	IDENT // string or keyword
	// Keywords
	keywords_start
	ELSE // else
	keywords_end
	INT // 42

)

const eof rune = 0

var keywords = map[string]TokenType{
	"else": ELSE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
