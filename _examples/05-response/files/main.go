// File responses: inline, attachment, in-memory and rate-limited.
//
// ServeFile renders the file in the browser, SendFile is ServeFile plus a
// Content-Disposition: attachment header, and ServeContent serves any
// io.ReadSeeker. All three delegate to http.ServeContent, so Range requests and
// If-Modified-Since work out of the box. Each has a WithRate variant that
// throttles the transfer: limit is bytes per second, burst the bucket size in
// bytes.
//
// Learn: serve a file inline or as a download, from disk or from memory, with
// Range support and an optional transfer rate.
// Before: ../content-negotiation
//
// The asset paths are relative to this directory, so start the program from here.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i http://localhost:8080/notes                   # 200, inline, text/markdown
//	curl -OJ http://localhost:8080/download/report        # 200, saved as report-2026.csv
//	curl -i http://localhost:8080/download/slow           # 200, attachment at ~16 KB/s
//	curl -i -r 0-4 http://localhost:8080/generated        # 206 Partial Content, "id,to"
//
// Next: ../streaming-sse
package main

import (
	"bytes"
	"time"

	"github.com/kataras/iris/v14"
)

const (
	notesFile  = "./assets/notes.md"
	reportFile = "./assets/report.csv"
)

func main() {
	app := iris.New()

	// Inline: no attachment header, so the client renders it. A failure sets
	// 404, 403 or 500 and returns the error, so there is nothing left to write.
	app.Get("/notes", func(ctx iris.Context) {
		if err := ctx.ServeFile(notesFile); err != nil {
			ctx.Application().Logger().Warnf("serve %s: %v", notesFile, err)
		}
	})

	// Attachment: the client saves it under the given name. An empty second
	// argument derives the name from the source path.
	app.Get("/download/report", func(ctx iris.Context) {
		ctx.SendFile(reportFile, "report-2026.csv")
	})

	// The same download, throttled to roughly 16 KB/s with a 4 KB bucket. The
	// sample asset fits in the first burst, so point it at a large file to see
	// the throttle take effect.
	app.Get("/download/slow", func(ctx iris.Context) {
		ctx.SendFileWithRate(reportFile, "report-2026.csv", 16*1024, 4*1024)
	})

	// From memory: any io.ReadSeeker plus a modification time. The name only
	// drives content-type detection, it is never sent.
	app.Get("/generated", func(ctx iris.Context) {
		content := bytes.NewReader([]byte("id,total\n1,42\n"))
		ctx.ServeContent(content, "invoice.csv", time.Now())
	})

	app.Listen(":8080")
}
