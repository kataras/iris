package main

import (
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/cache"
	"github.com/kataras/iris/context"
)

var markdownContents = []byte(`## Hello Markdown

This is a sample of Markdown contents

 

Features
--------

All features of Sundown are supported, including:

*   **Compatibility**. The Markdown v1.0.3 test suite passes with
    the --tidy option.  Without --tidy, the differences are
    mostly in whitespace and entity escaping, where blackfriday is
    more consistent and cleaner.

*   **Common extensions**, including table support, fenced code
    blocks, autolinks, strikethroughs, non-strict emphasis, etc.

*   **Safety**. Blackfriday is paranoid when parsing, making it safe
    to feed untrusted user input without fear of bad things
    happening. The test suite stress tests this and there are no
    known inputs that make it crash.  If you find one, please let me
    know and send me the input that does it.

    NOTE: "safety" in this context means *runtime safety only*. In order to
    protect yourself against JavaScript injection in untrusted content, see
    [this example](https://github.com/russross/blackfriday#sanitize-untrusted-content).

*   **Fast processing**. It is fast enough to render on-demand in
    most web applications without having to cache the output.

*   **Routine safety**. You can run multiple parsers in different
    goroutines without ill effect. There is no dependence on global
    shared state.

*   **Minimal dependencies**. Blackfriday only depends on standard
    library packages in Go. The source code is pretty
    self-contained, so it is easy to add to any project, including
    Google App Engine projects.

*   **Standards compliant**. Output successfully validates using the
    W3C validation tool for HTML 4.01 and XHTML 1.0 Transitional.

	[this is a link](https://github.com/kataras/iris) `)

// Cache should not be used on handlers that contain dynamic data.
// Cache is a good and a must-feature on static content, i.e "about page" or for a whole blog site.
func main() {
	app := iris.New()

	// first argument is the handler which response's we want to apply the cache.
	// if second argument, expiration, is <=time.Second then the cache tries to set the expiration from the "cache-control" maxage header's value(in seconds)
	// and if that header is empty or not exist then it sets a default of 5 minutes.
	writeMarkdownCached := cache.Cache(writeMarkdown, 10*time.Second) // or CacheHandler to get the handler

	app.Get("/", writeMarkdownCached.ServeHTTP)
	// saves its content on the first request and serves it instead of re-calculating the content.
	// After 10 seconds it will be cleared and resetted.

	app.Run(iris.Addr(":8080"))
}

func writeMarkdown(ctx context.Context) {
	// tap multiple times the browser's refresh button and you will
	// see this println only once every 10 seconds.
	println("Handler executed. Content refreshed.")

	ctx.Markdown(markdownContents)
}

// Notes:
// Cached handler is not changing your pre-defined headers,
// so it will not send any additional headers to the client.
// The cache happening at the server-side's memory.
//
// see "DefaultRuleSet" in "cache/client/ruleset.go" to see how you can add separated
// rules to each of the cached handlers (`.AddRule`) with the help of "/cache/client/rule"'s definitions.
//
// The default rules are:
/*
	    // #1 A shared cache MUST NOT use a cached response to a request with an
		// Authorization header field
		rule.HeaderClaim(ruleset.AuthorizationRule),
		// #2 "must-revalidate" and/or
		// "s-maxage" response directives are not allowed to be served stale
		// (Section 4.2.4) by shared caches.  In particular, a response with
		// either "max-age=0, must-revalidate" or "s-maxage=0" cannot be used to
		// satisfy a subsequent request without revalidating it on the origin
		// server.
		rule.HeaderClaim(ruleset.MustRevalidateRule),
		rule.HeaderClaim(ruleset.ZeroMaxAgeRule),
		// #3 custom No-Cache header used inside this library
		// for BOTH request and response (after get-cache action)
		rule.Header(ruleset.NoCacheRule, ruleset.NoCacheRule)
*/
