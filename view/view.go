package view

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/kataras/iris/v12/context"

	"github.com/kataras/golog"
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
type ErrNotExist = context.ErrViewNotExist

// View is just a wrapper on top of the registered template engine.
type View struct{ Engine }

// Register registers a view engine.
func (v *View) Register(e Engine) {
	if v.Engine != nil {
		golog.Warnf("Engine already exists, replacing the old %q with the new one %q", v.Engine.Name(), e.Name())
	}

	v.Engine = e
}

// Registered reports whether an engine was registered.
func (v *View) Registered() bool {
	return v.Engine != nil
}

func (v *View) ensureTemplateName(s string) string {
	if s == "" || s == NoLayout {
		return s
	}

	s = strings.TrimPrefix(s, "/")

	if ext := v.Engine.Ext(); ext != "" {
		if !strings.HasSuffix(s, ext) {
			return s + ext
		}
	}

	return s
}

// ExecuteWriter calls the correct view Engine's ExecuteWriter func
func (v *View) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	filename = v.ensureTemplateName(filename)
	layout = v.ensureTemplateName(layout)

	return v.Engine.ExecuteWriter(w, filename, layout, bindingData)
}

// AddFunc adds a function to all registered engines.
// Each template engine that supports functions has its own AddFunc too.
func (v *View) AddFunc(funcName string, funcBody interface{}) {
	if !v.Registered() {
		return
	}

	if e, ok := v.Engine.(EngineFuncer); ok {
		e.AddFunc(funcName, funcBody)
	}
}

// Funcs registers a template func map to the registered view engine(s).
func (v *View) Funcs(m template.FuncMap) *View {
	if !v.Registered() {
		return v
	}

	if e, ok := v.Engine.(EngineFuncer); ok {
		for k, v := range m {
			e.AddFunc(k, v)
		}
	}

	return v
}

// Load compiles all the registered engines.
func (v *View) Load() error {
	if !v.Registered() {
		return fmt.Errorf("no engine was registered")
	}
	return v.Engine.Load()
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
