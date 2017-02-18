package lexer

import "fmt"

const (
	// TokenError represents an error
	TokenError TokenKind = iota

	// TokenEOF represents an End Of File
	TokenEOF

	//
	// Mustache delimiters
	//

	// TokenOpen is the OPEN token
	TokenOpen

	// TokenClose is the CLOSE token
	TokenClose

	// TokenOpenRawBlock is the OPEN_RAW_BLOCK token
	TokenOpenRawBlock

	// TokenCloseRawBlock is the CLOSE_RAW_BLOCK token
	TokenCloseRawBlock

	// TokenOpenEndRawBlock is the END_RAW_BLOCK token
	TokenOpenEndRawBlock

	// TokenOpenUnescaped is the OPEN_UNESCAPED token
	TokenOpenUnescaped

	// TokenCloseUnescaped is the CLOSE_UNESCAPED token
	TokenCloseUnescaped

	// TokenOpenBlock is the OPEN_BLOCK token
	TokenOpenBlock

	// TokenOpenEndBlock is the OPEN_ENDBLOCK token
	TokenOpenEndBlock

	// TokenInverse is the INVERSE token
	TokenInverse

	// TokenOpenInverse is the OPEN_INVERSE token
	TokenOpenInverse

	// TokenOpenInverseChain is the OPEN_INVERSE_CHAIN token
	TokenOpenInverseChain

	// TokenOpenPartial is the OPEN_PARTIAL token
	TokenOpenPartial

	// TokenComment is the COMMENT token
	TokenComment

	//
	// Inside mustaches
	//

	// TokenOpenSexpr is the OPEN_SEXPR token
	TokenOpenSexpr

	// TokenCloseSexpr is the CLOSE_SEXPR token
	TokenCloseSexpr

	// TokenEquals is the EQUALS token
	TokenEquals

	// TokenData is the DATA token
	TokenData

	// TokenSep is the SEP token
	TokenSep

	// TokenOpenBlockParams is the OPEN_BLOCK_PARAMS token
	TokenOpenBlockParams

	// TokenCloseBlockParams is the CLOSE_BLOCK_PARAMS token
	TokenCloseBlockParams

	//
	// Tokens with content
	//

	// TokenContent is the CONTENT token
	TokenContent

	// TokenID is the ID token
	TokenID

	// TokenString is the STRING token
	TokenString

	// TokenNumber is the NUMBER token
	TokenNumber

	// TokenBoolean is the BOOLEAN token
	TokenBoolean
)

const (
	// Option to generate token position in its string representation
	dumpTokenPos = false

	// Option to generate values for all token kinds for their string representations
	dumpAllTokensVal = true
)

// TokenKind represents a Token type.
type TokenKind int

// Token represents a scanned token.
type Token struct {
	Kind TokenKind // Token kind
	Val  string    // Token value

	Pos  int // Byte position in input string
	Line int // Line number in input string
}

// tokenName permits to display token name given token type
var tokenName = map[TokenKind]string{
	TokenError:            "Error",
	TokenEOF:              "EOF",
	TokenContent:          "Content",
	TokenComment:          "Comment",
	TokenOpen:             "Open",
	TokenClose:            "Close",
	TokenOpenUnescaped:    "OpenUnescaped",
	TokenCloseUnescaped:   "CloseUnescaped",
	TokenOpenBlock:        "OpenBlock",
	TokenOpenEndBlock:     "OpenEndBlock",
	TokenOpenRawBlock:     "OpenRawBlock",
	TokenCloseRawBlock:    "CloseRawBlock",
	TokenOpenEndRawBlock:  "OpenEndRawBlock",
	TokenOpenBlockParams:  "OpenBlockParams",
	TokenCloseBlockParams: "CloseBlockParams",
	TokenInverse:          "Inverse",
	TokenOpenInverse:      "OpenInverse",
	TokenOpenInverseChain: "OpenInverseChain",
	TokenOpenPartial:      "OpenPartial",
	TokenOpenSexpr:        "OpenSexpr",
	TokenCloseSexpr:       "CloseSexpr",
	TokenID:               "ID",
	TokenEquals:           "Equals",
	TokenString:           "String",
	TokenNumber:           "Number",
	TokenBoolean:          "Boolean",
	TokenData:             "Data",
	TokenSep:              "Sep",
}

// String returns the token kind string representation for debugging.
func (k TokenKind) String() string {
	s := tokenName[k]
	if s == "" {
		return fmt.Sprintf("Token-%d", int(k))
	}
	return s
}

// String returns the token string representation for debugging.
func (t Token) String() string {
	result := ""

	if dumpTokenPos {
		result += fmt.Sprintf("%d:", t.Pos)
	}

	result += fmt.Sprintf("%s", t.Kind)

	if (dumpAllTokensVal || (t.Kind >= TokenContent)) && len(t.Val) > 0 {
		if len(t.Val) > 100 {
			result += fmt.Sprintf("{%.20q...}", t.Val)
		} else {
			result += fmt.Sprintf("{%q}", t.Val)
		}
	}

	return result
}
