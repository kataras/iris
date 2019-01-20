package netutil

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	loopbackRegex    *regexp.Regexp
	loopbackSubRegex *regexp.Regexp
	machineHostname  string
)

func init() {
	loopbackRegex, _ = regexp.Compile(`^localhost$|^127(?:\.[0-9]+){0,2}\.[0-9]+$|^(?:0*\:)*?:?0*1$`)
	loopbackSubRegex, _ = regexp.Compile(`^127(?:\.[0-9]+){0,2}\.[0-9]+$|^(?:0*\:)*?:?0*1$`)
	machineHostname, _ = os.Hostname()
}

// IsLoopbackSubdomain checks if a string is a subdomain or a hostname.
var IsLoopbackSubdomain = func(s string) bool {
	if strings.HasPrefix(s, "127.0.0.1:") || s == "127.0.0.1" {
		return true
	}

	valid := loopbackSubRegex.MatchString(s)
	if !valid { // if regex failed to match it, then try with the pc's name.
		if !strings.Contains(machineHostname, ".") { // if machine name's is not a loopback by itself
			valid = s == machineHostname
		}

	}
	return valid
}

// IsLoopbackHost tries to catch the local addresses when a developer
// navigates to a subdomain that its hostname differs from Application.Config.Addr.
// Developer may want to override this function to return always false
// in order to not allow different hostname from Application.Config.Addr in local environment (remote is not reached).
var IsLoopbackHost = func(requestHost string) bool {
	// this func will be called if we have a subdomain actually, not otherwise, so we are
	// safe to do some hacks.

	// if subdomain.127.0.0.1:8080/path, we need to compare the 127.0.0.1
	// if subdomain.localhost:8080/mypath, we need to compare the localhost
	// if subdomain.127.0.0.1/mypath, we need to compare the 127.0.0.1
	// if subdomain.127.0.0.1, we need to compare the 127.0.0.1

	// find the first index of [:]8080 or [/]mypath or nothing(root with loopback address like 127.0.0.1)
	// remember: we are not looking for .com or these things, if is up and running then the developer
	// would probably not want to reach the server with different Application.Config.Addr than
	// he/she declared.
	portOrPathIdx := strings.LastIndexByte(requestHost, ':')

	if portOrPathIdx == 0 { //  0.0.0.0:[...]/localhost:[...]/127.0.0.1:[...]/ipv6 local...
		return true
	}
	// this will not catch ipv6 loopbacks like subdomain.0000:0:0000::01.1:8080
	// but, again, is for developers only, is hard to try to navigate with something like this,
	// and if that happened, I provide a way to override the whole "algorithm" to a custom one via "IsLoopbackHost".
	if portOrPathIdx == -1 {
		portOrPathIdx = strings.LastIndexByte(requestHost, '/')
		if portOrPathIdx == -1 {
			portOrPathIdx = len(requestHost) // if not port or / then it should be something like subodmain.127.0.0.1
		}
	}

	// remove the left part of subdomain[.]<- and the right part of ->[:]8080/[/]mypath
	// so result should be 127.0.0.1/localhost/0.0.0.0 or any ip
	subdomainFinishIdx := strings.IndexByte(requestHost, '.') + 1
	if l := len(requestHost); l <= subdomainFinishIdx || l < portOrPathIdx {
		return false // for any case to not panic here.
	}

	hostname := requestHost[subdomainFinishIdx:portOrPathIdx]
	if hostname == "" {
		return false
	}
	// we use regex here to catch all posibilities, we compiled the regex at init func
	// so it shouldn't hurt so much, but we don't care a lot because it's a special case here
	// because this function will be called only if developer him/herself can reach the server
	// with a loopback/local address, so we are totally safe.
	valid := loopbackRegex.MatchString(hostname)
	if !valid { // if regex failed to match it, then try with the pc's name.
		valid = hostname == machineHostname
	}
	return valid
}

const (
	// defaultServerHostname returns the default hostname which is "localhost"
	defaultServerHostname = "localhost"
	// defaultServerPort returns the default port which is 8080, not used
	defaultServerPort = 8080
)

var (
	// defaultServerAddr the default server addr which is: localhost:8080
	defaultServerAddr = defaultServerHostname + ":" + strconv.Itoa(defaultServerPort)
)

// ResolveAddr tries to convert a given string to an address which is compatible with net.Listener and server
func ResolveAddr(addr string) string {
	// check if addr has :port, if not do it +:80 ,we need the hostname for many cases
	a := addr
	if a == "" {
		// check for os environments
		if oshost := os.Getenv("ADDR"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOST"); oshost != "" {
			a = oshost
		} else if oshost := os.Getenv("HOSTNAME"); oshost != "" {
			a = oshost
			// check for port also here
			if osport := os.Getenv("PORT"); osport != "" {
				a += ":" + osport
			}
		} else if osport := os.Getenv("PORT"); osport != "" {
			a = ":" + osport
		} else {
			a = ":http"
		}
	}
	if portIdx := strings.IndexByte(a, ':'); portIdx == 0 {
		if a[portIdx:] == ":https" {
			a = defaultServerHostname + ":443"
		} else {
			// if contains only :port	,then the : is the first letter, so we dont have setted a hostname, lets set it
			a = defaultServerHostname + a
		}
	}

	return a
}

// ResolveHostname receives an addr of form host[:port] and returns the hostname part of it
// ex: localhost:8080 will return the `localhost`, mydomain.com:8080 will return the 'mydomain'
func ResolveHostname(addr string) string {
	if idx := strings.IndexByte(addr, ':'); idx == 0 {
		// only port, then return the localhost hostname
		return "localhost"
	} else if idx > 0 {
		return addr[0:idx]
	}
	// it's already hostname
	return addr
}

// ResolveVHost tries to get the hostname if port is no needed for Addr's usage.
// Addr is being used inside router->subdomains
// and inside {{ url }} template funcs.
// It should be the same as "browser's"
// usually they removing :80 or :443.
func ResolveVHost(addr string) string {
	if addr == ":https" || addr == ":http" {
		return "localhost"
	}

	if idx := strings.IndexByte(addr, ':'); idx == 0 {
		// only port, then return the localhost hostname
		return "localhost" + addr[idx:]
	}

	// with ':' in order to not replace the ipv6 loopback addresses
	addr = strings.Replace(addr, "0.0.0.0:", "localhost:", 1)
	port := ResolvePort(addr)
	if port == 80 || port == 443 {
		return ResolveHostname(addr)
	}

	return addr
}

const (
	// SchemeHTTPS the "https" url scheme.
	SchemeHTTPS = "https"
	// SchemeHTTP the "http" url scheme.
	SchemeHTTP = "http"
)

// ResolvePort receives an addr of form host[:port] and returns the port part of it
// ex: localhost:8080 will return the `8080`, mydomain.com will return the '80'
func ResolvePort(addr string) int {
	if portIdx := strings.IndexByte(addr, ':'); portIdx != -1 {
		afP := addr[portIdx+1:]
		p, err := strconv.Atoi(afP)
		if err == nil {
			return p
		} else if afP == SchemeHTTPS { // it's not number, check if it's :https
			return 443
		}
	}
	return 80
}

// ResolveScheme returns "https" if "isTLS" receiver is true,
// otherwise "http".
func ResolveScheme(isTLS bool) string {
	if isTLS {
		return SchemeHTTPS
	}

	return SchemeHTTP
}

// ResolveSchemeFromVHost returns the scheme based on the "vhost".
func ResolveSchemeFromVHost(vhost string) string {
	// pure check
	isTLS := strings.HasPrefix(vhost, SchemeHTTPS) || ResolvePort(vhost) == 443
	return ResolveScheme(isTLS)
}

// ResolveURL takes the scheme and an address
// and returns its URL, pure implementation but it does the job.
func ResolveURL(scheme string, addr string) string {
	host := ResolveVHost(addr)
	return scheme + "://" + host
}
