// Copyright 2017 Gerasimos Maropoulos. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/iris/core/router/macro/interpreter/ast"
	"github.com/kataras/iris/core/router/macro/interpreter/lexer"
	"github.com/kataras/iris/core/router/macro/interpreter/token"
)

func Parse(fullpath string) ([]*ast.ParamStatement, error) {
	pathParts := strings.SplitN(fullpath, "/", -1)
	p := new(ParamParser)
	statements := make([]*ast.ParamStatement, 0)
	for i, s := range pathParts {
		if s == "" { // if starts with /
			continue
		}

		// if it's not a named path parameter of the new syntax then continue to the next
		if s[0] != lexer.Begin || s[len(s)-1] != lexer.End {
			continue
		}

		p.Reset(s)
		stmt, err := p.Parse()
		if err != nil {
			// exit on first error
			return nil, err
		}
		// if we have param type path but it's not the last path part
		if stmt.Type == ast.ParamTypePath && i < len(pathParts)-1 {
			return nil, fmt.Errorf("param type 'path' should be lived only inside the last path segment, but was inside: %s", s)
		}

		statements = append(statements, stmt)
	}

	return statements, nil
}

type ParamParser struct {
	src    string
	errors []string
}

func NewParamParser(src string) *ParamParser {
	p := new(ParamParser)
	p.Reset(src)
	return p
}

func (p *ParamParser) Reset(src string) {
	p.src = src

	p.errors = []string{}
}

func (p *ParamParser) appendErr(format string, a ...interface{}) {
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

func (p ParamParser) Error() error {
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}

func (p *ParamParser) Parse() (*ast.ParamStatement, error) {
	l := lexer.New(p.src)

	stmt := &ast.ParamStatement{
		ErrorCode: DefaultParamErrorCode,
		Type:      DefaultParamType,
		Src:       p.src,
	}

	lastParamFunc := ast.ParamFunc{}

	for {
		t := l.NextToken()
		if t.Type == token.EOF {
			if stmt.Name == "" {
				p.appendErr("[1:] parameter name is missing")
			}
			break
		}

		switch t.Type {
		case token.LBRACE:
			// name, alphabetical and _, param names are not allowed to contain any number.
			nextTok := l.NextToken()
			stmt.Name = nextTok.Literal
		case token.COLON:
			// type
			nextTok := l.NextToken()
			paramType := ast.LookupParamType(nextTok.Literal)
			if paramType == ast.ParamTypeUnExpected {
				p.appendErr("[%d:%d] unexpected parameter type: %s", t.Start, t.End, nextTok.Literal)
			}
			stmt.Type = paramType
			// param func
		case token.IDENT:
			lastParamFunc.Name = t.Literal
		case token.LPAREN:
			// param function without arguments ()
			if l.PeekNextTokenType() == token.RPAREN {
				// do nothing, just continue to the RPAREN
				continue
			}

			argValTok := l.NextDynamicToken() // catch anything from "(" and forward, until ")", because we need to
			// be able to use regex expression as a macro type's func argument too.
			argVal, err := parseParamFuncArg(argValTok)
			if err != nil {
				p.appendErr("[%d:%d] expected param func argument to be a string or number but got %s", t.Start, t.End, argValTok.Literal)
				continue
			}

			// fmt.Printf("argValTok: %#v\n", argValTok)
			// fmt.Printf("argVal: %#v\n", argVal)
			lastParamFunc.Args = append(lastParamFunc.Args, argVal)

		case token.COMMA:
			argValTok := l.NextToken()
			argVal, err := parseParamFuncArg(argValTok)
			if err != nil {
				p.appendErr("[%d:%d] expected param func argument to be a string or number type but got %s", t.Start, t.End, argValTok.Literal)
				continue
			}

			lastParamFunc.Args = append(lastParamFunc.Args, argVal)
		case token.RPAREN:
			stmt.Funcs = append(stmt.Funcs, lastParamFunc)
			lastParamFunc = ast.ParamFunc{} // reset
		case token.ELSE:
			errCodeTok := l.NextToken()
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
			// check if } but not {
			if stmt.Name == "" {
				p.appendErr("[%d:%d] illegal token: }, forgot '{' ?", t.Start, t.End)
			}
			break
		case token.ILLEGAL:
			p.appendErr("[%d:%d] illegal token: %s", t.Start, t.End, t.Literal)
		default:
			p.appendErr("[%d:%d] unexpected token type: %q with value %s", t.Start, t.End, t.Type, t.Literal)
		}
	}

	return stmt, p.Error()
}
