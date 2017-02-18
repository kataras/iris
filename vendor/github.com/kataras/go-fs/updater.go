package fs

import (
	"bufio"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"github.com/kataras/go-errors"
	"io"
	"os"
	"os/exec"
)

// updater.go Go app updater hosted on github, based on 'releases & tags',
// unique and simple source code, no signs or other 'secure' methods.
//
// Note: the previous installed files(in $GOPATH) should not be edited before, if edited the go get tool will fail to upgrade these packages.
//
// tag name (github version) should be compatible with the Semantic Versioning 2.0.0
// Read more about Semantic Versioning 2.0.0: http://semver.org/
//
// quick example:
// package main
//
// import (
// 	"github.com/kataras/go-fs"
// 	"fmt"
// )
//
// func main(){
// 	fmt.Println("Current version is: 0.0.3")
//
// 	updater, err := fs.GetUpdater("kataras","rizla", "0.0.3")
// 	if err !=nil{
// 		panic(err)
// 	}
//
// 	updated := updater.Run()
// 	_ = updated
// }

var (
	errUpdaterUnknown = errors.New("Updater: Unknown error: %s")
	errCantFetchRepo  = errors.New("Updater: Error while trying to fetch the repository: %s. Trace: %s")
	errAccessRepo     = errors.New("Updater: Couldn't access to the github repository, please make sure you're connected to the internet")
)

// Updater is the base struct for the Updater feature
type Updater struct {
	currentVersion *version.Version
	latestVersion  *version.Version
	owner          string
	repo           string
}

// GetUpdater returns a new Updater based on a github repository and the latest local release version(string, example: "4.2.3" or "v4.2.3-rc1")
func GetUpdater(owner string, repo string, currentReleaseVersion string) (*Updater, error) {
	client := github.NewClient(nil) // unuthenticated client, 60 req/hour
	///TODO: rate limit error catching( impossible to same client checks 60 times for github updates, but we should do that check)

	// get the latest release, delay depends on the user's internet connection's download speed
	latestRelease, response, err := client.Repositories.GetLatestRelease(owner, repo)
	if err != nil {
		return nil, errCantFetchRepo.Format(owner+":"+repo, err)
	}

	if c := response.StatusCode; c != 200 && c != 201 && c != 202 && c != 301 && c != 302 && c == 304 {
		return nil, errAccessRepo
	}

	currentVersion, err := version.NewVersion(currentReleaseVersion)
	if err != nil {
		return nil, err
	}

	latestVersion, err := version.NewVersion(*latestRelease.TagName)
	if err != nil {
		return nil, err
	}

	u := &Updater{
		currentVersion: currentVersion,
		latestVersion:  latestVersion,
		owner:          owner,
		repo:           repo,
	}

	return u, nil
}

// HasUpdate returns true if a new update is available
// the second output parameter is the latest ,full, version
func (u *Updater) HasUpdate() (bool, string) {
	return u.currentVersion.LessThan(u.latestVersion), u.latestVersion.String()
}

var (
	// DefaultUpdaterAlreadyInstalledMessage "\nThe latest version '%s' was already installed."
	DefaultUpdaterAlreadyInstalledMessage = "\nThe latest version '%s' was already installed."
)

// Run runs the update, returns true if update has been found and installed, otherwise false
func (u *Updater) Run(setters ...optionSetter) bool {
	opt := &Options{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Silent: false} // default options

	for _, setter := range setters {
		setter.Set(opt)
	}

	writef := func(s string, a ...interface{}) {
		if !opt.Silent {
			opt.Stdout.Write([]byte(fmt.Sprintf(s, a...)))
		}
	}

	has, v := u.HasUpdate()
	if has {

		var scanner *bufio.Scanner
		if opt.Stdin != nil {
			scanner = bufio.NewScanner(opt.Stdin)
		}

		shouldProceedUpdate := func() bool {
			return shouldProceedUpdate(scanner)
		}

		writef("\nA newer version has been found[%s > %s].\n"+
			"Release notes: %s\n"+
			"Update now?[%s]: ",
			u.latestVersion.String(), u.currentVersion.String(),
			fmt.Sprintf("https://github.com/%s/%s/releases/latest", u.owner, u.repo),
			DefaultUpdaterYesInput[0]+"/n")

		if shouldProceedUpdate() {
			if !opt.Silent {
				finish := ShowIndicator(opt.Stdout, true)

				defer func() {
					finish <- true
				}()
			}
			// go get -u github.com/:owner/:repo
			cmd := exec.Command("go", "get", "-u", fmt.Sprintf("github.com/%s/%s", u.owner, u.repo))
			cmd.Stdout = opt.Stdout
			cmd.Stderr = opt.Stderr

			if err := cmd.Run(); err != nil {
				writef("\nError while trying to get the package: %s.", err.Error())
			}

			writef("\010\010\010") // remove the loading bars
			writef("Update has been installed, current version: %s. Please re-start your App.\n", u.latestVersion.String())

			// TODO: normally, this should be in dev-mode machine, so a 'go build' and' & './$executable' on the current working path should be ok
			// for now just log a message to re-run the app manually
			//writef("\nUpdater was not able to re-build and re-run your updated App.\nPlease run your App again, by yourself.")
			return true
		}

	} else {
		writef(fmt.Sprintf(DefaultUpdaterAlreadyInstalledMessage, v))
	}

	return false
}

// DefaultUpdaterYesInput the string or character which user should type to proceed the update, if !silent
var DefaultUpdaterYesInput = [...]string{"y", "yes", "nai", "si"}

func shouldProceedUpdate(sc *bufio.Scanner) bool {
	silent := sc == nil

	inputText := ""
	if !silent {
		if sc.Scan() {
			inputText = sc.Text()
		}
	}

	for _, s := range DefaultUpdaterYesInput {
		if inputText == s {
			return true
		}
	}
	// if silent, then return 'yes/true' always
	return silent
}

// Options the available options used iside the updater.Run func
type Options struct {
	Silent bool
	// Stdin specifies the process's standard input.
	// If Stdin is nil, the process reads from the null device (os.DevNull).
	// If Stdin is an *os.File, the process's standard input is connected
	// directly to that file.
	// Otherwise, during the execution of the command a separate
	// goroutine reads from Stdin and delivers that data to the command
	// over a pipe. In this case, Wait does not complete until the goroutine
	// stops copying, either because it has reached the end of Stdin
	// (EOF or a read error) or because writing to the pipe returned an error.
	Stdin io.Reader

	// Stdout and Stderr specify the process's standard output and error.
	//
	// If either is nil, Run connects the corresponding file descriptor
	// to the null device (os.DevNull).
	//
	// If Stdout and Stderr are the same writer, at most one
	// goroutine at a time will call Write.
	Stdout io.Writer
	Stderr io.Writer
}

// Set implements the optionSetter
func (o *Options) Set(main *Options) {
	main.Silent = o.Silent
}

type optionSetter interface {
	Set(*Options)
}

// OptionSet sets an option
type OptionSet func(*Options)

// Set implements the optionSetter
func (o OptionSet) Set(main *Options) {
	o(main)
}

// Silent sets the Silent option to the 'val'
func Silent(val bool) OptionSet {
	return func(o *Options) {
		o.Silent = val
	}
}

// Stdin specifies the process's standard input.
// If Stdin is nil, the process reads from the null device (os.DevNull).
// If Stdin is an *os.File, the process's standard input is connected
// directly to that file.
// Otherwise, during the execution of the command a separate
// goroutine reads from Stdin and delivers that data to the command
// over a pipe. In this case, Wait does not complete until the goroutine
// stops copying, either because it has reached the end of Stdin
// (EOF or a read error) or because writing to the pipe returned an error.
func Stdin(val io.Reader) OptionSet {
	return func(o *Options) {
		o.Stdin = val
	}
}

// Stdout specify the process's standard output and error.
//
// If either is nil, Run connects the corresponding file descriptor
// to the null device (os.DevNull).
//
func Stdout(val io.Writer) OptionSet {
	return func(o *Options) {
		o.Stdout = val
	}
}

// Stderr specify the process's standard output and error.
//
// If Stdout and Stderr are the same writer, at most one
// goroutine at a time will call Write.
func Stderr(val io.Writer) OptionSet {
	return func(o *Options) {
		o.Stderr = val
	}
}

// simple way to compare version is to make them numbers
// and remove any dots and 'v' or 'version' or 'release'
// so
// the v4.2.2 will be 422
// which is bigger from v4.2.1 (which will be 421)
// also a version could be something like: 1.0.0-beta+exp.sha.5114f85
// so we should add a number of any alpha,beta,rc and so on
// maybe this way is not the best but I think it will cover our needs
// and the simplicity of source I keep to all of my packages.
//var removeChars = [...]string{".","v","version","prerelease","pre-release","release","-","alpha","beta","rc"}
// or just remove any non-numeric chars using regex...
// ok.. just found a better way, to use a third-party package 'go-version' which will cover all version formats
//func parseVersion(s string) int {
