package lexer

import (
	"testing"

	"github.com/kataras/iris/macro/interpreter/token"
)

func TestNextToken(t *testing.T) {
	input := `{id:int min(1) max(5) else 404}`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.LBRACE, "{"},  // 0
		{token.IDENT, "id"},  // 1
		{token.COLON, ":"},   // 2
		{token.IDENT, "int"}, // 3
		{token.IDENT, "min"}, // 4
		{token.LPAREN, "("},  // 5
		{token.INT, "1"},     // 6
		{token.RPAREN, ")"},  // 7
		{token.IDENT, "max"}, // 8
		{token.LPAREN, "("},  // 9
		{token.INT, "5"},     // 10
		{token.RPAREN, ")"},  // 11
		{token.ELSE, "else"}, // 12
		{token.INT, "404"},   // 13
		{token.RBRACE, "}"},  // 14
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

	}
}

// EMEINA STO:
// 30/232 selida apto making a interpeter in Go.
// den ekana to skipWhitespaces giati skeftomai
// an borei na to xreiastw 9a dw aurio.
