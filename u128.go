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

func U128FromRaw(hi, lo uint64) U128 { return U128{hi: hi, lo: lo} }
func U128From64(v uint64) U128       { return U128{hi: 0, lo: v} }
func U128From32(v uint32) U128       { return U128{hi: 0, lo: uint64(v)} }
func U128From16(v uint16) U128       { return U128{hi: 0, lo: uint64(v)} }
func U128From8(v uint8) U128         { return U128{hi: 0, lo: uint64(v)} }

// U128FromString creates a U128 from a string. Overflow truncates to MaxU128
// and sets accurate to 'false'. Only decimal strings are currently supported.
func U128FromString(s string) (out U128, accurate bool, err error) {
	// This deliberately limits the scope of what we accept as input just in case
	// we decide to hand-roll our own fast decimal-only parser:
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return out, false, fmt.Errorf("num: u128 string %q invalid", s)
	}
	out, accurate = U128FromBigInt(b)
	return out, accurate, nil
}

// U128FromBigInt creates a U128 from a big.Int. Overflow truncates to MaxU128
// and sets accurate to 'false'.
func U128FromBigInt(v *big.Int) (out U128, accurate bool) {
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

func U128FromFloat32(f float32) (out U128, inRange bool) {
	return U128FromFloat64(float64(f))
}

// U128FromFloat64 creates a U128 from a float64. Any fractional portion
// will be truncated towards zero. Floats outside the bounds of a U128
// may be discarded or clamped.
//
// NaN is treated as 0, inRange is set to false. This may change to a panic
// at some point.
func U128FromFloat64(f float64) (out U128, inRange bool) {
	if f == 0 {
		return U128{}, true

	} else if f < 0 {
		return U128{}, false

	} else if f <= maxUint64Float {
		return U128{lo: uint64(f)}, true

	} else if f <= maxU128Float {
		lo := mod(f, wrapUint64Float)
		return U128{hi: uint64(f / wrapUint64Float), lo: uint64(lo)}, true

	} else if f != f { // (f != f) == NaN
		return U128{}, false

	} else {
		return MaxU128, false
	}
}

// RandU128 generates an unsigned 128-bit random integer from an external source.
func RandU128(source RandSource) (out U128) {
	return U128{hi: source.Uint64(), lo: source.Uint64()}
}

func (u U128) IsZero() bool { return u == zeroU128 }

// Raw returns access to the U128 as a pair of uint64s. See U128FromRaw() for
// the counterpart.
func (u U128) Raw() (hi, lo uint64) { return u.hi, u.lo }

func (u U128) String() string {
	// FIXME: This is good enough for now, but not forever.
	if u == zeroU128 {
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

// IsI128 reports wehether i can be represented in an I128.
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

func (u U128) Inc() (v U128) {
	v.lo = u.lo + 1
	v.hi = u.hi
	if u.lo > v.lo {
		v.hi++
	}
	return v
}

func (u U128) Dec() (v U128) {
	v.lo = u.lo - 1
	v.hi = u.hi
	if u.lo < v.lo {
		v.hi--
	}
	return v
}

func (u U128) Add(n U128) (v U128) {
	v.lo = u.lo + n.lo
	v.hi = u.hi + n.hi
	if u.lo > v.lo {
		v.hi++
	}
	return v
}

func (u U128) Sub(n U128) (v U128) {
	v.lo = u.lo - n.lo
	v.hi = u.hi - n.hi
	if u.lo < v.lo {
		v.hi--
	}
	return v
}

func (u U128) Cmp(n U128) int {
	if u.hi > n.hi {
		return 1
	} else if u.hi < n.hi {
		return -1
	} else if u.lo > n.lo {
		return 1
	} else if u.lo < n.lo {
		return -1
	}
	return 0
}

func (u U128) Equal(n U128) bool {
	return u.hi == n.hi && u.lo == n.lo
}

func (u U128) GreaterThan(n U128) bool {
	return u.hi > n.hi || (u.hi == n.hi && u.lo > n.lo)
}

func (u U128) GreaterOrEqualTo(n U128) bool {
	if u.hi > n.hi {
		return true
	} else if u.hi < n.hi {
		return false
	} else if u.lo > n.lo {
		return true
	} else if u.lo < n.lo {
		return false
	}
	return true
}

func (u U128) LessThan(n U128) bool {
	return u.hi < n.hi || (u.hi == n.hi && u.lo < n.lo)
}

func (u U128) LessOrEqualTo(n U128) bool {
	if u.hi > n.hi {
		return false
	} else if u.hi < n.hi {
		return true
	} else if u.lo > n.lo {
		return false
	} else if u.lo < n.lo {
		return true
	}
	return true
}

func (u U128) And(v U128) (out U128) {
	out.hi = u.hi & v.hi
	out.lo = u.lo & v.lo
	return out
}

func (u U128) Or(v U128) (out U128) {
	out.hi = u.hi | v.hi
	out.lo = u.lo | v.lo
	return out
}

func (u U128) Xor(v U128) (out U128) {
	out.hi = u.hi ^ v.hi
	out.lo = u.lo ^ v.lo
	return out
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

func (u U128) Mul(n U128) (dest U128) {
	// Adapted from Warren, Hacker's Delight, p. 132.
	hl := u.hi*n.lo + u.lo*n.hi

	dest.lo = u.lo * n.lo // lower 64 bits are easy

	// break the multiplication into (x1 << 32 + x0)(y1 << 32 + y0)
	// which is x1*y1 << 64 + (x0*y1 + x1*y0) << 32 + x0*y0
	// so now we can do 64 bit multiplication and addition and
	// shift the results into the right place
	x0, x1 := u.lo&0x00000000ffffffff, u.lo>>32
	y0, y1 := n.lo&0x00000000ffffffff, n.lo>>32
	t := x1*y0 + (x0*y0)>>32
	w1 := (t & 0x00000000ffffffff) + (x0 * y1)
	dest.hi = (x1 * y1) + (t >> 32) + (w1 >> 32) + hl

	return dest
}

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

	byLeading0 := by.LeadingZeros()
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
	if byLeading0-uLeading0 > 5 {
		q, _ = quorem128by128(u, by)
		return q
	} else {
		return quo128bin(u, by, uLeading0, byLeading0)
	}
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

	byLeading0 := by.LeadingZeros()
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

	// See BenchmarkU128QuoRemTZ for the test that helps determine this magic number:
	uLeading0 := u.LeadingZeros()
	if byLeading0-uLeading0 > 16 {
		return quorem128by128(u, by)
	} else {
		return quorem128bin(u, by, uLeading0, byLeading0)
	}
}

// Rem returns the remainder of x%y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Rem implements truncated modulus (like Go); see
// QuoRem for more details.
func (u U128) Rem(by U128) (r U128) {
	_, r = u.QuoRem(by)
	return r
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
func quo128by64(u1, u0, v uint64) (q uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, vs, left, right uint64

	s := uint(bits.LeadingZeros64(v))
	vs = v << s

	vn1 = vs >> 32
	vn0 = vs & 0xffffffff

	if s > 0 {
		un32 = (u1 << s) | (u0 >> (64 - s))
		un10 = u0 << s
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
func quorem128by64(u1, u0, v uint64) (q, r uint64) {
	var b uint64 = 1 << 32
	var un1, un0, vn1, vn0, q1, q0, un32, un21, un10, rhat, left, right uint64

	s := uint(bits.LeadingZeros64(v))
	v <<= s

	vn1 = v >> 32
	vn0 = v & 0xffffffff

	if s > 0 {
		un32 = (u1 << s) | (u0 >> (64 - s))
		un10 = u0 << s
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

	return (q1 << 32) | q0, ((un21 << 32) + (un0 - (q0 * v))) >> s
}

func quorem128by128(m, v U128) (q, r U128) {
	if v.hi == 0 {
		if m.hi < v.lo {
			q.lo, r.lo = quorem128by64(m.hi, m.lo, v.lo)
			return q, r

		} else {
			q.hi = m.hi / v.lo
			r.hi = m.hi % v.lo
			q.lo, r.lo = quorem128by64(r.hi, m.lo, v.lo)
			r.hi = 0
			return q, r
		}

	} else {
		sh := uint(bits.LeadingZeros64(v.hi))

		v1 := v.Lsh(sh)
		u1 := m.Rsh(1)

		var q1 U128
		q1.lo = quo128by64(u1.hi, u1.lo, v1.hi)
		q1 = q1.Rsh(63 - sh)

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
		// {{{ Lsh(1)
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1
		// }}}

		// performance tweak: simulate greater than or equal by hand-inlining "not less than".
		if !(u.hi < by.hi || (u.hi == by.hi && u.lo < by.lo)) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// {{{ Rsh(1)
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1
		// }}}

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
		// {{{ Lsh(1)
		q.hi = (q.hi << 1) | (q.lo >> 63)
		q.lo = q.lo << 1
		// }}}

		// performance tweak: simulate greater than or equal by hand-inlining "not less than".
		if !(u.hi < by.hi || (u.hi == by.hi && u.lo < by.lo)) {
			u = u.Sub(by)
			q.lo |= 1
		}

		// {{{ Rsh(1)
		by.lo = (by.lo >> 1) | (by.hi << 63)
		by.hi = by.hi >> 1
		// }}}

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

func mul64to128(u, v uint64) (hi, lo uint64) {
	var (
		u1 = (u & 0xffffffff)
		v1 = (v & 0xffffffff)
		t  = (u1 * v1)
		w3 = (t & 0xffffffff)
		k  = (t >> 32)
	)

	u >>= 32
	t = (u * v1) + k
	k = (t & 0xffffffff)
	var w1 = (t >> 32)

	v >>= 32
	t = (u1 * v) + k
	k = (t >> 32)

	return (u * v) + w1 + k,
		(t << 32) + w3
}

func mul128to256(n, by U128) (hi, lo U128) {
	{ // hi = U128FromRaw(mul64to128(n.hi, by.hi))
		// "cannot inline mul64to128: function too complex: cost 85 exceeds budget 80"
		// All this rotten hand-inlining doubles the speed of this function. Not a bad
		// quick win, all things considered.
		// :'(
		var (
			u  = n.hi
			v  = by.hi
			u1 = (u & 0xffffffff)
			v1 = (v & 0xffffffff)
			t  = (u1 * v1)
			w3 = (t & 0xffffffff)
			k  = (t >> 32)
		)

		u >>= 32
		t = (u * v1) + k
		k = (t & 0xffffffff)
		var w1 = (t >> 32)

		v >>= 32
		t = (u1 * v) + k
		k = (t >> 32)

		hi = U128{
			hi: (u * v) + w1 + k,
			lo: (t << 32) + w3,
		}
	}

	{ // lo = U128FromRaw(mul64to128(n.lo, by.lo))
		var (
			u  = n.lo
			v  = by.lo
			u1 = (u & 0xffffffff)
			v1 = (v & 0xffffffff)
			t  = (u1 * v1)
			w3 = (t & 0xffffffff)
			k  = (t >> 32)
		)

		u >>= 32
		t = (u * v1) + k
		k = (t & 0xffffffff)
		var w1 = (t >> 32)

		v >>= 32
		t = (u1 * v) + k
		k = (t >> 32)

		lo = U128{
			hi: (u * v) + w1 + k,
			lo: (t << 32) + w3,
		}
	}

	var t U128

	{ // t = U128FromRaw(mul64to128(n.hi, by.lo))
		var (
			u  = n.hi
			v  = by.lo
			u1 = (u & 0xffffffff)
			v1 = (v & 0xffffffff)
			ti = (u1 * v1)
			w3 = (ti & 0xffffffff)
			k  = (ti >> 32)
		)

		u >>= 32
		ti = (u * v1) + k
		k = (ti & 0xffffffff)
		var w1 = (ti >> 32)

		v >>= 32
		ti = (u1 * v) + k
		k = (ti >> 32)

		t = U128{
			hi: (u * v) + w1 + k,
			lo: (ti << 32) + w3,
		}
	}

	lo.hi += t.lo

	if lo.hi < t.lo { // if lo.Hi overflowed
		hi = hi.Inc()
	}

	hi.lo += t.hi
	if hi.lo < t.hi { // if hi.Lo overflowed
		hi.hi++
	}

	{ // t = U128FromRaw(mul64to128(n.hi, by.lo))
		var (
			u  = n.lo
			v  = by.hi
			u1 = (u & 0xffffffff)
			v1 = (v & 0xffffffff)
			ti = (u1 * v1)
			w3 = (ti & 0xffffffff)
			k  = (ti >> 32)
		)

		u >>= 32
		ti = (u * v1) + k
		k = (ti & 0xffffffff)
		var w1 = (ti >> 32)

		v >>= 32
		ti = (u1 * v) + k
		k = (ti >> 32)

		t = U128{
			hi: (u * v) + w1 + k,
			lo: (ti << 32) + w3,
		}
	}

	lo.hi += t.lo
	if lo.hi < t.lo { // if L.Hi overflowed
		hi = hi.Inc()
	}

	hi.lo += t.hi
	if hi.lo < t.hi { // if H.Lo overflowed
		hi.hi++
	}

	return hi, lo
}
