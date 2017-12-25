package hero

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

// Session is a binder that will fill a *sessions.Session function input argument
// or a Controller struct's field.
func Session(sess *sessions.Sessions) func(context.Context) *sessions.Session {
	return sess.Start
}
