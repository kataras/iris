package controllers

import "github.com/kataras/iris/mvc"

type ContactController struct{ mvc.Controller }

func (c *ContactController) Get() {
	c.Data["Title"] = "Contact"
	c.Data["Message"] = "Your contact page."
	c.Tmpl = "contact.html"
}
