package internal

import (
	"fmt"
	"sort"
)

// Renderer is responsible to render a translation based
// on the given "args".
type Renderer interface {
	Render(args ...interface{}) (string, error)
}

// Message is the default Renderer for translation messages.
// Holds the variables and the plurals of this key.
// Each Locale has its own list of messages.
type Message struct {
	Locale *Locale

	Key   string
	Value string

	Plural  bool
	Plurals []*PluralMessage // plural forms by order.

	Vars []Var
}

// AddPlural adds a plural message to the Plurals list.
func (m *Message) AddPlural(form PluralForm, r Renderer) {
	msg := &PluralMessage{
		Form:     form,
		Renderer: r,
	}

	if len(m.Plurals) == 0 {
		m.Plural = true
		m.Plurals = append(m.Plurals, msg)
		return
	}

	for i, p := range m.Plurals {
		if p.Form.String() == form.String() {
			// replace
			m.Plurals[i] = msg
			return
		}
	}

	m.Plurals = append(m.Plurals, msg)
	sort.SliceStable(m.Plurals, func(i, j int) bool {
		return m.Plurals[i].Form.Less(m.Plurals[j].Form)
	})
}

// Render completes the Renderer interface.
// It accepts arguments, which can resolve the pluralization type of the message
// and its variables. If the Message is wrapped by a Template then the
// first argument should be a map. The map key resolves to the pluralization
// of the message is the "PluralCount". And for variables the user
// should set a message key which looks like: %VAR_NAME%Count, e.g. "DogsCount"
// to set plural count for the "Dogs" variable, case-sensitive.
func (m *Message) Render(args ...interface{}) (string, error) {
	if m.Plural {
		if len(args) > 0 {
			if pluralCount, ok := findPluralCount(args[0]); ok {
				for _, plural := range m.Plurals {
					if plural.Form.MatchPlural(pluralCount) {
						return plural.Renderer.Render(args...)
					}
				}

				return "", fmt.Errorf("key: %q: no registered plurals for <%d>", m.Key, pluralCount)
			}
		}

		return "", fmt.Errorf("key: %q: missing plural count argument", m.Key)
	}

	return m.Locale.Printer.Sprintf(m.Key, args...), nil
}
