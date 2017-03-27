package parser

import (
	"fmt"
	"strings"

	"gopkg.in/kataras/iris.v6/_future/ipel/ast"
	"gopkg.in/kataras/iris.v6/_future/ipel/lexer"
	"gopkg.in/kataras/iris.v6/_future/ipel/token"
)

type Parser struct {
	l      *lexer.Lexer
	errors []string
}

func New(lexer *lexer.Lexer) *Parser {
	p := &Parser{
		l: lexer,
	}

	return p
}

func (p *Parser) appendErr(format string, a ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, a...))
}

func (p *Parser) Parse() (*ast.ParamStatement, error) {
	stmt := new(ast.ParamStatement)
	for {
		t := p.l.NextToken()
		if t.Type == token.EOF {
			break
		}

		switch t.Type {
		case token.LBRACE:
			// name
			nextTok := p.l.NextToken()
			stmt.Name = nextTok.Literal
		case token.COLON:
			// type
			nextTok := p.l.NextToken()
			paramType := ast.LookupParamType(nextTok.Literal)
			if paramType == ast.ParamTypeUnExpected {
				p.appendErr("[%d:%d] unexpected parameter type: %s", t.Start, t.End, nextTok.Literal)
			}
		case token.ILLEGAL:
			p.appendErr("[%d:%d] illegal token: %s", t.Start, t.End, t.Literal)
		default:
			p.appendErr("[%d:%d] unexpected token type: %q", t.Start, t.End, t.Type)

		}
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return stmt, nil
}
