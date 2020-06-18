// Package api contains the handlers for our HTTP Endpoints.
package api

import (
	"time"

	"myapp/service"
	"myapp/sql"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
	"github.com/kataras/iris/v12/middleware/requestid"
)

// Router accepts any required dependencies and returns the main server's handler.
func Router(db sql.Database, secret string) func(iris.Party) {
	return func(r iris.Party) {
		j := jwt.HMAC(15*time.Minute, secret)

		r.Use(requestid.New())
		r.Use(verifyToken(j))
		// Generate a token for testing by navigating to
		// http://localhost:8080/token endpoint.
		// Copy-paste it to a ?token=$token url parameter or
		// open postman and put an Authentication: Bearer $token to get
		// access on create, update and delete endpoinds.

		r.Get("/token", writeToken(j))

		var (
			categoryService = service.NewCategoryService(db)
			productService  = service.NewProductService(db)
		)

		cat := r.Party("/category")
		{
			// TODO: new Use to add middlewares to specific
			// routes per METHOD ( we already have the per path through parties.)
			handler := NewCategoryHandler(categoryService)

			cat.Get("/", handler.List)
			cat.Post("/", handler.Create)
			cat.Put("/", handler.Update)

			cat.Get("/{id:int64}", handler.GetByID)
			cat.Patch("/{id:int64}", handler.PartialUpdate)
			cat.Delete("/{id:int64}", handler.Delete)
			/* You can also do something like that:
			cat.PartyFunc("/{id:int64}", func(c iris.Party) {
				c.Get("/", handler.GetByID)
				c.Post("/", handler.PartialUpdate)
				c.Delete("/", handler.Delete)
			})
			*/

			cat.Get("/{id:int64}/products", handler.ListProducts)
			cat.Post("/{id:int64}/products", handler.InsertProducts(productService))
		}

		prod := r.Party("/product")
		{
			handler := NewProductHandler(productService)

			prod.Get("/", handler.List)
			prod.Post("/", handler.Create)
			prod.Put("/", handler.Update)

			prod.Get("/{id:int64}", handler.GetByID)
			prod.Patch("/{id:int64}", handler.PartialUpdate)
			prod.Delete("/{id:int64}", handler.Delete)
		}

	}
}

func writeToken(j *jwt.JWT) iris.Handler {
	return func(ctx iris.Context) {
		claims := jwt.Claims{
			Issuer:   "https://iris-go.com",
			Audience: jwt.Audience{requestid.Get(ctx)},
		}

		j.WriteToken(ctx, claims)
	}
}

func verifyToken(j *jwt.JWT) iris.Handler {
	return func(ctx iris.Context) {
		// Allow all GET.
		if ctx.Method() == iris.MethodGet {
			ctx.Next()
			return
		}

		j.Verify(ctx)
	}
}
