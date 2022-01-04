package main

import (
	"context"
	"time"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/test", func(ctx iris.Context) {
		w := new(worker)
		result := w.Work(ctx)
		ctx.WriteString(result)
	})

	app.Listen(":8080", iris.WithTimeout(4*time.Second))
}

type worker struct{}

func (w *worker) Work(ctx context.Context) string {
	t := time.Tick(time.Second)
	times := 0
	for {
		select {
		case <-ctx.Done():
			println("context.Done: canceled")
			return "Work canceled"
		case <-t:
			times++
			println("Doing some work...")

			if times > 5 {
				return "Work is done with success"
			}
		}
	}

	return "nothing to do here"
}
