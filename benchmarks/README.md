## Benchmarks

Run the benchmarks by yourself and check which is the faster framework out there.

1. Clone/Download the [https://github.com/kataras/go-http-routing-benchmark](https://github.com/kataras/go-http-routing-benchmark)
2. Open a terminal window inside the directory you saved it
3. Run go test -bench=. -timeout=60m



>Note that the kataras/go-http-routing-benchmark is clone of [https://github.com/julienschmidt/go-http-routing-benchmark](https://github.com/julienschmidt/go-http-routing-benchmark) which is the standar way to benchmark frameworks for go.

-------------------------------------------

**-bench=.** means run and print all tests 


**-timeout=60m** means override the default timeout which is 10m, give the tests their time, on my pc it finished at 971.163s



## Console Output 

At this folder (iris/benchmarks)  you can see that we have 14 .png files, these are the images from my output when I ran  all the benchmarks (22 March 2016), check them if you can't run your benchmarks by yourself. Note that on Output log: the last line is the next's first line. The first 2 images is just the loading of the frameworks, you can skip them.

### Explanation of the letters you will see


1. Name
	- The first left part is the name of the framework/router
2. Total Operations( Executions )
	- The second left part is how many Total Operations gets on x time ( Bigger the best )
3. ns/op 
	- The third part is how many nanoseconds per operation ( Lower the best )
4. B/op
	- The forth part is how much Bytes per operation used ( Lower the best )
5. allocs/op
	- The final part is how many memory allocations per operations are done ( Lower the best )




------------------------------------------
My hardware:


Intel(R) Core(TM) i7-4710HQ CPU @ 2.50Ghz 2.50Ghz


8.00 GB RAM

64-bit Operating System