package view

import (
	"github.com/kataras/go-template/amber"
)

// AmberAdaptor is the  adaptor for the Amber, simple, engine.
// Read more about the Amber Go Template at:
// https://github.com/eknkc/amber
// and https://github.com/kataras/go-template/tree/master/amber
type AmberAdaptor struct {
	*Adaptor
	engine *amber.Engine
}

// Amber returns a new kataras/go-template/amber template engine
// with the same features as all iris' view engines have:
// Binary assets load (templates inside your executable with .go extension)
// Layout, Funcs, {{ url }} {{ urlpath}} for reverse routing and much more.
//
// Read more: https://github.com/eknkc/amber
func Amber(directory string, extension string) *AmberAdaptor {
	e := amber.New()
	return &AmberAdaptor{
		Adaptor: NewAdaptor(directory, extension, e),
		engine:  e,
	}
}

// Funcs adds the elements of the argument map to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (template.HTML, error).
func (a *AmberAdaptor) Funcs(funcMap map[string]interface{}) *AmberAdaptor {
	if len(funcMap) == 0 {
		return a
	}

	// configuration maps are never nil, because
	// they are initialized at each of the engine's New func
	// so we're just passing them inside it.
	for k, v := range funcMap {
		a.engine.Config.Funcs[k] = v
	}

	return a
}
