# Static file server example

* Serves files from the given directory.
* Supports transparent response compression.
* Supports byte range responses.
* Generates directory index pages.
* Supports TLS (aka SSL or HTTPS).
* Supports virtual hosts.
* Exports various stats on /stats path.

# How to build

```
make
```

# How to run

```
./fileserver -h
./fileserver -addr=tcp.addr.to.listen:to -dir=/path/to/directory/to/serve
```

# fileserver vs nginx performance comparison

Serving default nginx path (`/usr/share/nginx/html` on ubuntu).

* nginx

```
$ ./wrk -t 4 -c 16 -d 10 http://localhost:80
Running 10s test @ http://localhost:80
  4 threads and 16 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   397.76us    1.08ms  20.23ms   95.19%
    Req/Sec    21.20k     2.49k   31.34k    79.65%
  850220 requests in 10.10s, 695.65MB read
Requests/sec:  84182.71
Transfer/sec:     68.88MB
```

* fileserver

```
$ ./wrk -t 4 -c 16 -d 10 http://localhost:8080
Running 10s test @ http://localhost:8080
  4 threads and 16 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   447.99us    1.59ms  27.20ms   94.79%
    Req/Sec    37.13k     3.99k   47.86k    76.00%
  1478457 requests in 10.02s, 1.03GB read
Requests/sec: 147597.06
Transfer/sec:    105.15MB
```

8 pipelined requests

* nginx

```
$ ./wrk -s pipeline.lua -t 4 -c 16 -d 10 http://localhost:80 -- 8
Running 10s test @ http://localhost:80
  4 threads and 16 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.34ms    2.15ms  30.91ms   92.16%
    Req/Sec    33.54k     7.36k  108.12k    76.81%
  1339908 requests in 10.10s, 1.07GB read
Requests/sec: 132705.81
Transfer/sec:    108.58MB
```

* fileserver

```
$ ./wrk -s pipeline.lua -t 4 -c 16 -d 10 http://localhost:8080 -- 8
Running 10s test @ http://localhost:8080
  4 threads and 16 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.08ms    6.33ms  88.26ms   92.83%
    Req/Sec   116.54k    14.66k  167.98k    69.00%
  4642226 requests in 10.03s, 3.23GB read
Requests/sec: 462769.41
Transfer/sec:    329.67MB
```
