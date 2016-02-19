package iris

import (
	"fmt"
	"net/http"
	"os"
)

// Logger is a future feature, it has not been used at the moment,
// it's usage will be to optionaly logs the server's requests
type Logger struct {
	FilePath string
}

// Log just writes to a file the requests
func (l Logger) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		message := req.Method + " " + req.URL.RequestURI()
		_file, _ := os.OpenFile(l.FilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if _, err := _file.Write([]byte(message + "\r\n")); err != nil {
			fmt.Println("Logger.log(): Error writing to the file ", l.FilePath, err)
			return
		}

		_file.Close()
		if next != nil {
			next.ServeHTTP(res, req)
		}

	})
}
