package redis

import (
	"time"

	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/providers/redis/service"
	"github.com/kataras/iris/sessions/store"
)

func init() {
	register()
}

var (
	Provider = sessions.NewProvider("redis")
	// redis is the default redis service, you can set configs via this object
	redis = service.New()
	// Config is just the Redis(service)' config
	Config = redis.Config

// Empty() because maybe the user wants to edit the default configs.
//the Connect goes to the first NewStore, when user ask for session, so you have the time to change the default configs
)

// register registers itself (the new provider with its memory store) to the sessions providers
// must runs only once
func register() {
	// the actual work is here.
	Provider.NewStore = func(sessionId string, cookieLifeDuration time.Duration) store.IStore {
		//println("memory.go:49-> requesting new memory store with sessionid: " + sessionId)
		if !redis.Connected {
			redis.Connect()
			_, err := redis.PingPong()
			if err != nil {
				if err != nil {
					// don't use to get the logger, just prin these to the console... atm
					println("Redis Connection error on iris/sessions/providers/redisstore.Connect: " + err.Error())
					println("But don't panic, auto-switching to memory store right now!")
				}
			}
		}
		return NewStore(sessionId, cookieLifeDuration)
	}

	sessions.Register(Provider)
}
