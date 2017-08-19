## Hardware

* Processor: Intel(R) Core(TM) **i7-4710HQ** CPU @ 2.50GHz 2.50GHz
* RAM: **8.00 GB**

## Software

* OS: Microsoft **Windows** [Version **10**.0.15063], power plan is "High performance"
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
 5000000 / 5000000 [=====================================================================================] 100.00% 2m8s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec     39311.56   11744.49     264000
  Latency        3.19ms     1.61ms   229.73ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:     8.61MB/s
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
 5000000 / 5000000 [======================================================================================] 100.00% 47s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec    105643.81    7687.79     122564
  Latency        1.18ms   366.55us    22.01ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:    19.65MB/s
```
> 126024K Memory (private working set)

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
 5000000 / 5000000 [======================================================================================] 100.00% 45s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec    110809.98    8209.87     128212
  Latency        1.13ms   307.86us    18.02ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:    20.61MB/s
```
> 11052K Memory (private working set)

Click [here](screens) to navigate to the screenshots.

#### Summary

* Time to complete the `5000000 requests` - smaller is better.
* Reqs/sec - bigger is better.
* Latency - smaller is better
* Throughput - bigger is better.
* Memory usage - smaller is better.
* LOC (Lines Of Code) - smaller is better.

.NET Core MVC Application, written using 86 lines of code, ran for **2 minutes and 8 seconds** serving **39311.56** requests per second within **3.19ms** latency in average and **229.73ms** max, the memory usage of all these was 126MB (without the dotnet host).

Iris MVC Application, written using 27 lines of code, ran for **47 seconds** serving **105643.71** requests per second within **1.18ms** latency in average and **22.01ms** max, the memory usage of all these was 12MB.

Iris Application, written using 22 lines of code, ran for **45 seconds** serving **110809.98** requests per second within **1.13ms** latency in average and **18.02ms** max, the memory usage of all these was 11MB.