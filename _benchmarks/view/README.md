# View Engine Benchmarks

Benchmark between all 8 supported template parsers.

Amber, Ace and Pug parsers minifies the template before render. So, to have a fair benchmark, we must make sure that the byte amount of the total response body is exactly the same across all. Therefore, all other template files are minified too.

![Benchmarks Chart Graph](chart.png)

> Last updated: Oct 1, 2020 at 12:46pm (UTC)

## System

|    |    |
|----|:---|
| Processor | Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz |
| RAM | 15.85 GB |
| OS | Microsoft Windows 10 Pro |
| [Bombardier](https://github.com/codesenberg/bombardier) | v1.2.4 |
| [Go](https://golang.org) | go1.15.2 |

## Terminology

**Name** is the name of the template engine used under a particular test.

**Reqs/sec** is the avg number of total requests could be processed per second (the higher the better).

**Latency** is the amount of time it takes from when a request is made by the client to the time it takes for the response to get back to that client (the smaller the better).

**Throughput** is the rate of production or the rate at which data are transferred (the higher the better, it depends from response length (body + headers).

**Time To Complete** is the total time (in seconds) the test completed (the smaller the better).

## Results

### Test:Template Layout, Partial and Data

📖 Fires 1000000 requests with 125 concurrent clients. It receives HTML response. The server handler sets some template **data** and renders a template file which consists of a **layout** and a **partial** footer.

| Name | Language | Reqs/sec | Latency | Throughput | Time To Complete |
|------|:---------|:---------|:--------|:-----------|:-----------------|
| [Amber](./amber) | Go |125698 |0.99ms |44.67MB |7.96s |
| [Blocks](./blocks) | Go |123974 |1.01ms |43.99MB |8.07s |
| [Django](./django) | Go |118831 |1.05ms |42.17MB |8.41s |
| [Handlebars](./handlebars) | Go |101214 |1.23ms |35.91MB |9.88s |
| [Pug](./pug) | Go |89002 |1.40ms |31.81MB |11.24s |
| [Ace](./ace) | Go |64782 |1.93ms |22.98MB |15.44s |
| [HTML](./html) | Go |53918 |2.32ms |19.13MB |18.55s |
| [Jet](./jet) | Go |4829 |25.88ms |1.71MB |207.07s |

## How to Run

```sh
$ go get -u github.com/kataras/server-benchmarks
$ go get -u github.com/codesenberg/bombardier
$ server-benchmarks --wait-run=3s -o ./results
```
