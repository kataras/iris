package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

// UserClaims a custom claims structure. You can just use jwt.Claims too.
type UserClaims struct {
	jwt.Claims
	Username string
}

func main() {
	// hmac
	key := []byte("secret")
	j, err := jwt.New(1*time.Minute, jwt.HS256, key)
	if err != nil {
		panic(err)
	}

	// OPTIONAL encryption:
	encryptionKey := []byte("itsa16bytesecret")
	err = j.WithEncryption(jwt.A128GCM, jwt.DIRECT, encryptionKey)
	if err != nil {
		panic(err)
	}

	app := iris.New()
	app.Logger().SetLevel("debug")

	app.Get("/authenticate", func(ctx iris.Context) {
		standardClaims := jwt.Claims{Issuer: "an-issuer", Audience: jwt.Audience{"an-audience"}}
		// NOTE: if custom claims then the `j.Expiry(claims)` (or jwt.Expiry(duration, claims))
		// MUST be called in order to set the expiration time.
		customClaims := UserClaims{
			Claims:   j.Expiry(standardClaims),
			Username: "kataras",
		}

		j.WriteToken(ctx, customClaims)
	})

	userRouter := app.Party("/user")
	{
		// userRouter.Use(j.Verify)
		// userRouter.Get("/", func(ctx iris.Context) {
		// 	var claims UserClaims
		// 	if err := jwt.ReadClaims(ctx, &claims); err != nil {
		//      // Validation-only errors, the rest are already
		//      // checked on `j.Verify` middleware.
		// 		ctx.StopWithStatus(iris.StatusUnauthorized)
		// 		return
		// 	}
		//
		// 	ctx.Writef("Claims: %#+v\n", claims)
		// })
		//
		// OR:
		userRouter.Get("/", func(ctx iris.Context) {
			var claims UserClaims
			if err := j.VerifyToken(ctx, &claims); err != nil {
				ctx.StopWithStatus(iris.StatusUnauthorized)
				return
			}

			ctx.Writef("Claims: %#+v\n", claims)
		})
	}

	app.Listen(":8080")
}

/*
func load_From_File_Example() {
	b, err := ioutil.ReadFile("./private_rsa.pem")
	if err != nil {
		panic(err)
	}
	signKey := jwt.MustParseRSAPrivateKey(b, []byte("pass"))

	j, err := jwt.New(15*time.Minute, jwt.RS256, signKey)
	if err != nil {
		panic(err)
	}
}
*/

/*
func random_RSA_Sign_And_Encrypt_Example() {
	j := jwt.Random(1 * time.Minute)
}
*/

/*
func random_manually_generate_RSA_Example() {
	signey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	encryptionKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	j, err := jwt.New(1*time.Minute, jwt.RS256, signey)
	if err != nil {
		panic(err)
	}
	err = j.WithEncryption(jwt.A128CBCHS256, jwt.RSA15, encryptionKey)
	if err != nil {
		panic(err)
	}
}
*/
