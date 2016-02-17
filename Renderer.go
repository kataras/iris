package iris

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"reflect"
	"strconv"
)

const (
	CHARSET        = "UTF-8"
	CONTENT_HTML   = "text/html" + "; " + CHARSET
	CONTENT_JSON   = "application/json" + "; " + CHARSET
	CONTENT_JSONP  = "application/javascript"
	CONTENT_BINARY = "application/octet-stream"
	CONTENT_LENGTH = "Content-Length"
	CONTENT_TEXT   = "text/plain" + "; " + CHARSET
	CONTENT_TYPE   = "Content-Type"
	CONTENT_XML    = "text/xml" + "; " + CHARSET
)

var rendererType reflect.Type

type Renderer struct {
	//Only one TemplateCache per app/router/iris instance.
	//and for now Renderer writer content-type  doesn't checks for methods (get,post...)
	templateCache  *TemplateCache
	responseWriter http.ResponseWriter
}

//Use at HTTPRoute.run
func NewRenderer(writer http.ResponseWriter) *Renderer {
	return &Renderer{responseWriter: writer}
}

func (r *Renderer) check() error {
	if r.templateCache == nil {
		return errors.New("iris:Error on Renderer : No Template Cache was created yet, please refer to docs at github.com/kataras/iris.")
	}
	return nil
}

func (r *Renderer) RenderFile(file string, pageContext interface{}) error {
	err := r.check()
	if err != nil {
		return err
	}

	return r.templateCache.ExecuteTemplate(r.responseWriter, file, pageContext)

}

func (r *Renderer) Render(pageContext interface{}) error {
	err := r.check()
	if err != nil {
		return err
	}
	return r.templateCache.Execute(r.responseWriter, pageContext)

}

///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (r *Renderer) WriteHTML(httpStatus int, htmlContents string) {
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_HTML)
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write([]byte(htmlContents))
}

func (r *Renderer) HTML(htmlContents string) {
	r.WriteHTML(http.StatusOK, htmlContents)
}

func (r *Renderer) WriteData(httpStatus int, binaryData []byte) {
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_BINARY)
	r.responseWriter.Header().Set(CONTENT_LENGTH, strconv.Itoa(len(binaryData)))
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write(binaryData)
}

func (r *Renderer) Data(binaryData []byte) {
	r.WriteData(http.StatusOK, binaryData)
}

func (r *Renderer) WriteText(httpStatus int, text string) {
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_TEXT)
	r.responseWriter.WriteHeader(httpStatus)
}

func (r *Renderer) Text(text string) {
	r.WriteText(http.StatusOK, text)
}
func (r *Renderer) WriteJSON(httpStatus int, jsonStructs ...interface{}) error {

	//	return json.NewEncoder(r.responseWriter).Encode(obj)
	var _json string
	for _, jsonStruct := range jsonStructs {
		theJson, err := json.MarshalIndent(jsonStruct, "", "  ")
		if err != nil {
			//http.Error(r.responseWriter, err.Error(), http.StatusInternalServerError)
			return err
		}
		_json += string(theJson)+"\n"
	}

	//keep in mind http.DetectContentType(data)
	//also we don't check if already header's content-type exists.
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_JSON)
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write([]byte(_json))
	return nil
}

func (r *Renderer) JSON(jsonStructs ...interface{}) error {
	return r.WriteJSON(http.StatusOK, jsonStructs)
}

///TODO:
func (r *Renderer) WriteJSONP(httpStatus int, obj interface{}) {
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_JSONP)
	r.responseWriter.WriteHeader(httpStatus)
}

func (r *Renderer) JSONP(obj interface{}) {
	r.WriteJSONP(http.StatusOK, obj)
}

func (r *Renderer) WriteXML(httpStatus int, xmlStructs ...interface{}) error {

	var _xmlDoc string
	for _, xmlStruct := range xmlStructs {
		theDoc, err := xml.MarshalIndent(xmlStruct, "", "  ")
		if err != nil {
			return err
		}
		_xmlDoc += string(theDoc)+"\n"
	}
	r.responseWriter.Header().Set(CONTENT_TYPE, CONTENT_XML)
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write([]byte(xml.Header + _xmlDoc))

	return nil
}

func (r *Renderer) XML(xmlStructs ...interface{}) error {
	return r.WriteXML(http.StatusOK, xmlStructs)
}