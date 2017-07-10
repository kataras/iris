// white-box testing
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
	su.Shutdown(context.TODO())
	time.Sleep(1 * time.Second)

	// Output:
	// http: Server closed
	// http: Server closed
	// http: Server closed
}

type myTestTask struct {
	restartEvery time.Duration
	maxRestarts  int
	logger       *log.Logger
}

func (m myTestTask) OnServe(host TaskHost) {
	host.Supervisor.DeferFlow() // don't exit on underline server's Shutdown.

	ticker := time.NewTicker(m.restartEvery)
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
				exitAfterXRestarts := m.maxRestarts
				if rans == exitAfterXRestarts {
					m.logger.Println("exit")
					ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
					defer cancel()
					host.Supervisor.Shutdown(ctx) // total shutdown
					host.Supervisor.RestoreFlow() // free to exit (if shutdown)
					return
				}

				rans++

				m.logger.Println(fmt.Sprintf("closed %d times", rans))
				host.Shutdown(context.TODO())

				startDelay := 2 * time.Second
				time.AfterFunc(startDelay, func() {
					m.logger.Println("restart")
					host.Serve() // restart

				})

			}
		}
	}
}

func ExampleSupervisor_RegisterOnServe() {
	h := New(&http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}),
	})

	logger := log.New(os.Stdout, "Supervisor: ", 0)

	mytask := myTestTask{
		restartEvery: 6 * time.Second,
		maxRestarts:  2,
		logger:       logger,
	}

	h.RegisterOnServe(mytask.OnServe)

	ln, err := net.Listen("tcp4", ":9394")
	if err != nil {
		panic(err.Error())
	}

	logger.Println("server started...")
	h.Serve(ln)

	// Output:
	// Supervisor: server started...
	// Supervisor: closed 1 times
	// Supervisor: restart
	// Supervisor: closed 2 times
	// Supervisor: restart
	// Supervisor: exit
}
