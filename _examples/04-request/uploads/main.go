// Multipart file uploads.
//
// ctx.UploadFormFiles saves every file of the form into a directory. It
// sanitizes each client-supplied name against context.ValidFilenameRegexp
// (letters, digits, '.', '_', '-'), blocks path traversal such as
// "../../etc/crontab" and silently skips names it cannot make safe. The
// optional callback runs before each save, so it can reject or rename a file.
//
// PostMaxMemory bounds how much of a multipart form is buffered in memory
// while parsing (32MB by default); the spillover goes to temporary files. It
// is NOT a body size limit - for that see ../body-limits.
//
// The upload calls return their errors rather than answering the client, so
// the handlers forward them: errors.HandleRequestError for a malformed
// request, errors.HandleError for a failure that is the server's, and
// errors.InvalidArgument for a name that cannot be made safe. Each renders
// the rest/errors JSON envelope with the status its classification implies.
//
// Learn: save multipart uploads under safe file names and know which error
// helper answers which kind of failure.
// Before: ../pagination
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST http://localhost:8080/upload -F 'files=@main.go' -F 'files=@go.mod'   # 200, both names and the byte count
//	curl -i -X POST http://localhost:8080/upload/one -F 'file=@main.go'                  # 200, one name and its size
//	curl -i -X POST http://localhost:8080/upload -d 'not=multipart'                      # 400, not a multipart form
//
// Next: ../body-limits
package main

import (
	"mime/multipart"
	"os"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

const uploadDir = "./uploads"

func main() {
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			PostMaxMemory: 8 * iris.MB,
		},
	})

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		app.Logger().Fatal(err)
	}

	app.Post("/upload", uploadMany)
	app.Post("/upload/one", uploadOne)

	app.Listen(":8080")
}

// uploadMany is the batteries-included path.
func uploadMany(ctx iris.Context) {
	files, n, err := ctx.UploadFormFiles(uploadDir,
		func(_ iris.Context, file *multipart.FileHeader) bool {
			return file.Size <= 2*iris.MB // skip anything larger.
		})
	if err != nil {
		// e.g. the request was not a multipart/form-data one. An unknown
		// error from a request read is a 400 through HandleRequestError.
		errors.HandleRequestError(ctx, err)
		return
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.Filename) // the sanitized name.
	}

	ctx.OK(iris.Map{"uploaded": names, "bytes": n}, nil)
}

// uploadOne is the manual path: pick one form key, sanitize the name with
// context.SafeFilename (the same check UploadFormFiles applies), then write
// it with ctx.SaveFormFile.
func uploadOne(ctx iris.Context) {
	file, header, err := ctx.FormFile("file")
	if err != nil {
		errors.HandleRequestError(ctx, err)
		return
	}
	defer file.Close()

	destPath, filename, safe, err := context.SafeFilename(uploadDir, header.Filename)
	if err != nil {
		// HandleError classifies an unknown error as a 500 and logs it, so
		// the client never sees the filesystem detail.
		errors.HandleError(ctx, err)
		return
	}
	if !safe {
		errors.InvalidArgument.Detailf(ctx, "unsafe file name",
			"%q cannot be made safe", header.Filename)
		return
	}

	n, err := ctx.SaveFormFile(header, destPath)
	if err != nil {
		errors.HandleError(ctx, err)
		return
	}

	ctx.OK(iris.Map{"uploaded": filename, "bytes": n}, nil)
}
