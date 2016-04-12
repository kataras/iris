package main

import (
	"net/http"
	_"time"
	"log"
)

func main() {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/rest/hello", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		//time.Sleep(time.Duration(500) * time.Millisecond)
		res.Write([]byte("Hello world"))
	}))
	

	log.Fatal(http.ListenAndServe(":8080", mux))
}
