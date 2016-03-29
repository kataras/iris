// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package gzip

import (
	compressGzip "compress/gzip"
	"github.com/kataras/iris"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	encodingGzip = "gzip"

	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
	headerSecWebSocketKey = "Sec-WebSocket-Key"
	// check https://golang.org/src/compress/gzip/gzip.go and compress/flate package
	// for the values of these
	BestCompression    = compressGzip.BestCompression
	BestSpeed          = compressGzip.BestSpeed
	DefaultCompression = compressGzip.DefaultCompression
	NoCompression      = compressGzip.NoCompression
)

type gzipMiddleware struct {
	pool sync.Pool
}

type responseWriter struct {
	iris.IMemoryWriter
	gzipWriter *compressGzip.Writer
}

// Writes to the gzipWriter, we need this in order to the gzip writer acts like a  ResponseWriter
// when .Write called this will be executed
// see: Serve(..) //set the wrapper
func (res responseWriter) Write(b []byte) (int, error) {
	//first check if content type is given, if not set it
	if len(res.Header().Get(headerContentType)) == 0 {
		res.Header().Set(headerContentType, http.DetectContentType(b))
	}
	return res.gzipWriter.Write(b)
}

// Gzip creates the middleware and returns it, for direct Use
// parameter compLevel value between BestSpeed and BestCompression inclusive.
// check https://golang.org/src/compress/gzip/gzip.go
func Gzip(compLevel int) *gzipMiddleware {
	m := &gzipMiddleware{}
	m.pool.New = func() interface{} {
		writer, err := compressGzip.NewWriterLevel(ioutil.Discard, compLevel)
		if err != nil {
			panic(err)
		}
		return writer
	}
	return m
}

func (g *gzipMiddleware) Serve(ctx *iris.Context) {
	res := ctx.ResponseWriter
	header := ctx.Request.Header
	//first, we check if the browser accepts encoding to gzip
	if !strings.Contains(header.Get(headerAcceptEncoding), encodingGzip) {
		ctx.Next()
		return
	}

	//secondly, we check and skip if this is a websocket
	if len(header.Get(headerSecWebSocketKey)) > 0 {

		ctx.Next()
		return
	}

	//don't compress if already compressed
	if header.Get(headerContentEncoding) == encodingGzip {

		ctx.Next()
		return
	}
	//get the gzip writer from the pool clear it's contents and get the new responsewriter's
	writer := g.pool.Get().(*compressGzip.Writer)
	writer.Reset(res)

	//set the headers of the gzip writer
	ctx.SetHeader(headerContentEncoding, []string{encodingGzip}) // "true" kai doulevei kapws
	ctx.SetHeader(headerVary, []string{headerAcceptEncoding})
	//set the wrapper
	newResponseWriter := responseWriter{res, writer}
	//set the ResponseWriter to this, for the serving
	ctx.ResponseWriter = newResponseWriter

	ctx.Next()
	//clear the len,we already write
	ctx.ResponseWriter.Header().Del(headerContentLength)

	//finaly close the gzip writer
	writer.Close()
}
