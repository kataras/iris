package fileserver

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

// Favicon returns a Controller which caches and serves a favicon file.
// Accepts the "favPath" which is the system path of the __.ico file.
// Usage:
//
//	app.Router("/", fileserver.Favicon("./path/to/favicon.ico"))
//
// This will serve the ./path/to/favicon.ico to the "/favicon.ico" request path on GET and HEAD HTTP Methods.
func Favicon(favPath string) router.Controller {
	description := favPath
	favPath = Abs(favPath)
	f, err := os.Open(favPath)
	if err != nil {
		panic(fmt.Errorf("favicon: file or directory %s not found: %w", favPath, err))
	}

	defer f.Close()
	fi, _ := f.Stat()
	if fi.IsDir() { // if it's dir the try to get the favicon.ico
		return Favicon(path.Join(favPath, "favicon.ico"))
	}

	// copy the bytes here in order to cache and not read the ico on each request.
	faviconCachedContents := make([]byte, fi.Size())
	// io.ReadFull, not a single Read: Read is allowed to return fewer bytes than
	// the buffer holds, and the short tail would then be served as zero bytes
	// with no error at all.
	if _, err = io.ReadFull(f, faviconCachedContents); err != nil {
		// Here we are before actually run the server.
		// So we could panic but we don't,
		// we just interrupt with a message
		// to the (user-defined) logger.
		panic(fmt.Errorf("favicon: couldn't read the data bytes for %s: %w", favPath, err))
	}

	modtime := time.Now()
	cType := TypeByFilename(favPath)

	reqPath := "/favicon" + path.Ext(fi.Name()) // we could use the filename, but because standards is /favicon.ico.
	handler := func(ctx *context.Context) {
		ctx.SetContentType(cType)
		if _, err := ctx.WriteWithExpiration(faviconCachedContents, modtime); err != nil {
			// Private: a failed write is the server's own trouble.
			ctx.StopWithPlainError(500, context.PrivateError(err))
			return
		}
	}

	return router.Routing(func(r router.Router) {
		r.Resource(reqPath, handler).Get().Describe(description)
	})
}
