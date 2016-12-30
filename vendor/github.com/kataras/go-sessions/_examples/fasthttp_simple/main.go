// +build ignore

package main

import (
	"fmt"
	"github.com/kataras/go-sessions"
	"github.com/valyala/fasthttp"
)

func main() {

	// set some values to the session
	setHandler := func(reqCtx *fasthttp.RequestCtx) {
		values := map[string]interface{}{
			"Name":   "go-sessions",
			"Days":   "1",
			"Secret": "dsads£2132215£%%Ssdsa",
		}

		sess := sessions.StartFasthttp(reqCtx) // init the session
		// sessions.StartFasthttp returns:
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
		reqCtx.WriteString("Session saved, go to /get to view the results")
	}

	// get the values from the session
	getHandler := func(reqCtx *fasthttp.RequestCtx) {
		sess := sessions.StartFasthttp(reqCtx) // init the session
		sessValues := sess.GetAll()            // get all values from this session

		reqCtx.WriteString(fmt.Sprintf("%#v", sessValues))
	}

	// clear all values from the session
	clearHandler := func(reqCtx *fasthttp.RequestCtx) {
		sess := sessions.StartFasthttp(reqCtx)
		sess.Clear()
	}

	// destroys the session, clears the values and removes the server-side entry and client-side sessionid cookie
	destroyHandler := func(reqCtx *fasthttp.RequestCtx) {
		sessions.DestroyFasthttp(reqCtx)
	}

	fmt.Println("Open a browser tab and navigate to the localhost:8080/set")
	fasthttp.ListenAndServe(":8080", func(reqCtx *fasthttp.RequestCtx) {
		path := string(reqCtx.Path())

		if path == "/set" {
			setHandler(reqCtx)
		} else if path == "/get" {
			getHandler(reqCtx)
		} else if path == "/clear" {
			clearHandler(reqCtx)
		} else if path == "/destroy" {
			destroyHandler(reqCtx)
		} else {
			reqCtx.WriteString("Please navigate to /set or /get or /clear or /destroy")
		}
	})
}
