package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kataras/cli"
	"github.com/kataras/iris/utils"
)

const (
	// PackagesURL the url to download all the packages
	PackagesURL = "https://github.com/iris-contrib/iris-command-assets/archive/master.zip"
	// PackagesExportedName the folder created after unzip
	PackagesExportedName = "iris-command-assets-master"
)

var (
	packagesInstallDir = utils.AssetsDirectory + utils.PathSeparator + "iris-command-assets" + utils.PathSeparator
	// packages should install with go get before create the package
	packagesDependencies = []string{"github.com/iris-contrib/middleware/logger"}
)

func isValidInstallDir(targetDir string) bool {
	// https://github.com/kataras/iris/issues/237
	gopath := os.Getenv("GOPATH")
	// remove the last ;/: for any case before the split
	if idxLSep := strings.IndexByte(gopath, os.PathListSeparator); idxLSep == len(gopath)-1 {
		gopath = gopath[0 : len(gopath)-2]
	}

	// check if we have more than one gopath
	gopaths := strings.Split(gopath, string(os.PathListSeparator))
	// the package MUST be installed only inside a valid gopath, if not then print an error to the user.
	for _, gpath := range gopaths {
		if strings.HasPrefix(targetDir, gpath+utils.PathSeparator) {
			return true
		}
	}
	return false
}

func create(flags cli.Flags) (err error) {

	targetDir, err := filepath.Abs(flags.String("dir"))
	if err != nil {
		panic(err)
	}

	if !isValidInstallDir(targetDir) {
		printer.Dangerf("\nPlease make sure you are targeting a directory inside $GOPATH, type iris -h for help.")
		return
	}

	if !utils.DirectoryExists(packagesInstallDir) || !flags.Bool("offline") {
		// install/update go dependencies at the same time downloading the zip from the github iris-contrib assets
		finish := make(chan bool)
		go func() {
			go func() {
				for _, source := range packagesDependencies {
					gogetCmd := utils.CommandBuilder("go", "get", source)
					if msg, err := gogetCmd.CombinedOutput(); err != nil {
						panic("Unable to go get " + source + " please make sure you're connected to the internet.\nSolution: Remove your $GOPATH/src/github.com/iris-contrib/middleware folder and re-run the iris create\nReason:\n" + string(msg))
					}
				}
				finish <- true

			}()

			downloadPackages()
			<-finish
		}()

		<-finish
		close(finish)
	}
	createPackage(flags.String("type"), targetDir)
	return
}

func downloadPackages() {
	errMsg := "\nProblem while downloading the assets from the internet for the first time. Trace: %s"

	installedDir, err := utils.Install(PackagesURL, packagesInstallDir)
	if err != nil {
		printer.Dangerf(errMsg, err.Error())
		return
	}

	// installedDir is the packagesInstallDir+PackagesExportedName, we will copy these contents to the parent, to the packagesInstallDir, because of import paths.

	err = utils.CopyDir(installedDir, packagesInstallDir)
	if err != nil {
		printer.Dangerf(errMsg, err.Error())
		return
	}

	// we don't exit on errors here.

	// try to remove the unzipped folder
	utils.RemoveFile(installedDir[0 : len(installedDir)-1])
}

func createPackage(packageName string, targetDir string) error {
	installTo := targetDir // os.Getenv("GOPATH") + utils.PathSeparator + "src" + utils.PathSeparator + targetDir

	packageDir := packagesInstallDir + utils.PathSeparator + packageName
	err := utils.CopyDir(packageDir, installTo)
	if err != nil {
		printer.Dangerf("\nProblem while copying the %s package to the %s. Trace: %s", packageName, installTo, err.Error())
		return err
	}

	// now replace main.go's 'github.com/iris-contrib/iris-command-assets/basic/' with targetDir
	// hardcode all that, we don't have anything special and neither will do
	targetDir = strings.Replace(targetDir, "\\", "/", -1) // for any case
	mainFile := installTo + utils.PathSeparator + "backend" + utils.PathSeparator + "main.go"

	input, err := ioutil.ReadFile(mainFile)
	if err != nil {
		printer.Warningf("Error while preparing main file: %#v", err)
	}

	output := strings.Replace(string(input), "github.com/iris-contrib/iris-command-assets/"+packageName+"/", filepath.Base(targetDir)+"/", -1)

	err = ioutil.WriteFile(mainFile, []byte(output), 0777)
	if err != nil {
		printer.Warningf("Error while preparing main file: %#v", err)
	}

	printer.Infof("%s package was installed successfully [%s]", packageName, installTo)

	// build & run the server

	// go build
	buildCmd := utils.CommandBuilder("go", "build")
	if installTo[len(installTo)-1] != os.PathSeparator || installTo[len(installTo)-1] != '/' {
		installTo += utils.PathSeparator
	}
	buildCmd.Dir = installTo + "backend"
	buildCmd.Stderr = os.Stderr
	err = buildCmd.Start()
	if err != nil {
		printer.Warningf("\n Failed to build the %s package. Trace: %s", packageName, err.Error())
	}
	buildCmd.Wait()
	print("\n\n")

	// run backend/backend(.exe)
	executable := "backend"
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}

	runCmd := utils.CommandBuilder("." + utils.PathSeparator + executable)
	runCmd.Dir = buildCmd.Dir
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	err = runCmd.Start()
	if err != nil {
		printer.Warningf("\n Failed to run the %s package. Trace: %s", packageName, err.Error())
	}
	runCmd.Wait()

	return err
}
