# History

## 2.2.4 -> 2.3.0

**Changed**

- `&iris.RenderConfig{}` -> `&render.Config{}` from package github.com/kataras/iris/render but you don't need to import it, just do `iris.Config().Render.Directory = "mytemplates"` for example
- `iris.Config().Render typeof *iris.RenderConfig` -> `iris.Config().Render typeof *render.Config`
- `iris.HTMLOptions{Layout: "your_overrided_layout"}` -> now passed just as string `"your_overrided_layout"`
- `iris.Delims` -> `render.Delims` from package github.com/kataras/iris/render, but you don't need to import it, just do `iris.Config().Render.Delims.Left = "${"; iris.Config().Render.Delims.Right = "}"` for example


**Added**

- `iris.Render()` : returns the Template Engine, you can access the root `*template.Template` via `iris.Render().Templates`
- `iris.Config().Session` = :
```go
&iris.SessionConfig{
			Provider: "memory", // the default provider is "memory", if you set it to ""  means that sessions are disabled.
			Secret:   DefaultCookieName,
			Life:     DefaultCookieDuration,
}

// example:  iris.Config().Session.Secret = "mysessionsecretcookieadmin!123"
// iris.Config().Session.Provider = "redis"
```

- `context.Session()` : returns the current session for this user which is `github/kataras/iris/sessions/store/store.go/IStore`. So you have:

		- `context.Session().Get("key")` ,`context.Session().Set("key","value")`, Delete.. `and all these which IStore contains`


- `context.SessionDestroy()` : destroys the whole session, the provider's values and the client's cookie (same as sessions.Destroy(context))

-----------
