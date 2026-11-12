package router

// Side-effect import: internal/mimetypes' init registers the extended MIME type
// table into the standard library's mime package, so mime.TypeByExtension (used
// across the framework) resolves the additional types. core/router is always
// imported by an Iris application, which keeps this registration unconditional.
//
// The MIME lookup helpers (TypeByExtension/TypeByFilename) live in
// internal/mimetypes; controller/fileserver re-exports them.
import _ "github.com/kataras/iris/v14/internal/mimetypes"
