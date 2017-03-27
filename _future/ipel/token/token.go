package token

type TokenType int

type Token struct {
	Type    TokenType
	Literal string
	Start   int // including the first char, Literal[index:]
	End     int // including the last char, Literal[start:end+1)
}

// /about/{fullname:alphabetical}
// /profile/{anySpecialName:string}
// {id:int range(1,5) else 404}
// /admin/{id:int eq(1) else 402}
// /file/{filepath:file else 405}
const (
	EOF = iota // 0
	ILLEGAL

	// Identifiers + literals
	LBRACE // {
	RBRACE // }
	//	PARAM_IDENTIFIER // id
	COLON // :
	// let's take them in parser
	//	PARAM_TYPE       // int, string, alphabetical, file, path or unexpected
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
