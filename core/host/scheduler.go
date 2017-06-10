// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"sync/atomic"
)

type task struct {
	runner TaskRunner
	proc   TaskProcess

	// atomic-accessed, if != 0 means that is already
	// canceled before it ever ran, this happens to interrupt handlers too.
	alreadyCanceled int32
	Cancel          func()
}

func (t *task) isCanceled() bool {
	return atomic.LoadInt32(&t.alreadyCanceled) != 0
}

// Scheduler is a type of an event emmiter.
// Can register a specific task for a specific event
// when host is starting the server or host is interrupted by CTRL+C/CMD+C.
// It's being used internally on host supervisor.
type Scheduler struct {
	onServeTasks     []*task
	onInterruptTasks []*task
}

// TaskCancelFunc cancels a Task when called.
type TaskCancelFunc func()

// Schedule schedule/registers a Task,
// it will be executed/run to when host starts the server
// or when host is interrupted by CTRL+C/CMD+C  based on the TaskRunner type.
//
// See `OnInterrupt` and `ScheduleFunc` too.
func (s *Scheduler) Schedule(runner TaskRunner) TaskCancelFunc {

	t := new(task)
	t.runner = runner
	t.Cancel = func() {
		// it's not running yet, so if canceled now
		// set to already canceled to not run it at all.
		atomic.StoreInt32(&t.alreadyCanceled, 1)
	}

	if _, ok := runner.(OnInterrupt); ok {
		s.onInterruptTasks = append(s.onInterruptTasks, t)
	} else {
		s.onServeTasks = append(s.onServeTasks, t)
	}

	return func() {
		t.Cancel()
	}
}

// ScheduleFunc schedule/registers a task function,
// it will be executed/run to when host starts the server
// or when host is interrupted by CTRL+C/CMD+C  based on the TaskRunner type.
//
// See `OnInterrupt` and `ScheduleFunc` too.
func (s *Scheduler) ScheduleFunc(runner func(TaskProcess)) TaskCancelFunc {
	return s.Schedule(TaskRunnerFunc(runner))
}

func cancelTasks(tasks []*task) {
	for _, t := range tasks {
		if atomic.LoadInt32(&t.alreadyCanceled) != 0 {
			continue // canceled, don't run it
		}
		go t.Cancel()
	}
}

// CancelOnServeTasks cancels all tasks that are scheduled to run when
// host is starting the server, when the server is alive and online.
func (s *Scheduler) CancelOnServeTasks() {
	cancelTasks(s.onServeTasks)
}

// CancelOnInterruptTasks cancels all tasks that are scheduled to run when
// host is being interrupted by CTRL+C/CMD+C, when the server is alive and online as well.
func (s *Scheduler) CancelOnInterruptTasks() {
	cancelTasks(s.onInterruptTasks)
}

func runTaskNow(task *task, host TaskHost) {
	proc := newTaskProcess(host)
	task.proc = proc
	task.Cancel = func() {
		proc.canceledChan <- struct{}{}
	}

	go task.runner.Run(proc)
}

func runTasks(tasks []*task, host TaskHost) {
	for _, t := range tasks {
		if t.isCanceled() {
			continue
		}
		runTaskNow(t, host)
	}
}

func (s *Scheduler) runOnServe(host TaskHost) {
	runTasks(s.onServeTasks, host)
}

func (s *Scheduler) runOnInterrupt(host TaskHost) {
	runTasks(s.onInterruptTasks, host)
}

func (s *Scheduler) visit(visitor func(*task)) {
	for _, t := range s.onServeTasks {
		visitor(t)
	}

	for _, t := range s.onInterruptTasks {
		visitor(t)
	}
}

func (s *Scheduler) notifyShutdown() {
	s.visit(func(t *task) {
		go func() {
			t.proc.Host().doneChan <- struct{}{}
		}()
	})
}

func (s *Scheduler) notifyErr(err error) {
	s.visit(func(t *task) {
		go func() {
			t.proc.Host().errChan <- err
		}()
	})
}

// CopyTo copies all tasks from "s" to "to" Scheduler.
// It doesn't care about anything else.
func (s *Scheduler) CopyTo(to *Scheduler) {
	s.visit(func(t *task) {
		rnner := t.runner
		to.Schedule(rnner)
	})
}
