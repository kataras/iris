// Global patches: replacing framework defaults through iris.Patches().
//
// CAUTION: patches are process-global. A patch replaces a package-level
// variable, so it applies to every Iris application in the binary, including
// the ones your tests spin up, and the last writer wins. Apply them once, from
// main, before serving; never from a library or from the init of an imported
// package.
//
// The patch points:
//
//	Patches().Context().Writers().JSON/JSONP/XML/Markdown/YAML(fn)
//	Patches().Context().GetDomain(fn)                 // Context.Domain derivation.
//	Patches().Context().SetDefaultCookieExpiration(d) // Context.AddCookie default.
//	Patches().Context().ResolveHTTPFS(fn)             // file server filesystem.
//	Patches().Context().ResolveFS(fn)                 // view engine filesystem.
//
// Learn: change a framework default once for the whole process, such as how every ctx.JSON call writes or how long a cookie lives.
// Before: ../sitemap
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/           # 200, indented JSON, HTML left unescaped
//	curl -i http://localhost:8080/session # 200, Set-Cookie expires one week out, not one year
//
// Next: ../ai-gateway
package main

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"time"

	"github.com/kataras/iris/v14"
)

func main() {
	// Replaces the writer behind every ctx.JSON call. This is the signature
	// Iris calls internally, after it has set the content type, so the patch
	// only writes the body. Iris already encodes through the standard
	// library's encoding/json/v2; the third-party encoders this seam used to
	// host (jsoniter, goccy/go-json, sonic) existed to outrun encoding/json
	// v1 and have lost that reason. The seam remains for POLICY changes; this
	// one indents every response, even when the caller asked for none.
	iris.Patches().Context().Writers().JSON(
		func(ctx iris.Context, v any, options *iris.JSON) error {
			indent := options.Indent
			if indent == "" {
				indent = "  "
			}

			return json.MarshalWrite(ctx.ResponseWriter(), v,
				jsontext.Multiline(true), jsontext.WithIndent(indent))
		})

	// Changes the expiration ctx.AddCookie applies when the call carries no
	// CookieExpires option: one week here instead of the default 365 days.
	iris.Patches().Context().SetDefaultCookieExpiration(7 * 24 * time.Hour)

	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"encoder": "encoding/json, patched",
			"docs":    `<a href="https://iris-go.com">iris-go.com</a>`,
		})
	})

	app.Get("/session", func(ctx iris.Context) {
		// No iris.CookieExpires option, so the patched default applies.
		ctx.AddCookie("session_id", "abc123")
		ctx.WriteString("cookie set\n")
	})

	app.Listen(":8080")
}
