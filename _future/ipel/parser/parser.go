package parser

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/kataras/iris.v6/_future/ipel/ast"
	"gopkg.in/kataras/iris.v6/_future/ipel/lexer"
	"gopkg.in/kataras/iris.v6/_future/ipel/token"
)

type Parser struct {
	l      *lexer.Lexer
	errors []string
}

func New(src string) *Parser {
	p := new(Parser)
	p.Reset(src)
	return p
}

func (p *Parser) Reset(src string) {
	p.l = lexer.New(src)
	p.errors = []string{}
}

func (p *Parser) appendErr(format string, a ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, a...))
}

const DefaultParamErrorCode = 404
const DefaultParamType = ast.ParamTypeString

func parseParamFuncArg(t token.Token) (a ast.ParamFuncArg, err error) {
	if t.Type == token.INT {
		return ast.ParamFuncArgToInt(t.Literal)
	}
	return t.Literal, nil
}

func (p Parser) Error() error {
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}

func (p *Parser) Parse() (*ast.ParamStatement, error) {
	p.errors = []string{}

	stmt := &ast.ParamStatement{
		ErrorCode: DefaultParamErrorCode,
		Type:      DefaultParamType,
	}

	lastParamFunc := ast.ParamFunc{}

	for {
		t := p.l.NextToken()
		if t.Type == token.EOF {
			if stmt.Name == "" {
				p.appendErr("[1:] parameter name is missing")
			}
			break
		}

		switch t.Type {
		case token.LBRACE:
			// name, alphabetical and _, param names are not allowed to contain any number.
			nextTok := p.l.NextToken()
			stmt.Name = nextTok.Literal
		case token.COLON:
			// type
			nextTok := p.l.NextToken()
			paramType := ast.LookupParamType(nextTok.Literal)
			if paramType == ast.ParamTypeUnExpected {
				p.appendErr("[%d:%d] unexpected parameter type: %s", t.Start, t.End, nextTok.Literal)
			}
			stmt.Type = paramType
			// param func
		case token.IDENT:
			lastParamFunc.Name = t.Literal
		case token.LPAREN:
			argValTok := p.l.NextToken()
			argVal, err := parseParamFuncArg(argValTok)
			if err != nil {
				p.appendErr("[%d:%d] expected param func argument to be an integer but got %s", t.Start, t.End, argValTok.Literal)
				continue
			}

			lastParamFunc.Args = append(lastParamFunc.Args, argVal)
		case token.COMMA:
			argValTok := p.l.NextToken()
			argVal, err := parseParamFuncArg(argValTok)
			if err != nil {
				p.appendErr("[%d:%d] expected param func argument to be an integer but got %s", t.Start, t.End, argValTok.Literal)
				continue
			}

			lastParamFunc.Args = append(lastParamFunc.Args, argVal)
		case token.RPAREN:
			stmt.Funcs = append(stmt.Funcs, lastParamFunc)
			lastParamFunc = ast.ParamFunc{} // reset
		case token.ELSE:
			errCodeTok := p.l.NextToken()
			if errCodeTok.Type != token.INT {
				p.appendErr("[%d:%d] expected error code to be an integer but got %s", t.Start, t.End, errCodeTok.Literal)
				continue
			}
			errCode, err := strconv.Atoi(errCodeTok.Literal)
			if err != nil {
				// this is a bug on lexer if throws because we already check for token.INT
				p.appendErr("[%d:%d] unexpected lexer error while trying to convert error code to an integer, %s", t.Start, t.End, err.Error())
				continue
			}
			stmt.ErrorCode = errCode
		case token.RBRACE:
			break
		case token.ILLEGAL:
			p.appendErr("[%d:%d] illegal token: %s", t.Start, t.End, t.Literal)
		default:
			p.appendErr("[%d:%d] unexpected token type: %q with value %s", t.Start, t.End, t.Type, t.Literal)
		}
	}

	return stmt, p.Error()
}
