package main

import (
	"io/ioutil"
	"os"
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
)

func create(flags cli.Flags) (err error) {

	if !utils.DirectoryExists(packagesInstallDir) || !flags.Bool("offline") {
		downloadPackages()
	}

	targetDir := flags.String("dir")

	// remove first and last / if any
	if strings.HasPrefix(targetDir, "./") || strings.HasPrefix(targetDir, "."+utils.PathSeparator) {
		targetDir = targetDir[2:]
	}
	if targetDir[len(targetDir)-1] == '/' {
		targetDir = targetDir[0 : len(targetDir)-1]
	}
	//

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
	installTo := os.Getenv("GOPATH") + utils.PathSeparator + "src" + utils.PathSeparator + targetDir

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

	output := strings.Replace(string(input), "github.com/iris-contrib/iris-command-assets/"+packageName+"/", targetDir+"/", -1)

	err = ioutil.WriteFile(mainFile, []byte(output), 0644)
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
