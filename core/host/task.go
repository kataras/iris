// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

// the 24hour name was "Supervisor" but it's not cover its usage
// 100%, best name is Task or Thead, I'll chouse Task.
// and re-name the host to "Supervisor" because that is the really
// supervisor.
import (
	"context"
	"github.com/kataras/iris/core/nettools"
	"net/http"
)

type (
	// FlowController exports the `DeferFlow`
	// and `RestoreFlow` capabilities.
	// Read more at Supervisor.
	FlowController interface {
		DeferFlow()
		RestoreFlow()
	}
)

// TaskHost contains all the necessary information
// about the host supervisor, its server
// and the exports the whole flow controller of it.
type TaskHost struct {
	su *Supervisor
	// Supervisor with access fields when server is running, i.e restrict access to "Schedule"
	// Server that running, is active and open
	// Flow controller
	FlowController
	// Various
	pid int

	doneChan chan struct{}
	errChan  chan error
}

// Done filled when server was shutdown.
func (h TaskHost) Done() <-chan struct{} {
	return h.doneChan
}

// Err filled when server received an error.
func (h TaskHost) Err() <-chan error {
	return h.errChan
}

// Serve can (re)run the server with the latest known configuration.
func (h TaskHost) Serve() error {
	// the underline server's serve, using the "latest known" listener from the supervisor.
	l, err := h.su.newListener()
	if err != nil {
		return err
	}

	// if http.serverclosed ignroe the error, it will have this error
	// from the previous close
	if err := h.su.server.Serve(l); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// HostURL returns the listening full url (scheme+host)
// based on the supervisor's server's address.
func (h TaskHost) HostURL() string {
	return nettools.ResolveURLFromServer(h.su.server)
}

// Hostname returns the underline server's hostname.
func (h TaskHost) Hostname() string {
	return nettools.ResolveHostname(h.su.server.Addr)
}

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// then the context's error is returned.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired.
func (h TaskHost) Shutdown(ctx context.Context) error {
	// the underline server's Shutdown (otherwise we will cancel all tasks and do cycles)
	return h.su.server.Shutdown(ctx)
}

// TaskProcess is the context of the Task runner.
// Contains the host's information and actions
// and its self cancelation emmiter.
type TaskProcess struct {
	canceledChan chan struct{}
	host         TaskHost
}

// Done filled when this task is canceled.
func (p TaskProcess) Done() <-chan struct{} {
	return p.canceledChan
}

// Host returns the TaskHost.
//
// TaskHost contains all the necessary information
// about the host supervisor, its server
// and the exports the whole flow controller of it.
func (p TaskProcess) Host() TaskHost {
	return p.host
}

func createTaskHost(su *Supervisor) TaskHost {
	host := TaskHost{
		su:             su,
		FlowController: su,
		doneChan:       make(chan struct{}),
		errChan:        make(chan error),
	}

	return host
}

func newTaskProcess(host TaskHost) TaskProcess {
	return TaskProcess{
		host:         host,
		canceledChan: make(chan struct{}),
	}
}

// A TaskRunner is an independent stream of instructions in a Supervisor.
// A routine is similar to a sequential program.
// However, a routine itself is not a program,
// it can't run on its own, instead it runs within a Supervisor's context.
//
// The real usage of a routine is not about a single sequential thread,
// but rather using multiple tasks in a single Supervisor.
// Multiple tasks running at the same time and performing various tasks is referred as Multithreading.
// A Task is considered to be a lightweight process because it runs within the context of a Supervisor
// and takes advantage of resources allocated for that Supervisor and its Server.
type TaskRunner interface {
	// Run runs the task based on its TaskProcess which contains
	// all the necessary information and actions to control the host supervisor
	// and its server.
	Run(TaskProcess)
}

// TaskRunnerFunc "converts" a func(TaskProcess) to a complete TaskRunner.
// Its functionality is exactly the same as TaskRunner.
//
// See `TaskRunner` too.
type TaskRunnerFunc func(TaskProcess)

// Run runs the task based on its TaskProcess which contains
// all the necessary information and actions to control the host supervisor
// and its server.
func (s TaskRunnerFunc) Run(proc TaskProcess) {
	s(proc)
}
