package index

import (
	"github.com/kataras/iris"
)

type Controller struct {
	iris.Controller
}

func (c *Controller) Get() {
	c.Data["Title"] = "Index"
	c.Tmpl = "index.html"
}
