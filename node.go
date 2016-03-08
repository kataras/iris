package iris

import ()

type node struct {
	prefix string
	routes []*Route
}

type tree map[string][]*node

func (tr tree) addRoute(method string, route *Route) {
	_nodes := tr[method]

	if _nodes == nil {
		_nodes = make([]*node, 0)
	}
	ok := false
	var _node *node
	index := 0
	for index, _node = range _nodes {
		//check if route has parameters or * after the prefix, if yes then add a slash to the end
		routePref := route.pathPrefix
		if _node.prefix == routePref {
			tr[method][index].routes = append(_node.routes, route)
			ok = true
			break
		}
	}
	if !ok {
		_node = &node{prefix: route.pathPrefix, routes: make([]*Route, 0)}
		_node.routes = append(_node.routes, route)
		//_node.makePriority(route)
		tr[method] = append(tr[method], _node)
	}
}
