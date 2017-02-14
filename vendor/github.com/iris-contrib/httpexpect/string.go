package httpexpect

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

// String provides methods to inspect attached string value
// (Go representation of JSON string).
type String struct {
	chain chain
	value string
}

// NewString returns a new String given a reporter used to report failures
// and value to be inspected.
//
// reporter should not be nil.
//
// Example:
//  str := NewString(t, "Hello")
func NewString(reporter Reporter, value string) *String {
	return &String{makeChain(reporter), value}
}

// Raw returns underlying value attached to String.
// This is the value originally passed to NewString.
//
// Example:
//  str := NewString(t, "Hello")
//  assert.Equal(t, "Hello", str.Raw())
func (s *String) Raw() string {
	return s.value
}

// Path is similar to Value.Path.
func (s *String) Path(path string) *Value {
	return getPath(&s.chain, s.value, path)
}

// Schema is similar to Value.Schema.
func (s *String) Schema(schema interface{}) *String {
	checkSchema(&s.chain, s.value, schema)
	return s
}

// Length returns a new Number object that may be used to inspect string length.
//
// Example:
//  str := NewString(t, "Hello")
//  str.Length().Equal(5)
func (s *String) Length() *Number {
	return &Number{s.chain, float64(len(s.value))}
}

// DateTime parses date/time from string an returns a new DateTime object.
//
// If layout is given, DateTime() uses time.Parse() with given layout.
// Otherwise, it uses http.ParseTime(). If pasing error occurred,
// DateTime reports failure and returns empty (but non-nil) object.
//
// Example:
//   str := NewString(t, "Tue, 15 Nov 1994 08:12:31 GMT")
//   str.DateTime().Lt(time.Now())
//
//   str := NewString(t, "15 Nov 94 08:12 GMT")
//   str.DateTime(time.RFC822).Lt(time.Now())
func (s *String) DateTime(layout ...string) *DateTime {
	if s.chain.failed() {
		return &DateTime{s.chain, time.Unix(0, 0)}
	}
	var (
		t   time.Time
		err error
	)
	if len(layout) != 0 {
		t, err = time.Parse(layout[0], s.value)
	} else {
		t, err = http.ParseTime(s.value)
	}
	if err != nil {
		s.chain.fail(err.Error())
		return &DateTime{s.chain, time.Unix(0, 0)}
	}
	return &DateTime{s.chain, t}
}

// Empty succeeds if string is empty.
//
// Example:
//  str := NewString(t, "")
//  str.Empty()
func (s *String) Empty() *String {
	return s.Equal("")
}

// NotEmpty succeeds if string is non-empty.
//
// Example:
//  str := NewString(t, "Hello")
//  str.NotEmpty()
func (s *String) NotEmpty() *String {
	return s.NotEqual("")
}

// Equal succeeds if string is equal to another str.
//
// Example:
//  str := NewString(t, "Hello")
//  str.Equal("Hello")
func (s *String) Equal(value string) *String {
	if !(s.value == value) {
		s.chain.fail("\nexpected string equal to:\n %q\n\nbut got:\n %q",
			value, s.value)
	}
	return s
}

// NotEqual succeeds if string is not equal to another str.
//
// Example:
//  str := NewString(t, "Hello")
//  str.NotEqual("Goodbye")
func (s *String) NotEqual(value string) *String {
	if !(s.value != value) {
		s.chain.fail("\nexpected string not equal to:\n %q", value)
	}
	return s
}

// EqualFold succeeds if string is equal to another string under Unicode case-folding
// (case-insensitive match).
//
// Example:
//  str := NewString(t, "Hello")
//  str.EqualFold("hELLo")
func (s *String) EqualFold(value string) *String {
	if !strings.EqualFold(s.value, value) {
		s.chain.fail(
			"\nexpected string equal to (case-insensitive):\n %q\n\nbut got:\n %q",
			value, s.value)
	}
	return s
}

// NotEqualFold succeeds if string is not equal to another string under Unicode
// case-folding (case-insensitive match).
//
// Example:
//  str := NewString(t, "Hello")
//  str.NotEqualFold("gOODBYe")
func (s *String) NotEqualFold(value string) *String {
	if strings.EqualFold(s.value, value) {
		s.chain.fail(
			"\nexpected string not equal to (case-insensitive):\n %q\n\nbut got:\n %q",
			value, s.value)
	}
	return s
}

// Contains succeeds if string contains given substr.
//
// Example:
//  str := NewString(t, "Hello")
//  str.Contains("ell")
func (s *String) Contains(value string) *String {
	if !strings.Contains(s.value, value) {
		s.chain.fail(
			"\nexpected string containing substring:\n %q\n\nbut got:\n %q",
			value, s.value)
	}
	return s
}

// NotContains succeeds if string doesn't contain given substr.
//
// Example:
//  str := NewString(t, "Hello")
//  str.NotContains("bye")
func (s *String) NotContains(value string) *String {
	if strings.Contains(s.value, value) {
		s.chain.fail(
			"\nexpected string not containing substring:\n %q\n\nbut got:\n %q",
			value, s.value)
	}
	return s
}

// ContainsFold succeeds if string contains given substring under Unicode case-folding
// (case-insensitive match).
//
// Example:
//  str := NewString(t, "Hello")
//  str.ContainsFold("ELL")
func (s *String) ContainsFold(value string) *String {
	if !strings.Contains(strings.ToLower(s.value), strings.ToLower(value)) {
		s.chain.fail(
			"\nexpected string containing substring (case-insensitive):\n %q"+
				"\n\nbut got:\n %q", value, s.value)
	}
	return s
}

// NotContainsFold succeeds if string doesn't contain given substring under Unicode
// case-folding (case-insensitive match).
//
// Example:
//  str := NewString(t, "Hello")
//  str.NotContainsFold("BYE")
func (s *String) NotContainsFold(value string) *String {
	if strings.Contains(strings.ToLower(s.value), strings.ToLower(value)) {
		s.chain.fail(
			"\nexpected string not containing substring (case-insensitive):\n %q"+
				"\n\nbut got:\n %q", value, s.value)
	}
	return s
}

// Match matches the string with given regexp and returns a new Match object
// with found submatches.
//
// If regexp is invalid or string doesn't match regexp, Match fails and returns
// empty (but non-nil) object. regexp.Compile is used to construct regexp, and
// Regexp.FindStringSubmatch is used to construct matches.
//
// Example:
//   s := NewString(t, "http://example.com/users/john")
//   m := s.Match(`http://(?P<host>.+)/users/(?P<user>.+)`)
//
//   m.NotEmpty()
//   m.Length().Equal(3)
//
//   m.Index(0).Equal("http://example.com/users/john")
//   m.Index(1).Equal("example.com")
//   m.Index(2).Equal("john")
//
//   m.Name("host").Equal("example.com")
//   m.Name("user").Equal("john")
func (s *String) Match(re string) *Match {
	r, err := regexp.Compile(re)
	if err != nil {
		s.chain.fail(err.Error())
		return makeMatch(s.chain, nil, nil)
	}

	m := r.FindStringSubmatch(s.value)
	if m == nil {
		s.chain.fail("\nexpected string matching regexp:\n `%s`\n\nbut got:\n %q",
			re, s.value)
		return makeMatch(s.chain, nil, nil)
	}

	return makeMatch(s.chain, m, r.SubexpNames())
}

// MatchAll find all matches in string for given regexp and returns a list
// of found matches.
//
// If regexp is invalid or string doesn't match regexp, MatchAll fails and
// returns empty (but non-nil) slice. regexp.Compile is used to construct
// regexp, and Regexp.FindAllStringSubmatch is used to find matches.
//
// Example:
//   s := NewString(t,
//      "http://example.com/users/john http://example.com/users/bob")
//
//   m := s.MatchAll(`http://(?P<host>\S+)/users/(?P<user>\S+)`)
//
//   m[0].Name("user").Equal("john")
//   m[1].Name("user").Equal("bob")
func (s *String) MatchAll(re string) []Match {
	r, err := regexp.Compile(re)
	if err != nil {
		s.chain.fail(err.Error())
		return []Match{}
	}

	matches := r.FindAllStringSubmatch(s.value, -1)
	if matches == nil {
		s.chain.fail("\nexpected string matching regexp:\n `%s`\n\nbut got:\n %q",
			re, s.value)
		return []Match{}
	}

	ret := []Match{}
	for _, m := range matches {
		ret = append(ret, *makeMatch(
			s.chain,
			m,
			r.SubexpNames()))
	}

	return ret
}

// NotMatch succeeds if the string doesn't match to given regexp.
//
// regexp.Compile is used to construct regexp, and Regexp.MatchString
// is used to perform match.
//
// Example:
//   s := NewString(t, "a")
//   s.NotMatch(`[^a]`)
func (s *String) NotMatch(re string) *String {
	r, err := regexp.Compile(re)
	if err != nil {
		s.chain.fail(err.Error())
		return s
	}

	if r.MatchString(s.value) {
		s.chain.fail("\nexpected string not matching regexp:\n `%s`\n\nbut got:\n %q",
			re, s.value)
		return s
	}

	return s
}
