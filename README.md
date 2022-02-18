num: 128-bit signed and unsigned integers for Go
================================================

[![GoDoc](https://godoc.org/github.com/shabbyrobe/go-num?status.svg)](https://godoc.org/github.com/shabbyrobe/go-num)
![Go](https://github.com/shabbyrobe/go-num/workflows/Go/badge.svg)

Fastish `int128` (`num.I128`) and `uint128` (`num.U128`) 128-bit integer types
for Go, providing the majority of methods found in `big.Int`.

**WARNING**: Function execution times in this library _almost always_ depend on the
inputs. This library is inappropriate for use in any domain where it is important
that the execution time does not reveal details about the inputs used.

`I128` is a signed "two's complement" implementation that should behave the
same way on overflow as `int64`.

`U128` and `I128` are immutable value types by default; operations always return a
new value rather than mutating the existing one.

Simple usage:

```go
a := num.U128From64(1234)
b := num.U128From64(5678)
b := a.Add(a)
fmt.Printf("%x", x)
```

Most operations that operate on 2 128-bit numbers have a variant that accepts
a 64-bit number:

```go
a := num.U128From64(1234)
b := a.Add64(5678)
fmt.Printf("%x", x)
```

Performance on x86-64/amd64 architectures is the focus. Performance
improvements for other architectures will only be made if they are done without
affecting the performance of amd64 processors. Code readability is less of a
concern than raw performance, but where direct readability is sacrificed it
should be exchanged for comments. Anything insufficiently explained is a bug.


## Testing

**DISCLAIMER**: I have put a significant amount of effort into testing this
library and the coverage is very broad and very deep. I have not found much in
the way of bugs in a while, but without enough universes worth of time to test
exhaustively, there is no guarantee I haven't missed some insane edge case
(like the last one, which was tripped by this absolutely _bonkers_ behaviour in
Go itself: https://github.com/golang/go/issues/29463).

Please be very careful if you choose to use this for production workloads, and
take note of the clause regarding warranty in the LICENSE file.

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

The fuzzer can do 10,000 iterations of all ops and all types per second. Most
of this time is spent dealing with `big.Int`, which is used as a reference.
Unless there is a flaw in the fuzzer itself, I believe it has picked all the
low hanging fruit, and quite a bit of high-hanging fruit too.


## Silly benchmarks game

Here are some hopelessly artificial comparsions between U128, uint64 and big.Int.
I128 is typically a bit slower than U128 but both are quite adequate for common
arithmetic operations.

Please help yourself to as many grains of salt as you like from this enormous vat.

Using U128:

    BenchmarkU128Mul-8          300000000     4.99 ns/op      0 B/op    0 allocs/op
    BenchmarkU128Add-8         2000000000     0.53 ns/op      0 B/op    0 allocs/op
    BenchmarkU128QuoRem-8       100000000     13.4 ns/op      0 B/op    0 allocs/op
    BenchmarkU128CmpEqual-8    2000000000     0.55 ns/op      0 B/op    0 allocs/op

Same tests using uint64:

    BenchmarkUint64Mul-8         2000000000   0.59 ns/op
    BenchmarkUint64Add-8         2000000000   0.53 ns/op
    BenchmarkUint64Div-8         200000000    8.32 ns/op
    BenchmarkUint64Equal-8       2000000000   0.58 ns/op

Same tests using big.Int:

    BenchmarkBigIntMul-8        100000000     13.9 ns/op      0 B/op    0 allocs/op
    BenchmarkBigIntAdd-8         20000000     97.8 ns/op     48 B/op    1 allocs/op
    BenchmarkBigIntDiv-8         20000000      262 ns/op     96 B/op    2 allocs/op
    BenchmarkBigIntCmpEqual-8   200000000     8.71 ns/op      0 B/op    0 allocs/op

Some conversion operations (mostly string-related) are still quite slow. U128 and I128
fall back to big.Int for conversion where faster implementations are not yet available.
This will hopefully change over time:

    BenchmarkU128String/fedcba9876543210-8            10000000    175 ns/op
    BenchmarkU128String/fedcba9876543210fedcba98-8     1000000   1002 ns/op
    BenchmarkU128AsBigInt/0,fedcba98-8                 5000000    238 ns/op

At least in the case of `AsBigInt`, you can use `IntoBigInt` which allows you
to recycle memory and is significantly faster:

    BenchmarkU128IntoBigInt/0,fedcba98-8             100000000   13.2 ns/op


## Credit Where Credit is Due

This is a really tricky one; the provenance of a lot of the trickier code (the
math for which is far beyond my potato brain) is difficult to determine. A lot
of it traces back to Hacker's Delight, which includes the following copyright
disclaimer:

    You are free to use, copy, and distribute any of the code on this web site,
    whether modified by you or not. You need not give attribution. This
    includes the algorithms (some of which appear in Hacker's Delight), the
    Hacker's Assistant, and any code submitted by readers. Submitters
    implicitly agree to this.

    The textural material and pictures are copyright by the author, and the
    usual copyright rules apply. E.g., you may store the material on your
    computer and make hard or soft copies for your own use. However, you may
    not incorporate this material into another publication without written
    permission from the author (which the author may give by email).

    The author has taken care in the preparation of this material, but makes no
    expressed or implied warranty of any kind and assumes no responsibility for
    errors or omissions. No liability is assumed for incidental or
    consequential damages in connection with or arising out of the use of the
    information or programs contained herein. 

Other parts, especially the division code, traces a line back to the widely
referenced Code Project article by "Jacob F. W.", found
[here](https://www.codeproject.com/Tips/785014/UInt-Division-Modulus). This code
also owes a large debt to Hacker's Delight, and is licensed under a BSD license.

The easier routines, the structure, the tester, etc are written by me (Blake
Williams) as they're obvious enough for that to be possible, but if it wasn't
for the contributions of the giants that came before, you'd be able to
bit-shift, add, negate, convert, and swear about being unable to multiply quickly
or divide at all.

Some credit should also go to "ridiculousfish" for their
[libdivide](https://github.com/ridiculousfish/libdivide/) library. There is
currently no direct code in here from this library, but it has been a huge
help. libdivide is licensed under the zlib license, which is a BSD-alike.

