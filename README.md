num: 128-bit signed and unsigned integers for Go
================================================

> [!WARNING]
> This repo has moved to sourcehut (https://git.sr.ht/~shabbyrobe/go-num). This
> version will remain here for the time being but may be removed at a later date. You
> may also consider https://pkg.go.dev/lukechampine.com/uint128 for your use case as
> I am unlikely to spend much time on this in future unless a serious bug is found.

---

Fastish `int128` (`num.I128`) and `uint128` (`num.U128`) 128-bit integer types
for Go, providing the majority of methods found in `big.Int`.

> [!WARNING]
> Function execution times in this library _almost always_ depend on the
> inputs. This library is inappropriate for use in any domain where it is important
> that the execution time does not reveal details about the inputs used.

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
