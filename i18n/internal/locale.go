package internal

import (
	"fmt"
	"text/template"

	"github.com/kataras/iris/v12/context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

// Locale is the default Locale.
// Created by Catalog.
// One Locale maps to one registered and loaded language.
// Stores the translation variables and most importantly, the Messages (keys and their renderers).
type Locale struct {
	// The index of the language registered by the user, starting from zero.
	index int
	tag   language.Tag
	// ID is the tag.String().
	ID string
	// Options given by the Catalog
	Options Options

	// Fields set by Catalog.
	FuncMap template.FuncMap
	Printer *message.Printer
	//

	// Fields set by this Load method.
	Messages map[string]Renderer
	Vars     []Var // shared per-locale variables.
}

// Ensures that the Locale completes the context.Locale interface.
var _ context.Locale = (*Locale)(nil)

// Load sets the translation messages based on the Catalog's key values.
func (loc *Locale) Load(c *Catalog, keyValues Map) error {
	return loc.setMap(c, "", keyValues)
}

func (loc *Locale) setMap(c *Catalog, key string, keyValues Map) error {
	// unique locals or the shared ones.
	isRoot := key == ""

	vars := getVars(loc, VarsKey, keyValues)
	if isRoot {
		loc.Vars = vars
	} else {
		vars = removeVarsDuplicates(append(vars, loc.Vars...))
	}

	for k, v := range keyValues {
		form, isPlural := loc.Options.PluralFormDecoder(loc, k)
		if isPlural {
			k = key
		} else if !isRoot {
			k = key + "." + k
		}

		switch value := v.(type) {
		case string:
			if err := loc.setString(c, k, value, vars, form); err != nil {
				return fmt.Errorf("%s:%s parse string: %w", loc.ID, key, err)
			}
		case Map:
			// fmt.Printf("%s is map\n", fullKey)
			if err := loc.setMap(c, k, value); err != nil {
				return fmt.Errorf("%s:%s parse map: %w", loc.ID, key, err)
			}

		default:
			return fmt.Errorf("%s:%s unexpected type of %T as value", loc.ID, key, value)
		}
	}

	return nil
}

func (loc *Locale) setString(c *Catalog, key string, value string, vars []Var, form PluralForm) (err error) {
	isPlural := form != nil

	// fmt.Printf("setStringVars: %s=%s\n", key, value)
	msgs, vars := makeSelectfVars(value, vars, isPlural)
	msgs = append(msgs, catalog.String(value))

	m := &Message{
		Locale: loc,
		Key:    key,
		Value:  value,
		Vars:   vars,
		Plural: isPlural,
	}

	var (
		renderer, pluralRenderer Renderer = m, m
	)

	if stringIsTemplateValue(value, loc.Options.Left, loc.Options.Right) {
		t, err := NewTemplate(c, m)
		if err != nil {
			return err
		}

		pluralRenderer = t
		if !isPlural {
			renderer = t
		}
	} else {
		if isPlural {
			pluralRenderer, err = newIndependentPluralRenderer(c, loc, key, msgs...)
			if err != nil {
				return fmt.Errorf("<%s = %s>: %w", key, value, err)
			}
		} else if err = c.Set(loc.tag, key, msgs...); err != nil {
			// let's make normal keys direct fire:
			// renderer = &simpleRenderer{key, loc.Printer}
			return fmt.Errorf("<%s = %s>: %w", key, value, err)
		}

	}

	if isPlural {
		if existingMsg, ok := loc.Messages[key]; ok {
			if msg, ok := existingMsg.(*Message); ok {
				msg.AddPlural(form, pluralRenderer)
				return
			}
		}

		m.AddPlural(form, pluralRenderer)
	}

	loc.Messages[key] = renderer
	return
}

/* context.Locale interface */

// Index returns the current locale index from the languages list.
func (loc *Locale) Index() int {
	return loc.index
}

// Tag returns the full language Tag attached to this Locale,
// it should be unique across different Locales.
func (loc *Locale) Tag() *language.Tag {
	return &loc.tag
}

// Language should return the exact languagecode of this `Locale`
//that the user provided on `New` function.
//
// Same as `Tag().String()` but it's static.
func (loc *Locale) Language() string {
	return loc.ID
}

// GetMessage should return translated text based on the given "key".
func (loc *Locale) GetMessage(key string, args ...interface{}) string {
	if msg, ok := loc.Messages[key]; ok {
		result, err := msg.Render(args...)
		if err != nil {
			result = err.Error()
		}

		return result
	}

	return ""
}
