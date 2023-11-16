package apps

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

type (
	// Host holds the pattern for the SwitchCase filter
	// and the Target host or application.
	Host struct {
		// Pattern is the incoming host matcher regexp or a literal.
		Pattern string
		// Target is the target Host that incoming requests will be redirected on pattern match
		// or an Application's Name that will handle the incoming request matched the Pattern.
		Target interface{} // It was a string in my initial design but let's do that interface{}, we may support more types here in the future, until generics are in, keep it interface{}.
	}
	// Hosts is a switch provider.
	// It can be used as input argument to the `Switch` function
	// to map host to existing Iris Application instances, e.g.
	// { "www.mydomain.com": "mydomainApp" } .
	// It can accept regexp as a host too, e.g.
	// { "^my.*$": "mydomainApp" } .
	Hosts []Host

	// Good by we need order and map can't provide it for us
	//  (e.g. "fallback" regexp }
	// Hosts map[string]*iris.Application
)

var _ SwitchProvider = Hosts{}

// AnyDomain is a regexp that matches any domain.
// It can be used as the Pattern field of a Host.
//
// Example:
//
//	apps.Switch(apps.Hosts{
//		{
//			Pattern: "^id.*$", Target: identityApp,
//		},
//		{
//			Pattern: apps.AnyDomain, Target: app,
//		},
//	}).Listen(":80")
const AnyDomain = `^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z
	]{2,3})$`

// GetSwitchCases completes the SwitchProvider.
// It returns a slice of SwitchCase which
// if passed on `Switch` function, they act
// as a router between matched domains and subdomains
// between existing Iris Applications.
func (hosts Hosts) GetSwitchCases() []SwitchCase {
	cases := make([]SwitchCase, 0, len(hosts))

	for _, host := range hosts {
		cases = append(cases, SwitchCase{
			Filter: hostFilter(host.Pattern),
			App:    hostApp(host),
		})
	}

	return cases
}

// GetFriendlyName implements the FriendlyNameProvider.
func (hosts Hosts) GetFriendlyName() string {
	var patterns []string
	for _, host := range hosts {
		if strings.TrimSpace(host.Pattern) != "" {
			patterns = append(patterns, host.Pattern)
		}
	}

	return strings.Join(patterns, ", ")
}

func hostApp(host Host) *iris.Application {
	if host.Target == nil {
		return nil
	}

	switch target := host.Target.(type) {
	case context.Application:
		return target.(*iris.Application)
	case string:
		// Check if the given target is an application name, if so
		// we must not redirect (loop) we must serve the request
		// using that app.
		if targetApp, ok := context.GetApplication(target); ok {
			// It's always iris.Application so we are totally safe here.
			return targetApp.(*iris.Application)
		}
		// If it's a real host, warn the user of invalid input.
		u, err := url.Parse(target)
		if err == nil && u.IsAbs() {
			// remember, we redirect hosts, not full URLs here.
			panic(fmt.Sprintf(`iris: switch: hosts: invalid target host: "%s"`, target))
		}

		if regex := regexp.MustCompile(host.Pattern); regex.MatchString(target) {
			panic(fmt.Sprintf(`iris: switch: hosts: loop detected between expression: "%s" and target host: "%s"`, host.Pattern, host.Target))
		}

		return newHostRedirectApp(target, HostsRedirectCode)
	default:
		panic(fmt.Sprintf("iris: switch: hosts: invalid target type: %T", target))
	}
}

func hostFilter(expr string) iris.Filter {
	regex := regexp.MustCompile(expr)
	return func(ctx iris.Context) bool {
		return regex.MatchString(ctx.Host())
	}
}

// HostsRedirectCode is the default status code is used
// to redirect a matching host to a url.
var HostsRedirectCode = iris.StatusMovedPermanently

func newHostRedirectApp(targetHost string, code int) *iris.Application {
	app := iris.New()
	app.Downgrade(func(w http.ResponseWriter, r *http.Request) {
		if targetHost == context.GetHost(r) {
			// Note(@kataras):
			// this should never happen as the HostsRedirect
			// carefully checks if the expression already matched the "redirectTo"
			// to avoid the redirect loops at all.
			// iris: switch: hosts redirect: loop detected between expression: "^my.*$" and target host: "mydomain.com"
			http.Error(w, iris.StatusText(iris.StatusTooManyRequests), iris.StatusTooManyRequests)
			return
		}

		r.Host = targetHost
		r.URL.Host = targetHost
		// r.URL.User = nil
		http.Redirect(w, r, r.URL.String(), code)
	})
	return app
}
