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

	"github.com/kataras/iris"
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/logger"
	"github.com/kataras/iris/npm"
	"github.com/kataras/iris/utils"
)

const (
	// Name the name of the Plugin, which is "EditorPlugin"
	Name = "EditorPlugin"
)

type (
	// Plugin is an Editor Plugin the struct which implements the iris.IPlugin
	// it holds a logger from the iris' station
	// username,password for basic auth
	// directory which the client side code is
	// keyfile,certfile for TLS listening
	// and a host which is listening for
	Plugin struct {
		config   *config.Editor
		logger   *logger.Logger
		enabled  bool // default true
		keyfile  string
		certfile string
		// after alm started
		process *os.Process
	}
)

// New creates and returns an Editor Plugin instance
func New(cfg ...config.Editor) *Plugin {
	c := config.DefaultEditor().Merge(cfg)
	e := &Plugin{enabled: true, config: &c}
	return e
}

// User set a user, accepts two parameters: username (string), string (string)
func (e *Plugin) User(username string, password string) *Plugin {
	e.config.Username = username
	e.config.Password = password
	return e
}

// Dir sets the directory which the client side source code alive
func (e *Plugin) Dir(workingDir string) *Plugin {
	e.config.WorkingDir = workingDir
	return e
}

// Port sets the port (int) for the editor plugin's standalone server
func (e *Plugin) Port(port int) *Plugin {
	e.config.Port = port
	return e
}

//

// SetEnable if true enables the editor plugin, otherwise disables it
func (e *Plugin) SetEnable(enable bool) {
	e.enabled = enable
}

// GetName returns the name of the Plugin
func (e *Plugin) GetName() string {
	return Name
}

// GetDescription EditorPlugin is a bridge between Iris and the alm-tools, the browser-based IDE for client-side sources.
func (e *Plugin) GetDescription() string {
	return Name + " is a bridge between Iris and the alm-tools, the browser-based IDE for client-side sources. \n"
}

// PreListen runs before the server's listens, saves the keyfile,certfile and the host from the Iris station to listen for
func (e *Plugin) PreListen(s *iris.Iris) {
	e.logger = s.Logger()
	e.keyfile = s.Server().Config.KeyFile
	e.certfile = s.Server().Config.CertFile

	if e.config.Host == "" {
		h := s.Server().Config.ListeningAddr

		if idx := strings.Index(h, ":"); idx >= 0 {
			h = h[0:idx]
		}
		if h == "" {
			h = "127.0.0.1"
		}

		e.config.Host = h

	}
	e.start()
}

// PreClose kills the editor's server when Iris is closed
func (e *Plugin) PreClose(s *iris.Iris) {
	if e.process != nil {
		err := e.process.Kill()
		if err != nil {
			e.logger.Printf("\nError while trying to terminate the (Editor)Plugin, please kill this process by yourself, process id: %d", e.process.Pid)
		}
	}
}

// start starts the job
func (e *Plugin) start() {
	if e.config.Username == "" || e.config.Password == "" {
		e.logger.Println("Error before running alm-tools. You have to set username & password for security reasons, otherwise this plugin won't run.")
		return
	}

	if !npm.Exists("alm/bin/alm") {
		e.logger.Println("Installing alm-tools, please wait...")
		res := npm.Install("alm")
		if res.Error != nil {
			e.logger.Print(res.Error.Error())
			return
		}
		e.logger.Print(res.Message)
	}

	cmd := utils.CommandBuilder("node", npm.Abs("alm/src/server.js"))
	cmd.AppendArguments("-a", e.config.Username+":"+e.config.Password, "-h", e.config.Host, "-t", strconv.Itoa(e.config.Port), "-d", e.config.WorkingDir[0:len(e.config.WorkingDir)-1])
	// for auto-start in the browser: cmd.AppendArguments("-o")
	if e.keyfile != "" && e.certfile != "" {
		cmd.AppendArguments("--httpskey", e.keyfile, "--httpscert", e.certfile)
	}

	//For debug only:
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	//os.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		e.logger.Println("Error while running alm-tools. Trace: " + err.Error())
		return
	}

	//we lose the internal error handling but ok...
	e.logger.Printf("Editor is running at %s:%d | %s", e.config.Host, e.config.Port, e.config.WorkingDir)

}
