// Package main shows how to create a quite fast custom Log Formatter.
// Note that, this example requires a little more knowledge about Go.
package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/requestid"
)

func logFields(ctx iris.Context, fields *accesslog.Fields) {
	fields.Set("reqid", ctx.GetID())
}

func main() {
	app := iris.New()

	ac := accesslog.File("./access.log").
		AddFields(logFields).
		SetFormatter(newCustomFormatter(' ', "-\t\t\t\t\t"))
	ac.RequestBody = false
	ac.BytesReceivedBody = false
	ac.BytesSentBody = false
	defer ac.Close()

	app.UseRouter(ac.Handler)
	app.UseRouter(requestid.New())

	app.OnErrorCode(iris.StatusNotFound, notFound)
	app.Get("/", index)

	app.Listen(":8080")
}

func notFound(ctx iris.Context) {
	ctx.WriteString("The page you're looking for does not exist!")
}

func index(ctx iris.Context) {
	ctx.WriteString("OK Index")
}

type customFormatter struct {
	w       io.Writer
	bufPool *sync.Pool

	delim byte
	blank string
}

var _ accesslog.Formatter = (*customFormatter)(nil)

func newCustomFormatter(delim byte, blank string) *customFormatter {
	return &customFormatter{delim: delim, blank: blank}
}

func (f *customFormatter) SetOutput(dest io.Writer) {
	f.w = dest
	f.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	if f.delim == 0 {
		f.delim = ' '
	}
}

const newLine = '\n'

func (f *customFormatter) Format(log *accesslog.Log) (bool, error) {
	buf := f.bufPool.Get().(*bytes.Buffer)

	buf.WriteString(log.Now.Format(log.TimeFormat))
	buf.WriteByte(f.delim)

	reqid := log.Fields.GetString("reqid")
	f.writeTextOrBlank(buf, reqid)

	buf.WriteString(uniformDuration(log.Latency))
	buf.WriteByte(f.delim)

	buf.WriteString(log.IP)
	buf.WriteByte(f.delim)

	buf.WriteString(strconv.Itoa(log.Code))
	buf.WriteByte(f.delim)

	buf.WriteString(log.Method)
	buf.WriteByte(f.delim)

	buf.WriteString(log.Path)

	buf.WriteByte(newLine)

	// _, err := buf.WriteTo(f.w)
	// or (to make sure that it resets on errors too):
	_, err := f.w.Write(buf.Bytes())
	buf.Reset()
	f.bufPool.Put(buf)

	return true, err
}

func (f *customFormatter) writeTextOrBlank(buf *bytes.Buffer, s string) {
	if len(s) == 0 {
		if len(f.blank) == 0 {
			return
		}

		buf.WriteString(f.blank)
	} else {
		buf.WriteString(s)
	}

	buf.WriteByte(f.delim)
}

func uniformDuration(t time.Duration) string {
	return fmt.Sprintf("%*s", 12, t.String())
}
