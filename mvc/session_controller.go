package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

// SessionController is a simple `Controller` implementation
// which requires a binded session manager in order to give
// direct access to the current client's session via its `Session` field.
type SessionController struct {
	Controller

	Manager *sessions.Sessions
	Session *sessions.Session
}

var managerMissing = "MVC SessionController: session manager field is nil, you have to bind it to a *sessions.Sessions"

// BeginRequest calls the Controller's BeginRequest
// and tries to initialize the current user's Session.
func (s *SessionController) BeginRequest(ctx context.Context) {
	s.Controller.BeginRequest(ctx)
	if s.Manager == nil {
		ctx.Application().Logger().Errorf(managerMissing)
		return
	}

	s.Session = s.Manager.Start(ctx)
}

/* TODO:
Maybe add struct tags on `binder` for required binded values
in order to log error if some of the bindings are missing or leave that to the end-developers?
*/
