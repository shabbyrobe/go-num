package main

import (
	"fmt"
	"log"
	"math/big"
	"math/bits"
	"os"
	"strconv"

	num "github.com/shabbyrobe/go-num"
)

// This is a cheap-and-nasty experiment to try to understand the algorithm
// used by compilers to do fast integer division using multiplication. I wanted
// to use it for certain fast division operations in U128 but it didn't really
// end up helping much. The routine I was trying to use it for is pasted down
// the bottom, along with the benchmark code that showed me I had wasted my
// time.
//
// It contains the first half of an unpolished, untested U256 implementation,
// which could be spun into an implementation in the library proper at some
// point.
//
// It has been kept with the repository just in case it comes in handy, but I
// wouldn't recommend using it for anything serious.

const usage = `Reciprocal finder

Usage: <bits> <numer> <denom>`

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 3 {
		fmt.Println(usage)
		return fmt.Errorf("missing args")
	}

	bitv := os.Args[1]
	numerStr, denomStr := os.Args[2], os.Args[3]

	if bitv == "64" {
		numer, err := strconv.ParseUint(numerStr, 10, 64)
		if err != nil {
			return err
		}

		denom, err := strconv.ParseUint(denomStr, 10, 64)
		if err != nil {
			return err
		}

		recip, shift, add := divFindMulU64(denom)
		result := divMulU64(numer, recip, shift, add)

		fmt.Printf("%d / %d == %d\n", numer, denom, result)
		fmt.Printf("recip:%#x shift:%d 65bit:%v\n", recip, shift, add)

	} else if bitv == "128" {
		numer, _, err := num.U128FromString(numerStr)
		if err != nil {
			return err
		}

		denom, _, err := num.U128FromString(denomStr)
		if err != nil {
			return err
		}

		recip, shift, add := divFindMulU128(denom)
		result := divMulU128(numer, recip, shift, add)

		fmt.Printf("%d / %d == %d\n", numer, denom, result)
		fmt.Printf("recip:%#x shift:%d 65bit:%v\n", recip, shift, add)

		nhi, nlo := recip.Raw()
		fmt.Printf("recip: U128{hi: %#x, lo: %#x}\n", nhi, nlo)

	} else {
		return fmt.Errorf("bits must be 64 or 128")
	}

	return nil
}

func divFindMulU128(denom num.U128) (recip num.U128, shift uint, add bool) {
	var floorLog2d = uint(127 - denom.LeadingZeros())
	var proposedM, rem = U256From64(1).
		Lsh(floorLog2d).
		Lsh(128). // move into the hi 128 bits of a 256-bit number
		QuoRem(U256From128(denom))

	if rem.Cmp(U256From64(0)) <= 0 {
		panic(fmt.Errorf("remainder should not be less than 0, found %s", rem))
	}
	if rem.Cmp(U256From128(denom)) >= 0 {
		panic("unexpected rem")
	}

	if !proposedM.IsU128() {
		panic(fmt.Errorf("proposedM overflows 128 bit, found %s (%x)", proposedM, proposedM))
	}
	if !rem.IsU128() {
		panic(fmt.Errorf("remainder overflows 128 bit, found %s", rem))
	}
	var proposedM128, rem128 = proposedM.AsU128(), rem.AsU128()

	var e = denom.Sub(rem128)
	if e.LessThan(num.U128From64(1).Lsh(floorLog2d)) {
		shift = floorLog2d
	} else {
		// 0.65 bit version:
		proposedM128 = proposedM128.Add(proposedM128)
		twiceRem := rem128.Add(rem128)
		if twiceRem.GreaterOrEqualTo(denom) || twiceRem.LessThan(rem128) {
			proposedM128.Add(num.U128From64(1))
		}
		shift = floorLog2d
		add = true
	}

	recip = proposedM128.Add(num.U128From64(1))
	return
}

func divFindMulU64(denom uint64) (recip uint64, shift uint, add bool) {
	var floorLog2d = uint(63 - bits.LeadingZeros64(denom))
	var proposedM, rem = num.U128From64(1).
		Lsh(floorLog2d).
		Lsh(64). // move into the hi bits of a 128-bit number
		QuoRem(num.U128From64(denom))

	if !proposedM.IsUint64() {
		panic("proposedM overflows 64 bit")
	}
	if !rem.IsUint64() {
		panic("remainder overflows 64 bit")
	}
	var proposedM64, rem64 = proposedM.AsUint64(), rem.AsUint64()

	var e = denom - rem64
	if e < 1<<floorLog2d {
		shift = floorLog2d
	} else {
		// 0.65 bit version:
		proposedM64 += proposedM64
		twiceRem := rem64 + rem64
		if twiceRem >= denom || twiceRem < rem64 {
			proposedM64 += 1
		}
		shift = floorLog2d
		add = true
	}

	recip = 1 + proposedM64
	return
}

func divMulU128(numer, recip num.U128, shift uint, add bool) num.U128 {
	q, _ := mul128to256(numer, recip)

	if add {
		return numer.Sub(q).Rsh(1).Add(q).Rsh(shift)
	} else {
		return q.Rsh(shift)
	}
}

func divMulU64(numer, recip uint64, shift uint, add bool) uint64 {
	q := num.U128From64(numer).
		Mul(num.U128From64(recip)).
		Rsh(64).
		AsUint64()

	if add {
		t := ((numer - q) >> 1) + q
		return t >> shift
	} else {
		return q >> shift
	}
}

type U256 struct {
	hi, hm, lm, lo uint64
}

func U256From128(in num.U128) U256 {
	hi, lo := in.Raw()
	return U256{lm: hi, lo: lo}
}

func U256From64(in uint64) U256 {
	return U256{lo: in}
}

func (u U256) And(v U256) (out U256) {
	out.hi = u.hi & v.hi
	out.hm = u.hm & v.hm
	out.lm = u.lm & v.lm
	out.lo = u.lo & v.lo
	return out
}

func (u U256) IntoBigInt(b *big.Int) {
	const intSize = 32 << (^uint(0) >> 63)

	switch intSize {
	case 64:
		bits := b.Bits()
		ln := len(bits)
		if len(bits) < 4 {
			bits = append(bits, make([]big.Word, 4-ln)...)
		}
		bits = bits[:4]
		bits[0] = big.Word(u.lo)
		bits[1] = big.Word(u.lm)
		bits[2] = big.Word(u.hm)
		bits[3] = big.Word(u.hi)
		b.SetBits(bits)

	default:
		panic("not implemented")
	}
}

func (u U256) AsBigInt() (b *big.Int) {
	var v big.Int
	u.IntoBigInt(&v)
	return &v
}

func (u U256) Cmp(n U256) int {
	if u.hi > n.hi {
		return 1
	} else if u.hi < n.hi {
		return -1
	} else if u.hm > n.hm {
		return 1
	} else if u.hm < n.hm {
		return -1
	} else if u.lm > n.lm {
		return 1
	} else if u.lm < n.lm {
		return -1
	} else if u.lo > n.lo {
		return 1
	} else if u.lo < n.lo {
		return -1
	}
	return 0
}

func (u U256) Dec() (out U256) {
	out = u
	out.lo = u.lo - 1
	if u.lo < out.lo {
		out.lm--
	}
	if u.lm < out.lm {
		out.hm--
	}
	if u.hm < out.hm {
		out.hi--
	}
	return out
}

func (u U256) Format(s fmt.State, c rune) {
	// FIXME: This is good enough for now, but not forever.
	u.AsBigInt().Format(s, c)
}

func (u U256) LeadingZeros() uint {
	if u.hi != 0 {
		return uint(bits.LeadingZeros64(u.hi))
	} else if u.hm != 0 {
		return uint(bits.LeadingZeros64(u.hm)) + 64
	} else if u.lm != 0 {
		return uint(bits.LeadingZeros64(u.lm)) + 128
	} else if u.lo != 0 {
		return uint(bits.LeadingZeros64(u.lo)) + 192
	}
	return 256
}

func (u U256) Lsh(n uint) (v U256) {
	if n == 0 {
		return u

	} else if n < 64 {
		return U256{
			hi: (u.hi << n) | (u.hm >> (64 - n)),
			hm: (u.hm << n) | (u.lm >> (64 - n)),
			lm: (u.lm << n) | (u.lo >> (64 - n)),
			lo: u.lo << n,
		}

	} else if n == 64 {
		return U256{hi: u.hm, hm: u.lm, lm: u.lo}

	} else if n < 128 {
		n -= 64
		return U256{
			hi: (u.hm << n) | (u.lm >> (64 - n)),
			hm: (u.lm << n) | (u.lo >> (64 - n)),
			lm: u.lo << n,
		}

	} else if n == 128 {
		return U256{hi: u.lm, hm: u.lo}

	} else if n < 192 {
		n -= 128
		return U256{
			hi: (u.lm << n) | (u.lo >> (64 - n)),
			hm: u.lo << n,
		}

	} else if n == 192 {
		return U256{hi: u.lo}
	} else if n < 256 {
		return U256{hi: u.lo << (n - 192)}
	} else {
		return U256{}
	}
}

func (u U256) QuoRem(by U256) (q, r U256) {
	if by.hi == 0 && by.hm == 0 && by.lm == 0 && by.lo == 0 {
		panic("u256: division by zero")
	}

	byLeading0 := by.LeadingZeros()
	if byLeading0 == 255 {
		return u, r
	}

	byTrailing0 := by.TrailingZeros()
	if (byLeading0 + byTrailing0) == 255 {
		q = u.Rsh(byTrailing0)
		by = by.Dec()
		r = by.And(u)
		return
	}

	if cmp := u.Cmp(by); cmp < 0 {
		return q, u // it's 100% remainder

	} else if cmp == 0 {
		q.lo = 1 // dividend and divisor are the same
		return q, r
	}

	uLeading0 := u.LeadingZeros()
	return quorem256bin(u, by, uLeading0, byLeading0)
}

func (u U256) Rsh(n uint) (v U256) {
	if n == 0 {
		return u

	} else if n < 64 {
		return U256{
			hi: u.hi >> n,
			hm: (u.hm >> n) | (u.hi << (64 - n)),
			lm: (u.lm >> n) | (u.hm << (64 - n)),
			lo: (u.lo >> n) | (u.lm << (64 - n)),
		}

	} else if n == 64 {
		return U256{hm: u.hi, lm: u.hm, lo: u.lm}

	} else if n < 128 {
		n -= 64
		return U256{
			hm: u.hi >> n,
			lm: (u.hm >> n) | (u.hi << (64 - n)),
			lo: (u.lm >> n) | (u.hm << (64 - n)),
		}

	} else if n == 128 {
		return U256{lm: u.hi, lo: u.hm}

	} else if n < 192 {
		n -= 128
		return U256{
			lm: u.hi >> n,
			lo: (u.hm >> n) | (u.hi << (64 - n)),
		}

	} else if n == 192 {
		return U256{lo: u.hi}

	} else if n < 256 {
		return U256{lo: u.hi >> (n - 192)}

	} else {
		return U256{}
	}
}

func (u U256) String() string {
	var zeroU256 U256
	if u == zeroU256 {
		return "0"
	}
	if u.hi == 0 && u.hm == 0 && u.lm == 0 {
		return strconv.FormatUint(u.lo, 10)
	}
	v := u.AsBigInt()
	return v.String()
}

func (u U256) Sub(n U256) (v U256) {
	v.lo = u.lo - n.lo
	if u.lo < v.lo {
		u.lm--
	}
	v.lm = u.lm - n.lm
	if u.lm < v.lm {
		u.hm--
	}
	v.hm = u.hm - n.hm
	if u.hm < v.hm {
		u.hi--
	}
	v.hi = u.hi - n.hi
	return v
}

func (u U256) TrailingZeros() uint {
	if u.lo != 0 {
		return uint(bits.TrailingZeros64(u.lo))
	} else if u.lm != 0 {
		return uint(bits.LeadingZeros64(u.lm)) + 64
	} else if u.hm != 0 {
		return uint(bits.LeadingZeros64(u.hm)) + 128
	} else if u.hi != 0 {
		return uint(bits.LeadingZeros64(u.hi)) + 192
	}
	return 256
}

// IsUint64 truncates the U256 to fit in a uint64. Values outside the range
// will over/underflow. See IsUint64() if you want to check before you convert.
func (u U256) AsUint64() uint64 { return u.lo }

// IsUint64 reports whether u can be represented as a uint64.
func (u U256) IsUint64() bool { return u.hi == 0 && u.hm == 0 && u.lm == 0 }

func (u U256) AsU128() num.U128 { return num.U128FromRaw(u.lm, u.lo) }

func (u U256) IsU128() bool { return u.hi == 0 && u.hm == 0 }

func quorem256bin(u, by U256, uLeading0, byLeading0 uint) (q, r U256) {
	shift := int(byLeading0 - uLeading0)
	by = by.Lsh(uint(shift))

	for {
		q = q.Lsh(1)

		if u.Cmp(by) >= 0 {
			u = u.Sub(by)
			q.lo |= 1
		}

		by = by.Rsh(1)

		if shift <= 0 {
			break
		}
		shift--
	}

	r = u
	return q, r
}

func mul128to256(n, by num.U128) (hi, lo num.U128) {
	// Lot of gymnastics in here because U128 doesn't expose lo and hi:

	nHi, nLo := n.Raw()
	byHi, byLo := by.Raw()

	hiHi, hiLo := num.U128From64(nHi).Mul(num.U128From64(byHi)).Raw()
	loHi, loLo := num.U128From64(nLo).Mul(num.U128From64(byLo)).Raw()

	tLo, tHi := num.U128From64(nHi).Mul(num.U128From64(byLo)).Raw()
	loHi += tLo

	if loHi < tLo { // if lo.Hi overflowed
		hiHi, hiLo = num.U128FromRaw(hiHi, hiLo).Inc().Raw()
	}

	hiLo += tHi
	if hiLo < tHi { // if hi.Lo overflowed
		hiHi++
	}

	tHi, tLo = num.U128From64(nLo).Mul(num.U128From64(byHi)).Raw()
	loHi += tLo
	if loHi < tLo { // if L.Hi overflowed
		hiHi, hiLo = num.U128FromRaw(hiHi, hiLo).Inc().Raw()
	}

	hiLo += tHi
	if hiLo < tHi { // if H.Lo overflowed
		hiHi++
	}

	return num.U128FromRaw(hiHi, hiLo), num.U128FromRaw(loHi, loLo)
}

/*
func (u U128) DivPow10(pow uint) U128 {
	switch pow {
	case 0:
		panic("divide by 0")
	case 1: // 10
		q, _ := mul128to256(u.hi, u.lo, 0xcccccccccccccccc, 0xcccccccccccccccd)
		return q.Rsh(3)
	case 2: // 100
		q, _ := mul128to256(u.hi, u.lo, 0xa3d70a3d70a3d70a, 0x3d70a3d70a3d70a4)
		return q.Rsh(6)
	case 3: // 1,000
		q, _ := mul128to256(u.hi, u.lo, 0x624dd2f1a9fbe76, 0xc8b4395810624dd3)
		return u.Sub(q).Rsh(1).Add(q).Rsh(9)
	case 4: // 10,000
		q, _ := mul128to256(u.hi, u.lo, 0xd1b71758e219652b, 0xd3c36113404ea4a9)
		return q.Rsh(13)
	case 5: // 100,000
		q, _ := mul128to256(u.hi, u.lo, 0xa7c5ac471b478423, 0xfcf80dc33721d54)
		return q.Rsh(16)
	case 6: // 1,000,000
		q, _ := mul128to256(u.hi, u.lo, 0x8637bd05af6c69b5, 0xa63f9a49c2c1b110)
		return q.Rsh(19)
	case 7: // 10,000,000
		q, _ := mul128to256(u.hi, u.lo, 0xd6bf94d5e57a42bc, 0x3d32907604691b4d)
		return q.Rsh(23)
	case 8: // 100,000,000
		q, _ := mul128to256(u.hi, u.lo, 0x5798ee2308c39df9, 0xfb841a566d74f87b)
		return u.Sub(q).Rsh(1).Add(q).Rsh(26)
	case 9: // 1,000,000,000
		q, _ := mul128to256(u.hi, u.lo, 0x89705f4136b4a597, 0x31680a88f8953031)
		return q.Rsh(29)
	case 10: // 10,000,000,000
		q, _ := mul128to256(u.hi, u.lo, 0xdbe6fecebdedd5be, 0xb573440e5a884d1c)
		return q.Rsh(33)
	case 11: // 100,000,000,000
		q, _ := mul128to256(u.hi, u.lo, 0xafebff0bcb24aafe, 0xf78f69a51539d749)
		return q.Rsh(36)
	case 12: // 1,000,000,000,000
		q, _ := mul128to256(u.hi, u.lo, 0x8cbccc096f5088cb, 0xf93f87b7442e45d4)
		return q.Rsh(39)
	case 13: // 10,000,000,000,000
		q, _ := mul128to256(u.hi, u.lo, 0xe12e13424bb40e13, 0x2865a5f206b06fba)
		return q.Rsh(43)

	default: // TODO: 39 decimal digits in MaxU128
		ten := U128From64(10)
		div := ten
		for i := 1; i < i; i++ {
			div = div.Mul(ten)
		}
		return u.Quo(div)
	}
}

func TestU128DivPow10(t *testing.T) {
	for idx, tc := range []struct {
		u   U128
		pow uint
		out U128
	}{
		{u64(1000), 2, u64(10)},
		{u64(9999), 2, u64(99)},
		{u64(99999), 3, u64(99)},
		{u128s("340282366920938463463374607431768211455"), 10, u128s("34028236692093846346337460743")},
	} {
		t.Run(fmt.Sprintf("%d/%d÷(10^%d)=%d", idx, tc.u, tc.pow, tc.out), func(t *testing.T) {
			tt := assert.WrapTB(t)
			tt.MustAssert(tc.out.Equal(tc.u.DivPow10(tc.pow)))
		})
	}
}

func BenchmarkDivPow10(b *testing.B) {
	for idx, tc := range []struct {
		u   U128
		pow uint
	}{
		{u64(9999), 2},
		{u64(99999999), 3},
		{u128s("340282366920938463463374607431768211455"), 10},
	} {
		b.Run(fmt.Sprintf("%d/%d÷(10^%d)", idx, tc.u, tc.pow), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.u.DivPow10(tc.pow)
			}
		})
	}
}

func BenchmarkDivPow10UsingQuo(b *testing.B) {
	for idx, tc := range []struct {
		u   U128
		pow uint
	}{
		{u64(9999), 2},
		{u64(99999999), 3},
		{u128s("340282366920938463463374607431768211455"), 10},
	} {
		ten := U128From64(10)
		div := ten
		for i := uint(1); i < tc.pow; i++ {
			div = div.Mul(ten)
		}

		b.Run(fmt.Sprintf("%d/%d÷%d", idx, tc.u, div), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BenchU128Result = tc.u.Quo(div)
			}
		})
	}
}
*/
