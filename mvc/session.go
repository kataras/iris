package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

// Session -> TODO: think of move all bindings to
// a different folder like "bindings"
// so it will be used as .Bind(bindings.Session(manager))
// or let it here but change the rest of the binding names as well
// because they are not "binders", their result are binders to be percise.
func Session(sess *sessions.Sessions) func(context.Context) *sessions.Session {
	return func(ctx context.Context) *sessions.Session {
		return sess.Start(ctx)
	}
}
