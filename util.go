package num

type RandSource interface {
	Uint64() uint64
}

// DifferenceU128 subtracts the smaller of a and b from the larger.
func DifferenceU128(a, b U128) U128 {
	if a.hi > b.hi {
		return a.Sub(b)
	} else if a.hi < b.hi {
		return b.Sub(a)
	} else if a.lo > b.lo {
		return a.Sub(b)
	} else if a.lo < b.lo {
		return b.Sub(a)
	}
	return U128{}
}

func LargerU128(a, b U128) U128 {
	if a.hi > b.hi {
		return a
	} else if a.hi < b.hi {
		return b
	} else if a.lo > b.lo {
		return a
	} else if a.lo < b.lo {
		return b
	}
	return a
}

func SmallerU128(a, b U128) U128 {
	if a.hi < b.hi {
		return a
	} else if a.hi > b.hi {
		return b
	} else if a.lo < b.lo {
		return a
	} else if a.lo > b.lo {
		return b
	}
	return a
}

// DifferenceI128 subtracts the smaller of a and b from the larger.
func DifferenceI128(a, b I128) I128 {
	if a.hi > b.hi {
		return a.Sub(b)
	} else if a.hi < b.hi {
		return b.Sub(a)
	} else if a.lo > b.lo {
		return a.Sub(b)
	} else if a.lo < b.lo {
		return b.Sub(a)
	}
	return I128{}
}
