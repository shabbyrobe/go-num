//+build !amd64

package num

func quo128bin(uhi, ulo, byhi, bylo uint64, uLeading0, byLeading0 uint) (qhi, qlo uint64) {
	shift := byLeading0 - uLeading0

	if shift > 64 {
		byhi, bylo = (bylo << (shift - 64)), 0
	} else if shift < 64 {
		byhi, bylo = ((byhi << shift) | (bylo >> (64 - shift))), (bylo << shift)
	} else { // shift == 64
		byhi, bylo = bylo, 0
	}

	for {
		// {{{ Lsh(1)
		qhi = (qhi << 1) | (qlo >> 63)
		qlo = qlo << 1
		// }}}

		if uhi > byhi || (uhi == byhi && ulo >= bylo) {
			tmpLo := ulo - bylo
			uhi = uhi - byhi
			if ulo < tmpLo {
				uhi--
			}
			ulo = tmpLo
			qlo |= 1
		}

		// {{{ Rsh(1)
		bylo = (bylo >> 1) | (byhi << 63)
		byhi = byhi >> 1
		// }}}

		if shift <= 0 {
			break
		}
		shift--
	}

	return qhi, qlo
}
