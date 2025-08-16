package router

import (
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12/core/netutil"
	"github.com/kataras/iris/v12/macro"
	"github.com/kataras/iris/v12/macro/interpreter/ast"
	"github.com/kataras/iris/v12/macro/interpreter/lexer"
)

// Param receives a parameter name prefixed with the ParamStart symbol.
func Param(name string) string {
	return prefix(name, ParamStart)
}

// WildcardParam receives a parameter name prefixed with the WildcardParamStart symbol.
func WildcardParam(name string) string {
	if name == "" {
		return ""
	}
	return prefix(name, WildcardParamStart)
}

// WildcardFileParam wraps a named parameter "file" with the trailing "path" macro parameter type.
// At build state this "file" parameter is prefixed with the request handler's `WildcardParamStart`.
// Created mostly for routes that serve static files to be visibly collected by
// the `Application#GetRouteReadOnly` via the `Route.Tmpl().Src` instead of
// the underline request handler's representation (`Route.Path()`).
func WildcardFileParam() string {
	return "{file:path}"
}

func convertMacroTmplToNodePath(tmpl macro.Template) string {
	routePath := tmpl.Src
	if len(routePath) > 1 && routePath[len(routePath)-1] == '/' {
		routePath = routePath[0 : len(routePath)-1] // remove any last "/"
	}

	// if it has started with {} and it's valid
	// then the tmpl.Params will be filled,
	// so no any further check needed.
	for i := range tmpl.Params {
		p := tmpl.Params[i]
		if ast.IsTrailing(p.Type) {
			routePath = strings.Replace(routePath, p.Src, WildcardParam(p.Name), 1)
		} else {
			routePath = strings.Replace(routePath, p.Src, Param(p.Name), 1)
		}
	}

	return routePath
}

func prefix(s string, prefix string) string {
	if !strings.HasPrefix(s, prefix) {
		return prefix + s
	}

	return s
}

func splitMethod(methodMany string) []string {
	methodMany = strings.Trim(methodMany, " ")
	return strings.Split(methodMany, " ")
}

func splitPath(pathMany string) (paths []string) {
	pathMany = strings.Trim(pathMany, " ")
	pathsWithoutSlashFromFirstAndSoOn := strings.Split(pathMany, " /")
	for _, path := range pathsWithoutSlashFromFirstAndSoOn {
		if path == "" {
			continue
		}
		if path[0] != '/' {
			path = "/" + path
		}
		paths = append(paths, path)
	}
	return
}

func joinPath(path1 string, path2 string) string {
	return path.Join(path1, path2)
}

// cleanPath applies the following rules
// iteratively until no further processing can be done:
//
//  1. Replace multiple slashes with a single slash.
//  2. Replace '\' with '/'
//  3. Replace "\\" with '/'
//  4. Ignore anything inside '{' and '}'
//  5. Makes sure that prefixed with '/'
//  6. Remove trailing '/'.
//
// The returned path ends in a slash only if it is the root "/".
// The function does not modify the dynamic path parts.
func cleanPath(path string) string {
	// note that we don't care about the performance here, it's before the server ran.
	if path == "" || path == "." {
		return "/"
	}

	// remove suffix "/", if it's root "/" then it will add it as a prefix below.
	if lidx := len(path) - 1; path[lidx] == '/' {
		path = path[:lidx]
	}

	// prefix with "/".
	path = prefix(path, "/")

	s := []rune(path)

	// If you're learning go through Iris I will ask you to ignore the
	// following part, it's not the recommending way to do that,
	// but it's understable to me.
	var (
		insideMacro = false
		i           = -1
	)

	for {
		i++
		if len(s) <= i {
			break
		}

		if s[i] == lexer.Begin {
			insideMacro = true
			continue
		}

		if s[i] == lexer.End {
			insideMacro = false
			continue
		}

		// when inside {} then don't try to clean it.
		if !insideMacro {
			if s[i] == '\\' {
				s[i] = '/'

				if len(s)-1 > i+1 && s[i+1] == '\\' {
					s = deleteCharacter(s, i+1)
				} else {
					i-- // set to minus in order for the next check to be applied for prev tokens too.
				}
			}

			if s[i] == '/' && len(s)-1 > i+1 && s[i+1] == '/' {
				s[i] = '/'
				s = deleteCharacter(s, i+1)
				i--
				continue
			}
		}
	}

	if len(s) > 1 && s[len(s)-1] == '/' { // remove any last //.
		s = s[:len(s)-1]
	}

	return string(s)
}

func deleteCharacter(s []rune, index int) []rune {
	return append(s[0:index], s[index+1:]...)
}

const (
	// SubdomainWildcardIndicator where a registered path starts with '*.'.
	// if subdomain == "*." then its wildcard.
	//
	// used internally by router and api builder.
	SubdomainWildcardIndicator = "*."

	// SubdomainWildcardPrefix where a registered path starts with "*./",
	// then this route should accept any subdomain.
	SubdomainWildcardPrefix = SubdomainWildcardIndicator + "/"
	// SubdomainPrefix where './' exists in a registered path then it contains subdomain
	//
	// used on api builder.
	SubdomainPrefix = "./" // i.e subdomain./ -> Subdomain: subdomain. Path: /
)

func hasSubdomain(s string) bool {
	if s == "" {
		return false
	}

	// subdomain./path
	// .*/path
	//
	// remember: path always starts with "/"
	// if not start with "/" then it should be something else,
	// we don't assume anything else but subdomain.
	slashIdx := strings.IndexByte(s, '/')
	return slashIdx > 0 || // for route paths
		s[0] == SubdomainPrefix[0] || // for route paths
		(len(s) >= 2 && s[0:2] == SubdomainWildcardIndicator) || // for party rel path or route paths
		(len(s) >= 2 && slashIdx != 0 && s[len(s)-1] == '.') // for party rel, i.e www., or subsub.www.
}

// splitSubdomainAndPath checks if the path has subdomain and if it's
// it splits the subdomain and path and returns them, otherwise it returns
// an empty subdomain and the clean path.
//
// First return value is the subdomain, second is the path.
func splitSubdomainAndPath(fullUnparsedPath string) (subdomain string, path string) {
	s := fullUnparsedPath
	if s == "" || s == "/" {
		return "", "/"
	}

	splitPath := strings.Split(s, ".")
	if len(splitPath) == 2 && splitPath[1] == "" {
		return splitPath[0] + ".", "/"
	}

	slashIdx := strings.IndexByte(s, '/')
	if slashIdx > 0 {
		// has subdomain
		subdomain = s[0:slashIdx]
	}

	if slashIdx == -1 {
		// this will only happen when this function
		// is called to Party's relative path (e.g. control.admin.),
		// and not a route's one (the route's one always contains a slash).
		// return all as subdomain and "/" as path.
		return s, "/"
	}

	path = s[slashIdx:]
	if !strings.Contains(path, "{") {
		path = strings.ReplaceAll(path, "//", "/")
		path = strings.ReplaceAll(path, "\\", "/")
	}

	// remove any left trailing slashes, i.e "//api/users".
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[0:i] + path[i+1:]
		} else {
			break
		}
	}

	// remove last /.
	path = strings.TrimRight(path, "/")

	// no cleanPath(path) in order
	// to be able to parse macro function regexp(\\).
	return // return subdomain without slash, path with slash
}

func staticPath(src string) string {
	bidx := strings.IndexByte(src, '{')
	if bidx == -1 || len(src) <= bidx {
		return src // no dynamic part found
	}
	if bidx <= 1 { // found at first{...} or second index (/{...}),
		// although first index should never happen because of the prepended slash.
		return "/"
	}

	return src[:bidx-1] // (/static/{...} -> /static)
}

// RoutePathReverserOption option signature for the RoutePathReverser.
type RoutePathReverserOption func(*RoutePathReverser)

// WithScheme is an option for the RoutepathReverser,
// it sets the optional field "vscheme",
// v for virtual.
// if vscheme is empty then it will try to resolve it from
// the RoutePathReverser's vhost field.
//
// See WithHost or WithServer to enable the URL feature.
func WithScheme(scheme string) RoutePathReverserOption {
	return func(ps *RoutePathReverser) {
		ps.vscheme = scheme
	}
}

// WithHost enables the RoutePathReverser's URL feature.
// Both "WithHost" and "WithScheme" can be different from
// the real server's listening address, i.e nginx in front.
func WithHost(host string) RoutePathReverserOption {
	return func(ps *RoutePathReverser) {
		ps.vhost = host
		if ps.vscheme == "" {
			ps.vscheme = netutil.ResolveSchemeFromVHost(host)
		}
	}
}

// WithServer enables the RoutePathReverser's URL feature.
// It receives an *http.Server and tries to resolve
// a scheme and a host to be used in the URL function.
func WithServer(srv *http.Server) RoutePathReverserOption {
	return func(ps *RoutePathReverser) {
		ps.vhost = netutil.ResolveVHost(srv.Addr)
		ps.vscheme = netutil.ResolveSchemeFromServer(srv)
	}
}

// RoutePathReverser contains methods that helps to reverse a
// (dynamic) path from a specific route,
// route name is required because a route may being registered
// on more than one http method.
type RoutePathReverser struct {
	provider RoutesProvider
	// both vhost and vscheme are being used, optionally, for the URL feature.
	vhost   string
	vscheme string
}

// NewRoutePathReverser returns a new path reverser based on
// a routes provider, needed to get a route based on its name.
// Options is required for the URL function.
// See WithScheme and WithHost or WithServer.
func NewRoutePathReverser(apiRoutesProvider RoutesProvider, options ...RoutePathReverserOption) *RoutePathReverser {
	ps := &RoutePathReverser{
		provider: apiRoutesProvider,
	}
	for _, o := range options {
		o(ps)
	}
	return ps
}

// Path  returns a route path based on a route name and any dynamic named parameter's values-only.
func (ps *RoutePathReverser) Path(routeName string, paramValues ...any) string {
	r := ps.provider.GetRoute(routeName)
	if r == nil {
		return ""
	}

	if len(paramValues) == 0 {
		return r.Path
	}

	return r.ResolvePath(toStringSlice(paramValues)...)
}

func toStringSlice(args []any) (argsString []string) {
	argsSize := len(args)
	if argsSize <= 0 {
		return
	}

	argsString = make([]string, argsSize)
	for i, v := range args {
		if s, ok := v.(string); ok {
			argsString[i] = s
		} else if num, ok := v.(int); ok {
			argsString[i] = strconv.Itoa(num)
		} else if b, ok := v.(bool); ok {
			argsString[i] = strconv.FormatBool(b)
		} else if arr, ok := v.([]string); ok {
			if len(arr) > 0 {
				argsString[i] = arr[0]
				argsString = append(argsString, arr[1:]...)
			}
		}
	}
	return
}

// Remove the URL for now, it complicates things for the whole framework without a specific benefits,
// developers can just concat the subdomain, (host can be auto-retrieve by browser using the Path).

// URL same as Path but returns the full uri, i.e https://mysubdomain.mydomain.com/hello/iris
func (ps *RoutePathReverser) URL(routeName string, paramValues ...any) (url string) {
	if ps.vhost == "" || ps.vscheme == "" {
		return "not supported"
	}

	r := ps.provider.GetRoute(routeName)
	if r == nil {
		return
	}

	host := ps.vhost
	scheme := ps.vscheme
	args := toStringSlice(paramValues)

	// if it's dynamic subdomain then the first argument is the subdomain part
	// for this part we are responsible not the custom routers
	if len(args) > 0 && r.Subdomain == SubdomainWildcardIndicator {
		subdomain := args[0]
		host = subdomain + "." + host
		args = args[1:] // remove the subdomain part for the arguments,
	}

	if parsedPath := r.ResolvePath(args...); parsedPath != "" {
		url = scheme + "://" + host + parsedPath
	}

	return
}
