package gapi

import (
	"fmt"
	"net/http"
	"os"
)

type Logger struct {
	FilePath string
}

func (this Logger) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		message := req.Method + " " + req.URL.RequestURI()
		_file, _ := os.OpenFile(this.FilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if _, err := _file.Write([]byte(message+"\r\n")); err != nil {
			fmt.Println("Logger.log(): Error writing to the file ", this.FilePath, err)
			return
		}

		_file.Close()
		if next != nil {
			next.ServeHTTP(res, req)
		}

	})
}
