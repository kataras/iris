//+build !noasm
//+build !appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

// func matchLenSSE4(a, b []byte, max int) int
TEXT ·matchLenSSE4(SB), 4, $0
	MOVQ  a+0(FP), SI        // RSI: &a
	MOVQ  b+24(FP), DI       // RDI: &b
	MOVQ  max+48(FP), R10    // R10: max
	XORQ  R11, R11           // R11: match length
	MOVQ  R10, R12           // R12: Remainder
	SHRQ  $4, R10            // max / 16
	MOVQ  $16, AX            // Set length for PCMPESTRI
	MOVQ  $16, DX            // Set length for PCMPESTRI
	ANDQ  $15, R12           // max & 15
	TESTQ R10, R10
	JZ    matchlen_verysmall

loopback_matchlen:
	MOVOU (SI), X0 // a[x]
	MOVOU (DI), X1 // b[x]

	// PCMPESTRI $0x18, X1, X0
	// 0x18 = _SIDD_UBYTE_OPS (0x0) | _SIDD_CMP_EQUAL_EACH (0x8) | _SIDD_NEGATIVE_POLARITY (0x10)
	BYTE $0x66; BYTE $0x0f; BYTE $0x3a
	BYTE $0x61; BYTE $0xc1; BYTE $0x18

	JC match_ended

	ADDQ $16, SI
	ADDQ $16, DI
	ADDQ $16, R11

	SUBQ $1, R10
	JNZ  loopback_matchlen

	// Check the remainder using REP CMPSB
matchlen_verysmall:
	TESTQ R12, R12
	JZ    done_matchlen
	MOVQ  R12, CX
	ADDQ  R12, R11

	// Compare CX bytes at [SI] [DI]
	// Subtract one from CX for every match.
	// Terminates when CX is zero (checked pre-compare)
	CLD
	REP; CMPSB

	// Check if last was a match.
	JZ done_matchlen

	// Subtract remanding bytes.
	SUBQ CX, R11
	SUBQ $1, R11
	MOVQ R11, ret+56(FP)
	RET

match_ended:
	ADDQ CX, R11

done_matchlen:
	MOVQ R11, ret+56(FP)
	RET

