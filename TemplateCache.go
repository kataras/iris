package gapi

import (
	"html/template"
	"net/http"
	"strings"
)
/* Use on HTTPServer */
type TemplateCache struct {
	templates     *template.Template
	filesTemp     []string
	filesGlobTemp string
}

func NewTemplateCache() *TemplateCache {
	tc := &TemplateCache{filesTemp: make([]string, 0)}

	return tc
}

func (tc *TemplateCache) Add(files ...string) {
	tc.filesTemp = append(tc.filesTemp, files...)
}

func (tc *TemplateCache) SetGlob(filesPattern string) {
	tc.filesGlobTemp = filesPattern
}

func (tc *TemplateCache) template() *template.Template {
	if tc.templates == nil {
		if len(tc.filesTemp) > 0 {
			tc.templates = template.Must(template.ParseFiles(tc.filesTemp...))
		}

		if tc.filesGlobTemp != "" {
			if tc.templates == nil {
				//no filesTemp too
				tc.templates = template.Must(template.ParseGlob(tc.filesGlobTemp))
			} else {
				tc.templates.ParseGlob(tc.filesGlobTemp)
			}
		}
		
		tc.filesTemp = nil
		tc.filesGlobTemp = ""

	}

	return tc.templates
}

func (tc *TemplateCache) ExecuteTemplate(writer http.ResponseWriter, fileTmplPath string, page interface{}) error {
	if !strings.HasSuffix(fileTmplPath, ".html") {
		fileTmplPath += ".html"
	}
	return tc.template().ExecuteTemplate(writer, fileTmplPath, page)
}
