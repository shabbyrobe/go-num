package num

import (
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/shabbyrobe/golib/assert"
)

func TestMul128To256(t *testing.T) {
	tt := assert.WrapTB(t)

	scratch := make([]byte, 32)

	for i := 0; i < 50000; i++ {
		u1, u2 := randU128(scratch), randU128(scratch)
		b1, b2 := u1.AsBigInt(), u2.AsBigInt()
		rhi, rhm, rlm, rlo := mul128to256(u1.hi, u1.lo, u2.hi, u2.lo)

		rb := new(big.Int).Set(b1)
		rb.Mul(rb, b2)

		binary.BigEndian.PutUint64(scratch, rhi)
		binary.BigEndian.PutUint64(scratch[8:], rhm)
		binary.BigEndian.PutUint64(scratch[16:], rlm)
		binary.BigEndian.PutUint64(scratch[24:], rlo)

		rc := new(big.Int).SetBytes(scratch[:32])
		tt.MustEqual(rb.String(), rc.String(), "failed at index %d", i)
	}
}

var BenchU128In1, BenchU128In2 = U128{hi: 1234, lo: 5678}, U128{hi: 9123, lo: 5678}

func BenchmarkMul128to256(b *testing.B) {
	// fmt.Println(asmtest(math.MaxUint64, math.MaxUint64))
	for i := 0; i < b.N; i++ {
		BenchUint64Result, _, _, _ = mul128to256(BenchU128In1.hi, BenchU128In1.lo, BenchU128In2.hi, BenchU128In2.lo)
	}
}
