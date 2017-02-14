package httpexpect

import (
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/moul/http2curl"
)

// CompactPrinter implements Printer. It prints requests in compact form.
type CompactPrinter struct {
	logger Logger
}

// NewCompactPrinter returns a new CompactPrinter given a logger.
func NewCompactPrinter(logger Logger) CompactPrinter {
	return CompactPrinter{logger}
}

// Request implements Printer.Request.
func (p CompactPrinter) Request(req *http.Request) {
	if req != nil {
		p.logger.Logf("%s %s", req.Method, req.URL)
	}
}

// Response implements Printer.Response.
func (CompactPrinter) Response(*http.Response, time.Duration) {
}

// DebugPrinter implements Printer. Uses net/http/httputil to dump
// both requests and responses.
type DebugPrinter struct {
	logger Logger
	body   bool
}

// NewDebugPrinter returns a new DebugPrinter given a logger and body
// flag. If body is true, request and response body is also printed.
func NewDebugPrinter(logger Logger, body bool) DebugPrinter {
	return DebugPrinter{logger, body}
}

// Request implements Printer.Request.
func (p DebugPrinter) Request(req *http.Request) {
	if req == nil {
		return
	}

	dump, err := httputil.DumpRequest(req, p.body)
	if err != nil {
		panic(err)
	}
	p.logger.Logf("%s", dump)
}

// Response implements Printer.Response.
func (p DebugPrinter) Response(resp *http.Response, duration time.Duration) {
	if resp == nil {
		return
	}

	dump, err := httputil.DumpResponse(resp, p.body)
	if err != nil {
		panic(err)
	}

	text := strings.Replace(string(dump), "\r\n", "\n", -1)
	lines := strings.SplitN(text, "\n", 2)

	p.logger.Logf("%s %s\n%s", lines[0], duration, lines[1])
}

// CurlPrinter implements Printer. Uses http2curl to dump requests as
// curl commands.
type CurlPrinter struct {
	logger Logger
}

// NewCurlPrinter returns a new CurlPrinter given a logger.
func NewCurlPrinter(logger Logger) CurlPrinter {
	return CurlPrinter{logger}
}

// Request implements Printer.Request.
func (p CurlPrinter) Request(req *http.Request) {
	if req != nil {
		cmd, err := http2curl.GetCurlCommand(req)
		if err != nil {
			panic(err)
		}
		p.logger.Logf("%s", cmd.String())
	}
}

// Response implements Printer.Response.
func (CurlPrinter) Response(*http.Response, time.Duration) {
}
