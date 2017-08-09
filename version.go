package iris

import (
	"bufio"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/kataras/golog"
)

const (
	versionURL = "https://raw.githubusercontent.com/kataras/iris/master/VERSION"
	updateCmd  = "go get -u -f -v github.com/kataras/iris"
)

var checkVersionOnce = sync.Once{}

// CheckVersion checks for any available updates.
func CheckVersion() {
	checkVersionOnce.Do(func() {
		checkVersion()
	})
}

func checkVersion() {

	// open connection and read/write timeouts
	timeout := time.Duration(10 * time.Second)

	transport := http.Transport{
		Dial: func(network string, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, timeout)
			if err != nil {
				golog.Debugf("%v", err)
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(timeout)) // skip error
			return conn, nil
		},
	}

	client := http.Client{
		Transport: &transport,
	}

	r, err := client.Get(versionURL)
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

	var (
		fetchedVersion = string(b)
		changelogURL   string
	)

	// 8.2.1:https://github.com/kataras/iris/blob/master/HISTORY.md#tu-08-august-2017--v821
	if idx := strings.IndexByte(fetchedVersion, ':'); idx > 0 {
		changelogURL = fetchedVersion[idx+1:]
		fetchedVersion = fetchedVersion[0:idx]
	}

	latestVersion, err := version.NewVersion(fetchedVersion)
	if err != nil {
		golog.Debugf("while parsing latest version: %v", err)
		return
	}

	currentVersion, err := version.NewVersion(Version)
	if err != nil {
		golog.Debugf("while parsing current version: %v", err)
		return
	}

	if currentVersion.GreaterThan(latestVersion) {
		golog.Debugf("current version is greater than latest version, report as bug")
		return
	}

	if currentVersion.Equal(latestVersion) {
		return
	}

	// currentVersion.LessThan(latestVersion)

	var updaterYesInput = [...]string{"y", "yes"}

	text := "A more recent version has been found[%s > %s].\n"

	if changelogURL != "" {
		text += "Release notes: %s\n"
	}

	text += "Update now?[%s]: "

	golog.Warnf(text,
		latestVersion.String(), currentVersion.String(),
		changelogURL,
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

	if shouldUpdate {
		goget := strings.Split(updateCmd, " ")
		// go get -u github.com/:owner/:repo
		cmd := exec.Command(goget[0], goget[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout

		if err := cmd.Run(); err != nil {
			golog.Warnf("unexpected message while trying to go get: %v", err)
			return
		}

		golog.Infof("Update process finished, current version: %s.\nManual restart is required to apply the changes.\n", latestVersion.String())
	}
}
