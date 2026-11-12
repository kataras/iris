package fileserver

import (
	"net/http"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

// WebDAVMethods contains a list of WebDAV HTTP Verbs.
var WebDAVMethods = [...]string{
	"OPTIONS",
	"GET",
	"HEAD",
	"POST",
	"DELETE",
	"PUT",
	//
	"MKCOL",
	"COPY",
	"MOVE",
	"LOCK",
	"UNLOCK",
	"PROPFIND",
	"PROPPATCH",
	//
	"LINK",
	"UNLINK",
	"PURGE",
	"VIEW",
}

// WebDAV returns a new WebDAV controller.
// Usage:
//
//	import "golang.org/x/net/webdav"
//
//
//	webdavHandler := &webdav.Handler{
//		FileSystem: webdav.Dir("./"),
//		LockSystem: webdav.NewMemLS(),
//		Logger: func(r *http.Request, err error) {
//			if err != nil {
//				app.Logger().Error(err)
//			}
//		},
//	}
//
//	app.Router("/", WebDAV(webdavHandler))
//	app.Listen(":8080")
//
// Test with cURL or postman:
//   - List files:
//     curl --location --request PROPFIND 'http://localhost:8080'
//   - Get File:
//     curl --location --request GET 'http://localhost:8080/test.txt'
//   - Upload File:
//     curl --location --request PUT 'http://localhost:8080/newfile.txt' \
//     --header 'Content-Type: text/plain' \
//     --data-raw 'This is a new file!'
//   - Copy File:
//     curl --location --request COPY 'http://localhost:8080/test.txt' \
//     --header 'Destination: newdir/test.txt'
//   - Create New Directory:
//     curl --location --request MKCOL 'http://localhost:8080/anewdir/'
func WebDAV(webdavHandler http.Handler) router.Controller {
	return router.Routing(func(r router.Router) {
		r.Handle(router.JoinMethods(WebDAVMethods[:]...), "/{p:path}", context.FromStd(webdavHandler))
	})
}
