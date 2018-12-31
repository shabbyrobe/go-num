#include "textflag.h"

// func quo128bin(uhi, ulo, byhi, bylo uint64, uLeading0, byLeading0 uint) (qhi, qlo uint64)
TEXT ·quo128bin(SB),NOSPLIT,$0
	MOVQ $0, R13 // qhi = 0
	MOVQ $0, R14 // qlo = 0

	// R8: shift counter
	MOVQ uhi+0(FP), R9
	MOVQ ulo+8(FP), R10
	MOVQ byhi+16(FP), R11
	MOVQ bylo+24(FP), R12

	// shift (AX) := byLeading0 - uLeading0
	MOVQ uLeading0+32(FP), AX
	MOVQ byLeading0+40(FP), R8
	SUBQ AX, R8
	CMPQ R8, $64

	JHI shift_gt_64
	JCS shift_lt_64
	// fallthrough if shift == 64

shift_eq_64:
	MOVQ R12, R11 // byhi = bylo
	MOVQ $0,  R12 // bylo = 0
	JMP div_step_start

shift_gt_64:
	MOVQ R12, R11 // byhi = bylo
	MOVQ $0, R12  // bylo = 0
	MOVQ R8, CX
	SUBQ $64, CX
	SHLQ CX, R11  // byhi = bylo << (shift - 64)
	JMP div_step_start

shift_lt_64:
	// byhi, bylo = ((byhi << shift) | (bylo >> (64 - shift))), (bylo << shift)
	MOVQ R8, CX
	SHLQ CX, R11 // byhi = byhi << shift
	MOVQ $64, CX
	SUBQ R8, CX  // '64 - shift' into CX
	MOVQ R12, DX
	SHRQ CX, DX  // 'bylo >> CX' into DX
	ORQ  DX, R11 // byhi = ((byhi << shift) | (bylo >> (64 - shift)))
	SHLQ CX, R12 // bylo = bylo << shift
	JMP div_step_start

div_step_start:

div_step:
	// (qhi, qlo) << 1:
	MOVQ R14, AX
	SHRQ $63, AX // AX = qlo >> 63
	SHLQ $1, R13 // qhi = qhi << 1
	ORQ  AX, R13 // qhi = (qhi << 1) | (qlo >> 63)
	SHLQ $1, R14 // qlo = qlo << 1
	
	CMPQ R9, R11
	JHI  change      // if uhi > byhi, change (u >= by)
	JCS  no_change   // if uhi < byhi, no change

	CMPQ R10, R12    // if uhi == byhi && ulo >= bylo, change (u >= by)
	JCS no_change    // if ulo < bylo, no change

change:
	MOVQ R10, AX // tmpLo := ulo - bylo
	SUBQ R12, AX // ^^^
	SUBQ R11, R9 // uhi = uhi - byhi
	CMPQ R10, AX
	JCC no_carry // if ulo >= tmpLo, no carry.
	SUBQ $1, R9  // if ulo < tmpLo, uhi--

no_carry:
	MOVQ AX, R10 // ulo = tmpLo
	ORQ  $1, R14 // qlo |= 1

no_change:
	// (byhi, bylo) >> 1
	MOVQ R11, AX
	SHLQ $63, AX // AX = byhi << 63
	SHRQ $1, R12 // bylo = bylo >> 1
	ORQ  AX, R12 // bylo = (bylo >> 1) | (byhi << 63)
	SHRQ $1, R11 // byhi = byhi >> 1

	CMPQ R8, $0 // if shift <= 0 
	JLE done    //     break

	SUBQ $1, R8
	JMP div_step

done:
	MOVQ R13, qhi+48(FP)
	MOVQ R14, qlo+56(FP)

	RET
	
