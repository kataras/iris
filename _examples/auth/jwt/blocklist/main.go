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

type userClaims struct {
	Username string `json:"username"`
}

func main() {
	app := iris.New()

	// IMPORTANT
	//
	// To use the in-memory blocklist just:
	// verifier.WithDefaultBlocklist()
	// To use a persistence blocklist, e.g. redis,
	// start your redis-server and:
	blocklist := redis.NewBlocklist()
	// To configure single client or a cluster one:
	// blocklist.ClientOptions.Addr = "127.0.0.1:6379"
	// blocklist.ClusterOptions.Addrs = []string{...}
	// To set a prefix for jwt ids:
	// blocklist.Prefix = "myapp-"
	//
	// To manually connect and check its error before continue:
	// err := blocklist.Connect()
	// By default the verifier will try to connect, if failed then it will throw http error.
	//
	// And then register it:
	verifier.Blocklist = blocklist
	verifyMiddleware := verifier.Verify(func() interface{} {
		return new(userClaims)
	})

	app.Get("/", authenticate)

	protectedAPI := app.Party("/protected", verifyMiddleware)
	protectedAPI.Get("/", protected)
	protectedAPI.Get("/logout", logout)

	// http://localhost:8080
	// http://localhost:8080/protected?token=$token
	// http://localhost:8080/logout?token=$token
	// http://localhost:8080/protected?token=$token (401)
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

func logout(ctx iris.Context) {
	ctx.Logout()

	ctx.Redirect("/", iris.StatusTemporaryRedirect)
}
