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

	"github.com/shabbyrobe/go-num/internal/assert"
)

var u64 = U128From64

func bigU64(u uint64) *big.Int { return new(big.Int).SetUint64(u) }

func u128s(s string) U128 {
	s = strings.Replace(s, " ", "", -1)
	b, ok := new(big.Int).SetString(s, 0)
	if !ok {
		panic(fmt.Errorf("num: u128 string %q invalid", s))
	}
	out, acc := U128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate u128 %s", s))
	}
	return out
}

func randU128(scratch []byte) U128 {
	rand.Read(scratch)
	u := U128{}
	u.lo = binary.LittleEndian.Uint64(scratch)

	if scratch[0]%2 == 1 {
		// if we always generate hi bits, the universe will die before we
		// test a number < maxInt64
		u.hi = binary.LittleEndian.Uint64(scratch[8:])
	}
	return u
}

func TestLargerSmallerU128(t *testing.T) {
	for idx, tc := range []struct {
		a, b        U128
		firstLarger bool
	}{
		{u64(0), u64(1), false},
		{MaxU128, u64(1), true},
		{u64(1), u64(1), false},
		{u64(2), u64(1), true},
		{u128s("0xFFFFFFFF FFFFFFFF"), u128s("0x1 00000000 00000000"), false},
		{u128s("0x1 00000000 00000000"), u128s("0xFFFFFFFF FFFFFFFF"), true},
	} {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			tt := assert.WrapTB(t)
			if tc.firstLarger {
				tt.MustEqual(tc.a, LargerU128(tc.a, tc.b))
				tt.MustEqual(tc.b, SmallerU128(tc.a, tc.b))
			} else {
				tt.MustEqual(tc.b, LargerU128(tc.a, tc.b))
				tt.MustEqual(tc.a, SmallerU128(tc.a, tc.b))
			}
		})
	}
}

func TestMustU128FromI64(t *testing.T) {
	tt := assert.WrapTB(t)
	assert := func(ok bool, expected U128, v int64) {
		tt.Helper()
		defer func() {
			tt.Helper()
			tt.MustAssert((recover() == nil) == ok)
		}()
		tt.MustEqual(expected, MustU128FromI64(v))
	}

	assert(true, u128s("1234"), 1234)
	assert(false, u64(0), -1234)
}

func TestMustU128FromString(t *testing.T) {
	tt := assert.WrapTB(t)
	assert := func(ok bool, expected U128, s string) {
		tt.Helper()
		defer func() {
			tt.Helper()
			tt.MustAssert((recover() == nil) == ok)
		}()
		tt.MustEqual(expected, MustU128FromString(s))
	}

	assert(true, u128s("1234"), "1234")
	assert(false, u64(0), "quack")
	assert(false, u64(0), "120481092481092840918209481092380192830912830918230918")
}

func TestU128Add(t *testing.T) {
	for _, tc := range []struct {
		a, b, c U128
	}{
		{u64(1), u64(2), u64(3)},
		{u64(10), u64(3), u64(13)},
		{MaxU128, u64(1), u64(0)},                               // Overflow wraps
		{u64(maxUint64), u64(1), u128s("18446744073709551616")}, // lo carries to hi
		{u128s("18446744073709551615"), u128s("18446744073709551615"), u128s("36893488147419103230")},
	} {
		t.Run(fmt.Sprintf("%s+%s=%s", tc.a, tc.b, tc.c), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.c.Equal(tc.a.Add(tc.b)))
		})
	}
}

func TestU128Add64(t *testing.T) {
	for _, tc := range []struct {
		a U128
		b uint64
		c U128
	}{
		{u64(1), 2, u64(3)},
		{u64(10), 3, u64(13)},
		{MaxU128, 1, u64(0)}, // Overflow wraps
	} {
		t.Run(fmt.Sprintf("%s+%d=%s", tc.a, tc.b, tc.c), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.c.Equal(tc.a.Add64(tc.b)))
		})
	}
}

func TestU128AsBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a U128
		b *big.Int
	}{
		{U128{0, 2}, bigU64(2)},
		{U128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFE}, bigs("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFE")},
		{U128{0x1, 0x0}, bigs("18446744073709551616")},
		{U128{0x1, 0xFFFFFFFFFFFFFFFF}, bigs("36893488147419103231")}, // (1<<65) - 1
		{U128{0x1, 0x8AC7230489E7FFFF}, bigs("28446744073709551615")},
		{U128{0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("170141183460469231731687303715884105727")},
		{U128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF")},
		{U128{0x8000000000000000, 0}, bigs("0x 8000000000000000 0000000000000000")},
	} {
		t.Run(fmt.Sprintf("%d/%d,%d=%s", idx, tc.a.hi, tc.a.lo, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			v := tc.a.AsBigInt()
			tt.MustAssert(tc.b.Cmp(v) == 0, "found: %s", v)
		})
	}
}

func TestU128AsFloat64Random(t *testing.T) {
	tt := assert.WrapTB(t)

	bts := make([]byte, 16)

	for i := 0; i < 10000; i++ {
		rand.Read(bts)

		num := U128{}
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

func TestU128AsFloat64Direct(t *testing.T) {
	for _, tc := range []struct {
		a   U128
		out string
	}{
		{u128s("2384067163226812360730"), "2384067163226812448768"},
	} {
		t.Run(fmt.Sprintf("float64(%s)=%s", tc.a, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustEqual(tc.out, cleanFloatStr(fmt.Sprintf("%f", tc.a.AsFloat64())))
		})
	}
}

func TestU128AsFloat64Epsilon(t *testing.T) {
	for _, tc := range []struct {
		a U128
	}{
		{u128s("120")},
		{u128s("12034267329883109062163657840918528")},
		{MaxU128},
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

func TestU128Dec(t *testing.T) {
	for _, tc := range []struct {
		a, b U128
	}{
		{u64(1), u64(0)},
		{u64(10), u64(9)},
		{u64(maxUint64), u128s("18446744073709551614")},
		{u64(0), MaxU128},
		{u64(maxUint64).Add(u64(1)), u64(maxUint64)},
	} {
		t.Run(fmt.Sprintf("%s-1=%s", tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			dec := tc.a.Dec()
			tt.MustAssert(tc.b.Equal(dec), "%s - 1 != %s, found %s", tc.a, tc.b, dec)
		})
	}
}

func TestU128Format(t *testing.T) {
	for idx, tc := range []struct {
		v   U128
		fmt string
		out string
	}{
		{u64(1), "%d", "1"},
		{u64(1), "%s", "1"},
		{u64(1), "%v", "1"},
		{MaxU128, "%d", "340282366920938463463374607431768211455"},
		{MaxU128, "%#d", "340282366920938463463374607431768211455"},
		{MaxU128, "%o", "3777777777777777777777777777777777777777777"},
		{MaxU128, "%b", strings.Repeat("1", 128)},
		{MaxU128, "%#o", "03777777777777777777777777777777777777777777"},
		{MaxU128, "%#x", "0xffffffffffffffffffffffffffffffff"},
		{MaxU128, "%#X", "0XFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},

		// No idea why big.Int doesn't support this:
		// {MaxU128, "%#b", "0b" + strings.Repeat("1", 128)},
	} {
		t.Run(fmt.Sprintf("%d/%s/%s", idx, tc.fmt, tc.v), func(t *testing.T) {
			tt := assert.WrapTB(t)
			result := fmt.Sprintf(tc.fmt, tc.v)
			tt.MustEqual(tc.out, result)
		})
	}
}

func TestU128FromBigInt(t *testing.T) {
	for idx, tc := range []struct {
		a   *big.Int
		b   U128
		acc bool
	}{
		{bigU64(2), u64(2), true},
		{bigs("18446744073709551616"), U128{hi: 0x1, lo: 0x0}, true},                // 1 << 64
		{bigs("36893488147419103231"), U128{hi: 0x1, lo: 0xFFFFFFFFFFFFFFFF}, true}, // (1<<65) - 1
		{bigs("28446744073709551615"), u128s("28446744073709551615"), true},
		{bigs("170141183460469231731687303715884105727"), u128s("170141183460469231731687303715884105727"), true},
		{bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), U128{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF}, true},
		{bigs("0x 1 0000000000000000 00000000000000000"), MaxU128, false},
		{bigs("0x FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFFF"), MaxU128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s=%d,%d", idx, tc.a, tc.b.lo, tc.b.hi), func(t *testing.T) {
			tt := assert.WrapTB(t)
			v, acc := U128FromBigInt(tc.a)
			tt.MustEqual(acc, tc.acc)
			tt.MustAssert(tc.b.Cmp(v) == 0, "found: (%d, %d), expected (%d, %d)", v.hi, v.lo, tc.b.hi, tc.b.lo)
		})
	}
}

func TestU128FromFloat64Random(t *testing.T) {
	tt := assert.WrapTB(t)

	bts := make([]byte, 16)

	for i := 0; i < 10000; i++ {
		rand.Read(bts)

		num := U128{}
		num.lo = binary.LittleEndian.Uint64(bts)
		num.hi = binary.LittleEndian.Uint64(bts[8:])
		rbf := num.AsBigFloat()

		rf, _ := rbf.Float64()
		rn, inRange := U128FromFloat64(rf)
		tt.MustAssert(inRange)

		diff := DifferenceU128(num, rn)

		ibig, diffBig := num.AsBigFloat(), diff.AsBigFloat()
		pct := new(big.Float).Quo(diffBig, ibig)
		tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%s: %.20f > %.20f", num, pct, floatDiffLimit)
	}
}

func TestU128FromFloat64(t *testing.T) {
	for idx, tc := range []struct {
		f       float64
		out     U128
		inRange bool
	}{
		{math.NaN(), u128s("0"), false},
		{math.Inf(0), MaxU128, false},
		{math.Inf(-1), u128s("0"), false},
		{1.0, u64(1), true},

		// {{{ Explore weird corner cases around uint64(float64(math.MaxUint64)) nonsense.
		// 1 greater than maxUint64 because maxUint64 is not representable in a float64 exactly:
		{maxUint64Float, u128s("18446744073709551616"), true},

		// Largest whole number representable in a float64 without exceeding the size of a uint64:
		{maxRepresentableUint64Float, u128s("18446744073709549568"), true},

		// Largest whole number representable in a float64 without exceeding the size of a U128:
		{maxRepresentableU128Float, u128s("340282366920938425684442744474606501888"), true},

		// Not inRange because maxU128Float is not representable in a float64 exactly:
		{maxU128Float, MaxU128, false},
		// }}}
	} {
		t.Run(fmt.Sprintf("%d/fromfloat64(%f)==%s", idx, tc.f, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)

			rn, inRange := U128FromFloat64(tc.f)
			tt.MustEqual(tc.inRange, inRange)
			tt.MustEqual(tc.out, rn)

			diff := DifferenceU128(tc.out, rn)

			ibig, diffBig := tc.out.AsBigFloat(), diff.AsBigFloat()
			pct := new(big.Float)
			if diff != zeroU128 {
				pct.Quo(diffBig, ibig)
			}
			pct.Abs(pct)
			tt.MustAssert(pct.Cmp(floatDiffLimit) < 0, "%.20f -> %s: %.20f > %.20f", tc.f, tc.out, pct, floatDiffLimit)
		})
	}
}

func TestU128FromI64(t *testing.T) {
	for idx, tc := range []struct {
		in      int64
		out     U128
		inRange bool
	}{
		{0, zeroU128, true},
		{-1, zeroU128, false},
		{minInt64, zeroU128, false},
		{maxInt64, u64(0x7fffffffffffffff), true},
	} {
		t.Run(fmt.Sprintf("%d/fromint64(%d)==%s", idx, tc.in, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			rn, inRange := U128FromI64(tc.in)
			tt.MustAssert(rn.Equal(tc.out))
			tt.MustEqual(tc.inRange, inRange)
		})
	}
}

func TestU128FromSize(t *testing.T) {
	tt := assert.WrapTB(t)
	assertInRange := func(expected U128) func(v U128, inRange bool) {
		return func(v U128, inRange bool) {
			tt.MustEqual(expected, v)
			tt.MustAssert(inRange)
		}
	}

	tt.MustEqual(U128From8(255), u128s("255"))
	tt.MustEqual(U128From16(65535), u128s("65535"))
	tt.MustEqual(U128From32(4294967295), u128s("4294967295"))

	assertInRange(u128s("12345"))(U128FromFloat32(12345))
	assertInRange(u128s("12345"))(U128FromFloat32(12345.6))
	assertInRange(u128s("12345"))(U128FromFloat64(12345))
	assertInRange(u128s("12345"))(U128FromFloat64(12345.6))
}

func TestU128Inc(t *testing.T) {
	for _, tc := range []struct {
		a, b U128
	}{
		{u64(1), u64(2)},
		{u64(10), u64(11)},
		{u64(maxUint64), u128s("18446744073709551616")},
		{u64(maxUint64), u64(maxUint64).Add(u64(1))},
		{MaxU128, u64(0)},
	} {
		t.Run(fmt.Sprintf("%s+1=%s", tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			inc := tc.a.Inc()
			tt.MustAssert(tc.b.Equal(inc), "%s + 1 != %s, found %s", tc.a, tc.b, inc)
		})
	}
}

func TestU128Lsh(t *testing.T) {
	for idx, tc := range []struct {
		u  U128
		by uint
		r  U128
	}{
		{u: u64(2), by: 1, r: u64(4)},
		{u: u64(1), by: 2, r: u64(4)},
		{u: u128s("18446744073709551615"), by: 1, r: u128s("36893488147419103230")}, // (1<<64) - 1

		// These cases were found by the fuzzer:
		{u: u128s("5080864651895"), by: 57, r: u128s("732229764895815899943471677440")},
		{u: u128s("63669103"), by: 85, r: u128s("2463079120908903847397520463364096")},
		{u: u128s("0x1f1ecfd29cb51500c1a0699657"), by: 104, r: u128s("0x69965700000000000000000000000000")},
		{u: u128s("0x4ff0d215cf8c26f26344"), by: 58, r: u128s("0xc348573e309bc98d1000000000000000")},
		{u: u128s("0x6b5823decd7ef067f78e8cc3d8"), by: 74, r: u128s("0xc19fde3a330f60000000000000000000")},
		{u: u128s("0x8b93924e1f7b6ac551d66f18ab520a2"), by: 50, r: u128s("0xdab154759bc62ad48288000000000000")},
		{u: u128s("173760885"), by: 68, r: u128s("51285161209860430747989442560")},
		{u: u128s("213"), by: 65, r: u128s("7858312975400268988416")},
		{u: u128s("0x2203b9f3dbe0afa82d80d998641aa0"), by: 75, r: u128s("0x6c06ccc320d500000000000000000000")},
		{u: u128s("40625"), by: 55, r: u128s("1463669878895411200000")},
	} {
		t.Run(fmt.Sprintf("%d/%s<<%d=%s", idx, tc.u, tc.by, tc.r), func(t *testing.T) {
			tt := assert.WrapTB(t)

			ub := tc.u.AsBigInt()
			ub.Lsh(ub, tc.by).And(ub, maxBigU128)

			ru := tc.u.Lsh(tc.by)
			tt.MustEqual(tc.r.String(), ru.String(), "%s != %s; big: %s", tc.r, ru, ub)
			tt.MustEqual(ub.String(), ru.String())
		})
	}
}

func TestU128MarshalJSON(t *testing.T) {
	tt := assert.WrapTB(t)
	bts := make([]byte, 16)

	for i := 0; i < 5000; i++ {
		u := randU128(bts)

		bts, err := json.Marshal(u)
		tt.MustOK(err)

		var result U128
		tt.MustOK(json.Unmarshal(bts, &result))
		tt.MustAssert(result.Equal(u))
	}
}

func TestU128Mul(t *testing.T) {
	tt := assert.WrapTB(t)

	u := U128From64(maxUint64)
	v := u.Mul(U128From64(maxUint64))

	var v1, v2 big.Int
	v1.SetUint64(maxUint64)
	v2.SetUint64(maxUint64)
	tt.MustEqual(v.String(), v1.Mul(&v1, &v2).String())
}

func TestU128MustUint64(t *testing.T) {
	for _, tc := range []struct {
		a  U128
		ok bool
	}{
		{u64(0), true},
		{u64(1), true},
		{u64(maxInt64), true},
		{u64(maxUint64), true},
		{U128FromRaw(1, 0), false},
		{MaxU128, false},
	} {
		t.Run(fmt.Sprintf("(%s).64?==%v", tc.a, tc.ok), func(t *testing.T) {
			tt := assert.WrapTB(t)
			defer func() {
				tt.Helper()
				tt.MustAssert((recover() == nil) == tc.ok)
			}()

			tt.MustEqual(tc.a, U128From64(tc.a.MustUint64()))
		})
	}
}

func TestU128Not(t *testing.T) {
	for idx, tc := range []struct {
		a, b U128
	}{
		{u64(0), MaxU128},
		{u64(1), u128s("340282366920938463463374607431768211454")},
		{u64(2), u128s("340282366920938463463374607431768211453")},
		{u64(maxUint64), u128s("340282366920938463444927863358058659840")},
	} {
		t.Run(fmt.Sprintf("%d/%s=^%s", idx, tc.a, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			out := tc.a.Not()
			tt.MustAssert(tc.b.Equal(out), "^%s != %s, found %s", tc.a, tc.b, out)

			back := out.Not()
			tt.MustAssert(tc.a.Equal(back), "^%s != %s, found %s", out, tc.a, back)
		})
	}
}

func TestU128QuoRem(t *testing.T) {
	for idx, tc := range []struct {
		u, by, q, r U128
	}{
		{u: u64(1), by: u64(2), q: u64(0), r: u64(1)},
		{u: u64(10), by: u64(3), q: u64(3), r: u64(1)},

		// Investigate possible div/0 where lo of divisor is 0:
		{u: U128{hi: 0, lo: 1}, by: U128{hi: 1, lo: 0}, q: u64(0), r: u64(1)},

		// 128-bit 'cmp == 0' shortcut branch:
		{u128s("0x1234567890123456"), u128s("0x1234567890123456"), u64(1), u64(0)},

		// 128-bit 'cmp < 0' shortcut branch:
		{u128s("0x123456789012345678901234"), u128s("0x222222229012345678901234"), u64(0), u128s("0x123456789012345678901234")},

		// 128-bit 'cmp == 0' shortcut branch:
		{u128s("0x123456789012345678901234"), u128s("0x123456789012345678901234"), u64(1), u64(0)},

		// These test cases were found by the fuzzer and exposed a bug in the 128-bit divisor
		// branch of divmod128by128:
		// 3289699161974853443944280720275488 / 9261249991223143249760: u128(48100516172305203) != big(355211139435)
		// 51044189592896282646990963682604803 / 15356086376658915618524: u128(16290274193854465) != big(3324036368438)
		// 555579170280843546177 / 21475569273528505412: u128(12) != big(25)
	} {
		t.Run(fmt.Sprintf("%d/%s÷%s=%s,%s", idx, tc.u, tc.by, tc.q, tc.r), func(t *testing.T) {
			tt := assert.WrapTB(t)
			q, r := tc.u.QuoRem(tc.by)
			tt.MustEqual(tc.q.String(), q.String())
			tt.MustEqual(tc.r.String(), r.String())

			uBig := tc.u.AsBigInt()
			byBig := tc.by.AsBigInt()

			qBig, rBig := new(big.Int).Set(uBig), new(big.Int).Set(uBig)
			qBig = qBig.Quo(qBig, byBig)
			rBig = rBig.Rem(rBig, byBig)

			tt.MustEqual(tc.q.String(), qBig.String())
			tt.MustEqual(tc.r.String(), rBig.String())
		})
	}
}

func TestU128Rsh(t *testing.T) {
	for _, tc := range []struct {
		u  U128
		by uint
		r  U128
	}{
		{u: u64(2), by: 1, r: u64(1)},
		{u: u64(1), by: 2, r: u64(0)},
		{u: u128s("36893488147419103232"), by: 1, r: u128s("18446744073709551616")}, // (1<<65) - 1

		// These test cases were found by the fuzzer:
		{u: u128s("2465608830469196860151950841431"), by: 104, r: u64(0)},
		{u: u128s("377509308958315595850564"), by: 58, r: u64(1309748)},
		{u: u128s("8504691434450337657905929307096"), by: 74, r: u128s("450234615")},
		{u: u128s("11595557904603123290159404941902684322"), by: 50, r: u128s("10298924295251697538375")},
		{u: u128s("176613673099733424757078556036831904"), by: 75, r: u128s("4674925001596")},
		{u: u128s("3731491383344351937489898072501894878"), by: 112, r: u64(718)},
	} {
		t.Run(fmt.Sprintf("%s>>%d=%s", tc.u, tc.by, tc.r), func(t *testing.T) {
			tt := assert.WrapTB(t)

			ub := tc.u.AsBigInt()
			ub.Rsh(ub, tc.by).And(ub, maxBigU128)

			ru := tc.u.Rsh(tc.by)
			tt.MustEqual(tc.r.String(), ru.String(), "%s != %s; big: %s", tc.r, ru, ub)
			tt.MustEqual(ub.String(), ru.String())
		})
	}
}

func TestU128Scan(t *testing.T) {
	for idx, tc := range []struct {
		in  string
		out U128
		ok  bool
	}{
		{"1", u64(1), true},
		{"0xFF", zeroU128, false},
		{"-1", zeroU128, false},
		{"340282366920938463463374607431768211456", zeroU128, false},
	} {
		t.Run(fmt.Sprintf("%d/%s==%d", idx, tc.in, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			var result U128
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

	for idx, ws := range []string{" ", "\n", "   ", " \t "} {
		t.Run(fmt.Sprintf("scan/3/%d", idx), func(t *testing.T) {
			tt := assert.WrapTB(t)
			var a, b, c U128
			n, err := fmt.Sscan(strings.Join([]string{"123", "456", "789"}, ws), &a, &b, &c)
			tt.MustEqual(3, n)
			tt.MustOK(err)
			tt.MustEqual("123", a.String())
			tt.MustEqual("456", b.String())
			tt.MustEqual("789", c.String())
		})
	}
}

func TestSetBit(t *testing.T) {
	for i := 0; i < 128; i++ {
		t.Run(fmt.Sprintf("setcheck/%d", i), func(t *testing.T) {
			tt := assert.WrapTB(t)
			var u U128
			tt.MustEqual(uint(0), u.Bit(i))
			u = u.SetBit(i, 1)
			tt.MustEqual(uint(1), u.Bit(i))
			u = u.SetBit(i, 0)
			tt.MustEqual(uint(0), u.Bit(i))
		})
	}

	for idx, tc := range []struct {
		in  U128
		out U128
		i   int
		b   uint
	}{
		{in: u64(0), out: u128s("0b 1"), i: 0, b: 1},
		{in: u64(0), out: u128s("0b 10"), i: 1, b: 1},
		{in: u64(0), out: u128s("0x 8000000000000000"), i: 63, b: 1},
		{in: u64(0), out: u128s("0x 10000000000000000"), i: 64, b: 1},
		{in: u64(0), out: u128s("0x 20000000000000000"), i: 65, b: 1},
	} {
		t.Run(fmt.Sprintf("%d/%s/%d/%d", idx, tc.in, tc.i, tc.b), func(t *testing.T) {
			tt := assert.WrapTB(t)
			out := tc.in.SetBit(tc.i, tc.b)
			tt.MustEqual(tc.out, out)
		})
	}

	for idx, tc := range []struct {
		i int
		b uint
	}{
		{i: -1, b: 0},
		{i: 128, b: 0},
		{i: 0, b: 2},
	} {
		t.Run(fmt.Sprintf("failures/%d/%d/%d", idx, tc.i, tc.b), func(t *testing.T) {
			defer func() {
				if v := recover(); v == nil {
					t.Fatal()
				}
			}()
			var u U128
			u.SetBit(tc.i, tc.b)
		})
	}
}

func BenchmarkU128Add(b *testing.B) {
	for idx, tc := range []struct {
		a, b U128
		name string
	}{
		{zeroU128, zeroU128, "0+0"},
		{MaxU128, MaxU128, "max+max"},
		{u128s("0x7FFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF"), "lo-only"},
		{u128s("0xFFFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF"), "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.a.Add(tc.b)
			}
		})
	}
}

func BenchmarkU128Add64(b *testing.B) {
	for idx, tc := range []struct {
		a    U128
		b    uint64
		name string
	}{
		{zeroU128, 0, "0+0"},
		{MaxU128, maxUint64, "max+max"},
		{u128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.a.Add64(tc.b)
			}
		})
	}
}

func BenchmarkU128AsBigFloat(b *testing.B) {
	n := u128s("36893488147419103230")
	for i := 0; i < b.N; i++ {
		BenchBigFloatResult = n.AsBigFloat()
	}
}

func BenchmarkU128AsBigInt(b *testing.B) {
	u := U128{lo: 0xFEDCBA9876543210, hi: 0xFEDCBA9876543210}
	BenchBigIntResult = new(big.Int)

	for i := uint(0); i <= 128; i += 32 {
		v := u.Rsh(128 - i)
		b.Run(fmt.Sprintf("%x,%x", v.hi, v.lo), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBigIntResult = v.AsBigInt()
			}
		})
	}
}

func BenchmarkU128AsFloat(b *testing.B) {
	n := u128s("36893488147419103230")
	for i := 0; i < b.N; i++ {
		BenchFloatResult = n.AsFloat64()
	}
}

var benchU128CmpCases = []struct {
	a, b U128
	name string
}{
	{U128From64(maxUint64), U128From64(maxUint64), "equal64"},
	{MaxU128, MaxU128, "equal128"},
	{u128s("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), "lesshi"},
	{u128s("0xEFFFFFFFFFFFFFFF"), u128s("0xFFFFFFFFFFFFFFFF"), "lesslo"},
	{u128s("0xFFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF FFFFFFFFFFFFFFFF"), "greaterhi"},
	{u128s("0xFFFFFFFFFFFFFFFF"), u128s("0xEFFFFFFFFFFFFFFF"), "greaterlo"},
}

func BenchmarkU128Cmp(b *testing.B) {
	for _, tc := range benchU128CmpCases {
		b.Run(fmt.Sprintf("u128cmp/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchIntResult = tc.a.Cmp(tc.b)
			}
		})
	}
}

func BenchmarkU128FromBigInt(b *testing.B) {
	for _, bi := range []*big.Int{
		bigs("0"),
		bigs("0xfedcba98"),
		bigs("0xfedcba9876543210"),
		bigs("0xfedcba9876543210fedcba98"),
		bigs("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result, _ = U128FromBigInt(bi)
			}
		})
	}
}

func BenchmarkU128FromFloat(b *testing.B) {
	for _, pow := range []float64{1, 63, 64, 65, 127, 128} {
		b.Run(fmt.Sprintf("pow%d", int(pow)), func(b *testing.B) {
			f := math.Pow(2, pow)
			for i := 0; i < b.N; i++ {
				BenchU128Result, _ = U128FromFloat64(f)
			}
		})
	}
}

func BenchmarkU128GreaterThan(b *testing.B) {
	for _, tc := range benchU128CmpCases {
		b.Run(fmt.Sprintf("u128gt/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = tc.a.GreaterThan(tc.b)
			}
		})
	}
}

func BenchmarkU128GreaterOrEqualTo(b *testing.B) {
	for _, tc := range benchU128CmpCases {
		b.Run(fmt.Sprintf("u128gte/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = tc.a.GreaterOrEqualTo(tc.b)
			}
		})
	}
}

func BenchmarkU128Inc(b *testing.B) {
	for idx, tc := range []struct {
		name string
		a    U128
	}{
		{"0", zeroU128},
		{"max", MaxU128},
		{"carry", u64(maxUint64)},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.a.Inc()
			}
		})
	}
}

func BenchmarkU128IntoBigInt(b *testing.B) {
	u := U128{lo: 0xFEDCBA9876543210, hi: 0xFEDCBA9876543210}
	BenchBigIntResult = new(big.Int)

	for i := uint(0); i <= 128; i += 32 {
		v := u.Rsh(128 - i)
		b.Run(fmt.Sprintf("%x,%x", v.hi, v.lo), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v.IntoBigInt(BenchBigIntResult)
			}
		})
	}
}

func BenchmarkU128LessThan(b *testing.B) {
	for _, tc := range benchU128CmpCases {
		b.Run(fmt.Sprintf("u128lt/%s", tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchBoolResult = tc.a.LessThan(tc.b)
			}
		})
	}
}

func BenchmarkU128Lsh(b *testing.B) {
	for _, tc := range []struct {
		in U128
		sh uint
	}{
		{u64(maxUint64), 1},
		{u64(maxUint64), 2},
		{u64(maxUint64), 8},
		{u64(maxUint64), 64},
		{u64(maxUint64), 126},
		{u64(maxUint64), 127},
		{u64(maxUint64), 128},
		{MaxU128, 1},
		{MaxU128, 2},
		{MaxU128, 8},
		{MaxU128, 64},
		{MaxU128, 126},
		{MaxU128, 127},
		{MaxU128, 128},
	} {
		b.Run(fmt.Sprintf("%s>>%d", tc.in, tc.sh), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.in.Lsh(tc.sh)
			}
		})
	}
}

func BenchmarkU128Mul(b *testing.B) {
	u := U128From64(maxUint64)
	for i := 0; i < b.N; i++ {
		BenchU128Result = u.Mul(u)
	}
}

func BenchmarkU128Mul64(b *testing.B) {
	u := U128From64(maxUint64)
	lim := uint64(b.N)
	for i := uint64(0); i < lim; i++ {
		BenchU128Result = u.Mul64(i)
	}
}

var benchQuoCases = []struct {
	name     string
	dividend U128
	divisor  U128
}{
	// 128-bit divide by 1 branch:
	{"128bit/1", MaxU128, u64(1)},

	// 128-bit divide by power of 2 branch:
	{"128bit/pow2", MaxU128, u64(2)},

	// 64-bit divide by 1 branch:
	{"64-bit/1", u64(maxUint64), u64(1)},

	// 128-bit divisor lz+tz > threshold branch:
	{"128bit/lz+tz>thresh", u128s("0x123456789012345678901234567890"), u128s("0xFF0000000000000000000")},

	// 128-bit divisor lz+tz <= threshold branch:
	{"128bit/lz+tz<=thresh", u128s("0x12345678901234567890123456789012"), u128s("0x10000000000000000000000000000001")},

	// 128-bit 'cmp == 0' shortcut branch:
	{"128bit/samesies", u128s("0x1234567890123456"), u128s("0x1234567890123456")},
}

func BenchmarkU128Quo(b *testing.B) {
	for idx, bc := range benchQuoCases {
		b.Run(fmt.Sprintf("%d/%s", idx, bc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = bc.dividend.Quo(bc.divisor)
			}
		})
	}
}

func BenchmarkU128QuoRem(b *testing.B) {
	for idx, bc := range benchQuoCases {
		b.Run(fmt.Sprintf("%d/%s", idx, bc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result, _ = bc.dividend.QuoRem(bc.divisor)
			}
		})
	}
}

func BenchmarkU128QuoRemTZ(b *testing.B) {
	type tc struct {
		zeros  int
		useRem int
		da, db U128
	}
	var cases []tc

	// If there's a big jump in speed from one of these cases to the next, it
	// could be indicative that the algorithm selection spill point
	// (divAlgoLeading0Spill) needs to change.
	//
	// This could probably be automated a little better, and the result is also
	// likely platform and possibly CPU specific.
	for zeros := 0; zeros < 31; zeros++ {
		for useRem := 0; useRem < 2; useRem++ {
			bs := "0b"
			for j := 0; j < 128; j++ {
				if j >= zeros {
					bs += "1"
				} else {
					bs += "0"
				}
			}

			da := u128s("0x98765432109876543210987654321098")
			db := u128s(bs)
			cases = append(cases, tc{
				zeros:  zeros,
				useRem: useRem,
				da:     da,
				db:     db,
			})
		}
	}

	for _, tc := range cases {
		b.Run(fmt.Sprintf("z=%d/rem=%v", tc.zeros, tc.useRem == 1), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if tc.useRem == 1 {
					BenchU128Result, _ = tc.da.QuoRem(tc.db)
				} else {
					BenchU128Result = tc.da.Quo(tc.db)
				}
			}
		})
	}
}

func BenchmarkU128QuoRem64(b *testing.B) {
	// FIXME: benchmark numbers of various sizes
	u, v := u64(1234), uint64(56)
	for i := 0; i < b.N; i++ {
		BenchU128Result, _ = u.QuoRem64(v)
	}
}

func BenchmarkU128QuoRem64TZ(b *testing.B) {
	type tc struct {
		zeros  int
		useRem int
		da     U128
		db     uint64
	}
	var cases []tc

	for zeros := 0; zeros < 31; zeros++ {
		for useRem := 0; useRem < 2; useRem++ {
			bs := "0b"
			for j := 0; j < 64; j++ {
				if j >= zeros {
					bs += "1"
				} else {
					bs += "0"
				}
			}

			da := u128s("0x98765432109876543210987654321098")
			db128 := u128s(bs)
			if !db128.IsUint64() {
				panic("oh dear!")
			}
			db := db128.AsUint64()

			cases = append(cases, tc{
				zeros:  zeros,
				useRem: useRem,
				da:     da,
				db:     db,
			})
		}
	}

	for _, tc := range cases {
		b.Run(fmt.Sprintf("z=%d/rem=%v", tc.zeros, tc.useRem == 1), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if tc.useRem == 1 {
					BenchU128Result, _ = tc.da.QuoRem64(tc.db)
				} else {
					BenchU128Result = tc.da.Quo64(tc.db)
				}
			}
		})
	}
}

func BenchmarkU128Rem64(b *testing.B) {
	b.Run("fast", func(b *testing.B) {
		u, v := U128{1, 0}, uint64(56) // u.hi < v
		for i := 0; i < b.N; i++ {
			BenchU128Result = u.Rem64(v)
		}
	})

	b.Run("slow", func(b *testing.B) {
		u, v := U128{100, 0}, uint64(56) // u.hi >= v
		for i := 0; i < b.N; i++ {
			BenchU128Result = u.Rem64(v)
		}
	})
}

func BenchmarkU128String(b *testing.B) {
	for _, bi := range []U128{
		u128s("0"),
		u128s("0xfedcba98"),
		u128s("0xfedcba9876543210"),
		u128s("0xfedcba9876543210fedcba98"),
		u128s("0xfedcba9876543210fedcba9876543210"),
	} {
		b.Run(fmt.Sprintf("%x", bi.AsBigInt()), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchStringResult = bi.String()
			}
		})
	}
}

func BenchmarkU128Sub(b *testing.B) {
	for idx, tc := range []struct {
		name string
		a, b U128
	}{
		{"0+0", zeroU128, zeroU128},
		{"0-max", zeroU128, MaxU128},
		{"lo-only", u128s("0x7FFFFFFFFFFFFFFF"), u128s("0x7FFFFFFFFFFFFFFF")},
		{"carry", MaxU128, u64(maxUint64).Add64(1)},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.a.Sub(tc.b)
			}
		})
	}
}

func BenchmarkU128Sub64(b *testing.B) {
	for idx, tc := range []struct {
		a    U128
		b    uint64
		name string
	}{
		{zeroU128, 0, "0+0"},
		{MaxU128, maxUint64, "max+max"},
		{u128s("0xFFFFFFFFFFFFFFFF"), 1, "carry"},
	} {
		b.Run(fmt.Sprintf("%d/%s", idx, tc.name), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.a.Sub64(tc.b)
			}
		})
	}
}
