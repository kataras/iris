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
package logger

import (
	"github.com/kataras/iris"
	"io"
)

type Options struct {
}

func DefaultOptions() Options {
	return Options{}
}

type loggerMiddleware struct {
	*iris.Logger
	options Options
}

func (l *loggerMiddleware) Serve(ctx *iris.Context) {

}

func newLoggerMiddleware(writer io.Writer, prefix string, flag int, options ...Options) *loggerMiddleware {
	l := &loggerMiddleware{Logger: iris.NewLogger(writer, prefix, flag)}

	if len(options) > 0 {
		l.options = options[0]
	} else {
		l.options = DefaultOptions()
	}

	return l
}

func DefaultHandler(options ...Options) iris.Handler {
	return newLoggerMiddleware(iris.LoggerOutTerminal, "", 0)
}

func Default(options ...Options) iris.HandlerFunc {
	return DefaultHandler(options...).Serve
}

func CustomHandler(writer io.Writer, prefix string, flag int, options ...Options) iris.Handler {
	return newLoggerMiddleware(writer, prefix, flag, options...)
}

func Custom(writer io.Writer, prefix string, flag int, options ...Options) iris.HandlerFunc {
	return CustomHandler(writer, prefix, flag, options...).Serve
}
