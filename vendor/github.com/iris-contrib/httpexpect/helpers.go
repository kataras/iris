package httpexpect

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/xeipuuv/gojsonschema"
	"github.com/yalp/jsonpath"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

func toString(str interface{}) (s string, ok bool) {
	ok = true
	defer func() {
		if err := recover(); err != nil {
			ok = false
		}
	}()
	s = reflect.ValueOf(str).Convert(reflect.TypeOf("")).String()
	return
}

func getPath(chain *chain, value interface{}, path string) *Value {
	if chain.failed() {
		return &Value{*chain, nil}
	}

	result, err := jsonpath.Read(value, path)
	if err != nil {
		chain.fail(err.Error())
		return &Value{*chain, nil}
	}

	return &Value{*chain, result}
}

func checkSchema(chain *chain, value, schema interface{}) {
	if chain.failed() {
		return
	}

	valueLoader := gojsonschema.NewGoLoader(value)

	var schemaLoader gojsonschema.JSONLoader

	if str, ok := toString(schema); ok {
		if ok, _ := regexp.MatchString(`^\w+://`, str); ok {
			schemaLoader = gojsonschema.NewReferenceLoader(str)
		} else {
			schemaLoader = gojsonschema.NewStringLoader(str)
		}
	} else {
		schemaLoader = gojsonschema.NewGoLoader(schema)
	}

	result, err := gojsonschema.Validate(schemaLoader, valueLoader)
	if err != nil {
		chain.fail("\n%s\n\nschema:\n%s\n\nvalue:\n%s",
			err.Error(),
			dumpSchema(schema),
			dumpValue(value))
		return
	}

	if !result.Valid() {
		errors := ""
		for _, err := range result.Errors() {
			errors += fmt.Sprintf(" %s\n", err)
		}

		chain.fail(
			"\njson schema validation failed, schema:\n%s\n\nvalue:%s\n\nerrors:\n%s",
			dumpSchema(schema),
			dumpValue(value),
			errors)

		return
	}
}

func dumpSchema(schema interface{}) string {
	if s, ok := toString(schema); ok {
		schema = s
	}
	return regexp.MustCompile(`(?m:^)`).
		ReplaceAllString(fmt.Sprintf("%v", schema), " ")
}

func canonNumber(chain *chain, number interface{}) (f float64, ok bool) {
	ok = true
	defer func() {
		if err := recover(); err != nil {
			chain.fail("%v", err)
			ok = false
		}
	}()
	f = reflect.ValueOf(number).Convert(reflect.TypeOf(float64(0))).Float()
	return
}

func canonArray(chain *chain, in interface{}) ([]interface{}, bool) {
	var out []interface{}
	data, ok := canonValue(chain, in)
	if ok {
		out, ok = data.([]interface{})
		if !ok {
			chain.fail("expected array, got %v", out)
		}
	}
	return out, ok
}

func canonMap(chain *chain, in interface{}) (map[string]interface{}, bool) {
	var out map[string]interface{}
	data, ok := canonValue(chain, in)
	if ok {
		out, ok = data.(map[string]interface{})
		if !ok {
			chain.fail("expected map, got %v", out)
		}
	}
	return out, ok
}

func canonValue(chain *chain, in interface{}) (interface{}, bool) {
	b, err := json.Marshal(in)
	if err != nil {
		chain.fail(err.Error())
		return nil, false
	}

	var out interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		chain.fail(err.Error())
		return nil, false
	}

	return out, true
}

func dumpValue(value interface{}) string {
	b, err := json.MarshalIndent(value, " ", "  ")
	if err != nil {
		return " " + fmt.Sprintf("%#v", value)
	}
	return " " + string(b)
}

func diffValues(expected, actual interface{}) string {
	differ := gojsondiff.New()

	var diff gojsondiff.Diff

	if ve, ok := expected.(map[string]interface{}); ok {
		if va, ok := actual.(map[string]interface{}); ok {
			diff = differ.CompareObjects(ve, va)
		} else {
			return " (unavailable)"
		}
	} else if ve, ok := expected.([]interface{}); ok {
		if va, ok := actual.([]interface{}); ok {
			diff = differ.CompareArrays(ve, va)
		} else {
			return " (unavailable)"
		}
	} else {
		return " (unavailable)"
	}

	config := formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
	}
	formatter := formatter.NewAsciiFormatter(expected, config)

	str, err := formatter.Format(diff)
	if err != nil {
		return " (unavailable)"
	}

	return "--- expected\n+++ actual\n" + str
}
