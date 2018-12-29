package num

func mul64to128(u, v uint64) (hi, lo uint64)
func mul128to256(uhi, ulo, vhi, vlo uint64) (hi, hm, hl, lo uint64)
func mul128to128(uhi, ulo, nhi, nlo uint64) (ohi, olo uint64)
