package template

import (
	"io"
)

type (
	// Engine the interface that all template engines must implement
	Engine interface {
		// LoadDirectory builds the templates, usually by directory and extension but these are engine's decisions
		LoadDirectory(directory string, extension string) error
		// LoadAssets loads the templates by binary
		// assetFn is a func which returns bytes, use it to load the templates by binary
		// namesFn returns the template filenames
		LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error

		// ExecuteWriter finds, execute a template and write its result to the out writer
		// options are the optional runtime options can be passed by user and catched by the template engine when render
		// an example of this is the "layout" or "gzip" option
		ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error
	}

	// EngineFuncs is optional interface for the Engine
	// used to insert the helper funcs
	EngineFuncs interface {
		// Funcs should returns the context or the funcs,
		// this property is used in order to register any optional helper funcs
		Funcs() map[string]interface{}
	}

	// EngineRawExecutor is optional interface for the Engine
	// used to receive and parse a raw template string instead of a filename
	EngineRawExecutor interface {
		// ExecuteRaw is super-simple function without options and funcs, it's not used widely
		ExecuteRaw(src string, wr io.Writer, binding interface{}) error
	}
)

// Below are just helpers for my two web frameworks which you can use also to your web app

// NoLayout to disable layout for a particular template file
const NoLayout = "@.|.@no_layout@.|.@" // html/html.go. handlebars/handlebars.go. It's the same but the var is separated care for the future here.

// GetGzipOption receives a default value and the render options map and returns if gzip is enabled for this render action
func GetGzipOption(defaultValue bool, options map[string]interface{}) bool {
	gzipOpt := options["gzip"] // we only need that, so don't create new map to keep the options.
	if b, isBool := gzipOpt.(bool); isBool {
		return b
	}
	return defaultValue
}

// GetCharsetOption receives a default value and the render options  map and returns the correct charset for this render action
func GetCharsetOption(defaultValue string, options map[string]interface{}) string {
	charsetOpt := options["charset"]
	if s, isString := charsetOpt.(string); isString {
		return s
	}
	return defaultValue
}
