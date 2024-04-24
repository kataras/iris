package controllers

import (
	"net"
	"time"

	"myapp/models"

	"github.com/kataras/golog"
	"gopkg.in/reform.v1"
)

// PersonController is the model.Person's web controller.
type PersonController struct {
	DB *reform.DB

	// Logger and IP fields are automatically binded by the framework.
	Logger *golog.Logger // binds to the application's logger.
	IP     net.IP        // binds to the client's IP.
}

// Get handles
// GET /persons
func (c *PersonController) Get() ([]reform.Struct, error) {
	return c.DB.SelectAllFrom(models.PersonTable, "")
}

// GetBy handles
// GET /persons/{ID}
func (c *PersonController) GetBy(id int32) (reform.Record, error) {
	return c.DB.FindByPrimaryKeyFrom(models.PersonTable, id)
}

// Post handles
// POST /persons with JSON request body of model.Person.
func (c *PersonController) Post(p *models.Person) int {
	p.CreatedAt = time.Now()

	if err := c.DB.Save(p); err != nil {
		c.Logger.Errorf("[%s] create person: %v", c.IP.String(), err)
		return 500 // iris.StatusInternalServerError
	}

	c.Logger.Debugf("[%s] create person [%s] succeed", c.IP.String(), p.Name)

	return 201 // iris.StatusCreated
}
