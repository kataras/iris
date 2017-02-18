package jsonp

import (
	"encoding/json"
)

const (
	// ContentType the custom key for the serializer, when used inside iris, Q web frameworks or simply net/http
	ContentType = "application/javascript"
)

// Serializer the serializer which renders a JSONP 'object' with its callback
type Serializer struct {
	config Config
}

// New returns a new jsonp serializer
func New(cfg ...Config) *Serializer {
	c := DefaultConfig().Merge(cfg)
	return &Serializer{config: c}
}

func (e *Serializer) getCallbackOption(options map[string]interface{}) string {
	callbackOpt := options["callback"]
	if s, isString := callbackOpt.(string); isString {
		return s
	}
	return e.config.Callback
}

var (
	finishCallbackB = []byte(");")
	newLineB        = []byte("\n")
)

// Serialize accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the go-serializer.Serializer interface
func (e *Serializer) Serialize(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	var result []byte
	var err error
	if e.config.Indent {
		result, err = json.MarshalIndent(val, "", "  ")
	} else {
		result, err = json.Marshal(val)
	}

	if err != nil {
		return nil, err
	}

	// the config's callback can be overriden with the options
	callback := e.config.Callback
	if len(options) > 0 {
		callback = e.getCallbackOption(options[0])
	}

	if callback != "" {
		result = append([]byte(callback+"("), result...)
		result = append(result, finishCallbackB...)
	}

	if e.config.Indent {
		result = append(result, newLineB...)
	}
	return result, nil
}
