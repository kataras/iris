package gapi

import (
	"errors"
	"net/http"
	"reflect"
)

const (
	CHARSET        = "UTF-8"
	CONTENT_HTML   = "text/html"
	CONTENT_JSON   = "application/json"
	CONTENT_JSONP  = "application/javascript"
	CONTENT_BINARY = "application/octet-stream"
	CONTENT_LENGTH = "Content-Length"
	CONTENT_TEXT   = "text/plain"
	CONTENT_TYPE   = "Content-Type"
	CONTENT_XML    = "text/xml"
)

var rendererType reflect.Type

type Renderer struct {
	//Only one TemplateCache per app/router/gapi instance.
	templateCache  *TemplateCache
	responseWriter http.ResponseWriter
}

//Use at HTTPRoute.run
func NewRenderer(writer http.ResponseWriter) *Renderer {
	return &Renderer{responseWriter: writer}
}

func (r *Renderer) check() error {
	if r.templateCache == nil {
		return errors.New("gapi:Error on Renderer : No Template Cache was created yet, please refer to docs at github.com/kataras/gapi.")
	}
	return nil
}

func (r *Renderer) RenderFile(file string, pageContext interface{}) error {
	err := r.check()
	if err != nil {
		return err
	}

	return r.templateCache.ExecuteTemplate(r.responseWriter, file, pageContext)

}

func (r *Renderer) Render(pageContext interface{}) error {
	err := r.check()
	if err != nil {
		return err
	}
	return r.templateCache.Execute(r.responseWriter, pageContext)

}

///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (r *Renderer) HTML(httpStatus int, pageContext interface{}) error {
	r.responseWriter.WriteHeader(httpStatus)
	return r.Render(pageContext)
}

func (r *Renderer) Data(httpStatus int, binaryData []byte) {
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write(binaryData)
}

func (r *Renderer) Text(httpStatus int, text string) {

}

func (r *Renderer) JSON(httpStatus int, obj map[string]string) {

}

func (r *Renderer) JSONP(httpStatus int, obj map[string]string) {

}

func (r *Renderer) XML(httpStatus int, xmlStruct interface{}) {

}
