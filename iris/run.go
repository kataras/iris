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
	goExt          = ".go"
)

func runAndWatch(flags cli.Flags) error {
	if len(os.Args) <= 2 {
		err := errInvalidArgs.Format(strings.Join(os.Args, ","))
		printer.Dangerf(err.Error())
		return err
	}
	programPath := ""
	filenameCh := make(chan string)

	if len(os.Args) > 2 { // iris run main.go
		programPath = os.Args[2]
		if programPath[len(programPath)-1] == '/' {
			programPath = programPath[0 : len(programPath)-1]
		}

		if filepath.Ext(programPath) != goExt {
			return errInvalidExt.Format(programPath)
		}
	}
	// here(below), we don't return the error because the -help command doesn't help the user for these errors.

	// run the file watcher before all, because the user maybe has a go syntax error before the first run
	utils.WatchDirectoryChanges(workingDir, func(fname string) {
		if filepath.Ext(fname) == goExt {
			filenameCh <- fname
		}

	}, printer)

	// we don't use go build and run from the executable, for performance reasons, no need this is a development action already
	goRun := utils.CommandBuilder("go", "run", programPath)
	goRun.Dir = workingDir
	goRun.Stdout = os.Stdout
	goRun.Stderr = os.Stderr
	if err := goRun.Start(); err != nil {
		printer.Dangerf("\n [ERROR] Failed to run the %s iris program. Trace: %s", programPath, err.Error())
		return nil
	}

	isWindows := runtime.GOOS == "windows"
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
				printer.Warningf("\n/-%d-/  File '%s' changed, re-running...", atomic.LoadUint32(&times), fname)
				// force kill, sometimes runCmd.Process.Kill or Signal(os.Kill) doesn't kill the child of the go's go run command ( which is the iris program)
				if isWindows {
					utils.CommandBuilder("taskkill", "/F", "/T", "/PID", strconv.Itoa(goRun.Process.Pid)).Run()
				} else {
					utils.CommandBuilder("kill", "-INT", "-"+strconv.Itoa(goRun.Process.Pid)).Run()
				}

				goRun = utils.CommandBuilder("go", "run", programPath)
				goRun.Dir = workingDir
				goRun.Stderr = os.Stderr

				if err := goRun.Start(); err != nil {
					printer.Warningf("\n [ERROR ON RELOAD] Failed to run the %s iris program. Trace: %s", programPath, err.Error())

				} else {
					atomic.AddUint32(&times, 1)
					// don't print success on anything here because we may have error on iris itself, no need to print any message we are no spammers.
				}
			}
		}
	}

}
