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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package utils

import (
	"github.com/kataras/iris/errors"
)

var (
	// ErrNoZip returns an error with message: 'While creating file '+filename'. It's not a zip'
	ErrNoZip = errors.NewError("While installing file '%s'. It's not a zip")
	// ErrFileOpen returns an error with message: 'While opening a file. Trace: +specific error'
	ErrFileOpen = errors.NewError("While opening a file. Trace: %s")
	// ErrFileCreate returns an error with message: 'While creating a file. Trace: +specific error'
	ErrFileCreate = errors.NewError("While creating a file. Trace: %s")
	// ErrFileRemove returns an error with message: 'While removing a file. Trace: +specific error'
	ErrFileRemove = errors.NewError("While removing a file. Trace: %s")
	// ErrFileCopy returns an error with message: 'While copying files. Trace: +specific error'
	ErrFileCopy = errors.NewError("While copying files. Trace: %s")
	// ErrFileDownload returns an error with message: 'While downloading from +specific url. Trace: +specific error'
	ErrFileDownload = errors.NewError("While downloading from %s. Trace: %s")
	// ErrDirCreate returns an error with message: 'Unable to create directory on '+root dir'. Trace: +specific error
	ErrDirCreate = errors.NewError("Unable to create directory on '%s'. Trace: %s")
)
