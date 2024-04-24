// Package main shows an example of pug actions based on https://github.com/Joker/jade/tree/master/example/actions
package main

import "github.com/kataras/iris/v12"

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

func main() {
	app := iris.New()

	tmpl := iris.Pug("./templates", ".pug")
	app.RegisterView(tmpl)

	app.Get("/", index)

	// http://localhost:8080
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	job1 := Job{Employer: "Monash B", Role: "Honorary"}
	job2 := Job{Employer: "Box Hill", Role: "Head of HE"}

	person := Person{
		Name:   "jan",
		Age:    50,
		Emails: []string{"jan@newmarch.name", "jan.newmarch@gmail.com"},
		Jobs:   []*Job{&job1, &job2},
	}

	if err := ctx.View("index.pug", person); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
