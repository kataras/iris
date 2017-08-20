package controllers

import "github.com/kataras/iris/mvc"

type IndexController struct{ mvc.Controller }

func (c *IndexController) Get() {
	c.Data["Title"] = "Home Page"
	c.Tmpl = "index.html"
}
