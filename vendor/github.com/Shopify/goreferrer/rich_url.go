package goreferrer

import (
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type richUrl struct {
	*url.URL
	Subdomain string
	Domain    string
	Tld       string
}

func parseRichUrl(s string) (*richUrl, bool) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, false
	}

	// assume a default scheme of http://
	if u.Scheme == "" {
		s = "http://" + s
		u, err = url.Parse(s)
		if err != nil {
			return nil, false
		}
	}

	tld, _ := publicsuffix.PublicSuffix(u.Host)
	if tld == "" || len(u.Host)-len(tld) < 2 {
		return nil, false
	}

	hostWithoutTld := u.Host[:len(u.Host)-len(tld)-1]
	lastDot := strings.LastIndex(hostWithoutTld, ".")
	if lastDot == -1 {
		return &richUrl{URL: u, Domain: hostWithoutTld, Tld: tld}, true
	}

	return &richUrl{
		URL:       u,
		Subdomain: hostWithoutTld[:lastDot],
		Domain:    hostWithoutTld[lastDot+1:],
		Tld:       tld,
	}, true
}

func (u *richUrl) RegisteredDomain() string {
	return u.Domain + "." + u.Tld
}
