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
