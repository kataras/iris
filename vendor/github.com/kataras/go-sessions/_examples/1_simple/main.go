// +build ignore

package main

import (
	"fmt"
	"github.com/kataras/go-sessions"
	"net/http"
)

func main() {

	// set some values to the session
	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := sessions.Start(res, req) // init the session
		// sessions.Start returns:
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
		sess := sessions.Start(res, req) // init the session
		sessValues := sess.GetAll()      // get all values from this session

		res.Write([]byte(fmt.Sprintf("%#v", sessValues)))
	})
	http.Handle("/get/", getHandler)

	// clear all values from the session
	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req)
		sess.Clear()
	})
	http.Handle("/clear/", clearHandler)

	// destroys the session, clears the values and removes the server-side entry and client-side sessionid cookie
	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sessions.Destroy(res, req)
	})
	http.Handle("/destroy/", destroyHandler)

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set/")
	http.ListenAndServe(":8080", nil)
}
