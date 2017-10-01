package routes

import (
	"github.com/kataras/iris/_examples/structuring/handler-based/bootstrap"
)

// Configure registers the nessecary routes to the app.
func Configure(b *bootstrap.Bootstrapper) {
	// routes
	b.Get("/", GetIndexHandler)
	b.Get("/follower/{id:long}", GetFollowerHandler)
	b.Get("/following/{id:long}", GetFollowingHandler)
	b.Get("/like/{id:long}", GetLikeHandler)
}
