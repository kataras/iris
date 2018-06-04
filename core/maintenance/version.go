package maintenance

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kataras/iris/core/maintenance/version"

	"github.com/kataras/golog"
	"github.com/kataras/survey"
)

const (
	// Version is the string representation of the current local Iris Web Framework version.
	Version = "10.6.6"
)

// CheckForUpdates checks for any available updates
// and asks for the user if want to update now or not.
func CheckForUpdates() {
	v := version.Acquire()
	updateAvailale := v.Compare(Version) == version.Smaller

	if updateAvailale {
		if confirmUpdate(v) {
			installVersion()
			return
		}
	}
}

func confirmUpdate(v version.Version) bool {
	// on help? when asking for installing the new update.
	ignoreUpdatesMsg := "Would you like to ignore future updates? Disable the version checker via:\napp.Run(..., iris.WithoutVersionChecker)"

	// if update available ask for update action.
	shouldUpdateNowMsg :=
		fmt.Sprintf("A new version is available online[%s > %s]. Type '?' for help.\nRelease notes: %s.\nUpdate now?",
			v.String(), Version, v.ChangelogURL)

	var confirmUpdate bool
	survey.AskOne(&survey.Confirm{
		Message: shouldUpdateNowMsg,
		Help:    ignoreUpdatesMsg,
	}, &confirmUpdate, nil)
	return confirmUpdate // it's true only when update was available and user typed "yes".
}

func installVersion() {
	golog.Infof("Downloading...\n")
	repo := "github.com/kataras/iris/..."
	cmd := exec.Command("go", "get", "-u", "-v", repo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Run(); err != nil {
		golog.Warnf("unexpected message while trying to go get,\nif you edited the original source code then you've to remove the whole $GOPATH/src/github.com/kataras folder and execute `go get -u github.com/kataras/iris/...` manually\n%v", err)
		return
	}

	golog.Infof("Update process finished.\nManual rebuild and restart is required to apply the changes...\n")
	return
}

/* Author's note:
We could use github webhooks to automatic notify for updates
when a new update is pushed to the repository
even when server is already started and running but this would expose
a route which dev may don't know about, so let it for now but if
they ask it then I should add an optional configuration field
to "live/realtime update" and implement the idea (which is already implemented in the iris-go server).
*/

/* Author's note:
The old remote endpoint for version checker is still available on the server for backwards
compatibility with older clients, it will stay there for a long period of time.
*/
