// Package rewrite redirects or rewrites requests according to a list of
// regular-expression rules, and can force a primary subdomain such as "www".
//
// Its entry point is New(Options), which returns an *Engine rather than a
// Handler because the engine is also a router wrapper with its own logger:
// register its `Handler` method with Application.UseRouter. Use `Must` to panic
// on a bad rule instead of reporting it, or `Load` to read the rules from a
// YAML or JSON file.
package rewrite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/core/server"

	"github.com/kataras/golog"
	"gopkg.in/yaml.v3"
)

func init() {
	context.SetHandlerName("iris/middleware/rewrite.*", "iris.rewrite")
}

// Options holds the developer input to customize
// the redirects for the Rewrite Engine.
// Look the `New` and `Load` package-level functions.
type Options struct {
	// RedirectMatch accepts a slice of lines
	// of form:
	// REDIRECT_CODE PATH_PATTERN TARGET_PATH
	// Example: []{"301 /seo/(.*) /$1"}.
	RedirectMatch []string `json:"redirectMatch" yaml:"RedirectMatch"`

	// Root domain requests redirect automatically to primary subdomain.
	// Example: "www" to redirect always to www.
	// Note that you SHOULD NOT create a www subdomain inside the Iris Application.
	// This field takes care of it for you, the root application instance
	// will be used to serve the requests.
	PrimarySubdomain string `json:"primarySubdomain" yaml:"PrimarySubdomain"`
}

// LoadOptions loads rewrite Options from a system file.
func LoadOptions(filename string) (opts Options) {
	ext := ".yml"
	if before, after, ok := strings.CutLast(filename, "."); ok && len(before) > 1 && after != "" {
		ext = "." + after
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

// Engine is a struct that represents a rewrite engine for Iris web framework.
// It contains a slice of redirect rules, an options struct, a logger, and a domain validator function.
// It provides methods to create, configure, and apply rewrite rules to HTTP requests and responses.
type Engine struct {
	redirects []*redirectMatch
	options   Options

	logger          *golog.Logger
	domainValidator func(string) bool
}

// Load decodes the "filename" options
// and returns a new Rewrite Engine Router Middleware.
// It panics on errors.
// Usage:
// redirects := Load("redirects.yml")
// app.UseRouter(redirects)
// See `New` too.
func Load(filename string) context.Handler {
	return Must(LoadOptions(filename)).Handler
}

// Must is New but it panics on a parser error.
// Usage:
//
//	engine := rewrite.Must(rewrite.Options{RedirectMatch: []string{"301 /seo/(.*) /$1"}})
//	app.UseRouter(engine.Handler)
func Must(opts Options) *Engine {
	engine, err := New(opts)
	if err != nil {
		panic(err)
	}

	return engine
}

// New returns a new Rewrite Engine based on "opts".
// It reports any parser error.
// See its `Handler` method and the `Must` companion.
func New(opts Options) (*Engine, error) {
	redirects := make([]*redirectMatch, 0, len(opts.RedirectMatch))

	for _, line := range opts.RedirectMatch {
		r, err := parseRedirectMatchLine(line)
		if err != nil {
			return nil, err
		}
		redirects = append(redirects, r)
	}

	if opts.PrimarySubdomain != "" && !strings.HasSuffix(opts.PrimarySubdomain, ".") {
		opts.PrimarySubdomain += "." // www -> www.
	}

	e := &Engine{
		options:   opts,
		redirects: redirects,
		domainValidator: func(root string) bool {
			return !strings.HasSuffix(root, localhost)
		},
	}
	return e, nil
}

// SetLogger attachs a logger to the Rewrite Engine,
// used only for debugging.
// Defaults to nil.
func (e *Engine) SetLogger(logger *golog.Logger) *Engine {
	e.logger = logger.Child(e).SetChildPrefix("rewrite")
	return e
}

// init the request logging with [DBUG].
func (e *Engine) initDebugf(format string, args ...any) {
	if e.logger == nil {
		return
	}

	e.logger.Debugf(format, args...)
}

var skipDBUGSpace = strings.Repeat(" ", 7)

// continue debugging the same request with new lines and spacing,
// easier to read.
func (e *Engine) debugf(format string, args ...any) {
	if e.logger == nil || e.logger.Level < golog.DebugLevel {
		return
	}

	fmt.Fprintf(e.logger.Printer, skipDBUGSpace+format, args...)
}

const localhost = "localhost"

// Handler is a router middleware, it is used to wrap the entire Iris Router to apply the given rewrite rules.
//
// Usage:
// app.UseRouter(engine.Handler).
func (e *Engine) Handler(ctx *context.Context) {
	r := ctx.Request()

	if primarySubdomain := e.options.PrimarySubdomain; primarySubdomain != "" {
		hostport := ctx.Host()

		// if it's an IP address, don't do anything.
		if _, err := netip.ParseAddr(hostport); err == nil {
			// IP addresses don't have subdomains: Unlike domain names, IP addresses don’t have a hierarchical structure that includes subdomains.
			// Redirecting an IP address to a “www” subdomain doesn’t make sense because it’s not a valid operation for an IP address.
			ctx.Next()
			return
		}

		root := server.GetRootDomain(hostport)

		e.initDebugf("Begin request: full host: %s and root domain: %s\n", hostport, root)
		// Note:
		// localhost and 127.0.0.1 are not supported for subdomain rewrite, by purpose,
		// use a virtual host instead.
		// GetDomain will return will return localhost or www.localhost
		// on expected loopbacks.
		if e.domainValidator(root) {
			root += getPort(hostport)
			subdomain := strings.TrimSuffix(hostport, root)

			e.debugf("Domain is not a loopback, requested subdomain: %s\n", subdomain)

			if subdomain == "" {
				// we are in root domain, full redirect to its primary subdomain.
				newHost := primarySubdomain + root
				e.debugf("Redirecting from root domain to: %s\n", newHost)
				u := r.URL.Clone()
				u.Host = newHost
				ctx.Redirect(u.String(), http.StatusMovedPermanently)
				return
			}

			if subdomain == primarySubdomain {
				// keep root domain as the Host field inside the next handlers,
				// for consistently use and
				// to bypass the subdomain router (`routeHandler`)
				// do not return, redirects should be respected.
				rootHost := strings.TrimPrefix(hostport, subdomain)
				e.debugf("Request host field was modified to: %s. Proceed without redirection\n", rootHost)
				// modify those for the next redirects or the route handler.
				r.Host = rootHost
				r.URL.Host = rootHost
			}

			// maybe other subdomain or not at all, let's continue.
		} else {
			e.debugf("Primary subdomain is: %s but redirect response was not sent. Domain is a loopback?\n", primarySubdomain)
		}
	}

	for _, rd := range e.redirects {
		src := r.URL.Path
		if !rd.isRelativePattern {
			// don't change the request, use a full redirect.
			src = context.GetScheme(r) + context.GetHost(r) + r.URL.RequestURI()
		}

		if target, ok := rd.matchAndReplace(src); ok {
			if target == src {
				e.debugf("WARNING: source and target URLs match: %s\n", src)
				ctx.Next()
				return
			}

			if rd.noRedirect {
				u, err := r.URL.Parse(target)
				if err != nil {
					// Private: the parse error quotes the rewrite target, which is
					// built from the server's own rules.
					ctx.StopWithPlainError(http.StatusMisdirectedRequest, context.PrivateError(err))
					return
				}

				e.debugf("No redirect: handle request: %s as: %s\n", r.RequestURI, u)
				r.URL = u
				ctx.Next()
				return
			}

			if !rd.isRelativePattern {
				// this performs better, no need to check query or host,
				// the uri already built.
				e.debugf("Full redirect: from: %s to: %s\n", src, target)
				router.RedirectAbsolute(ctx, target, rd.code)
			} else {
				// Note(@kataras): a relative rule must produce a relative target.
				//
				// With the documented rule `301 /seo/(.*) /$1`, the request
				// GET /seo//evil.com yields the target "//evil.com" - which every
				// browser resolves as https://evil.com, not as a path on this host.
				// That is an open redirect handed to an attacker through a rule
				// whose author believed it could only ever rewrite local paths.
				if isProtocolRelativeOrAbsolute(target) {
					e.debugf("WARNING: relative rule produced a non-relative target: %s\n", target)
					ctx.Next()
					return
				}

				e.debugf("Path redirect: from: %s to: %s\n", src, target)
				ctx.Redirect(target, rd.code)
			}

			return
		}
	}

	ctx.Next()
}

type redirectMatch struct {
	code    int
	pattern *regexp.Regexp
	target  string

	isRelativePattern bool
	noRedirect        bool
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

	regex := regexp.MustCompile(pattern)
	if regex.MatchString(target) {
		return nil, fmt.Errorf("redirect match: loop detected: pattern: %s vs target: %s", pattern, target)
	}

	v := &redirectMatch{
		code:              code,
		pattern:           regex,
		target:            target,
		noRedirect:        code <= 0,
		isRelativePattern: pattern[0] == '/', // search by path.
	}

	return v, nil
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func getPort(hostport string) string { // returns :port, note that this is only called on non-loopbacks.
	if portIdx := strings.IndexByte(hostport, ':'); portIdx > 0 {
		return hostport[portIdx:]
	}

	return ""
}

// isProtocolRelativeOrAbsolute reports whether "target" would take the client
// off this host.
//
// A leading "//" is protocol-relative: the browser reads what follows as an
// authority, not as a path. A "scheme:" prefix is absolute. Backslashes are
// included because several browsers normalise them to forward slashes, so
// "/\evil.com" and "\evil.com" behave like "//evil.com".
func isProtocolRelativeOrAbsolute(target string) bool {
	if target == "" {
		return false
	}

	normalized := strings.ReplaceAll(target, "\\", "/")

	if strings.HasPrefix(normalized, "//") {
		return true
	}

	// A scheme must appear before any slash, e.g. "javascript:", "https://".
	if i := strings.IndexByte(normalized, ':'); i > 0 {
		if slash := strings.IndexByte(normalized, '/'); slash == -1 || i < slash {
			return true
		}
	}

	return false
}
