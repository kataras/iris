package iris

import ()

//I made the domains works as fast as possible,
//but even now we have +3k nanoseconds on normal router and +6k on memory router (because of path+host)
//so I decide to make new router types as I did with the router and memory router,I am doing this only for performance matters.
//so this plugin will check if the router we have it has subdomains on it's garden,
//if yes then convert normal router to normaldomain router and memory router to memorydomain router

type preparePlugin struct {
	IPlugin
}

func (p preparePlugin) GetName() string {
	return "prepare_plugin.go"
}

// GetDescription has to returns the description of what the plugins is used for
func (p preparePlugin) GetDescription() string {
	return "Build'n prepare before listen"
}

func (p preparePlugin) Activate(IPluginContainer) error { return nil }
func (p preparePlugin) PreHandle(s string, r IRoute)    {}
func (p preparePlugin) PostHandle(string, IRoute)       {}

func (p preparePlugin) hasHosts(s *Station) bool {
	gLen := len(s.IRouter.getGarden())
	for i := 0; i < gLen; i++ {
		if s.IRouter.getGarden()[i].hosts {
			return true
		}
	}
	return false
}

// For performance only,in order to not check at runtime for hosts and subdomains, I think it's better to do this:
func (p preparePlugin) PreListen(s *Station) {
	if p.hasHosts(s) {
		switch s.IRouter.getType() {
		case Normal:
			s.IRouter = NewRouterDomain(s.IRouter.(*Router))
			break
		case Memory:
			s.IRouter = NewMemoryRouterDomain(s.IRouter.(*MemoryRouter))
			break
		}

	}
}

func (p preparePlugin) PostListen(*Station, error) {}
func (p preparePlugin) PreClose(*Station)          {}
