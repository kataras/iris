package main

import (
	"os"
	"time"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/sessions/sessiondb/boltdb"

	"github.com/kataras/iris/v12/_examples/sessions/overview/example"
)

func main() {
	db, err := boltdb.New("./sessions.db", os.FileMode(0750))
	if err != nil {
		panic(err)
	}

	// close and unlobkc the database when control+C/cmd+C pressed
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	defer db.Close() // close and unlock the database if application errored.
	sess := sessions.New(sessions.Config{
		Cookie:       "sessionscookieid",
		Expires:      45 * time.Minute, // <=0 means unlimited life. Defaults to 0.
		AllowReclaim: true,
	})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	// The default database's values encoder and decoder
	// calls the value's `Marshal/Unmarshal` methods (if any)
	// otherwise JSON is selected,
	// the JSON format can be stored to any database and
	// it supports both builtin language types(e.g. string, int) and custom struct values.
	// Also, and the most important, the values can be
	// retrieved/logged/monitored by a third-party program
	// written in any other language as well.
	//
	// You can change this behavior by registering a custom `Transcoder`.
	// Iris provides a `GobTranscoder` which is mostly suitable
	// if your session values are going to be custom Go structs.
	// Select this if you always retrieving values through Go.
	// Don't forget to initialize a call of gob.Register when necessary.
	// Read https://golang.org/pkg/encoding/gob/ for more.
	//
	// You can also implement your own `sessions.Transcoder` and use it,
	// i.e: a transcoder which will allow(on Marshal: return its byte representation and nil error)
	// or dissalow(on Marshal: return non nil error) certain types.
	//
	// gob.Register(example.BusinessModel{})
	// sessions.DefaultTranscoder = sessions.GobTranscoder{}

	app := example.NewApp(sess)
	app.Listen(":8080")
}
