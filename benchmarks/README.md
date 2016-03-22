## Benchmarks

Run the benchmarks by yourself and check which is the faster framework out there.

1. Clone/Download the [https://github.com/kataras/go-http-routing-benchmark](https://github.com/kataras/go-http-routing-benchmark)
2. Open a terminal window inside the directory you saved it
3. Run go test -bench=. -timeout=60m


-------------------------------------------

**-bench=.** means run and print all tests 


**-timeout=60m** means override the default timeout which is 10m, give the tests their time, on my pc it finished at 971.163s



## Console Output 

At this folder (iris/benchmarks)  you can see that we have 14 .png files, these are the images from my output when I ran  all the benchmarks (22 March 2016), check them if you can't run your benchmarks by yourself. Note that on Output log: the last line is the next's first line

------------------------------------------
My hardware:


Intel(R) Core(TM) i7-4710HQ CPU @ 2.50Ghz 2.50Ghz


8.00 GB RAM

64-bit Operating System