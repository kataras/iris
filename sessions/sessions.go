package sessions

import "github.com/kataras/iris/config"

// New creates & returns a new Manager and start its GC
func New(cfg ...config.Sessions) *Manager {
	manager, err := newManager(config.DefaultSessions().Merge(cfg))
	if err != nil {
		panic(err.Error()) // we have to panic here because we will start GC after and if provider is nil then many panics will come
	}
	//run the GC here
	go manager.GC()
	return manager
}
