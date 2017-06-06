package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"

	"github.com/Joker/jade"
)

type Person struct {
	Name   string
	Age    int
	Emails []string
	Jobs   []*Job
}

type Job struct {
	Employer string
	Role     string
}

func handler(w http.ResponseWriter, r *http.Request) {

	buf, err := ioutil.ReadFile("template.jade")
	if err != nil {
		fmt.Printf("\nReadFile error: %v", err)
		return
	}
	jadeTpl, err := jade.Parse("jade_tp", string(buf))
	if err != nil {
		fmt.Printf("\nParse error: %v", err)
		return
	}
	fmt.Printf("%s", jadeTpl)

	//

	job1 := Job{Employer: "Monash B", Role: "Honorary"}
	job2 := Job{Employer: "Box Hill", Role: "Head of HE"}

	person := Person{
		Name:   "jan",
		Age:    50,
		Emails: []string{"jan@newmarch.name", "jan.newmarch@gmail.com"},
		Jobs:   []*Job{&job1, &job2},
	}

	//

	goTpl, err := template.New("html").Parse(jadeTpl)
	if err != nil {
		fmt.Printf("\nTemplate parse error: %v", err)
		return
	}
	err = goTpl.Execute(w, person)
	if err != nil {
		fmt.Printf("\nExecute error: %v", err)
		return
	}
}

func main() {
	fmt.Println("open  http://localhost:8080/")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
