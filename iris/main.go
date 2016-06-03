package main

import (
	"os"

	"strings"

	"runtime"

	"github.com/fatih/color"
	"github.com/kataras/cli"
	"github.com/kataras/iris"
	"github.com/kataras/iris/utils"
)

const (
	// PackagesURL the url to download all the packages
	PackagesURL = "https://github.com/iris-contrib/iris-command-assets/archive/master.zip"
	// PackagesExportedName the folder created after unzip
	PackagesExportedName = "iris-command-assets-master"
)

var (
	app *cli.App
	// SuccessPrint prints with a green color
	SuccessPrint = color.New(color.FgGreen).Add(color.Bold).PrintfFunc()
	// InfoPrint prints with the cyan color
	InfoPrint          = color.New(color.FgHiCyan).Add(color.Bold).PrintfFunc()
	packagesInstallDir = os.Getenv("GOPATH") + utils.PathSeparator + "src" + utils.PathSeparator + "github.com" + utils.PathSeparator + "kataras" + utils.PathSeparator + "iris" + utils.PathSeparator + "iris" + utils.PathSeparator
	packagesDir        = packagesInstallDir + PackagesExportedName + utils.PathSeparator
)

func init() {
	app = cli.NewApp("iris", "Command line tool for Iris web framework", "0.0.2")
	app.Command(cli.Command("version", "\t      prints your iris version").Action(func(cli.Flags) error { app.Printf("%s", iris.Version); return nil }))

	createCmd := cli.Command("create", "create a project to a given directory").
		Flag("dir", "./", "-d ./ creates an iris starter kit to the current directory").
		Flag("type", "basic", "creates the project based on the -t package. Currently, available types are 'basic' & 'static'").
		Action(create)

	app.Command(createCmd)
}

func main() {
	app.Run(func(cli.Flags) error { return nil })
}

func create(flags cli.Flags) (err error) {

	/*	///TODO: add a version.json and check if the repository's version is bigger than local and then do the downloadPackages.

		if !utils.DirectoryExists(packagesDir) {
			downloadPackages()
		}
	*/

	// currently: always download packages, because I'm adding new things to the packages every day.
	downloadPackages()

	targetDir := flags.String("dir")

	if strings.HasPrefix(targetDir, "./") || strings.HasPrefix(targetDir, "."+utils.PathSeparator) {
		currentWdir, err := os.Getwd()
		if err != nil {
			return err
		}
		targetDir = currentWdir + utils.PathSeparator + targetDir[2:]
	}

	createPackage(flags.String("type"), targetDir)
	return
}

func downloadPackages() {
	_, err := utils.Install(PackagesURL, packagesInstallDir)
	if err != nil {
		app.Printf("\nProblem while downloading the assets from the internet for the first time. Trace: %s", err.Error())
	}
}

func createPackage(packageName string, targetDir string) error {
	packageDir := packagesDir + packageName
	err := utils.CopyDir(packageDir, targetDir)
	if err != nil {
		app.Printf("\nProblem while copying the %s package to the %s. Trace: %s", packageName, targetDir, err.Error())
		return err
	}

	InfoPrint("\n%s package was installed successfully", packageName)

	// build & run the server

	// go build
	buildCmd := utils.CommandBuilder("go", "build")
	if targetDir[len(targetDir)-1] != os.PathSeparator || targetDir[len(targetDir)-1] != '/' {
		targetDir += utils.PathSeparator
	}
	buildCmd.Dir = targetDir + "backend"
	buildCmd.Stderr = os.Stderr
	err = buildCmd.Start()
	if err != nil {
		app.Printf("\n Failed to build the %s package. Trace: %s", packageName, err.Error())
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
		app.Printf("\n Failed to run the %s package. Trace: %s", packageName, err.Error())
	}
	runCmd.Wait()

	return err
}
