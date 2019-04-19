package num

import (
	"fmt"
	"math/big"
	"math/bits"
	"strconv"
)

// U256 exists to implement just enough of a 256-bit integer to calculate reciprocals
// for fast 128-bit division.
type U256 struct {
	hi, hm, lm, lo uint64
}

func U256From128(in U128) U256 {
	hi, lo := in.Raw()
	return U256{lm: hi, lo: lo}
}

func U256From64(in uint64) U256 {
	return U256{lo: in}
}

// U256FromBigInt creates a U256 from a big.Int. Overflow truncates to MaxU256
// and sets inRange to 'false'.
func U256FromBigInt(v *big.Int) (out U256, inRange bool) {
	if v.Sign() < 0 {
		return out, false
	}

	words := v.Bits()

	switch intSize {
	case 64:
		lw := len(words)
		switch lw {
		case 0:
			return U256{}, true
		case 1:
			return U256{lo: uint64(words[0])}, true
		case 2:
			return U256{lm: uint64(words[1]), lo: uint64(words[0])}, true
		case 3:
			return U256{hm: uint64(words[2]), lm: uint64(words[1]), lo: uint64(words[0])}, true
		case 4:
			return U256{hi: uint64(words[3]), hm: uint64(words[2]), lm: uint64(words[1]), lo: uint64(words[0])}, true
		default:
			return MaxU256, false
		}

	case 32:
		lw := len(words)
		switch lw {
		case 0:
			return U256{}, true
		case 1:
			return U256{lo: uint64(words[0])}, true
		case 2:
			return U256{lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 3:
			return U256{lm: uint64(words[2]), lo: (uint64(words[1]) << 32) | (uint64(words[0]))}, true
		case 4:
			return U256{
				lm: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true

		case 5:
			return U256{
				hm: uint64(words[4]),
				lm: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		case 6:
			return U256{
				hm: (uint64(words[5]) << 32) | (uint64(words[4])),
				lm: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		case 7:
			return U256{
				hi: uint64(words[6]),
				hm: (uint64(words[5]) << 32) | (uint64(words[4])),
				lm: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		case 8:
			return U256{
				hi: (uint64(words[7]) << 32) | (uint64(words[6])),
				hm: (uint64(words[5]) << 32) | (uint64(words[4])),
				lm: (uint64(words[3]) << 32) | (uint64(words[2])),
				lo: (uint64(words[1]) << 32) | (uint64(words[0])),
			}, true
		default:
			return MaxU256, false
		}

	default:
		panic("num: unsupported bit size")
	}
}

func (u U256) And(n U256) U256 {
	u.hi = u.hi & n.hi
	u.hm = u.hm & n.hm
	u.lm = u.lm & n.lm
	u.lo = u.lo & n.lo
	return u
}

func (u U256) AndNot(n U256) U256 {
	u.hi = u.hi &^ n.hi
	u.hm = u.hm &^ n.hm
	u.lm = u.lm &^ n.lm
	u.lo = u.lo &^ n.lo
	return u
}

func (u U256) Not() U256 {
	u.hi = ^u.hi
	u.hm = ^u.hm
	u.lm = ^u.lm
	u.lo = ^u.lo
	return u
}

func (u U256) Or(n U256) U256 {
	u.hi = u.hi | n.hi
	u.hm = u.hm | n.hm
	u.lm = u.lm | n.lm
	u.lo = u.lo | n.lo
	return u
}

func (u U256) Xor(n U256) U256 {
	u.hi = u.hi ^ n.hi
	u.hm = u.hm ^ n.hm
	u.lm = u.lm ^ n.lm
	u.lo = u.lo ^ n.lo
	return u
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

func (u U256) Equal(v U256) bool            { return u.Cmp(v) == 0 }
func (u U256) GreaterThan(v U256) bool      { return u.Cmp(v) > 0 }
func (u U256) GreaterOrEqualTo(v U256) bool { return u.Cmp(v) >= 0 }
func (u U256) LessThan(v U256) bool         { return u.Cmp(v) < 0 }
func (u U256) LessOrEqualTo(v U256) bool    { return u.Cmp(v) <= 0 }

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

func (u U256) Quo(by U256) (q U256) {
	q, _ = u.QuoRem(by)
	return q
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

func (u U256) Rem(by U256) (r U256) {
	_, r = u.QuoRem(by)
	return r
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

func (u U256) AsU128() U128 { return U128FromRaw(u.lm, u.lo) }

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
