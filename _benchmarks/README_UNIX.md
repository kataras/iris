![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new.png)]

## Hardware

* [Processor](screens/unix/system_info_cpu.png): Intel(R) Core(TM) **i7-4710HQ** CPU @ 2.50GHz
* [RAM](screens/unix/system_info_ram.png): **8.00 GB**

## Software

* OS: Linux **Ubuntu** [Version **17.10**] with latest kernel version **4.14.0-041400-generic x86_64 GNU/Linux**
* HTTP Benchmark Tool: https://github.com/codesenberg/bombardier, latest version **1.1**
* **Iris [Go]**: https://github.com/kataras/iris, latest version **8.5.7** built with [go1.9.2](https://golang.org)
* **.NET Core [C#]**: https://www.microsoft.com/net/core, latest version **2.0.2**
* **Node.js (express + throng) [Javascript]**: https://nodejs.org/, latest version **9.2.0**, express: https://github.com/expressjs/express latest version **4.16.0** and [throng](https://www.npmjs.com/package/throng) latest version **4.0.0**

Go ahead to the [README.md](README.md) and read how you can reproduce the benchmarks. Don't be scary it's actually very easy, you can do these things as well!

## Results

* Throughput - `bigger is better`.
* Reqs/sec (Requests Per Second in Average) - `bigger is better`.
* Latency - `smaller is better`.
* Time To Complete - `smaller is better`.
* Total Requests in this fortune are all 1 million, in order to be easier to do the graph later on.

### Native

| Name | Throughput | Reqs/sec | Latency | Time To Complete | Total Requests |
|-------|:-----------|:--------|:-------------|---------|------|
| Iris | **29.31MB/s** | 157628 | 791.58us | 6s | 1000000 |
| Kestrel | **25.28MB/s** | 139642 | 0.89ms | 7s | 1000000 |
| Node.js | **13.69MB/s** | 50907 | 2.45ms | 19s | 1000000 |
| Iris with Sessions | **22.37MB/s** | 71922 | 1.74ms | 14s | 1000000 |
| Kestrel with Sessions | **14.51MB/s** | 31102 | 4.02ms | 32s | 1000000 |
| Node.js with Sessions | **5.08MB/s** | 19358 | 6.48ms | 51s | 1000000 |

> each test has its own screenshot, click [here](screens/unix) to explore

### MVC (Model View Controller)

| Name | Throughput | Reqs/sec | Latency | Time To Complete | Total Requests |
|-------|:-----------|:--------|:-------------|---------|------|
| Iris MVC | **26.39MB/s** | 141868 | 0.88ms | 7s | 1000000 |
| .Net Core MVC | **11.99MB/s** | 54418 | 2.30ms | 18s | 1000000 |
| - | - | - | - | - | - |
| Iris MVC with Templates | **136.58MB/s** | 18933 | 6.60ms | 52s | 1000000 |
| .Net Core MVC with Templates | **88.95MB/s** | 12347 | 10.12ms | 1m21s | 1000000 |
| - | - | - | - | - | - |

> nodejs express does not contain any MVC features

### Updates

- 21 November 2017: initial run and publish

## Articles (ms windows OS)

- https://hackernoon.com/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8
- https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5

**Thank you all** for the 100% green feedback, have fun!