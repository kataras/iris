package view

// EngineFuncer is an addition of a view engine,
// if a view engine implements that interface
// then iris can add some closed-relative iris functions
// like {{ urlpath }} and {{ urlpath }}.
type EngineFuncer interface {
	// AddFunc should adds a function to the template's function map.
	AddFunc(funcName string, funcBody interface{})
}

// these will be added to all template engines used
// and completes the EngineFuncer interface.
//
// There are a lot of default functions but some of them are placed inside of each
// template engine because of the different behavior, i.e urlpath and url are inside framework itself,
// yield,partial,partial_r,current and render as inside html engine etc...
var defaultSharedFuncs = map[string]interface{}{}
