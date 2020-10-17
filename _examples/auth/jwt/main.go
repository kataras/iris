// Package main shows how you can use the Iris unique JWT middleware.
// The file contains different kind of examples that all do the same job but,
// depending on your code style and your application's requirements, you may choose one over other.
package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

// Claims a custom claims structure.
type Claims struct {
	// Optionally define JWT's "iss" (Issuer),
	// "sub" (Subject) and "aud" (Audience) for issuer and subject.
	// The JWT's "exp" (expiration) and "iat" (issued at) are automatically
	// set by the middleware.
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience []string `json:"aud"`
	/*
		Note that the above fields can be also extracted via:
		  jwt.GetTokenInfo(ctx).Claims
		But in that example, we just showcase how these info can be embedded
		inside your own Go structure.
	*/

	// Optionally define a "exp" (Expiry),
	// unlike the rest, this is unset on creation
	// (unless you want to override the middleware's max age option),
	// it's filled automatically by the JWT middleware
	// when the request token is verified.
	// See the POST /user route.
	Expiry *jwt.NumericDate `json:"exp"`

	Username string `json:"username"`
}

func main() {
	// Get keys from system's environment variables
	// JWT_SECRET (for signing and verification) and JWT_SECRET_ENC(for encryption and decryption),
	// or defaults to "secret" and "itsa16bytesecret" respectfully.
	//
	// Use the `jwt.New` instead for more flexibility, if necessary.
	j := jwt.HMAC(15*time.Minute, "secret", "itsa16bytesecret")

	/*
		By default it extracts the token from url parameter "token={token}"
		and the Authorization Bearer {token} header.
		You can also take token from JSON body:
		j.Extractors = append(j.Extractors, jwt.FromJSON)
	*/

	/* Optionally, enable block list to force-invalidate
	verified tokens even before their expiration time.
	This is useful when the client doesn't clear
	the token on a user logout by itself.

	The duration argument clears any expired token on each every tick.
	There is a GC() method that can be manually called to clear expired blocked tokens
	from the memory.

	j.Blocklist = jwt.NewBlocklist(30*time.Minute)
	OR NewBlocklistContext(stdContext, 30*time.Minute)


	To invalidate a verified token just call:
	j.Invalidate(ctx) inside a route handler.
	*/

	app := iris.New()
	app.Logger().SetLevel("debug")

	app.OnErrorCode(iris.StatusUnauthorized, func(ctx iris.Context) {
		// Note that, any error stored by an authentication
		// method in Iris is an iris.ErrPrivate.
		// Available jwt errors:
		// - ErrMissing
		// - ErrMissingKey
		// - ErrExpired
		// - ErrNotValidYet
		// - ErrIssuedInTheFuture
		// - ErrBlocked
		// An iris.ErrPrivate SHOULD never be displayed to the client as it is;
		// because it may contain critical security information about the server.
		//
		// Also keep in mind that JWT middleware logs verification errors to the
		// application's logger ("debug") so, normally you don't have to
		// bother showing the verification error to the browser/client.
		// However, you can retrieve that error and do what ever you feel right:
		if err := ctx.GetErr(); err != nil {
			// If we have an error stored,
			// (JWT middleware stores any verification errors to the Context),
			// set the error as response body,
			// which is the default behavior if that
			// wasn't an authentication error (as explained above)
			ctx.WriteString(err.Error())
		} else {
			// Else, the default behavior when no error was occured;
			// write the status text of the status code:
			ctx.WriteString(iris.StatusText(iris.StatusUnauthorized))
		}
	})

	app.Get("/authenticate", func(ctx iris.Context) {
		claims := &Claims{
			Issuer:   "server",
			Audience: []string{"user"},
			Username: "kataras",
		}

		// WriteToken generates and sends the token to the client.
		// To generate a token use: tok, err := j.Token(claims)
		// then you can write it in any form you'd like.
		// The expiration JWT fields are automatically
		// set by the middleware, that means that your claims value
		// only needs to fill fields that your application specifically requires.
		j.WriteToken(ctx, claims)
	})

	// Middleware + type-safe method,
	// useful in 99% of the cases, when your application
	// requires token verification under a whole path prefix, e.g. /protected:
	protectedAPI := app.Party("/protected")
	{
		protectedAPI.Use(j.Verify(func() interface{} {
			// Must return a pointer to a type.
			//
			// The Iris JWT implementation is very sophisticated.
			// We keep our claims in type-safe form.
			// However, you are free to use raw Go maps
			// (map[string]interface{} or iris.Map) too (example later on).
			//
			// Note that you can use the same "j" JWT instance
			// to serve different types of claims on other group of routes,
			// e.g. postRouter.Use(j.Verify(... return new(Post))).
			return new(Claims)
		}))

		protectedAPI.Get("/", func(ctx iris.Context) {
			claims := jwt.Get(ctx).(*Claims)
			// All fields parsed from token are set to the claims,
			// including the Expiry (if defined).
			ctx.Writef("Username: %s\nExpires at: %s\nAudience: %s",
				claims.Username, claims.Expiry.Time(), claims.Audience)
		})
	}

	// Verify token inside a handler method,
	// useful when you just need to verify a token on a single spot:
	app.Get("/inline", func(ctx iris.Context) {
		var claims Claims
		_, err := j.VerifyToken(ctx, &claims)
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}

		ctx.Writef("Username: %s\nExpires at: %s\n",
			claims.Username, claims.Expiry.Time())
	})

	// Use a common map as claims method,
	// not recommended, as we support typed claims but
	// you can do it:
	app.Get("/map/authenticate", func(ctx iris.Context) {
		claims := map[string]interface{}{ // or iris.Map for shortcut.
			"username": "kataras",
		}

		j.WriteToken(ctx, claims)
	})

	app.Get("/map/verify/middleware", j.Verify(func() interface{} {
		return &iris.Map{} // or &map[string]interface{}{}
	}), func(ctx iris.Context) {
		claims := jwt.Get(ctx).(iris.Map)
		// The Get method will unwrap the *iris.Map for you,
		// so its values are directly accessible:
		ctx.Writef("Username: %s\nExpires at: %s\n",
			claims["username"], claims["exp"].(*jwt.NumericDate).Time())
	})

	app.Get("/map/verify", func(ctx iris.Context) {
		claims := make(iris.Map) // or make(map[string]interface{})

		tokenInfo, err := j.VerifyToken(ctx, &claims)
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}

		ctx.Writef("Username: %s\nExpires at: %s\n",
			claims["username"], tokenInfo.Claims.Expiry.Time()) /* the claims["exp"] is also set. */
	})

	// Use the new Context.User() to retrieve the verified client method:
	// 1. Create a go stuct that implements the context.User interface:
	app.Get("/users/authenticate", func(ctx iris.Context) {
		user := &User{Username: "kataras"}
		j.WriteToken(ctx, user)
	})
	usersAPI := app.Party("/users")
	{
		usersAPI.Use(j.Verify(func() interface{} {
			return new(User)
		}))

		usersAPI.Get("/", func(ctx iris.Context) {
			user := ctx.User()
			userToken, _ := user.GetToken()
			/*
				You can also cast it to the underline implementation
				and work with its fields:
				expires := user.(*User).Expiry.Time()
			*/
			// OR use the GetTokenInfo to get the parsed token information:
			expires := jwt.GetTokenInfo(ctx).Claims.Expiry.Time()
			lifetime := expires.Sub(time.Now()) // remeaning time to be expired.

			ctx.Writef("Username: %s\nAuthenticated at: %s\nLifetime: %s\nToken: %s\n",
				user.GetUsername(), user.GetAuthorizedAt(), lifetime, userToken)
		})
	}

	// http://localhost:8080/authenticate
	// http://localhost:8080/protected?token={token}
	// http://localhost:8080/inline?token={token}
	//
	// http://localhost:8080/map/authenticate
	// http://localhost:8080/map/verify?token={token}
	// http://localhost:8080/map/verify/middleware?token={token}
	//
	// http://localhost:8080/users/authenticate
	// http://localhost:8080/users?token={token}
	app.Listen(":8080")
}

// User is a custom implementation of the Iris Context User interface.
// Optionally, for JWT, you can also implement
// the SetToken(tok string) and
// Validate(ctx iris.Context, claims jwt.Claims, e jwt.Expected) error
// methods to set a token and add custom validation
// to a User value parsed from a token.
type User struct {
	iris.User
	Username string `json:"username"`

	// Optionally, declare some JWT fields,
	// they are automatically filled by the middleware itself.
	IssuedAt *jwt.NumericDate `json:"iat"`
	Expiry   *jwt.NumericDate `json:"exp"`
	Token    string           `json:"-"`
}

// GetUsername returns the Username.
// Look the iris/context.SimpleUser type
// for all the methods you can implement.
func (u *User) GetUsername() string {
	return u.Username
}

// GetAuthorizedAt returns the IssuedAt time.
// This and the Get/SetToken methods showcase how you can map JWT standard fields
// to an Iris Context User.
func (u *User) GetAuthorizedAt() time.Time {
	return u.IssuedAt.Time()
}

// GetToken is a User interface method.
func (u *User) GetToken() (string, error) {
	return u.Token, nil
}

// SetToken is a special jwt.TokenSetter interface which is
// called automatically when a token is parsed to this User value.
func (u *User) SetToken(tok string) {
	u.Token = tok
}

/*
func default_RSA_Example() {
	j := jwt.RSA(15*time.Minute)
}

Same as:

func load_File_Or_Generate_RSA_Example() {
	signKey, err := jwt.LoadRSA("jwt_sign.key", 2048)
	if err != nil {
		panic(err)
	}

	j, err := jwt.New(15*time.Minute, jwt.RS256, signKey)
	if err != nil {
		panic(err)
	}

	encKey, err := jwt.LoadRSA("jwt_enc.key", 2048)
	if err != nil {
		panic(err)
	}
	err = j.WithEncryption(jwt.A128CBCHS256, jwt.RSA15, encKey)
	if err != nil {
		panic(err)
	}
}
*/

/*
func hmac_Example() {
	// hmac
	key := []byte("secret")
	j, err := jwt.New(15*time.Minute, jwt.HS256, key)
	if err != nil {
		panic(err)
	}

	// OPTIONAL encryption:
	encryptionKey := []byte("itsa16bytesecret")
	err = j.WithEncryption(jwt.A128GCM, jwt.DIRECT, encryptionKey)
	if err != nil {
		panic(err)
	}
}
*/

/*
func load_From_File_With_Password_Example() {
	b, err := ioutil.ReadFile("./rsa_password_protected.key")
	if err != nil {
		panic(err)
	}
	signKey,err := jwt.ParseRSAPrivateKey(b, []byte("pass"))
	if err != nil {
		panic(err)
	}

	j, err := jwt.New(15*time.Minute, jwt.RS256, signKey)
	if err != nil {
		panic(err)
	}
}
*/

/*
func generate_RSA_Example() {
	signKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	encryptionKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	j, err := jwt.New(15*time.Minute, jwt.RS512, signKey)
	if err != nil {
		panic(err)
	}
	err = j.WithEncryption(jwt.A128CBCHS256, jwt.RSA15, encryptionKey)
	if err != nil {
		panic(err)
	}
}
*/
