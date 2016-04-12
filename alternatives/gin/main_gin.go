package main

import (
	"github.com/gin-gonic/gin"
	_"time"
)

func main() {
	gin.SetMode("release") // turn off console debug messages
	r := gin.New()
	
	r.GET("/rest/hello", func(c *gin.Context) {
	    //time.Sleep(time.Duration(500) * time.Millisecond)
		c.Writer.Write([]byte("Hello world"))
	})
	r.Run() // listen and server on 0.0.0.0:8080
}