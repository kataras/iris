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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

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
	// redis is the empty redis service, you can set configs via this object
	redis = service.Empty()
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
