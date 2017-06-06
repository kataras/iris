// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"context"
	"time"
)

// OnInterrupt is a built'n supervisor task type which fires its
// value(Task) when an OS interrupt/kill signal received.
type OnInterrupt TaskRunnerFunc

func (t OnInterrupt) Run(proc TaskProcess) {
	t(proc)
}

// ShutdownOnInterruptTask returns a supervisor's built'n task which
// shutdowns the server when InterruptSignalTask fire this task.
func ShutdownOnInterruptTask(shutdownTimeout time.Duration) TaskRunner {
	return OnInterrupt(func(proc TaskProcess) {
		ctx, cancel := context.WithTimeout(context.TODO(), shutdownTimeout)
		defer cancel()
		proc.Host().Shutdown(ctx)
		proc.Host().RestoreFlow()
	})
}
