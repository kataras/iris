// Package serializer helps GoLang Developers to serialize any custom type to []byte or string.
// Your custom serializers are finally, organised.
//
// Built'n supported serializers: JSON, JSONP, XML, Markdown.
//
// This package is already used by Iris web framework.
package serializer

import (
	"strings"
	"sync"

	"github.com/kataras/go-errors"
	"github.com/kataras/go-serializer/json"
	"github.com/kataras/go-serializer/jsonp"
	"github.com/kataras/go-serializer/markdown"
	"github.com/kataras/go-serializer/xml"
)

const (
	// Version current version number
	Version = "0.0.5"
)

type (
	// Serializer is the interface which all serializers should implement
	Serializer interface {
		// Serialize accepts an object with serialization options and returns its bytes representation
		Serialize(interface{}, ...map[string]interface{}) ([]byte, error)
	}
	// SerializeFunc is the alternative way to implement a Serializer using a simple function
	SerializeFunc func(interface{}, ...map[string]interface{}) ([]byte, error)
)

// Serialize accepts an object with serialization options and returns its bytes representation
func (s SerializeFunc) Serialize(obj interface{}, options ...map[string]interface{}) ([]byte, error) {
	return s(obj, options...)
}

// NotAllowedKeyChar the rune which is not allowed to be inside a serializer key string
// this exists because almost all package's users will use kataras/go-template with kataras/go-serializer
// in one method, so we need something to tell if the 'renderer' wants to render a
// serializer's result or the template's result, you don't have to worry about these things.
const NotAllowedKeyChar = '.'

// Serializers is optionally, used when your app needs to manage more than one serializer
// keeps a map with a key of the ContentType(or any string) and a collection of Serializers
//
// if a ContentType(key) has more than one serializer registered to it then the final result will be all of the serializer's results combined
type Serializers map[string][]Serializer

var defaultSerializers = Serializers{}

// For puts a serializer(s) to the map
func For(key string, serializer ...Serializer) {
	defaultSerializers.For(key, serializer...)
}

// For puts a serializer(s) to the map
func (s Serializers) For(key string, serializer ...Serializer) {
	if s == nil {
		s = make(map[string][]Serializer)
	}

	if strings.IndexByte(key, NotAllowedKeyChar) != -1 {
		return
	}

	if s[key] == nil {
		s[key] = make([]Serializer, 0)
	}

	s[key] = append(s[key], serializer...)
}

var (
	errKeyMissing         = errors.New("please specify a key")
	errSerializersEmpty   = errors.New("serializers map is empty")
	errSerializerNotFound = errors.New("serializer with key '%s' couldn't be found")
)

var (
	once                   sync.Once
	defaultSerializersKeys = [...]string{json.ContentType, jsonp.ContentType, xml.ContentType, markdown.ContentType}
)

// RegisterDefaults register defaults serializer for each of the default serializer keys (data,json,jsonp,markdown,text,xml)
func RegisterDefaults(serializers Serializers) {
	for _, ctype := range defaultSerializersKeys {

		if sers := serializers[ctype]; len(sers) == 0 {
			// if not exists
			switch ctype {
			case json.ContentType:
				serializers.For(ctype, json.New())
			case jsonp.ContentType:
				serializers.For(ctype, jsonp.New())
			case xml.ContentType:
				serializers.For(ctype, xml.New())
			case markdown.ContentType:
				serializers.For(ctype, markdown.New())
			}
		}
	}
}

// Serialize returns the result as bytes representation of the serializer(s)
func Serialize(key string, obj interface{}, options ...map[string]interface{}) ([]byte, error) {
	// only to the default serializer, check if no of the built'n serializer's key are registered, if not register them here
	// I don't put it to the initialize of the defaultSerializers because the developer may don't want the built'n serializers at all
	once.Do(func() {
		RegisterDefaults(defaultSerializers)
	})

	return defaultSerializers.Serialize(key, obj, options...)
}

// Serialize returns the result as bytes representation of the serializer(s)
func (s Serializers) Serialize(key string, obj interface{}, options ...map[string]interface{}) ([]byte, error) {
	if key == "" {
		return nil, errKeyMissing
	}
	if s == nil {
		return nil, errSerializersEmpty
	}

	serializers := s[key]
	if serializers == nil {
		return nil, errSerializerNotFound.Format(key)
	}
	var finalResult []byte

	for i, n := 0, len(serializers); i < n; i++ {
		result, err := serializers[i].Serialize(obj, options...)
		if err != nil {
			return nil, err
		}
		finalResult = append(finalResult, result...)
	}
	return finalResult, nil
}

// SerializeToString returns the string representation of the serializer(s)
// same as Serialize but returns string
func SerializeToString(key string, obj interface{}, options ...map[string]interface{}) (string, error) {
	return defaultSerializers.SerializeToString(key, obj, options...)
}

// SerializeToString returns the string representation of the serializer(s)
// same as Serialize but returns string
func (s Serializers) SerializeToString(key string, obj interface{}, options ...map[string]interface{}) (string, error) {
	result, err := s.Serialize(key, obj, options...)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// Len returns the length of the serializers map
func Len() int {
	return defaultSerializers.Len()
}

// Len returns the length of the serializers map
func (s Serializers) Len() int {
	if s == nil {
		return 0
	}

	return len(s)
}

// Options is just a shortcut of a map[string]interface{}, which can be passed to the Serialize/SerializeToString funcs
type Options map[string]interface{}
