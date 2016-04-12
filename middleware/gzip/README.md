## Gzip

From the version 1.1+ the gzip middleware is unuseful and deleted, you can write using gzip compression using:

```go

//...

iris.Get("/something", func(ctx *iris.Context){
	ctx.Response.WriteGzip(...) // it takes  *buffio.Writer as parameter and returns an error 
})


//...


```