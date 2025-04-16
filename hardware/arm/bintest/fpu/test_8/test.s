.syntax unified
.thumb
.thumb_func
.global _start

.section .text
.align 2

_start:
    MOVS    R0, #0
    MOV     R1, #0x20000000

    // Basic VLDR + comparison sanity check
    ADR     R4, lit_3_0
    VLDR    S0, [R4]
    VMOV    R5, S0
    ADR     R6, lit_3_0
    LDR     R6, [R6]
    CMP     R5, R6
    BNE     fail

    // VADD/VSUB with negative value
    ADR     R4, lit_2_0
    VLDR    S0, [R4]
    ADR     R4, lit_m1_0
    VLDR    S1, [R4]
    VADD.F32 S2, S0, S1      // 2.0 + (-1.0) = 1.0
    ADR     R4, lit_1_0
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    VSUB.F32 S2, S3, S1      // 1.0 - (-1.0) = 2.0
    ADR     R4, lit_2_0
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    // VMUL with normalized vector (dot product scenario)
    ADR     R4, lit_0_577
    VLDR    S0, [R4]
    VMUL.F32 S1, S0, S0
    VMUL.F32 S2, S1, S0      // S2 = S0^3 ≈ 0.577^3
    ADR     R4, lit_0_193
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    // VDIV with small epsilon divisor
    ADR     R4, lit_1_0
    VLDR    S0, [R4]
    ADR     R4, lit_eps
    VLDR    S1, [R4]
    VDIV.F32 S2, S0, S1      // S2 = 1.0 / epsilon = large
    ADR     R4, lit_large
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    // VCVT with negative int and float
    MOV     R3, #-42
    VMOV    S0, R3
    VCVT.F32.S32 S1, S0
    VCVT.S32.F32 S2, S1
    VMOV    R4, S2
    CMP     R4, #-42
    BNE     fail

    // VFMA dot product (raycaster-like)
    ADR     R4, lit_1_0
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VMOV    S2, S0
    VFMA.F32 S2, S0, S1      // S2 = 1.0 + 1.0*1.0 = 2.0
    ADR     R4, lit_2_0
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    // Special Values: Inf, -Inf, NaN
    ADR     R4, lit_inf
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    ADR     R4, lit_nan
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BEQ     fail      // NaN != NaN — should NOT be equal

    MOVS    R0, #0
    B       pass

fail:
    MOVS    R0, #1

pass:
    BKPT    #0
    B .

// literal pool 
.align 4
lit_0_0:   .word 0x00000000  // 0.0f
lit_1_0:   .word 0x3f800000  // 1.0f
lit_2_0:   .word 0x40000000  // 2.0f
lit_3_0:   .word 0x40400000  // 3.0f
lit_m1_0:  .word 0xbf800000  // -1.0f
lit_0_193: .word 0x3e47cb6f  // ~0.193 (0.577^3)
lit_0_577: .word 0x3f147ae1  // ~0.577
lit_0_75:  .word 0x3f400000  // 0.75f
lit_0_5:   .word 0x3f000000  // 0.5f
lit_eps:   .word 0x33d6bf95  // ~1.0e-7f
lit_large: .word 0x4b189680  // ~1.0e7f
lit_inf:   .word 0x7f800000  // +inf
lit_nan:   .word 0x7fc00000  // quiet NaN
