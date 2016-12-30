// +build ignore

package main

import (
	"fmt"
	"github.com/kataras/go-sessions"
	"github.com/kataras/go-sessions/sessiondb/redis"
	"github.com/kataras/go-sessions/sessiondb/redis/service"
	"net/http"
	"time"
)

var mySessionsConfig = sessions.Config{Cookie: "mysessioncookieid",
	DecodeCookie:                false,
	Expires:                     time.Duration(2) * time.Hour,
	GcDuration:                  time.Duration(2) * time.Hour,
	DisableSubdomainPersistence: false,
}

var mySessions = sessions.New(mySessionsConfig)

func main() {
	db := redis.New(service.Config{Network: service.DefaultRedisNetwork,
		Addr:          service.DefaultRedisAddr,
		Password:      "",
		Database:      "",
		MaxIdle:       0,
		MaxActive:     0,
		IdleTimeout:   service.DefaultRedisIdleTimeout,
		Prefix:        "",
		MaxAgeSeconds: service.DefaultRedisMaxAgeSeconds}) // optionally configure the bridge between your redis server

	// register the database, which will load from redis to the memory and update the redis database from memory when a session is updated
	// you can use unlimited number of databases, same type or other type.
	mySessions.UseDatabase(db)

	// set some values to the session
	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := mySessions.Start(res, req) // init the session
		// mySessions.Start returns:
		// type Session interface {
		//  ID() string
		//	Get(string) interface{}
		//	GetString(key string) string
		//	GetInt(key string) int
		//	GetAll() map[string]interface{}
		//	VisitAll(cb func(k string, v interface{}))
		//	Set(string, interface{})
		//	Delete(string)
		//	Clear()
		//}
		for k, v := range values {
			sess.Set(k, v) // fill session, set each of the key-value pair
		}
		res.Write([]byte("Session saved, go to /get to view the results"))
	})
	http.Handle("/set/", setHandler)

	// get the values from the session
	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := mySessions.Start(res, req) // init the session
		sessValues := sess.GetAll()        // get all values from this session

		res.Write([]byte(fmt.Sprintf("%#v", sessValues)))
	})
	http.Handle("/get/", getHandler)

	// clear all values from the session
	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := mySessions.Start(res, req) // init the session
		sess.Clear()
	})
	http.Handle("/clear/", clearHandler)

	// destroys the session, clears the values and removes the server-side entry and client-side sessionid cookie
	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		mySessions.Destroy(res, req)
	})
	http.Handle("/destroy/", destroyHandler)

	fmt.Println("Start your redis server first\nOpen a browser tab and navigate to the localhost:8080/set/\n See your redis database getting update ")
	http.ListenAndServe(":8080", nil)
}
