// Package parser provides a handlebars syntax analyser. It consumes the tokens provided by the lexer to build an AST.
package parser

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"

	"github.com/aymerick/raymond/ast"
	"github.com/aymerick/raymond/lexer"
)

// References:
//   - https://github.com/wycats/handlebars.js/blob/master/src/handlebars.yy
//   - https://github.com/golang/go/blob/master/src/text/template/parse/parse.go

// parser is a syntax analyzer.
type parser struct {
	// Lexer
	lex *lexer.Lexer

	// Root node
	root ast.Node

	// Tokens parsed but not consumed yet
	tokens []*lexer.Token

	// All tokens have been retreieved from lexer
	lexOver bool
}

var (
	rOpenComment  = regexp.MustCompile(`^\{\{~?!-?-?`)
	rCloseComment = regexp.MustCompile(`-?-?~?\}\}$`)
	rOpenAmp      = regexp.MustCompile(`^\{\{~?&`)
)

// new instanciates a new parser
func new(input string) *parser {
	return &parser{
		lex: lexer.Scan(input),
	}
}

// Parse analyzes given input and returns the AST root node.
func Parse(input string) (result *ast.Program, err error) {
	// recover error
	defer errRecover(&err)

	parser := new(input)

	// parse
	result = parser.parseProgram()

	// check last token
	token := parser.shift()
	if token.Kind != lexer.TokenEOF {
		// Parsing ended before EOF
		errToken(token, "Syntax error")
	}

	// fix whitespaces
	processWhitespaces(result)

	// named returned values
	return
}

// errRecover recovers parsing panic
func errRecover(errp *error) {
	e := recover()
	if e != nil {
		switch err := e.(type) {
		case runtime.Error:
			panic(e)
		case error:
			*errp = err
		default:
			panic(e)
		}
	}
}

// errPanic panics
func errPanic(err error, line int) {
	panic(fmt.Errorf("Parse error on line %d:\n%s", line, err))
}

// errNode panics with given node infos
func errNode(node ast.Node, msg string) {
	errPanic(fmt.Errorf("%s\nNode: %s", msg, node), node.Location().Line)
}

// errNode panics with given Token infos
func errToken(tok *lexer.Token, msg string) {
	errPanic(fmt.Errorf("%s\nToken: %s", msg, tok), tok.Line)
}

// errNode panics because of an unexpected Token kind
func errExpected(expect lexer.TokenKind, tok *lexer.Token) {
	errPanic(fmt.Errorf("Expecting %s, got: '%s'", expect, tok), tok.Line)
}

// program : statement*
func (p *parser) parseProgram() *ast.Program {
	result := ast.NewProgram(p.next().Pos, p.next().Line)

	for p.isStatement() {
		result.AddStatement(p.parseStatement())
	}

	return result
}

// statement : mustache | block | rawBlock | partial | content | COMMENT
func (p *parser) parseStatement() ast.Node {
	var result ast.Node

	tok := p.next()

	switch tok.Kind {
	case lexer.TokenOpen, lexer.TokenOpenUnescaped:
		// mustache
		result = p.parseMustache()
	case lexer.TokenOpenBlock:
		// block
		result = p.parseBlock()
	case lexer.TokenOpenInverse:
		// block
		result = p.parseInverse()
	case lexer.TokenOpenRawBlock:
		// rawBlock
		result = p.parseRawBlock()
	case lexer.TokenOpenPartial:
		// partial
		result = p.parsePartial()
	case lexer.TokenContent:
		// content
		result = p.parseContent()
	case lexer.TokenComment:
		// COMMENT
		result = p.parseComment()
	}

	return result
}

// isStatement returns true if next token starts a statement
func (p *parser) isStatement() bool {
	if !p.have(1) {
		return false
	}

	switch p.next().Kind {
	case lexer.TokenOpen, lexer.TokenOpenUnescaped, lexer.TokenOpenBlock,
		lexer.TokenOpenInverse, lexer.TokenOpenRawBlock, lexer.TokenOpenPartial,
		lexer.TokenContent, lexer.TokenComment:
		return true
	}

	return false
}

// content : CONTENT
func (p *parser) parseContent() *ast.ContentStatement {
	// CONTENT
	tok := p.shift()
	if tok.Kind != lexer.TokenContent {
		// @todo This check can be removed if content is optional in a raw block
		errExpected(lexer.TokenContent, tok)
	}

	return ast.NewContentStatement(tok.Pos, tok.Line, tok.Val)
}

// COMMENT
func (p *parser) parseComment() *ast.CommentStatement {
	// COMMENT
	tok := p.shift()

	value := rOpenComment.ReplaceAllString(tok.Val, "")
	value = rCloseComment.ReplaceAllString(value, "")

	result := ast.NewCommentStatement(tok.Pos, tok.Line, value)
	result.Strip = ast.NewStripForStr(tok.Val)

	return result
}

// param* hash?
func (p *parser) parseExpressionParamsHash() ([]ast.Node, *ast.Hash) {
	var params []ast.Node
	var hash *ast.Hash

	// params*
	if p.isParam() {
		params = p.parseParams()
	}

	// hash?
	if p.isHashSegment() {
		hash = p.parseHash()
	}

	return params, hash
}

// helperName param* hash?
func (p *parser) parseExpression(tok *lexer.Token) *ast.Expression {
	result := ast.NewExpression(tok.Pos, tok.Line)

	// helperName
	result.Path = p.parseHelperName()

	// param* hash?
	result.Params, result.Hash = p.parseExpressionParamsHash()

	return result
}

// rawBlock : openRawBlock content endRawBlock
// openRawBlock : OPEN_RAW_BLOCK helperName param* hash? CLOSE_RAW_BLOCK
// endRawBlock : OPEN_END_RAW_BLOCK helperName CLOSE_RAW_BLOCK
func (p *parser) parseRawBlock() *ast.BlockStatement {
	// OPEN_RAW_BLOCK
	tok := p.shift()

	result := ast.NewBlockStatement(tok.Pos, tok.Line)

	// helperName param* hash?
	result.Expression = p.parseExpression(tok)

	openName := result.Expression.Canonical()

	// CLOSE_RAW_BLOCK
	tok = p.shift()
	if tok.Kind != lexer.TokenCloseRawBlock {
		errExpected(lexer.TokenCloseRawBlock, tok)
	}

	// content
	// @todo Is content mandatory in a raw block ?
	content := p.parseContent()

	program := ast.NewProgram(tok.Pos, tok.Line)
	program.AddStatement(content)

	result.Program = program

	// OPEN_END_RAW_BLOCK
	tok = p.shift()
	if tok.Kind != lexer.TokenOpenEndRawBlock {
		// should never happen as it is caught by lexer
		errExpected(lexer.TokenOpenEndRawBlock, tok)
	}

	// helperName
	endID := p.parseHelperName()

	closeName, ok := ast.HelperNameStr(endID)
	if !ok {
		errNode(endID, "Erroneous closing expression")
	}

	if openName != closeName {
		errNode(endID, fmt.Sprintf("%s doesn't match %s", openName, closeName))
	}

	// CLOSE_RAW_BLOCK
	tok = p.shift()
	if tok.Kind != lexer.TokenCloseRawBlock {
		errExpected(lexer.TokenCloseRawBlock, tok)
	}

	return result
}

// block : openBlock program inverseChain? closeBlock
func (p *parser) parseBlock() *ast.BlockStatement {
	// openBlock
	result, blockParams := p.parseOpenBlock()

	// program
	program := p.parseProgram()
	program.BlockParams = blockParams
	result.Program = program

	// inverseChain?
	if p.isInverseChain() {
		result.Inverse = p.parseInverseChain()
	}

	// closeBlock
	p.parseCloseBlock(result)

	setBlockInverseStrip(result)

	return result
}

// setBlockInverseStrip is called when parsing `block` (openBlock | openInverse) and `inverseChain`
//
// TODO: This was totally cargo culted ! CHECK THAT !
//
// cf. prepareBlock() in:
//   https://github.com/wycats/handlebars.js/blob/master/lib/handlebars/compiler/helper.js
func setBlockInverseStrip(block *ast.BlockStatement) {
	if block.Inverse == nil {
		return
	}

	if block.Inverse.Chained {
		b, _ := block.Inverse.Body[0].(*ast.BlockStatement)
		b.CloseStrip = block.CloseStrip
	}

	block.InverseStrip = block.Inverse.Strip
}

// block : openInverse program inverseAndProgram? closeBlock
func (p *parser) parseInverse() *ast.BlockStatement {
	// openInverse
	result, blockParams := p.parseOpenBlock()

	// program
	program := p.parseProgram()

	program.BlockParams = blockParams
	result.Inverse = program

	// inverseAndProgram?
	if p.isInverse() {
		result.Program = p.parseInverseAndProgram()
	}

	// closeBlock
	p.parseCloseBlock(result)

	setBlockInverseStrip(result)

	return result
}

// helperName param* hash? blockParams?
func (p *parser) parseOpenBlockExpression(tok *lexer.Token) (*ast.BlockStatement, []string) {
	var blockParams []string

	result := ast.NewBlockStatement(tok.Pos, tok.Line)

	// helperName param* hash?
	result.Expression = p.parseExpression(tok)

	// blockParams?
	if p.isBlockParams() {
		blockParams = p.parseBlockParams()
	}

	// named returned values
	return result, blockParams
}

// inverseChain : openInverseChain program inverseChain?
//              | inverseAndProgram
func (p *parser) parseInverseChain() *ast.Program {
	if p.isInverse() {
		// inverseAndProgram
		return p.parseInverseAndProgram()
	}

	result := ast.NewProgram(p.next().Pos, p.next().Line)

	// openInverseChain
	block, blockParams := p.parseOpenBlock()

	// program
	program := p.parseProgram()

	program.BlockParams = blockParams
	block.Program = program

	// inverseChain?
	if p.isInverseChain() {
		block.Inverse = p.parseInverseChain()
	}

	setBlockInverseStrip(block)

	result.Chained = true
	result.AddStatement(block)

	return result
}

// Returns true if current token starts an inverse chain
func (p *parser) isInverseChain() bool {
	return p.isOpenInverseChain() || p.isInverse()
}

// inverseAndProgram : INVERSE program
func (p *parser) parseInverseAndProgram() *ast.Program {
	// INVERSE
	tok := p.shift()

	// program
	result := p.parseProgram()
	result.Strip = ast.NewStripForStr(tok.Val)

	return result
}

// openBlock : OPEN_BLOCK helperName param* hash? blockParams? CLOSE
// openInverse : OPEN_INVERSE helperName param* hash? blockParams? CLOSE
// openInverseChain: OPEN_INVERSE_CHAIN helperName param* hash? blockParams? CLOSE
func (p *parser) parseOpenBlock() (*ast.BlockStatement, []string) {
	// OPEN_BLOCK | OPEN_INVERSE | OPEN_INVERSE_CHAIN
	tok := p.shift()

	// helperName param* hash? blockParams?
	result, blockParams := p.parseOpenBlockExpression(tok)

	// CLOSE
	tokClose := p.shift()
	if tokClose.Kind != lexer.TokenClose {
		errExpected(lexer.TokenClose, tokClose)
	}

	result.OpenStrip = ast.NewStrip(tok.Val, tokClose.Val)

	// named returned values
	return result, blockParams
}

// closeBlock : OPEN_ENDBLOCK helperName CLOSE
func (p *parser) parseCloseBlock(block *ast.BlockStatement) {
	// OPEN_ENDBLOCK
	tok := p.shift()
	if tok.Kind != lexer.TokenOpenEndBlock {
		errExpected(lexer.TokenOpenEndBlock, tok)
	}

	// helperName
	endID := p.parseHelperName()

	closeName, ok := ast.HelperNameStr(endID)
	if !ok {
		errNode(endID, "Erroneous closing expression")
	}

	openName := block.Expression.Canonical()
	if openName != closeName {
		errNode(endID, fmt.Sprintf("%s doesn't match %s", openName, closeName))
	}

	// CLOSE
	tokClose := p.shift()
	if tokClose.Kind != lexer.TokenClose {
		errExpected(lexer.TokenClose, tokClose)
	}

	block.CloseStrip = ast.NewStrip(tok.Val, tokClose.Val)
}

// mustache : OPEN helperName param* hash? CLOSE
//          | OPEN_UNESCAPED helperName param* hash? CLOSE_UNESCAPED
func (p *parser) parseMustache() *ast.MustacheStatement {
	// OPEN | OPEN_UNESCAPED
	tok := p.shift()

	closeToken := lexer.TokenClose
	if tok.Kind == lexer.TokenOpenUnescaped {
		closeToken = lexer.TokenCloseUnescaped
	}

	unescaped := false
	if (tok.Kind == lexer.TokenOpenUnescaped) || (rOpenAmp.MatchString(tok.Val)) {
		unescaped = true
	}

	result := ast.NewMustacheStatement(tok.Pos, tok.Line, unescaped)

	// helperName param* hash?
	result.Expression = p.parseExpression(tok)

	// CLOSE | CLOSE_UNESCAPED
	tokClose := p.shift()
	if tokClose.Kind != closeToken {
		errExpected(closeToken, tokClose)
	}

	result.Strip = ast.NewStrip(tok.Val, tokClose.Val)

	return result
}

// partial : OPEN_PARTIAL partialName param* hash? CLOSE
func (p *parser) parsePartial() *ast.PartialStatement {
	// OPEN_PARTIAL
	tok := p.shift()

	result := ast.NewPartialStatement(tok.Pos, tok.Line)

	// partialName
	result.Name = p.parsePartialName()

	// param* hash?
	result.Params, result.Hash = p.parseExpressionParamsHash()

	// CLOSE
	tokClose := p.shift()
	if tokClose.Kind != lexer.TokenClose {
		errExpected(lexer.TokenClose, tokClose)
	}

	result.Strip = ast.NewStrip(tok.Val, tokClose.Val)

	return result
}

// helperName | sexpr
func (p *parser) parseHelperNameOrSexpr() ast.Node {
	if p.isSexpr() {
		// sexpr
		return p.parseSexpr()
	}

	// helperName
	return p.parseHelperName()
}

// param : helperName | sexpr
func (p *parser) parseParam() ast.Node {
	return p.parseHelperNameOrSexpr()
}

// Returns true if next tokens represent a `param`
func (p *parser) isParam() bool {
	return (p.isSexpr() || p.isHelperName()) && !p.isHashSegment()
}

// param*
func (p *parser) parseParams() []ast.Node {
	var result []ast.Node

	for p.isParam() {
		result = append(result, p.parseParam())
	}

	return result
}

// sexpr : OPEN_SEXPR helperName param* hash? CLOSE_SEXPR
func (p *parser) parseSexpr() *ast.SubExpression {
	// OPEN_SEXPR
	tok := p.shift()

	result := ast.NewSubExpression(tok.Pos, tok.Line)

	// helperName param* hash?
	result.Expression = p.parseExpression(tok)

	// CLOSE_SEXPR
	tok = p.shift()
	if tok.Kind != lexer.TokenCloseSexpr {
		errExpected(lexer.TokenCloseSexpr, tok)
	}

	return result
}

// hash : hashSegment+
func (p *parser) parseHash() *ast.Hash {
	var pairs []*ast.HashPair

	for p.isHashSegment() {
		pairs = append(pairs, p.parseHashSegment())
	}

	firstLoc := pairs[0].Location()

	result := ast.NewHash(firstLoc.Pos, firstLoc.Line)
	result.Pairs = pairs

	return result
}

// returns true if next tokens represents a `hashSegment`
func (p *parser) isHashSegment() bool {
	return p.have(2) && (p.next().Kind == lexer.TokenID) && (p.nextAt(1).Kind == lexer.TokenEquals)
}

// hashSegment : ID EQUALS param
func (p *parser) parseHashSegment() *ast.HashPair {
	// ID
	tok := p.shift()

	// EQUALS
	p.shift()

	// param
	param := p.parseParam()

	result := ast.NewHashPair(tok.Pos, tok.Line)
	result.Key = tok.Val
	result.Val = param

	return result
}

// blockParams : OPEN_BLOCK_PARAMS ID+ CLOSE_BLOCK_PARAMS
func (p *parser) parseBlockParams() []string {
	var result []string

	// OPEN_BLOCK_PARAMS
	tok := p.shift()

	// ID+
	for p.isID() {
		result = append(result, p.shift().Val)
	}

	if len(result) == 0 {
		errExpected(lexer.TokenID, p.next())
	}

	// CLOSE_BLOCK_PARAMS
	tok = p.shift()
	if tok.Kind != lexer.TokenCloseBlockParams {
		errExpected(lexer.TokenCloseBlockParams, tok)
	}

	return result
}

// helperName : path | dataName | STRING | NUMBER | BOOLEAN | UNDEFINED | NULL
func (p *parser) parseHelperName() ast.Node {
	var result ast.Node

	tok := p.next()

	switch tok.Kind {
	case lexer.TokenBoolean:
		// BOOLEAN
		p.shift()
		result = ast.NewBooleanLiteral(tok.Pos, tok.Line, (tok.Val == "true"), tok.Val)
	case lexer.TokenNumber:
		// NUMBER
		p.shift()

		val, isInt := parseNumber(tok)
		result = ast.NewNumberLiteral(tok.Pos, tok.Line, val, isInt, tok.Val)
	case lexer.TokenString:
		// STRING
		p.shift()
		result = ast.NewStringLiteral(tok.Pos, tok.Line, tok.Val)
	case lexer.TokenData:
		// dataName
		result = p.parseDataName()
	default:
		// path
		result = p.parsePath(false)
	}

	return result
}

// parseNumber parses a number
func parseNumber(tok *lexer.Token) (result float64, isInt bool) {
	var valInt int
	var err error

	valInt, err = strconv.Atoi(tok.Val)
	if err == nil {
		isInt = true

		result = float64(valInt)
	} else {
		isInt = false

		result, err = strconv.ParseFloat(tok.Val, 64)
		if err != nil {
			errToken(tok, fmt.Sprintf("Failed to parse number: %s", tok.Val))
		}
	}

	// named returned values
	return
}

// Returns true if next tokens represent a `helperName`
func (p *parser) isHelperName() bool {
	switch p.next().Kind {
	case lexer.TokenBoolean, lexer.TokenNumber, lexer.TokenString, lexer.TokenData, lexer.TokenID:
		return true
	}

	return false
}

// partialName : helperName | sexpr
func (p *parser) parsePartialName() ast.Node {
	return p.parseHelperNameOrSexpr()
}

// dataName : DATA pathSegments
func (p *parser) parseDataName() *ast.PathExpression {
	// DATA
	p.shift()

	// pathSegments
	return p.parsePath(true)
}

// path : pathSegments
// pathSegments : pathSegments SEP ID
//              | ID
func (p *parser) parsePath(data bool) *ast.PathExpression {
	var tok *lexer.Token

	// ID
	tok = p.shift()
	if tok.Kind != lexer.TokenID {
		errExpected(lexer.TokenID, tok)
	}

	result := ast.NewPathExpression(tok.Pos, tok.Line, data)
	result.Part(tok.Val)

	for p.isPathSep() {
		// SEP
		tok = p.shift()
		result.Sep(tok.Val)

		// ID
		tok = p.shift()
		if tok.Kind != lexer.TokenID {
			errExpected(lexer.TokenID, tok)
		}

		result.Part(tok.Val)

		if len(result.Parts) > 0 {
			switch tok.Val {
			case "..", ".", "this":
				errToken(tok, "Invalid path: "+result.Original)
			}
		}
	}

	return result
}

// Ensures there is token to parse at given index
func (p *parser) ensure(index int) {
	if p.lexOver {
		// nothing more to grab
		return
	}

	nb := index + 1

	for len(p.tokens) < nb {
		// fetch next token
		tok := p.lex.NextToken()

		// queue it
		p.tokens = append(p.tokens, &tok)

		if (tok.Kind == lexer.TokenEOF) || (tok.Kind == lexer.TokenError) {
			p.lexOver = true
			break
		}
	}
}

// have returns true is there are a list given number of tokens to consume left
func (p *parser) have(nb int) bool {
	p.ensure(nb - 1)

	return len(p.tokens) >= nb
}

// nextAt returns next token at given index, without consuming it
func (p *parser) nextAt(index int) *lexer.Token {
	p.ensure(index)

	return p.tokens[index]
}

// next returns next token without consuming it
func (p *parser) next() *lexer.Token {
	return p.nextAt(0)
}

// shift returns next token and remove it from the tokens buffer
//
// Panics if next token is `TokenError`
func (p *parser) shift() *lexer.Token {
	var result *lexer.Token

	p.ensure(0)

	result, p.tokens = p.tokens[0], p.tokens[1:]

	// check error token
	if result.Kind == lexer.TokenError {
		errToken(result, "Lexer error")
	}

	return result
}

// isToken returns true if next token is of given type
func (p *parser) isToken(kind lexer.TokenKind) bool {
	return p.have(1) && p.next().Kind == kind
}

// isSexpr returns true if next token starts a sexpr
func (p *parser) isSexpr() bool {
	return p.isToken(lexer.TokenOpenSexpr)
}

// isPathSep returns true if next token is a path separator
func (p *parser) isPathSep() bool {
	return p.isToken(lexer.TokenSep)
}

// isID returns true if next token is an ID
func (p *parser) isID() bool {
	return p.isToken(lexer.TokenID)
}

// isBlockParams returns true if next token starts a block params
func (p *parser) isBlockParams() bool {
	return p.isToken(lexer.TokenOpenBlockParams)
}

// isInverse returns true if next token starts an INVERSE sequence
func (p *parser) isInverse() bool {
	return p.isToken(lexer.TokenInverse)
}

// isOpenInverseChain returns true if next token is OPEN_INVERSE_CHAIN
func (p *parser) isOpenInverseChain() bool {
	return p.isToken(lexer.TokenOpenInverseChain)
}
