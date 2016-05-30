// Copyright (c)  2013 Julien Schmidt, Copyright (c) 2016 Gerasimos Maropoulos,

package iris

import (
	"bytes"
	"strings"

	"github.com/kataras/iris/utils"
)

const (
	isStatic BranchCase = iota
	isRoot
	hasParams
	matchEverything
)

type (
	// PathParameter is a struct which contains Key and Value, used for named path parameters
	PathParameter struct {
		Key   string
		Value string
	}

	// PathParameters type for a slice of PathParameter
	// Tt's a slice of PathParameter type, because it's faster than map
	PathParameters []PathParameter

	// BranchCase is the type which the type of Branch using in order to determinate what type (parameterized, anything, static...) is the perticular node
	BranchCase uint8

	// IBranch is the interface which the type Branch must implement
	IBranch interface {
		AddBranch(string, Middleware)
		AddNode(uint8, string, string, Middleware)
		GetBranch(string, PathParameters) (Middleware, PathParameters, bool)
		GivePrecedenceTo(index int) int
	}

	// Branch is the node of a tree of the routes,
	// in order to learn how this is working, google 'trie' or watch this lecture: https://www.youtube.com/watch?v=uhAUk63tLRM
	// this method is used by the BSD's kernel also
	Branch struct {
		part        string
		BranchCase  BranchCase
		hasWildNode bool
		tokens      string
		nodes       []*Branch
		middleware  Middleware
		precedence  uint64
		paramsLen   uint8
	}
)

var _ IBranch = &Branch{}

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

// String returns a string implementation of all parameters that this PathParameters object keeps
// hasthe form of key1=value1,key2=value2...
func (params PathParameters) String() string {
	var buff bytes.Buffer
	for i := range params {
		buff.WriteString(params[i].Key)
		buff.WriteString("=")
		buff.WriteString(params[i].Value)
		if i < len(params)-1 {
			buff.WriteString(",")
		}

	}
	return buff.String()
}

// ParseParams receives a string and returns PathParameters (slice of PathParameter)
// received string must have this form:  key1=value1,key2=value2...
func ParseParams(str string) PathParameters {
	_paramsstr := strings.Split(str, ",")
	if len(_paramsstr) == 0 {
		return nil
	}

	params := make(PathParameters, 0) // PathParameters{}

	//	for i := 0; i < len(_paramsstr); i++ {
	for i := range _paramsstr {
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

// GetParamsLen returns the parameters length from a given path
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

// AddBranch adds a branch to the existing branch or to the tree if no branch has the prefix of
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
			max := utils.FindLower(len(path), len(b.part))
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
				//we need the i here to be re-setting, so use the same i variable as we declare it on line 176
				for i := range b.tokens {
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

// AddNode adds a branch as children to other Branch
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

// GetBranch is used by the Router, it finds and returns the correct branch for a path
func (b *Branch) GetBranch(path string, _params PathParameters) (middleware Middleware, params PathParameters, mustRedirect bool) {
	params = _params
loop:
	for {
		if len(path) > len(b.part) {
			if path[:len(b.part)] == b.part {
				path = path[len(b.part):]

				if !b.hasWildNode {
					c := path[0]
					for i := range b.tokens {
						if c == b.tokens[i] {
							b = b.nodes[i]
							continue loop
						}
					}

					mustRedirect = (path == Slash && b.middleware != nil)
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
						mustRedirect = (b.part == Slash && b.middleware != nil)
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

			if path == Slash && b.hasWildNode && b.BranchCase != isRoot {
				mustRedirect = true
				return
			}

			for i := range b.tokens {
				if b.tokens[i] == '/' {
					b = b.nodes[i]
					mustRedirect = (len(b.part) == 1 && b.middleware != nil) ||
						(b.BranchCase == matchEverything && b.nodes[0].middleware != nil)
					return
				}
			}

			return
		}

		mustRedirect = (path == Slash) ||
			(len(b.part) == len(path)+1 && b.part[len(path)] == '/' &&
				path == b.part[:len(b.part)-1] && b.middleware != nil)
		return
	}
}

// GivePrecedenceTo just adds the priority of this branch by an index
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
