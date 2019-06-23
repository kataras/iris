package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kataras/golog"
)

var bundles = []bundle{
	{
		name:             "dotnet",
		installDir:       "./dotnet_bin",
		installArguments: []string{"-NoPath", "-InstallDir", "$installDir", "-Channel", "Current", "-Version", "3.0.100-preview6-012264"},
	},
}

func install(b bundle) error {
	switch b.name {
	case "dotnet":
		return installDotnet(b)
	default:
		return nil
	}
}

type bundle struct {
	name       string
	installDir string

	installArguments []string
}

func (b bundle) parseArguments() []string {
	for i, arg := range b.installArguments {
		if arg[0] == '$' {
			// let's not use reflection here.
			switch arg[1:] {
			case "name":
				b.installArguments[i] = b.name
			case "installDir":
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

func (p *platform) exec(args ...string) string {
	cmd := exec.Command(p.executable, args...)
	b, err := cmd.Output()
	if err != nil {
		golog.Error(err)
		return ""
	}

	return string(b)
}

func (p *platform) attach(args ...string) error {
	cmd := exec.Command(p.executable, args...)
	attachCmd(cmd)
	return cmd.Run()
}

func attachCmd(cmd *exec.Cmd) {
	outputReader, err := cmd.StdoutPipe()
	if err == nil {
		outputScanner := bufio.NewScanner(outputReader)

		go func() {
			defer outputReader.Close()
			for outputScanner.Scan() {
				golog.Println(outputScanner.Text())
			}
		}()

		errReader, err := cmd.StderrPipe()
		if err == nil {
			errScanner := bufio.NewScanner(errReader)
			go func() {
				defer errReader.Close()
				for errScanner.Scan() {
					golog.Println(errScanner.Text())
				}
			}()
		}
	}
}

func getPlatform(name string) (p *platform) {
	for _, b := range bundles {
		if b.name != name {
			continue
		}

		// temporarily set the path env to the installation directories
		// in order the exec.LookPath to check for programs there too.
		pathEnv := os.Getenv("PATH")
		if len(pathEnv) > 1 {
			if pathEnv[len(pathEnv)-1] != ';' {
				pathEnv += ";"
			}
		}

		pathEnv += b.installDir
		os.Setenv("PATH", pathEnv)
		executable, err := exec.LookPath(name)
		if err != nil {
			golog.Debugf("%s executable couldn't be found from PATH. Trying to install it...", name)

			err = install(b)
			if err != nil {
				golog.Fatalf("unable to auto-install %s, please do it yourself: %v", name, err)
			}

			executable = filepath.Join(b.installDir, name)
			if runtime.GOOS == "windows" {
				executable += ".exe"
			}
		}

		return &platform{
			executable: executable,
		}
	}

	golog.Fatalf("%s not found", name)
	return nil
}

func main() {
	golog.SetLevel("debug")

	dotnet := getPlatform("dotnet")
	dotnetVersion := dotnet.exec("--version")
	golog.Infof("Dotnet version: %s", dotnetVersion)
}
