# Benchmarks (internal)

Internal selected benchmarks between modified features across different versions.

* These benchmarks SHOULD run locally with a break of ~2 minutes between stress tests at the same machine, power plan and Turbo Boost set to ON
* The system's `GOPATH` environment variable SHOULD match the [vNext/go.mod replace directive](vNext/go.mod#L5) one
* A stress test may ran from a `name_test.go` file to measure _BUILD TIME_
    * each version executes: `go test -run=NONE --bench=. -count=5 --benchmem > name_test.txt`
    * the result will be presented through [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool
* _Or/and_ by firing [bombardier](https://github.com/codesenberg/bombardier/releases/tag/v1.2.4) _HTTP requests_ when it (the test) listens to an address
* Each benchmark SHOULD contain a brief explanation of what it does.

## Dependency Injection

Measures handler factory time.

```sh
$ cd v12.1.x
$ go test -run=NONE --bench=. -count=5 --benchmem > di_test.txt
$ cd ../vNext
$ go test -run=NONE --bench=. -count=5 --benchmem > di_test.txt
```

| Name    | Ops | Ns/op | B/op | Allocs/op |
|---------|:------|:--------|:--------|----|
| vNext   | 184512 | 6607  | 1544 | 17 |
| v12.1.x |  95974 | 12653 | 976  | 26 |

It accepts a dynamic path parameter and a JSON request. It returns a JSON response. Fires 500000 requests with 125 concurrent connections.

```sh
# di.go
$ bombardier -c 125 -n 500000 --method="POST" --body-file=./request.json http://localhost:5000/42
```

| Name    | Throughput | Reqs/sec | Latency | Time To Complete |
|---------|:-----------|:----------|:---------|----------------|
| vNext   |  46.51MB/s | 160480.74 | 777.33us | 3s |
| v12.1.x |  32.43MB/s | 108839.19 | 1.14ms   | 4s |
