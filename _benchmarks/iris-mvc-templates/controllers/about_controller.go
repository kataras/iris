package controllers

import "github.com/kataras/iris/mvc"

type AboutController struct{}

var aboutView = mvc.View{
	Name: "about.html",
	Data: map[string]interface{}{
		"Title":   "About",
		"Message": "Your application description page..",
	},
}

func (c *AboutController) Get() mvc.View {
	return aboutView
}
