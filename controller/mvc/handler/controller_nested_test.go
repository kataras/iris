package handler_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

func newNestedAppWithGuide() *iris.Application {
	return iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("*").
		// NoTimeout().
		Compression(true).
		LogRequests(false).
		Health(false, "dev", "kataras").
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, ErrEntityNotFound).
			MapErrors(errors.InvalidArgument, ErrInvalidRecipe)).
		Services(NewFoodRepository, NewRecipeRepository, NewNutritionService).
		Middlewares().
		API("/", newRootController).
		Build()
}

func newNestedApp() *iris.Application {
	app := iris.New()
	opts := errors.NewOptions()
	opts.MapErrors(errors.NotFound, ErrEntityNotFound)
	opts.MapErrors(errors.InvalidArgument, ErrInvalidRecipe)
	app.Use(opts.Handler())

	api := app.Router("/api")
	api.RegisterDependency(NewFoodRepository, NewRecipeRepository, NewNutritionService)
	api.Router("/", newRootController)

	if err := app.Build(); err != nil {
		panic(err)
	}
	return app
}

func TestNestedController(t *testing.T) {
	app := newNestedApp()

	e := httptest.New(t, app)
	e.GET("/api").Expect().Status(httptest.StatusOK).Body().IsEqual("{\"id\":\"root_id\"}\n")
	e.GET("/api/test").Expect().Status(httptest.StatusOK).Body().IsEqual("test normal handler")
	e.GET("/api/nested").Expect().Status(httptest.StatusOK).Body().IsEqual("index of nested controller")
	doNutritionRequests(e, "/api")
}

type rootController struct {
	service *NutritionService
}

func newRootController(service *NutritionService) *rootController {
	return &rootController{service: service}
}

func (c *rootController) Routing(r iris.Router) {
	r.Router("/", iris.Join(
		iris.Join(
			iris.Prefix("/test", func(testR iris.Router) {
				testR.Get("/", func(ctx iris.Context) {
					ctx.WriteString("test normal handler")
				})
			}),
			handler.Index(func(ctx iris.Context) (string, error) {
				return "root_id", nil
			})),
		iris.Prefix("/nested", newNestedController),
		iris.Prefix("/recipes", iris.Join(
			handler.Create(c.service.CreateRecipe),
			handler.UpdateBy(c.service.UpdateRecipe),
			handler.Delete(c.service.DeleteRecipe),
			handler.Get(c.service.GetRecipe),
			handler.List(c.service.ListRecipes),
		)),
		iris.Prefix("/foods", iris.Join(
			handler.Create(c.service.CreateFood),
			handler.Get(c.service.GetFood),
		)),
	))
}

type nestedController struct {
	service *NutritionService
}

func newNestedController(service *NutritionService) *nestedController {
	return &nestedController{service: service}
}

// Test the other way too which is a Routing method returns a Controller.
func (c *nestedController) Routing() iris.Controller {
	return iris.Routing(func(r iris.Router) {
		if c.service == nil {
			panic("nestedController.service is nil")
		}

		r.Get("/", func(ctx iris.Context) {
			ctx.WriteString("index of nested controller")
		})
	})
}
