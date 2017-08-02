package editor

/* Package editor provides alm-tools cloud editor automation for the iris web framework.

Usage:


	import "github.com/kataras/iris/typescript/editor"
	[...]

	app := iris.New()
	e := editor.New(editor.Config{})
	e.Run(app.Logger().Infof)

	[...]
	app.Run(iris.Addr(":8080"))
	e.Stop()


General notes for authentication


The Authorization specifies the authentication mechanism (in this case Basic) followed by the username and password.
Although, the string aHR0cHdhdGNoOmY= may look encrypted it is simply a base64 encoded version of <username>:<password>.
Would be readily available to anyone who could intercept the HTTP request.
*/
import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kataras/iris/typescript/npm"
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
		log     func(format string, a ...interface{})
		enabled bool // default true
		// after alm started
		process     *os.Process
		debugOutput io.Writer
	}
)

var (
	// NoOpLogger can be used as the logger argument, it prints nothing.
	NoOpLogger = func(string, ...interface{}) {}
)

// New creates and returns an Editor Plugin instance
func New(cfg ...Config) *Editor {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.WorkingDir = validateWorkingDir(c.WorkingDir) // add "/" if not exists

	return &Editor{
		enabled: true,
		config:  &c,
	}
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

// GetDescription EditorPlugin is a bridge between iris and the alm-tools, the browser-based IDE for client-side sources.
func (e *Editor) GetDescription() string {
	return "A bridge between iris and the alm-tools, the browser-based IDE."
}

// we use that editorWriter to prefix the editor's output with "Editor Adaptor: "
type editorWriter struct {
	underline io.Writer
}

// Run starts the editor's server.
//
// Developers should call the `Stop` to shutdown the editor's server when main server will be closed.
func (e *Editor) Run(logger func(format string, a ...interface{})) {
	if logger == nil {
		logger = NoOpLogger
	}

	e.log = logger
	if e.config.Hostname == "" {
		e.config.Hostname = "0.0.0.0"
	}

	if e.config.Port <= 0 {
		e.config.Port = DefaultPort
	}

	if s, err := filepath.Abs(e.config.WorkingDir); err == nil {
		e.config.WorkingDir = s
	}

	e.start()
}

// Stop kills the editor's server.
func (e *Editor) Stop() {
	if e.process != nil {
		err := e.process.Kill()
		if err != nil {
			e.log("error while trying to terminate the Editor,please kill this process by yourself, process id: %d", e.process.Pid)
		}
	}
}

// start starts the job
func (e *Editor) start() {
	if e.config.Username == "" || e.config.Password == "" {
		e.log("error before running alm-tools. You have to set username & password for security reasons, otherwise this adaptor won't run.")
		return
	}

	if !npm.NodeModuleExists("alm/bin/alm") {
		e.log("installing alm-tools, please wait...")
		res := npm.NodeModuleInstall("alm")
		if res.Error != nil {
			e.log(res.Error.Error())
			return
		}
		e.log(res.Message)
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
					e.log(prefix + outputScanner.Text())
				}
			}()

			errReader, err := cmd.StderrPipe()
			if err == nil {
				errScanner := bufio.NewScanner(errReader)
				go func() {
					for errScanner.Scan() {
						e.log(prefix + errScanner.Text())
					}
				}()
			}
		}
	}

	err := cmd.Start()
	if err != nil {
		e.log(prefix + err.Error())
		return
	}

	// no need, alm-tools post these
	// e.logger.Printf("Editor is running at %s:%d | %s", e.config.Hostname, e.config.Port, e.config.WorkingDir)
}
