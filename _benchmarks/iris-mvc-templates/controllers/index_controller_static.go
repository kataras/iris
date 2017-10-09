package controllers

import "github.com/kataras/iris/mvc"

type IndexControllerStatic struct{ mvc.C }

var index = mvc.View{
	Name: "index.html",
	Data: map[string]interface{}{
		"Title": "Home Page",
	},
}

func (c *IndexControllerStatic) Get() mvc.View {
	return index
}
