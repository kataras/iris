package rest

import (
	"github.com/kataras/iris/utils"
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
	// ContentXML header value for XML data.
	ContentXML = "text/xml"
	// Default character encoding.
	defaultCharset = "UTF-8"
)

// bufPool represents a reusable buffer pool for executing templates into.
var bufPool *utils.BufferPool

// Config is a struct for specifying configuration options for the render.Render object.
type Config struct {
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
	// Unescape HTML characters "&<>" to their original values. Default is false.
	UnEscapeHTML bool
	// Streams JSON responses instead of marshalling prior to sending. Default is false.
	StreamingJSON bool
	// Disables automatic rendering of http.StatusInternalServerError when an error occurs. Default is false.
	DisableHTTPErrorRendering bool
}

// Render is a service that provides functions for easily writing JSON, XML,
// binary data, and HTML templates out to a HTTP Response.
type Render struct {
	// Customize Secure with an Options struct.
	Config          *Config
	compiledCharset string
}

// New constructs a new Render instance with the supplied configs.
func New(config ...*Config) *Render {
	var c *Config
	if bufPool == nil {
		bufPool = utils.NewBufferPool(64)
	}

	if len(config) == 0 {
		c = &Config{}
	} else {
		c = config[0]
	}

	r := &Render{
		Config: c,
	}

	r.prepareConfig()

	return r
}

func DefaultConfig() *Config {
	return &Config{Charset: defaultCharset}
}

func (r *Render) prepareConfig() {
	// Fill in the defaults if need be.
	if len(r.Config.Charset) == 0 {
		r.Config = DefaultConfig()
	}
	r.compiledCharset = "; charset=" + r.Config.Charset
}

// Render is the generic function called by XML, JSON, Data, HTML, and can be called by custom implementations.
func (r *Render) Render(ctx *fasthttp.RequestCtx, e Engine, data interface{}) error {
	var err error
	if r.Config.Gzip {
		err = e.RenderGzip(ctx, data)
	} else {
		err = e.Render(ctx, data)
	}

	if err != nil && !r.Config.DisableHTTPErrorRendering {
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

// JSON marshals the given interface object and writes the JSON response.
func (r *Render) JSON(ctx *fasthttp.RequestCtx, status int, v interface{}) error {
	head := Head{
		ContentType: ContentJSON + r.compiledCharset,
		Status:      status,
	}

	j := JSON{
		Head:          head,
		Indent:        r.Config.IndentJSON,
		Prefix:        r.Config.PrefixJSON,
		UnEscapeHTML:  r.Config.UnEscapeHTML,
		StreamingJSON: r.Config.StreamingJSON,
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
		Indent:   r.Config.IndentJSON,
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
		Indent: r.Config.IndentXML,
		Prefix: r.Config.PrefixXML,
	}

	return r.Render(ctx, x, v)
}
