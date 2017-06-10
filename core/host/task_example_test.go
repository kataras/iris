// white-box testing
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
