
.syntax unified
.thumb
.thumb_func
.global _start

.section .text
.align 2

_start:
    MOVS    R0, #0              @ R0 = 0 (pass unless failure)
    MOV     R1, #0x20000000          @ Init memory space

    @---------------------------------------
    @ VLDR Test (via PC-relative load)
    @---------------------------------------
    ADR     R4, lit_3_0
    VLDR    S0, [R4]
    VMOV    R5, S0
    ADR     R6, lit_3_0
    LDR     R6, [R6]
    CMP     R5, R6
    BNE     fail

    @ VSTR and reload
    VSTR    S0, [R1, #4]
    LDR     R5, [R1, #4]
    CMP     R5, R6
    BNE     fail

    @ VADD.F32
    ADR     R4, lit_1_0
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VADD.F32 S2, S0, S1
    ADR     R4, lit_2_0
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VSUB.F32
    ADR     R4, lit_1_0
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VSUB.F32 S2, S0, S1
    ADR     R4, lit_0_0
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VMUL.F32
    VMUL.F32 S2, S0, S1
    VCMP.F32 S2, S0
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VDIV.F32
    VDIV.F32 S2, S0, S1
    VCMP.F32 S2, S0
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VFMA.F32
    ADR     R4, lit_0_5
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VLDR    S2, [R4]
    VFMA.F32 S2, S0, S1
    ADR     R4, lit_0_75
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VFMS.F32
    VFMS.F32 S2, S0, S1
    ADR     R4, lit_0_5
    VLDR    S3, [R4]
    VCMP.F32 S2, S3
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VCMP / VMRS
    VLDR    S0, [R4]
    VLDR    S1, [R4]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    @ VCVT.f32.s32 / VCVT.s32.f32 / VCVT.u32.f32
    MOV     R3, #5
    VMOV    S0, R3
    VCVT.F32.S32 S1, S0
    VCVT.S32.F32 S2, S1
    VMOV    R4, S2
    CMP     R4, #5
    BNE     fail

    VCVT.U32.F32 S2, S1
    VMOV    R4, S2
    CMP     R4, #5
    BNE     fail

    @ VPUSH / VPOP
    ADR     R4, lit_0_5
    VLDR    S0, [R4]
    VPUSH   {S0}
    ADR     R4, lit_0_0
    VLDR    S0, [R4]
    VPOP    {S0}
    ADR     R4, lit_0_5
    VLDR    S1, [R4]
    VCMP.F32 S0, S1
    VMRS    APSR_nzcv, FPSCR
    BNE     fail

    MOVS    R0, #0
    B       pass

fail:
    MOVS    R0, #1

pass:
    BKPT    #0
	B .

.align 4
lit_0_0:   .word 0x00000000  @ 0.0f
lit_0_5:   .word 0x3f000000  @ 0.5f
lit_0_75:  .word 0x3f400000  @ 0.75f
lit_1_0:   .word 0x3f800000  @ 1.0f
lit_2_0:   .word 0x40000000  @ 2.0f
lit_3_0:   .word 0x40400000  @ 3.0f
