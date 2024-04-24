package routes

import (
	"github.com/kataras/iris/v12/_examples/bootstrapper/bootstrap"
)

// Configure registers the necessary routes to the app.
func Configure(b *bootstrap.Bootstrapper) {
	b.Get("/", GetIndexHandler)
	b.Get("/follower/{id:int64}", GetFollowerHandler)
	b.Get("/following/{id:int64}", GetFollowingHandler)
	b.Get("/like/{id:int64}", GetLikeHandler)
}
