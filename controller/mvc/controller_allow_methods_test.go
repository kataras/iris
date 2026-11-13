package mvc_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
)

type Project struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ProjectFilter struct {
	Name string `json:"name"`
}

// ProjectController names the method that answered in the Name it returns.
type ProjectController struct{}

func (c *ProjectController) Create(ctx iris.Context, p Project) (Project, error) {
	return Project{ID: 1, Name: "create"}, nil
}

func (c *ProjectController) Get(ctx iris.Context, id int64) (Project, error) {
	return Project{ID: id, Name: "get"}, nil
}

func (c *ProjectController) Update(ctx iris.Context, id int64, p Project, onlyColumns ...string) (*Project, bool, error) {
	return &Project{ID: id, Name: "update"}, true, nil
}

func (c *ProjectController) Delete(ctx iris.Context, id int64) (bool, error) {
	return true, nil
}

func (c *ProjectController) List(ctx iris.Context, filter ProjectFilter, limit, offset int) ([]Project, int, error) {
	return nil, 0, nil
}

// A controller registers GET, PUT and DELETE on the same path. On a router
// with AllowMethods(GET, POST) the GET copies of Update and Delete must not
// take the place of the Get route: GET /project/42 used to run Delete.
// See https://github.com/kataras/iris/issues/1487.
func TestControllerWithAllowMethods(t *testing.T) {
	app := iris.New()

	project := app.Router("/project")
	project.AllowMethods(iris.MethodGet, iris.MethodPost)
	project.Router("/", mvc.UseController[*ProjectController]())

	e := httptest.New(t, app)

	e.GET("/project/42").Expect().Status(httptest.StatusOK).
		JSON().IsEqual(Project{ID: 42, Name: "get"})
	e.PUT("/project/42").WithJSON(Project{}).Expect().Status(httptest.StatusOK).
		JSON().IsEqual(Project{ID: 42, Name: "update"})
	e.DELETE("/project/42").Expect().Status(httptest.StatusNoContent)
	e.POST("/project").WithJSON(Project{}).Expect().Status(httptest.StatusCreated).
		JSON().IsEqual(Project{ID: 1, Name: "create"})
}
