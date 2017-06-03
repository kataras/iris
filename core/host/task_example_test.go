// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package host

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func TaskHostError() {
	su := New(&http.Server{Addr: ":8273", Handler: http.DefaultServeMux})

	su.ScheduleFunc(func(proc TaskProcess) {
		select {
		case err := <-proc.Host().Err():
			fmt.Println(err.Error())
		}
	})

	su.ScheduleFunc(func(proc TaskProcess) {
		select {
		case err := <-proc.Host().Err():
			fmt.Println(err.Error())
		}
	})

	su.ScheduleFunc(func(proc TaskProcess) {
		select {
		case err := <-proc.Host().Err():
			fmt.Println(err.Error())
		}
	})

	go su.ListenAndServe()
	time.Sleep(1 * time.Second)
	su.Shutdown(context.TODO())
	time.Sleep(1 * time.Second)

	// Output:
	// http: Server closed
	// http: Server closed
	// http: Server closed
}
