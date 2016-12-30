// +build ignore

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kataras/go-sessions"
	"github.com/kataras/go-sessions/sessiondb/leveldb"
)

var mySessionsConfig = sessions.Config{Cookie: "mysessioncookieid",
	DecodeCookie:                false,
	Expires:                     time.Duration(2) * time.Hour,
	GcDuration:                  time.Duration(2) * time.Hour,
	DisableSubdomainPersistence: false,
}

var mySessions = sessions.New(mySessionsConfig)

func main() {
	Main()
}

func Main() {
	db := leveldb.New(leveldb.Config{
		Path: "dbpath",
		//		MaxAge:       time.Second * 15, // 15 seconds for test auto clean old session
		//		CleanTimeout: time.Second,      // faster for test auto clean old session
	}) // optionally configure the bridge between your redis server

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

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set/")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Error listen: %s", err.Error())
		return
	}

	// Intercept Ctrl-C for normal shutdown
	var sig = make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	go func(sig chan os.Signal, ln net.Listener) {
		select {
		case <-sig:
			fmt.Println(" Interrupt! Close http server...")
			_ = ln.Close()
		}
	}(sig, ln)

	http.Serve(ln, nil)
}
