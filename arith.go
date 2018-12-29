//+build !amd64

package num

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

func mul128to256(uhi, ulo, vhi, vlo uint64) (hi, hm, lm, lo uint64) {
	hi, hm = mul64to128(uhi, vhi)
	lm, lo = mul64to128(ulo, vlo)

	thi, tlo := mul64to128(uhi, vlo)

	lm += tlo

	if lm < tlo { // if lo.Hi overflowed
		hi, hm = U128{hi: hi, lo: hm}.Inc().Raw()
	}

	hm += thi
	if hm < thi { // if hi.Lo overflowed
		hi++
	}

	thi, tlo = mul64to128(ulo, vhi)

	lm += tlo
	if lm < tlo { // if L.Hi overflowed
		hi, hm = U128{hi: hi, lo: hm}.Inc().Raw()
	}

	hm += thi
	if hm < thi { // if H.Lo overflowed
		hi++
	}

	return hi, hm, lm, lo
}

func mul128to128(uhi, ulo, nhi, nlo uint64) (ohi, olo uint64) {
	// Adapted from Warren, Hacker's Delight, p. 132.
	hl := uhi*nlo + ulo*nhi

	olo = ulo * nlo // lower 64 bits are easy

	// break the multiplication into (x1 << 32 + x0)(y1 << 32 + y0)
	// which is x1*y1 << 64 + (x0*y1 + x1*y0) << 32 + x0*y0
	// so now we can do 64 bit multiplication and addition and
	// shift the results into the right place
	x0, x1 := ulo&0x00000000ffffffff, ulo>>32
	y0, y1 := nlo&0x00000000ffffffff, nlo>>32
	t := x1*y0 + (x0*y0)>>32
	w1 := (t & 0x00000000ffffffff) + (x0 * y1)
	ohi = (x1 * y1) + (t >> 32) + (w1 >> 32) + hl
	return ohi, olo
}
