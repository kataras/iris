# Sessions

Iris provides a fast, fully featured and easy to use sessions manager.

Iris sessions manager lives on its own [kataras/iris/sessions](https://github.com/kataras/iris/tree/master/sessions) package.

Some trivial examples,

- [Overview](https://github.com/kataras/iris/blob/master/_examples/sessions/overview/main.go)
- [Standalone](https://github.com/kataras/iris/blob/master/_examples/sessions/standalone/main.go)
- [Secure Cookie](https://github.com/kataras/iris/blob/master/_examples/sessions/securecookie/main.go)
- [Flash Messages](https://github.com/kataras/iris/blob/master/_examples/sessions/flash-messages/main.go)
- [Databases](https://github.com/kataras/iris/tree/master/_examples/sessions/database)
    * [Badger](https://github.com/kataras/iris/blob/master/_examples/sessions/database/badger/main.go) **fastest**
    * [BoltDB](https://github.com/kataras/iris/blob/master/_examples/sessions/database/boltdb/main.go)
    * [Redis](https://github.com/kataras/iris/blob/master/_examples/sessions/database/redis/main.go)

## Overview

```go
import "github.com/kataras/iris/sessions"

manager := sessions.Start(iris.Context)
manager.
  ID() string
  Get(string) interface{}
  HasFlash() bool
  GetFlash(string) interface{}
  GetFlashString(string) string
  GetString(key string) string
  GetInt(key string) (int, error)
  GetInt64(key string) (int64, error)
  GetFloat32(key string) (float32, error)
  GetFloat64(key string) (float64, error)
  GetBoolean(key string) (bool, error)
  GetAll() map[string]interface{}
  GetFlashes() map[string]interface{}
  VisitAll(cb func(k string, v interface{}))
  Set(string, interface{})
  SetImmutable(key string, value interface{})
  SetFlash(string, interface{})
  Delete(string)
  Clear()
  ClearFlashes()
```

This example will show how to store data from a session.

You don't need any third-party library except Iris, but if you want you can use anything, remember Iris is fully compatible with the standard library. You can find a more detailed examples by pressing [here](https://github.com/kataras/iris/tree/master/_examples/sessions).

In this example we will only allow authenticated users to view our secret message on the `/secret` age. To get access to it, the will first have to visit `/login` to get a valid session cookie, hich logs him in. Additionally he can visit `/logout` to revoke his access to our secret message.

```go
// sessions.go
package main

import (
    "github.com/kataras/iris"

    "github.com/kataras/iris/sessions"
)

var (
    cookieNameForSessionID = "mycookiesessionnameid"
    sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID, AllowReclaim: true})
)

func secret(ctx iris.Context) {
    // Check if user is authenticated
    if auth, _ := sess.Start(ctx).GetBoolean("authenticated"); !auth {
        ctx.StatusCode(iris.StatusForbidden)
        return
    }

    // Print secret message
    ctx.WriteString("The cake is a lie!")
}

func login(ctx iris.Context) {
    session := sess.Start(ctx)

    // Authentication goes here
    // ...

    // Set user as authenticated
    session.Set("authenticated", true)
}

func logout(ctx iris.Context) {
    session := sess.Start(ctx)

    // Revoke users authentication
    session.Set("authenticated", false)
}

func main() {
    app := iris.New()

    app.Get("/secret", secret)
    app.Get("/login", login)
    app.Get("/logout", logout)

    app.Run(iris.Addr(":8080"))
}

```

```bash
$ go run sessions.go

$ curl -s http://localhost:8080/secret
Forbidden

$ curl -s -I http://localhost:8080/login
Set-Cookie: mysessionid=MTQ4NzE5Mz...

$ curl -s --cookie "mysessionid=MTQ4NzE5Mz..." http://localhost:8080/secret
The cake is a lie!
```