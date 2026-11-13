package accesslog

import (
	"bytes"
	"io"
	"text/template"
)

// Template is a Formatter.
// It's used to print the Log in a text/template way.
// The caller has full control over the printable result;
// certain fields can be ignored, change the display order and e.t.c.
//
// For faster execution you can create a custom Formatter
// and compile your own quicktemplate: https://github.com/valyala/quicktemplate
//
// This one uses the standard text/template syntax.
//
// The template is executed with a copy of the Log in which the values a client
// controls (Method, Path, IP, Query, PathParams, Fields, Request and Response)
// are sanitized as in the default format: a line break or a tab becomes a space
// and any other control character is dropped, so a request cannot start a new
// line in the log. A value of Fields or PathParams that is not a string is
// judged by the text it prints as: it keeps its type when that text is clean,
// and its sanitized text takes its place otherwise. Log.Ctx is the request's
// Context itself, and what a template reads through it is printed as it is.
type Template struct {
	// Custom template source.
	// Use this or `Tmpl/TmplName` fields.
	Text string
	// Custom template funcs to used when `Text` is not empty.
	Funcs template.FuncMap

	// Custom template to use, overrides the `Text` and `Funcs` fields.
	Tmpl *template.Template
	// If not empty then this named template/block renders the log line.
	TmplName string

	ac *AccessLog
}

// SetOutput creates the default template if missing
// when this formatter is registered.
func (f *Template) SetOutput(dest io.Writer) {
	f.ac, _ = dest.(*AccessLog)

	if f.Tmpl == nil {
		tmpl := template.New("")

		text := f.Text
		if text != "" {
			tmpl.Funcs(f.Funcs)
		} else {
			text = defaultTmplText
		}

		f.Tmpl = template.Must(tmpl.Parse(text))
	}
}

const defaultTmplText = "{{.Now.Format .TimeFormat}}|{{.Latency}}|{{.Code}}|{{.Method}}|{{.Path}}|{{.IP}}|{{.RequestValuesLine}}|{{.BytesReceivedLine}}|{{.BytesSentLine}}|{{.Request}}|{{.Response}}|\n"

func (f *Template) LogText(log *Log) (string, error) {
	var err error

	// A template prints the values as they are and a line break in one of
	// them starts a new line in the log, so it gets a sanitized copy.
	log = sanitizeLog(log)

	// A template may be executed safely in parallel, although if parallel
	// executions share a Writer the output may be interleaved.
	// We solve that using a buffer pool, no locks when template is executing (x2 performance boost).
	temp := f.ac.bufPool.Get().(*bytes.Buffer)

	if f.TmplName != "" {
		err = f.Tmpl.ExecuteTemplate(temp, f.TmplName, log)
	} else {
		err = f.Tmpl.Execute(temp, log)
	}

	if err != nil {
		// Reset before returning it: a dirty buffer put back into the shared pool
		// poisons whichever log line picks it up next, prepending the failed
		// render and corrupting the delimiter count that downstream consumers
		// rely on.
		temp.Reset()
		f.ac.bufPool.Put(temp)
		return "", err
	}

	text := temp.String()
	temp.Reset()
	f.ac.bufPool.Put(temp)

	return text, nil
}

// Format prints the logs in text/template format.
func (f *Template) Format(log *Log) (bool, error) {
	var err error

	log = sanitizeLog(log) // see the note in the sibling method above.

	// A template may be executed safely in parallel, although if parallel
	// executions share a Writer the output may be interleaved.
	// We solve that using a buffer pool, no locks when template is executing (x2 performance boost).
	temp := f.ac.bufPool.Get().(*bytes.Buffer)

	if f.TmplName != "" {
		err = f.Tmpl.ExecuteTemplate(temp, f.TmplName, log)
	} else {
		err = f.Tmpl.Execute(temp, log)
	}

	if err != nil {
		temp.Reset() // see the note in the sibling method above.
		f.ac.bufPool.Put(temp)
		return true, err
	}

	f.ac.Write(temp.Bytes())
	temp.Reset()
	f.ac.bufPool.Put(temp)
	return true, nil
}
