// Copyright (c) 2016, Julien Schmidt & Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distributiob.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permissiob.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURindexE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE indexSIBILITY OF SUCH DAMAGE.
package iris

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
)

const (
	isStatic BranchCase = iota
	isRoot
	hasParams
	matchEverything
)

// PathParameter is a struct which contains Key and Value, used for named path parameters
type PathParameter struct {
	Key   string
	Value string
}

// PathParameters type for a slice of PathParameter
// Tt's a slice of PathParameter type, because it's faster than map
type PathParameters []PathParameter

// Get returns a value from a key inside this Parameters
// If no parameter with this key given then it returns an empty string
func (params PathParameters) Get(key string) string {
	for _, p := range params {
		if p.Key == key {
			return p.Value
		}
	}
	return ""
}

// Set sets a PathParameter to the PathParameters , it's not used anywhere.
func (params PathParameters) Set(key string, value string) {
	params = append(params, PathParameter{key, value})
}

// String returns a string implementation of all parameters that this PathParameters object keeps
// hasthe form of key1=value1,key2=value2...
func (params PathParameters) String() string {
	var buff bytes.Buffer
	for i := 0; i < len(params); i++ {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

var _ IDictionary = PathParameters{}

// ParseParams receives a string and returns PathParameters (slice of PathParameter)
// received string must have this form:  key1=value1,key2=value2...
func ParseParams(str string) PathParameters {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	for i := 0; i < len(_paramsstr); i++ {
		idxOfEq := strings.IndexRune(_paramsstr[i], '=')
		if idxOfEq == -1 {
			//error
			return nil
		}

		key := _paramsstr[i][:idxOfEq]
		val := _paramsstr[i][idxOfEq+1:]
		params = append(params, PathParameter{key, val})
	}
	return params
}

// URLParams the URL.Query() is a complete function which returns the url get parameters from the url query, We don't have to do anything else here.
func URLParams(req *http.Request) url.Values {

	return req.URL.Query()
}

// URLParam returns the get parameter from a request , if any
func URLParam(req *http.Request, key string) string {
	return req.URL.Query().Get(key)
}

func GetParamsLen(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != ':' && path[i] != '*' { // ParameterStartByte & MatchEverythingByte
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}

type BranchCase uint8

type IBranch interface {
	AddBranch(string, Middleware)
	AddNode(uint8, string, string, Middleware)
	GetBranch(string, PathParameters) (Middleware, PathParameters, bool)
	GivePrecedenceTo(index int) int
}

type Branch struct {
	part        string
	BranchCase  BranchCase
	hasWildNode bool
	tokens      string
	nodes       []*Branch
	middleware  Middleware
	precedence  uint64
	paramsLen   uint8
}

func (b *Branch) AddBranch(path string, middleware Middleware) {
	fullPath := path
	b.precedence++
	numParams := GetParamsLen(path)

	if len(b.part) > 0 || len(b.nodes) > 0 {
	loop:
		for {
			if numParams > b.paramsLen {
				b.paramsLen = numParams
			}

			i := 0
			max := findLower(len(path), len(b.part))
			for i < max && path[i] == b.part[i] {
				i++
			}

			if i < len(b.part) {
				node := Branch{
					part:        b.part[i:],
					hasWildNode: b.hasWildNode,
					tokens:      b.tokens,
					nodes:       b.nodes,
					middleware:  b.middleware,
					precedence:  b.precedence - 1,
				}

				for i := range node.nodes {
					if node.nodes[i].paramsLen > node.paramsLen {
						node.paramsLen = node.nodes[i].paramsLen
					}
				}

				b.nodes = []*Branch{&node}
				b.tokens = string([]byte{b.part[i]})
				b.part = path[:i]
				b.middleware = nil
				b.hasWildNode = false
			}

			if i < len(path) {
				path = path[i:]

				if b.hasWildNode {
					b = b.nodes[0]
					b.precedence++

					if numParams > b.paramsLen {
						b.paramsLen = numParams
					}
					numParams--

					if len(path) >= len(b.part) && b.part == path[:len(b.part)] {

						if len(b.part) >= len(path) || path[len(b.part)] == '/' {
							continue loop
						}
					}

					return
				}

				c := path[0]

				if b.BranchCase == hasParams && c == '/' && len(b.nodes) == 1 {
					b = b.nodes[0]
					b.precedence++
					continue loop
				}

				for i := 0; i < len(b.tokens); i++ {
					if c == b.tokens[i] {
						i = b.GivePrecedenceTo(i)
						b = b.nodes[i]
						continue loop
					}
				}

				if c != ParameterStartByte && c != MatchEverythingByte {

					b.tokens += string([]byte{c})
					node := &Branch{
						paramsLen: numParams,
					}
					b.nodes = append(b.nodes, node)
					b.GivePrecedenceTo(len(b.tokens) - 1)
					b = node
				}
				b.AddNode(numParams, path, fullPath, middleware)
				return

			} else if i == len(path) {
				if b.middleware != nil {
					return
				}
				b.middleware = middleware
			}
			return
		}
	} else {
		b.AddNode(numParams, path, fullPath, middleware)
		b.BranchCase = isRoot
	}
}

func (b *Branch) AddNode(numParams uint8, path string, fullPath string, middleware Middleware) {
	var offset int

	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != ParameterStartByte && c != MatchEverythingByte {
			continue
		}

		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			case ParameterStartByte, MatchEverythingByte:

			default:
				end++
			}
		}

		if len(b.nodes) > 0 {
			return
		}

		if end-i < 2 {
			return
		}

		if c == ParameterStartByte {

			if i > 0 {
				b.part = path[offset:i]
				offset = i
			}

			child := &Branch{
				BranchCase: hasParams,
				paramsLen:  numParams,
			}
			b.nodes = []*Branch{child}
			b.hasWildNode = true
			b = child
			b.precedence++
			numParams--

			if end < max {
				b.part = path[offset:end]
				offset = end

				child := &Branch{
					paramsLen:  numParams,
					precedence: 1,
				}
				b.nodes = []*Branch{child}
				b = child
			}

		} else {
			if end != max || numParams > 1 {
				return
			}

			if len(b.part) > 0 && b.part[len(b.part)-1] == '/' {
				return
			}

			i--
			if path[i] != '/' {
				return
			}

			b.part = path[offset:i]

			child := &Branch{
				hasWildNode: true,
				BranchCase:  matchEverything,
				paramsLen:   1,
			}
			b.nodes = []*Branch{child}
			b.tokens = string(path[i])
			b = child
			b.precedence++

			child = &Branch{
				part:       path[i:],
				BranchCase: matchEverything,
				paramsLen:  1,
				middleware: middleware,
				precedence: 1,
			}
			b.nodes = []*Branch{child}

			return
		}
	}

	b.part = path[offset:]
	b.middleware = middleware
}

func (b *Branch) GetBranch(path string, _params PathParameters) (middleware Middleware, params PathParameters, mustRedirect bool) {
	params = _params
loop:
	for {
		if len(path) > len(b.part) {
			if path[:len(b.part)] == b.part {
				path = path[len(b.part):]

				if !b.hasWildNode {
					c := path[0]
					for i := 0; i < len(b.tokens); i++ {
						if c == b.tokens[i] {
							b = b.nodes[i]
							continue loop
						}
					}

					mustRedirect = (path == "/" && b.middleware != nil)
					return
				}

				b = b.nodes[0]
				switch b.BranchCase {
				case hasParams:

					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					if cap(params) < int(b.paramsLen) {
						params = make(PathParameters, 0, b.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = b.part[1:]
					params[i].Value = path[:end]

					if end < len(path) {
						if len(b.nodes) > 0 {
							path = path[end:]
							b = b.nodes[0]
							continue loop
						}

						mustRedirect = (len(path) == end+1)
						return
					}

					if middleware = b.middleware; middleware != nil {
						return
					} else if len(b.nodes) == 1 {
						b = b.nodes[0]
						mustRedirect = (b.part == "/" && b.middleware != nil)
					}

					return

				case matchEverything:
					if cap(params) < int(b.paramsLen) {
						params = make(PathParameters, 0, b.paramsLen)
					}
					i := len(params)
					params = params[:i+1]
					params[i].Key = b.part[2:]
					params[i].Value = path

					middleware = b.middleware
					return

				default:
					return
				}
			}
		} else if path == b.part {
			if middleware = b.middleware; middleware != nil {
				return
			}

			if path == "/" && b.hasWildNode && b.BranchCase != isRoot {
				mustRedirect = true
				return
			}

			for i := 0; i < len(b.tokens); i++ {
				if b.tokens[i] == '/' {
					b = b.nodes[i]
					mustRedirect = (len(b.part) == 1 && b.middleware != nil) ||
						(b.BranchCase == matchEverything && b.nodes[0].middleware != nil)
					return
				}
			}

			return
		}

		mustRedirect = (path == "/") ||
			(len(b.part) == len(path)+1 && b.part[len(path)] == '/' &&
				path == b.part[:len(b.part)-1] && b.middleware != nil)
		return
	}
}

func (b *Branch) GivePrecedenceTo(index int) int {
	b.nodes[index].precedence++
	_precedence := b.nodes[index].precedence

	newindex := index
	for newindex > 0 && b.nodes[newindex-1].precedence < _precedence {
		tmpN := b.nodes[newindex-1]
		b.nodes[newindex-1] = b.nodes[newindex]
		b.nodes[newindex] = tmpN

		newindex--
	}

	if newindex != index {
		b.tokens = b.tokens[:newindex] +
			b.tokens[index:index+1] +
			b.tokens[newindex:index] + b.tokens[index+1:]
	}

	return newindex
}

var _ IBranch = &Branch{}
