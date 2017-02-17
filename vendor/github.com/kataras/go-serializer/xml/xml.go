package xml

import (
	"encoding/xml"
)

const (
	// ContentType the custom key for the serializer, when used inside iris, Q web frameworks or simply net/http
	ContentType = "text/xml"
)

// Serializer the serializer which renders an XML 'object'
type Serializer struct {
	config Config
}

// New returns a new xml serializer
func New(cfg ...Config) *Serializer {
	c := DefaultConfig().Merge(cfg)
	return &Serializer{config: c}
}

// Serialize accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the go-serializer.Serializer interface
func (e *Serializer) Serialize(val interface{}, options ...map[string]interface{}) ([]byte, error) {

	var result []byte
	var err error

	if e.config.Indent {
		result, err = xml.MarshalIndent(val, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = xml.Marshal(val)
	}
	if err != nil {
		return nil, err
	}
	if len(e.config.Prefix) > 0 {
		result = append(e.config.Prefix, result...)
	}
	return result, nil
}
