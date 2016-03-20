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
package iris

import (
	"bufio"
	"github.com/kataras/iris/domain"
	"io"
	"net"
	"net/http"
)

type IMemoryWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.CloseNotifier
	WriteString(s string) (int, error)
	//implement http response writer

	Size() int

	Status() int

	//
	IsWritten() bool

	ForceHeader()
}

type MemoryWriter struct {
	http.ResponseWriter
	size   int
	status int
}

func (m *MemoryWriter) New(underlineRes http.ResponseWriter) {
	m.ResponseWriter = underlineRes
	//we have 3 possibilities
	//1 > nothing has written , isWritten() = false, is size -1
	//2 > something written but nothing actualy, the size is 0
	//3 > something written and has contents, size > 0
	m.size = -1
	//default status will be http.StatusOK
	m.status = http.StatusOK
}

func (m *MemoryWriter) IsWritten() bool {
	return m.size != -1
}

func (m *MemoryWriter) Write(data []byte) (int, error) {
	m.ForceHeader()
	size, err := m.ResponseWriter.Write(data)
	m.size += size
	return size, err
}

func (m *MemoryWriter) ForceHeader() {
	if !m.IsWritten() {
		m.size = 0
	}
}

func (m *MemoryWriter) WriteHeader(statusCode int) {
	if statusCode > 0 && statusCode != m.status {
		m.status = statusCode
		m.ResponseWriter.WriteHeader(statusCode)
	}

}

func (m *MemoryWriter) WriteString(s string) (size int, err error) {
	m.ForceHeader()
	size, err = io.WriteString(m.ResponseWriter, s)
	m.size += size
	return
}

func (m *MemoryWriter) Status() int {
	return m.status
}

func (m *MemoryWriter) Size() int {
	return m.size
}

func (m *MemoryWriter) Flush() {
	flusher, done := m.ResponseWriter.(http.Flusher)
	if done {
		flusher.Flush()
	}
}

func (m *MemoryWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if m.size == -1 {
		m.size = 0
	}
	return m.ResponseWriter.(http.Hijacker).Hijack()
}

func (m *MemoryWriter) CloseNotify() <-chan bool {
	return m.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

var _ domain.IMemoryWriter = &MemoryWriter{}
