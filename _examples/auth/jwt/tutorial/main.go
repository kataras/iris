package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
	"github.com/kataras/iris/v12/middleware/jwt/blocklist/redis"

	// Optionally to set token identifier.
	"github.com/google/uuid"
)

var (
	signatureSharedKey = []byte("sercrethatmaycontainch@r32length")

	signer   = jwt.NewSigner(jwt.HS256, signatureSharedKey, 15*time.Minute)
	verifier = jwt.NewVerifier(jwt.HS256, signatureSharedKey)
)

func main() {
	app := iris.New()

	blocklist := redis.NewBlocklist()
	verifier.Blocklist = blocklist
	verifyMiddleware := verifier.Verify(func() interface{} {
		return new(userClaims)
	})

	app.Get("/", loginView)

	api := app.Party("/api")
	{
		api.Post("/login", login)
		api.Post("/logout", verifyMiddleware, logout)

		todoAPI := api.Party("/todos", verifyMiddleware)
		{
			todoAPI.Post("/", createTodo)
			todoAPI.Get("/", listTodos)
			todoAPI.Get("/{id:uint64}", getTodo)
		}
	}

	protectedAPI := app.Party("/protected", verifyMiddleware)
	protectedAPI.Get("/", protected)
	protectedAPI.Get("/logout", logout)

	// GET  http://localhost:8080
	// POST http://localhost:8080/api/login
	// POST http://localhost:8080/api/logout
	// POST http://localhost:8080/api/todos
	// GET  http://localhost:8080/api/todos
	// GET  http://localhost:8080/api/todos/{id}
	app.Listen(":8080")
}

func authenticate(ctx iris.Context) {
	claims := userClaims{
		Username: "kataras",
	}

	// Generate JWT ID.
	random, err := uuid.NewRandom()
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}
	id := random.String()

	// Set the ID with the jwt.ID.
	token, err := signer.Sign(claims, jwt.ID(id))

	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.Write(token)
}

func protected(ctx iris.Context) {
	claims := jwt.Get(ctx).(*userClaims)

	// To the standard claims, e.g. the generated ID:
	// jwt.GetVerifiedToken(ctx).StandardClaims.ID

	ctx.WriteString(claims.Username)
}
