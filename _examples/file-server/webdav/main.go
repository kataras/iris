package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
	"github.com/kataras/iris/v12/middleware/recover"

	"golang.org/x/net/webdav"
)

func main() {
	app := iris.New()

	app.Logger().SetLevel("debug")
	app.Use(recover.New())
	app.Use(accesslog.New(os.Stdout).Handler)

	webdavHandler := &webdav.Handler{
		FileSystem: webdav.Dir("./"),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				app.Logger().Error(err)
			}
		},
	}

	app.HandleMany(strings.Join(iris.WebDAVMethods, " "), "/{p:path}", iris.FromStd(webdavHandler))

	app.Listen(":8080",
		iris.WithoutServerError(iris.ErrServerClosed, iris.ErrURLQuerySemicolon),
		iris.WithoutPathCorrection,
	)
}

/* Test with cURL or postman:

* List files:
	curl --location --request PROPFIND 'http://localhost:8080'
* Get File:
	curl --location --request GET 'http://localhost:8080/test.txt'
* Upload File:
	curl --location --request PUT 'http://localhost:8080/newfile.txt' \
	--header 'Content-Type: text/plain' \
	--data-raw 'This is a new file!'
* Copy File:
	curl --location --request COPY 'http://localhost:8080/test.txt' \
	--header 'Destination: newdir/test.txt'
* Create New Directory:
	curl --location --request MKCOL 'http://localhost:8080/anewdir/'

And e.t.c.
*/
