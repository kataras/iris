package iris

import (
	"encoding/json"
	"encoding/xml"
	"html/template"
	"io"
	"net/http"
	"strconv"
)

const (
	// DefaultCharset represents the default charset for content headers
	DefaultCharset = "UTF-8"
	// ContentType represents the header["Content-Type"]
	ContentType = "Content-Type"
	// ContentLength represents the header["Content-Length"]
	ContentLength = "Content-Length"
	// ContentHTML is the  string of text/html response headers
	ContentHTML = "text/html" + "; " + DefaultCharset
	// ContentJSON is the  string of application/json response headers
	ContentJSON = "application/json" + "; " + DefaultCharset
	// ContentJSONP is the  string of application/javascript response headers
	ContentJSONP = "application/javascript"
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
	// ContentTEXT is the  string of text/plain response headers
	ContentTEXT = "text/plain" + "; " + DefaultCharset
	// ContentXML is the  string of text/xml response headers
	ContentXML = "text/xml" + "; " + DefaultCharset
)

// Renderer is the container of the template cache which developer creates for EACH route
type Renderer struct {
	//Only one TemplateCache per app/router/iris instance.
	//and for now Renderer writer content-type  doesn't checks for methods (get,post...)
	templates      *template.Template
	responseWriter http.ResponseWriter
}

// RenderFile renders a file by its path and a context passed to the function
func (r *Renderer) RenderFile(file string, pageContext interface{}) error {
	return r.templates.ExecuteTemplate(r.responseWriter, file, pageContext)

}

// Render renders the template file html which is already registed to the template cache, with it's pageContext passed to the function
func (r *Renderer) Render(pageContext interface{}) error {
	return r.templates.Execute(r.responseWriter, pageContext)

}

// WriteHTML writes html string with a http status
///TODO or I will think to pass an interface on handlers as second parameter near to the Context, with developer's custom Renderer package .. I will think about it.
func (r *Renderer) WriteHTML(httpStatus int, htmlContents string) {
	r.responseWriter.Header().Set(ContentType, ContentHTML)
	r.responseWriter.WriteHeader(httpStatus)
	io.WriteString(r.responseWriter, htmlContents)
}

//HTML calls the WriteHTML with the 200 http status ok
func (r *Renderer) HTML(htmlContents string) {
	r.WriteHTML(http.StatusOK, htmlContents)
}

// WriteData writes binary data with a http status
func (r *Renderer) WriteData(httpStatus int, binaryData []byte) {
	r.responseWriter.Header().Set(ContentType, ContentBINARY)
	r.responseWriter.Header().Set(ContentLength, strconv.Itoa(len(binaryData)))
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write(binaryData)
}

//Data calls the WriteData with the 200 http status ok
func (r *Renderer) Data(binaryData []byte) {
	r.WriteData(http.StatusOK, binaryData)
}

// WriteText writes text with a http status
func (r *Renderer) WriteText(httpStatus int, text string) {
	r.responseWriter.Header().Set(ContentType, ContentTEXT)
	r.responseWriter.WriteHeader(httpStatus)
	io.WriteString(r.responseWriter, text)
}

//Text calls the WriteText with the 200 http status ok
func (r *Renderer) Text(text string) {
	r.WriteText(http.StatusOK, text)
}

// WriteJSON writes which is converted from struct(s) with a http status which they passed to the function via parameters
func (r *Renderer) WriteJSON(httpStatus int, jsonStructs ...interface{}) error {
	var _json []byte

	for _, jsonStruct := range jsonStructs {

		theJSON, err := json.MarshalIndent(jsonStruct, "", "  ")
		if err != nil {
			return err
		}
		_json = append(_json, theJSON...)
	}

	//keep in mind http.DetectContentType(data)
	r.responseWriter.Header().Set(ContentType, ContentJSON)
	r.responseWriter.WriteHeader(httpStatus)
	r.responseWriter.Write(_json)

	return nil
}

//JSON calls the WriteJSON with the 200 http status ok
func (r *Renderer) JSON(jsonStructs ...interface{}) error {
	return r.WriteJSON(http.StatusOK, jsonStructs...)
}

// WriteXML writes xml which is converted from struct(s) with a http status which they passed to the function via parameters
func (r *Renderer) WriteXML(httpStatus int, xmlStructs ...interface{}) error {
	var _xmlDoc string
	for _, xmlStruct := range xmlStructs {
		theDoc, err := xml.MarshalIndent(xmlStruct, "", "  ")
		if err != nil {
			return err
		}
		_xmlDoc += string(theDoc) + "\n"
	}
	r.responseWriter.Header().Set(ContentType, ContentXML)
	r.responseWriter.WriteHeader(httpStatus)
	io.WriteString(r.responseWriter, xml.Header+_xmlDoc)
	return nil
}

//XML calls the WriteXML with the 200 http status ok
func (r *Renderer) XML(xmlStructs ...interface{}) error {
	return r.WriteXML(http.StatusOK, xmlStructs...)
}
