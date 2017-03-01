package iris

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/valyala/bytebufferpool"
)

// these are the default render policies for basic REST-type render for content types:
// - application/javascript (json)
// - text/javascript (jsonp)
// - text/xml (xml)
// - custom internal text/markdown -> text/html (markdown)

// the fastest buffer pool is maden by valyala, we use that because Iris should be fast at every step.
var buffer bytebufferpool.Pool

// some options-helpers here
func tryParseStringOption(options map[string]interface{}, key string, defValue string) string {
	if tryVal := options[key]; tryVal != nil {
		if val, ok := tryVal.(string); ok {
			return val
		}
	}
	return defValue
}

func tryParseBoolOption(options map[string]interface{}, key string, defValue bool) bool {
	if tryVal := options[key]; tryVal != nil {
		if val, ok := tryVal.(bool); ok {
			return val
		}
	}
	return defValue
}

func tryParseByteSliceOption(options map[string]interface{}, key string, defValue []byte) []byte {
	if tryVal := options[key]; tryVal != nil {
		if val, ok := tryVal.([]byte); ok {
			return val
		}
	}
	return defValue
}

//  +------------------------------------------------------------+
//  |                           JSON                             |
//  +------------------------------------------------------------+

var (
	newLineB = []byte("\n")
	// the html codes for unescaping
	ltHex = []byte("\\u003c")
	lt    = []byte("<")

	gtHex = []byte("\\u003e")
	gt    = []byte(">")

	andHex = []byte("\\u0026")
	and    = []byte("&")
)

// Let's no use map here and do a func which will do simple and fast if statements.
// var serializers = map[string]func(interface{}, ...map[string]interface{}) ([]byte, error){
// 	contentJSON:     serializeJSON,
// 	contentJSONP:    serializeJSONP,
// 	contentXML:      serializeXML,
// 	contentMarkdown: serializeMarkdown,
// }

var restRenderPolicy = RenderPolicy(func(out io.Writer, name string, val interface{}, options ...map[string]interface{}) (bool, error) {
	var (
		b   []byte
		err error
	)

	if name == contentJSON {
		b, err = serializeJSON(val, options...)
	} else if name == contentJSONP {
		b, err = serializeJSONP(val, options...)
	} else if name == contentXML {
		b, err = serializeXML(val, options...)
	} else if name == contentMarkdown {
		b, err = serializeMarkdown(val, options...)
	}

	if err != nil {
		return false, err // errors are wrapped
	}
	if len(b) > 0 {
		_, err = out.Write(b)
		return true, err
	}

	// continue to the next if any or notice there is no available renderer for that name
	return false, nil
})

// serializeJSON accepts the 'object' value and converts it to bytes in order to be json 'renderable'
func serializeJSON(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	// parse the options
	var (
		indent        bool
		unEscapeHTML  bool
		streamingJSON bool
		prefix        []byte
	)

	if options != nil && len(options) > 0 {
		opt := options[0]
		indent = tryParseBoolOption(opt, "indent", false)
		unEscapeHTML = tryParseBoolOption(opt, "unEscapeHTML", false)
		streamingJSON = tryParseBoolOption(opt, "streamingJSON", false)
		prefix = tryParseByteSliceOption(opt, "prefix", []byte(""))
	}

	// serialize the 'object'
	if streamingJSON {
		w := buffer.Get()
		if len(prefix) > 0 {
			w.Write(prefix)
		}
		err := json.NewEncoder(w).Encode(val)
		result := w.Bytes()
		buffer.Put(w)
		return result, err
	}

	var result []byte
	var err error

	if indent {
		result, err = json.MarshalIndent(val, "", "  ")
		result = append(result, newLineB...)
	} else {
		result, err = json.Marshal(val)
	}
	if err != nil {
		return nil, err
	}

	if unEscapeHTML {
		result = bytes.Replace(result, ltHex, lt, -1)
		result = bytes.Replace(result, gtHex, gt, -1)
		result = bytes.Replace(result, andHex, and, -1)
	}
	if len(prefix) > 0 {
		result = append(prefix, result...)
	}
	return result, nil
}

//  +------------------------------------------------------------+
//  |                           JSONP                            |
//  +------------------------------------------------------------+

var (
	finishCallbackB = []byte(");")
)

// serializeJSONP accepts the 'object' value and converts it to bytes in order to be jsonp 'renderable'
func serializeJSONP(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	// parse the options
	var (
		indent   bool
		callback string
	)
	if options != nil && len(options) > 0 {
		opt := options[0]
		indent = tryParseBoolOption(opt, "indent", false)
		callback = tryParseStringOption(opt, "callback", "")
	}

	var result []byte
	var err error

	if indent {
		result, err = json.MarshalIndent(val, "", "  ")
	} else {
		result, err = json.Marshal(val)
	}

	if err != nil {
		return nil, err
	}

	if callback != "" {
		result = append([]byte(callback+"("), result...)
		result = append(result, finishCallbackB...)
	}

	if indent {
		result = append(result, newLineB...)
	}
	return result, nil
}

//  +------------------------------------------------------------+
//  |                           XML                              |
//  +------------------------------------------------------------+

// serializeXML accepts the 'object' value and converts it to bytes in order to be xml 'renderable'
func serializeXML(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	// parse the options
	var (
		indent bool
		prefix []byte
	)
	if options != nil && len(options) > 0 {
		opt := options[0]
		indent = tryParseBoolOption(opt, "indent", false)
		prefix = tryParseByteSliceOption(opt, "prefix", []byte(""))
	}

	var result []byte
	var err error

	if indent {
		result, err = xml.MarshalIndent(val, "", "  ")
		result = append(result, '\n')
	} else {
		result, err = xml.Marshal(val)
	}
	if err != nil {
		return nil, err
	}
	if len(prefix) > 0 {
		result = append(prefix, result...)
	}
	return result, nil
}

//  +------------------------------------------------------------+
//  |                           MARKDOWN                         |
//  +------------------------------------------------------------+

// serializeMarkdown accepts the 'object' value and converts it to bytes in order to be markdown(text/html) 'renderable'
func serializeMarkdown(val interface{}, options ...map[string]interface{}) ([]byte, error) {

	// parse the options
	var (
		sanitize bool
	)
	if options != nil && len(options) > 0 {
		opt := options[0]
		sanitize = tryParseBoolOption(opt, "sanitize", false)
	}

	var b []byte
	if s, isString := val.(string); isString {
		b = []byte(s)
	} else {
		b = val.([]byte)
	}
	buf := blackfriday.MarkdownCommon(b)
	if sanitize {
		buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
	}

	return buf, nil
}
