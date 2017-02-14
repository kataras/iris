package template

import (
	"io"
	"path/filepath"

	"github.com/kataras/go-errors"
	"github.com/valyala/bytebufferpool"
)

type (
	// Entries the template Engines with their loader
	Entries []*Entry

	// Entry contains a template Engine and its Loader
	Entry struct {
		Loader *Loader
		Engine Engine
	}

	// Mux is an optional feature, used when you want to use multiple template engines
	// It stores the loaders with each of the template engine,
	// the identifier of each template engine is the (loader's) Extension
	// the registry finds the correct template engine and executes the template
	// so you can use and render a template file by it's file extension
	Mux struct {
		// Reload reloads the template engine on each execute, used when the project is under development status
		// if true the template will reflect the runtime template files changes
		// defaults to false
		Reload bool

		// Entries the template Engines with their loader
		Entries Entries
		// SharedFuncs funcs that will be shared all over the supported template engines
		SharedFuncs map[string]interface{}

		buffer *bytebufferpool.Pool
	}
)

// LoadEngine loads the Engine using its registered loader
// Internal Note:
// Loader can be used without a mux because of this we have this type of function here which just pass itself's field into other itself's field
// which, normally, is not a smart choice.
func (entry *Entry) LoadEngine() error {
	return entry.Loader.LoadEngine(entry.Engine)
}

// LoadAll loads all template engines entries, returns the first error
func (entries Entries) LoadAll() error {
	for i, n := 0, len(entries); i < n; i++ {
		if err := entries[i].LoadEngine(); err != nil {
			return err
		}
	}
	return nil
}

// Find receives a filename, gets its extension and returns the template engine responsible for that file extension
func (entries Entries) Find(filename string) *Entry {
	extension := filepath.Ext(filename)
	// Read-Only no locks needed, at serve/runtime-time the library is not supposed to add new template engines
	for i, n := 0, len(entries); i < n; i++ {
		e := entries[i]
		if e.Loader.Extension == extension {
			return e
		}
	}
	return nil
}

// DefaultMux is the default template mux
// use that to share template engines across multiple sources in the same app.
var DefaultMux = NewMux()

// NewMux returns a new Mux
// Mux is an optional feature, used when you want to use multiple template engines
// It stores the loaders with each of the template engine,
// the identifier of each template engine is the (loader's) Extension
// the registry finds the correct template engine and executes the template
// so you can use and render a template file by it's file extension
func NewMux(sharedFuncs ...map[string]interface{}) *Mux {
	var funcs map[string]interface{}
	if len(sharedFuncs) > 0 {
		funcs = sharedFuncs[0]
	} else {
		funcs = make(map[string]interface{}, 0)
	}

	m := &Mux{
		Entries:     Entries{},
		SharedFuncs: funcs,
		Reload:      false,
		buffer:      &bytebufferpool.Pool{},
	}
	return m
}

// UseFuncMap adds shared template funcs to the Mux
// these funcs are applied to all template engines.
//
// Should be called not called after Load.
func UseFuncMap(funcMap map[string]interface{}) *Mux {
	return DefaultMux.UseFuncMap(funcMap)
}

// UseFuncMap adds shared template funcs to the Mux
// these funcs are applied to all template engines
//
// Should be called not called after Load.
func (m *Mux) UseFuncMap(funcMap map[string]interface{}) *Mux {
	if len(funcMap) == 0 {
		return m
	}

	for k, v := range m.SharedFuncs {
		m.SharedFuncs[k] = v
	}

	// if template engine is already added (but not loaded)
	// then it's valid to add these funcs there.
	for i, n := 0, len(m.Entries); i < n; i++ {
		// add the shared  funcs if template engine supports funcs.
		m.SetFuncMapToEngine(funcMap, m.Entries[i].Engine)
	}

	return m
}

// SetFuncMapToEngine sets or overrides a specific func map to a specific template engine
// SharedFuncs stays untouched here.
//
// Call UseFuncMap if you want to set SharedFuncs and be applied
// to all registered and future template engines registrations.
func SetFuncMapToEngine(funcMap map[string]interface{}, e Engine) *Mux {
	return DefaultMux.SetFuncMapToEngine(funcMap, e)
}

// SetFuncMapToEngine sets or overrides a specific func map to a specific template engine
// SharedFuncs stays untouched here.
//
// Call UseFuncMap if you want to set SharedFuncs and be applied
// to all registered and future template engines registrations.
func (m *Mux) SetFuncMapToEngine(funcMap map[string]interface{}, e Engine) *Mux {
	if len(funcMap) == 0 {
		return m
	}
	// add the shared  funcs if template engine supports funcs.
	if funcer, ok := e.(EngineFuncs); ok {
		if funcer.Funcs() != nil {
			for k, v := range funcMap {
				funcer.Funcs()[k] = v
			}
		}
	}
	return m
}

// AddEngine adds but not loads a template engine, returns the entry's Loader
func AddEngine(e Engine) *Loader {
	return DefaultMux.AddEngine(e)
}

// AddEngine adds but not loads a template engine, returns the entry's Loader
func (m *Mux) AddEngine(e Engine) *Loader {
	// add the shared  funcs
	m.SetFuncMapToEngine(m.SharedFuncs, e)

	entry := &Entry{Engine: e, Loader: NewLoader()}

	m.Entries = append(m.Entries, entry)
	// returns the entry's Loader(pointer)
	return entry.Loader
}

// Load loads all template engines entries, returns the first error
// it just calls and returns the Entries.LoadALl
func Load() error {
	return DefaultMux.Load()
}

// Load loads all template engines entries, returns the first error
// it just calls and returns the Entries.LoadALl
func (m *Mux) Load() error {
	return m.Entries.LoadAll()
}

var (
	errNoTemplateEngineForExt = errors.New("No template engine found for '%s'")
	errTemplateNotFound       = errors.New("Template %s was not found")
)

// ExecuteWriter calls the correct template Engine's ExecuteWriter func
func ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) (err error) {
	return DefaultMux.ExecuteWriter(out, name, binding, options...)
}

// ExecuteWriter calls the correct template Engine's ExecuteWriter func
func (m *Mux) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) (err error) {
	if m == nil {
		//file extension, but no template engine registered
		return errNoTemplateEngineForExt.Format(filepath.Ext(name))
	}

	entry := m.Entries.Find(name)
	if entry == nil {
		return errTemplateNotFound.Format(name)
	}

	if m.Reload {
		if err = entry.LoadEngine(); err != nil {
			return
		}
	}

	return entry.Engine.ExecuteWriter(out, name, binding, options...)
}

// ExecuteString executes a template from a specific template engine and returns its contents result as string, it doesn't renders
func ExecuteString(name string, binding interface{}, options ...map[string]interface{}) (result string, err error) {
	return DefaultMux.ExecuteString(name, binding, options...)
}

// ExecuteString executes a template from a specific template engine and returns its contents result as string, it doesn't renders
func (m *Mux) ExecuteString(name string, binding interface{}, options ...map[string]interface{}) (result string, err error) {
	out := m.buffer.Get()
	defer m.buffer.Put(out)
	err = m.ExecuteWriter(out, name, binding, options...)
	if err == nil {
		result = out.String()
	}
	return
}

var errNoTemplateEngineSupportsRawParsing = errors.New("Not found a valid template engine found which supports raw parser")

// ExecuteRaw read moreon template.go:EngineRawParser
// parse with the first valid EngineRawParser
func ExecuteRaw(src string, wr io.Writer, binding interface{}) error {
	return DefaultMux.ExecuteRaw(src, wr, binding)
}

// ExecuteRaw read moreon template.go:EngineRawParser
// parse with the first valid EngineRawParser
func (m *Mux) ExecuteRaw(src string, wr io.Writer, binding interface{}) error {
	if m == nil {
		//file extension, but no template engine registered
		return errNoTemplateEngineForExt.Format(src)
	}

	for _, e := range m.Entries {
		if p, is := e.Engine.(EngineRawExecutor); is {
			return p.ExecuteRaw(src, wr, binding)
		}
	}
	return errNoTemplateEngineSupportsRawParsing
}

// ExecuteRawString receives raw template source contents and returns it's result as string
func ExecuteRawString(src string, binding interface{}) (result string, err error) {
	return DefaultMux.ExecuteRawString(src, binding)
}

// ExecuteRawString receives raw template source contents and returns it's result as string
func (m *Mux) ExecuteRawString(src string, binding interface{}) (result string, err error) {
	out := m.buffer.Get()
	defer m.buffer.Put(out)
	err = m.ExecuteRaw(src, out, binding)
	if err == nil {
		result = out.String()
	}
	return
}
