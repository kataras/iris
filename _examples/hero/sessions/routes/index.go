package routes

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/sessions"
)

// Index will increment a simple int version based on the visits that this user/session did.
func Index(ctx iris.Context, session *sessions.Session) {
	// it increments a "visits" value of integer by one,
	// if the entry with key 'visits' doesn't exist it will create it for you.
	visits := session.Increment("visits", 1)

	// write the current, updated visits.
	ctx.Writef("%d visit(s) from my current session", visits)
}

/*
You can also do anything that an MVC function can, i.e:

func Index(ctx iris.Context,session *sessions.Session) string {
	visits := session.Increment("visits", 1)
	return fmt.Spritnf("%d visit(s) from my current session", visits)
}
// you can also omit iris.Context input parameter and use dependency injection for LoginForm and etc. <- look the mvc examples.
*/
