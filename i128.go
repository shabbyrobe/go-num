package num

import (
	"fmt"
	"math/big"
)

type I128 struct {
	hi uint64
	lo uint64
}

const (
	signBit  = 0x8000000000000000
	signMask = 0x7FFFFFFFFFFFFFFF
)

// I128FromString creates a I128 from a string. Overflow truncates to
// MaxI128/MinI128 and sets accurate to 'false'. Only decimal strings are
// currently supported.
func I128FromString(s string) (out I128, accurate bool, err error) {
	// This deliberately limits the scope of what we accept as input just in case
	// we decide to hand-roll our own fast decimal-only parser:
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return out, false, fmt.Errorf("num: i128 string %q invalid", s)
	}
	out, accurate = I128FromBigInt(b)
	return out, accurate, nil
}

// I128FromRaw is the complement to I128.Raw(); it creates an I128 from two
// uint64s representing the hi and lo bits.
func I128FromRaw(hi, lo uint64) I128 {
	return I128{hi: hi, lo: lo}
}

func I128From64(v int64) I128 {
	var hi uint64
	if v < 0 {
		hi = maxUint64
	}
	return I128{hi: hi, lo: uint64(v)}
}

func I128From32(v int32) I128   { return I128From64(int64(v)) }
func I128From16(v int16) I128   { return I128From64(int64(v)) }
func I128From8(v int8) I128     { return I128From64(int64(v)) }
func I128FromInt(v int) I128    { return I128From64(int64(v)) }
func I128FromU64(v uint64) I128 { return I128{lo: v} }

var (
	minI128AsAbsU128 = U128{hi: 0x8000000000000000, lo: 0}
	maxI128AsU128    = U128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}
)

func I128FromBigInt(v *big.Int) (out I128, accurate bool) {
	neg := v.Sign() < 0

	words := v.Bits()

	var u U128
	accurate = true

	switch intSize {
	case 64:
		lw := len(words)
		switch lw {
		case 0:
		case 1:
			u.lo = uint64(words[0])
		case 2:
			u.hi = uint64(words[1])
			u.lo = uint64(words[0])
		default:
			u, accurate = MaxU128, false
		}

	case 32:
		lw := len(words)
		switch lw {
		case 0:
		case 1:
			u.lo = uint64(words[0])
		case 2:
			u.lo = (uint64(words[1]) << 32) | (uint64(words[0]))
		case 3:
			u.hi = uint64(words[2])
			u.lo = (uint64(words[1]) << 32) | (uint64(words[0]))
		case 4:
			u.hi = (uint64(words[3]) << 32) | (uint64(words[2]))
			u.lo = (uint64(words[1]) << 32) | (uint64(words[0]))
		default:
			u, accurate = MaxU128, false
		}

	default:
		panic("num: unsupported bit size")
	}

	if !neg {
		if cmp := u.Cmp(maxI128AsU128); cmp == 0 {
			out = MaxI128
		} else if cmp > 0 {
			out, accurate = MaxI128, false
		} else {
			out = u.AsI128()
		}

	} else {
		if cmp := u.Cmp(minI128AsAbsU128); cmp == 0 {
			out = MinI128
		} else if cmp > 0 {
			out, accurate = MinI128, false
		} else {
			out = u.AsI128().Neg()
		}
	}

	return out, accurate
}

func I128FromFloat32(f float32) (out I128, inRange bool) {
	return I128FromFloat64(float64(f))
}

// I128FromFloat64 creates a I128 from a float64.
//
// Any fractional portion will be truncated towards zero.
//
// Floats outside the bounds of a I128 may be discarded or clamped and inRange
// will be set to false.
//
// NaN is treated as 0, inRange is set to false. This may change to a panic
// at some point.
func I128FromFloat64(f float64) (out I128, inRange bool) {
	const spillPos = float64(maxUint64) // (1<<64) - 1
	const spillNeg = -float64(maxUint64) - 1

	if f == 0 {
		return out, true

	} else if f != f { // f != f == isnan
		return out, false

	} else if f < 0 {
		if f >= spillNeg {
			return I128{hi: maxUint64, lo: uint64(f)}, true
		} else if f >= minI128Float {
			f = -f
			lo := modpos(f, wrapUint64Float) // f is guaranteed to be < 0 here.
			return I128{hi: ^uint64(f / wrapUint64Float), lo: ^uint64(lo)}, true
		} else {
			return MinI128, false
		}

	} else {
		if f <= spillPos {
			return I128{lo: uint64(f)}, true
		} else if f <= maxI128Float {
			lo := modpos(f, wrapUint64Float) // f is guaranteed to be > 0 here.
			return I128{hi: uint64(f / wrapUint64Float), lo: uint64(lo)}, true
		} else {
			return MaxI128, false
		}
	}
}

// RandI128 generates a positive signed 128-bit random integer from an external
// source.
func RandI128(source RandSource) (out I128) {
	return I128{hi: source.Uint64() & maxInt64, lo: source.Uint64()}
}

func (i I128) IsZero() bool { return i == zeroI128 }

// Raw returns access to the I128 as a pair of uint64s. See I128FromRaw() for
// the counterpart.
func (i I128) Raw() (hi uint64, lo uint64) { return i.hi, i.lo }

func (i I128) String() string {
	// FIXME: This is good enough for now, but not forever.
	v := i.AsBigInt()
	return v.String()
}

func (i I128) Format(s fmt.State, c rune) {
	// FIXME: This is good enough for now, but not forever.
	i.AsBigInt().Format(s, c)
}

// IntoBigInt copies this I128 into a big.Int, allowing you to retain and
// recycle memory.
func (i I128) IntoBigInt(b *big.Int) {
	neg := i.hi&signBit != 0
	if i.hi > 0 {
		b.SetUint64(i.hi)
		b.Lsh(b, 64)
	}
	var lo big.Int
	lo.SetUint64(i.lo)
	b.Add(b, &lo)

	if neg {
		b.Xor(b, maxBigU128).Add(b, big1).Neg(b)
	}
}

// AsBigInt allocates a new big.Int and copies this I128 into it.
func (i I128) AsBigInt() (b *big.Int) {
	b = new(big.Int)
	neg := i.hi&signBit != 0
	if i.hi > 0 {
		b.SetUint64(i.hi)
		b.Lsh(b, 64)
	}
	var lo big.Int
	lo.SetUint64(i.lo)
	b.Add(b, &lo)

	if neg {
		b.Xor(b, maxBigU128).Add(b, big1).Neg(b)
	}

	return b
}

// AsU128 performs a direct cast of an I128 to a U128. Negative numbers
// become values > math.MaxI128.
func (i I128) AsU128() U128 {
	return U128{lo: i.lo, hi: i.hi}
}

// IsU128 reports wehether i can be represented in a U128.
func (i I128) IsU128() bool {
	return i.hi&signBit == 0
}

func (i I128) AsBigFloat() (b *big.Float) {
	return new(big.Float).SetInt(i.AsBigInt())
}

func (i I128) AsFloat64() float64 {
	if i.hi == 0 && i.lo == 0 {
		return 0
	} else if i.hi&signBit != 0 {
		if i.hi == maxUint64 {
			return -float64((^i.lo) + 1)
		} else {
			return (-float64(^i.hi) * maxUint64Float) + -float64(^i.lo)
		}
	} else {
		if i.hi == 0 {
			return float64(i.lo)
		} else {
			return (float64(i.hi) * maxUint64Float) + float64(i.lo)
		}
	}
}

// AsInt64 truncates the I128 to fit in a int64. Values outside the range will
// over/underflow. See IsInt64() if you want to check before you convert.
func (i I128) AsInt64() int64 {
	if i.hi&signBit != 0 {
		return -int64(^(i.lo - 1))
	} else {
		return int64(i.lo)
	}
}

// IsInt64 reports whether i can be represented as a int64.
func (i I128) IsInt64() bool {
	if i.hi&signBit != 0 {
		return i.hi == maxUint64 && i.lo >= 0x8000000000000000
	} else {
		return i.hi == 0 && i.lo <= maxInt64
	}
}

func (i I128) Sign() int {
	if i == zeroI128 {
		return 0
	} else if i.hi&signBit == 0 {
		return 1
	}
	return -1
}

func (i I128) Inc() (v I128) {
	v.lo = i.lo + 1
	v.hi = i.hi
	if i.lo > v.lo {
		v.hi++
	}
	return v
}

func (i I128) Dec() (v I128) {
	v.lo = i.lo - 1
	v.hi = i.hi
	if i.lo < v.lo {
		v.hi--
	}
	return v
}

func (i I128) Add(n I128) (v I128) {
	v.lo = i.lo + n.lo
	v.hi = i.hi + n.hi
	if i.lo > v.lo {
		v.hi++
	}
	return v
}

func (i I128) Sub(n I128) (out I128) {
	out.lo = i.lo - n.lo
	out.hi = i.hi - n.hi
	if i.lo < out.lo {
		out.hi--
	}
	return out
}

func (i I128) Neg() (v I128) {
	if i.hi == 0 && i.lo == 0 {
		return v
	}

	if i == MinI128 {
		// Overflow case: -MinI128 == MinI128
		return i

	} else if i.hi&signBit != 0 {
		v.hi = ^i.hi
		v.lo = ^(i.lo - 1)
	} else {
		v.hi = ^i.hi
		v.lo = (^i.lo) + 1
	}
	if v.lo == 0 { // handle overflow
		v.hi++
	}
	return v
}

func (i I128) Abs() I128 {
	if i.hi&signBit != 0 {
		i.hi = ^i.hi
		i.lo = ^(i.lo - 1)
		if i.lo == 0 { // handle overflow
			i.hi++
		}
	}
	return i
}

// Cmp compares u to n and returns:
//
//	< 0 if x <  y
//	  0 if x == y
//	> 0 if x >  y
//
// The specific value returned by Cmp is undefined, but it is guaranteed to
// satisfy the above constraints.
//
func (i I128) Cmp(n I128) int {
	if i.hi == n.hi && i.lo == n.lo {
		return 0
	} else if i.hi&signBit == n.hi&signBit {
		if i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo) {
			return 1
		}
	} else if i.hi&signBit == 0 {
		return 1
	}
	return -1
}

func (i I128) Equal(n I128) bool {
	return i.hi == n.hi && i.lo == n.lo
}

func (i I128) GreaterThan(n I128) bool {
	if i.hi&signBit == n.hi&signBit {
		return i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo)
	} else if i.hi&signBit == 0 {
		return true
	}
	return false
}

func (i I128) GreaterOrEqualTo(n I128) bool {
	if i.hi == n.hi && i.lo == n.lo {
		return true
	}
	if i.hi&signBit == n.hi&signBit {
		return i.hi > n.hi || (i.hi == n.hi && i.lo > n.lo)
	} else if i.hi&signBit == 0 {
		return true
	}
	return false
}

func (i I128) LessThan(n I128) bool {
	if i.hi&signBit == n.hi&signBit {
		return i.hi < n.hi || (i.hi == n.hi && i.lo < n.lo)
	} else if i.hi&signBit != 0 {
		return true
	}
	return false
}

func (i I128) LessOrEqualTo(n I128) bool {
	if i.hi == n.hi && i.lo == n.lo {
		return true
	}
	if i.hi&signBit == n.hi&signBit {
		return i.hi < n.hi || (i.hi == n.hi && i.lo < n.lo)
	} else if i.hi&signBit != 0 {
		return true
	}
	return false
}

// Mul returns the product of two I128s.
//
// Overflow should wrap around, as per the Go spec.
//
func (i I128) Mul(n I128) (dest I128) {
	// Unfortunately, this is slightly too complex for Go 1.11 to inline.

	// Adapted from Warren, Hacker's Delight, p. 132.
	hl := i.hi*n.lo + i.lo*n.hi

	dest.lo = i.lo * n.lo // lower 64 bits

	// break the multiplication into (x1 << 32 + x0)(y1 << 32 + y0)
	// which is x1*y1 << 64 + (x0*y1 + x1*y0) << 32 + x0*y0
	// so now we can do 64 bit multiplication and addition and
	// shift the results into the right place
	x0, x1 := i.lo&0x00000000ffffffff, i.lo>>32
	y0, y1 := n.lo&0x00000000ffffffff, n.lo>>32
	t := x1*y0 + (x0*y0)>>32
	w1 := (t & 0x00000000ffffffff) + (x0 * y1)
	dest.hi = (x1 * y1) + (t >> 32) + (w1 >> 32) + hl

	return dest
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
func (i I128) QuoRem(by I128) (q, r I128) {
	qSign, rSign := 1, 1
	if i.LessThan(zeroI128) {
		qSign, rSign = -1, -1
		i = i.Neg()
	}
	if by.LessThan(zeroI128) {
		qSign = -qSign
		by = by.Neg()
	}

	qu, ru := i.AsU128().QuoRem(by.AsU128())
	q, r = qu.AsI128(), ru.AsI128()
	if qSign < 0 {
		q = q.Neg()
	}
	if rSign < 0 {
		r = r.Neg()
	}
	return q, r
}

// Quo returns the quotient x/y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Quo implements truncated division (like Go); see
// QuoRem for more details.
func (i I128) Quo(by I128) (q I128) {
	qSign := 1
	if i.LessThan(zeroI128) {
		qSign = -1
		i = i.Neg()
	}
	if by.LessThan(zeroI128) {
		qSign = -qSign
		by = by.Neg()
	}

	qu := i.AsU128().Quo(by.AsU128())
	q = qu.AsI128()
	if qSign < 0 {
		q = q.Neg()
	}
	return q
}

// Rem returns the remainder of x%y for y != 0. If y == 0, a division-by-zero
// run-time panic occurs. Rem implements truncated modulus (like Go); see
// QuoRem for more details.
func (i I128) Rem(by I128) (r I128) {
	_, r = i.QuoRem(by)
	return r
}

func (u I128) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *I128) UnmarshalText(bts []byte) (err error) {
	v, _, err := I128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

func (u I128) MarshalJSON() ([]byte, error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u *I128) UnmarshalJSON(bts []byte) (err error) {
	if bts[0] == '"' {
		ln := len(bts)
		if bts[ln-1] != '"' {
			return fmt.Errorf("num: i128 invalid JSON %q", string(bts))
		}
		bts = bts[1 : ln-1]
	}

	v, _, err := I128FromString(string(bts))
	if err != nil {
		return err
	}
	*u = v
	return nil
}
