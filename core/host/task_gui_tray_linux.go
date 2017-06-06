// +build linux

// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"os"
	"time"
)

// ShowTrayTask is a supervisor's built'n task which shows
// the iris tray icon to the taskbar (cross-platform).
//
// It's responsible for the server's status button.
func ShowTrayTask(version string, shutdownTimeout time.Duration) TaskRunnerFunc {
	return func(proc TaskProcess) {
		os.Stdout.WriteString("Tray icon is not enabled by-default for linux systems,\nyou have to install a dependency first and re-get the Iris pgk:\n")
		os.Stdout.WriteString("$ sudo apt-get install libgtk-3-dev libappindicator3-dev\n")
		os.Stdout.WriteString("$ go get -u github.com/kataras/iris\n")
		// manually:
		// os.Stdout.WriteString("remove $GOPATH/src/github.com/kataras/iris/core/host/task_gui_tray_linux.go\n")
		// os.Stdout.WriteString("edit $GOPATH/src/github.com/kataras/iris/core/host/task_gui_tray.go and remove the // +build !linux\n")
		// os.Stdout.WriteString("edit $GOPATH/src/github.com/kataras/iris/core/gui/tray.go and remove the // +build !linux\n")
	}
}
