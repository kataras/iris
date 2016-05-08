// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package iriscontrol

import (
	"strings"

	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
	// _ empty because it auto-registers
	_ "github.com/kataras/iris/sessions/providers/memory"
)

var panelSessions *sessions.Manager

func init() {
	panelSessions = sessions.New("memory", "irisControlPanel$", time.Duration(time.Minute)*60)
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
