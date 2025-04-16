.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:
    // (((1.5 + 2.5) * 4.0) - 1.0) / 2.0 = 7.5
    LDR     R0, =literal_1_5
    VLDR    S0, [R0]
    LDR     R0, =literal_2_5
    VLDR    S1, [R0]
    VADD.F32 S2, S0, S1

    LDR     R0, =literal_4_0
    VLDR    S3, [R0]
    VMUL.F32 S2, S2, S3

    LDR     R0, =literal_1_0
    VLDR    S4, [R0]
    VSUB.F32 S2, S2, S4

    LDR     R0, =literal_2_0
    VLDR    S5, [R0]
    VDIV.F32 S2, S2, S5

    LDR     R0, =literal_7_5
    VLDR    S6, [R0]
    VCMP.F32 S2, S6
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_1

    // Negative multiply
	// -2.0 * 3.0 = -6.0
    LDR     R0, =literal_neg2_0
    VLDR    S0, [R0]
    LDR     R0, =literal_3_0
    VLDR    S1, [R0]
    VMUL.F32 S2, S0, S1

    LDR     R0, =literal_neg6
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_2

    // Division by zero
	// 1.0 / 0.0 = +inf
    LDR     R0, =literal_1_0
    VLDR    S0, [R0]
    LDR     R0, =literal_0_0
    VLDR    S1, [R0]
    VDIV.F32 S2, S0, S1

    LDR     R0, =literal_posinf
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_3

    // Negative zero test
	// -0.0 + 0.0 = 0.0
    LDR     R0, =literal_neg0
    VLDR    S0, [R0]
    LDR     R0, =literal_0_0
    VLDR    S1, [R0]
    VADD.F32 S2, S0, S1

    LDR     R0, =literal_0_0
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_4

    // NaN propagation
	// NaN + 1.0 = NaN
    LDR     R0, =literal_nan
    VLDR    S0, [R0]
    LDR     R0, =literal_1_0
    VLDR    S1, [R0]
    VADD.F32 S2, S0, S1

    VCMP.F32 S2, S2
    VMRS    APSR_nzcv, FPSCR
    BEQ     fail_5              // NaN is never equal to itself

    // Subnormal addition
	// tiny + tiny = tiny*2
    LDR     R0, =literal_sub1
    VLDR    S0, [R0]
    VLDR    S1, [R0]
    VADD.F32 S2, S0, S1

    LDR     R0, =literal_sub2
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_6

pass:
    MOVS    R0, #0
    BKPT    #0x00
    B       .

fail_1:
    MOVS    R0, #1
    BKPT    #0x00
    B       .

fail_2:
    MOVS    R0, #2
    BKPT    #0x00
    B       .

fail_3:
    MOVS    R0, #3
    BKPT    #0x00
    B       .

fail_4:
    MOVS    R0, #4
    BKPT    #0x00
    B       .

fail_5:
    MOVS    R0, #5
    BKPT    #0x00
    B       .

fail_6:
    MOVS    R0, #6
    BKPT    #0x00
    B       .

.align 4
// Floating-point literal pool
literal_0_0:     .word 0x00000000  // 0.0
literal_1_0:     .word 0x3f800000  // 1.0
literal_1_5:     .word 0x3fc00000  // 1.5
literal_2_0:     .word 0x40000000  // 2.0
literal_2_5:     .word 0x40200000  // 2.5
literal_3_0:     .word 0x40400000  // 3.0
literal_4_0:     .word 0x40800000  // 4.0
literal_7_5:     .word 0x40f00000  // 7.5
literal_neg2_0:  .word 0xc0000000  // -2.0
literal_neg6:    .word 0xc0c00000  // -6.0
literal_neg0:    .word 0x80000000  // -0.0
literal_posinf:  .word 0x7f800000  // +Inf
literal_nan:     .word 0x7fc00001  // Quiet NaN
literal_sub1:    .word 0x00000010  // Very small subnormal
literal_sub2:    .word 0x00000020  // 2x sub1
