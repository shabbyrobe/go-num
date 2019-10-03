//+build !amd64

package num

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
