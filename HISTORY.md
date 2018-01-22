# History/Changelog <a href="HISTORY_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

### Looking for free and real-time support?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### Looking for previous versions?

    https://github.com/kataras/iris/releases

### Should I upgrade my Iris?

Developers are not forced to upgrade if they don't really need it. Upgrade whenever you feel ready.

> Iris uses the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature, so you get truly reproducible builds, as this method guards against upstream renames and deletes.

**How to upgrade**: Open your command-line and execute this command: `go get -u github.com/kataras/iris` or let the automatic updater do that for you.

# Tu, 16 January 2018 | v10.0.2

## Security | `iris.AutoTLS`

**Every server should be upgraded to this version**, it contains fixes for the _tls-sni challenge disabled_ some days ago by letsencrypt.org which caused almost every https-enabled golang server to be unable to be functional, therefore support for the _http-01 challenge type_ added. Now the server is testing all available letsencrypt challenges.

Read more at:

- https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241
- https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac

# Mo, 15 January 2018 | v10.0.1

Not any serious problems were found to be resolved here but one, the first one which is important for devs that used the [cache](cache) package.

- fix a single one cache handler didn't work across multiple route handlers at the same time https://github.com/kataras/iris/pull/852, as reported at https://github.com/kataras/iris/issues/850
- merge PR https://github.com/kataras/iris/pull/862
- do not allow concurrent access to the `ExecuteWriter -> Load` when `view#Engine##Reload` was true, as requested at https://github.com/kataras/iris/issues/872
- badge for open-source projects powered by Iris, learn how to add that badge to your open-source project at [FAQ.md](FAQ.md) file
- upstream update for `golang/crypto` to apply the fix about the [tls-sni challenge disabled](https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241) https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac (**relative to iris.AutoTLS**)

## New Backers

1. https://opencollective.com/cetin-basoz

## New Translations

1. The Chinese README_ZH.md and HISTORY_ZH.md was translated by @Zeno-Code via https://github.com/kataras/iris/pull/858
2. New Russian README_RU.md translations by @merrydii via https://github.com/kataras/iris/pull/857
3. New Greek README_GR.md and HISTORY_GR.md translations via https://github.com/kataras/iris/commit/8c4e17c2a5433c36c148a51a945c4dc35fbe502a#diff-74b06c740d860f847e7b577ad58ddde0 and https://github.com/kataras/iris/commit/bb5a81c540b34eaf5c6c8e993f644a0e66a78fb8

## New Examples

1. [MVC - Register Middleware](_examples/mvc/middleware)

## New Articles

1. [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
2. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)

# Mo, 01 January 2018 | v10.0.0

We must thanks [Mrs. Diana](https://www.instagram.com/merry.dii/) for our awesome new [logo](https://iris-go.com/images/icon.svg)!

You can [contact](mailto:Kovalenkodiana8@gmail.com) her for any  design-related enquiries or explore and send a direct message via [instagram](https://www.instagram.com/merry.dii/).

<p align="center">
<img width="145px" src="https://iris-go.com/images/icon.svg?v=a" />
</p>

At this version we have many internal improvements but just two major changes and one big feature, called **hero**.

> The new version adds 75 plus new commits, the PR is located [here](https://github.com/kataras/iris/pull/849) read the internal changes if you are developing a web framework based on Iris. Why 9 was skipped? Because.

## Hero

The new package [hero](hero) contains features for binding any object or function that `handlers` may use, these are called dependencies. Hero funcs can also return any type of values, these values will be dispatched to the client.

> You may saw binding before but you didn't have code editor's support, with Iris you get truly safe binding thanks to the new `hero` package. It's also fast, near to raw handlers performance because Iris calculates everything before server ran!

Below you will see some screenshots we prepared for you in order to be easier to understand:

### 1. Path Parameters - Built'n Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-1-monokai.png)

### 2. Services - Static Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-2-monokai.png)

### 3. Per-Request - Dynamic Dependencies

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-3-monokai.png)

`hero funcs` are very easy to understand and when you start using them **you never go back**.

Examples:

- [Basic](_examples/hero/basic/main.go)
- [Overview](_examples/hero/overview)

## MVC

You have to understand the `hero` package in order to use the `mvc`, because `mvc` uses the `hero` internally for the controller's methods you use as routes, the same rules applied to those controller's methods of yours as well.

With this version you can register **any controller's methods as routes manually**, you can **get a route based on a method name and change its `Name` (useful for reverse routing inside templates)**, you can use any **dependencies** registered from `hero.Register` or `mvc.New(iris.Party).Register` per mvc application or per-controller, **you can still use `BeginRequest` and `EndRequest`**, you can catch **`BeforeActivation(b mvc.BeforeActivation)` to add dependencies per controller and `AfterActivation(a mvc.AfterActivation)` to make any post-validations**, **singleton controllers when no dynamic dependencies are used**, **Websocket controller, as simple as a `websocket.Connection` dependency** and more...

Examples:

**If you used MVC before then read very carefully: MVC CONTAINS SOME BREAKING CHANGES BUT YOU CAN DO A LOT MORE AND EVEN FASTER THAN BEFORE**

**PLEASE READ THE EXAMPLES CAREFULLY, WE'VE MADE THEM FOR YOU**

Old examples are here as well. Compare the two different versions of each example to understand what you win if you upgrade now.

| NEW | OLD |
| -----------|-------------|
| [Hello world](_examples/mvc/hello-world/main.go) | [OLD Hello world](https://github.com/kataras/iris/blob/v8/_examples/mvc/hello-world/main.go) |
| [Session Controller](_examples/mvc/session-controller/main.go) | [OLD Session Controller](https://github.com/kataras/iris/blob/v8/_examples/mvc/session-controller/main.go) |
| [Overview - Plus Repository and Service layers](_examples/mvc/overview) | [OLD Overview - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/overview) |
| [Login showcase - Plus Repository and Service layers](_examples/mvc/login) | [OLD Login showcase - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/login) |
| [Singleton](_examples/mvc/singleton) |  **NEW** |
| [Websocket Controller](_examples/mvc/websocket) |  **NEW** |
| [Vue.js Todo MVC](_examples/tutorial/vuejs-todo-mvc) |  **NEW** |

## context#PostMaxMemory

Remove the old static variable `context.DefaultMaxMemory` and replace it with the configuration `WithPostMaxMemory`.

```go
// WithPostMaxMemory sets the maximum post data size
// that a client can send to the server, this differs
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
func WithPostMaxMemory(limit int64) Configurator
```

If you used that old static field you will have to change that single line.

Usage:

```go
import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // [...]

    app.Run(iris.Addr(":8080"), iris.WithPostMaxMemory(10 << 20))
}
```

## context#UploadFormFiles

New method to upload multiple files, should be used for common upload actions, it's just a helper function.

```go
// UploadFormFiles uploads any received file(s) from the client
// to the system physical location "destDirectory".
//
// The second optional argument "before" gives caller the chance to
// modify the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs, the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by the
//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// See `FormFile` to a more controlled to receive a file.
func (ctx *context) UploadFormFiles(
        destDirectory string,
        before ...func(string, string),
    ) (int64, error)
```

Example can be found [here](_examples/http_request/upload-files/main.go).

## context#View

Just a minor addition, add a second optional variadic argument to the `context#View` method to accept a single value for template binding.
When you just want one value and not key-value pairs, you used to use an empty string on the `ViewData`, which is fine, especially if you preload these from a previous handler/middleware in the request handlers chain.

```go
func(ctx iris.Context) {
    ctx.ViewData("", myItem{Name: "iris" })
    ctx.View("item.html")
}
```

Same as:

```go
func(ctx iris.Context) {
    ctx.View("item.html", myItem{Name: "iris" })
}
```

```html
Item's name: {{.Name}}
```

## context#YAML

Add a new `context#YAML` function, it renders a yaml from a structured value.

```go
// YAML marshals the "v" using the yaml marshaler and renders its result to the client.
func YAML(v interface{}) (int, error)
```

## Session#GetString

`sessions/session#GetString` can now return a filled value even if the stored value is a type of integer, just like the memstore, the context's temp store, the context's path parameters and the context's url parameters.