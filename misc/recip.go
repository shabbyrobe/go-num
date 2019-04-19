package main

import (
	"fmt"
	"log"
	"math/bits"
	"os"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	num "github.com/shabbyrobe/go-num"
)

// This is a cheap-and-nasty experiment to try to understand the algorithm
// used by compilers to do fast integer division using multiplication. I wanted
// to use it for certain fast division operations in U128 but it didn't really
// end up helping much. The routine I was trying to use it for is pasted down
// the bottom, along with the benchmark code that showed me I had wasted my
// time.
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

	numType := os.Args[1]
	numerStr, denomStr := os.Args[2], os.Args[3]

	if numType == "u64" {
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

	} else if numType == "u128" {
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

	} else if numType == "i128" {
		numer, _, err := num.I128FromString(numerStr)
		if err != nil {
			return err
		}

		denom, _, err := num.I128FromString(denomStr)
		if err != nil {
			return err
		}

		divider := divFindMulI128(denom)
		result := divMulI128(numer, divider)

		fmt.Printf("%d / %d == %d\n", numer, denom, result)
		fmt.Printf("recip:%#x shift:%d 65bit:%v\n", divider.magic, divider.more, divider.add)

		nhi, nlo := divider.magic.Raw()
		fmt.Printf("recip: I128{hi: %#x, lo: %#x}\n", nhi, nlo)

	} else {
		return fmt.Errorf("numtype must be u64, u128, i128")
	}

	return nil
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

func divFindMulU128(denom num.U128) (recip num.U128, shift uint, add bool) {
	var floorLog2Denom = uint(127 - denom.LeadingZeros())

	if denom.And(denom.Sub64(1)).IsZero() {
		add = true
		shift = floorLog2Denom - 1

	} else {
		var proposedM, rem = num.U256From64(1).
			Lsh(floorLog2Denom).
			Lsh(128). // move into the hi 128 bits of a 256-bit number
			QuoRem(num.U256From128(denom))

		if rem.Cmp(num.U256From64(0)) <= 0 {
			panic(fmt.Errorf("remainder should not be less than 0, found %s", rem))
		}
		if rem.Cmp(num.U256From128(denom)) >= 0 {
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
		if e.LessThan(num.U128From64(1).Lsh(floorLog2Denom)) {
			shift = floorLog2Denom
		} else {
			// 0.65 bit version:
			proposedM128 = proposedM128.Add(proposedM128)
			twiceRem := rem128.Add(rem128)
			if twiceRem.GreaterOrEqualTo(denom) || twiceRem.LessThan(rem128) {
				proposedM128.Add(num.U128From64(1))
			}
			shift = floorLog2Denom
			add = true
		}

		recip = proposedM128.Add(num.U128From64(1))
	}

	return
}

type i128Divider struct {
	magic num.I128
	more  uint
	add   bool
	neg   bool
}

func divMulI128(numer num.I128, denom i128Divider) num.I128 {
	// q, _ := mul128to256(numer, recip)

	// if add {
	//     return numer.Sub(q).Rsh(1).Add(q).Rsh(shift)
	// } else {
	//     return q.Rsh(shift)
	// }

	unumer := numer.AsU128()
	absNumer := numer.AbsU128()

	spew.Dump(denom)

	if denom.magic.IsZero() {
		mask := num.U128From64(1).Lsh(denom.more).Sub64(1)
		uq := unumer.Add(unumer.Rsh(127).And(mask)).Rsh(denom.more)
		q := uq.AsI128()
		if denom.neg {
			q = q.Neg()
		}
		return q

	} else {
		uq, _ := mul128to256(absNumer, denom.magic.AsU128())
		if denom.add {
			fmt.Println(1, uq)
			uq = uq.Add(unumer.Xor64(1).Sub64(1))
			fmt.Println(2, uq)
		}
		uq = uq.Rsh(denom.more)
		q := uq.AsI128()
		if q.Sign() < 0 {
			q = q.Add64(1)
		}
		return q

		// uint32_t uq = (uint32_t)libdivide__mullhi_s32(denom->magic, numer);
		// if (more & LIBDIVIDE_ADD_MARKER) {
		//     // must be arithmetic shift and then sign extend
		//     int32_t sign = (int8_t)more >> 7;
		//     // q += (more < 0 ? -numer : numer), casts to avoid UB
		//     uq += ((uint32_t)numer ^ sign) - sign;
		// }
		// int32_t q = (int32_t)uq;
		// q >>= more & LIBDIVIDE_32_SHIFT_MASK;
		// q += (q < 0);
		// return q;
	}
	/*
			  uint8_t more = denom->more;
		      if (more & LIBDIVIDE_S32_SHIFT_PATH) {
		          uint32_t sign = (int8_t)more >> 7;
		          uint8_t shifter = more & LIBDIVIDE_32_SHIFT_MASK;
		          uint32_t uq = (uint32_t)(numer + ((numer >> 31) & ((1U << shifter) - 1)));
		          int32_t q = (int32_t)uq;
		          q = q >> shifter;
		          q = (q ^ sign) - sign;
		          return q;
		      } else {
		          uint32_t uq = (uint32_t)libdivide__mullhi_s32(denom->magic, numer);
		          if (more & LIBDIVIDE_ADD_MARKER) {
		              // must be arithmetic shift and then sign extend
		              int32_t sign = (int8_t)more >> 7;
		              // q += (more < 0 ? -numer : numer), casts to avoid UB
		              uq += ((uint32_t)numer ^ sign) - sign;
		          }
		          int32_t q = (int32_t)uq;
		          q >>= more & LIBDIVIDE_32_SHIFT_MASK;
		          q += (q < 0);
		          return q;
		      }
	*/
}

func divFindMulI128(denom num.I128) (divider i128Divider) {
	absDenom := denom.Abs().AsU128()
	floorLog2Denom := uint(127 - absDenom.LeadingZeros())

	if absDenom.And(absDenom.Sub64(1)).IsZero() {
		divider.neg = denom.Sign() < 0
		divider.more = floorLog2Denom

	} else {
		if floorLog2Denom < 1 {
			panic("unexpected more")
		}

		var proposedM, rem = num.U256From64(1).
			Lsh(floorLog2Denom - 1).
			Lsh(128). // move into the hi 128 bits of a 256-bit number
			QuoRem(num.U256From128(absDenom))

		if rem.Cmp(num.U256From64(0)) <= 0 {
			panic(fmt.Errorf("remainder should not be less than 0, found %s", rem))
		}
		if rem.Cmp(num.U256From128(absDenom)) >= 0 {
			panic("unexpected rem")
		}

		if !proposedM.IsU128() {
			panic(fmt.Errorf("proposedM overflows 128 bit, found %s (%x)", proposedM, proposedM))
		}
		if !rem.IsU128() {
			panic(fmt.Errorf("remainder overflows 128 bit, found %s", rem))
		}
		var proposedM128, rem128 = proposedM.AsU128(), rem.AsU128()

		var e = absDenom.Sub(rem128)

		// We are going to start with a power of floor_log_2_d - 1.
		// This works if works if e < 2**floor_log_2_d.
		if e.LessThan(num.U128From64(1).Lsh(floorLog2Denom)) {
			divider.more = floorLog2Denom - 1

		} else {
			// We need to go one higher. This should not make proposed_m
			// overflow, but it will make it negative when interpreted as an
			// int32_t.
			// 0.65 bit version:
			proposedM128 = proposedM128.Add(proposedM128)
			twiceRem := rem128.Add(rem128)

			if twiceRem.GreaterOrEqualTo(absDenom) || twiceRem.LessThan(rem128) {
				proposedM128.Add(num.U128From64(1))
			}
			divider.more = floorLog2Denom
			divider.add = true
		}

		divider.magic = proposedM128.Add(num.U128From64(1)).AsI128()

		if denom.Sign() < 0 {
			divider.neg = true
			divider.magic = divider.magic.Neg()
		}
	}

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
