package gapi

import (
	"net/http"
	"strconv"
)

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         Parameters
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
