/*
Package num provides uint128 (U128) and int128 (I128) types, implementing
most of the big.Int API.

U128 and I128 are value types; all operations return new values.

Simple example:

	u1 := U128From64(math.MaxUint64)
	u2 := U128From64(math.MaxUint64)
	fmt.Println(u1.Mul(u2))
	// Output: 340282366920938463426481119284349108225

U128 and I128 can be created from a variety of sources:

	U128FromRaw(hi, lo uint64) U128
	U128From64(v uint64) U128
	U128From32(v uint32) U128
	U128From16(v uint16) U128
	U128From8(v uint8) U128
	U128FromString(s string) (out U128, accurate bool, err error)
	U128FromBigInt(v *big.Int) (out U128, accurate bool)
	U128FromFloat32(f float32) (out U128, inRange bool)
	U128FromFloat64(f float64) (out U128, inRange bool)

U128 and I128 support the following formatting and marshalling interfaces:

	- fmt.Formatter
	- mt.Stringer
	- json.Marshaler
	- json.Unmarshaler
	- encoding.TextMarshaler
	- encoding.TextUnmarshaler

*/
package num
