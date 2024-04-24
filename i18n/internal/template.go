package internal

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"golang.org/x/text/message/catalog"
)

const (
	// VarsKey is the key for the message's variables, per locale(global) or per key (local).
	VarsKey = "Vars"
	// PluralCountKey is the key for the template's message pluralization.
	PluralCountKey = "PluralCount"
	// VarCountKeySuffix is the key suffix for the template's variable's pluralization,
	// e.g. HousesCount for ${Houses}.
	VarCountKeySuffix = "Count"
	// VarsKeySuffix is the key which the template message's variables
	// are stored with,
	// e.g. welcome.human.other_vars
	VarsKeySuffix = "_vars"
)

// Template is a Renderer which renders template messages.
type Template struct {
	*Message
	tmpl    *template.Template
	bufPool *sync.Pool
}

// NewTemplate returns a new Template message based on the
// catalog and the base translation Message. See `Locale.Load` method.
func NewTemplate(c *Catalog, m *Message) (*Template, error) {
	tmpl, err := template.New(m.Key).
		Delims(m.Locale.Options.Left, m.Locale.Options.Right).
		Funcs(m.Locale.FuncMap).
		Parse(m.Value)

	if err != nil {
		return nil, err
	}

	if err := registerTemplateVars(c, m); err != nil {
		return nil, fmt.Errorf("template vars: <%s = %s>: %w", m.Key, m.Value, err)
	}

	bufPool := &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	t := &Template{
		Message: m,
		tmpl:    tmpl,
		bufPool: bufPool,
	}

	return t, nil
}

func registerTemplateVars(c *Catalog, m *Message) error {
	if len(m.Vars) == 0 {
		return nil
	}

	msgs := selectfVars(m.Vars, false)

	variableText := ""

	for _, variable := range m.Vars {
		variableText += variable.Literal + " "
	}

	variableText = variableText[0 : len(variableText)-1]

	fullKey := m.Key + "." + VarsKeySuffix

	return c.Set(m.Locale.tag, fullKey, append(msgs, catalog.String(variableText))...)
}

// Render completes the Renderer interface.
// It renders a template message.
// Each key has its own Template, plurals too.
func (t *Template) Render(args ...interface{}) (string, error) {
	var (
		data   interface{}
		result string
	)

	argsLength := len(args)

	if argsLength > 0 {
		data = args[0]
	}

	buf := t.bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	if err := t.tmpl.Execute(buf, data); err != nil {
		t.bufPool.Put(buf)
		return "", err
	}

	result = buf.String()
	t.bufPool.Put(buf)

	if len(t.Vars) > 0 {
		// get the variables plurals.
		if argsLength > 1 {
			// if has more than the map/struct
			// then let's assume the user passes variable counts by raw integer arguments.
			args = args[1:]
		} else if data != nil {
			// otherwise try to resolve them by the map(%var_name%Count)/struct(PlrualCounter).
			args = findVarsCount(data, t.Vars)
		}
		result = t.replaceTmplVars(result, args...)
	}

	return result, nil
}

func findVarsCount(data interface{}, vars []Var) (args []interface{}) {
	if data == nil {
		return nil
	}

	switch dataValue := data.(type) {
	case PluralCounter:
		for _, v := range vars {
			if count := dataValue.VarCount(v.Name); count >= 0 {
				args = append(args, count)
			}
		}
	case Map:
		for _, v := range vars {
			varCountKey := v.Name + VarCountKeySuffix
			if value, ok := dataValue[varCountKey]; ok {
				args = append(args, value)
			}
		}
	case map[string]string:
		for _, v := range vars {
			varCountKey := v.Name + VarCountKeySuffix
			if value, ok := dataValue[varCountKey]; ok {
				if count, err := strconv.Atoi(value); err == nil {
					args = append(args, count)
				}
			}
		}
	case map[string]int:
		for _, v := range vars {
			varCountKey := v.Name + VarCountKeySuffix
			if value, ok := dataValue[varCountKey]; ok {
				args = append(args, value)
			}
		}
	default:
		return nil
	}

	return
}

func findPluralCount(data interface{}) (int, bool) {
	if data == nil {
		return -1, false
	}

	switch dataValue := data.(type) {
	case PluralCounter:
		if count := dataValue.PluralCount(); count >= 0 {
			return count, true
		}
	case Map:
		if v, ok := dataValue[PluralCountKey]; ok {
			if count, ok := v.(int); ok {
				return count, true
			}
		}
	case map[string]string:
		if v, ok := dataValue[PluralCountKey]; ok {
			count, err := strconv.Atoi(v)
			if err != nil {
				return -1, false
			}

			return count, true
		}

	case map[string]int:
		if count, ok := dataValue[PluralCountKey]; ok {
			return count, true
		}
	case int:
		return dataValue, true // when this is not a template data, the caller's argument should be args[1:] now.
	case int64:
		count := int(dataValue)
		return count, true
	}

	return -1, false
}

func (t *Template) replaceTmplVars(result string, args ...interface{}) string {
	varsKey := t.Key + "." + VarsKeySuffix
	translationVarsText := t.Locale.Printer.Sprintf(varsKey, args...)
	if translationVarsText != "" {
		translatioVars := strings.Split(translationVarsText, " ")
		for i, variable := range t.Vars {
			result = strings.Replace(result, variable.Literal, translatioVars[i], 1)
		}
	}

	return result
}

func stringIsTemplateValue(value, left, right string) bool {
	leftIdx, rightIdx := strings.Index(value, left), strings.Index(value, right)
	return leftIdx != -1 && rightIdx > leftIdx
}

func getFuncs(loc *Locale) template.FuncMap {
	// set the template funcs for this locale.
	funcs := template.FuncMap{
		"tr": loc.GetMessage,
	}

	if getFuncs := loc.Options.Funcs; getFuncs != nil {
		// set current locale's template's funcs.
		for k, v := range getFuncs(loc) {
			funcs[k] = v
		}
	}

	return funcs
}
