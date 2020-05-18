package main

import (
	"time"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/sessions/sessiondb/badger"

	"github.com/kataras/iris/v12/_examples/sessions/overview/example"
)

func main() {
	db, err := badger.New("./data")
	if err != nil {
		panic(err)
	}

	// close and unlock the database when control+C/cmd+C pressed
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	defer db.Close() // close and unlock the database if application errored.

	// The default transcoder is the JSON one,
	// based on the https://golang.org/pkg/encoding/json/#Unmarshal
	// you can only retrieve numbers as float64 types:
	// * bool, for booleans
	// * float64, for numbers
	// * string, for strings
	// * []interface{}, for arrays
	// * map[string]interface{}, for objects.
	// If you want to save the data per go-specific types
	// you should change the DefaultTranscoder to the GobTranscoder one:
	// sessions.DefaultTranscoder = sessions.GobTranscoder{}

	sess := sessions.New(sessions.Config{
		Cookie:       "sessionscookieid",
		Expires:      1 * time.Minute, // <=0 means unlimited life. Defaults to 0.
		AllowReclaim: true,
	})

	sess.OnDestroy(func(sid string) {
		println(sid + " expired and destroyed from memory and its values from database")
	})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)

	app := example.NewApp(sess)
	app.Listen(":8080")
}
