package num

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/shabbyrobe/golib/assert"
)

var i64 = I128From64

func bigI64(i int64) *big.Int { return new(big.Int).SetInt64(i) }
func bigs(s string) *big.Int {
	v, _ := new(big.Int).SetString(strings.Replace(s, " ", "", -1), 0)
	return v
}

func i128s(s string) I128 {
	s = strings.Replace(s, " ", "", -1)
	b, ok := new(big.Int).SetString(s, 0)
	if !ok {
		panic(s)
	}
	i, acc := I128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate i128 %s", s))
	}
	return i
}

func randI128(scratch []byte) I128 {
	rand.Read(scratch)
	i := I128{}
	i.lo = binary.LittleEndian.Uint64(scratch)

	if scratch[0]%2 == 1 {
		// if we always generate hi bits, the universe will die before we
		// test a number < maxInt64
		i.hi = binary.LittleEndian.Uint64(scratch[8:])
	}
	if scratch[1]%2 == 1 {
		i = i.Neg()
	}
	return i
}

func TestI128Abs(t *testing.T) {
	for idx, tc := range []struct {
		a, b I128
	}{
		{i64(0), i64(0)},
		{i64(1), i64(1)},
		{I128{lo: maxUint64}, I128{lo: maxUint64}},
		{i64(-1), i64(1)},
		{I128{hi: maxUint64}, I128{hi: 1}},

		{MinI128, MinI128}, // Overflow
	} {
		t.Run(fmt.Sprintf("%d/|%s|=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.Abs()
			tt.MustEqual(tc.b, result)
		})
	}
}

func TestI128Add(t *testing.T) {
	for idx, tc := range []struct {
		a, b, c I128
	}{
		{i64(-2), i64(-1), i64(-3)},
		{i64(-2), i64(1), i64(-1)},
		{i64(-1), i64(1), i64(0)},
		{i64(1), i64(2), i64(3)},
		{i64(10), i64(3), i64(13)},

		// Hi/lo carry:
		{I128{lo: 0xFFFFFFFFFFFFFFFF}, i64(1), I128{hi: 1, lo: 0}},
		{I128{hi: 1, lo: 0}, i64(-1), I128{lo: 0xFFFFFFFFFFFFFFFF}},

		// Overflow:
		{I128{hi: 0xFFFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}, i64(1), I128{}},

		// Overflow wraps:
		{MaxI128, i64(1), MinI128},
	} {
		t.Run(fmt.Sprintf("%d/%s+%s=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.c.Equal(tc.a.Add(tc.b)))
		})
	}
}

func TestI128AsBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a I128
		b *big.Int
	}{
		{I128{0, 2}, bigI64(2)},
		{I128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFE}, bigI64(-2)},
		{I128{0x1, 0x0}, bigs("18446744073709551616")},
		{I128{0x1, 0xFFFFFFFFFFFFFFFF}, bigs("36893488147419103231")}, // (1<<65) - 1
		{I128{0x1, 0x8AC7230489E7FFFF}, bigs("28446744073709551615")},
		{I128{0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("170141183460469231731687303715884105727")},
		{I128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("-1")},
		{I128{0x8000000000000000, 0}, bigs("-170141183460469231731687303715884105728")},
	} {
		t.Run(fmt.Sprintf("%d/%d,%d=%s", idx, tc.a.hi, tc.a.lo, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			v := tc.a.AsBigInt()
			tt.MustAssert(tc.b.Cmp(v) == 0, "found: %s", v)
		})
	}
}

func TestI128AsFloat64Random(t *testing.T) {
	tt := assert.WrapTB(t)

	bts := make([]byte, 16)

	for i := 0; i < 100000; i++ {
		rand.Read(bts)

		num := I128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])

		af := num.AsFloat64()
		bf := new(big.Float).SetFloat64(af)
		rf := num.AsBigFloat()

		diff := new(big.Float).Sub(rf, bf)
		pct := new(big.Float).Quo(diff, rf)
		tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, diff, floatDiffLimit)
	}
}

func TestI128AsFloat64(t *testing.T) {
	for _, tc := range []struct {
		a I128
	}{
		{i128s("-120")},
		{i128s("12034267329883109062163657840918528")},
		{MaxI128},
	} {
		t.Run(fmt.Sprintf("float64(%s)", tc.a), func(t *testing.T) {
			tt := assert.WrapTB(t)

			af := tc.a.AsFloat64()
			bf := new(big.Float).SetFloat64(af)
			rf := tc.a.AsBigFloat()

			diff := new(big.Float).Sub(rf, bf)
			pct := new(big.Float).Quo(diff, rf)
			tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", tc.a, diff, floatDiffLimit)
		})
	}
}

func TestI128AsInt64(t *testing.T) {
	for idx, tc := range []struct {
		a   I128
		out int64
	}{
		{i64(-1), -1},
		{i64(minInt64), minInt64},
		{i64(maxInt64), maxInt64},
		{i128s("9223372036854775808"), minInt64},  // (maxInt64 + 1) overflows to min
		{i128s("-9223372036854775809"), maxInt64}, // (minInt64 - 1) underflows to max
	} {
		t.Run(fmt.Sprintf("%d/int64(%s)=%d", idx, tc.a, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			iv := tc.a.AsInt64()
			tt.MustEqual(tc.out, iv)
		})
	}
}

func TestI128Cmp(t *testing.T) {
	for idx, tc := range []struct {
		a, b   I128
		result int
	}{
		{i64(0), i64(0), 0},
		{i64(1), i64(0), 1},
		{i64(10), i64(9), 1},
		{i64(-1), i64(1), -1},
		{i64(1), i64(-1), 1},
		{MinI128, MaxI128, -1},
	} {
		t.Run(fmt.Sprintf("%d/%s-1=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.Cmp(tc.b)
			tt.MustEqual(tc.result, result)
		})
	}
}

func TestI128Dec(t *testing.T) {
	for _, tc := range []struct {
		a, b I128
	}{
		{i64(1), i64(0)},
		{i64(10), i64(9)},
		{MinI128, MaxI128}, // underflow
		{I128{hi: 1}, I128{lo: 0xFFFFFFFFFFFFFFFF}}, // carry
	} {
		t.Run(fmt.Sprintf("%s-1=%s", tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			dec := tc.a.Dec()
			tt.MustAssert(tc.b.Equal(dec), "%s - 1 != %s, found %s", tc.a, tc.b, dec)
		})
	}
}

func TestI128FromBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a *big.Int
		b I128
	}{
		{bigI64(0), i64(0)},
		{bigI64(2), i64(2)},
		{bigI64(-2), i64(-2)},
		{bigs("18446744073709551616"), I128{0x1, 0x0}}, // 1 << 64
		{bigs("36893488147419103231"), I128{0x1, 0xFFFFFFFFFFFFFFFF}}, // (1<<65) - 1
		{bigs("28446744073709551615"), i128s("28446744073709551615")},
		{bigs("170141183460469231731687303715884105727"), i128s("170141183460469231731687303715884105727")},
		{bigs("-1"), I128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}},
	} {
		t.Run(fmt.Sprintf("%d/%s=%d,%d", idx, tc.a, tc.b.lo, tc.b.hi), func(t *testing.T) {
			tt := assert.WrapTB(t)
			v := accI128FromBigInt(tc.a)
			tt.MustAssert(tc.b.Cmp(v) == 0, "found: (%d, %d), expected (%d, %d)", v.hi, v.lo, tc.b.hi, tc.b.lo)
		})
	}
}

func TestI128FromFloat64(t *testing.T) {
	for idx, tc := range []struct {
		f       float64
		out     I128
		inRange bool
	}{
		{math.NaN(), i128s("0"), false},
		{math.Inf(0), MaxI128, false},
		{math.Inf(-1), MinI128, false},
	} {
		t.Run(fmt.Sprintf("%d/fromfloat64(%f)==%s", idx, tc.f, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)

			rn, inRange := I128FromFloat64(tc.f)
			tt.MustEqual(tc.inRange, inRange)
			diff := DifferenceI128(tc.out, rn)

			ibig, diffBig := tc.out.AsBigFloat(), diff.AsBigFloat()
			pct := new(big.Float)
			if diff != zeroI128 {
				pct.Quo(diffBig, ibig)
			}
			pct.Abs(pct)
			tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", tc.out, pct, floatDiffLimit)
		})
	}
}

func TestI128FromFloat64Random(t *testing.T) {
	tt := assert.WrapTB(t)

	bts := make([]byte, 16)

	for i := 0; i < 100000; i++ {
		rand.Read(bts)

		num := I128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])
		rbf := num.AsBigFloat()

		rf, _ := rbf.Float64()
		rn, acc := I128FromFloat64(rf)
		tt.MustAssert(acc)
		diff := DifferenceI128(num, rn)

		ibig, diffBig := num.AsBigFloat(), diff.AsBigFloat()
		pct := new(big.Float).Quo(diffBig, ibig)
		tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, pct, floatDiffLimit)
	}
}

func TestI128FromSize(t *testing.T) {
	tt := assert.WrapTB(t)
	tt.MustEqual(I128From8(127), i128s("127"))
	tt.MustEqual(I128From8(-128), i128s("-128"))
	tt.MustEqual(I128From16(32767), i128s("32767"))
	tt.MustEqual(I128From16(-32768), i128s("-32768"))
	tt.MustEqual(I128From32(2147483647), i128s("2147483647"))
	tt.MustEqual(I128From32(-2147483648), i128s("-2147483648"))
}

func TestI128Inc(t *testing.T) {
	for idx, tc := range []struct {
		a, b I128
	}{
		{i64(-1), i64(0)},
		{i64(-2), i64(-1)},
		{i64(1), i64(2)},
		{i64(10), i64(11)},
		{i64(maxInt64), i128s("9223372036854775808")},
		{i128s("18446744073709551616"), i128s("18446744073709551617")},
		{i128s("-18446744073709551617"), i128s("-18446744073709551616")},
	} {
		t.Run(fmt.Sprintf("%d/%s+1=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			inc := tc.a.Inc()
			tt.MustAssert(tc.b.Equal(inc), "%s + 1 != %s, found %s", tc.a, tc.b, inc)
		})
	}
}

func TestI128IsInt64(t *testing.T) {
	for idx, tc := range []struct {
		a  I128
		is bool
	}{
		{i64(-1), true},
		{i64(minInt64), true},
		{i64(maxInt64), true},
		{i128s("9223372036854775808"), false},  // (maxInt64 + 1)
		{i128s("-9223372036854775809"), false}, // (minInt64 - 1)
	} {
		t.Run(fmt.Sprintf("%d/isint64(%s)=%v", idx, tc.a, tc.is), func(t *testing.T) {
			tt := assert.WrapTB(t)
			iv := tc.a.IsInt64()
			tt.MustEqual(tc.is, iv)
		})
	}
}

func TestI128MarshalJSON(t *testing.T) {
	tt := assert.WrapTB(t)
	bts := make([]byte, 16)

	for i := 0; i < 5000; i++ {
		n := randI128(bts)

		bts, err := json.Marshal(n)
		tt.MustOK(err)

		var result I128
		tt.MustOK(json.Unmarshal(bts, &result))
		tt.MustAssert(result.Equal(n))
	}
}

func TestI128Mul(t *testing.T) {
	for _, tc := range []struct {
		a, b, out I128
	}{
		{i64(1), i64(0), i64(0)},
		{i64(-2), i64(2), i64(-4)},
		{i64(-2), i64(-2), i64(4)},
		{i64(10), i64(9), i64(90)},
		{i64(maxInt64), i64(maxInt64), i128s("85070591730234615847396907784232501249")},
		{i64(minInt64), i64(minInt64), i128s("85070591730234615865843651857942052864")},
		{i64(minInt64), i64(maxInt64), i128s("-85070591730234615856620279821087277056")},
		{MaxI128, i64(2), i128s("-2")}, // Overflow. "math.MaxInt64 * 2" produces the same result, "-2".
		{MaxI128, MaxI128, i128s("1")}, // Overflow
	} {
		t.Run(fmt.Sprintf("%s*%s=%s", tc.a, tc.b, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)

			v := tc.a.Mul(tc.b)
			tt.MustAssert(tc.out.Equal(v), "%s * %s != %s, found %s", tc.a, tc.b, tc.out, v)
		})
	}
}

func TestI128Neg(t *testing.T) {
	for idx, tc := range []struct {
		a, b I128
	}{
		{i64(0), i64(0)},
		{i64(-2), i64(2)},
		{i64(2), i64(-2)},

		// hi/lo carry:
		{I128{lo: 0xFFFFFFFFFFFFFFFF}, I128{hi: 0xFFFFFFFFFFFFFFFF, lo: 1}},
		{I128{hi: 0xFFFFFFFFFFFFFFFF, lo: 1}, I128{lo: 0xFFFFFFFFFFFFFFFF}},

		// These cases popped up as a weird regression when refactoring I128FromBigInt:
		{i128s("18446744073709551616"), i128s("-18446744073709551616")},
		{i128s("-18446744073709551616"), i128s("18446744073709551616")},
		{i128s("-18446744073709551617"), i128s("18446744073709551617")},
		{I128{hi: 1, lo: 0}, I128{hi: 0xFFFFFFFFFFFFFFFF, lo: 0x0}},

		{i128s("28446744073709551615"), i128s("-28446744073709551615")},
		{i128s("-28446744073709551615"), i128s("28446744073709551615")},

		// Negating MaxI128 should yield MinI128 + 1:
		{I128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}, I128{hi: 0x8000000000000000, lo: 1}},

		// Negating MinI128 should yield MinI128:
		{I128{hi: 0x8000000000000000, lo: 0}, I128{hi: 0x8000000000000000, lo: 0}},
	} {
		t.Run(fmt.Sprintf("%d/-%s=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.Neg()
			tt.MustAssert(tc.b.Equal(result))
		})
	}
}

func TestI128QuoRem(t *testing.T) {
	for _, tc := range []struct {
		i, by, q, r I128
	}{
		{i: i64(1), by: i64(2), q: i64(0), r: i64(1)},
		{i: i64(10), by: i64(3), q: i64(3), r: i64(1)},
		{i: i64(10), by: i64(-3), q: i64(-3), r: i64(1)},
		{i: i64(10), by: i64(10), q: i64(1), r: i64(0)},

		// Hit the 128-bit division 'lz+tz == 127' branch:
		{i: i128s("0x10000000000000000"), by: i128s("0x10000000000000000"), q: i64(1), r: i64(0)},

		// Hit the 128-bit division 'cmp == 0' branch
		{i: i128s("0x12345678901234567"), by: i128s("0x12345678901234567"), q: i64(1), r: i64(0)},
	} {
		t.Run(fmt.Sprintf("%s÷%s=%s,%s", tc.i, tc.by, tc.q, tc.r), func(t *testing.T) {
			tt := assert.WrapTB(t)
			q, r := tc.i.QuoRem(tc.by)
			tt.MustEqual(tc.q.String(), q.String())
			tt.MustEqual(tc.r.String(), r.String())

			iBig := tc.i.AsBigInt()
			byBig := tc.by.AsBigInt()

			qBig, rBig := new(big.Int).Set(iBig), new(big.Int).Set(iBig)
			qBig = qBig.Div(qBig, byBig)
			rBig = rBig.Mod(rBig, byBig)

			tt.MustEqual(tc.q.String(), qBig.String())
			tt.MustEqual(tc.r.String(), rBig.String())
		})
	}
}

func TestI128Sub(t *testing.T) {
	for idx, tc := range []struct {
		a, b, c I128
	}{
		{i64(-2), i64(-1), i64(-1)},
		{i64(-2), i64(1), i64(-3)},
		{i64(2), i64(1), i64(1)},
		{i64(2), i64(-1), i64(3)},
		{i64(1), i64(2), i64(-1)},  // crossing zero
		{i64(-1), i64(-2), i64(1)}, // crossing zero

		{MinI128, i64(1), MaxI128},  // Overflow wraps
		{MaxI128, i64(-1), MinI128}, // Overflow wraps

		{i128s("0x10000000000000000"), i64(1), i128s("0xFFFFFFFFFFFFFFFF")},  // carry down
		{i128s("0xFFFFFFFFFFFFFFFF"), i64(-1), i128s("0x10000000000000000")}, // carry up

		// {i64(maxInt64), i64(1), i128s("18446744073709551616")}, // lo carries to hi
		// {i128s("18446744073709551615"), i128s("18446744073709551615"), i128s("36893488147419103230")},
	} {
		t.Run(fmt.Sprintf("%d/%s-%s=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.c.Equal(tc.a.Sub(tc.b)))
		})
	}
}

var (
	BenchI128Result I128
)

func BenchmarkI128FromBigInt(b *testing.B) {
	for _, bi := range []*big.Int{
		bigs("0"),
		bigs("0xfedcba98"),
		bigs("0xfedcba9876543210"),
		bigs("0xfedcba9876543210fedcba98"),
		bigs("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchI128Result, _ = I128FromBigInt(bi)
			}
		})
	}
}

func BenchmarkI128LessThan(b *testing.B) {
	for _, iv := range []struct {
		a, b I128
	}{
		{i64(1), i64(1)},
		{i64(2), i64(1)},
		{i64(1), i64(2)},
		{i64(-1), i64(-1)},
		{i64(-1), i64(-2)},
		{i64(-2), i64(-1)},
	} {
		b.Run(fmt.Sprintf("%s<%s", iv.a, iv.b), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = iv.a.LessThan(iv.b)
			}
		})
	}
}

func BenchmarkI128Sub(b *testing.B) {
	sub := i64(1)
	for _, iv := range []I128{i64(1), i128s("0x10000000000000000"), MaxI128} {
		b.Run(fmt.Sprintf("%s", iv), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchI128Result = iv.Sub(sub)
			}
		})
	}
}
