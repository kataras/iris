## Benchmark

Run the benchmark by yourself and check which is the faster framework out there.

1. Clone/Download the [https://github.com/kataras/go-http-routing-benchmark](https://github.com/kataras/go-http-routing-benchmark)
2. Open a terminal window inside the directory you saved it
3. Run go test -bench=. -timeout=60m



>Note that the kataras/go-http-routing-benchmark is clone of [https://github.com/julienschmidt/go-http-routing-benchmark](https://github.com/julienschmidt/go-http-routing-benchmark) which is the standar way to benchmark frameworks for go.

-------------------------------------------

**-bench=.** means run and print all tests 


**-timeout=60m** means override the default timeout which is 10m, give the tests their time, on my pc it finished at 971.163s



## Console Output 

At this folder (iris/benchmark)  you can see that we have 14 .png files, these are the images from my output when I ran  full benchmark (22 March 2016), check them if you can't run your benchmark by yourself. Note that on Output log: the last line is the next's first line. The first 2 images is just the loading of the frameworks, you can skip them.

### Explanation of the letters you will see


1. Framework Name + test name
	- The first left part is the name of the framework/router + the test name
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

---------------------------------------

> Note that on Output log: the last line is the next's first line. You can skip the first 2 images, they show the loading of the frameworks, . 

---------------------------------------
![1](https://raw.githubusercontent.com/kataras/iris/development/benchmark/1.png)

![2](https://raw.githubusercontent.com/kataras/iris/development/benchmark/2.png)

![3](https://raw.githubusercontent.com/kataras/iris/development/benchmark/3.png)

![4](https://raw.githubusercontent.com/kataras/iris/development/benchmark/4.png)

![5](https://raw.githubusercontent.com/kataras/iris/development/benchmark/5.png)

![6](https://raw.githubusercontent.com/kataras/iris/development/benchmark/6.png)

![7](https://raw.githubusercontent.com/kataras/iris/development/benchmark/7.png)

![8](https://raw.githubusercontent.com/kataras/iris/development/benchmark/8.png)

![9](https://raw.githubusercontent.com/kataras/iris/development/benchmark/9.png)

![10](https://raw.githubusercontent.com/kataras/iris/development/benchmark/10.png)

![11](https://raw.githubusercontent.com/kataras/iris/development/benchmark/11.png)

![12](https://raw.githubusercontent.com/kataras/iris/development/benchmark/12.png)

![13](https://raw.githubusercontent.com/kataras/iris/development/benchmark/13.png)

![14](https://raw.githubusercontent.com/kataras/iris/development/benchmark/14.png)


