package main

import (
	"fmt"
	"net/http"

	"github.com/kataras/go-sessions"
)

func main() {

	// set some flash messages
	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := sessions.Start(res, req) // init the session once

		for k, v := range values {
			sess.SetFlash(k, v) // fill flashes, set each of the key-value pair
		}
		res.Write([]byte("Session saved, go to /get or /get_single to view the flash messages"))
	})
	http.Handle("/set/", setHandler)

	// get all the flash messages
	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req) // init the session once
		sessValues := sess.GetFlashes()  // get all flash messages from this session

		res.Write([]byte(fmt.Sprintf("%#v", sessValues)))
	})
	http.Handle("/get/", getHandler)

	getSingleHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req)
		flashMsgString := sess.GetFlashString("Name")
		res.Write([]byte(flashMsgString))
	})

	http.Handle("/get_single/", getSingleHandler)

	// clear all flash messages
	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := sessions.Start(res, req)
		sess.ClearFlashes()
	})

	http.Handle("/clear/", clearHandler)

	// destroys the session, clears the values & the flash messages
	// and removes the server-side entry and client-side sessionid cookie
	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sessions.Destroy(res, req)
	})
	http.Handle("/destroy/", destroyHandler)

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set/")
	http.ListenAndServe(":8080", nil)
}
