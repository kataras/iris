// +build go1.9

package controllers

import (
	"github.com/kataras/iris"
)

// Index is our index example controller.
type Index struct {
	iris.Controller
}

func (c *Index) Get() {
	c.Tmpl = "index.html"
	c.Data["title"] = "Index page"
	c.Data["message"] = "Hello world!"
}
