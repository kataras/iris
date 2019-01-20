package main

import (
	"fmt" // just an optional helper
	"io"
	"time" // showcase the delay

	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.ContentType("text/html")
		ctx.Header("Transfer-Encoding", "chunked")
		i := 0
		ints := []int{1, 2, 3, 5, 7, 9, 11, 13, 15, 17, 23, 29}
		// Send the response in chunks and wait for half a second between each chunk.
		ctx.StreamWriter(func(w io.Writer) bool {
			fmt.Fprintf(w, "Message number %d<br>", ints[i])
			time.Sleep(500 * time.Millisecond) // simulate delay.
			if i == len(ints)-1 {
				return false // close and flush
			}
			i++
			return true // continue write
		})
	})

	type messageNumber struct {
		Number int `json:"number"`
	}

	app.Get("/alternative", func(ctx iris.Context) {
		ctx.ContentType("application/json")
		ctx.Header("Transfer-Encoding", "chunked")
		i := 0
		ints := []int{1, 2, 3, 5, 7, 9, 11, 13, 15, 17, 23, 29}
		// Send the response in chunks and wait for half a second between each chunk.
		for {
			ctx.JSON(messageNumber{Number: ints[i]})
			ctx.WriteString("\n")
			time.Sleep(500 * time.Millisecond) // simulate delay.
			if i == len(ints)-1 {
				break
			}
			i++
			ctx.ResponseWriter().Flush()
		}
	})

	app.Run(iris.Addr(":8080"))
}
