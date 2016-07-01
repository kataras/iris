package memory

import (
	"time"

	"github.com/kataras/iris/sessions"
	"github.com/kataras/iris/sessions/store"
)

func init() {
	register()
}

var (
	// Provider the memory provider
	Provider = sessions.NewProvider("memory")
)

// register registers itself (the new provider with its memory store) to the sessions providers
// must runs only once
func register() {
	// the actual work is here.
	Provider.NewStore = func(sessionId string, cookieLifeDuration time.Duration) store.IStore {
		return &Store{sid: sessionId, lastAccessedTime: time.Now(), values: make(map[string]interface{}, 0)}
	}
	sessions.Register(Provider)
}
