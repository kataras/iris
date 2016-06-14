package sessions

import "github.com/kataras/iris/config"

// New creates & returns a new Manager and start its GC
func New(cfg ...config.Sessions) *Manager {
	c := config.DefaultSessions().Merge(cfg)
	// If provider is empty then return nil manager, means that the sessions are disabled
	if c.Provider == "" {
		return nil
	}
	manager, err := newManager(c)
	if err != nil {
		panic(err.Error()) // we have to panic here because we will start GC after and if provider is nil then many panics will come
	}
	//run the GC here
	go manager.GC()
	return manager
}
