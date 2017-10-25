package iris

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/netutil"
	"github.com/kataras/survey"

	"github.com/skratchdot/open-golang/open"
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
	FirstTime       bool   `json:"first_time"`
}

func checkVersion() {
	client := netutil.Client(30 * time.Second)
	r, err := client.PostForm("https://iris-go.com/version", url.Values{"current_version": {Version}})

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

	// shouldUpdate := false
	// prompt := &survey.Confirm{
	// 	Message: shouldUpdateNowMsg,
	// }

	// if err := survey.AskOne(prompt, &shouldUpdate, nil); err != nil {
	// 	return
	// }
	var qs []*survey.Question

	// on help? when asking for installing the new update
	// and when answering "No".
	ignoreUpdatesMsg := "Would you like to ignore future updates? Disable the version checker via:\napp.Run(..., iris.WithoutVersionChecker)"

	if v.UpdateAvailable {
		// if update available ask for update action.
		shouldUpdateNowMsg :=
			fmt.Sprintf("A new version is available online[%s > %s].\nRelease notes: %s.\nUpdate now?",
				v.Version, Version,
				v.ChangelogURL)

		qs = append(qs, &survey.Question{
			Name: "shouldUpdateNow",
			Prompt: &survey.Confirm{
				Message: shouldUpdateNowMsg,
				Help:    ignoreUpdatesMsg,
			},
			Validate: survey.Required,
		})
	}

	// firs time and update available is not relative because if no update often server will decide when to ask this,
	// so separate the actions and if statements here.
	if v.FirstTime {
		// if first time that this server was updated then ask if enjoying the framework.
		qs = append(qs, &survey.Question{
			Name: "enjoyingIris",
			Prompt: &survey.Confirm{
				Message: "Enjoying Iris Framework?",
				Help:    "yes or no",
			},
			Validate: survey.Required,
		})
	}

	// Ask if should update(if available) and enjoying iris(if first time) in the same survey.
	ans := struct {
		ShouldUpdateNow bool `survey:"shouldUpdateNow"`
		EnjoyingIris    bool `survey:"enjoyingIris"`
	}{}

	survey.Ask(qs, &ans)

	if ans.EnjoyingIris {
		// if the answer to the previous survey about enjoying the framework
		// was positive then do the survey (currently only one question and its action).
		qs2 := []*survey.Question{
			{
				Name: "starNow",
				Prompt: &survey.Confirm{
					Message: "Would you mind giving us a star on GitHub? It really helps us out! Thanks for your support:)",
					Help:    "Its free so let's do that, type 'y'",
				},
				Validate: survey.Required,
			},
			/* any future questions should be here, at this second survey. */
		}
		ans2 := struct {
			StarNow bool `survey:"starNow"`
		}{}
		survey.Ask(qs2, &ans2)
		if ans2.StarNow {
			starRepo := "https://github.com/kataras/iris/stargazers"
			if err := open.Run(starRepo); err != nil {
				golog.Warnf("tried to open the browser for you but failed, please give us a star at: %s\n", starRepo)
			}
		}
	}

	// run the updater last, so the user can star the repo and at the same time
	// the app will update her/his local iris.
	if ans.ShouldUpdateNow { // it's true only when update was available and user typed "yes".
		repo := "github.com/kataras/iris/..."
		cmd := exec.Command("go", "get", "-u", "-v", repo)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout

		if err := cmd.Run(); err != nil {
			golog.Warnf("unexpected message while trying to go get,\nif you edited the original source code then you've to remove the whole $GOPATH/src/github.com/kataras folder and execute `go get -u github.com/kataras/iris/...` manually\n%v", err)
			return
		}

		golog.Infof("Update process finished.\nManual rebuild and restart is required to apply the changes...\n")
	} else {
		golog.Infof(ignoreUpdatesMsg)
	}
}
