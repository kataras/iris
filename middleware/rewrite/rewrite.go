package rewrite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12/context"

	"gopkg.in/yaml.v3"
)

// Options holds the developer input to customize
// the redirects for the Rewrite Engine.
// Look the `New` package-level function.
type Options struct {
	// RedirectMatch accepts a slice of lines
	// of form:
	// REDIRECT_CODE PATH_PATTERN TARGET_PATH
	// Example: []{"301 /seo/(.*) /$1"}.
	RedirectMatch []string `json:"redirectMatch" yaml:"RedirectMatch"`
}

// Engine is the rewrite engine master structure.
// Navigate through _examples/routing/rewrite for more.
type Engine struct {
	redirects []*redirectMatch
}

// New returns a new Rewrite Engine based on "opts".
// It reports any parser error.
// See its `Handler` or `Wrapper` methods. Depending
// on the needs, select one.
func New(opts Options) (*Engine, error) {
	redirects := make([]*redirectMatch, 0, len(opts.RedirectMatch))

	for _, line := range opts.RedirectMatch {
		r, err := parseRedirectMatchLine(line)
		if err != nil {
			return nil, err
		}
		redirects = append(redirects, r)
	}

	e := &Engine{
		redirects: redirects,
	}
	return e, nil
}

// Handler returns a new rewrite Iris Handler.
// It panics on any error.
// Same as engine, _ := New(opts); engine.Handler.
// Usage:
// app.UseRouter(Handler(opts)).
func Handler(opts Options) context.Handler {
	engine, err := New(opts)
	if err != nil {
		panic(err)
	}
	return engine.Handler
}

// Handler is an Iris Handler that can be used as a router or party or route middleware.
// For a global alternative, if you want to wrap the entire Iris Application
// use the `Wrapper` instead.
// Usage:
// app.UseRouter(engine.Handler)
func (e *Engine) Handler(ctx *context.Context) {
	// We could also do that:
	// but we don't.
	// 	e.WrapRouter(ctx.ResponseWriter(), ctx.Request(), func(http.ResponseWriter, *http.Request) {
	// 		ctx.Next()
	// 	})
	for _, rd := range e.redirects {
		src := ctx.Path()
		if !rd.isRelativePattern {
			src = ctx.Request().URL.String()
		}

		if target, ok := rd.matchAndReplace(src); ok {
			if target == src {
				// this should never happen: StatusTooManyRequests.
				// keep the router flow.
				ctx.Next()
				return
			}

			ctx.Redirect(target, rd.code)
			return
		}
	}

	ctx.Next()
}

// Wrapper wraps the entire Iris Router.
// Wrapper is a bit faster than Handler because it's executed
// even before any route matched and it stops on redirect pattern match.
// Use it to wrap the entire Iris Application, otherwise look `Handler` instead.
//
// Usage:
// app.WrapRouter(engine.Wrapper).
func (e *Engine) Wrapper(w http.ResponseWriter, r *http.Request, routeHandler http.HandlerFunc) {
	for _, rd := range e.redirects {
		src := r.URL.Path
		if !rd.isRelativePattern {
			src = r.URL.String()
		}

		if target, ok := rd.matchAndReplace(src); ok {
			if target == src {
				routeHandler(w, r)
				return
			}

			http.Redirect(w, r, target, rd.code)
			return
		}
	}

	routeHandler(w, r)
}

type redirectMatch struct {
	code    int
	pattern *regexp.Regexp
	target  string

	isRelativePattern bool
}

func (r *redirectMatch) matchAndReplace(src string) (string, bool) {
	if r.pattern.MatchString(src) {
		if match := r.pattern.ReplaceAllString(src, r.target); match != "" {
			return match, true
		}
	}

	return "", false
}

func parseRedirectMatchLine(s string) (*redirectMatch, error) {
	parts := strings.Split(strings.TrimSpace(s), " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("redirect match: invalid line: %s", s)
	}

	codeStr, pattern, target := parts[0], parts[1], parts[2]

	for i, ch := range codeStr {
		if !isDigit(ch) {
			return nil, fmt.Errorf("redirect match: status code digits: %s [%d:%c]", codeStr, i, ch)
		}
	}

	code, err := strconv.Atoi(codeStr)
	if err != nil {
		// this should not happen, we check abt digit
		// and correctly position the error too but handle it.
		return nil, fmt.Errorf("redirect match: status code digits: %s: %v", codeStr, err)
	}

	if code <= 0 {
		code = http.StatusMovedPermanently
	}

	regex := regexp.MustCompile(pattern)
	if regex.MatchString(target) {
		return nil, fmt.Errorf("redirect match: loop detected: pattern: %s vs target: %s", pattern, target)
	}

	v := &redirectMatch{
		code:    code,
		pattern: regex,
		target:  target,

		isRelativePattern: pattern[0] == '/', // search by path.
	}

	return v, nil
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

// LoadOptions loads rewrite Options from a system file.
func LoadOptions(filename string) (opts Options) {
	ext := ".yml"
	if index := strings.LastIndexByte(filename, '.'); index > 1 && len(filename)-1 > index {
		ext = filename[index:]
	}

	f, err := os.Open(filename)
	if err != nil {
		panic("iris: rewrite: " + err.Error())
	}
	defer f.Close()

	switch ext {
	case ".yaml", ".yml":
		err = yaml.NewDecoder(f).Decode(&opts)
	case ".json":
		err = json.NewDecoder(f).Decode(&opts)
	default:
		panic("iris: rewrite: unexpected file extension: " + filename)
	}

	if err != nil {
		panic("iris: rewrite: decode: " + err.Error())
	}

	return
}
