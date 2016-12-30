package httpexpect

import (
	"reflect"
)

// Match provides methods to inspect attached regexp match results.
type Match struct {
	chain      chain
	submatches []string
	names      map[string]int
}

// NewMatch returns a new Match object given a reporter used to report
// failures and submatches to be inspected.
//
// reporter should not be nil. submatches and names may be nil.
//
// Example:
//   s := "http://example.com/users/john"
//   r := regexp.MustCompile(`http://(?P<host>.+)/users/(?P<user>.+)`)
//   m := NewMatch(reporter, r.FindStringSubmatch(s), r.SubexpNames())
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
func NewMatch(reporter Reporter, submatches []string, names []string) *Match {
	return makeMatch(makeChain(reporter), submatches, names)
}

func makeMatch(chain chain, submatches []string, names []string) *Match {
	if submatches == nil {
		submatches = []string{}
	}
	namemap := map[string]int{}
	for n, name := range names {
		if name != "" {
			namemap[name] = n
		}
	}
	return &Match{chain, submatches, namemap}
}

// Raw returns underlying submatches attached to Match.
// This is the value originally passed to NewMatch.
//
// Example:
//  m := NewMatch(t, submatches, names)
//  assert.Equal(t, submatches, m.Raw())
func (m *Match) Raw() []string {
	return m.submatches
}

// Length returns a new Number object that may be used to inspect
// number of submatches.
//
// Example:
//  m := NewMatch(t, submatches, names)
//  m.Length().Equal(len(submatches))
func (m *Match) Length() *Number {
	return &Number{m.chain, float64(len(m.submatches))}
}

// Index returns a new String object that may be used to inspect submatch
// with given index.
//
// Note that submatch with index 0 contains the whole match. If index is out
// of bounds, Index reports failure and returns empty (but non-nil) value.
//
// Example:
//   s := "http://example.com/users/john"
//
//   r := regexp.MustCompile(`http://(.+)/users/(.+)`)
//   m := NewMatch(t, r.FindStringSubmatch(s), nil)
//
//   m.Index(0).Equal("http://example.com/users/john")
//   m.Index(1).Equal("example.com")
//   m.Index(2).Equal("john")
func (m *Match) Index(index int) *String {
	if index < 0 || index >= len(m.submatches) {
		m.chain.fail(
			"\nsubmatch index out of bounds:\n  index %d\n\n  bounds [%d; %d)",
			index,
			0,
			len(m.submatches))
		return &String{m.chain, ""}
	}
	return &String{m.chain, m.submatches[index]}
}

// Name returns a new String object that may be used to inspect submatch
// with given name.
//
// If there is no submatch with given name, Name reports failure and returns
// empty (but non-nil) value.
//
// Example:
//   s := "http://example.com/users/john"
//
//   r := regexp.MustCompile(`http://(?P<host>.+)/users/(?P<user>.+)`)
//   m := NewMatch(t, r.FindStringSubmatch(s), r.SubexpNames())
//
//   m.Name("host").Equal("example.com")
//   m.Name("user").Equal("john")
func (m *Match) Name(name string) *String {
	index, ok := m.names[name]
	if !ok {
		m.chain.fail(
			"\nsubmatch name not found:\n %q\n\navailable names:\n%s",
			name,
			dumpValue(m.names))
		return &String{m.chain, ""}
	}
	return m.Index(index)
}

// Empty succeeds if submatches array is empty.
//
// Example:
//  m := NewMatch(t, submatches, names)
//  m.Empty()
func (m *Match) Empty() *Match {
	if len(m.submatches) != 0 {
		m.chain.fail("\nexpected zero submatches, but got:\n  %s",
			dumpValue(m.submatches))
	}
	return m
}

// NotEmpty succeeds if submatches array is non-empty.
//
// Example:
//  m := NewMatch(t, submatches, names)
//  m.NotEmpty()
func (m *Match) NotEmpty() *Match {
	if len(m.submatches) == 0 {
		m.chain.fail("expected non-zero submatches")
	}
	return m
}

// Values succeeds if submatches array, starting from index 1, is equal to
// given array.
//
// Note that submatch with index 0 contains the whole match and is not
// included into this check.
//
// Example:
//   s := "http://example.com/users/john"
//   r := regexp.MustCompile(`http://(.+)/users/(.+)`)
//   m := NewMatch(t, r.FindStringSubmatch(s), nil)
//   m.Values("example.com", "john")
func (m *Match) Values(values ...string) *Match {
	if values == nil {
		values = []string{}
	}
	if !reflect.DeepEqual(values, m.getValues()) {
		m.chain.fail("\nexpected submatches equal to:\n%s\n\nbut got:\n%s",
			dumpValue(values),
			dumpValue(m.getValues()))
	}
	return m
}

// NotValues succeeds if submatches array, starting from index 1, is not
// equal to given array.
//
// Note that submatch with index 0 contains the whole match and is not
// included into this check.
//
// Example:
//   s := "http://example.com/users/john"
//   r := regexp.MustCompile(`http://(.+)/users/(.+)`)
//   m := NewMatch(t, r.FindStringSubmatch(s), nil)
//   m.NotValues("example.com", "bob")
func (m *Match) NotValues(values ...string) *Match {
	if values == nil {
		values = []string{}
	}
	if reflect.DeepEqual(values, m.getValues()) {
		m.chain.fail("\nexpected submatches not equal to:\n%s",
			dumpValue(values))
	}
	return m
}

func (m *Match) getValues() []string {
	if len(m.submatches) > 1 {
		return m.submatches[1:]
	}
	return []string{}
}
