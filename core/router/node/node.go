package node

import (
	"sort"
	"strings"

	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/errors"
)

// Nodes a conversion type for []*node.
type Nodes []*node

type node struct {
	s                 string
	wildcardParamName string   // name of the wildcard parameter, only one per whole Node is allowed
	paramNames        []string // only-names
	children          Nodes
	handlers          context.Handlers
	root              bool
}

// ErrDublicate returned from MakeChild when more than one routes have the same registered path.
var ErrDublicate = errors.New("more than one routes have the same registered path")

// Add adds a node to the tree, returns an ErrDublicate error on failure.
func (nodes *Nodes) Add(path string, handlers context.Handlers) error {
	// resolve params and if that node should be added as root
	var params []string
	var paramStart, paramEnd int
	for {
		paramStart = strings.IndexByte(path[paramEnd:], ':')
		if paramStart == -1 {
			break
		}
		paramStart += paramEnd
		paramStart++
		paramEnd = strings.IndexByte(path[paramStart:], '/')

		if paramEnd == -1 {
			params = append(params, path[paramStart:])
			path = path[:paramStart]
			break
		}
		paramEnd += paramStart
		params = append(params, path[paramStart:paramEnd])
		path = path[:paramStart] + path[paramEnd:]
		paramEnd -= paramEnd - paramStart
	}

	for _, idx := range paramsPos(path) {

		if err := nodes.add(path[:idx], nil, nil, true); err != nil { // take the static path to its own node
			return err
		}
		// create a second, empty, dynamic parameter node without the last slash
		if nidx := idx + 1; len(path) > nidx {
			if err := nodes.add(path[:nidx], nil, nil, true); err != nil {
				return err
			}
		}
	}

	// last,  create the node filled by the full path, parameters and its handlers
	if err := nodes.add(path, params, handlers, true); err != nil {
		return err
	}

	// sort by static path, remember, they were already sorted by subdomains too.
	nodes.Sort()
	return nil
}

func (nodes *Nodes) add(path string, paramNames []string, handlers context.Handlers, root bool) (err error) {
	// wraia etsi doulevei ara
	// na to kanw na exei to node to diko tou wildcard parameter name
	// kai sto telos na pernei auto, me vasi to *paramname
	// alla edw mesa 9a ginete register vasi tou last /

	// set the wildcard param name to the root and its children.
	wildcardIdx := strings.IndexByte(path, '*')
	wildcardParamName := ""
	if wildcardIdx > 0 {
		wildcardParamName = path[wildcardIdx+1:]
		path = path[0:wildcardIdx-1] + "/" // replace *paramName with single slash
	}

loop:
	for _, n := range *nodes {

		minlen := len(n.s)
		if len(path) < minlen {
			minlen = len(path)
		}

		for i := 0; i < minlen; i++ {
			if n.s[i] == path[i] {
				continue
			}
			if i == 0 {
				continue loop
			}

			*n = node{
				s: n.s[:i],
				children: Nodes{
					{
						s:                 n.s[i:],
						wildcardParamName: wildcardParamName,
						paramNames:        n.paramNames,
						children:          n.children,
						handlers:          n.handlers,
					},
					{
						s:                 path[i:],
						wildcardParamName: wildcardParamName,
						paramNames:        paramNames,
						handlers:          handlers,
					},
				},
				root: n.root,
			}
			return
		}

		if len(path) < len(n.s) {
			*n = node{
				s:                 n.s[:len(path)],
				wildcardParamName: wildcardParamName,
				paramNames:        paramNames,
				children: Nodes{
					{
						s:                 n.s[len(path):],
						wildcardParamName: wildcardParamName,
						paramNames:        n.paramNames,
						children:          n.children,
						handlers:          n.handlers,
					},
				},
				handlers: handlers,
				root:     n.root,
			}

			return
		}

		if len(path) > len(n.s) {
			err = n.children.add(path[len(n.s):], paramNames, handlers, false)
			return err
		}

		if len(handlers) == 0 { // missing handlers
			return nil
		}
		if len(n.handlers) > 0 { // n.handlers already setted
			return ErrDublicate.Append("for: %s", n.s)
		}
		n.paramNames = paramNames
		n.handlers = handlers

		return
	}

	n := &node{
		s:                 path,
		wildcardParamName: wildcardParamName,
		paramNames:        paramNames,
		handlers:          handlers,
		root:              root,
	}

	*nodes = append(*nodes, n)
	return
}

// Find resolves the path, fills its params
// and returns the registered to the resolved node's handlers.
func (nodes Nodes) Find(path string, params *context.RequestParams) context.Handlers {
	n, paramValues := nodes.findChild(path, nil)
	if n != nil {
		//	map the params,
		// n.params are the param names
		if len(paramValues) > 0 {
			for i, name := range n.paramNames {
				params.Set(name, paramValues[i])
			}
			// last is the wildcard,
			// if paramValues are exceed from the registered param names.
			// Note that n.wildcardParamName can be not empty but that doesn't meaning
			// that it contains a wildcard path, so the check is required.
			if len(paramValues) > len(n.paramNames) {
				lastWildcardVal := paramValues[len(paramValues)-1]
				params.Set(n.wildcardParamName, lastWildcardVal)
			}
		}
		return n.handlers
	}

	return nil
}

func (nodes Nodes) findChild(path string, params []string) (*node, []string) {
	for _, n := range nodes {
		if n.s == ":" {
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 {
				if len(n.handlers) == 0 {
					return nil, nil
				}
				return n, append(params, path)
			}
			return n.children.findChild(path[paramEnd:], append(params, path[:paramEnd]))
		}

		if !strings.HasPrefix(path, n.s) {
			continue
		}

		if len(path) == len(n.s) { // Node matched until the end of path.
			if len(n.handlers) == 0 {
				return nil, nil
			}
			return n, params
		}

		child, childParamNames := n.children.findChild(path[len(n.s):], params)

		if child == nil || len(child.handlers) == 0 {
			// is wildcard and it is not root neither has children
			if n.s[len(n.s)-1] == '/' && !(n.root && (n.s == "/" || len(n.children) > 0)) {
				if len(n.handlers) == 0 {
					return nil, nil
				}
				return n, append(params, path[len(n.s):])
			}
			continue
		}

		return child, childParamNames
	}
	return nil, nil
}

// childLen returns all the children's and their children's length.
func (n *node) childLen() (i int) {
	for _, n := range n.children {
		i++
		i += n.childLen()
	}
	return
}

func (n *node) isDynamic() bool {
	return n.s == ":"
}

// Sort sets the static paths first.
func (nodes Nodes) Sort() {

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].isDynamic() {
			return false
		}
		if nodes[j].isDynamic() {
			return true
		}
		return nodes[i].childLen() > nodes[j].childLen()
	})

	for _, n := range nodes {
		n.children.Sort()
	}
}

func paramsPos(s string) (pos []int) {
	for i := 0; i < len(s); i++ {
		p := strings.IndexByte(s[i:], ':')
		if p == -1 {
			break
		}
		pos = append(pos, p+i)
		i = p + i
	}
	return
}
