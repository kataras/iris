# Security Policy

## Supported Versions

We are focusing on fixing issues and security vulnerabilities to the latest and greatest version of Iris.
Iris users SHOULD always update their backend code to the latest version of Iris API, there is a complete [HISTORY](https://github.com/kataras/iris/blob/main/HISTORY.md) file
which acts as a code migration assistant.

## Reporting a Vulnerability

Please report (suspected) security vulnerabilities to
**[iris-go@outlook.com](mailto:iris-go@outlook.com)**. You will receive a response from
us within 3-4 working days. If the issue is confirmed, we will release a new minor version as soon
as possible depending on complexity but historically within a few hours or days.

## Defaults you should review

Some Iris defaults favour "it works out of the box" over "it is locked down".
They are kept for backwards compatibility, and each has an opt-in hardening.
Review this list when you put an application on the public internet.

| Default | What it means | Harden with |
| --- | --- | --- |
| `cors.New()` reflects any `Origin` and advertises `Access-Control-Allow-Credentials: true` | A page on any domain can read your authenticated responses with `fetch(..., {credentials: "include"})` | `cors.Strict()`, and `AllowOrigin`/`AllowOrigins` |
| `ContextOptions.BodyMaxMemory` is `0` (unlimited) | A client picks the size of the allocation your handler makes | Set `BodyMaxMemory`, or use the `bodylimit` middleware |
| `compress.Handler` decompresses without a ceiling | A ~1MB gzip body expands to ~1GB; a handful of requests exhausts memory | `compress.Safe`, or `compress.Limited(n)` |
| `cache.DefaultOptions` marks responses `public, max-age=3600` | A CDN or proxy may store a per-user response and serve it to someone else | `cache.PrivateOptions` (credentialed requests are already skipped) |
| `httpcache` keys do not include `Authorization` or cookies | Two users' identical requests collapse onto one cache entry | Set `Options.Headers` / `Options.Cookies` (credentialed requests are skipped until you do) |
| `accesslog` logs the request body | Passwords, tokens and card numbers land in your logs | Set `AccessLog.Redact`, or `RequestBody = false` |
| `methodoverride` accepts `_method` from the query string and form body | A cross-site form POST can reach a `DELETE`-only route | `methodoverride.New(methodoverride.Options{HeadersOnly: true})` |
| `fileserver` serves dotfiles | `.git/config` and `.env` are published if they sit under the served root | `Options.AssetValidator = fileserver.BlockHidden` |
| `pprof` and `monitor` have no built-in auth | Goroutine dumps and host metrics are readable by anyone who finds the route | Wrap them in `basicauth` or `ipaccess`, or bind them to an internal listener |
| `Context.Addr()` returns the client-supplied proxy header value | Not usable for access control | Use `Context.RemoteAddr()`, and only enable `RemoteAddrHeaders` behind a proxy that overwrites them |
| Sessions do not rotate the id on login | Session fixation | Call `Session.RegenerateID` after any privilege change |

Two more things worth knowing:

- **Proxy headers are opt-in and all-or-nothing.** `Options.RemoteAddrHeaders` is
  empty by default. Once you set it, every client can choose what those headers
  say, so only enable it when a proxy you control overwrites them on the way in.
- **MVC controllers are singletons.** A controller is resolved once and its
  methods are bound into handlers at registration. Fields are shared across all
  concurrent requests; keep per-request state on the Context.
