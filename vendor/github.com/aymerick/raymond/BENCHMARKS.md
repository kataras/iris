# Benchmarks

Hardware: MacBookPro11,1 - Intel Core i5 - 2,6 GHz - 8 Go RAM

With:

    - handlebars.js #8cba84df119c317fcebc49fb285518542ca9c2d0
    - raymond #7bbaaf50ed03c96b56687d7fa6c6e04e02375a98


## handlebars.js (ops/ms)

        arguments          198 ±4 (5)
        array-each         568 ±23 (5)
        array-mustache     522 ±18 (4)
        complex             71 ±7 (3)
        data                67 ±2 (3)
        depth-1             47 ±2 (3)
        depth-2             14 ±1 (2)
        object-mustache   1099 ±47 (5)
        object             907 ±58 (4)
        partial-recursion   46 ±3 (4)
        partial             68 ±3 (3)
        paths             1650 ±50 (3)
        string            2552 ±157 (3)
        subexpression      141 ±2 (4)
        variables         2671 ±83 (4)


## raymond

        BenchmarkArguments          200000     6642 ns/op   151 ops/ms
        BenchmarkArrayEach          100000    19584 ns/op    51 ops/ms
        BenchmarkArrayMustache      100000    17305 ns/op    58 ops/ms
        BenchmarkComplex            30000     50270 ns/op    20 ops/ms
        BenchmarkData               50000     25551 ns/op    39 ops/ms
        BenchmarkDepth1             100000    20162 ns/op    50 ops/ms
        BenchmarkDepth2             30000     47782 ns/op    21 ops/ms
        BenchmarkObjectMustache     200000     7668 ns/op   130 ops/ms
        BenchmarkObject             200000     8843 ns/op   113 ops/ms
        BenchmarkPartialRecursion   50000     23139 ns/op    43 ops/ms
        BenchmarkPartial            50000     31015 ns/op    32 ops/ms
        BenchmarkPath               200000     8997 ns/op   111 ops/ms
        BenchmarkString             1000000    1879 ns/op   532 ops/ms
        BenchmarkSubExpression      300000     4935 ns/op   203 ops/ms
        BenchmarkVariables          200000     6478 ns/op   154 ops/ms
