package iris

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/netutil"
)

var checkVersionOnce = sync.Once{}

// CheckVersion checks for any available updates.
func CheckVersion() {
	checkVersionOnce.Do(func() {
		checkVersion()
	})
}

type versionInfo struct {
	Version         string `json:"version"`
	ChangelogURL    string `json:"changelog_url"`
	UpdateAvailable bool   `json:"update_available"`
}

func checkVersion() {
	client := netutil.Client(20 * time.Second)
	r, err := client.PostForm("http://iris-go.com/version", url.Values{"current_version": {Version}})

	if err != nil {
		golog.Debugf("%v", err)
		return
	}
	defer r.Body.Close()

	if r.StatusCode >= 400 {
		return
	}

	b, err := ioutil.ReadAll(r.Body)

	if len(b) == 0 || err != nil {
		golog.Debugf("%v", err)
		return
	}

	v := new(versionInfo)
	if err := json.Unmarshal(b, v); err != nil {
		golog.Debugf("error while unmarshal the response body: %v", err)
		return
	}

	if !v.UpdateAvailable {
		return
	}

	format := "A new version is available online[%s > %s].\n"

	if v.ChangelogURL != "" {
		format += "Release notes: %s\n"
	}

	format += "Update now?[%s]: "

	// currentVersion.LessThan(latestVersion)
	updaterYesInput := [...]string{"y", "yes"}

	golog.Warnf(format, v.Version, Version,
		v.ChangelogURL,
		updaterYesInput[0]+"/n")

	silent := false

	sc := bufio.NewScanner(os.Stdin)

	shouldUpdate := silent

	if !silent {
		if sc.Scan() {
			inputText := sc.Text()

			for _, s := range updaterYesInput {
				if inputText == s {
					shouldUpdate = true
				}
			}
		}
	}

	if !shouldUpdate {
		golog.Infof("Ignore updates? Disable version checker via:\napp.Run(..., iris.WithoutVersionChecker)")
		return
	}

	if shouldUpdate {
		repo := "github.com/kataras/iris"
		cmd := exec.Command("go", "get", "-u", "-f", "-v", repo)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout

		if err := cmd.Run(); err != nil {
			golog.Warnf("unexpected message while trying to go get,\nif you edited the original source code then you've to remove the whole $GOPATH/src/github.com/kataras folder and execute `go get github.com/kataras/iris` manually\n%v", err)
			return
		}

		golog.Infof("Update process finished.\nManual rebuild and restart is required to apply the changes...")
	}
}
