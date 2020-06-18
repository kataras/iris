package view

// EngineFuncer is an addition of a view engine,
// if a view engine implements that interface
// then iris can add some closed-relative iris functions
// like {{ urlpath }} and {{ urlpath }}.
type EngineFuncer interface {
	// AddFunc should adds a function to the template's function map.
	AddFunc(funcName string, funcBody interface{})
}
