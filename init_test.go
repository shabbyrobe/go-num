package num

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	fuzzIterations  = fuzzDefaultIterations
	fuzzOpsActive   = allFuzzOps
	fuzzTypesActive = allFuzzTypes
	fuzzSeed        int64

	globalRNG *rand.Rand
)

func TestMain(m *testing.M) {
	var ops StringList
	var types StringList

	flag.IntVar(&fuzzIterations, "num.fuzziter", fuzzIterations, "Number of iterations to fuzz each op")
	flag.Int64Var(&fuzzSeed, "num.fuzzseed", fuzzSeed, "Seed the RNG (0 == current nanotime)")
	flag.Var(&ops, "num.fuzzop", "Fuzz op to run (can pass multiple times, or a comma separated list)")
	flag.Var(&types, "num.fuzztype", "Fuzz type (u128, i128) (can pass multiple)")
	flag.Parse()

	if fuzzSeed == 0 {
		fuzzSeed = time.Now().UnixNano()
	}
	globalRNG = rand.New(rand.NewSource(fuzzSeed))

	if len(ops) > 0 {
		fuzzOpsActive = nil
		for _, op := range ops {
			fuzzOpsActive = append(fuzzOpsActive, fuzzOp(op))
		}
	}

	if len(types) > 0 {
		fuzzTypesActive = nil
		for _, t := range types {
			fuzzTypesActive = append(fuzzTypesActive, fuzzType(t))
		}
	}

	log.Println("rando seed:", fuzzSeed) // classic rando!
	log.Println("active ops:", fuzzOpsActive)
	log.Println("iterations:", fuzzIterations)
	log.Println("integer sz:", intSize)

	code := m.Run()
	os.Exit(code)
}

var trimFloatPattern = regexp.MustCompile(`(\.0+$|(\.\d+[1-9])\0+$)`)

func cleanFloatStr(str string) string {
	return trimFloatPattern.ReplaceAllString(str, "$2")
}

func accU128FromBigInt(b *big.Int) U128 {
	u, acc := U128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate conversion to U128 in fuzz tester for %s", b))
	}
	return u
}

func accI128FromBigInt(b *big.Int) I128 {
	i, acc := I128FromBigInt(b)
	if !acc {
		panic(fmt.Errorf("num: inaccurate conversion to I128 in fuzz tester for %s", b))
	}
	return i
}

type StringList []string

func (s StringList) Strings() []string { return s }

func (s *StringList) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *StringList) Set(v string) error {
	vs := strings.Split(v, ",")
	for _, vi := range vs {
		vi = strings.TrimSpace(vi)
		if vi != "" {
			*s = append(*s, vi)
		}
	}
	return nil
}

func randomBigU128(rng *rand.Rand) *big.Int {
	if rng == nil {
		rng = globalRNG
	}

	var v = new(big.Int)
	bits := rng.Intn(129) - 1 // 128 bits, +1 for "0 bits"
	if bits < 0 {
		return v // "-1 bits" == "0"
	} else if bits <= 64 {
		v = v.Rand(rng, maxBigUint64)
	} else {
		v = v.Rand(rng, maxBigU128)
	}
	v.And(v, masks[bits])
	v.SetBit(v, bits, 1)
	return v
}

func simulateBigU128Overflow(rb *big.Int) *big.Int {
	if rb.Cmp(maxBigI128) > 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	} else if rb.Cmp(big0) < 0 {
		rb = new(big.Int).Add(rb, wrapBigU128) // simulate underflow
	}
	return rb
}

func simulateBigI128Overflow(rb *big.Int) *big.Int {
	if rb.Cmp(wrapOverBigI128) >= 0 {
		rb = new(big.Int).Sub(rb, wrapBigU128) // simulate overflow
	} else if rb.Cmp(wrapUnderBigI128) <= 0 {
		rb = new(big.Int).Add(rb, wrapBigU128) // simulate underflow
	}
	return rb
}
