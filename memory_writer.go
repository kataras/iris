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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

/*
var (
	encodingGzip = "gzip"

	headerAcceptEncoding = "Accept-Encoding"
)*/

// IMemoryWriter is the interface which the MemoryWriter should implement, implements the http.ResponseWriter
type IMemoryWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.CloseNotifier
	Reset(underlineRes http.ResponseWriter)
	WriteString(s string) (int, error)
	//implement http response writer

	Size() int

	Status() int

	//
	IsWritten() bool

	ForceHeader()
}

// MemoryWriter is used inside Context instead of the http.ResponseWriter, used to have the maximum access and modification via middlewares to the ResponseWriter
// also offers faster produce of each request's response writer
type MemoryWriter struct {
	http.ResponseWriter
	size   int
	status int
}

var _ IMemoryWriter = &MemoryWriter{}

// Reset takes an underline http.ResponseWriter and resets the particular MemoryWriter with this underline http.ResponseWriter
func (m *MemoryWriter) Reset(underlineRes http.ResponseWriter) {
	m.ResponseWriter = underlineRes
	//we have 3 possibilities
	//1 > nothing has written , isWritten() = false, is size -1
	//2 > something written but nothing actualy, the size is 0
	//3 > something written and has contents, size > 0
	m.size = -1
	//default status will be http.StatusOK
	m.status = http.StatusOK
}

// IsWritten returns true if we already wrote to the MemoryWriter
func (m *MemoryWriter) IsWritten() bool {
	return m.size != -1
}

func (m *MemoryWriter) Write(data []byte) (int, error) {
	m.ForceHeader()
	size, err := m.ResponseWriter.Write(data)
	m.size += size
	return size, err
}

// ForceHeader forces  to write the header and reset's the size of the ResponseWriter
func (m *MemoryWriter) ForceHeader() {
	if !m.IsWritten() {
		m.size = 0
	}
}

// WriteHeader writes an http status code
func (m *MemoryWriter) WriteHeader(statusCode int) {
	if statusCode > 0 && statusCode != m.status {
		m.status = statusCode
		m.ResponseWriter.WriteHeader(statusCode)
	}

}

// WriteString using io.WriteString to write a string
func (m *MemoryWriter) WriteString(s string) (size int, err error) {
	m.ForceHeader()
	/* doesn't work and it's stupid to make it here, let it for now.. if strings.Contains(header.Get(headerAcceptEncoding), encodingGzip) {
		b := []byte(s)
		size = len(b)
		m.size += size
		m.ResponseWriter.Write(b)
		return
	}*/
	size, err = io.WriteString(m.ResponseWriter, s)
	m.size += size
	return
}

// Status returns the http status code
func (m *MemoryWriter) Status() int {
	return m.status
}

// Size returns the size of the writer
func (m *MemoryWriter) Size() int {
	return m.size
}

// Flush flushes the contents of the writer
func (m *MemoryWriter) Flush() {
	flusher, done := m.ResponseWriter.(http.Flusher)
	if done {
		flusher.Flush()
	}
}

// Hijack look inside net/http package
func (m *MemoryWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if m.size == -1 {
		m.size = 0
	}
	return m.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify look inside net/http package
func (m *MemoryWriter) CloseNotify() <-chan bool {
	return m.ResponseWriter.(http.CloseNotifier).CloseNotify()
}
