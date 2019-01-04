package goreferrer

import (
	"encoding/json"
	"io"
	"net/url"
	"path"
	"strings"
)

type DomainRule struct {
	Type       ReferrerType
	Label      string
	Domain     string
	Parameters []string
}

type UaRule struct {
	Url    string
	Domain string
	Tld    string
}

func (u UaRule) RegisteredDomain() string {
	if u.Domain == "" || u.Tld == "" {
		return ""
	}

	return u.Domain + "." + u.Tld
}

type RuleSet struct {
	DomainRules map[string]DomainRule
	UaRules     map[string]UaRule
}

func NewRuleSet() RuleSet {
	return RuleSet{
		DomainRules: make(map[string]DomainRule),
		UaRules:     make(map[string]UaRule),
	}
}

func (r RuleSet) Merge(other RuleSet) {
	for k, v := range other.DomainRules {
		r.DomainRules[k] = v
	}
	for k, v := range other.UaRules {
		r.UaRules[k] = v
	}
}

func (r RuleSet) Parse(URL string) Referrer {
	return r.ParseWith(URL, nil, "")
}

func (r RuleSet) ParseWith(URL string, domains []string, agent string) Referrer {
	ref := Referrer{
		Type: Indirect,
		URL:  strings.Trim(URL, " \t\r\n"),
	}

	uaRule := r.getUaRule(agent)
	if ref.URL == "" {
		ref.URL = uaRule.Url
	}
	if ref.URL == "" {
		ref.Type = Direct
		return ref
	}

	u, ok := parseRichUrl(ref.URL)
	if !ok {
		ref.Type = Invalid
		return ref
	}

	ref.Subdomain = u.Subdomain
	ref.Domain = u.Domain
	ref.Tld = u.Tld
	ref.Path = cleanPath(u.Path)

	if ref.Domain == "" {
		ref.Domain = uaRule.Domain
	}
	if ref.Tld == "" {
		ref.Tld = uaRule.Tld
	}

	for _, domain := range domains {
		if u.Host == domain {
			ref.Type = Direct
			return ref
		}
	}

	variations := []string{
		path.Join(u.Host, u.Path),
		path.Join(u.RegisteredDomain(), u.Path),
		u.Host,
		u.RegisteredDomain(),
	}

	for _, host := range variations {
		domainRule, exists := r.DomainRules[host]
		if !exists {
			continue
		}

		query := getQuery(u.Query(), domainRule.Parameters)
		if query == "" {
			values, err := url.ParseQuery(u.Fragment)
			if err == nil {
				query = getQuery(values, domainRule.Parameters)
			}
		}

		ref.Type = domainRule.Type
		ref.Label = domainRule.Label
		ref.Query = query
		ref.GoogleType = googleSearchType(ref)
		return ref
	}

	ref.Label = strings.Title(u.Domain)
	return ref
}

func (r *RuleSet) getUaRule(agent string) UaRule {
	for pattern, rule := range r.UaRules {
		if strings.Contains(agent, pattern) {
			return rule
		}
	}

	return UaRule{}
}

func getQuery(values url.Values, params []string) string {
	for _, param := range params {
		query := values.Get(param)
		if query != "" {
			return query
		}
	}

	return ""
}

func googleSearchType(ref Referrer) GoogleSearchType {
	if ref.Type != Search || !strings.Contains(ref.Label, "Google") {
		return NotGoogleSearch
	}

	if strings.HasPrefix(ref.Path, "/aclk") || strings.HasPrefix(ref.Path, "/pagead/aclk") {
		return Adwords
	}

	return OrganicSearch
}

func cleanPath(path string) string {
	if i := strings.Index(path, ";"); i != -1 {
		return path[:i]
	}
	return path
}

type jsonRule struct {
	Domains    []string
	Parameters []string
}

type jsonRules struct {
	Email  map[string]jsonRule
	Search map[string]jsonRule
	Social map[string]jsonRule
}

func LoadJsonDomainRules(reader io.Reader) (map[string]DomainRule, error) {
	var decoded jsonRules
	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		return nil, err
	}

	rules := NewRuleSet()
	rules.Merge(extractRules(decoded.Email, Email))
	rules.Merge(extractRules(decoded.Search, Search))
	rules.Merge(extractRules(decoded.Social, Social))
	return rules.DomainRules, nil
}

func extractRules(ruleMap map[string]jsonRule, Type ReferrerType) RuleSet {
	rules := NewRuleSet()
	for label, jsonRule := range ruleMap {
		for _, domain := range jsonRule.Domains {
			rules.DomainRules[domain] = DomainRule{
				Type:       Type,
				Label:      label,
				Domain:     domain,
				Parameters: jsonRule.Parameters,
			}
		}
	}
	return rules
}
