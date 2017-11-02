package version

import (
	"io/ioutil"
	"strings"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/kataras/golog"
	"github.com/kataras/iris/core/netutil"
)

const (
	versionURL = "https://raw.githubusercontent.com/kataras/iris/master/VERSION"
)

func fetch() (*version.Version, string) {
	client := netutil.Client(time.Duration(30 * time.Second))

	r, err := client.Get(versionURL)
	if err != nil {
		golog.Debugf("err: %v\n", err)
		return nil, ""
	}
	defer r.Body.Close()

	if r.StatusCode >= 400 {
		golog.Debugf("Internet connection is missing, updater is unable to fetch the latest Iris version\n", err)
		return nil, ""
	}

	b, err := ioutil.ReadAll(r.Body)

	if len(b) == 0 || err != nil {
		golog.Debugf("err: %v\n", err)
		return nil, ""
	}

	var (
		fetchedVersion = string(b)
		changelogURL   string
	)
	// Example output:
	// Version(8.5.5)
	// 8.5.5:https://github.com/kataras/iris/blob/master/HISTORY.md#tu-02-november-2017--v855
	if idx := strings.IndexByte(fetchedVersion, ':'); idx > 0 {
		changelogURL = fetchedVersion[idx+1:]
		fetchedVersion = fetchedVersion[0:idx]
	}

	latestVersion, err := version.NewVersion(fetchedVersion)
	if err != nil {
		golog.Debugf("while fetching and parsing latest version from github: %v\n", err)
		return nil, ""
	}

	return latestVersion, changelogURL
}
