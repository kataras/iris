package internal

import (
	"reflect"
	"regexp"
	"sort"

	"golang.org/x/text/message/catalog"
)

// Var represents a message variable.
// The variables, like the sub messages are sorted.
// First: plurals (which again, are sorted)
// and then any custom keys.
// In variables, the sorting depends on the exact
// order the associated message uses the variables.
// This is extremely handy.
// This package requires the golang.org/x/text/message capabilities
// only for the variables feature, the message itself's pluralization is managed by the package.
type Var struct {
	Name    string        // Variable name, e.g. Name
	Literal string        // Its literal is ${Name}
	Cases   []interface{} // one:...,few:...,...
	Format  string        // defaults to "%d".
	Argth   int           // 1, 2, 3...
}

func getVars(loc *Locale, key string, src map[string]interface{}) []Var {
	if len(src) == 0 {
		return nil
	}

	varsKey, ok := src[key]
	if !ok {
		return nil
	}

	varValue, ok := varsKey.([]interface{})
	if !ok {
		return nil
	}

	vars := make([]Var, 0, len(varValue))

	for _, v := range varValue {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		for k, inner := range m {
			varFormat := "%d"

			innerMap, ok := inner.(map[string]interface{})
			if !ok {
				continue
			}

			for kk, vv := range innerMap {
				if kk == "format" {
					if format, ok := vv.(string); ok {
						varFormat = format
					}
					break
				}
			}

			cases := getCases(loc, innerMap)

			if len(cases) > 0 {
				// cases = sortCases(cases)
				vars = append(vars, Var{
					Name:    k,
					Literal: "${" + k + "}",
					Cases:   cases,
					Format:  varFormat,
					Argth:   1,
				})
			}
		}
	}

	delete(src, key) // delete the key after.
	return vars
}

var unescapeVariableRegex = regexp.MustCompile("\\$\\{(.*?)}")

func sortVars(text string, vars []Var) (newVars []Var) {
	argth := 1
	for _, submatches := range unescapeVariableRegex.FindAllStringSubmatch(text, -1) {
		name := submatches[1]
		for _, variable := range vars {
			if variable.Name == name {
				variable.Argth = argth
				newVars = append(newVars, variable)
				argth++
				break
			}
		}
	}

	sort.SliceStable(newVars, func(i, j int) bool {
		return newVars[i].Argth < newVars[j].Argth
	})
	return
}

// it will panic if the incoming "elements" are not catmsg.Var (internal text package).
func removeVarsDuplicates(elements []Var) (result []Var) {
	seen := make(map[string]struct{})

	for v := range elements {
		variable := elements[v]
		name := variable.Name
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			result = append(result, variable)
		}
	}

	return result
}

func removeMsgVarsDuplicates(elements []catalog.Message) (result []catalog.Message) {
	seen := make(map[string]struct{})

	for _, elem := range elements {
		val := reflect.Indirect(reflect.ValueOf(elem))
		if val.Type().String() != "catmsg.Var" {
			// keep.
			result = append(result, elem)
			continue // it's not a var.
		}
		name := val.FieldByName("Name").Interface().(string)
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			result = append(result, elem)
		}
	}

	return
}

func getCases(loc *Locale, src map[string]interface{}) []interface{} {
	type PluralCase struct {
		Form  PluralForm
		Value interface{}
	}

	pluralCases := make([]PluralCase, 0, len(src))

	for key, value := range src {
		form, ok := loc.Options.PluralFormDecoder(loc, key)
		if !ok {
			continue
		}

		pluralCases = append(pluralCases, PluralCase{
			Form:  form,
			Value: value,
		})
	}

	if len(pluralCases) == 0 {
		return nil
	}

	sort.SliceStable(pluralCases, func(i, j int) bool {
		left, right := pluralCases[i].Form, pluralCases[j].Form
		return left.Less(right)
	})

	cases := make([]interface{}, 0, len(pluralCases)*2)
	for _, pluralCase := range pluralCases {
		// fmt.Printf("%s=%v\n", pluralCase.Form, pluralCase.Value)
		cases = append(cases, pluralCase.Form.String())
		cases = append(cases, pluralCase.Value)
	}

	return cases
}
