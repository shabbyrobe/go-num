package num

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"
)

type fuzzOp string
type fuzzType string

// This is the equivalent of passing -num.fuzziter=10000 to 'go test':
const fuzzDefaultIterations = 10000

// These ops are all enabled by default. You can instead pass them explicitly
// on the command line like so: '-num.fuzzop=add -num.fuzzop=sub', or you can
// use the short form '-num.fuzzop=add,sub,mul'.
//
// If you add a new op, search for the string 'NEWOP' in this file for all the
// places you need to update.
const (
	fuzzAbs              fuzzOp = "abs"
	fuzzAdd              fuzzOp = "add"
	fuzzAnd              fuzzOp = "and"
	fuzzAndNot           fuzzOp = "andnot"
	fuzzAsFloat64        fuzzOp = "asfloat64"
	fuzzBit              fuzzOp = "bit"
	fuzzBitLen           fuzzOp = "bitlen"
	fuzzCmp              fuzzOp = "cmp"
	fuzzDec              fuzzOp = "dec"
	fuzzEqual            fuzzOp = "equal"
	fuzzFromFloat64      fuzzOp = "fromfloat64"
	fuzzGreaterOrEqualTo fuzzOp = "gte"
	fuzzGreaterThan      fuzzOp = "gt"
	fuzzInc              fuzzOp = "inc"
	fuzzLessOrEqualTo    fuzzOp = "lte"
	fuzzLessThan         fuzzOp = "lt"
	fuzzLsh              fuzzOp = "lsh"
	fuzzMul              fuzzOp = "mul"
	fuzzNeg              fuzzOp = "neg"
	fuzzNot              fuzzOp = "not"
	fuzzOr               fuzzOp = "or"
	fuzzQuo              fuzzOp = "quo"
	fuzzQuoRem           fuzzOp = "quorem"
	fuzzRem              fuzzOp = "rem"
	fuzzRsh              fuzzOp = "rsh"
	fuzzString           fuzzOp = "string"
	fuzzSetBit           fuzzOp = "setbit"
	fuzzSub              fuzzOp = "sub"
	fuzzXor              fuzzOp = "xor"
)

// These types are all enabled by default. You can instead pass them explicitly
// on the command line like so: '-num.fuzztype=u128 -num.fuzztype=i128'
const (
	fuzzTypeU128 fuzzType = "u128"
	fuzzTypeI128 fuzzType = "i128"
)

var allFuzzTypes = []fuzzType{fuzzTypeU128, fuzzTypeI128}

// allFuzzOps are active by default.
//
// NEWOP: Update this list if a NEW op is added otherwise it won't be
// enabled by default.
//
// Please keep this list alphabetised.
var allFuzzOps = []fuzzOp{
	fuzzAbs,
	fuzzAdd,
	fuzzAnd,
	fuzzAndNot,
	fuzzAsFloat64,
	fuzzBit,
	fuzzBitLen,
	fuzzCmp,
	fuzzDec,
	fuzzEqual,
	fuzzFromFloat64,
	fuzzGreaterOrEqualTo,
	fuzzGreaterThan,
	fuzzInc,
	fuzzLessOrEqualTo,
	fuzzLessThan,
	fuzzLsh,
	fuzzMul,
	fuzzNeg,
	fuzzNot,
	fuzzOr,
	fuzzQuo,
	fuzzQuoRem,
	fuzzRem,
	fuzzRsh,
	fuzzSetBit,
	fuzzString,
	fuzzSub,
	fuzzXor,
}

// NEWOP: update this interface if a new op is added.
type fuzzOps interface {
	Name() string // Not an op

	Abs() error
	Add() error
	And() error
	AndNot() error
	AsFloat64() error
	Bit() error
	BitLen() error
	Cmp() error
	Dec() error
	Equal() error
	FromFloat64() error
	GreaterOrEqualTo() error
	GreaterThan() error
	Inc() error
	LessOrEqualTo() error
	LessThan() error
	Lsh() error
	Mul() error
	Neg() error
	Not() error
	Or() error
	Quo() error
	QuoRem() error
	Rem() error
	Rsh() error
	SetBit() error
	String() error
	Sub() error
	Xor() error
}

// classic rando!
type rando struct {
	operands []*big.Int
	rng      *rand.Rand
}

func (r *rando) Operands() []*big.Int { return r.operands }

func (r *rando) Clear() {
	for i := range r.operands {
		r.operands[i] = nil
	}
	r.operands = r.operands[:0]
}

func (r *rando) Intn(n int) int {
	v := int(r.rng.Intn(n))
	r.operands = append(r.operands, new(big.Int).SetInt64(int64(v)))
	return v
}

func (r *rando) Uintn(n int) uint {
	v := uint(r.rng.Intn(n))
	r.operands = append(r.operands, new(big.Int).SetUint64(uint64(v)))
	return v
}

// samesies returns the number of arguments up to n - 1 that should be the same
// for this request. Only used for randos that are 'x2', 'x3', etc.
//
// We need this because the chance of even two random 128-bit operands being
// the same is unfathomable.
func (r *rando) samesies(n int) int {
	const samesiesChance = 0.03
	if r.rng.Float64() < samesiesChance {
		return r.rng.Intn(n)
	}
	return 0
}

func (r *rando) BigU128x2() (b1, b2 *big.Int) {
	b1 = r.BigU128()
	if r.samesies(2) > 0 {
		b2 = new(big.Int).Set(b1)
	} else {
		b2 = r.BigU128()
	}
	r.operands = append(r.operands, b2)
	return b1, b2
}

func (r *rando) BigI128x2() (b1, b2 *big.Int) {
	b1 = r.BigI128()
	if r.samesies(2) > 0 {
		b2 = new(big.Int).Set(b1)
	} else {
		b2 = r.BigI128()
	}
	r.operands = append(r.operands, b2)
	return b1, b2
}

func (r *rando) BigU128() *big.Int {
	var v = new(big.Int)
	bits := r.rng.Intn(129) - 1 // 128 bits, +1 for "0 bits"
	if bits < 0 {
		return v // "-1 bits" == "0"
	} else if bits <= 64 {
		v = v.Rand(r.rng, maxBigUint64)
	} else {
		v = v.Rand(r.rng, maxBigU128)
	}
	v.And(v, masks[bits])
	v.SetBit(v, bits, 1)
	r.operands = append(r.operands, v)
	return v
}

func (r *rando) BigI128() *big.Int {
	neg := r.rng.Intn(2) == 1

	var v = new(big.Int)
	bits := r.rng.Intn(128) - 1 // 127 bits, 1 sign bit (skipped), +1 for "0 bits"
	if bits < 0 {
		return v
	} else if bits <= 64 {
		v = v.Rand(r.rng, maxBigUint64)
	} else {
		v = v.Rand(r.rng, maxBigU128)
	}
	v.And(v, masks[bits])
	v.SetBit(v, bits, 1)
	if neg {
		v.Neg(v)
	}

	r.operands = append(r.operands, v)
	return v
}

// masks contains a pre-calculated set of 128-bit masks for use when generating
// random U128s/I128s. It's used to ensure we generate an even distribution of
// bit sizes.
var masks [128]*big.Int

func init() {
	for i := 0; i < 128; i++ {
		bi := new(big.Int)
		for b := 0; b <= i; b++ {
			bi.SetBit(bi, b, 1)
		}
		masks[i] = bi
	}
}

func checkEqualInt(u int, b int) error {
	if u != b {
		return fmt.Errorf("128(%v) != big(%v)", u, b)
	}
	return nil
}

func checkEqualBool(u bool, b bool) error {
	if u != b {
		return fmt.Errorf("128(%v) != big(%v)", u, b)
	}
	return nil
}

func checkEqualU128(u U128, b *big.Int) error {
	if u.String() != b.String() {
		return fmt.Errorf("u128(%s) != big(%s)", u.String(), b.String())
	}
	return nil
}

func checkEqualString(u fmt.Stringer, b fmt.Stringer) error {
	if u.String() != b.String() {
		return fmt.Errorf("128(%s) != big(%s)", u.String(), b.String())
	}
	return nil
}

func checkFloat(orig *big.Int, result float64, bf *big.Float) error {
	diff := new(big.Float).SetFloat64(result)
	diff.Sub(diff, bf)
	diff.Abs(diff)

	isZero := orig.Cmp(big0) == 0
	if !isZero {
		diff.Quo(diff, bf)
	}

	if (isZero && result != 0) || diff.Abs(diff).Cmp(floatDiffLimit) > 0 {
		return fmt.Errorf("|128(%f) - big(%f)| = %s, > %s", result, bf,
			cleanFloatStr(fmt.Sprintf("%.20f", diff)),
			cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
	}
	return nil
}

func checkEqualI128(i I128, b *big.Int) error {
	if i.String() != b.String() {
		return fmt.Errorf("i128(%s) != big(%s)", i.String(), b.String())
	}
	return nil
}

func TestFuzz(t *testing.T) {
	// fuzzOpsActive comes from the -num.fuzzop flag, in TestMain:
	var runFuzzOps = fuzzOpsActive

	// fuzzTypesActive comes from the -num.fuzzop flag, in TestMain:
	var runFuzzTypes = fuzzTypesActive

	var source = &rando{rng: globalRNG} // Classic rando!
	var totalFailures int

	var fuzzTypes []fuzzOps

	for _, fuzzType := range runFuzzTypes {
		switch fuzzType {
		case fuzzTypeU128:
			fuzzTypes = append(fuzzTypes, &fuzzU128{source: source})
		case fuzzTypeI128:
			fuzzTypes = append(fuzzTypes, &fuzzI128{source: source})
		default:
			panic("unknown fuzz type")
		}
	}

	for _, fuzzImpl := range fuzzTypes {
		var failures = make([]int, len(runFuzzOps))

		for opIdx, op := range runFuzzOps {
			for i := 0; i < fuzzIterations; i++ {
				source.Clear()

				var err error

				// NEWOP: add a new branch here in alphabetical order if a new
				// op is added.
				switch op {
				case fuzzAbs:
					err = fuzzImpl.Abs()
				case fuzzAdd:
					err = fuzzImpl.Add()
				case fuzzAnd:
					err = fuzzImpl.And()
				case fuzzAndNot:
					err = fuzzImpl.AndNot()
				case fuzzAsFloat64:
					err = fuzzImpl.AsFloat64()
				case fuzzBit:
					err = fuzzImpl.Bit()
				case fuzzBitLen:
					err = fuzzImpl.BitLen()
				case fuzzCmp:
					err = fuzzImpl.Cmp()
				case fuzzDec:
					err = fuzzImpl.Dec()
				case fuzzEqual:
					err = fuzzImpl.Equal()
				case fuzzFromFloat64:
					err = fuzzImpl.FromFloat64()
				case fuzzGreaterOrEqualTo:
					err = fuzzImpl.GreaterOrEqualTo()
				case fuzzGreaterThan:
					err = fuzzImpl.GreaterThan()
				case fuzzInc:
					err = fuzzImpl.Inc()
				case fuzzLessOrEqualTo:
					err = fuzzImpl.LessOrEqualTo()
				case fuzzLessThan:
					err = fuzzImpl.LessThan()
				case fuzzLsh:
					err = fuzzImpl.Lsh()
				case fuzzMul:
					err = fuzzImpl.Mul()
				case fuzzNeg:
					err = fuzzImpl.Neg()
				case fuzzNot:
					err = fuzzImpl.Not()
				case fuzzOr:
					err = fuzzImpl.Or()
				case fuzzQuo:
					err = fuzzImpl.Quo()
				case fuzzQuoRem:
					err = fuzzImpl.QuoRem()
				case fuzzRem:
					err = fuzzImpl.Rem()
				case fuzzRsh:
					err = fuzzImpl.Rsh()
				case fuzzSetBit:
					err = fuzzImpl.SetBit()
				case fuzzString:
					err = fuzzImpl.String()
				case fuzzSub:
					err = fuzzImpl.Sub()
				case fuzzXor:
					err = fuzzImpl.Xor()
				default:
					panic(fmt.Errorf("unsupported op %q", op))
				}

				if err != nil {
					failures[opIdx]++
					t.Logf("%s: %s\n", op.Print(source.Operands()...), err)
				}
			}
		}

		for opIdx, cnt := range failures {
			if cnt > 0 {
				totalFailures += cnt
				t.Logf("impl %s, op %s: %d/%d failed", fuzzImpl.Name(), string(runFuzzOps[opIdx]), cnt, fuzzIterations)
			}
		}
	}

	if totalFailures > 0 {
		t.Fail()
	}
}

func (op fuzzOp) Print(operands ...*big.Int) string {
	// NEWOP: please add a human-readale format for your op here; this is used
	// for reporting errors and should show the operation, i.e. "2 + 2".
	//
	// It should be safe to assume the appropriate number of operands are set
	// in 'operands'; if not, it's a bug to be fixed elsewhere.
	switch op {
	case fuzzAsFloat64,
		fuzzFromFloat64,
		fuzzBitLen,
		fuzzString:
		s := strings.TrimRight(op.String(), "()")
		return fmt.Sprintf("%s(%d)", s, operands[0])

	case fuzzSetBit:
		return fmt.Sprintf("%d|(1<<%d)", operands[0], operands[1])

	case fuzzBit:
		return fmt.Sprintf("(%b>>%d)&1", operands[0], operands[1])

	case fuzzInc, fuzzDec:
		return fmt.Sprintf("%d%s", operands[0], op.String())

	case fuzzNeg, fuzzNot:
		return fmt.Sprintf("%s%d", op.String(), operands[0])

	case fuzzAbs:
		return fmt.Sprintf("|%d|", operands[0])

	case fuzzAdd,
		fuzzAnd,
		fuzzAndNot,
		fuzzLessOrEqualTo,
		fuzzLessThan,
		fuzzLsh,
		fuzzMul,
		fuzzOr,
		fuzzQuo,
		fuzzQuoRem,
		fuzzRem,
		fuzzRsh,
		fuzzXor,
		fuzzCmp,
		fuzzEqual,
		fuzzGreaterOrEqualTo,
		fuzzGreaterThan,
		fuzzSub:

		// simple binary case:
		return fmt.Sprintf("%d %s %d", operands[0], op.String(), operands[1])

	default:
		return string(op)
	}
}

func (op fuzzOp) String() string {
	// NEWOP: please add a short string representation of this op, as if
	// the operands were in a sum (if that's possible)
	switch op {
	case fuzzAbs:
		return "|x|"
	case fuzzAdd:
		return "+"
	case fuzzAnd:
		return "&"
	case fuzzAndNot:
		return "&^"
	case fuzzAsFloat64:
		return "float64()"
	case fuzzBit:
		return "bit()"
	case fuzzBitLen:
		return "bitlen()"
	case fuzzCmp:
		return "<=>"
	case fuzzDec:
		return "--"
	case fuzzEqual:
		return "=="
	case fuzzFromFloat64:
		return "fromfloat64()"
	case fuzzGreaterThan:
		return ">"
	case fuzzGreaterOrEqualTo:
		return ">="
	case fuzzInc:
		return "++"
	case fuzzLessThan:
		return "<"
	case fuzzLessOrEqualTo:
		return "<="
	case fuzzLsh:
		return "<<"
	case fuzzMul:
		return "*"
	case fuzzNeg:
		return "-"
	case fuzzNot:
		return "^"
	case fuzzOr:
		return "|"
	case fuzzQuo:
		return "/"
	case fuzzQuoRem:
		return "/%"
	case fuzzRem:
		return "%"
	case fuzzRsh:
		return ">>"
	case fuzzSetBit:
		return "setbit()"
	case fuzzString:
		return "string()"
	case fuzzSub:
		return "-"
	case fuzzXor:
		return "^"
	default:
		return string(op)
	}
}

type fuzzU128 struct {
	source *rando
}

func (f fuzzU128) Name() string { return "u128" }

func (f fuzzU128) Abs() error {
	return nil // Always succeeds!
}

func (f fuzzU128) Inc() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)
	rb := new(big.Int).Add(b1, big1)
	ru := u1.Inc()
	if rb.Cmp(wrapBigU128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	}
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Dec() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)
	rb := new(big.Int).Sub(b1, big1)
	if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(wrapBigU128, rb) // simulate underflow
	}
	ru := u1.Dec()
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Add() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	if rb.Cmp(wrapBigU128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	}
	ru := u1.Add(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Sub() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(wrapBigU128, rb) // simulate underflow
	}
	ru := u1.Sub(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Mul() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)
	for rb.Cmp(wrapBigU128) >= 0 {
		rb = rb.And(rb, maxBigU128) // simulate overflow
	}
	ru := u1.Mul(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Quo() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Quo(b1, b2)
	ru := u1.Quo(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Rem() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Rem(b1, b2)
	ru := u1.Rem(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) QuoRem() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	ruq, rur := u1.QuoRem(u2)
	if err := checkEqualU128(ruq, rbq); err != nil {
		return err
	}
	if err := checkEqualU128(rur, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzU128) Cmp() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualInt(b1.Cmp(b2), u1.Cmp(u2))
}

func (f fuzzU128) Equal() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualBool(b1.Cmp(b2) == 0, u1.Equal(u2))
}

func (f fuzzU128) GreaterThan() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualBool(b1.Cmp(b2) > 0, u1.GreaterThan(u2))
}

func (f fuzzU128) GreaterOrEqualTo() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualBool(b1.Cmp(b2) >= 0, u1.GreaterOrEqualTo(u2))
}

func (f fuzzU128) LessThan() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualBool(b1.Cmp(b2) < 0, u1.LessThan(u2))
}

func (f fuzzU128) LessOrEqualTo() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	return checkEqualBool(b1.Cmp(b2) <= 0, u1.LessOrEqualTo(u2))
}

func (f fuzzU128) And() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).And(b1, b2)
	ru := u1.And(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) AndNot() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).AndNot(b1, b2)
	ru := u1.AndNot(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Or() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).Or(b1, b2)
	ru := u1.Or(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Xor() error {
	b1, b2 := f.source.BigU128x2()
	u1, u2 := accU128FromBigInt(b1), accU128FromBigInt(b2)
	rb := new(big.Int).Xor(b1, b2)
	ru := u1.Xor(u2)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Lsh() error {
	b1 := f.source.BigU128()
	by := f.source.Uintn(128)
	u1 := accU128FromBigInt(b1)
	rb := new(big.Int).Lsh(b1, by)
	rb.And(rb, maxBigU128)
	ru := u1.Lsh(by)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Rsh() error {
	b1 := f.source.BigU128()
	by := f.source.Uintn(128)
	u1 := accU128FromBigInt(b1)
	rb := new(big.Int).Rsh(b1, by)
	ru := u1.Rsh(by)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Neg() error {
	return nil // nothing to do here
}

func (f fuzzU128) AsFloat64() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)
	bf := new(big.Float).SetInt(b1)
	ruf := u1.AsFloat64()
	return checkFloat(b1, ruf, bf)
}

func (f fuzzU128) FromFloat64() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)
	bf1 := new(big.Float).SetInt(b1)
	f1, _ := bf1.Float64()
	r1, inRange := U128FromFloat64(f1)
	if !inRange {
		panic("float out of range") // FIXME: error
	}

	diff := DifferenceU128(u1, r1)

	isZero := b1.Cmp(big0) == 0
	if isZero {
		return checkEqualU128(r1, b1)
	} else {
		difff := new(big.Float).Quo(diff.AsBigFloat(), bf1)
		if difff.Cmp(floatDiffLimit) > 0 {
			return fmt.Errorf("|128(%s) - big(%s)| = %s, > %s", r1, b1,
				cleanFloatStr(fmt.Sprintf("%s", diff)),
				cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
		}
	}
	return nil
}

func (f fuzzU128) String() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)
	return checkEqualString(u1, b1)
}

func (f fuzzU128) SetBit() error {
	b1 := f.source.BigU128()
	bt := int(f.source.Uintn(128))
	bv := f.source.Uintn(2)
	u1 := accU128FromBigInt(b1)

	rb := new(big.Int).SetBit(b1, bt, bv)
	ru := u1.SetBit(bt, bv)
	return checkEqualU128(ru, rb)
}

func (f fuzzU128) Bit() error {
	b1 := f.source.BigU128()
	bt := int(f.source.Uintn(128))
	u1 := accU128FromBigInt(b1)
	return checkEqualInt(int(b1.Bit(bt)), int(u1.Bit(bt)))
}

func (f fuzzU128) Not() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)

	ru := u1.Not()
	if ru.Equal(u1) {
		return fmt.Errorf("input unchanged by Not: %v", u1)
	}
	rd := ru.Not()
	if !rd.Equal(u1) {
		return fmt.Errorf("double-not does not equal input. expected %d, found %d", u1, rd)
	}

	return nil
}

func (f fuzzU128) BitLen() error {
	b1 := f.source.BigU128()
	u1 := accU128FromBigInt(b1)

	rb := b1.BitLen()
	ru := u1.BitLen()

	return checkEqualInt(rb, ru)
}

// NEWOP: func (f fuzzU128) ...() error {}

type fuzzI128 struct {
	source *rando
}

func (f fuzzI128) Name() string { return "i128" }

func (f fuzzI128) Abs() error {
	b1 := f.source.BigI128()
	i1 := accI128FromBigInt(b1)
	rb := new(big.Int).Abs(b1)
	ru := i1.Abs()
	if rb.Cmp(maxBigI128) > 0 { // overflow is possible if you abs minBig128
		rb = new(big.Int).Add(wrapBigU128, rb)
	}
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Inc() error {
	b1 := f.source.BigI128()
	u1 := accI128FromBigInt(b1)
	rb := new(big.Int).Add(b1, big1)
	ru := u1.Inc()
	if rb.Cmp(maxBigI128) > 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	}
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Dec() error {
	b1 := f.source.BigI128()
	u1 := accI128FromBigInt(b1)
	rb := new(big.Int).Sub(b1, big1)
	if rb.Cmp(minBigI128) < 0 {
		rb = new(big.Int).Add(wrapBigU128, rb) // simulate underflow
	}
	ru := u1.Dec()
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Add() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	rb := new(big.Int).Add(b1, b2)
	if rb.Cmp(wrapOverBigI128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	} else if rb.Cmp(wrapUnderBigI128) <= 0 {
		rb = new(big.Int).Add(rb, wrapBigU128) // simulate underflow
	}
	ru := u1.Add(u2)
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Sub() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	rb := new(big.Int).Sub(b1, b2)
	if rb.Cmp(wrapOverBigI128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	} else if rb.Cmp(wrapUnderBigI128) <= 0 {
		rb = new(big.Int).Add(rb, wrapBigU128) // simulate underflow
	}
	ru := u1.Sub(u2)
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Mul() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	rb := new(big.Int).Mul(b1, b2)

	if rb.Cmp(maxBigI128) > 0 {
		// simulate overflow
		gap := new(big.Int)
		gap.Sub(rb, minBigI128)
		r := new(big.Int).Rem(gap, wrapBigU128)
		rb = r.Add(r, minBigI128)
	} else if rb.Cmp(minBigI128) < 0 {
		// simulate underflow
		gap := new(big.Int).Set(rb)
		gap.Sub(maxBigI128, gap)
		r := new(big.Int).Rem(gap, wrapBigU128)
		rb = r.Sub(maxBigI128, r)
	}

	ru := u1.Mul(u2)
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Quo() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Quo(b1, b2)
	ru := u1.Quo(u2)
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) Rem() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}
	rb := new(big.Int).Rem(b1, b2)
	ru := u1.Rem(u2)
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) QuoRem() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	if b2.Cmp(big0) == 0 {
		return nil // Just skip this iteration, we know what happens!
	}

	rbq := new(big.Int).Quo(b1, b2)
	rbr := new(big.Int).Rem(b1, b2)
	ruq, rur := u1.QuoRem(u2)
	if err := checkEqualI128(ruq, rbq); err != nil {
		return err
	}
	if err := checkEqualI128(rur, rbr); err != nil {
		return err
	}
	return nil
}

func (f fuzzI128) Cmp() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualInt(u1.Cmp(u2), b1.Cmp(b2))
}

func (f fuzzI128) Equal() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualBool(u1.Equal(u2), b1.Cmp(b2) == 0)
}

func (f fuzzI128) GreaterThan() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualBool(u1.GreaterThan(u2), b1.Cmp(b2) > 0)
}

func (f fuzzI128) GreaterOrEqualTo() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualBool(u1.GreaterOrEqualTo(u2), b1.Cmp(b2) >= 0)
}

func (f fuzzI128) LessThan() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualBool(u1.LessThan(u2), b1.Cmp(b2) < 0)
}

func (f fuzzI128) LessOrEqualTo() error {
	b1, b2 := f.source.BigI128x2()
	u1, u2 := accI128FromBigInt(b1), accI128FromBigInt(b2)
	return checkEqualBool(u1.LessOrEqualTo(u2), b1.Cmp(b2) <= 0)
}

func (f fuzzI128) AsFloat64() error {
	b1 := f.source.BigI128()
	i1 := accI128FromBigInt(b1)
	bf := new(big.Float).SetInt(b1)
	rif := i1.AsFloat64()
	return checkFloat(b1, rif, bf)
}

func (f fuzzI128) FromFloat64() error {
	b1 := f.source.BigI128()
	i1 := accI128FromBigInt(b1)
	bf1 := new(big.Float).SetInt(b1)
	f1, _ := bf1.Float64()
	r1, inRange := I128FromFloat64(f1)
	if !inRange {
		panic("float out of range") // FIXME: error
	}

	diff := DifferenceI128(i1, r1)

	isZero := b1.Cmp(big0) == 0
	if isZero {
		return checkEqualI128(r1, b1)
	} else {
		difff := new(big.Float).Quo(diff.AsBigFloat(), bf1)
		if difff.Cmp(floatDiffLimit) > 0 {
			return fmt.Errorf("|128(%s) - big(%s)| = %s, > %s", r1, b1,
				cleanFloatStr(fmt.Sprintf("%s", diff)),
				cleanFloatStr(fmt.Sprintf("%.20f", floatDiffLimit)))
		}
	}
	return nil
}

// Bitwise operations on I128 are not supported:
func (f fuzzI128) And() error    { return nil }
func (f fuzzI128) AndNot() error { return nil }
func (f fuzzI128) Or() error     { return nil }
func (f fuzzI128) Xor() error    { return nil }
func (f fuzzI128) Lsh() error    { return nil }
func (f fuzzI128) Rsh() error    { return nil }
func (f fuzzI128) SetBit() error { return nil }
func (f fuzzI128) Bit() error    { return nil }
func (f fuzzI128) BitLen() error { return nil }
func (f fuzzI128) Not() error    { return nil }

func (f fuzzI128) Neg() error {
	b1 := f.source.BigI128()
	u1 := accI128FromBigInt(b1)
	rb := new(big.Int).Neg(b1)
	if rb.Cmp(maxBigI128) > 0 { // overflow is possible if you negate minBig128
		rb = new(big.Int).Add(wrapBigU128, rb)
	}
	ru := u1.Neg()
	return checkEqualI128(ru, rb)
}

func (f fuzzI128) String() error {
	b1 := f.source.BigI128()
	i1 := accI128FromBigInt(b1)
	return checkEqualString(i1, b1)
}

// NEWOP: func (f fuzzI128) ...() error {}
