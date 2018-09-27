package controllers

import "github.com/kataras/iris/mvc"

type ContactController struct{}

var contactView = mvc.View{
	Name: "contact.html",
	Data: map[string]interface{}{
		"Title":   "Contact",
		"Message": "Your contact page.",
	},
}

func (c *ContactController) Get() mvc.View {
	return contactView
}
