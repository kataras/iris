package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kataras/golog"
)

var debug = false

var logger = golog.Default

func init() {
	if len(os.Args) > 1 && os.Args[1] == "debug" {
		debug = true
	}

	if debug {
		logger.SetLevel("debug")
	}
}

var bundles = []bundle{
	{
		names:            []string{"dotnet"},
		installDir:       "./platforms/dotnet",
		installArguments: []string{"-NoPath", "-InstallDir", "$installDir", "-Channel", "Current", "-Version", "3.0.100-preview6-012264"},
	},
	{
		names:            []string{"node", "npm"},
		installDir:       "./platforms/node", // do no change that.
		installArguments: []string{"$installDir", "12.4.0"},
	},
	{
		names:            []string{"git"},
		installDir:       "./platforms/git",
		installArguments: []string{"-InstallDir", "$installDir"},
	},
	{
		names:            []string{"go"}, // get-only, at least for now.
		installDir:       "./platforms/go",
		installArguments: []string{"-InstallDir", "$installDir"},
	},
	{
		names:      []string{"bombardier"},
		installDir: "./platforms/bombardier",
	},
}

func install(b bundle) error {
	for _, name := range b.names {
		switch name {
		case "dotnet":
			return installDotnet(b)
		case "node", "nodejs", "npm":
			return installNode(b)
		case "git":
			return installGit(b)
		case "bombardier":
			return installBombardier(b)
		}
	}

	return nil
}

type bundle struct {
	names      []string
	installDir string

	installArguments []string
}

func (b bundle) parseArguments() []string {
	for i, arg := range b.installArguments {
		if arg[0] == '$' {
			// let's not use reflection here.
			switch arg[1:] {
			case "name":
				b.installArguments[i] = b.names[0]
			case "installDir":
				if runtime.GOOS == "windows" {
					b.installDir = filepath.FromSlash(b.installDir)
				}
				b.installArguments[i] = b.installDir
			default:
				panic(arg + " not a bundle struct field")
			}
		}
	}

	return b.installArguments
}

type platform struct {
	executable string
}

func (p *platform) text(args ...string) string {
	cmd := exec.Command(p.executable, args...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err)
		return ""
	}

	return string(b)
}

func (p *platform) exec(args ...string) error {
	cmd := exec.Command(p.executable, args...)
	return cmd.Run()
}

func (p *platform) attach(logLevel string, args ...string) error {
	cmd := exec.Command(p.executable, args...)
	attachCmd(logLevel, cmd)
	return cmd.Run()
}

func attachCmd(logLevel string, cmd *exec.Cmd) *exec.Cmd {
	level := golog.ParseLevel(logLevel)
	outputReader, err := cmd.StdoutPipe()
	if err == nil {
		outputScanner := bufio.NewScanner(outputReader)

		go func() {
			defer outputReader.Close()
			for outputScanner.Scan() {
				logger.Log(level, outputScanner.Text())
			}
		}()

		errReader, err := cmd.StderrPipe()
		if err == nil {
			errScanner := bufio.NewScanner(errReader)
			go func() {
				defer errReader.Close()
				for errScanner.Scan() {
					logger.Log(level, errScanner.Text()) // don't print at error.
				}
			}()
		}
	}

	return cmd
}

func resolveFilename(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	return name
}

func getPlatform(name string) (p *platform) {
	for _, b := range bundles {
		for _, bName := range b.names {
			if bName == name {

				// temporarily set the path env to the installation directories
				// in order the exec.LookPath to check for programs there too.
				pathEnv := os.Getenv("PATH")
				if len(pathEnv) > 1 {
					if pathEnv[len(pathEnv)-1] != ';' {
						pathEnv += ";"
					}
				}

				pathEnv += b.installDir + "/bin;"
				pathEnv += b.installDir

				os.Setenv("PATH", pathEnv)
				executable, err := exec.LookPath(name)
				if err != nil {
					logger.Infof("%s executable couldn't be retrieved by PATH or PATHTEXT. Installation started...", name)

					err = install(b)
					if err != nil {
						logger.Fatalf("unable to auto-install %s, please do it yourself: %v", name, err)
					}

					name = resolveFilename(name)
					// first check for installDir/bin/+name before the installDir/+name to
					// find the installed executable (we could return it from our scripts but we don't).
					binExecutable := b.installDir + "/bin/" + name
					if _, err = os.Stat(binExecutable); err == nil {
						executable = binExecutable
					} else {
						executable = b.installDir + "/" + name
					}
				}

				return &platform{
					executable: executable,
				}
			}
		}
	}

	logger.Fatalf("%s not found", name)
	return nil
}

func main() {
	dotnet := getPlatform("dotnet")
	dotnetVersion := dotnet.text("--version")
	logger.Info("Dotnet version: ", dotnetVersion)

	node := getPlatform("node")
	nodeVersion := node.text("--version")
	logger.Info("Nodejs version: ", nodeVersion)

	npm := getPlatform("npm")
	npmVersion := npm.text("--version")
	logger.Info("NPM version: ", npmVersion)

	git := getPlatform("git")
	gitVersion := git.text("--version")
	logger.Info("Git version: ", gitVersion)

	golang := getPlatform("go")
	goVersion := golang.text("version")
	logger.Info("Go version: ", goVersion)

	bombardier := getPlatform("bombardier")
	bombardierVersion := bombardier.text("--version")
	logger.Info("Bombardier version: ", bombardierVersion)
}

func installBombardier(b bundle) error {
	const (
		repo          = "github.com/codesenberg/bombardier"
		latestVersion = "1.2.4"
	)

	dst := filepath.Join(os.Getenv("GOPATH"), "/src", repo)
	os.RemoveAll(dst) // remove any copy that $GOPATH may have.

	if err := getPlatform("git").exec("clone", "https://"+repo, dst); err != nil {
		return err
	}

	executable := resolveFilename(b.names[0])
	executableOutput := filepath.Join(b.installDir, executable)

	return getPlatform("go").attach("info", "build", "-ldflags", "-X main.version="+latestVersion, "-o", executableOutput, dst)
}
