# Iris Examples

Runnable programs for [Iris](https://github.com/kataras/iris) v14, ordered the
way the [book](https://iris-go.com/book/) teaches: a tutorial that grows one
small service step by step, then one section per topic. Every example was
compiled against the framework source and exercised over HTTP before it was
committed; each `main.go` opens with a comment saying what it shows, what you
will be able to do afterwards, how to try it, and which example to read next.

Requirements: Go 1.27 or newer (the examples import the standard library's
`uuid` package). New to Iris? Start with `01-getting-started` and do the steps
in order.

## Running an example

The `_examples` directory starts with an underscore, so the Go toolchain
ignores it and no example ships its own `go.mod`. Copy the code into your
own module:

```sh
mkdir myapp && cd myapp
go mod init myapp
go get github.com/kataras/iris/v14@latest
# copy the example's main.go here, then:
go run .
```

Every example listens on `:8080` and prints the endpoints it registers.

## 01. Getting started: the Bookshelf tutorial

Thirteen programs that grow one service, the Bookshelf (shelves hold books; a
locked shelf freezes its books), from a single route to a tested,
production-shaped API on the `iris.NewBuilder` chain. Each step is a complete
`main.go` that adds one idea to the previous one, so you can diff two
neighbours to see exactly what changed. Do them in order; afterwards the other
sections stand alone as references. You should be comfortable with Go and with
HTTP methods, headers and status codes. No Iris experience is assumed.
[TUTORIAL.md](../TUTORIAL.md) splits the final step into packages.
Book chapters 1 to 3: Getting Started, Core Concepts, Building Applications with iris.NewBuilder.

| Step | Shows |
| --- | --- |
| [01-hello-world](01-getting-started/01-hello-world) | The smallest Iris program: `iris.New`, one route, one JSON response, `app.Listen` |
| [02-routes-and-params](01-getting-started/02-routes-and-params) | A router prefix, `{id:uuid}`, `ctx.Params`, `ctx.OK` and a 404 in the standard envelope |
| [03-json-and-validation](01-getting-started/03-json-and-validation) | `ctx.BindJSON[T]` running the payload's `Validate()`, `ctx.Created`, the exact 400 clients receive |
| [04-errors-mapped-once](01-getting-started/04-errors-mapped-once) | Sentinel errors mapped to status codes once with `errors.NewOptions`; handlers forward with `ctx.OK` and `ctx.NoContent` |
| [05-storage-behind-an-interface](01-getting-started/05-storage-behind-an-interface) | `Repository` interface, `MemStore`, a `Service` holding the rules; the request context reaches storage |
| [06-controllers](01-getting-started/06-controllers) | A controller: constructor + `Routing(r iris.Router)`, mounted through the container |
| [07-nested-routes-and-guards](01-getting-started/07-nested-routes-and-guards) | Books under shelves: a nested controller, a group guard, positional guards, `ctx.Values` caching |
| [08-pagination](01-getting-started/08-pagination) | `ctx.Paginated` with a typed filter, the `Page[T]` envelope, the list over GET and QUERY |
| [09-middleware](01-getting-started/09-middleware) | Writing a middleware and choosing `UseRouter`, `Use` or a positional guard |
| [10-admin-and-basicauth](01-getting-started/10-admin-and-basicauth) | An admin controller behind `basicauth.Default`; protection applied at the mount, not in the controller |
| [11-configuration](01-getting-started/11-configuration) | `iris.LoadConfiguration[T]` with an inlined `iris.BaseConfiguration`; secrets from the environment only |
| [12-newbuilder](01-getting-started/12-newbuilder) | The whole service on the `iris.NewBuilder` chain: `Services`, `Errors`, `Health`, `API`, `Deferrables`, `Init`/`Close` |
| [13-testing](01-getting-started/13-testing) | `httptest.New` against `newApp(cfg, repo)`: CRUD, pagination, guards, auth, and a one-method failing fake |

## 02. Routing

Book chapter 4: Routing & Controllers.

| Example | Shows |
| --- | --- |
| [routing-basics](02-routing/routing-basics) | GET/POST/PUT/DELETE with `ctx.BindJSON`, `ctx.List`, `ctx.Created`, `ctx.OK`, `ctx.NoContent` and `errors.NotFoundID` |
| [path-parameters](02-routing/path-parameters) | Typed macros (`{id:int}`, `{uid:uuid}`, `{addr:mail}`), safe accessor defaults, `else 400` constraints, `ctx.BindParam` |
| [groups](02-routing/groups) | `app.Router("/api")`, nesting, per-group middleware |
| [subdomains](02-routing/subdomains) | Subdomain routing and wildcard subdomains |
| [custom-macro](02-routing/custom-macro) | Registering your own path macro and parameter functions; `errors.DefaultPathParameterTypeErrorHandler` |
| [wrap-router](02-routing/wrap-router) | `WrapRouter` for cross-cutting rewrites before routing |

## 03. Controllers

Book chapter 4: Routing & Controllers; generic MVC is in chapter 24.

| Example | Shows |
| --- | --- |
| [basic](03-controllers/basic) | The `NewXxxAPI` constructor + `Routing(r iris.Router)` convention |
| [nested-guards](03-controllers/nested-guards) | Nested routers, group guards, positional guards, cached context values |
| [mvc-generics](03-controllers/mvc-generics) | Generic MVC controllers with generated CRUD routes |
| [repository](03-controllers/repository) | `mvc.UseRepository`: the six resource routes over a repository, with `Guard`, `Pagination` and `BeforeWrite` |

## 04. Reading requests

Book chapter 5: Reading Requests.

| Example | Shows |
| --- | --- |
| [query-and-forms](04-request/query-and-forms) | `ctx.Query()` typed getters, `Lookup`, `GetSlice`, `ReadForm` and its `errors.InvalidArgument` rejection |
| [read-json-validate](04-request/read-json-validate) | `ctx.BindJSON[T]` running the payload's own `Validate()`; the 400 wire format |
| [read-json-optional](04-request/read-json-optional) | `ctx.BindJSONOptional[T]` when the body may be absent |
| [raw-body](04-request/raw-body) | `ctx.ReadBody` into `*string`, `*[]byte` and structs by content type; `errors.HandleRequestError` for what it returns |
| [strict-query](04-request/strict-query) | `ctx.QueryDate/QueryInt/QueryBool` with required, default and zero-value options; `ctx.BindUpdate` and its unknown-column 400 |
| [pagination](04-request/pagination) | `ctx.Paginated` with a filter and the `Page[T]` envelope |
| [uploads](04-request/uploads) | `UploadFormFiles`, filename safety, `PostMaxMemory` |
| [body-limits](04-request/body-limits) | `BodyMaxMemory` app-wide and `bodylimit` per route (413) |

## 05. Writing responses

Book chapter 6: Response Handling.

| Example | Shows |
| --- | --- |
| [rest-helpers](05-response/rest-helpers) | `ctx.OK/Created/NoContent`, `ctx.Count` and the `ctx.Helper(value, err)` contract |
| [writers](05-response/writers) | JSON (with options), JSONP, XML, YAML, Markdown, Text, HTML, Binary |
| [content-negotiation](05-response/content-negotiation) | `ctx.Negotiation()` / `Negotiate` by `Accept` |
| [files](05-response/files) | `ServeFile`, `SendFile`, `ServeContent`, rate-limited downloads |
| [streaming-sse](05-response/streaming-sse) | Server-Sent Events with `sse.Open`: deferred headers, heartbeat, terminal frame, batching |
| [streaming-ai](05-response/streaming-ai) | One `ai.Open` handler serving both `stream: true` and a single JSON document |

## 06. Middleware

Book chapter 7: Middleware, and chapter 12: Built-in Middleware.

| Example | Shows |
| --- | --- |
| [writing-middleware](06-middleware/writing-middleware) | Writing your own; `UseRouter` vs `Use` vs positional scopes; guards that refuse through `rest/errors` |
| [conditional](06-middleware/conditional) | `iris.NewConditionalHandler` and filters |
| [error-middleware](06-middleware/error-middleware) | Middleware that returns `error` |
| [cors](06-middleware/cors) | Cross-origin requests and preflight |
| [rate-limit](06-middleware/rate-limit) | Token-bucket limits, per-client identity, `Retry-After` |
| [request-id](06-middleware/request-id) | Correlation ids, including the server-only generator |
| [accesslog](06-middleware/accesslog) | Structured request logs with custom fields |
| [cache](06-middleware/cache) | Client cache headers and server-side response caching |
| [i18n](06-middleware/i18n) | Locale files, language detection, plurals and variables |

## 07. Dependency injection

Book chapter 8: Dependency Injection.

| Example | Shows |
| --- | --- |
| [constructors](07-dependency-injection/constructors) | `Services(...)` with values, constructors and interface resolution |
| [handler-func](07-dependency-injection/handler-func) | Handlers whose arguments are injected |
| [request-scoped](07-dependency-injection/request-scoped) | Per-request dependencies from the live context |
| [initializer](07-dependency-injection/initializer) | `iris.Initializer` warm-up, auto-closers, `Deferrables` |

## 08. Error handling

Book chapter 9: Error Handling.

| Example | Shows |
| --- | --- |
| [central-error-map](08-errors/central-error-map) | Domain sentinels mapped once; handlers just forward errors |
| [handle-errorf](08-errors/handle-errorf) | `errors.HandleErrorf` adding call-site detail to a mapped error |
| [validation-errors](08-errors/validation-errors) | Field validation and the exact JSON clients receive |
| [error-router](08-errors/error-router) | `errors.NewRouter` with handlers that return `error` |

## 09. Authentication

Book chapter 10: Authentication & Authorization.

| Example | Shows |
| --- | --- |
| [basicauth](09-authentication/basicauth) | HTTP Basic Auth on an admin group |
| [api-key](09-authentication/api-key) | `middleware/apikey`: Bearer and custom-header keys, hashed keys, a database `Lookup` |
| [jwt-middleware](09-authentication/jwt-middleware) | Signing and verifying tokens, claims, revocation |
| [sessions](09-authentication/sessions) | Typed sessions, secure cookie defaults, `RegenerateID` after sign-in |
| [auth-sdk](09-authentication/auth-sdk) | The generic `auth.Auth[User]` SDK: signin, refresh, signout, JWKS |

## 10. Configuration

Book chapter 11: Configuration Management.

| Example | Shows |
| --- | --- |
| [configuration](10-configuration/configuration) | `iris.Options` merging, YAML files, `SERVER_CONFIG`, secrets from the environment |
| [configuration-worker](10-configuration/configuration-worker) | `iris.LoadConfiguration` and `iris.BaseConfiguration`, plus an `iris.Worker` the builder starts and stops |

## 11. Views

Book chapter 12: Built-in Middleware (the view engine).

| Example | Shows |
| --- | --- |
| [templates](11-views/templates) | HTML pages from embedded templates through a layout: `blocks.New`, `view.Handler`, `view.Render` and `view.WithTemplate` |

## 12. Database

Book chapter 13: Database Integration.

| Example | Shows |
| --- | --- |
| [postgres-mapping](12-database/postgres-mapping) | The `pgerrors` opt-in import: no-rows to 404, duplicates to 409, without opening a database |
| [pg-crud](12-database/pg-crud) | A real Postgres behind the seam with `kataras/pg`: schema from struct tags, a typed repository, `Init`/`Close` on the builder, `DATABASE_URL` |

## 13. WebSockets

Book chapter 12: Built-in Middleware (websockets).

| Example | Shows |
| --- | --- |
| [chat](13-websockets/chat) | A chat room on `neffos`: namespaces, connect and custom events, `BroadcastOthers`, the `neffos.js` browser client |

## 14. Testing

Book chapter 16: Testing. Both examples test the same tasks API, so you can read them side by side.

| Example | Shows |
| --- | --- |
| [iris-httptest](14-testing/iris-httptest) | The `httptest` package: fluent requests, JSON/header/cookie assertions, table-driven subtests, fakes through the repository seam |
| [std-httptest](14-testing/std-httptest) | The same suite with `net/http/httptest` only: `app.Build()` + `ServeHTTP`, `encoding/json`, and a real server on a loopback port |

## 15. Production

Book chapters 17 to 19: Monitoring & Observability, Security Best Practices, Deployment & DevOps.

| Example | Shows |
| --- | --- |
| [serving-and-tls](15-production/serving-and-tls) | `Listen` vs `Run(iris.ServerOptions{...})`, TLS files and Let's Encrypt |
| [graceful-shutdown](15-production/graceful-shutdown) | Draining requests and running closers on interrupt |
| [observability](15-production/observability) | Monitor dashboard, pprof and per-request cost behind auth |

## 16. Advanced

Book chapter 24: Advanced Topics; `ai-gateway` belongs to chapter 21: AI Backends.

| Example | Shows |
| --- | --- |
| [std-interop](16-advanced/std-interop) | `iris.FromStd` and serving the app as an `http.Handler` |
| [typed-context](16-advanced/typed-context) | Your own context type through `iris.NewContextWrapper` |
| [file-server](16-advanced/file-server) | Static files with dotfile protection |
| [multi-app](16-advanced/multi-app) | Two applications behind one host switcher |
| [api-graph](16-advanced/api-graph) | A visual graph of the registered routes |
| [sitemap](16-advanced/sitemap) | Generating `sitemap.xml` from your routes |
| [patches](16-advanced/patches) | Process-global customization, such as swapping the JSON writer |
| [ai-gateway](16-advanced/ai-gateway) | Relaying an upstream model stream with typed failure classification and failover |

## Where to go next

- [TUTORIAL.md](../TUTORIAL.md) builds the Bookshelf service of section 01 as a multi-package project.
- The [book](https://iris-go.com/book/) explains the concepts behind these examples; every section above names its chapter.
- The [Code Wiki](https://codewiki.google/github.com/kataras/iris) answers
  questions about the framework's source in plain language.
