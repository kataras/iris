package like

import (
	"github.com/kataras/iris"
)

type Controller struct {
	iris.Controller
}

func (c *Controller) GetBy(id int64) {
	c.Ctx.Writef("from "+c.Route().Path()+" with ID: %d", id)
}
