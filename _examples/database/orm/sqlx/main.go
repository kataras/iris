package main

import (
	"github.com/kataras/iris/v12"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

/*
	go get -u github.com/mattn/go-sqlite3
	go get -u github.com/jmoiron/sqlx

	If you're on win64 and you can't install go-sqlite3:
		1. Download: https://sourceforge.net/projects/mingw-w64/files/latest/download
		2. Select "x86_x64" and "posix"
		3. Add C:\Program Files\mingw-w64\x86_64-7.1.0-posix-seh-rt_v5-rev1\mingw64\bin
		to your PATH env variable.

	Docs: https://github.com/jmoiron/sqlx
*/

// Person is our person table structure.
type Person struct {
	ID        int64  `db:"person_id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string
}

const schema = `
CREATE TABLE IF NOT EXISTS person (
	person_id INTEGER PRIMARY KEY,
	first_name text,
	last_name text,
	email text
);`

func main() {
	app := iris.New()

	db, err := sqlx.Connect("sqlite3", "./test.db")
	if err != nil {
		app.Logger().Fatalf("db failed to initialized: %v", err)
	}
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	db.MustExec(schema)

	app.Get("/insert", func(ctx iris.Context) {
		res, err := db.NamedExec(`INSERT INTO person (first_name,last_name,email) VALUES (:first,:last,:email)`,
			map[string]interface{}{
				"first": "John",
				"last":  "Doe",
				"email": "johndoe@example.com",
			})

		if err != nil {
			// Note: on production, don't give the error back to the user.
			// However for the sake of the example we do:
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		id, err := res.LastInsertId()
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("person inserted: id: %d", id)
	})

	app.Get("/get", func(ctx iris.Context) {
		// Select all persons.
		people := []Person{}
		db.Select(&people, "SELECT * FROM person ORDER BY first_name ASC")
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		if len(people) == 0 {
			ctx.Writef("no persons found, use /insert first.")
			return
		}

		ctx.Writef("persons found: %#v", people)
		/* Select a single or more with a first name of John from the database:
		person := Person{FirstName: "John"}
		rows, err := db.NamedQuery(`SELECT * FROM person WHERE first_name=:first_name`, person)
		if err != nil { ... }
		defer rows.Close()
		for rows.Next() {
			if err := rows.StructScan(&person); err != nil {
				if err == sql.ErrNoRows {
					ctx.StopWithText(iris.StatusNotFound, "Person: %s not found", person.FirstName)
				} else {
					ctx.StopWithError(iris.StatusInternalServerError, err)
				}

				return
			}
		}
		*/
	})

	// http://localhost:8080/insert
	// http://localhost:8080/get
	app.Listen(":8080")
}
