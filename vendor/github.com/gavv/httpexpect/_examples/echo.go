package examples

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// EchoHandler creates http.Handler using echo framework.
//
// Routes:
//  GET /login             authenticate user and return JWT token
//  GET /restricted/hello  return "hello, world!" (requires authentication)
func EchoHandler() http.Handler {
	e := echo.New()

	e.POST("/login", func(ctx echo.Context) error {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")

		if username == "ford" && password == "betelgeuse7" {
			// create token
			token := jwt.New(jwt.SigningMethodHS256)

			// generate encoded token and send it as response
			t, err := token.SignedString([]byte("secret"))
			if err != nil {
				return err
			}
			return ctx.JSON(http.StatusOK, map[string]string{
				"token": t,
			})
		}

		return echo.ErrUnauthorized
	})

	r := e.Group("/restricted")

	r.Use(middleware.JWT([]byte("secret")))

	r.GET("/hello", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "hello, world!")
	})

	return e
}
