// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type myTestTask struct {
	delay  time.Duration
	logger *log.Logger
}

func (m myTestTask) Run(proc TaskProcess) {
	ticker := time.NewTicker(m.delay)
	defer ticker.Stop()
	rans := 0
	for {
		select {
		case _, ok := <-ticker.C:
			{
				if !ok {
					m.logger.Println("ticker issue, closed channel, exiting from this task...")
					return
				}
				rans++
				m.logger.Println(fmt.Sprintf("%d", rans))
			}
		case <-proc.Done():
			{
				m.logger.Println("canceled, exiting from task AND SHUTDOWN the server...")
				proc.Host().Shutdown(context.TODO())
				return
			}
		}
	}
}

func SchedulerSchedule() {
	h := New(&http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}),
	})
	logger := log.New(os.Stdout, "Supervisor: ", 0)

	delaySeconds := 2

	mytask := myTestTask{
		delay:  time.Duration(delaySeconds) * time.Second,
		logger: logger,
	}

	cancel := h.Schedule(mytask)
	ln, err := net.Listen("tcp4", ":9090")
	if err != nil {
		panic(err.Error())
	}

	logger.Println("server started...")
	logger.Println("we will cancel the task after 2 runs (the third will be canceled)")
	cancelAfterRuns := 2
	time.AfterFunc(time.Duration(delaySeconds*cancelAfterRuns+(delaySeconds/2))*time.Second, func() {
		cancel()
		logger.Println("cancel sent")
	})
	h.Serve(ln)

	// Output:
	// Supervisor: server started...
	// Supervisor: we will cancel the task after 2 runs (the third will be canceled)
	// Supervisor: 1
	// Supervisor: 2
	// Supervisor: cancel sent
	// Supervisor: canceled, exiting from task AND SHUTDOWN the server...
}
