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

package iris

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/monoculum/formam"
)

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	data := ctx.RequestCtx.Request.Body()

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(jsonObject)

	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("JSON", err.Error())
	}

	return nil
}

// ReadXML reads XML from request's body
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	data := ctx.RequestCtx.Request.Body()

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	err := decoder.Decode(xmlObject)
	//err != nil fix by @shiena
	if err != nil && err != io.EOF {
		return ErrReadBody.Format("XML", err.Error())
	}

	return nil
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {

	// first check if we have multipart form
	form, err := ctx.RequestCtx.MultipartForm()
	if err == nil {
		//we have multipart form

		return ErrReadBody.With(formam.Decode(form.Value, formObject))
	}
	// if no multipart and post arguments ( means normal form)

	if ctx.RequestCtx.PostArgs().Len() > 0 {
		form := make(map[string][]string, ctx.RequestCtx.PostArgs().Len()+ctx.RequestCtx.QueryArgs().Len())
		ctx.RequestCtx.PostArgs().VisitAll(func(k []byte, v []byte) {
			key := string(k)
			value := string(v)
			// for slices
			if form[key] != nil {
				form[key] = append(form[key], value)
			} else {
				form[key] = []string{value}
			}

		})
		ctx.RequestCtx.QueryArgs().VisitAll(func(k []byte, v []byte) {
			key := string(k)
			value := string(v)
			// for slices
			if form[key] != nil {
				form[key] = append(form[key], value)
			} else {
				form[key] = []string{value}
			}
		})

		return ErrReadBody.With(formam.Decode(form, formObject))
	}

	return ErrReadBody.With(ErrNoForm.Return())
}
