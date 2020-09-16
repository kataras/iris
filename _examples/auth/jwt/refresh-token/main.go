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

// TokenPair holds the access token and refresh token response.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	app := iris.New()

	// Access token, short-live.
	accessJWT := jwt.HMAC(15*time.Minute, "secret", "itsa16bytesecret")
	// Refresh token, long-live. Important: Give different secret keys(!)
	refreshJWT := jwt.HMAC(1*time.Hour, "other secret", "other16bytesecre")
	// On refresh token, we extract it only from a request body
	// of JSON, e.g. {"refresh_token": $token }.
	// You can also do it manually in the handler level though.
	refreshJWT.Extractors = []jwt.TokenExtractor{
		jwt.FromJSON("refresh_token"),
	}

	// Generate access and refresh tokens and send to the client.
	app.Get("/authenticate", func(ctx iris.Context) {
		tokenPair, err := generateTokenPair(accessJWT, refreshJWT)
		if err != nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.JSON(tokenPair)
	})

	app.Get("/refresh", func(ctx iris.Context) {
		// Manual (if jwt.FromJSON missing):
		// var payload = struct {
		// 	RefreshToken string `json:"refresh_token"`
		// }{}
		//
		// err := ctx.ReadJSON(&payload)
		// if err != nil {
		// 	ctx.StatusCode(iris.StatusBadRequest)
		// 	return
		// }
		//
		// j.VerifyTokenString(ctx, payload.RefreshToken, &claims)

		var claims jwt.Claims
		if err := refreshJWT.VerifyToken(ctx, &claims); err != nil {
			ctx.Application().Logger().Warnf("verify refresh token: %v", err)
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		userID := claims.Subject
		if userID == "" {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		// Simulate a database call against our jwt subject.
		if userID != "53afcf05-38a3-43c3-82af-8bbbe0e4a149" {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		// All OK, re-generate the new pair and send to client.
		tokenPair, err := generateTokenPair(accessJWT, refreshJWT)
		if err != nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.JSON(tokenPair)
	})

	app.Get("/", func(ctx iris.Context) {
		var claims UserClaims
		if err := accessJWT.VerifyToken(ctx, &claims); err != nil {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		ctx.Writef("Username: %s\nExpires at: %s\n", claims.Username, claims.Expiry.Time())
	})

	// http://localhost:8080 (401)
	// http://localhost:8080/authenticate (200) (response JSON {access_token, refresh_token})
	// http://localhost:8080?token={access_token} (200)
	// http://localhost:8080?token={refresh_token} (401)
	// http://localhost:8080/refresh (request JSON{refresh_token = {refresh_token}}) (200) (response JSON {access_token, refresh_token})
	app.Listen(":8080")
}

func generateTokenPair(accessJWT, refreshJWT *jwt.JWT) (TokenPair, error) {
	standardClaims := jwt.Claims{Issuer: "an-issuer", Audience: jwt.Audience{"an-audience"}}

	customClaims := UserClaims{
		Claims:   accessJWT.Expiry(standardClaims),
		Username: "kataras",
	}

	accessToken, err := accessJWT.Token(customClaims)
	if err != nil {
		return TokenPair{}, err
	}

	// At refresh tokens you don't need any custom claims.
	refreshClaims := refreshJWT.Expiry(jwt.Claims{
		ID: "refresh_kataras",
		// For example, the User ID,
		// this is nessecary to check against the database
		// if the user still exist or has credentials to access our page.
		Subject: "53afcf05-38a3-43c3-82af-8bbbe0e4a149",
	})

	refreshToken, err := refreshJWT.Token(refreshClaims)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
