package view

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12/context"
)

type (
	// Engine is the interface for a compatible Iris view engine.
	// It's an alias of context.ViewEngine.
	Engine = context.ViewEngine
	// EngineFuncer is the interface for a compatible Iris view engine
	// which accepts builtin framework functions such as url, urlpath and tr.
	// It's an alias of context.ViewEngineFuncer.
	EngineFuncer = context.ViewEngineFuncer
)

// ErrNotExist reports whether a template was not found in the parsed templates tree.
type ErrNotExist struct {
	Name     string
	IsLayout bool
}

// Error implements the `error` interface.
func (e ErrNotExist) Error() string {
	title := "template"
	if e.IsLayout {
		title = "layout"
	}
	return fmt.Sprintf("%s '%s' does not exist", title, e.Name)
}

// View is responsible to
// load the correct templates
// for each of the registered view engines.
type View struct {
	engines []Engine
}

// Register registers a view engine.
func (v *View) Register(e Engine) {
	v.engines = append(v.engines, e)
}

// Find receives a filename, gets its extension and returns the view engine responsible for that file extension
func (v *View) Find(filename string) Engine {
	// Read-Only no locks needed, at serve/runtime-time the library is not supposed to add new view engines
	for i, n := 0, len(v.engines); i < n; i++ {
		e := v.engines[i]
		if strings.HasSuffix(filename, e.Ext()) {
			return e
		}
	}
	return nil
}

// Len returns the length of view engines registered so far.
func (v *View) Len() int {
	return len(v.engines)
}

// ExecuteWriter calls the correct view Engine's ExecuteWriter func
func (v *View) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	if len(filename) > 2 {
		if filename[0] == '/' { // omit first slash
			filename = filename[1:]
		}
	}

	e := v.Find(filename)
	if e == nil {
		return fmt.Errorf("no view engine found for '%s'", filepath.Ext(filename))
	}

	return e.ExecuteWriter(w, filename, layout, bindingData)
}

// AddFunc adds a function to all registered engines.
// Each template engine that supports functions has its own AddFunc too.
func (v *View) AddFunc(funcName string, funcBody interface{}) {
	for i, n := 0, len(v.engines); i < n; i++ {
		e := v.engines[i]
		if engineFuncer, ok := e.(EngineFuncer); ok {
			engineFuncer.AddFunc(funcName, funcBody)
		}
	}
}

// Load compiles all the registered engines.
func (v *View) Load() error {
	for i, n := 0, len(v.engines); i < n; i++ {
		e := v.engines[i]
		if err := e.Load(); err != nil {
			return err
		}
	}
	return nil
}

// NoLayout disables the configuration's layout for a specific execution.
const NoLayout = "iris.nolayout"

// returns empty if it's no layout or empty layout and empty configuration's layout.
func getLayout(layout string, globalLayout string) string {
	if layout == NoLayout {
		return ""
	}

	if layout == "" && globalLayout != "" {
		return globalLayout
	}

	return layout
}
