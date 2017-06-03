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

type Scheduler struct {
	onServeTasks     []*task
	onInterruptTasks []*task
}

type TaskCancelFunc func()

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

func (s *Scheduler) CancelOnServeTasks() {
	cancelTasks(s.onServeTasks)
}

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

func (s *Scheduler) CopyTo(to *Scheduler) {
	s.visit(func(t *task) {
		rnner := t.runner
		to.Schedule(rnner)
	})
}
