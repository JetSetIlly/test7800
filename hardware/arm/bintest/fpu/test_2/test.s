.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:
    // 1.5 + 2.5 = 4.0 
    VMOV.F32    S0, #1.5
    VMOV.F32    S1, #2.5
    VADD.F32    S2, S0, S1

    LDR         R2, =literal_4_0
    VLDR        S3, [R2]

    VCMP.F32    S2, S3
    VMRS        APSR_nzcv, FPSCR
    BNE         fail_1

    // 5.5 - 3.0 = 2.5
    VMOV.F32    S4, #5.5
    VMOV.F32    S5, #3.0
    VSUB.F32    S6, S4, S5

    LDR         R2, =literal_2_5
    VLDR        S7, [R2]

    VCMP.F32    S6, S7
    VMRS        APSR_nzcv, FPSCR
    BNE         fail_2

    // 1.5 * 2.0 = 3.0
    VMOV.F32    S0, #1.5
    VMOV.F32    S1, #2.0
    VMUL.F32    S2, S0, S1

    LDR         R2, =literal_3_0
    VLDR        S3, [R2]

    VCMP.F32    S2, S3
    VMRS        APSR_nzcv, FPSCR
    BNE         fail_3

    // 6.0 / 2.0 = 3.0
    VMOV.F32    S4, #6.0
    VMOV.F32    S5, #2.0
    VDIV.F32    S6, S4, S5

    LDR         R2, =literal_3_0
    VLDR        S7, [R2]

    VCMP.F32    S6, S7
    VMRS        APSR_nzcv, FPSCR
    BNE         fail_4

pass:
    MOVS        R0, #0
    BKPT        #0x00
    B           .

fail_1:
    MOVS        R0, #1
    BKPT        #0x00
    B           .

fail_2:
    MOVS        R0, #2
    BKPT        #0x00
    B           .

fail_3:
    MOVS        R0, #3
    BKPT        #0x00
    B           .

fail_4:
    MOVS        R0, #4
    BKPT        #0x00
    B           .

.align 4
literal_2_5:  .word 0x40200000      // 2.5
literal_3_0:  .word 0x40400000      // 3.0
literal_4_0:  .word 0x40800000      // 4.0
