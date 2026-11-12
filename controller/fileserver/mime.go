package fileserver

import "github.com/kataras/iris/v14/internal/mimetypes"

// TypeByExtension returns the MIME type associated with the file extension ext.
// The extension ext should begin with a leading dot, as in ".html". When ext has
// no associated type, TypeByExtension returns "".
//
// It answers from the framework's extended MIME table, the one the file
// server itself uses. Importing this package also registers that table with
// the standard library's mime package, so mime.TypeByExtension knows these
// extensions too.
func TypeByExtension(ext string) string {
	return mimetypes.TypeByExtension(ext)
}

// TypeByFilename is like TypeByExtension but receives a full filename path
// instead of just the extension.
func TypeByFilename(fullFilename string) string {
	return mimetypes.TypeByFilename(fullFilename)
}
