// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jade

import (
	"fmt"
	"runtime"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	Name string    // name of the template represented by the tree.
	Root *ListNode // top-level root of the tree.
	text string    // text parsed to create the template (or its parent)

	// Parsing only; cleared after parse.
	lex       *lexer
	token     [3]item // three-token lookahead for parser.
	peekCount int

	mixin map[string]*MixinNode
	block map[string]*ListNode
}

// Copy returns a copy of the Tree. Any parsing state is discarded.
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}
	return &Tree{
		Name: t.Name,
		Root: t.Root.CopyList(),
		text: t.text,
	}
}

// next returns the next token.
func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}
	return t.token[t.peekCount]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 item) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 item) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemIdent && token.typ != itemEndL && token.typ != itemEmptyLine {
			break
		}
	}
	// fmt.Println("\t\tnextNonSpace", token.val)
	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemIdent && token.typ != itemEndL && token.typ != itemEmptyLine {
			break
		}
	}
	t.backup()
	return token
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("template:%d: %s", t.token[0].line, format)
	panic(fmt.Errorf(format, args...))
}

//
//
//

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.lex.drain()
			t.lex = nil
		}
		*errp = e.(error)
	}
}

func (t *Tree) Parse(text []byte) (tree *Tree, err error) {
	defer t.recover(&err)
	t.lex = lex(t.Name, text)
	t.text = string(text)
	t.topParse()
	t.lex = nil
	return t, nil
}

// New allocates a new parse tree with the given name.
func New(name string) *Tree {
	return &Tree{
		Name:  name,
		mixin: map[string]*MixinNode{},
		block: map[string]*ListNode{},
	}
}
