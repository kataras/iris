package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/sessions"
)

var defaultSessionManager = sessions.New(sessions.Config{})

// SessionController is a simple `Controller` implementation
// which requires a binded session manager in order to give
// direct access to the current client's session via its `Session` field.
//
// SessionController is deprecated please use the new dependency injection's methods instead,
// i.e `mvcApp.Register(sessions.New(sessions.Config{}).Start)`.
// It's more controlled by you,
// also *sessions.Session type can now `Destroy` itself without the need of the manager, embrace it.
type SessionController struct {
	Manager *sessions.Sessions
	Session *sessions.Session
}

// BeforeActivation called, once per application lifecycle NOT request,
// every single time the dev registers a specific SessionController-based controller.
// It makes sure that its "Manager" field is filled
// even if the caller didn't provide any sessions manager via the MVC's Application's `Handle` function.
func (s *SessionController) BeforeActivation(b BeforeActivation) {
	if didntBindManually := b.Dependencies().AddOnce(defaultSessionManager); didntBindManually {
		b.Router().GetReporter().Add(
			`MVC SessionController: couldn't find any "*sessions.Sessions" bindable value to fill the "Manager" field, 
			therefore this controller is using the default sessions manager instead.
			Please refer to the documentation to learn how you can provide the session manager`)
	}
}

// BeginRequest initializes the current user's Session.
func (s *SessionController) BeginRequest(ctx context.Context) {
	if s.Manager == nil {
		ctx.Application().Logger().Errorf(`MVC SessionController: sessions manager is nil, report this as a bug 
because the SessionController should predict this on its activation state and use a default one automatically`)
		return
	}

	s.Session = s.Manager.Start(ctx)
}

// EndRequest is here to complete the `BaseController`.
func (s *SessionController) EndRequest(ctx context.Context) {}
