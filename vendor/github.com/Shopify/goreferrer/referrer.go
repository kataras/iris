package goreferrer

type ReferrerType int

const (
	Invalid ReferrerType = iota
	Indirect
	Direct
	Email
	Search
	Social
)

func (r ReferrerType) String() string {
	switch r {
	default:
		return "invalid"
	case Indirect:
		return "indirect"
	case Direct:
		return "direct"
	case Email:
		return "email"
	case Search:
		return "search"
	case Social:
		return "social"
	}
}

type Referrer struct {
	Type       ReferrerType
	Label      string
	URL        string
	Subdomain  string
	Domain     string
	Tld        string
	Path       string
	Query      string
	GoogleType GoogleSearchType
}

func (r *Referrer) RegisteredDomain() string {
	if r.Domain != "" && r.Tld != "" {
		return r.Domain + "." + r.Tld
	}

	return ""
}

func (r *Referrer) Host() string {
	if r.Subdomain != "" {
		return r.Subdomain + "." + r.RegisteredDomain()
	}

	return r.RegisteredDomain()
}

type GoogleSearchType int

const (
	NotGoogleSearch GoogleSearchType = iota
	OrganicSearch
	Adwords
)

func (g GoogleSearchType) String() string {
	switch g {
	default:
		return "not google search"
	case OrganicSearch:
		return "organic google search"
	case Adwords:
		return "google adwords referrer"
	}
}
