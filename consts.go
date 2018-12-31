package num

import (
	"math/big"
)

const (
	maxUint64 = 1<<64 - 1
	maxInt64  = 1<<63 - 1
	minInt64  = -1 << 63

	minInt64Float = float64(minInt64) // -(1<<63)
	maxInt64Float = float64(maxInt64) // (1<<63) - 1

	maxUint64Float  = float64(maxUint64)     // (1<<64) - 1
	wrapUint64Float = float64(maxUint64) + 1 // 1 << 64

	maxU128Float = float64(340282366920938463463374607431768211455)  // (1<<128) - 1
	maxI128Float = float64(170141183460469231731687303715884105727)  // (1<<127) - 1
	minI128Float = float64(-170141183460469231731687303715884105728) // -(1<<127)

	intSize = 32 << (^uint(0) >> 63)
)

var (
	MaxI128 = I128{hi: 0x7FFFFFFFFFFFFFFF, lo: 0xFFFFFFFFFFFFFFFF}
	MinI128 = I128{hi: 0x8000000000000000, lo: 0}
	MaxU128 = U128{hi: maxUint64, lo: maxUint64}

	zeroI128 I128
	zeroU128 U128

	big0 = new(big.Int).SetInt64(0)
	big1 = new(big.Int).SetInt64(1)

	maxBigUint64  = new(big.Int).SetUint64(maxUint64)
	maxBigU128, _ = new(big.Int).SetString("340282366920938463463374607431768211455", 10)
	maxBigInt64   = new(big.Int).SetUint64(maxInt64)

	minBigI128, _ = new(big.Int).SetString("-170141183460469231731687303715884105728", 10)
	maxBigI128, _ = new(big.Int).SetString("170141183460469231731687303715884105727", 10)

	// wrapBigU128 is 1 << 128, used to simulate over/underflow:
	wrapBigU128, _ = new(big.Int).SetString("340282366920938463463374607431768211456", 10)

	// wrapBigU64 is 1 << 64:
	wrapBigU64, _ = new(big.Int).SetString("18446744073709551616", 10)

	// wrapOverBigI128 is 1 << 127, used to simulate over/underflow:
	wrapOverBigI128, _ = new(big.Int).SetString("0x7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 0)

	// wrapUnderBigI128 is -(1 << 127) - 1, used to simulate over/underflow:
	wrapUnderBigI128, _ = new(big.Int).SetString("-170141183460469231731687303715884105729", 0)

	// This specifies the maximum error allowed between the float64 version of
	// a 128-bit int/uint and the result of the same operation performed by
	// big.Float.
	//
	// Calculate like so:
	//	return math.Nextafter(1.0, 2.0) - 1.0
	//
	floatDiffLimit, _ = new(big.Float).SetString("2.220446049250313080847263336181640625e-16")
)
