package main

import (
	"github.com/satori/go.uuid"
	"net/url"
)

// Generator the type to generate keys(short urls)
type Generator func() string

// DefaultGenerator is the defautl url generator
var DefaultGenerator = func() string {
	return uuid.NewV4().String()
}

// Factory is responsible to generate keys(short urls)
type Factory struct {
	store     Store
	generator Generator
}

// NewFactory receives a generator and a store and returns a new url Factory.
func NewFactory(generator Generator, store Store) *Factory {
	return &Factory{
		store:     store,
		generator: generator,
	}
}

// Gen generates the key.
func (f *Factory) Gen(uri string) (key string, err error) {
	// we don't return the parsed url because #hash are converted to uri-compatible
	// and we don't want to encode/decode all the time, there is no need for that,
	// we save the url as the user expects if the uri validation passed.
	_, err = url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}

	key = f.generator()
	// Make sure that the key is unique
	for {
		if v := f.store.Get(key); v == "" {
			break
		}
		key = f.generator()
	}

	return key, nil
}
