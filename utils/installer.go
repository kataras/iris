package utils

import (
	"os"
	"runtime"
)

const (
	// ContentBINARY is the  string of "application/octet-stream response headers
	ContentBINARY = "application/octet-stream"
)

var (
	// AssetsDirectory the path which iris saves some assets came from the internet ( used in iris control plugin (to download the html,css,js) and for iris command line tool to download the packages)
	AssetsDirectory = ""
)

// init just sets the iris path for assets, used in iris control plugin and for iris command line tool(create command)
// the AssetsDirectory path should be like: C:/users/kataras/.iris (for windows) and for linux you can imagine
func init() {
	homepath := ""
	if runtime.GOOS == "windows" {
		homepath = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	} else {
		homepath = os.Getenv("HOME")
	}
	AssetsDirectory = homepath + PathSeparator + ".iris"
}
