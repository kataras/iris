package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/iris/macro/interpreter/ast"
	"github.com/kataras/iris/macro/interpreter/lexer"
	"github.com/kataras/iris/macro/interpreter/token"
)

// Parse takes a route "fullpath"
// and returns its param statements
// or an error if failed.
func Parse(fullpath string, paramTypes []ast.ParamType) ([]*ast.ParamStatement, error) {
	if len(paramTypes) == 0 {
		return nil, fmt.Errorf("empty parameter types")
	}

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
		stmt, err := p.Parse(paramTypes)
		if err != nil {
			// exit on first error
			return nil, err
		}
		// if we have param type path but it's not the last path part
		if ast.IsTrailing(stmt.Type) && i < len(pathParts)-1 {
			return nil, fmt.Errorf("%s: parameter type \"%s\" should be registered to the very last of a path", s, stmt.Type.Indent())
		}

		statements = append(statements, stmt)
	}

	return statements, nil
}

// ParamParser is the parser
// which is being used by the Parse function
// to parse path segments one by one
// and return their parsed parameter statements (param name, param type its functions and the inline route's functions).
type ParamParser struct {
	src    string
	errors []string
}

// NewParamParser receives a "src" of a single parameter
// and returns a new ParamParser, ready to Parse.
func NewParamParser(src string) *ParamParser {
	p := new(ParamParser)
	p.Reset(src)
	return p
}

// Reset resets this ParamParser,
// reset the errors and set the source to the input "src".
func (p *ParamParser) Reset(src string) {
	p.src = src
	p.errors = []string{}
}

func (p *ParamParser) appendErr(format string, a ...interface{}) {
	p.errors = append(p.errors, fmt.Sprintf(format, a...))
}

const (
	// DefaultParamErrorCode is the default http error code, 404 not found,
	// per-parameter. An error code can be setted via
	// the "else" keyword inside a route's path.
	DefaultParamErrorCode = 404
)

// func parseParamFuncArg(t token.Token) (a ast.ParamFuncArg, err error) {
// 	if t.Type == token.INT {
// 		return ast.ParamFuncArgToInt(t.Literal)
// 	}
// 	// act all as strings here, because of int vs int64 vs uint64 and etc.
// 	return t.Literal, nil
// }

func parseParamFuncArg(t token.Token) (a string, err error) {
	// act all as strings here, because of int vs int64 vs uint64 and etc.
	return t.Literal, nil
}

func (p ParamParser) Error() error {
	if len(p.errors) > 0 {
		return fmt.Errorf(strings.Join(p.errors, "\n"))
	}
	return nil
}

// Parse parses the p.src based on the given param types and returns its param statement
// and an error on failure.
func (p *ParamParser) Parse(paramTypes []ast.ParamType) (*ast.ParamStatement, error) {
	l := lexer.New(p.src)

	stmt := &ast.ParamStatement{
		ErrorCode: DefaultParamErrorCode,
		Type:      ast.GetMasterParamType(paramTypes...),
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
			// can accept only letter or number only.
			nextTok := l.NextToken()
			stmt.Name = nextTok.Literal
		case token.COLON:
			// type can accept both letters and numbers but not symbols ofc.
			nextTok := l.NextToken()
			paramType, found := ast.LookupParamType(nextTok.Literal, paramTypes...)

			if !found {
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

			// fmt.Printf("argValTok: %#v\n", argValTok)
			// fmt.Printf("argVal: %#v\n", argVal)
			lastParamFunc.Args = append(lastParamFunc.Args, argValTok.Literal)

		case token.COMMA:
			argValTok := l.NextToken()
			lastParamFunc.Args = append(lastParamFunc.Args, argValTok.Literal)
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
