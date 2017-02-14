// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jade

import (
	"bytes"
	"fmt"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type node interface {
	Type() nodeType
	position() psn // byte position of start of node in full original input string
	String() string

	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() node

	// tree returns the containing *Tree.
	// It is unexported so all implementations of Node are in this package.
	tree() *tree
	tp() itemType
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t nodeType) Type() nodeType {
	return t
}

// Pos represents a byte position in the original input text from which
// this template was parsed.
type psn int

func (p psn) position() psn {
	return p
}

// listNode holds a sequence of nodes.
type listNode struct {
	nodeType
	psn
	tr    *tree
	Nodes []node // The element nodes in lexical order.
}

func (t *tree) newList(pos psn) *listNode {
	return &listNode{tr: t, nodeType: nodeList, psn: pos}
}

func (l *listNode) append(n node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *listNode) tree() *tree {
	return l.tr
}
func (l *listNode) tp() itemType {
	return 0
}

func (l *listNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

func (l *listNode) CopyList() *listNode {
	if l == nil {
		return l
	}
	n := l.tr.newList(l.psn)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *listNode) Copy() node {
	return l.CopyList()
}
