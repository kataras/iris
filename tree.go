package iris

// Routes is just a slice of Route pointers
type Routes []*Route
type trees map[string]Routes //key is the HTTPMethod, value is an array of Routes now.
// Garden is the main area which trees are planted/placed
type Garden map[string]*node // node here is the root node

func (_trees trees) addRoute(method string, route *Route) {
	if _trees[method] == nil {
		_trees[method] = make(Routes, 0)
	}
	_trees[method] = append(_trees[method], route)
}

// called only one at the Router's Build state
func (g Garden) plant(tempTrees trees) {
	for method, routes := range tempTrees {
		for _, _route := range routes {
			if g[method] == nil {
				g[method] = new(node)
			}
			g[method].addRoute(_route.fullpath, &_route.middleware)
		}
	}
}

func (g Garden) get(method string) *node {
	return g[method]
}
