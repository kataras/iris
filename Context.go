package gapi

import (
	"net/http"
)

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params     Parameters
}

func NewContext(res http.ResponseWriter, req *http.Request) *Context {
	params := Params(req)
	return &Context{ResponseWriter: res, Request: req, Params: params}
}
