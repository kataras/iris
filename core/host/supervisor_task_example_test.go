// white-box testing
package host

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func ExampleSupervisor_RegisterOnError() {
	su := New(&http.Server{Addr: ":8273", Handler: http.DefaultServeMux})

	su.RegisterOnError(func(err error) {
		fmt.Println(err.Error())
	})

	su.RegisterOnError(func(err error) {
		fmt.Println(err.Error())
	})

	su.RegisterOnError(func(err error) {
		fmt.Println(err.Error())
	})

	go su.ListenAndServe()
	time.Sleep(1 * time.Second)
	if err := su.Shutdown(context.TODO()); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)

	// Output:
	// http: Server closed
	// http: Server closed
	// http: Server closed
}
