package iris

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/hashicorp/go-version"
	"github.com/kataras/go-errors"
)

// global once because is not necessary to check for updates on more than one iris station*
var updateOnce sync.Once

// CheckForUpdates will try to search for newer version of Iris based on the https://github.com/kataras/iris/releases
// If a newer version found then the app will ask the he dev/user if want to update the 'x' version
// if 'y' is pressed then the updater will try to install the latest version
// the updater, will notify the dev/user that the update is finished and should restart the App manually.
// Note: exported func CheckForUpdates exists because of the reason that an update can be executed while Iris is running
func CheckForUpdates(force bool) {
	var (
		updated bool
		err     error
	)

	checker := func() {
		updated, err = update(os.Stdin, os.Stdout, false)
		if err != nil {
			// ignore writer's error
			os.Stdout.Write([]byte("update failed: " + err.Error()))
			return
		}
	}

	if force {
		checker()
	} else {
		updateOnce.Do(checker)
	}

	if updated { // if updated, then do not run the web server
		os.Stdout.Write([]byte("exiting now..."))
		os.Exit(1)
	}

}

//  +------------------------------------------------------------+
//  |                                                            |
//  |      Updater based on github repository's releases         |
//  |                                                            |
//  +------------------------------------------------------------+

// showIndicator shows a silly terminal indicator for a process, close of the finish channel is done here.
func showIndicator(wr io.Writer, newLine bool) chan bool {
	finish := make(chan bool)
	waitDur := 500 * time.Millisecond
	go func() {
		if newLine {
			wr.Write([]byte("\n"))
		}
		wr.Write([]byte("|"))
		wr.Write([]byte("_"))
		wr.Write([]byte("|"))

		for {
			select {
			case v := <-finish:
				{
					if v {
						wr.Write([]byte("\010\010\010")) //remove the loading chars
						close(finish)
						return
					}
				}
			default:
				wr.Write([]byte("\010\010-"))
				time.Sleep(waitDur)
				wr.Write([]byte("\010\\"))
				time.Sleep(waitDur)
				wr.Write([]byte("\010|"))
				time.Sleep(waitDur)
				wr.Write([]byte("\010/"))
				time.Sleep(waitDur)
				wr.Write([]byte("\010-"))
				time.Sleep(waitDur)
				wr.Write([]byte("|"))
			}
		}

	}()

	return finish
}

var updaterYesInput = [...]string{"y", "yes", "nai", "si"}

func shouldProceedUpdate(sc *bufio.Scanner) bool {
	silent := sc == nil

	inputText := ""
	if !silent {
		if sc.Scan() {
			inputText = sc.Text()
		}
	}

	for _, s := range updaterYesInput {
		if inputText == s {
			return true
		}
	}
	// if silent, then return 'yes/true' always
	return silent
}

var (
	errUpdaterUnknown = errors.New("updater: Unknown error: %s")
	errCantFetchRepo  = errors.New("updater: Error while trying to fetch the repository: %s. Trace: %s")
	errAccessRepo     = errors.New("updater: Couldn't access to the github repository, please make sure you're connected to the internet")

	// lastVersionAlreadyInstalledMessage "\nThe latest version '%s' was already installed."
	lastVersionAlreadyInstalledMessage = "the latest version '%s' is already installed."
)

// update runs the updater, returns true if update has been found and installed, otherwise false
func update(in io.Reader, out io.Writer, silent bool) (bool, error) {

	const (
		owner = "kataras"
		repo  = "iris"
	)

	client := github.NewClient(nil) // unuthenticated client, 60 req/hour
	///TODO: rate limit error catching( impossible to same client checks 60 times for github updates, but we should do that check)

	ctx := context.TODO()

	// get the latest release, delay depends on the user's internet connection's download speed
	latestRelease, response, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return false, errCantFetchRepo.Format(owner+":"+repo, err)
	}

	if c := response.StatusCode; c != 200 && c != 201 && c != 202 && c != 301 && c != 302 && c == 304 {
		return false, errAccessRepo
	}

	currentVersion, err := version.NewVersion(Version)
	if err != nil {
		return false, err
	}

	latestVersion, err := version.NewVersion(*latestRelease.TagName)
	if err != nil {
		return false, err
	}

	writef := func(s string, a ...interface{}) {
		if !silent {
			out.Write([]byte(fmt.Sprintf(s, a...)))
		}
	}

	has, v := currentVersion.LessThan(latestVersion), latestVersion.String()
	if has {

		var scanner *bufio.Scanner
		if in != nil {
			scanner = bufio.NewScanner(in)
		}

		shouldProceedUpdate := func() bool {
			return shouldProceedUpdate(scanner)
		}

		writef("A newer version has been found[%s > %s].\n"+
			"Release notes: %s\n"+
			"Update now?[%s]: ",
			latestVersion.String(), currentVersion.String(),
			fmt.Sprintf("https://github.com/%s/%s/releases/latest", owner, repo),
			updaterYesInput[0]+"/n")

		if shouldProceedUpdate() {
			if !silent {
				finish := showIndicator(out, true)

				defer func() {
					finish <- true
				}()
			}
			// go get -u github.com/:owner/:repo
			cmd := exec.Command("go", "get", "-u", fmt.Sprintf("github.com/%s/%s", owner, repo))
			cmd.Stdout = out
			cmd.Stderr = out

			if err := cmd.Run(); err != nil {
				return false, fmt.Errorf("error while trying to get the package: %s", err.Error())
			}

			writef("\010\010\010") // remove the loading bars
			writef("Update has been installed, current version: %s. Please re-start your App.\n", latestVersion.String())

			// TODO: normally, this should be in dev-mode machine, so a 'go build' and' & './$executable' on the current working path should be ok
			// for now just log a message to re-run the app manually
			//writef("\nUpdater was not able to re-build and re-run your updated App.\nPlease run your App again, by yourself.")
			return true, nil
		}

	} else {
		writef(fmt.Sprintf(lastVersionAlreadyInstalledMessage, v))
	}

	return false, nil
}
