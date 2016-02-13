package gapi

import (
	"errors"
	"net/http"
	"strconv"
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

var contextType reflect.Type

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         Parameters

	//Only one TemplateCache per app/router/gapi instance.
	//But I don't know if we will have perfomance issues with this. ///TODO: test and bench it.
	templateCache *TemplateCache
}

func NewContext(res http.ResponseWriter, req *http.Request) *Context {
	params := Params(req)
	return &Context{ResponseWriter: res, Request: req, Params: params}
}

//returns the string presantation of the key's value
func (this *Context) Param(key string) string {
	return this.Params.Get(key)
}

//returns the int presantation of the key's value
func (this *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(this.Params.Get(key))
	return val, err
}

func (this *Context) Write(contents string) {
	this.ResponseWriter.Write([]byte(contents))
}

func (this *Context) NotFound() {
	http.NotFound(this.ResponseWriter, this.Request)
}

func (this *Context) Close() {
	this.Request.Body.Close()
}

func (this *Context) ServeFile(path string) {
	http.ServeFile(this.ResponseWriter, this.Request, path)
}

func (this *Context) RenderFile(file string, pageContext interface{}) error {
	if this.templateCache != nil {
		return this.templateCache.ExecuteTemplate(this.ResponseWriter, file, pageContext)
	}

	return errors.New("gapi:Error on Context.Render() : No Template Cache was created yet, please refer to docs at github.com/kataras/gapi.")
}

func (this *Context) Render(pageContext interface{}) error {
	if this.templateCache != nil {
		return this.templateCache.Execute(this.ResponseWriter, pageContext)
	}

	return errors.New("gapi:Error on Context.Render() : No Template Cache was created yet, please refer to docs at github.com/kataras/gapi.")
}
///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (this *Context) HTML(httpStatus int, pageContext interface{}) error {
	this.ResponseWriter.WriteHeader(httpStatus)
	return this.Render(pageContext)
}

func (this *Context) Data(httpStatus int, binaryData []byte) {
	this.ResponseWriter.WriteHeader(httpStatus)
	this.ResponseWriter.Write(binaryData)
}

func (this *Context) Text(httpStatus int, text string) {

}

func (this *Context) JSON(httpStatus int, obj map[string]string) {

}

func (this *Context) JSONP(httpStatus int, obj map[string]string) {

}

func (this *Context) XML(httpStatus int, xmlStruct interface{}) {

}
