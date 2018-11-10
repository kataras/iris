package versioning

import (
	"strings"

	"github.com/kataras/iris/context"
)

const (
	AcceptVersionHeaderKey   = "Accept-Version"
	AcceptHeaderKey          = "Accept"
	AcceptHeaderVersionValue = "version"

	Key      = "iris.api.version" // for use inside the ctx.Values(), not visible by the user.
	NotFound = Key + ".notfound"
)

var NotFoundHandler = func(ctx context.Context) {
	// 303 is an option too,
	// end-dev has the chance to change that behavior by using the NotFound in the map:
	//
	// https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
	/*
		10.5.2 501 Not Implemented

		The server does not support the functionality required to fulfill the request.
		This is the appropriate response when the server does not
		recognize the request method and is not capable of supporting it for any resource.
	*/
	ctx.WriteString("version not found")
	ctx.StatusCode(501)
}

func GetVersion(ctx context.Context) string {
	// firstly by context store, if manually set-ed by a middleware.
	if version := ctx.Values().GetString(Key); version != "" {
		return version
	}

	// secondly by the "Accept-Version" header.
	if version := ctx.GetHeader(AcceptVersionHeaderKey); version != "" {
		return version
	}

	// thirdly by the "Accept" header which is like"...; version=1.0"
	acceptValue := ctx.GetHeader(AcceptHeaderKey)
	if acceptValue != "" {
		if idx := strings.Index(acceptValue, AcceptHeaderVersionValue); idx != -1 {
			rem := acceptValue[idx:]
			startVersion := strings.Index(rem, "=")
			if startVersion == -1 || len(rem) < startVersion+1 {
				return NotFound
			}

			rem = rem[startVersion+1:]

			end := strings.Index(rem, " ")
			if end == -1 {
				end = strings.Index(rem, ";")
			}
			if end == -1 {
				end = len(rem)
			}

			if version := rem[:end]; version != "" {
				return version
			}
		}
	}

	return NotFound
}
