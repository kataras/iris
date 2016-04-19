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

// editor package is not ready
package editor

/* Notes for Auth
The Authorization specifies the authentication mechanism (in this case Basic) followed by the username and password.
Although, the string aHR0cHdhdGNoOmY= may look encrypted it is simply a base64 encoded version of <username>:<password>.
Would be readily available to anyone who could intercept the HTTP request.
*/

import (
	"os"
	"strconv"
	"strings"

	_ "runtime"

	"github.com/kataras/iris"
	"github.com/kataras/iris/cli/npm"
	"github.com/kataras/iris/cli/system"
)

var (
	Name = "EditorPlugin"
)

type (
	EditorPlugin struct {
		logger    *iris.Logger
		enabled   bool              // default true
		host      string            // default 127.0.0.1
		port      int               // default 4444
		users     map[string]string // [username]password -> based on Basic Auth, // default -nothing, for security reasons you have to set it otherwise editor is not opening.
		keyfile   string
		certfile  string
		directory string // working directory

		// after alm started
		process *os.Process
	}
)

func New(username string, password string) *EditorPlugin {
	e := &EditorPlugin{enabled: true, port: 4444}
	if username != "" && password != "" {
		e.AddUser(username, password)
	}

	return e
}

func (e *EditorPlugin) AddUser(username string, password string) *EditorPlugin {
	if e.users == nil {
		e.users = make(map[string]string, 0)
	}

	e.users[username] = password
	return e
}

func (e *EditorPlugin) Dir(workingDir string) *EditorPlugin {
	e.directory = workingDir
	return e
}

func (e *EditorPlugin) Port(port int) *EditorPlugin {
	e.port = port
	return e
}

//
func (e *EditorPlugin) SetUsers(users map[string]string) {
	e.users = users
}

func (e *EditorPlugin) SetEnabled(enable bool) {
	e.enabled = enable
}

// implement the IPlugin, IPluginPreListen & IPluginPreClose
func (e *EditorPlugin) Activate(container iris.IPluginContainer) error {
	return nil
}

func (e *EditorPlugin) GetName() string {
	return Name
}

func (e *EditorPlugin) GetDescription() string {
	return Name + " a bridge between Iris and the alm-tools, the browser-based IDE for client-side sources. \n"
}

func (e *EditorPlugin) PreListen(s *iris.Station) {
	e.logger = s.Logger()
	e.keyfile = s.Server.Options().KeyFile
	e.certfile = s.Server.Options().CertFile
	e.host = s.Server.Options().ListeningAddr

	if idx := strings.Index(e.host, ":"); idx >= 0 {
		e.host = e.host[0:idx]
	}
	if e.host == "" {
		e.host = "127.0.0.1"
	}

	e.start()
}

// PreClose kills the editor's server when Iris is closed
func (e *EditorPlugin) PreClose(s *iris.Station) {
	if e.process != nil {
		err := e.process.Kill()
		if err != nil {
			e.logger.Printf("\nError while trying to terminate the EditorPlugin, please kill this process by yourself, process id: %s", e.process.Pid)
		}
	}
}

func (e *EditorPlugin) start() {
	if !npm.Exists("alm/bin/alm") {
		e.logger.Println("Installing alm-tools, please wait...")
		res := npm.Install("alm")
		if res.Error != nil {
			e.logger.Print(res.Error.Error())
			return
		}
		e.logger.Print(res.Message)
	}

	/* first option:
	binary := "alm"
	if runtime.GOOS == "windows" {
		binary += ".cmd"
	}
	rootNpm := npm.NodeModules[0:strings.LastIndex(npm.NodeModules, system.PathSeparator)]
	cmd := system.CommandBuilder(rootNpm + system.PathSeparator + binary)
	*/

	// second option:
	//cmd := system.CommandBuilder("alm")

	// third, cross platform for sure :)
	cmd := system.CommandBuilder("node", npm.Abs("alm/bin/alm"))

	cmd.AppendArguments("-d " + e.directory)
	cmd.AppendArguments("--host " + e.host)
	cmd.AppendArguments("-t " + strconv.Itoa(e.port))
	// for auto-start in the browser: cmd.AppendArguments("-o")
	if e.keyfile != "" && e.certfile != "" {
		cmd.AppendArguments("--httpskey "+e.keyfile, "--httpscert "+e.certfile)
	}

	//println("[DEBUG] alm-tools arguments: " + strings.Join(cmd.Args, " "))

	err := cmd.Start()
	if err != nil {
		e.logger.Println("Error while running alm-tools. Trace: " + err.Error())
		return
	}

	e.logger.Printf("Editor is running at %s:%d | %s", e.host, e.port, e.directory)

	///TODO: make the basic auth, I though I could run alm without it's server, but I can't. I have to ask
	// from @basarat to add basic auth in the command arguments and/or tsconfig.json (it would be nice for multiple users but impossible via command arguments)
	// example: alm -u "username,password"
	// -u stands for user(?)

}
