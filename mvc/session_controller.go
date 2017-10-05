package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator"
	"github.com/kataras/iris/sessions"

	"github.com/kataras/golog"
)

var defaultManager = sessions.New(sessions.Config{})

// SessionController is a simple `Controller` implementation
// which requires a binded session manager in order to give
// direct access to the current client's session via its `Session` field.
type SessionController struct {
	Controller

	Manager *sessions.Sessions
	Session *sessions.Session
}

// OnActivate called, once per application lifecycle NOT request,
// every single time the dev registers a specific SessionController-based controller.
// It makes sure that its "Manager" field is filled
// even if the caller didn't provide any sessions manager via the `app.Controller` function.
func (s *SessionController) OnActivate(p *activator.ActivatePayload) {
	if p.EnsureBindValue(defaultManager) {
		golog.Warnf(`MVC SessionController: couldn't find any "*sessions.Sessions" bindable value to fill the "Manager" field, 
therefore this controller is using the default sessions manager instead.
Please refer to the documentation to learn how you can provide the session manager`)
	}
}

// BeginRequest calls the Controller's BeginRequest
// and tries to initialize the current user's Session.
func (s *SessionController) BeginRequest(ctx context.Context) {
	s.Controller.BeginRequest(ctx)
	if s.Manager == nil {
		ctx.Application().Logger().Errorf(`MVC SessionController: sessions manager is nil, report this as a bug 
because the SessionController should predict this on its activation state and use a default one automatically`)
		return
	}

	s.Session = s.Manager.Start(ctx)
}
