.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:
    // NaN != NaN
    LDR     R1, =literal_nan
    VLDR    S0, [R1]
    VLDR    S1, [R1]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BNE     pass_1          // NaN != NaN, expect inequality
fail_1:
    MOVS    R0, #1
    BKPT    #0x00
    B       .

pass_1:
    // +INF + 1.0 == +INF
    LDR     R1, =literal_inf
    VLDR    S0, [R1]
    LDR     R2, =literal_1_0
    VLDR    S1, [R2]
    VADD.F32 S2, S0, S1
    VCMP.F32 S2, S0
    VMRS    APSR_nzcv, FPSCR
    BEQ     pass_3
fail_2:
    MOVS    R0, #2
    BKPT    #0x00
    B       .

pass_3:
    // -INF * 2 == -INF
    LDR     R1, =literal_ninf
    VLDR    S0, [R1]
    LDR     R2, =literal_2_0
    VLDR    S1, [R2]
    VMUL.F32 S2, S0, S1
    VCMP.F32 S2, S0
    VMRS    APSR_nzcv, FPSCR
    BEQ     pass_4
fail_3:
    MOVS    R0, #3
    BKPT    #0x00
    B       .

pass_4:
    // +0.0 == -0.0
    LDR     R1, =literal_pzero
    VLDR    S0, [R1]
    LDR     R2, =literal_nzero
    VLDR    S1, [R2]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BEQ     pass_5
fail_4:
    MOVS    R0, #4
    BKPT    #0x00
    B       .

pass_5:
    // 1.0 / 0.0 == +INF
    LDR     R1, =literal_1_0
    VLDR    S0, [R1]
    LDR     R2, =literal_pzero
    VLDR    S1, [R2]
    VDIV.F32 S2, S0, S1
    LDR     R3, =literal_inf
    VLDR    S3, [R3]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BEQ     success
fail_5:
    MOVS    R0, #5
    BKPT    #0x00
    B       .

success:
    MOVS    R0, #0
    BKPT    #0x00
    B       .

.align 4
literal_nan:     .word 0x7FC00000    // Quiet NaN
literal_inf:     .word 0x7F800000    // +Infinity
literal_ninf:    .word 0xFF800000    // -Infinity
literal_1_0:     .word 0x3F800000    // 1.0
literal_2_0:     .word 0x40000000    // 2.0
literal_pzero:   .word 0x00000000    // +0.0
literal_nzero:   .word 0x80000000    // -0.0
