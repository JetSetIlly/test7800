
.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:
    // large * Small = moderate
    LDR     R0, =literal_large
    VLDR    S0, [R0]
    LDR     R0, =literal_small
    VLDR    S1, [R0]
    VMUL.F32 S2, S0, S1

    LDR     R0, =literal_expect1
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_1

    // add + subtract cancelation (loss of precision)
    LDR     R0, =literal_1000000
    VLDR    S0, [R0]
    LDR     R0, =literal_0_000001
    VLDR    S1, [R0]
    VADD.F32 S2, S0, S1
    VSUB.F32 S2, S2, S0

    LDR     R0, =literal_expect2
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_2

    // multiplication of pi and e
    LDR     R0, =literal_pi
    VLDR    S0, [R0]
    LDR     R0, =literal_e
    VLDR    S1, [R0]
    VMUL.F32 S2, S0, S1         // pi * e

    LDR     R0, =literal_ln2
    VLDR    S3, [R0]
    VADD.F32 S2, S2, S3         // + ln2

    LDR     R0, =literal_expect3
    VLDR    S4, [R0]
    VCMP.F32 S2, S4
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_3

    // multiply of subnormals
    LDR     R0, =literal_submin
    VLDR    S0, [R0]
    VLDR    S1, [R0]            // submin * submin
    VMUL.F32 S2, S0, S1

    LDR     R0, =literal_expect4
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_4

    // inexact result (non-representable decimal)
    LDR     R0, =literal_10
    VLDR    S0, [R0]
    LDR     R0, =literal_3
    VLDR    S1, [R0]
    VDIV.F32 S2, S0, S1

    LDR     R0, =literal_expect5
    VLDR    S3, [R0]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_5

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

.align 4

// === Literal pool ===

literal_large:      .word 0x4f000000     // 2.1474836e+09
literal_small:      .word 0x32000000     // 7.450581e-09
literal_expect1:    .word 0x41800000     // 16

literal_1000000:    .word 0x49742400     // 1e+06
literal_0_000001:   .word 0x36900000     // 4.2915344e-06
literal_expect2:	.word 0x00000000     // 0.0

literal_pi:         .word 0x40490fdb     // 3.1415927
literal_e:          .word 0x402df854     // 2.7182817
literal_ln2:        .word 0x3f317218     // 0.6931472
literal_expect3:    .word 0x4113b9e2     // 9.232882 

literal_submin:     .word 0x00000010     // Tiny subnormal
literal_expect4:    .word 0x00000000     // Even tinier (underflow zone)

literal_10:         .word 0x41200000     // 10.0
literal_3:          .word 0x40400000     // 3.0
literal_expect5:    .word 0x40555555     // 3.333... (rounded repr)
