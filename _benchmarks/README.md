## Hardware

* Processor: Intel(R) Core(TM) **i7-4710HQ** CPU @ 2.50GHz 2.50GHz
* RAM: **8.00 GB**

## Software

* OS: Microsoft **Windows** [Version **10**.0.15063]
* HTTP Benchmark Tool: https://github.com/codesenberg/bombardier, latest version **1.1**
* **.NET Core**: https://www.microsoft.com/net/core, latest version **2.0**
* **Iris**: https://github.com/kataras/iris, latest version **8.3** built with [go1.8.3](https://golang.org)

## Results

### .NET Core MVC
```bash
$ cd netcore-mvc
$ dotnet run
Hosting environment: Production
Content root path: C:\mygopath\src\github.com\kataras\iris\_benchmarks\netcore-mvc
Now listening on: http://localhost:5000
Application started. Press Ctrl+C to shut down.
```

```bash
$ bombardier -c 125 -n 5000000 http://localhost:5000/api/values/5
Bombarding http://localhost:5000/api/values/5 with 5000000 requests using 125 connections
 5000000 / 5000000 [====================================================================================] 100.00% 2m39s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec     31532.63   10360.09     259792
  Latency        3.99ms     2.32ms   297.21ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:     6.89MB/s
```
> 127210K Memory (private working set)

### Iris MVC
```bash
$ cd iris-mvc
$ go run main.go
Now listening on: http://localhost:5000
Application started. Press CTRL+C to shut down.
```

```bash
$ bombardier -c 125 -n 5000000 http://localhost:5000/api/values/5
Bombarding http://localhost:5000/api/values/5 with 5000000 requests using 125 connections
 5000000 / 5000000 [======================================================================================] 100.00% 58s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec     86087.86    3432.38      93833
  Latency        1.45ms   464.12us    42.53ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:    16.01MB/s
```
> 11816K Memory (private working set)

### Iris
```bash
$ cd iris
$ go run main.go
Now listening on: http://localhost:5000
Application started. Press CTRL+C to shut down.
```

```bash
$ bombardier -c 125 -n 5000000 http://localhost:5000/api/values/5
Bombarding http://localhost:5000/api/values/5 with 5000000 requests using 125 connections
 5000000 / 5000000 [======================================================================================] 100.00% 48s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec    102987.54    8333.43     120069
  Latency        1.21ms   369.71us    23.52ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:    19.15MB/s
```
> 11040K Memory (private working set)

Click [here](screens) to navigate to the screenshots.

#### Summary

* Time to complete the `5000000 requests` - smaller is better.
* Reqs/sec - bigger is better.
* Latency - smaller is better
* Throughput - bigger is better.
* Memory usage - smaller is better.
* LOC (Lines Of Code) - smaller is better.

.NET Core MVC Application, written using 86 lines of code, ran for **2 minutes and 39 seconds** serving **31532.63** requests per second and **3.99ms** latency in average and **297.21ms** max, the memory usage of all these was 127MB (without the dotnet host).

Iris MVC Application, written using 27 lines of code, ran for **58 seconds** serving **86087.86** requests per second and **1.45ms** latency in average and **42.53ms** max, the memory usage of all these was 12MB.

Iris Application, written using 22 lines of code, ran for **48 seconds** serving **102987.54** requests per second and **1.21ms** latency in average and **23.52ms** max, the memory usage of all these was 11MB.