// Capping request body size, app-wide and per route.
//
// ContextOptions.BodyMaxMemory limits the bytes read from a body by Body,
// ReadJSON, ReadJSONStream and ReadMultipartRelated - and therefore by
// everything built on them, including ReadXML/ReadYAML/ReadBody and
// ctx.BindJSON. The default 0 means no limit. Reading past the cap fails
// with context.ErrRequestEntityTooLarge, which rest/errors renders as a
// structured 413 TOO_LARGE.
//
// The bodylimit middleware applies the same enforcement wherever it is
// registered. It rejects a request up front when the declared Content-Length
// already exceeds the limit, and wraps the body reader to catch chunked or
// compressed bodies that only reveal their size while being read: the read
// stops at the limit and fails with the same error. Forms are the exception:
// ctx.ReadForm returns no error for a chunked form past the limit and binds
// nothing of it.
//
// PostMaxMemory is a different knob - it bounds multipart form buffering, not
// body size. See ../uploads.
//
// Learn: cap request bodies for the whole app and tighter per route, and
// answer a structured 413 from either limit.
// Before: ../uploads
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST http://localhost:8080/notes -H 'Content-Type: application/json' \
//	  -d '{"text":"hello"}'                                                           # 200, {"chars":5}
//	printf '{"text":"%s"}' "$(head -c 2000000 /dev/zero | tr '\0' a)" > big.json      # a ~2MB JSON body
//	curl -i -X POST http://localhost:8080/notes -H 'Content-Type: application/json' \
//	  --data-binary @big.json                                                         # 413 TOO_LARGE, the 1MB app-wide cap
//	curl -i -X POST http://localhost:8080/uploads/blob --data-binary @big.json        # 413 TOO_LARGE, the 64KB route limit
//
// Next: ../../05-response/rest-helpers
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New(iris.Options{
		Context: iris.ContextOptions{
			BodyMaxMemory: 1 * iris.MB, // app-wide.
		},
	})

	app.Post("/notes", createNote)

	// Per route or group: the same enforcement, a tighter number.
	uploads := app.Router("/uploads", bodylimit.New(64*iris.KB))
	uploads.Post("/blob", receiveBlob)

	app.Listen(":8080")
}

// Note is the JSON payload of /notes.
type Note struct {
	Text string `json:"text"`
}

func createNote(ctx iris.Context) {
	note, ok := ctx.BindJSON[Note]()
	if !ok {
		return // 413 past BodyMaxMemory, 400 for malformed JSON.
	}

	ctx.OK(iris.Map{"chars": len(note.Text)}, nil)
}

func receiveBlob(ctx iris.Context) {
	var raw []byte
	if err := ctx.ReadBody(&raw); err != nil {
		// ReadBody returns the error instead of answering the client, so the
		// handler forwards it. HandleRequestError keeps the classification:
		// a body past the cap is still 413, anything else is 400.
		errors.HandleRequestError(ctx, err)
		return
	}

	ctx.OK(iris.Map{"bytes": len(raw)}, nil)
}
