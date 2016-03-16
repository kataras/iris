package iris

// Garden is the main area which routes are planted/placed
type Garden map[string]*node // node here is the root node
// plant plants/adds a route to the garden
func (g Garden) plant(method string, _route *Route) {
	if g[method] == nil {
		g[method] = new(node)
	}

	g[method].addRoute(_route.fullpath, _route.middleware)

}

func (g Garden) get(method string) *node {
	return g[method]
}
