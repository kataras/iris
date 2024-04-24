package main

/*
$ go get gopkg.in/reform.v1/reform
$ go generate ./models
$ go run .

Read more at: https://github.com/go-reform/reform
*/

import (
	"database/sql"

	"myapp/controllers"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/sqlite3"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	sqlDB, err := sql.Open("sqlite3", "./myapp.db")
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()
	sqlStmt := `
	drop table people;
	create table people (id integer not null primary key, name text, email text, created_at datetime not null, updated_at datetime null);
	delete from people;
	`
	_, err = sqlDB.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}

	db := reform.NewDB(sqlDB, sqlite3.Dialect, reform.NewPrintfLogger(app.Logger().Debugf))

	mvcApp := mvc.New(app.Party("/persons"))
	mvcApp.Register(db)
	mvcApp.Handle(new(controllers.PersonController))

	app.Listen(":8080")
}
