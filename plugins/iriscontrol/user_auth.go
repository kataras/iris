package iriscontrol

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

var store = sessions.NewCookieStore([]byte(RandStringBytesMaskImprSrc(10)))
var panelSessions = sessions.New("user_sessions", store)

type userAuth struct {
}

func (u *userAuth) Serve(ctx *iris.Context) {

	ctx.Next()
}
