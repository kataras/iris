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
| `httpcache` keys do not include `Authorization` or cookies | Two users' identical requests collapse onto one cache entry | Name the header in `Options.Headers` and every cookie the request sends in `Options.Cookies`: a request with credentials is skipped until the key includes each one of them. Register the cache handler before `sessions.Handler`, which otherwise writes its cookie after the page was stored. A response whose `Cache-Control` carries `private`, `no-store` or `no-cache` is never stored, so send `private` from a per-user handler |
| `accesslog` logs the request body | Passwords, tokens and card numbers land in your logs | Set `AccessLog.Redact`, or `RequestBody = false` |
| `methodoverride` accepts `_method` from the query string and form body | A cross-site form POST can reach a `DELETE`-only route | `methodoverride.New(methodoverride.Options{HeadersOnly: true})` |
| `fileserver` serves dotfiles | `.git/config` and `.env` are published if they sit under the served root | `Options.AssetValidator = fileserver.BlockHidden` |
| `pprof` and `monitor` have no built-in auth | Goroutine dumps and host metrics are readable by anyone who finds the route | Wrap them in `basicauth` or `ipaccess`, or bind them to an internal listener |
| `Context.Addr()` returns the client-supplied proxy header value | Not usable for access control | Use `Context.RemoteAddr()`, as `ipaccess` does, and only enable `RemoteAddrHeaders` behind a proxy that overwrites them |
| Sessions do not rotate the id on login | Session fixation | Call `Session.RegenerateID` after any privilege change |

Two more things worth knowing:

- **Proxy headers are opt-in and all-or-nothing.** `Options.RemoteAddrHeaders` is
  empty by default, and `Context.RemoteAddr()` is then the address of the
  connection. Once you set it, every client can send those headers, so only
  enable it behind a proxy you control. A proxy appends the address it got the
  request from, so `RemoteAddr()` reads each header from right to left: the
  first public address is the result, without its port, and a private address
  is skipped as one of your own proxies. An entry that is not an address
  (`unknown`, `unix:`) ends the reading of that header with no result. The next
  header is read then, and at last the address of the connection is used. The
  `ipaccess` middleware decides on `RemoteAddr()`, so what follows holds for
  its allowlist and its blocklist too. What the client wrote is on the left,
  and it can still be the result in three cases. The first is a header in
  which every entry to its right is a private address, as for a request that
  reached your proxy from inside your network. The second is
  `RemoteAddrHeadersForce`, which returns the first address from the left of a
  header whose entries are all addresses and none of them public. The third is
  a header your proxy does not write: the proxy passes it on as the client
  sent it, and it is read when it is named first in `RemoteAddrHeaders`, or
  when the headers named before it give no public address. A proxy that
  overwrites the header closes the first two, and naming only headers your
  proxy writes closes the third. `Context.Addr()` needs the overwriting proxy
  in any case, because it reads from the left. The result is not always the
  client either. A proxy with a public address in front of your own, a CDN for
  example, is the first public address from the right, so `RemoteAddr()`
  returns the CDN and not the client it got the request from, until you add
  its IPv4 range to `RemoteAddrPrivateSubnets`.
- **MVC controllers are singletons.** A controller is resolved once and its
  methods are bound into handlers at registration. Fields are shared across all
  concurrent requests; keep per-request state on the Context.
