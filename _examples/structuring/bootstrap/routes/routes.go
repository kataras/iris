package routes

import (
	"github.com/kataras/iris/_examples/structuring/bootstrap/bootstrap"
)

// Configure registers the necessary routes to the app.
func Configure(b *bootstrap.Bootstrapper) {
	b.Get("/", GetIndexHandler)
	b.Get("/follower/{id:long}", GetFollowerHandler)
	b.Get("/following/{id:long}", GetFollowingHandler)
	b.Get("/like/{id:long}", GetLikeHandler)
}
