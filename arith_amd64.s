#include "textflag.h"

// func mul64to128(u, v uint64) (hi, lo uint64)
TEXT ·mul64to128(SB),NOSPLIT,$0
	MOVQ x+0(FP), AX
	MULQ y+8(FP)
	MOVQ DX, z1+16(FP)
	MOVQ AX, z0+24(FP)
	RET

// func mul128to256(uhi, ulo, vhi, vlo uint64) (hi, hm, hl, lo uint64)
TEXT ·mul128to256(SB),NOSPLIT,$0
	// hi, hm = mul64to128(uhi, vhi)
	MOVQ uhi+0(FP), AX
	MULQ vhi+16(FP)	
	MOVQ DX, R8 // hi.hi
	MOVQ AX, R9 // hi.lo

	// t = mul64to128(uhi, vlo)
	MOVQ uhi+0(FP), AX
	MULQ vlo+24(FP)	
	MOVQ DX, R10 // t.hi
	MOVQ AX, R11 // t.lo

	// mul64to128(ulo, vlo)
	MOVQ ulo+8(FP), AX
	MULQ vlo+24(FP)	
	MOVQ DX, R12 // lo.hi
	MOVQ AX, R13 // lo.lo

	// lo.hi += t.lo
	ADDQ R11, R12
	JNC lohi_to_tlo_no_overflow
	ADDQ $1, R9 // hi.lo + 1
	JNC lohi_to_tlo_no_overflow
	ADDQ $1, R8 // hi.hi + 1 (carry)

lohi_to_tlo_no_overflow:
	ADDQ R10, R9 // hi.lo += t.hi
	JNC hilo_to_thi_no_overflow
	ADDQ $1, R8 // hi.hi++

hilo_to_thi_no_overflow:
	// t = mul64to128(ulo, vhi)
	MOVQ ulo+8(FP), AX
	MULQ vhi+16(FP)	
	MOVQ DX, R10 // t.hi
	MOVQ AX, R11 // t.lo

	// lo.hi += t.lo
	ADDQ R11, R12
	JNC lohi_to_tlo2_no_overflow
	ADDQ $1, R9 // hi.lo + 1
	JNC lohi_to_tlo2_no_overflow
	ADDQ $1, R8 // hi.hi + 1 (carry)

lohi_to_tlo2_no_overflow:
	ADDQ R10, R9 // hi.lo += t.hi
	JNC hilo_to_thi2_no_overflow
	ADDQ $1, R8 // hi.hi++

hilo_to_thi2_no_overflow:
	MOVQ R8,  hi+32(FP)
	MOVQ R9,  hm+40(FP)
	MOVQ R12, hi+48(FP)
	MOVQ R13, hm+56(FP)
	RET
