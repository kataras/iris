package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/iris-contrib/errors"
	"github.com/kataras/cli"
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
	runCmd.Stderr = os.Stderr
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

		// check if we have a path,change the workingdir and programpath
		if lidx := strings.LastIndexByte(programPath, os.PathSeparator); lidx > 0 { // no /
			workingDir = workingDir + utils.PathSeparator + programPath[0:lidx]
			programPath = programPath[lidx+1:]
		} else if lidx := strings.LastIndexByte(programPath, '/'); lidx > 0 { // no /
			workingDir = workingDir + "/" + programPath[0:lidx]
			programPath = programPath[lidx+1:]
		}

		executablePath = programPath[:len(programPath)-3]
		if isWindows {
			executablePath += ".exe"
		}

	}

	subfiles, err := ioutil.ReadDir(workingDir)
	if err != nil {
		printer.Dangerf(err.Error())
		return err
	}
	var paths []string
	paths = append(paths, workingDir)
	for _, subfile := range subfiles {
		if subfile.IsDir() {
			if abspath, err := filepath.Abs(workingDir + utils.PathSeparator + subfile.Name()); err == nil {
				paths = append(paths, abspath)
			}

		}
	}

	// run the file watcher before all, because the user maybe has a go syntax error before the first run
	utils.WatchDirectoryChanges(paths, func(fname string) {
		//remove the working dir from the fname path, printer should only print the relative changed file ( from the project's path)
		fname = fname[len(workingDir)+1:]

		if (filepath.Ext(fname) == goExt) || (!isWindows && strings.Contains(fname, goExt)) { // on !windows it sends a .gooutput_RANDOM_STRINGHERE, Note that: we do contains instead of HasPrefix
			filenameCh <- fname
		}

	}, printer)

	if err := build(programPath); err != nil {
		printer.Dangerf(err.Error())
		return err
	}

	runCmd, err := run(executablePath, true)

	if err != nil {
		printer.Dangerf(err.Error())
		return err
	}
	// here(below), we don't return the error because the -help command doesn't help the user for these errors.
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

				printer.Infof("\n[OP: %d] File %s changed, reloading...", atomic.LoadUint32(&times), fname)

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
