// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"fmt"
	"io"
	"runtime"
)

// WriteBannerTask is a task which accepts a logger(io.Writer)
// and a "banner" text to write to following
// by a generated message based on the host supervisor's server and writes it to the "w".
// This task runs on serve.
func WriteBannerTask(w io.Writer, banner string) TaskRunnerFunc {
	return func(proc TaskProcess) {
		listeningURI := proc.Host().HostURL()
		interruptkey := "CTRL"
		if runtime.GOOS == "darwin" {
			interruptkey = "CMD"
		}
		w.Write([]byte(fmt.Sprintf("%s\n\nNow listening on: %s\nApplication started. Press %s+C to shut down.\n",
			banner, listeningURI, interruptkey)))
	}
}
