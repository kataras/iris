package rest

import (
	"bytes"
	"encoding/json"
	"encoding/xml"

	"github.com/klauspost/compress/gzip"
	"github.com/valyala/fasthttp"
)

// Engine is the generic interface for all responses.
type Engine interface {
	Render(*fasthttp.RequestCtx, interface{}) error
	//used only if config gzip is enabled
	RenderGzip(*fasthttp.RequestCtx, interface{}) error
}

// Head defines the basic ContentType and Status fields.
type Head struct {
	ContentType string
	Status      int
}

// Data built-in renderer.
type Data struct {
	Head
}

// JSON built-in renderer.
type JSON struct {
	Head
	Indent        bool
	UnEscapeHTML  bool
	Prefix        []byte
	StreamingJSON bool
}

// JSONP built-in renderer.
type JSONP struct {
	Head
	Indent   bool
	Callback string
}

// Text built-in renderer.
type Text struct {
	Head
}

// XML built-in renderer.
type XML struct {
	Head
	Indent bool
	Prefix []byte
}

// Write outputs the header content.
func (h Head) Write(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set(ContentType, h.ContentType)
	ctx.SetStatusCode(h.Status)
}

// Render a data response.
func (d Data) Render(ctx *fasthttp.RequestCtx, v interface{}) error {
	c := string(ctx.Request.Header.Peek(ContentType))
	w := ctx.Response.BodyWriter()
	if c != "" {
		d.Head.ContentType = c
	}

	d.Head.Write(ctx)
	w.Write(v.([]byte))
	return nil

}

// RenderGzip a data response using gzip compression.
func (d Data) RenderGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	c := string(ctx.Request.Header.Peek(ContentType))
	if c != "" {
		d.Head.ContentType = c
	}

	d.Head.Write(ctx)
	_, err := fasthttp.WriteGzip(ctx.Response.BodyWriter(), v.([]byte))
	if err == nil {
		ctx.Response.Header.Add("Content-Encoding", "gzip")
	}
	return err
}

// Render a JSON response.
func (j JSON) Render(ctx *fasthttp.RequestCtx, v interface{}) error {
	if j.StreamingJSON {
		return j.renderStreamingJSON(ctx, v)
	}

	var result []byte
	var err error

	if j.Indent {
		result, err = json.MarshalIndent(v, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}

	// Unescape HTML if needed.
	if j.UnEscapeHTML {
		result = bytes.Replace(result, []byte("\\u003c"), []byte("<"), -1)
		result = bytes.Replace(result, []byte("\\u003e"), []byte(">"), -1)
		result = bytes.Replace(result, []byte("\\u0026"), []byte("&"), -1)
	}
	w := ctx.Response.BodyWriter()
	// JSON marshaled fine, write out the result.
	j.Head.Write(ctx)
	if len(j.Prefix) > 0 {
		w.Write(j.Prefix)
	}
	w.Write(result)
	return nil
}

// RenderGzip a JSON response using gzip compression.
func (j JSON) RenderGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	if j.StreamingJSON {
		return j.renderStreamingJSONGzip(ctx, v)
	}

	var result []byte
	var err error

	if j.Indent {
		result, err = json.MarshalIndent(v, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	ctx.Response.Header.Add("Content-Encoding", "gzip")

	// Unescape HTML if needed.
	if j.UnEscapeHTML {
		result = bytes.Replace(result, []byte("\\u003c"), []byte("<"), -1)
		result = bytes.Replace(result, []byte("\\u003e"), []byte(">"), -1)
		result = bytes.Replace(result, []byte("\\u0026"), []byte("&"), -1)
	}
	w := gzip.NewWriter(ctx.Response.BodyWriter())
	// JSON marshaled fine, write out the result.
	j.Head.Write(ctx)
	if len(j.Prefix) > 0 {
		w.Write(j.Prefix)
	}
	w.Write(result)
	w.Close()
	return nil
}

func (j JSON) renderStreamingJSON(ctx *fasthttp.RequestCtx, v interface{}) error {
	j.Head.Write(ctx)
	w := ctx.Response.BodyWriter()
	if len(j.Prefix) > 0 {
		w.Write(j.Prefix)
	}
	return json.NewEncoder(w).Encode(v)
}

func (j JSON) renderStreamingJSONGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	ctx.Response.Header.Add("Content-Encoding", "gzip")
	j.Head.Write(ctx)
	w := gzip.NewWriter(ctx.Response.BodyWriter())
	if len(j.Prefix) > 0 {
		w.Write(j.Prefix)
	}
	w.Close()
	return json.NewEncoder(w).Encode(v)
}

// Render a JSONP response.
func (j JSONP) Render(ctx *fasthttp.RequestCtx, v interface{}) error {
	var result []byte
	var err error

	if j.Indent {
		result, err = json.MarshalIndent(v, "", "  ")
	} else {
		result, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	w := ctx.Response.BodyWriter()

	// JSON marshaled fine, write out the result.
	j.Head.Write(ctx)
	w.Write([]byte(j.Callback + "("))
	w.Write(result)
	w.Write([]byte(");"))

	// If indenting, append a new line.
	if j.Indent {
		w.Write([]byte("\n"))
	}
	return nil
}

// RenderGzip a JSONP response using gzip compression.
func (j JSONP) RenderGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	var result []byte
	var err error

	if j.Indent {
		result, err = json.MarshalIndent(v, "", "  ")
	} else {
		result, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	w := gzip.NewWriter(ctx.Response.BodyWriter())

	ctx.Response.Header.Add("Content-Encoding", "gzip")
	// JSON marshaled fine, write out the result.
	j.Head.Write(ctx)
	w.Write([]byte(j.Callback + "("))
	w.Write(result)
	w.Write([]byte(");"))

	// If indenting, append a new line.
	if j.Indent {
		w.Write([]byte("\n"))
	}
	w.Close()
	return nil
}

// Render a text response.
func (t Text) Render(ctx *fasthttp.RequestCtx, v interface{}) error {
	c := string(ctx.Request.Header.Peek(ContentType))
	if c != "" {
		t.Head.ContentType = c
	}
	w := ctx.Response.BodyWriter()
	t.Head.Write(ctx)
	w.Write([]byte(v.(string)))
	return nil
}

// RenderGzip a Text response using gzip compression.
func (t Text) RenderGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	c := string(ctx.Request.Header.Peek(ContentType))
	if c != "" {
		t.Head.ContentType = c
	}
	ctx.Response.Header.Add("Content-Encoding", "gzip")
	t.Head.Write(ctx)
	fasthttp.WriteGzip(ctx.Response.BodyWriter(), []byte(v.(string)))

	return nil
}

// Render an XML response.
func (x XML) Render(ctx *fasthttp.RequestCtx, v interface{}) error {
	var result []byte
	var err error

	if x.Indent {
		result, err = xml.MarshalIndent(v, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = xml.Marshal(v)
	}
	if err != nil {
		return err
	}

	// XML marshaled fine, write out the result.
	x.Head.Write(ctx)
	w := ctx.Response.BodyWriter()
	if len(x.Prefix) > 0 {
		w.Write(x.Prefix)
	}
	w.Write(result)
	return nil
}

// RenderGzip an XML response using gzip compression.
func (x XML) RenderGzip(ctx *fasthttp.RequestCtx, v interface{}) error {
	var result []byte
	var err error

	if x.Indent {
		result, err = xml.MarshalIndent(v, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = xml.Marshal(v)
	}
	if err != nil {
		return err
	}
	ctx.Response.Header.Add("Content-Encoding", "gzip")
	// XML marshaled fine, write out the result.
	x.Head.Write(ctx)
	w := gzip.NewWriter(ctx.Response.BodyWriter())
	if len(x.Prefix) > 0 {
		w.Write(x.Prefix)
	}
	w.Write(result)
	w.Close()
	return nil
}
