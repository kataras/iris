// Copyright (c) 2016, Gerasimos Maropoulos
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
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

type tree struct {
	method     string
	rootBranch *Branch
}

// Garden is the main area which routes are planted/placed
type Garden []tree

func (g Garden) GetTreeByMethod(method string) *tree {
	for _, _tree := range g {
		if _tree.method == method {
			return &_tree
		}
	}
	return nil
}

func (g Garden) GetRootByMethod(method string) *Branch {
	for _, _tree := range g {
		if _tree.method == method {
			return _tree.rootBranch
		}
	}
	return nil
}

// Plant plants/adds a route to the garden
func (g Garden) Plant(method string, _route IRoute) Garden {
	theRoot := g.GetRootByMethod(method)
	//no tree with that method has found
	if theRoot == nil {
		theRoot = new(Branch)
		g = append(g, tree{method, theRoot})

	}
	theRoot.AddBranch(_route.GetPath(), _route.GetMiddleware())
	return g
}

func (g Garden) Len() int {
	return len(g)
}

func (g Garden) Get(index int) tree {
	return g[index]
}
