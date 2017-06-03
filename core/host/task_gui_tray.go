// +build !linux

// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"context"
	"time"

	"github.com/kataras/iris/core/gui"
)

// ShowTrayTask is a supervisor's built'n task which shows
// the iris tray icon to the taskbar (cross-platform).
//
// It's responsible for the server's status button.
func ShowTrayTask(version string, shutdownTimeout time.Duration) TaskRunnerFunc {
	return func(proc TaskProcess) {
		t := gui.Tray
		// set the label "Version" to the framework's current Version.
		t.SetVersion(version)

		//  active the status button(online/offline).
		t.OnServerStatusChange(
			// set the first callback (pressed when unchecked).
			func() {
				go proc.Host().Serve()
			},
			// set the second call back (pressed when checked, default status with its label setted to :"Offline".
			func() {
				// when server is shutting down it will send an "http closed" error ,
				// that DeferFlow stops from returning that error and exiting the app
				// postpone the execution flow, the interrupt signal will restore the flow
				// when ctrl/cmd+C pressed.
				proc.Host().DeferFlow()
				ctx, cancel := context.WithTimeout(context.TODO(), shutdownTimeout)
				defer cancel()
				proc.Host().Shutdown(ctx)
			})

		// render the tray icon and block this scheduled task(goroutine.
		t.Show()
	}
}
