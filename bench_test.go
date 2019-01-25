package num

import (
	"math/big"
	"testing"
)

var (
	BenchBigFloatResult *big.Float
	BenchBigIntResult   *big.Int
	BenchBoolResult     bool
	BenchFloatResult    float64
	BenchIntResult      int
	BenchStringResult   string
	BenchU128Result     U128
	BenchUint64Result   uint64

	BenchUint641, BenchUint642 uint64 = 12093749018, 18927348917
)

func BenchmarkUint64Mul(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BenchUint64Result = BenchUint641 * BenchUint642
	}
}

func BenchmarkUint64Add(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BenchUint64Result = BenchUint641 + BenchUint642
	}
}

func BenchmarkUint64Div(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BenchUint64Result = BenchUint641 / BenchUint642
	}
}

func BenchmarkUint64Equal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BenchBoolResult = BenchUint641 == BenchUint642
	}
}

func BenchmarkBigIntMul(b *testing.B) {
	var max big.Int
	max.SetUint64(maxUint64)

	for i := 0; i < b.N; i++ {
		var dest big.Int
		dest.Mul(&dest, &max)
	}
}

func BenchmarkBigIntAdd(b *testing.B) {
	var max big.Int
	max.SetUint64(maxUint64)

	for i := 0; i < b.N; i++ {
		var dest big.Int
		dest.Add(&dest, &max)
	}
}

func BenchmarkBigIntDiv(b *testing.B) {
	u := new(big.Int).SetUint64(maxUint64)
	by := new(big.Int).SetUint64(121525124)
	for i := 0; i < b.N; i++ {
		var z big.Int
		z.Div(u, by)
	}
}

func BenchmarkBigIntCmpEqual(b *testing.B) {
	var v1, v2 big.Int
	v1.SetUint64(maxUint64)
	v2.SetUint64(maxUint64)

	for i := 0; i < b.N; i++ {
		BenchIntResult = v1.Cmp(&v2)
	}
}
