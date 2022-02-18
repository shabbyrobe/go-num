package num

import (
	"fmt"
	"math/big"
	"math/bits"
	"strconv"
)

type U128 struct {
	hi, lo uint64
}

// U128FromRaw is the complement to U128.Raw(); it creates an U128 from two
// uint64s representing the hi and lo bits.
func U128FromRaw(hi, lo uint64) U128 { return U128{hi: hi, lo: lo} }

func U128From64(v uint64) U128 { return U128{lo: v} }
func U128From32(v uint32) U128 { return U128{lo: uint64(v)} }
func U128From16(v uint16) U128 { return U128{lo: uint64(v)} }
func U128From8(v uint8) U128   { return U128{lo: uint64(v)} }
func U128FromUint(v uint) U128 { return U128{lo: uint64(v)} }

// U128FromI64 creates a U128 from an int64 if the conversion is possible, and
// sets inRange to false if not.
func U128FromI64(v int64) (out U128, inRange bool) {
	if v < 0 {
		return zeroU128, false
	}
	return U128{lo: uint64(v)}, true
}

func MustU128FromI64(v int64) (out U128) {
	out, inRange := U128FromI64(v)
	if !inRange {
		panic(fmt.Errorf("num: int64 %d was not in valid U128 range", v))
	}
	return out
}

// U128FromString creates a U128 from a string. Overflow truncates to MaxU128
// and sets inRange to 'false'. Only decimal strings are currently supported.
func U128FromString(s string) (out U128, inRange bool, err error) {
	// This deliberately limits the scope of what we accept as input just in case
	// we decide to hand-roll our own fast decimal-only parser:
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return out, false, fmt.Errorf("num: u128 string %q invalid", s)
	}
	out, inRange = U128FromBigInt(b)
	return out, inRange, nil
}

func MustU128FromString(s string) U128 {
	out, inRange, err := U128FromString(s)
	if err != nil {
		panic(err)
	}
	if !inRange {
		panic(fmt.Errorf("num: string %q was not in valid U128 range", s))
	}
	return out
}

// U128FromBigInt creates a U128 from a big.Int. Overflow truncates to MaxU128
// and sets inRange to 'false'.
func U128FromBigInt(v *big.Int) (out U128, inRange bool) {
	if v.Sign() < 0 {
		return out, false
	}

	words := v.Bits()

	switch intSize {
	case 64:
		lw := len(words)
		switch lw {
		case 0:
			return U128{}, true
		case 1:
			return U128{lo: uint64(words[0])}, true
		case 2:
			return U128{hi: uint64(words[1]), lo: uint64(words[0])}, true
		default:
			return MaxU128, false
		}

	case 32:
		lw := len(words)
		switch lw {
		case 0:
			return U128{}, true
		case 1:
			return U128{lo: uint64(words[0])}, true
		case 2:
			return U128{lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 3:
			return U128{hi: uint64(words[2]), lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 4:
			return U128{
				hi: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		default:
			return MaxU128, false
		}

	default:
		panic("num: unsupported bit size")
	}
}

func MustU128FromBigInt(b *big.Int) U128 {
	out, inRange := U128FromBigInt(b)
	if !inRange {
		panic(fmt.Errorf("num: big.Int %d was not in valid U128 range", b))
	}
	return out
}

func U128FromFloat32(f float32) (out U128, inRange bool) {
	return U128FromFloat64(float64(f))
}

func MustU128FromFloat32(f float32) U128 {
	out, inRange := U128FromFloat32(f)
	if !inRange {
		panic(fmt.Errorf("num: float32 %f was not in valid U128 range", f))
	}
	return out
}

// U128FromFloat64 creates a U128 from a float64.
//
// Any fractional portion will be truncated towards zero.
//
// Floats outside the bounds of a U128 may be discarded or clamped and inRange
// will be set to false.
//
// NaN is treated as 0, inRange is set to false. This may change to a panic
// at some point.
func U128FromFloat64(f float64) (out U128, inRange bool) {
	// WARNING: casts from float64 to uint64 have some astonishing properties:
	// https://github.com/golang/go/issues/29463
	if f == 0 {
		return U128{}, true

	} else if f < 0 {
		return U128{}, false

	} else if f <= maxRepresentableUint64Float {
		return U128{lo: uint64(f)}, true

	} else if f <= maxRepresentableU128Float {
		lo := modpos(f, wrapUint64Float) // f is guaranteed to be > 0 here.
		return U128{hi: uint64(f / wrapUint64Float), lo: uint64(lo)}, true

	} else if f != f { // (f != f) == NaN
		return U128{}, false

	} else {
		return MaxU128, false
	}
}

func MustU128FromFloat64(f float64) U128 {
	out, inRange := U128FromFloat64(f)
	if !inRange {
		panic(fmt.Errorf("num: float64 %f was not in valid U128 range", f))
	}
	return out
}

// RandU128 generates an unsigned 128-bit random integer from an external source.
func RandU128(source RandSource) (out U128) {
	return U128{hi: source.Uint64(), lo: source.Uint64()}
}

func (u U128) IsZero() bool { return u.lo == 0 && u.hi == 0 }

// Raw returns access to the U128 as a pair of uint64s. See U128FromRaw() for
// the counterpart.
func (u U128) Raw() (hi, lo uint64) { return u.hi, u.lo }

func (u U128) String() string {
	// FIXME: This is good enough for now, but not forever.
	if u.lo == 0 && u.hi == 0 {
		return "0"
	}
	if u.hi == 0 {
		return strconv.FormatUint(u.lo, 10)
	}
	v := u.AsBigInt()
	return v.String()
}

func (u U128) Format(s fmt.State, c rune) {
	// FIXME: This is good enough for now, but not forever.
	u.AsBigInt().Format(s, c)
}

func (u *U128) Scan(state fmt.ScanState, verb rune) error {
	t, err := state.Token(true, nil)
	if err != nil {
		return err
	}
	ts := string(t)

	v, inRange, err := U128FromString(ts)
	if err != nil {
		return err
	} else if !inRange {
		return fmt.Errorf("num: u128 value %q is not in range", ts)
	}
	*u = v

	return nil
}

func (u U128) IntoBigInt(b *big.Int) {
	switch intSize {
	case 64:
		bits := b.Bits()
		ln := len(bits)
		if len(bits) < 2 {
			bits = append(bits, make([]big.Word, 2-ln)...)
		}
		bits = bits[:2]
		bits[0] = big.Word(u.lo)
		bits[1] = big.Word(u.hi)
		b.SetBits(bits)

	case 32:
		bits := b.Bits()
		ln := len(bits)
		if len(bits) < 4 {
			bits = append(bits, make([]big.Word, 4-ln)...)
		}
		bits = bits[:4]
		bits[0] = big.Word(u.lo & 0xFFFFFFFF)
		bits[1] = big.Word(u.lo >> 32)
		bits[2] = big.Word(u.hi & 0xFFFFFFFF)
		bits[3] = big.Word(u.hi >> 32)
		b.SetBits(bits)

	default:
		if u.hi > 0 {
			b.SetUint64(u.hi)
			b.Lsh(b, 64)
		}
		var lo big.Int
		lo.SetUint64(u.lo)
		b.Add(b, &lo)
	}
}

// AsBigInt returns the U128 as a big.Int. This will allocate memory. If
// performance is a concern and you are able to re-use memory, use
// U128.IntoBigInt().
func (u U128) AsBigInt() (b *big.Int) {
	var v big.Int
	u.IntoBigInt(&v)
	return &v
}

func (u U128) AsBigFloat() (b *big.Float) {
	return new(big.Float).SetInt(u.AsBigInt())
}

func (u U128) AsFloat64() float64 {
	if u.hi == 0 && u.lo == 0 {
		return 0
	} else if u.hi == 0 {
		return float64(u.lo)
	} else {
		return (float64(u.hi) * wrapUint64Float) + float64(u.lo)
	}
}

// AsI128 performs a direct cast of a U128 to an I128, which will interpret it
// as a two's complement value.
func (u U128) AsI128() I128 {
	return I128{lo: u.lo, hi: u.hi}
}

// IsI128 reports whether i can be represented in an I128.
func (u U128) IsI128() bool {
	return u.hi&signBit == 0
}

// AsUint64 truncates the U128 to fit in a uint64. Values outside the range
// will over/underflow. See IsUint64() if you want to check before you convert.
func (u U128) AsUint64() uint64 {
	return u.lo
}

// IsUint64 reports whether u can be represented as a uint64.
func (u U128) IsUint64() bool {
	return u.hi == 0
}

// MustUint64 converts i to an unsigned 64-bit integer if the conversion would succeed,
// and panics if it would not.
func (u U128) MustUint64() uint64 {
	if u.hi != 0 {
		panic(fmt.Errorf("U128 %v is not representable as a uint64", u))
	}
	return u.lo
}

func (u U128) Inc() (v U128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, 1, 0)
	v.hi = u.hi + carry
	return v
}

func (u U128) Dec() (v U128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, 1, 0)
	v.hi = u.hi - borrowed
	return v
}

func (u U128) Add(n U128) (v U128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, n.lo, 0)
	v.hi, _ = bits.Add64(u.hi, n.hi, carry)
	return v
}

func (u U128) Add64(n uint64) (v U128) {
	var carry uint64
	v.lo, carry = bits.Add64(u.lo, n, 0)
	v.hi = u.hi + carry
	return v
}

func (u U128) Sub(n U128) (v U128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, n.lo, 0)
	v.hi, _ = bits.Sub64(u.hi, n.hi, borrowed)
	return v
}

func (u U128) Sub64(n uint64) (v U128) {
	var borrowed uint64
	v.lo, borrowed = bits.Sub64(u.lo, n, 0)
	v.hi = u.hi - borrowed
	return v
}

// Cmp compares 'u' to 'n' and returns:
//
//	< 0 if u <  n
//	  0 if u == n
//	> 0 if u >  n
//
// The specific value returned by Cmp is undefined, but it is guaranteed to
// satisfy the above constraints.
//
func (u U128) Cmp(n U128) int {
	if u.hi == n.hi {
		if u.lo > n.lo {
			return 1
		} else if u.lo < n.lo {
			return -1
		}
	} else {
		if u.hi > n.hi {
			return 1
		} else if u.hi < n.hi {
			return -1
		}
	}
	return 0
}

func (u U128) Cmp64(n uint64) int {
	if u.hi > 0 || u.lo > n {
		return 1
	} else if u.lo < n {
		return -1
	}
	return 0
}

func (u U128) Equal(n U128) bool {
	return u.hi == n.hi && u.lo == n.lo
}

func (u U128) Equal64(n uint64) bool {
	return u.hi == 0 && u.lo == n
}

func (u U128) GreaterThan(n U128) bool {
	return u.hi > n.hi || (u.hi == n.hi && u.lo > n.lo)
}

func (u U128) GreaterThan64(n uint64) bool {
	return u.hi > 0 || u.lo > n
}

func (u U128) GreaterOrEqualTo(n U128) bool {
	return u.hi > n.hi || (u.hi == n.hi && u.lo >= n.lo)
}

func (u U128) GreaterOrEqualTo64(n uint64) bool {
	return u.hi > 0 || u.lo >= n
}

func (u U128) LessThan(n U128) bool {
	return u.hi < n.hi || (u.hi == n.hi && u.lo < n.lo)
}

func (u U128) LessThan64(n uint64) bool {
	return u.hi == 0 && u.lo < n
}

func (u U128) LessOrEqualTo(n U128) bool {
	return u.hi < n.hi || (u.hi == n.hi && u.lo <= n.lo)
}

func (u U128) LessOrEqualTo64(n uint64) bool {
	return u.hi == 0 && u.lo <= n
}

func (u U128) And(n U128) U128 {
	u.hi = u.hi & n.hi
	u.lo = u.lo & n.lo
	return u
}

func (u U128) And64(n uint64) U128 {
	return U128{lo: u.lo & n}
}

func (u U128) AndNot(n U128) U128 {
	u.hi = u.hi &^ n.hi
	u.lo = u.lo &^ n.lo
	return u
}

func (u U128) Not() (out U128) {
	out.hi = ^u.hi
	out.lo = ^u.lo
	return out
}

func (u U128) Or(n U128) (out U128) {
	out.hi = u.hi | n.hi
	out.lo = u.lo | n.lo
	return out
}

func (u U128) Or64(n uint64) U128 {
	u.lo = u.lo | n
	return u
}

func (u U128) Xor(v U128) U128 {
	u.hi = u.hi ^ v.hi
	u.lo = u.lo ^ v.lo
	return u
}

func (u U128) Xor64(v uint64) U128 {
	u.hi = u.hi ^ 0
	u.lo = u.lo ^ v
	return u
}

// BitLen returns the length of the absolute value of u in bits. The bit length of 0 is 0.
func (u U128) BitLen() int {
	if u.hi > 0 {
		return bits.Len64(u.hi) + 64
	}
	return bits.Len64(u.lo)
}

// OnesCount returns the number of one bits ("population count") in u.
func (u U128) OnesCount() int {
	if u.hi > 0 {
		return bits.OnesCount64(u.hi) + 64
	}
	return bits.OnesCount64(u.lo)
}

// Bit returns the value of the i'th bit of x. That is, it returns (x>>i)&1.
// The bit index i must be 0 <= i < 128
func (u U128) Bit(i int) uint {
	if i < 0 || i >= 128 {
		panic("num: bit out of range")
	}
	if i >= 64 {
		return uint((u.hi >> uint(i-64)) & 1)
	} else {
		return uint((u.lo >> uint(i)) & 1)
	}
}

// SetBit returns a U128 with u's i'th bit set to b (0 or 1).
// If b is not 0 or 1, SetBit will panic. If i < 0, SetBit will panic.
func (u U128) SetBit(i int, b uint) (out U128) {
	if i < 0 || i >= 128 {
		panic("num: bit out of range")
	}
	if b == 0 {
		if i >= 64 {
			u.hi = u.hi &^ (1 << uint(i-64))
		} else {
			u.lo = u.lo &^ (1 << uint(i))
		}
	} else if b == 1 {
		if i >= 64 {
			u.hi = u.hi | (1 << uint(i-64))
		} else {
			u.lo = u.lo | (1 << uint(i))
		}
	} else {
		panic("num: bit value not 0 or 1")
	}
	return u
}

func (u U128) Lsh(n uint) (v U128) {
	if n == 0 {
		return u
	} else if n > 64 {
		v.hi = u.lo << (n - 64)
		v.lo = 0
	} else if n < 64 {
		v.hi = (u.hi << n) | (u.lo >> (64 - n))
		v.lo = u.lo << n
	} else if n == 64 {
		v.hi = u.lo
		v.lo = 0
	}
	return v
}

func (u U128) Rsh(n uint) (v U128) {
	if n == 0 {
		return u
	} else if n > 64 {
		v.lo = u.hi >> (n - 64)
		v.hi = 0
	} else if n < 64 {
		v.lo = (u.lo >> n) | (u.hi << (64 - n))
		v.hi = u.hi >> n
	} else if n == 64 {
		v.lo = u.hi
		v.hi = 0
	}

	return v
}

func (u U128) Mul(n U128) U128 {
	hi, lo := bits.Mul64(u.lo, n.lo)
	hi += u.hi*n.lo + u.lo*n.hi
	return U128{hi, lo}
}

func (u U128) Mul64(n uint64) (dest U128) {
	dest.hi, dest.lo = bits.Mul64(u.lo, n)
	dest.hi += u.hi * n
	return dest
}

// See BenchmarkU128QuoRemTZ for the test that helps determine this magic number:
const divAlgoLeading0Spill = 16

// Quo returns the quotient x/y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Quo implements truncated division (like Go); see
// QuoRem for more details.
func (u U128) Quo(by U128) (q U128) {
	if by.lo == 0 && by.hi == 0 {
		panic("u128: division by zero")
	}

	if u.hi|by.hi == 0 {
		q.lo = u.lo / by.lo // FIXME: div/0 risk?
		return q
	}

	var byLoLeading0, byHiLeading0, byLeading0 uint
	if by.hi == 0 {
		byLoLeading0, byHiLeading0 = uint(bits.LeadingZeros64(by.lo)), 64
		byLeading0 = byLoLeading0 + 64
	} else {
		byHiLeading0 = uint(bits.LeadingZeros64(by.hi))
		byLeading0 = byHiLeading0
	}

	if byLeading0 == 127 {
		return u
	}

	byTrailing0 := by.TrailingZeros()
	if (byLeading0 + byTrailing0) == 127 {
		return u.Rsh(byTrailing0)
	}

	if cmp := u.Cmp(by); cmp < 0 {
		return q // it's 100% remainder
	} else if cmp == 0 {
		q.lo = 1 // dividend and divisor are the same
		return q
	}

	uLeading0 := u.LeadingZeros()
	if byLeading0-uLeading0 > divAlgoLeading0Spill {
		q, _ = quorem128by128(u, by, byHiLeading0, byLoLeading0)
		return q
	} else {
		return quo128bin(u, by, uLeading0, byLeading0)
	}
}

func (u U128) Quo64(by uint64) (q U128) {
	if u.hi < by {
		q.lo, _ = bits.Div64(u.hi, u.lo, by)
	} else {
		q.hi = u.hi / by
		q.lo, _ = bits.Div64(u.hi%by, u.lo, by)
	}
	return q
}

// QuoRem returns the quotient q and remainder r for y != 0. If y == 0, a
// division-by-zero run-time panic occurs.
//
// QuoRem implements T-division and modulus (like Go):
//
//	q = x/y      with the result truncated to zero
//	r = x - y*q
//
// U128 does not support big.Int.DivMod()-style Euclidean division.
//
func (u U128) QuoRem(by U128) (q, r U128) {
	if by.lo == 0 && by.hi == 0 {
		panic("u128: division by zero")
	}

	if u.hi|by.hi == 0 {
		// protected from div/0 because by.lo is guaranteed to be set if by.hi is 0:
		q.lo = u.lo / by.lo
		r.lo = u.lo % by.lo
		return q, r
	}

	var byLoLeading0, byHiLeading0, byLeading0 uint
	if by.hi == 0 {
		byLoLeading0, byHiLeading0 = uint(bits.LeadingZeros64(by.lo)), 64
		byLeading0 = byLoLeading0 + 64
	} else {
		byHiLeading0 = uint(bits.LeadingZeros64(by.hi))
		byLeading0 = byHiLeading0
	}

	if byLeading0 == 127 {
		return u, r
	}

	byTrailing0 := by.TrailingZeros()
	if (byLeading0 + byTrailing0) == 127 {
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
	if byLeading0-uLeading0 > divAlgoLeading0Spill {
		return quorem128by128(u, by, byHiLeading0, byLoLeading0)
	} else {
		return quorem128bin(u, by, uLeading0, byLeading0)
	}
}

func (u U128) QuoRem64(by uint64) (q, r U128) {
	if u.hi < by {
		q.lo, r.lo = bits.Div64(u.hi, u.lo, by)
	} else {
		q.hi, r.lo = bits.Div64(0, u.hi, by)
		q.lo, r.lo = bits.Div64(r.lo, u.lo, by)
	}
	return q, r
}

// Rem returns the remainder of x%y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Rem implements truncated modulus (like Go); see
// QuoRem for more details.
func (u U128) Rem(by U128) (r U128) {
	// FIXME: inline only the needed bits
	_, r = u.QuoRem(by)
	return r
}

func (u U128) Rem64(by uint64) (r U128) {
	// XXX: bits.Rem64 (added in 1.14) shows no noticeable improvement on my 8th-gen i7
	// (though it sounds like it isn't necessarily meant to):
	// https://github.com/golang/go/issues/28970
	// if u.hi < by {
	//     _, r.lo = bits.Rem64(u.hi, u.lo, by)
	// } else {
	//     _, r.lo = bits.Rem64(bits.Rem64(0, u.hi, by), u.lo, by)
	// }

	if u.hi < by {
		_, r.lo = bits.Div64(u.hi, u.lo, by)
	} else {
		_, r.lo = bits.Div64(0, u.hi, by)
		_, r.lo = bits.Div64(r.lo, u.lo, by)
	}
	return r
}

func (u U128) Reverse() U128 {
	return U128{hi: bits.Reverse64(u.lo), lo: bits.Reverse64(u.hi)}
}

func (u U128) ReverseBytes() U128 {
	return U128{hi: bits.ReverseBytes64(u.lo), lo: bits.ReverseBytes64(u.hi)}
}

func (u U128) RotateLeft(k int) U128 {
	s := uint(k) & (127)
	if s > 64 {
		s = 128 - s
		l := 64 - s
		return U128{
			hi: u.hi>>s | u.lo<<l,
			lo: u.lo>>s | u.hi<<l,
		}
	} else {
		l := 64 - s
		return U128{
			hi: u.hi<<s | u.lo>>l,
			lo: u.lo<<s | u.hi>>l,
		}
	}
}

func (u U128) LeadingZeros() uint {
	if u.hi == 0 {
		return uint(bits.LeadingZeros64(u.lo)) + 64
	} else {
		return uint(bits.LeadingZeros64(u.hi))
	}
}

func (u U128) TrailingZeros() uint {
	if u.lo == 0 {
		return uint(bits.TrailingZeros64(u.hi)) + 64
	} else {
		return uint(bits.TrailingZeros64(u.lo))
	}
}

// Hacker's delight 9-4, divlu:
func quo128by64(u1, u0, v uint64, vLeading0 uint) (q uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, vs, left, right uint64

	vs = v << vLeading0

	vn1 = vs >> 32
	vn0 = vs & 0xffffffff

	if vLeading0 > 0 {
		un32 = (u1 << vLeading0) | (u0 >> (64 - vLeading0))
		un10 = u0 << vLeading0
	} else {
		un32 = u1
		un10 = u0
	}

	un1 = un10 >> 32
	un0 = un10 & 0xffffffff

	q1 = un32 / vn1
	rhat = un32 % vn1

	left = q1 * vn0
	right = (rhat << 32) | un1

again1:
	if (q1 >= b) || (left > right) {
		q1--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un1
			goto again1
		}
	}

	un21 = (un32 << 32) + (un1 - (q1 * vs))

	q0 = un21 / vn1
	rhat = un21 % vn1

	left = q0 * vn0
	right = (rhat << 32) | un0

again2:
	if (q0 >= b) || (left > right) {
		q0--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un0
			goto again2
		}
	}

	return (q1 << 32) | q0
}

// Hacker's delight 9-4, divlu:
func quorem128by64(u1, u0, v uint64, vLeading0 uint) (q, r uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, left, right uint64

	v <<= vLeading0

	vn1 = v >> 32
	vn0 = v & 0xffffffff

	if vLeading0 > 0 {
		un32 = (u1 << vLeading0) | (u0 >> (64 - vLeading0))
		un10 = u0 << vLeading0
	} else {
		un32 = u1
		un10 = u0
	}

	un1 = un10 >> 32
	un0 = un10 & 0xffffffff

	q1 = un32 / vn1
	rhat = un32 % vn1

	left = q1 * vn0
	right = (rhat << 32) + un1

again1:
	if (q1 >= b) || (left > right) {
		q1--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un1
			goto again1
		}
	}

	un21 = (un32 << 32) + (un1 - (q1 * v))

	q0 = un21 / vn1
	rhat = un21 % vn1

	left = q0 * vn0
	right = (rhat << 32) | un0

again2:
	if (q0 >= b) || (left > right) {
		q0--
		rhat += vn1
		if rhat < b {
			left -= vn0
			right = (rhat << 32) | un0
			goto again2
		}
	}

	return (q1 << 32) | q0, ((un21 << 32) + (un0 - (q0 * v))) >> vLeading0
}

func quorem128by128(m, v U128, vHiLeading0, vLoLeading0 uint) (q, r U128) {
	if v.hi == 0 {
		if m.hi < v.lo {
			q.lo, r.lo = quorem128by64(m.hi, m.lo, v.lo, vLoLeading0)
			return q, r

		} else {
			q.hi = m.hi / v.lo
			r.hi = m.hi % v.lo
			q.lo, r.lo = quorem128by64(r.hi, m.lo, v.lo, vLoLeading0)
			r.hi = 0
			return q, r
		}

	} else {
		v1 := v.Lsh(vHiLeading0)
		u1 := m.Rsh(1)

		var q1 U128
		q1.lo = quo128by64(u1.hi, u1.lo, v1.hi, vLoLeading0)
		q1 = q1.Rsh(63 - vHiLeading0)

		if q1.hi|q1.lo != 0 {
			q1 = q1.Dec()
		}
		q = q1
		q1 = q1.Mul(v)
		r = m.Sub(q1)

		if r.Cmp(v) >= 0 {
			q = q.Inc()
			r = r.Sub(v)
		}

		return q, r
	}
}

func quorem128bin(u, by U128, uLeading0, byLeading0 uint) (q, r U128) {
	shift := int(byLeading0 - uLeading0)
	by = by.Lsh(uint(shift))

	for {
		// q << 1
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1

		// performance tweak: simulate greater than or equal by hand-inlining "not less than".
		if u.hi > by.hi || (u.hi == by.hi && u.lo >= by.lo) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// by >> 1
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1

		if shift <= 0 {
			break
		}
		shift--
	}

	r = u
	return q, r
}

func quo128bin(u, by U128, uLeading0, byLeading0 uint) (q U128) {
	shift := int(byLeading0 - uLeading0)
	by = by.Lsh(uint(shift))

	for {
		// q << 1
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1

		// u >= by
		if u.hi > by.hi || (u.hi == by.hi && u.lo >= by.lo) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// q >> 1
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1

		if shift <= 0 {
			break
		}
		shift--
	}

	return q
}

func (u U128) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *U128) UnmarshalText(bts []byte) (err error) {
	v, _, err := U128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

func (u U128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u *U128) UnmarshalJSON(bts []byte) (err error) {
	if bts[0] == '"' {
		ln := len(bts)
		if bts[ln-1] != '"' {
			return fmt.Errorf("num: u128 invalid JSON %q", string(bts))
		}
		bts = bts[1 : ln-1]
	}

	v, _, err := U128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

// Put big-endian encoded bytes representing this U128 into byte slice b.
// len(b) must be >= 16.
func (u U128) PutBigEndian(b []byte) {
	_ = b[15] // BCE
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = byte(u.hi>>56), byte(u.hi>>48), byte(u.hi>>40), byte(u.hi>>32), byte(u.hi>>24), byte(u.hi>>16), byte(u.hi>>8), byte(u.hi)
	b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15] = byte(u.lo>>56), byte(u.lo>>48), byte(u.lo>>40), byte(u.lo>>32), byte(u.lo>>24), byte(u.lo>>16), byte(u.lo>>8), byte(u.lo)
}

// Decode 16 bytes as a big-endian U128. Panics if len(b) < 16.
func MustU128FromBigEndian(b []byte) U128 {
	_ = b[15] // BCE
	return U128{
		lo: uint64(b[15]) | uint64(b[14])<<8 | uint64(b[13])<<16 | uint64(b[12])<<24 |
			uint64(b[11])<<32 | uint64(b[10])<<40 | uint64(b[9])<<48 | uint64(b[8])<<56,
		hi: uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
			uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56,
	}
}

// Put little-endian encoded bytes representing this U128 into byte slice b.
// len(b) must be >= 16.
func (u U128) PutLittleEndian(b []byte) {
	_ = b[15] // BCE
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7] = byte(u.lo), byte(u.lo>>8), byte(u.lo>>16), byte(u.lo>>24), byte(u.lo>>32), byte(u.lo>>40), byte(u.lo>>48), byte(u.lo>>56)
	b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15] = byte(u.hi), byte(u.hi>>8), byte(u.hi>>16), byte(u.hi>>24), byte(u.hi>>32), byte(u.hi>>40), byte(u.hi>>48), byte(u.hi>>56)
}

// Decode 16 bytes as a little-endian U128. Panics if len(b) < 16.
func MustU128FromLittleEndian(b []byte) U128 {
	_ = b[15] // BCE
	return U128{
		lo: uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56,
		hi: uint64(b[8]) | uint64(b[9])<<8 | uint64(b[10])<<16 | uint64(b[11])<<24 |
			uint64(b[12])<<32 | uint64(b[13])<<40 | uint64(b[14])<<48 | uint64(b[15])<<56,
	}
}
