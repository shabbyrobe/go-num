num: 128-bit signed and unsigned integers for Go
================================================

Fastish `int128` (`num.I128`) and `uint128` (`num.U128`) types for Go, providing the
majority of methods found in `big.Int`.

`I128` is a signed "two's complement" implementation that should behave the
same way on overflow as `int64`.

`U128` and `I128` are immutable value types by default; operations always return a
new value rather than mutating the existing one.

Simple usage:

    a := num.U128From64(1234)
    b := a.Add(num.U128From64(5678))
    fmt.Printf("%x", x)

The whole library is aggressively fuzzed (see `fuzz_test.go`). Configure the fuzzer
by playing with the following flags to `go test`:

    -num.fuzziter int
        Number of iterations to fuzz each op (default 10000)
    -num.fuzzop value
        Fuzz op to run (can pass multiple times, or a comma separated list)
    -num.fuzzseed int
        Seed the RNG (0 == current nanotime)
    -num.fuzztype value
        Fuzz type (u128, i128) (can pass multiple)

The fuzzer can do 10,000 iterations of all ops and all types per second. Most of this
time is spent dealing with `big.Int`, which is used as a reference.


Silly benchmarks game
---------------------

Here are some hopelessly artificial comparsions between U128, uint64 and big.Int.
I128 is typically a bit slower than U128 but both are quite adequate for common
arithmetic operations.

Please help yourself to as many grains of salt as you like from this enormous vat.

    BenchmarkU128Mul-8          300000000     4.99 ns/op      0 B/op    0 allocs/op
    BenchmarkU128Add-8         2000000000     0.53 ns/op      0 B/op    0 allocs/op
    BenchmarkU128QuoRem-8       100000000     13.4 ns/op      0 B/op    0 allocs/op
    BenchmarkU128CmpEqual-8    2000000000     0.55 ns/op      0 B/op    0 allocs/op

    BenchmarkBigIntMul-8        100000000     13.9 ns/op      0 B/op    0 allocs/op
    BenchmarkBigIntAdd-8         20000000     97.8 ns/op     48 B/op    1 allocs/op
    BenchmarkBigIntDiv-8         20000000      262 ns/op     96 B/op    2 allocs/op
    BenchmarkBigIntCmpEqual-8   200000000     8.71 ns/op      0 B/op    0 allocs/op

    BenchmarkUint64Mul-8         2000000000   0.59 ns/op
    BenchmarkUint64Add-8         2000000000   0.53 ns/op
    BenchmarkUint64Div-8         200000000    8.32 ns/op
    BenchmarkUint64Equal-8       2000000000   0.58 ns/op

Some operations are still horrifically slow. U128 and I128 fall back to big.Int
where faster implementations are not yet available. This will hopefully change
over time:

    BenchmarkU128AsBigInt/0,fedcba98-8                 5000000    238 ns/op
    BenchmarkU128String/fedcba9876543210-8            10000000    175 ns/op
    BenchmarkU128String/fedcba9876543210fedcba98-8     1000000   1002 ns/op

At least in the case of `AsBigInt`, you can use `IntoBigInt` which allows you
to recycle memory and is significantly faster:

    BenchmarkU128IntoBigInt/0,fedcba98-8             100000000   13.2 ns/op

