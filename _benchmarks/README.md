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
$ dotnet run -c Release
Hosting environment: Production
Content root path: C:\mygopath\src\github.com\kataras\iris\_benchmarks\netcore-mvc
Now listening on: http://localhost:5000
Application started. Press Ctrl+C to shut down.
```

```bash
$ bombardier -c 125 -n 5000000 http://localhost:5000/api/values/5
Bombarding http://localhost:5000/api/values/5 with 5000000 requests using 125 connections
 5000000 / 5000000 [=====================================================================================] 100.00% 2m3s
Done!
Statistics        Avg      Stdev        Max
  Reqs/sec     40226.03    8724.30     161919
  Latency        3.09ms     1.40ms   169.12ms
  HTTP codes:
    1xx - 0, 2xx - 5000000, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:     8.91MB/s
```

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

Click [here](screens) to navigate to the screenshots.

#### Summary

* Time to complete the `5000000 requests` - smaller is better.
* Reqs/sec - bigger is better.
* Latency - smaller is better
* Throughput - bigger is better.
* Memory usage - smaller is better.
* LOC (Lines Of Code) - smaller is better.

.NET Core MVC Application, written using 86 lines of code, ran for **2 minutes and 3 seconds** serving **40226.03** requests per second within **3.09ms** latency in average and **169.12ms** max, the memory usage of all these was ~123MB (without the dotnet host).

Iris MVC Application, written using 27 lines of code, ran for **47 seconds** serving **105643.71** requests per second within **1.18ms** latency in average and **22.01ms** max, the memory usage of all these was ~12MB.

Iris Application, written using 22 lines of code, ran for **45 seconds** serving **110809.98** requests per second within **1.13ms** latency in average and **18.02ms** max, the memory usage of all these was ~11MB.

#### Update: 20 August 2017

As [Josh Clark](https://twitter.com/clarkis117) and [Scott Hanselman‏](https://twitter.com/shanselman)‏ pointed out [on this status](https://twitter.com/shanselman/status/899005786826788865), on .NET Core `Startup.cs` file the line with `services.AddMvc();` can be replaced with `services.AddMvcCore();`. I followed their helpful instructions and re-run the benchmarks. The article now contains the latest benchmark output for the .NET Core application with the change both Josh and Scott noted.

The twitter conversion: https://twitter.com/MakisMaropoulos/status/899113215895982080

For those who want to compare with the standard services.AddMvc(); you can see the old output by pressing [here](screens/500m_requests_netcore-mvc.png).

**Thank you all** for the 100% green feedback, have fun!

- https://dev.to/kataras/go-vsnet-core-in-terms-of-http-performance
- https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8