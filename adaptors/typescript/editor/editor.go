package editor

//  +------------------------------------------------------------+
//  | Editor usage                                               |
//  +------------------------------------------------------------+
//
// 	import "gopkg.in/kataras/iris.v6/adaptors/editor"
//
// 	e := editor.New(editor.Config{})
// 	app.Adapt(e)
//
// 	app.Listen(":8080")

//
//  +------------------------------------------------------------+
//  | General notes for authentication                           |
//  +------------------------------------------------------------+
//
// The Authorization specifies the authentication mechanism (in this case Basic) followed by the username and password.
// Although, the string aHR0cHdhdGNoOmY= may look encrypted it is simply a base64 encoded version of <username>:<password>.
// Would be readily available to anyone who could intercept the HTTP request.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/typescript/npm"
)

type (
	// Editor is the alm-tools adaptor.
	//
	// It holds a logger from the iris' station
	// username,password for basic auth
	// directory which the client side code is
	// keyfile,certfile for TLS listening
	// and a host which is listening for
	Editor struct {
		config  *Config
		logger  func(iris.LogMode, string)
		enabled bool // default true
		// after alm started
		process     *os.Process
		debugOutput io.Writer
	}
)

// New creates and returns an Editor Plugin instance
func New(cfg ...Config) *Editor {
	c := DefaultConfig().Merge(cfg)
	c.WorkingDir = validateWorkingDir(c.WorkingDir) // add "/" if not exists

	return &Editor{
		enabled: true,
		config:  &c,
	}
}

// Adapt adapts the editor with Iris.
// Note:
// We use that method and not the return on New because we
// want to export the Editor's functionality to the user.
func (e *Editor) Adapt(frame *iris.Policies) {
	policy := iris.EventPolicy{
		Build:       e.build,
		Interrupted: e.close,
	}

	policy.Adapt(frame)
}

// User set a user, accepts two parameters: username (string), string (string)
func (e *Editor) User(username string, password string) *Editor {
	e.config.Username = username
	e.config.Password = password
	return e
}

func validateWorkingDir(workingDir string) string {
	l := workingDir[len(workingDir)-1]

	if l != '/' && l != os.PathSeparator {
		workingDir += "/"
	}
	return workingDir
}

// Dir sets the directory which the client side source code alive
func (e *Editor) Dir(workingDir string) *Editor {
	e.config.WorkingDir = validateWorkingDir(workingDir)
	return e
}

// Port sets the port (int) for the editor adaptor's standalone server
func (e *Editor) Port(port int) *Editor {
	e.config.Port = port
	return e
}

// SetEnable if true enables the editor adaptor, otherwise disables it
func (e *Editor) SetEnable(enable bool) {
	e.enabled = enable
}

// DisableOutput call that if you don't care about alm-tools' messages
// they are useful because that the default configuration shows them
func (e *Editor) DisableOutput() {
	e.config.DisableOutput = true
}

// GetDescription EditorPlugin is a bridge between Iris and the alm-tools, the browser-based IDE for client-side sources.
func (e *Editor) GetDescription() string {
	return "A bridge between Iris and the alm-tools, the browser-based IDE."
}

// we use that editorWriter to prefix the editor's output with "Editor Adaptor: "
type editorWriter struct {
	underline io.Writer
}

// build runs before the server's listens,  creates the listener ( use of port parent hostname:DefaultPort if not exist)
func (e *Editor) build(s *iris.Framework) {
	e.logger = s.Log
	if e.config.Hostname == "" {
		e.config.Hostname = iris.ParseHostname(s.Config.VHost)
	}

	if e.config.Port <= 0 {
		e.config.Port = DefaultPort
	}

	if s, err := filepath.Abs(e.config.WorkingDir); err == nil {
		e.config.WorkingDir = s
	}

	e.start()
}

// close kills the editor's server when Iris is closed
func (e *Editor) close(s *iris.Framework) {
	if e.process != nil {
		err := e.process.Kill()
		if err != nil {
			e.logger(iris.DevMode, fmt.Sprintf(`Error while trying to terminate the Editor,
				 please kill this process by yourself, process id: %d`, e.process.Pid))
		}
	}
}

// start starts the job
func (e *Editor) start() {
	if e.config.Username == "" || e.config.Password == "" {
		e.logger(iris.ProdMode, `Error before running alm-tools.
			You have to set username & password for security reasons, otherwise this adaptor won't run.`)
		return
	}

	if !npm.NodeModuleExists("alm/bin/alm") {
		e.logger(iris.DevMode, "Installing alm-tools, please wait...")
		res := npm.NodeModuleInstall("alm")
		if res.Error != nil {
			e.logger(iris.ProdMode, res.Error.Error())
			return
		}
		e.logger(iris.DevMode, res.Message)
	}

	cmd := npm.CommandBuilder("node", npm.NodeModuleAbs("alm/src/server.js"))
	cmd.AppendArguments("-a", e.config.Username+":"+e.config.Password,
		"-h", e.config.Hostname, "-t", strconv.Itoa(e.config.Port), "-d", e.config.WorkingDir)
	// for auto-start in the browser: cmd.AppendArguments("-o")
	if e.config.KeyFile != "" && e.config.CertFile != "" {
		cmd.AppendArguments("--httpskey", e.config.KeyFile, "--httpscert", e.config.CertFile)
	}

	prefix := ""
	// when debug is not disabled
	// show any messages to the user( they are useful here)
	// to the io.Writer that iris' user is defined from configuration
	if !e.config.DisableOutput {

		outputReader, err := cmd.StdoutPipe()
		if err == nil {
			outputScanner := bufio.NewScanner(outputReader)

			go func() {
				for outputScanner.Scan() {
					e.logger(iris.DevMode, prefix+outputScanner.Text())
				}
			}()

			errReader, err := cmd.StderrPipe()
			if err == nil {
				errScanner := bufio.NewScanner(errReader)
				go func() {
					for errScanner.Scan() {
						e.logger(iris.DevMode, prefix+errScanner.Text())
					}
				}()
			}
		}
	}

	err := cmd.Start()
	if err != nil {
		e.logger(iris.ProdMode, prefix+err.Error())
		return
	}

	// no need, alm-tools post these
	// e.logger.Printf("Editor is running at %s:%d | %s", e.config.Hostname, e.config.Port, e.config.WorkingDir)
}
