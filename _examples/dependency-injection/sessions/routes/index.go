package routes

import (
	"fmt"

	"github.com/kataras/iris/v12/sessions"
)

// Index will increment a simple int version based on the visits that this user/session did.
func Index(session *sessions.Session) string {
	// it increments a "visits" value of integer by one,
	// if the entry with key 'visits' doesn't exist it will create it for you.
	visits := session.Increment("visits", 1)

	// write the current, updated visits.
	return fmt.Sprintf("%d visit(s) from my current session", visits)
}
