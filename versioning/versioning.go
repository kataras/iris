package versioning

import (
	"github.com/kataras/iris/context"

	"github.com/hashicorp/go-version"
)

type Map map[string]context.Handler

func NewMatcher(versions Map) context.Handler {
	constraintsHandlers, notFoundHandler := buildConstraints(versions)

	return func(ctx context.Context) {
		versionString := GetVersion(ctx)
		if versionString == NotFound {
			notFoundHandler(ctx)
			return
		}

		version, err := version.NewVersion(versionString)
		if err != nil {
			notFoundHandler(ctx)
			return
		}

		for _, ch := range constraintsHandlers {
			if ch.constraints.Check(version) {
				ctx.Header("X-API-Version", version.String())
				ch.handler(ctx)
				return
			}
		}

		// pass the not matched version so the not found handler can have knowedge about it.
		// ctx.Values().Set(Key, versionString)
		// or let a manual cal of GetVersion(ctx) do that instead.
		notFoundHandler(ctx)
	}
}

type constraintsHandler struct {
	constraints version.Constraints
	handler     context.Handler
}

func buildConstraints(versionsHandler Map) (constraintsHandlers []*constraintsHandler, notfoundHandler context.Handler) {
	for v, h := range versionsHandler {
		if v == NotFound {
			notfoundHandler = h
			continue
		}

		constraints, err := version.NewConstraint(v)
		if err != nil {
			panic(err)
		}

		constraintsHandlers = append(constraintsHandlers, &constraintsHandler{
			constraints: constraints,
			handler:     h,
		})
	}

	if notfoundHandler == nil {
		notfoundHandler = NotFoundHandler
	}

	// no sort, the end-dev should declare
	// all version constraint, i.e < 4.0 may be catch 1.0 if not something like
	// >= 3.0, < 4.0.
	// I can make it ordered but I do NOT like the final API of it:
	/*
		app.Get("/api/user", NewMatcher( // accepts an array, ordered, see last elem.
			V("1.0", vHandler("v1 here")),
			V("2.0", vHandler("v2 here")),
			V("< 4.0", vHandler("v3.x here")),
		))
		instead we have:

		app.Get("/api/user", NewMatcher(Map{ // accepts a map, unordered, see last elem.
			"1.0":           Deprecated(vHandler("v1 here")),
			"2.0":           vHandler("v2 here"),
			">= 3.0, < 4.0": vHandler("v3.x here"),
			VersionUnknown: customHandlerForNotMatchingVersion,
		}))
	*/

	return
}
