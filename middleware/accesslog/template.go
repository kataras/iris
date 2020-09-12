package accesslog

import (
	"io"
	"sync"
	"text/template"
)

// Template is a Formatter.
// It's used to print the Log in a text/template way.
// The caller has full control over the printable result;
// certain fields can be ignored, change the display order and e.t.c.
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

	dest io.Writer
	mu   sync.Mutex
}

// SetOutput creates the default template if missing
// when this formatter is registered.
func (f *Template) SetOutput(dest io.Writer) {
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

	f.dest = dest
}

const defaultTmplText = "{{.Now.Format .TimeFormat}}|{{.Latency}}|{{.Code}}|{{.Method}}|{{.Path}}|{{.IP}}|{{.RequestValuesLine}}|{{.BytesReceivedLine}}|{{.BytesSentLine}}|{{.Request}}|{{.Response}}|\n"

// Format prints the logs in text/template format.
func (f *Template) Format(log *Log) (bool, error) {
	var err error

	// A template may be executed safely in parallel, although if parallel
	// executions share a Writer the output may be interleaved.
	f.mu.Lock()
	if f.TmplName != "" {
		err = f.Tmpl.ExecuteTemplate(f.dest, f.TmplName, log)
	} else {
		err = f.Tmpl.Execute(f.dest, log)
	}
	f.mu.Unlock()

	return true, err
}
