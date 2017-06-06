package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/Joker/jade"
)

func handler(w http.ResponseWriter, r *http.Request) {
	jade_tpl, err := jade.ParseFile("template.jade")
	if err != nil {
		log.Printf("\nParseFile error: %v", err)
	}
	log.Printf("%s\n\n", jade_tpl)

	//

	funcMap := template.FuncMap{
		"include": func(includePath string) (template.HTML, error) {
			include_tpl, err := jade.ParseFile(includePath)
			if err != nil {
				log.Printf("\nParseFile error: %v", err)
			}
			log.Printf("%s\n\n", include_tpl)

			go_partial_tpl, _ := template.New("partial").Parse(include_tpl)

			buf := new(bytes.Buffer)
			go_partial_tpl.Execute(buf, "")
			return template.HTML(buf.String()), nil

		},
		"bold": func(content string) (template.HTML, error) {
			return template.HTML("<b>" + content + "</b>"), nil
		},
	}

	//

	go_tpl, err := template.New("html").Funcs(funcMap).Parse(jade_tpl)
	if err != nil {
		log.Printf("\nTemplate parse error: %v", err)
	}

	err = go_tpl.Execute(w, "")
	if err != nil {
		log.Printf("\nExecute error: %v", err)
	}
}

func js(w http.ResponseWriter, r *http.Request) {}

func main() {
	log.Println("open  http://localhost:8080/")
	http.HandleFunc("/javascripts/", js)
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
