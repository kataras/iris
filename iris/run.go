package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/kataras/cli"
	"github.com/kataras/iris/errors"
	"github.com/kataras/iris/utils"
)

var (
	errInvalidArgs = errors.New("Invalid arguments [%s], type -h to get assistant")
	errInvalidExt  = errors.New("%s is not a go program")
	errUnexpected  = errors.New("Unexpected error!!! Please post an issue here: https://github.com/kataras/iris/issues")
	errBuild       = errors.New("\n Failed to build the %s iris program. Trace: %s")
	errRun         = errors.New("\n Failed to run the %s iris program. Trace: %s")
	goExt          = ".go"
)

func build(sourcepath string) error {
	goBuild := utils.CommandBuilder("go", "build", sourcepath)
	goBuild.Dir = workingDir
	goBuild.Stdout = os.Stdout
	goBuild.Stderr = os.Stderr
	if err := goBuild.Run(); err != nil {
		ferr := errBuild.Format(sourcepath, err.Error())
		printer.Dangerf(ferr.Error())
		return ferr
	}
	return nil
}

func run(executablePath string, stdout bool) (*utils.Cmd, error) {
	runCmd := utils.CommandBuilder("." + utils.PathSeparator + executablePath)
	runCmd.Dir = workingDir
	if stdout {
		runCmd.Stdout = os.Stdout
	}

	runCmd.Stderr = os.Stderr
	if err := runCmd.Start(); err != nil {
		ferr := errRun.Format(executablePath, err.Error())
		printer.Dangerf(ferr.Error())
		return nil, ferr
	}
	return runCmd, nil
}

func runAndWatch(flags cli.Flags) error {
	if len(os.Args) <= 2 {
		err := errInvalidArgs.Format(strings.Join(os.Args, ","))
		printer.Dangerf(err.Error())
		return err
	}
	isWindows := runtime.GOOS == "windows"
	programPath := ""
	executablePath := ""
	filenameCh := make(chan string)

	if len(os.Args) > 2 { // iris run main.go
		programPath = os.Args[2]
		if programPath[len(programPath)-1] == '/' {
			programPath = programPath[0 : len(programPath)-1]
		}

		if filepath.Ext(programPath) != goExt {
			return errInvalidExt.Format(programPath)
		}
		executablePath = programPath[:len(programPath)-3]
		if isWindows {
			executablePath += ".exe"
		}
	}
	// here(below), we don't return the error because the -help command doesn't help the user for these errors.

	// run the file watcher before all, because the user maybe has a go syntax error before the first run
	utils.WatchDirectoryChanges(workingDir, func(fname string) {
		if (filepath.Ext(fname) == goExt) || (!isWindows && strings.HasPrefix(fname, goExt)) { // on !windows it sends a .gooutput_RANDOM_STRINGHERE
			filenameCh <- fname
		}

	}, printer)

	if err := build(programPath); err != nil {
		return err
	}

	runCmd, err := run(executablePath, true)
	if err != nil {
		return err
	}

	defer func() {
		printer.Dangerf("")
		printer.Panic(errUnexpected)
	}()
	var times uint32 = 1
	for {
		select {
		case fname := <-filenameCh:
			{
				// it's not a warning but I like to use purple color for this message
				if !isWindows {
					fname = " " // we don't want to print the ".gooutput..." so dont print anything as a name
				}
				printer.Infof("\n%d-  File '%s' changed, reloading...", atomic.LoadUint32(&times), fname)

				//kill the prev run

				err := runCmd.Process.Kill()
				if err == nil {
					_, err = runCmd.Process.Wait()
				} else {

					// force kill, sometimes runCmd.Process.Kill or Signal(os.Kill) doesn't kills
					if isWindows {
						err = utils.CommandBuilder("taskkill", "/F", "/T", "/PID", strconv.Itoa(runCmd.Process.Pid)).Run()
					} else {
						err = utils.CommandBuilder("kill", "-INT", "-"+strconv.Itoa(runCmd.Process.Pid)).Run()
					}
				}

				err = build(programPath)
				if err != nil {
					printer.Warningf(err.Error())
				} else {

					if runCmd, err = run(executablePath, false); err != nil {
						printer.Warningf(err.Error())

					} else {
						// we did .Start, but it should be fast so no need to add a sleeper
						printer.Successf("ready!")
						atomic.AddUint32(&times, 1)
					}
				}

			}
		}
	}

}
