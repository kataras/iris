package iris

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"
)

var templatesDirectory string

func getCurrentDir() string {
	pwd, err := os.Getwd()
	if err != nil {
		println("Something wrong to your executable path")
		os.Exit(1)
	}
	return pwd
}

// SetTemplatesDirectory sets the templatesDirectory global variable
func SetTemplatesDirectory(dir string) {
	templatesDirectory = dir
}

// AppendTemplatesDirectory appends with the dir parameter the templatesDirectory global variable
func AppendTemplatesDirectory(dir string) {
	templatesDirectory = path.Join(templatesDirectory, dir)
}

// TemplateCache is the cache of each Renderer object which is created at the request time on the route.run.
//
// TemplateCache contains the templates.
// Use on server.go
type TemplateCache struct {
	templates     *template.Template
	filesTemp     []string
	filesGlobTemp string
}

// NewTemplateCache creates and returns an empty template cache
func NewTemplateCache() *TemplateCache {
	tc := &TemplateCache{filesTemp: make([]string, 0)}

	return tc
}

// Add files to the temp files
func (tc *TemplateCache) Add(files ...string) {
	for i := 0; i < len(files); i++ {
		files[0] = path.Join(templatesDirectory, files[0])
	}
	tc.filesTemp = append(tc.filesTemp, files...)
}

// SetGlob sets filesGlobTemp using a  str regex pattern
func (tc *TemplateCache) SetGlob(filesPattern string) {
	tc.filesGlobTemp = path.Join(templatesDirectory, filesPattern)
}

// template creates if not already exists, and returns the templates, resets the filesTemp and filesGlobTemp.
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

// ExecuteTemplate executes a template on a given writer with it's filename which is provided via parameters
func (tc *TemplateCache) ExecuteTemplate(writer http.ResponseWriter, fileTmplPath string, page interface{}) error {
	if !strings.HasSuffix(fileTmplPath, ".html") {
		fileTmplPath += ".html"
	}
	return tc.template().ExecuteTemplate(writer, path.Join(templatesDirectory, fileTmplPath), page)
}

// Execute executes the given templates (assuming that it's only one html file inside) on a given writer
func (tc *TemplateCache) Execute(writer http.ResponseWriter, page interface{}) error {
	return tc.template().Execute(writer, page)
}
