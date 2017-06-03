// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package basicauth provides http basic authentication via middleware. See _examples/beginner/basicauth
package basicauth

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type (
	encodedUser struct {
		HeaderValue string
		Username    string
		logged      bool
		expires     time.Time
	}
	encodedUsers []encodedUser

	basicAuthMiddleware struct {
		config Config
		// these are filled from the config.Users map at the startup
		auth             encodedUsers
		realmHeaderValue string
		expireEnabled    bool // if the config.Expires is a valid date, default disabled
	}
)

//

// New takes one parameter, the Config returns a Handler
// use: iris.Use(New(...)), iris.Get(...,New(...),...)
func New(c Config) context.Handler {
	config := DefaultConfig()
	if c.ContextKey != "" {
		config.ContextKey = c.ContextKey
	}
	if c.Realm != "" {
		config.Realm = c.Realm
	}
	config.Users = c.Users

	b := &basicAuthMiddleware{config: config}
	b.init()
	return b.Serve
}

// Default takes one parameter, the users returns a Handler
// use: iris.Use(Default(...)), iris.Get(...,Default(...),...)
func Default(users map[string]string) context.Handler {
	c := DefaultConfig()
	c.Users = users
	return New(c)
}

func (b *basicAuthMiddleware) init() {
	// pass the encoded users from the user's config's Users value
	b.auth = make(encodedUsers, 0, len(b.config.Users))

	for k, v := range b.config.Users {
		fullUser := k + ":" + v
		header := "Basic " + base64.StdEncoding.EncodeToString([]byte(fullUser))
		b.auth = append(b.auth, encodedUser{HeaderValue: header, Username: k, logged: false, expires: DefaultExpireTime})
	}

	// set the auth realm header's value
	b.realmHeaderValue = "Basic realm=" + strconv.Quote(b.config.Realm)

	if b.config.Expires > 0 {
		b.expireEnabled = true
	}
}

func (b *basicAuthMiddleware) findAuth(headerValue string) (auth *encodedUser, found bool) {
	if len(headerValue) == 0 {
		return
	}

	for _, user := range b.auth {
		if user.HeaderValue == headerValue {
			auth = &user
			found = true
			break
		}
	}

	return
}

func (b *basicAuthMiddleware) askForCredentials(ctx context.Context) {
	ctx.Header("WWW-Authenticate", b.realmHeaderValue)
	ctx.StatusCode(iris.StatusUnauthorized)
}

// Serve the actual middleware
func (b *basicAuthMiddleware) Serve(ctx context.Context) {

	if auth, found := b.findAuth(ctx.GetHeader("Authorization")); !found {
		b.askForCredentials(ctx)
		// don't continue to the next handler
	} else {
		// all ok set the context's value in order to be getable from the next handler
		ctx.Values().Set(b.config.ContextKey, auth.Username)
		if b.expireEnabled {

			if auth.logged == false {
				auth.expires = time.Now().Add(b.config.Expires)
				auth.logged = true
			}

			if time.Now().After(auth.expires) {
				b.askForCredentials(ctx) // ask for authentication again
				return
			}

		}
		ctx.Next() // continue
	}

}
