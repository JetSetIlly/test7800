.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:
    // immediate load and simple artithmetic sequence
    VMOV.F32    S0, #1.5
    VMOV.F32    S1, #2.5

    VADD.F32    S2, S0, S1          // S2 = 4.0
    VSUB.F32    S3, S1, S0          // S3 = 1.0
    VMUL.F32    S4, S0, S1          // S4 = 3.75
    VDIV.F32    S5, S1, S0          // S5 = 1.666...

    // execpted value loaded into R0
    LDR         R0, =literal_1_66
    VLDR        S6, [R0]

    VCMP.F32    S5, S6
    VMRS        APSR_nzcv, FPSCR
    BNE         fail

pass:
    MOVS        R0, #0              // success value of 0
    BKPT        #0x00
    B           .

fail:
    MOVS        R0, #1              // fail value of 1
    BKPT        #0x00
    B           .

.align 4
literal_1_66:
    .word 0x3fd55555                // IEEE-754 approx of 1.6666666

