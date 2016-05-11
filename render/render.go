package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/valyala/fasthttp"
)

const (
	// ContentBinary header value for binary data.
	ContentBinary = "application/octet-stream"
	// ContentHTML header value for HTML data.
	ContentHTML = "text/html"
	// ContentJSON header value for JSON data.
	ContentJSON = "application/json"
	// ContentJSONP header value for JSONP data.
	ContentJSONP = "application/javascript"
	// ContentLength header constant.
	ContentLength = "Content-Length"
	// ContentText header value for Text data.
	ContentText = "text/plain"
	// ContentType header constant.
	ContentType = "Content-Type"
	// ContentXHTML header value for XHTML data.
	ContentXHTML = "application/xhtml+xml"
	// ContentXML header value for XML data.
	ContentXML = "text/xml"
	// Default character encoding.
	defaultCharset = "UTF-8"
)

// helperFuncs had to be moved out. See helpers.go|helpers_pre16.go files.

// Delims represents a set of Left and Right delimiters for HTML template rendering.
type Delims struct {
	// Left delimiter, defaults to {{.
	Left string
	// Right delimiter, defaults to }}.
	Right string
}

// Config is a struct for specifying configuration options for the render.Render object.
type Config struct {
	// Directory to load templates. Default is "templates".
	Directory string
	// Asset function to use in place of directory. Defaults to nil.
	Asset func(name string) ([]byte, error)
	// AssetNames function to use in place of directory. Defaults to nil.
	AssetNames func() []string
	// Layout template name. Will not render a layout if blank (""). Defaults to blank ("").
	Layout string
	// Extensions to parse template files from. Defaults to [".tmpl"].
	Extensions []string
	// Funcs is a slice of FuncMaps to apply to the template upon compilation. This is useful for helper functions. Defaults to [].
	Funcs []template.FuncMap
	// Delims sets the action delimiters to the specified strings in the Delims struct.
	Delims Delims
	// Appends the given character set to the Content-Type header. Default is "UTF-8".
	Charset string
	// Gzip enable it if you want to render with gzip compression. Default is false
	Gzip bool
	// Outputs human readable JSON.
	IndentJSON bool
	// Outputs human readable XML. Default is false.
	IndentXML bool
	// Prefixes the JSON output with the given bytes. Default is false.
	PrefixJSON []byte
	// Prefixes the XML output with the given bytes.
	PrefixXML []byte
	// Allows changing of output to XHTML instead of HTML. Default is "text/html".
	HTMLContentType string
	// If IsDevelopment is set to true, this will recompile the templates on every request. Default is false.
	IsDevelopment bool
	// Unescape HTML characters "&<>" to their original values. Default is false.
	UnEscapeHTML bool
	// Streams JSON responses instead of marshalling prior to sending. Default is false.
	StreamingJSON bool
	// Require that all partials executed in the layout are implemented in all templates using the layout. Default is false.
	RequirePartials bool
	// Deprecated: Use the above `RequirePartials` instead of this. As of Go 1.6, blocks are built in. Default is false.
	RequireBlocks bool
	// Disables automatic rendering of http.StatusInternalServerError when an error occurs. Default is false.
	DisableHTTPErrorRendering bool
}

// Render is a service that provides functions for easily writing JSON, XML,
// binary data, and HTML templates out to a HTTP Response.
type Render struct {
	// Customize Secure with an Options struct.
	config *Config
	// Templates the *template.Template main object
	Templates       *template.Template
	compiledCharset string
}

// New constructs a new Render instance with the supplied configs.
func New(config ...*Config) *Render {
	var c *Config
	if len(config) == 0 {
		c = &Config{}
	} else {
		c = config[0]
	}

	r := &Render{
		config: c,
	}

	r.Prepare()

	return r
}

// Create constructs a new Render instance with the supplied configs. It doesn't build and prepare options yet, you should call the .Prepare for this.
func Create(config *Config) *Render {
	return &Render{config: config}
}

// Prepare if
// Prepare must is called once before anything else inside the New(), this exists because for example Iris doesn't want to compile
// the templates on Render creation but after
func (r *Render) Prepare() {
	r.prepareConfig()
	if err := r.compileTemplates(); err != nil {
		// We don't care about IsDevelopment, it's before server's run, panic
		panic(err)
	}

	// Create a new buffer pool for writing templates into.
	if bufPool == nil {
		bufPool = NewBufferPool(64)
	}
}

func (r *Render) prepareConfig() {
	// Fill in the defaults if need be.
	if len(r.config.Charset) == 0 {
		r.config.Charset = defaultCharset
	}
	r.compiledCharset = "; charset=" + r.config.Charset

	if len(r.config.Directory) == 0 {
		r.config.Directory = "templates"
	}
	if len(r.config.Extensions) == 0 {
		r.config.Extensions = []string{".html"}
	}
	if len(r.config.HTMLContentType) == 0 {
		r.config.HTMLContentType = ContentHTML
	}
}

func (r *Render) compileTemplates() error {
	if r.config.Asset == nil || r.config.AssetNames == nil {
		return r.compileTemplatesFromDir()

	}
	return r.compileTemplatesFromAsset()
}

func (r *Render) compileTemplatesFromDir() error {
	var templateErr error
	dir := r.config.Directory
	r.Templates = template.New(dir)
	r.Templates.Delims(r.config.Delims.Left, r.config.Delims.Right)

	// Walk the supplied directory and compile any files that match our extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Fix same-extension-dirs bug: some dir might be named to: "users.tmpl", "local.html".
		// These dirs should be excluded as they are not valid golang templates, but files under
		// them should be treat as normal.
		// If is a dir, return immediately (dir is not a valid golang template).
		if info == nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = filepath.Ext(rel)
		}

		for _, extension := range r.config.Extensions {
			if ext == extension {
				buf, err := ioutil.ReadFile(path)
				if err != nil {
					panic(err)
				}

				name := (rel[0 : len(rel)-len(ext)])
				tmpl := r.Templates.New(filepath.ToSlash(name))

				// Add our funcmaps.
				for _, funcs := range r.config.Funcs {
					tmpl.Funcs(funcs)
				}

				if !r.config.IsDevelopment {
					//panic in production.
					template.Must(tmpl.Funcs(helperFuncs).Parse(string(buf)))
				} else {
					if _, templateErr = tmpl.Funcs(helperFuncs).Parse(string(buf)); templateErr != nil {
						break //we dont continue to the next templates
					}

				}

				break
			}
		}
		return nil
	})
	return templateErr
}

func (r *Render) compileTemplatesFromAsset() error {
	var templateErr error
	dir := r.config.Directory
	r.Templates = template.New(dir)
	r.Templates.Delims(r.config.Delims.Left, r.config.Delims.Right)

	for _, path := range r.config.AssetNames() {
		if !strings.HasPrefix(path, dir) {
			continue
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			panic(err)
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = "." + strings.Join(strings.Split(rel, ".")[1:], ".")
		}

		for _, extension := range r.config.Extensions {
			if ext == extension {

				buf, err := r.config.Asset(path)
				if err != nil {
					panic(err)
				}

				name := (rel[0 : len(rel)-len(ext)])
				tmpl := r.Templates.New(filepath.ToSlash(name))

				// Add our funcmaps.
				for _, funcs := range r.config.Funcs {
					tmpl.Funcs(funcs)
				}

				if !r.config.IsDevelopment {
					//panic in production.
					template.Must(tmpl.Funcs(helperFuncs).Parse(string(buf)))
				} else {
					if _, templateErr = tmpl.Funcs(helperFuncs).Parse(string(buf)); templateErr != nil {
						break //we dont continue to the next templates
					}
				}
				break
			}
		}
	}
	return templateErr
}

// TemplateLookup is a wrapper around template.Lookup and returns
// the template with the given name that is associated with t, or nil
// if there is no such template.
func (r *Render) TemplateLookup(t string) *template.Template {
	return r.Templates.Lookup(t)
}

func (r *Render) Execute(name string, binding interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	return buf, r.Templates.ExecuteTemplate(buf, name, binding)
}

func (r *Render) addLayoutFuncs(name string, binding interface{}) {
	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf, err := r.Execute(name, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err
		},
		"current": func() (string, error) {
			return name, nil
		},
		"block": func(partialName string) (template.HTML, error) {
			log.Print("Render's `block` implementation is now depericated. Use `partial` as a drop in replacement.")
			fullPartialName := fmt.Sprintf("%s-%s", partialName, name)
			if r.config.RequireBlocks || r.TemplateLookup(fullPartialName) != nil {
				buf, err := r.Execute(fullPartialName, binding)
				// Return safe HTML here since we are rendering our own template.
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
		"partial": func(partialName string) (template.HTML, error) {
			fullPartialName := fmt.Sprintf("%s-%s", partialName, name)
			if r.config.RequirePartials || r.TemplateLookup(fullPartialName) != nil {
				buf, err := r.Execute(fullPartialName, binding)
				// Return safe HTML here since we are rendering our own template.
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
		"render": func(fullPartialName string) (template.HTML, error) {
			buf, err := r.Execute(fullPartialName, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err

		},
	}
	if tpl := r.Templates.Lookup(name); tpl != nil {
		tpl.Funcs(funcs)
	}
}

func (r *Render) prepareHTMLLayout(layout []string) string {
	if len(layout) > 0 {
		return layout[0]
	}

	return r.config.Layout
}

// Render is the generic function called by XML, JSON, Data, HTML, and can be called by custom implementations.
func (r *Render) Render(ctx *fasthttp.RequestCtx, e Engine, data interface{}) error {
	var err error
	if r.config.Gzip {
		err = e.RenderGzip(ctx, data)
	} else {
		err = e.Render(ctx, data)
	}

	if err != nil && !r.config.DisableHTTPErrorRendering {
		ctx.Response.SetBodyString(err.Error())
		ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
	}
	return err
}

// Data writes out the raw bytes as binary data.
func (r *Render) Data(ctx *fasthttp.RequestCtx, status int, v []byte) error {
	head := Head{
		ContentType: ContentBinary,
		Status:      status,
	}

	d := Data{
		Head: head,
	}

	return r.Render(ctx, d, v)
}

// HTML builds up the response from the specified template and bindings.
func (r *Render) HTML(ctx *fasthttp.RequestCtx, status int, name string, binding interface{}, layout ...string) error {
	// If we are in development mode, recompile the templates on every HTML request.
	if r.config.IsDevelopment {
		if err := r.compileTemplates(); err != nil {
			return err
		}
	}

	layoutName := r.prepareHTMLLayout(layout)
	// Assign a layout if there is one.
	if len(layoutName) > 0 {
		r.addLayoutFuncs(name, binding)
		name = layoutName
	}

	head := Head{
		ContentType: r.config.HTMLContentType + r.compiledCharset,
		Status:      status,
	}

	h := HTML{
		Head:      head,
		Name:      name,
		Templates: r.Templates,
	}

	return r.Render(ctx, h, binding)
}

// JSON marshals the given interface object and writes the JSON response.
func (r *Render) JSON(ctx *fasthttp.RequestCtx, status int, v interface{}) error {
	head := Head{
		ContentType: ContentJSON + r.compiledCharset,
		Status:      status,
	}

	j := JSON{
		Head:          head,
		Indent:        r.config.IndentJSON,
		Prefix:        r.config.PrefixJSON,
		UnEscapeHTML:  r.config.UnEscapeHTML,
		StreamingJSON: r.config.StreamingJSON,
	}

	return r.Render(ctx, j, v)
}

// JSONP marshals the given interface object and writes the JSON response.
func (r *Render) JSONP(ctx *fasthttp.RequestCtx, status int, callback string, v interface{}) error {
	head := Head{
		ContentType: ContentJSONP + r.compiledCharset,
		Status:      status,
	}

	j := JSONP{
		Head:     head,
		Indent:   r.config.IndentJSON,
		Callback: callback,
	}

	return r.Render(ctx, j, v)
}

// Text writes out a string as plain text.
func (r *Render) Text(ctx *fasthttp.RequestCtx, status int, v string) error {
	head := Head{
		ContentType: ContentText + r.compiledCharset,
		Status:      status,
	}

	t := Text{
		Head: head,
	}

	return r.Render(ctx, t, v)
}

// XML marshals the given interface object and writes the XML response.
func (r *Render) XML(ctx *fasthttp.RequestCtx, status int, v interface{}) error {
	head := Head{
		ContentType: ContentXML + r.compiledCharset,
		Status:      status,
	}

	x := XML{
		Head:   head,
		Indent: r.config.IndentXML,
		Prefix: r.config.PrefixXML,
	}

	return r.Render(ctx, x, v)
}
