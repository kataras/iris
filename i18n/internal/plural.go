package internal

import (
	"strconv"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

// PluralCounter if completes by an input argument of a message to render,
// then the plural renderer will resolve the plural count
// and any variables' counts. This is useful when the data is not a type of Map or integers.
type PluralCounter interface {
	// PluralCount returns the plural count of the message.
	// If returns -1 then this is not a valid plural message.
	PluralCount() int
	// VarCount should return the variable count, based on the variable name.
	VarCount(name string) int
}

// PluralMessage holds the registered Form and the corresponding Renderer.
// It is used on the `Message.AddPlural` method.
type PluralMessage struct {
	Form     PluralForm
	Renderer Renderer
}

type independentPluralRenderer struct {
	key     string
	printer *message.Printer
}

func newIndependentPluralRenderer(c *Catalog, loc *Locale, key string, msgs ...catalog.Message) (Renderer, error) {
	builder := catalog.NewBuilder(catalog.Fallback(c.Locales[0].tag))
	if err := builder.Set(loc.tag, key, msgs...); err != nil {
		return nil, err
	}
	printer := message.NewPrinter(loc.tag, message.Catalog(builder))
	return &independentPluralRenderer{key, printer}, nil
}

func (m *independentPluralRenderer) Render(args ...interface{}) (string, error) {
	return m.printer.Sprintf(m.key, args...), nil
}

// A PluralFormDecoder should report and return whether
// a specific "key" is a plural one. This function
// can be implemented and set on the `Options` to customize
// the plural forms and their behavior in general.
//
// See the `DefaultPluralFormDecoder` package-level
// variable for the default implementation one.
type PluralFormDecoder func(loc context.Locale, key string) (PluralForm, bool)

// DefaultPluralFormDecoder is the default `PluralFormDecoder`.
// Supprots "zero", "one", "two", "other", "=x", "<x", ">x".
var DefaultPluralFormDecoder = func(_ context.Locale, key string) (PluralForm, bool) {
	if isDefaultPluralForm(key) {
		return pluralForm(key), true
	}

	return nil, false
}

func isDefaultPluralForm(s string) bool {
	switch s {
	case "zero", "one", "two", "other":
		return true
	default:
		if len(s) > 1 {
			ch := s[0]
			if ch == '=' || ch == '<' || ch == '>' {
				if isDigit(s[1]) {
					return true
				}
			}
		}

		return false
	}
}

// A PluralForm is responsible to decode
// locale keys to plural forms and match plural forms
// based on the given pluralCount.
//
// See `pluralForm` package-level type for a default implementation.
type PluralForm interface {
	String() string
	// the string is a verified plural case's raw string value.
	// Field for priority on which order to register the plural cases.
	Less(next PluralForm) bool
	MatchPlural(pluralCount int) bool
}

type pluralForm string

func (f pluralForm) String() string {
	return string(f)
}

func (f pluralForm) Less(next PluralForm) bool {
	form1 := f.String()
	form2 := next.String()

	// Order by
	// - equals,
	// - less than
	// - greater than
	// - "zero", "one", "two"
	// - rest is last "other".
	dig1, typ1, hasDig1 := formAtoi(form1)
	if typ1 == eq {
		return true
	}

	dig2, typ2, hasDig2 := formAtoi(form2)
	if typ2 == eq {
		return false
	}

	// digits smaller, number.
	if hasDig1 {
		return !hasDig2 || dig1 < dig2
	}

	if hasDig2 {
		return false
	}

	if form1 == "other" {
		return false // other go to last.
	}

	if form2 == "other" {
		return true
	}

	if form1 == "zero" {
		return true
	}

	if form2 == "zero" {
		return false
	}

	if form1 == "one" {
		return true
	}

	if form2 == "one" {
		return false
	}

	if form1 == "two" {
		return true
	}

	if form2 == "two" {
		return false
	}

	return false
}

func (f pluralForm) MatchPlural(pluralCount int) bool {
	switch f {
	case "other":
		return true
	case "=0", "zero":
		return pluralCount == 0
	case "=1", "one":
		return pluralCount == 1
	case "=2", "two":
		return pluralCount == 2
	default:
		// <5 or =5

		n, typ, ok := formAtoi(string(f))
		if !ok {
			return false
		}

		switch typ {
		case eq:
			return n == pluralCount
		case lt:
			return pluralCount < n
		case gt:
			return pluralCount > n
		default:
			return false
		}
	}
}

func makeSelectfVars(text string, vars []Var, insidePlural bool) ([]catalog.Message, []Var) {
	newVars := sortVars(text, vars)
	newVars = removeVarsDuplicates(newVars)
	msgs := selectfVars(newVars, insidePlural)
	return msgs, newVars
}

func selectfVars(vars []Var, insidePlural bool) []catalog.Message {
	msgs := make([]catalog.Message, 0, len(vars))
	for _, variable := range vars {
		argth := variable.Argth
		if insidePlural {
			argth++
		}

		msg := catalog.Var(variable.Name, plural.Selectf(argth, variable.Format, variable.Cases...))
		// fmt.Printf("%s:%d | cases | %#+v\n", variable.Name, variable.Argth, variable.Cases)
		msgs = append(msgs, msg)
	}

	return msgs
}

const (
	eq uint8 = iota + 1
	lt
	gt
)

func formType(ch byte) uint8 {
	switch ch {
	case '=':
		return eq
	case '<':
		return lt
	case '>':
		return gt
	}

	return 0
}

func formAtoi(form string) (int, uint8, bool) {
	if len(form) < 2 {
		return -1, 0, false
	}

	typ := formType(form[0])
	if typ == 0 {
		return -1, 0, false
	}

	dig, err := strconv.Atoi(form[1:])
	if err != nil {
		return -1, 0, false
	}
	return dig, typ, true
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
