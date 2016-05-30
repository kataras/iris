package iriscontrol

import (
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
	// _ empty because it auto-registers
	_ "github.com/kataras/iris/sessions/providers/memory"
)

var panelSessions *sessions.Manager

func init() {
	//using the default
	panelSessions = sessions.New()
}

type user struct {
	username string
	password string
}
type userAuth struct {
	authenticatedUsers []user
}

// newUserAuth returns a new userAuth object, parameter is the authenticated users as map
func newUserAuth(usersMap map[string]string) *userAuth {
	if usersMap != nil {
		obj := &userAuth{make([]user, 0)}
		for key, val := range usersMap {
			obj.authenticatedUsers = append(obj.authenticatedUsers, user{key, val})
		}

		return obj
	}

	return nil
}

func (u *userAuth) login(ctx *iris.Context) {
	session := panelSessions.Start(ctx)

	username := ctx.PostFormValue("username")
	password := ctx.PostFormValue("password")

	for _, authenticatedUser := range u.authenticatedUsers {
		if authenticatedUser.username == username && authenticatedUser.password == password {
			session.Set("username", username)
			session.Set("password", password)
			ctx.Write("success")
			return
		}
	}
	ctx.Write("fail")

}

func (u *userAuth) logout(ctx *iris.Context) {
	session := panelSessions.Start(ctx)
	session.Set("user", nil)

	ctx.Redirect("/login")
}

// check if session stored, then check if this user is the correct, each time, then continue, else not
func (u *userAuth) Serve(ctx *iris.Context) {
	if ctx.PathString() == "/login" || strings.HasPrefix(ctx.PathString(), "/public") {
		ctx.Next()
		return
	}
	session := panelSessions.Start(ctx)

	if sessionVal := session.Get("username"); sessionVal != nil {
		username := sessionVal.(string)
		password := session.GetString("password")
		if username != "" && password != "" {

			for _, authenticatedUser := range u.authenticatedUsers {
				if authenticatedUser.username == username && authenticatedUser.password == password {
					ctx.Next()

					return
				}
			}
		}

	}
	//if not logged in the redirect to the /login
	ctx.Redirect("/login")

}

// Destroy this is called on PreClose by the iriscontrol.go
func (u *userAuth) Destroy() {

}
