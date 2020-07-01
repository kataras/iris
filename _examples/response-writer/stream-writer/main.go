package main

import (
	"errors"
	"io"
	"time" // showcase the delay

	"github.com/kataras/iris/v12"
)

var errDone = errors.New("done")

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.ContentType("text/html")
		ctx.Header("Transfer-Encoding", "chunked")
		i := 0
		ints := []int{1, 2, 3, 5, 7, 9, 11, 13, 15, 17, 23, 29}
		// Send the response in chunks and wait for half a second between each chunk,
		// until connection close.
		err := ctx.StreamWriter(func(w io.Writer) error {
			ctx.Writef("Message number %d<br>", ints[i])
			time.Sleep(500 * time.Millisecond) // simulate delay.
			if i == len(ints)-1 {
				return errDone // ends the loop.
			}
			i++
			return nil // continue write
		})

		if err != errDone {
			// Test it by canceling the request before the stream ends:
			// [ERRO] $DATETIME stream: context canceled.
			ctx.Application().Logger().Errorf("stream: %v", err)
		}
	})

	type messageNumber struct {
		Number int `json:"number"`
	}

	app.Get("/json", func(ctx iris.Context) {
		ctx.Header("Transfer-Encoding", "chunked")
		i := 0
		ints := []int{1, 2, 3, 5, 7, 9, 11, 13, 15, 17, 23, 29}
		// Send the response in chunks and wait for half a second between each chunk,
		// until connection close.
		notifyClose := ctx.Request().Context().Done()
		for {
			select {
			case <-notifyClose:
				// err := ctx.Request().Context().Err()
				ctx.Application().Logger().Infof("Connection closed, loop end.")
				return
			default:
				ctx.JSON(messageNumber{Number: ints[i]})
				ctx.WriteString("\n")
				time.Sleep(500 * time.Millisecond) // simulate delay.
				if i == len(ints)-1 {
					ctx.Application().Logger().Infof("Loop end.")
					return
				}
				i++
				ctx.ResponseWriter().Flush()
			}
		}
	})

	app.Listen(":8080")
}

/*
Look the following methods too:
- Context.OnClose(callback)
- Context.OnConnectionClose(callback) and
- Context.Request().Context().Done()/.Err() too
*/
