package gapi

import (
	"net/http"
	"strconv"
	"errors"
)

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
	return val,err
}


func (this *Context) Write(contents string) {
	this.ResponseWriter.Write([]byte(contents))
}

func (this *Context) NotFound() {
	http.NotFound(this.ResponseWriter,this.Request)
}

func (this *Context) Close() {
	this.Request.Body.Close()
}

func (this *Context) ServeFile(path string) {
	http.ServeFile(this.ResponseWriter, this.Request,path)
}

func (this *Context) RenderFile(file string, pageContext interface{}) error {
	if this.templateCache != nil {
		return this.templateCache.ExecuteTemplate(this.ResponseWriter,file,pageContext)
	}
	
	return errors.New("gapi:Error on Context.Render() : No Template Cache was created yet, please refer to docs at github.com/kataras/gapi.")
}

func (this *Context) Render(pageContext interface{}) error {
	if this.templateCache != nil {
		return this.templateCache.Execute(this.ResponseWriter,pageContext)
	}
	
	return errors.New("gapi:Error on Context.Render() : No Template Cache was created yet, please refer to docs at github.com/kataras/gapi.")
}
