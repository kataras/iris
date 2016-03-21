// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
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
	return "station_prepare_plugin.go"
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
