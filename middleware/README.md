# Middleware

We should mention that Iris is compatible with **ALL** net/http middleware out there,
You are not restricted to so-called 'iris-made' middleware. They do exists, mostly, for your learning curve.

Navigate through [iris-contrib/middleware](https://github.com/iris-contrib/through) repository to view iris-made 'middleware'.

>  By the word 'middleware', we mean a single or a collection of route handlers which may execute before/or after the main route handler.


## Installation

```sh
$ go get github.com/iris-contrib/middleware/...
```

## How can I register a middleware?

```go
app := iris.New()
/* per root path and all its children */
app.Use(logger)

/* execute always last */
// app.Done(logger)

/* per-route, order matters. */
// app.Get("/", logger, indexHandler)

/* per party (group of routes) */
// userRoutes := app.Party("/user", logger)
// userRoutes.Post("/login", loginAuthHandler)
```

## How 'hard' is to create an Iris middleware?

```go
myMiddleware := func(ctx *iris.Context){
  /* using ctx.Set you can transfer ANY data between handlers,
     use ctx.Get("welcomed") to get its value on the next handler(s).
  */
  ctx.Set("welcomed", true)

  println("My middleware!")
}
```
> func(ctx *iris.Context) is just the `iris.HandlerFunc` signature which implements the `iris.Handler`/ `Serve(Context)` method.

```go
app := iris.New()
/* root path and all its children */
app.UseFunc(myMiddleware)

app.Get("/", indexHandler)
```

## Convert `http.Handler` to `iris.Handler` using the `iris.ToHandler`

```go
app := iris.New()

sillyHTTPHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
     println(r.RequestURI)
})

app.Use(iris.ToHandler(sillyHTTPHandler))
```


## What next?

Read more about [iris.Handler](https://docs.iris-go.com/using-handlers.html), [iris.HandlerFunc](https://docs.iris-go.com/using-handlerfuncs.html) and [Middleware](https://docs.iris-go.com/middleware.html).
