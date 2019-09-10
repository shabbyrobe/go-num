package num

import (
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
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

		{MaxI128, MaxI128}, // Should work
		{MinI128, MinI128}, // Overflow
	} {
		t.Run(fmt.Sprintf("%d/|%s|=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.Abs()
			tt.MustEqual(tc.b, result)
		})
	}
}

func TestI128AbsU128(t *testing.T) {
	for idx, tc := range []struct {
		a I128
		b U128
	}{
		{i64(0), u64(0)},
		{i64(1), u64(1)},
		{I128{lo: maxUint64}, U128{lo: maxUint64}},
		{i64(-1), u64(1)},
		{I128{hi: maxUint64}, U128{hi: 1}},

		{MinI128, minI128AsAbsU128}, // Overflow does not affect this function
	} {
		t.Run(fmt.Sprintf("%d/|%s|=%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.AbsU128()
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

func TestI128AsBigIntAndIntoBigInt(t *testing.T) {
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

			var v2 big.Int
			tc.a.IntoBigInt(&v2)
			tt.MustAssert(tc.b.Cmp(&v2) == 0, "found: %s", v2)
		})
	}
}

func TestI128AsFloat64Random(t *testing.T) {
	tt := assert.WrapTB(t)

	bts := make([]byte, 16)

	for i := 0; i < 1000; i++ {
		for bits := uint(1); bits <= 127; bits++ {
			rand.Read(bts)

			var loMask, hiMask uint64
			var loSet, hiSet uint64
			if bits > 64 {
				loMask = maxUint64
				hiMask = (1 << (bits - 64)) - 1
				hiSet = 1 << (bits - 64 - 1)
			} else {
				loMask = (1 << bits) - 1
				loSet = 1 << (bits - 1)
			}

			num := I128{}
			num.lo = (binary.LittleEndian.Uint64(bts) & loMask) | loSet
			num.hi = (binary.LittleEndian.Uint64(bts[8:]) & hiMask) | hiSet

			for neg := 0; neg <= 1; neg++ {
				if neg == 1 {
					num = num.Neg()
				}

				af := num.AsFloat64()
				bf := new(big.Float).SetFloat64(af)
				rf := num.AsBigFloat()

				diff := new(big.Float).Sub(rf, bf)
				pct := new(big.Float).Quo(diff, rf)
				tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, diff, floatDiffLimit)
			}
		}
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

func TestI128Format(t *testing.T) {
	for _, tc := range []struct {
		in  I128
		f   string
		out string
	}{
		{i64(123456789), "%d", "123456789"},
		{i64(12), "%2d", "12"},
		{i64(12), "%3d", " 12"},
		{i64(12), "%02d", "12"},
		{i64(12), "%03d", "012"},
		{i64(123456789), "%s", "123456789"},
	} {
		t.Run("", func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustEqual(tc.out, fmt.Sprintf(tc.f, tc.in))
		})
	}
}

func TestI128From64(t *testing.T) {
	for idx, tc := range []struct {
		in  int64
		out I128
	}{
		{0, i64(0)},
		{maxInt64, i128s("0x7F FF FF FF FF FF FF FF")},
		{-1, i128s("-1")},
		{minInt64, i128s("-9223372036854775808")},
	} {
		t.Run(fmt.Sprintf("%d/%d=%s", idx, tc.in, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := I128From64(tc.in)
			tt.MustEqual(tc.out, result, "found: (%d, %d), expected (%d, %d)", result.hi, result.lo, tc.out.hi, tc.out.lo)
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

func TestI128MarshalText(t *testing.T) {
	tt := assert.WrapTB(t)
	bts := make([]byte, 16)

	type Encoded struct {
		Num I128
	}

	for i := 0; i < 5000; i++ {
		n := randI128(bts)

		var v = Encoded{Num: n}

		out, err := xml.Marshal(&v)
		tt.MustOK(err)

		tt.MustEqual(fmt.Sprintf("<Encoded><Num>%s</Num></Encoded>", n.String()), string(out))

		var v2 Encoded
		tt.MustOK(xml.Unmarshal(out, &v2))

		tt.MustEqual(v2.Num, n)
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

func TestI128MustInt64(t *testing.T) {
	for _, tc := range []struct {
		a  I128
		ok bool
	}{
		{i64(0), true},
		{i64(1), true},
		{i64(maxInt64), true},
		{i128s("9223372036854775808"), false},
		{MaxI128, false},

		{i64(-1), true},
		{i64(minInt64), true},
		{i128s("-9223372036854775809"), false},
		{MinI128, false},
	} {
		t.Run(fmt.Sprintf("(%s).64?==%v", tc.a, tc.ok), func(t *testing.T) {
			tt := assert.WrapTB(t)
			defer func() {
				tt.Helper()
				tt.MustAssert((recover() == nil) == tc.ok)
			}()

			tt.MustEqual(tc.a, I128From64(tc.a.MustInt64()))
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

		{i128s("-170141183460469231731687303715884105728"), i128s("-170141183460469231731687303715884105728")},
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

		{i: MinI128, by: i64(-1), q: MinI128, r: zeroI128},
	} {
		t.Run(fmt.Sprintf("%s÷%s=%s,%s", tc.i, tc.by, tc.q, tc.r), func(t *testing.T) {
			tt := assert.WrapTB(t)
			q, r := tc.i.QuoRem(tc.by)
			tt.MustEqual(tc.q.String(), q.String())
			tt.MustEqual(tc.r.String(), r.String())

			// Skip the weird overflow edge case where we divide MinI128 by -1:
			// this effectively becomes a negation operation, which overflows:
			//
			//   -170141183460469231731687303715884105728 / -1 == -170141183460469231731687303715884105728
			//
			if tc.i != MinI128 {
				iBig := tc.i.AsBigInt()
				byBig := tc.by.AsBigInt()

				qBig, rBig := new(big.Int).Set(iBig), new(big.Int).Set(iBig)
				qBig = qBig.Div(qBig, byBig)
				rBig = rBig.Mod(rBig, byBig)

				tt.MustEqual(tc.q.String(), qBig.String())
				tt.MustEqual(tc.r.String(), rBig.String())
			}
		})
	}
}

func TestI128Scan(t *testing.T) {
	for idx, tc := range []struct {
		in  string
		out I128
		ok  bool
	}{
		{"1", i64(1), true},
		{"0xFF", zeroI128, false},
		{"-1", i64(-1), true},
		{"170141183460469231731687303715884105728", zeroI128, false},
		{"-170141183460469231731687303715884105729", zeroI128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.in, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			var result I128
			n, err := fmt.Sscan(tc.in, &result)
			tt.MustEqual(tc.ok, err == nil, "%v", err)
			if err == nil {
				tt.MustEqual(1, n)
			} else {
				tt.MustEqual(0, n)
			}
			tt.MustEqual(tc.out, result)
		})
	}
}

func TestI128Sign(t *testing.T) {
	for idx, tc := range []struct {
		a    I128
		sign int
	}{
		{i64(0), 0},
		{i64(1), 1},
		{i64(-1), -1},
		{MinI128, -1},
		{MaxI128, 1},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.a, tc.sign), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := tc.a.Sign()
			tt.MustEqual(tc.sign, result)
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

func TestI128Sub64(t *testing.T) {
	for idx, tc := range []struct {
		a I128
		b int64
		c I128
	}{
		{i64(-2), -1, i64(-1)},
		{i64(-2), 1, i64(-3)},
		{i64(2), 1, i64(1)},
		{i64(2), -1, i64(3)},
		{i64(1), 2, i64(-1)},  // crossing zero
		{i64(-1), -2, i64(1)}, // crossing zero

		{MinI128, 1, MaxI128},  // Overflow wraps
		{MaxI128, -1, MinI128}, // Overflow wraps

		{i128s("0x10000000000000000"), 1, i128s("0xFFFFFFFFFFFFFFFF")},  // carry down
		{i128s("0xFFFFFFFFFFFFFFFF"), -1, i128s("0x10000000000000000")}, // carry up
	} {
		t.Run(fmt.Sprintf("%d/%s-%d=%s", idx, tc.a, tc.b, tc.c), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.c.Equal(tc.a.Sub64(tc.b)))
		})
	}
}

var (
	BenchI128Result            I128
	BenchInt64Result           int64
	BenchmarkI128Float64Result float64
)

func BenchmarkI128Add(b *testing.B) {
	for idx, tc := range []struct {
		a, b I128
		name string
	}{
		{zeroI128, zeroI128, "0+0"},
		{MaxI128, MaxI128, "max+max"},
		{i128s("0x7FFFFFFFFFFFFFFF"), i128s("0x7FFFFFFFFFFFFFFF"), "lo-only"},
		{i128s("0xFFFFFFFFFFFFFFFF"), i128s("0x7FFFFFFFFFFFFFFF"), "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchI128Result = tc.a.Add(tc.b)
			}
		})
	}
}

func BenchmarkI128Add64(b *testing.B) {
	for idx, tc := range []struct {
		a    I128
		b    int64
		name string
	}{
		{zeroI128, 0, "0+0"},
		{MaxI128, maxInt64, "max+max"},
		{i64(-1), -1, "-1+-1"},
		{i64(-1), 1, "-1+1"},
		{i64(minInt64), -1, "-min64-1"},
		{i128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchI128Result = tc.a.Add64(tc.b)
			}
		})
	}
}

func BenchmarkI128AsFloat(b *testing.B) {
	for idx, tc := range []struct {
		name string
		in   I128
	}{
		{"zero", I128{}},
		{"one", i64(1)},
		{"minusone", i64(-1)},
		{"maxInt64", i64(maxInt64)},
		{"gt64bit", i128s("0x1 00000000 00000000")},
		{"minInt64", i64(minInt64)},
		{"minusgt64bit", i128s("-0x1 00000000 00000000")},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchmarkI128Float64Result = tc.in.AsFloat64()
			}
		})
	}
}

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

var (
	I64CastInput int64  = 0x7FFFFFFFFFFFFFFF
	I32CastInput int32  = 0x7FFFFFFF
	U64CastInput uint64 = 0x7FFFFFFFFFFFFFFF
)

func BenchmarkI128FromCast(b *testing.B) {
	// Establish a baseline for a runtime 64-bit cast:
	b.Run("I64FromU64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchInt64Result = int64(U64CastInput)
		}
	})

	b.Run("I128FromI64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchI128Result = I128From64(I64CastInput)
		}
	})
	b.Run("I128FromU64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchI128Result = I128FromU64(U64CastInput)
		}
	})
	b.Run("I128FromI32", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			BenchI128Result = I128From32(I32CastInput)
		}
	})
}

func BenchmarkI128FromFloat(b *testing.B) {
	for _, pow := range []float64{1, 63, 64, 65, 127, 128} {
		b.Run(fmt.Sprintf("pow%d", int(pow)), func(b *testing.B) {
			f := math.Pow(2, pow)
			for i := 0; i < b.N; i++ {
				BenchI128Result, _ = I128FromFloat64(f)
			}
		})
	}
}

func BenchmarkI128IsZero(b *testing.B) {
	for idx, tc := range []struct {
		name string
		v    I128
	}{
		{"0", zeroI128},
		{"hizero", i64(1)},
		{"nozero", MaxI128},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = tc.v.IsZero()
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
		{MaxI128, MinI128},
		{MinI128, MaxI128},
	} {
		b.Run(fmt.Sprintf("%s<%s", iv.a, iv.b), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = iv.a.LessThan(iv.b)
			}
		})
	}
}

func BenchmarkI128LessOrEqualTo(b *testing.B) {
	for _, iv := range []struct {
		a, b I128
	}{
		{i64(1), i64(1)},
		{i64(2), i64(1)},
		{i64(1), i64(2)},
		{i64(-1), i64(-1)},
		{i64(-1), i64(-2)},
		{i64(-2), i64(-1)},
		{MaxI128, MinI128},
		{MinI128, MaxI128},
	} {
		b.Run(fmt.Sprintf("%s<%s", iv.a, iv.b), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = iv.a.LessOrEqualTo(iv.b)
			}
		})
	}
}

func BenchmarkI128Mul(b *testing.B) {
	v := I128From64(maxInt64)
	for i := 0; i < b.N; i++ {
		BenchI128Result = v.Mul(v)
	}
}

func BenchmarkI128Mul64(b *testing.B) {
	v := I128From64(maxInt64)
	lim := int64(b.N)
	for i := int64(0); i < lim; i++ {
		BenchI128Result = v.Mul64(i)
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
