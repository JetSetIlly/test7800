
.syntax unified
.thumb
.global _start

.section .text
.align 2

_start:

    // --- Safe way to set the stack pointer (SP) ---
    LDR     R0, =0x20001000
    MOV     SP, R0

    // --- Step 1: Load constants into S0-S3 ---
    LDR     R1, =literal_1_0
    VLDR    S0, [R1]
    LDR     R1, =literal_2_0
    VLDR    S1, [R1]
    LDR     R1, =literal_3_0
    VLDR    S2, [R1]
    LDR     R1, =literal_4_0
    VLDR    S3, [R1]

    // --- Step 2: VPUSH S0-S3 ---
    VPUSH   {S0-S3}

    // --- Step 3: Clobber S0-S3 with 0s ---
    LDR     R1, =literal_0_0
    VLDR    S0, [R1]
    VLDR    S1, [R1]
    VLDR    S2, [R1]
    VLDR    S3, [R1]

    // --- Step 4: VPOP S0-S3 ---
    VPOP    {S0-S3}

    // --- Step 5: Compare values back to originals ---
    LDR     R1, =literal_1_0
    VLDR    S4, [R1]
    VCMP.F32 S0, S4
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_1

    LDR     R1, =literal_2_0
    VLDR    S4, [R1]
    VCMP.F32 S1, S4
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_2

    LDR     R1, =literal_3_0
    VLDR    S4, [R1]
    VCMP.F32 S2, S4
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_3

    LDR     R1, =literal_4_0
    VLDR    S4, [R1]
    VCMP.F32 S3, S4
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_4

    // --- Step 6: VMOV between core and FPU ---
    LDR     R0, =0x3F800000       // 1.0 in bits
    VMOV    S5, R0
    LDR     R1, =literal_1_0
    VLDR    S6, [R1]
    VCMP.F32 S5, S6
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_5

    // Move from FPU to core
    VMOV    R2, S6
    LDR     R3, =0x3F800000
    CMP     R2, R3
    BNE     fail_6

    // --- Step 7: VSTM and VLDM ---
    LDR     R0, =0x20000000
    VSTM    R0!, {S0-S3}
    LDR     R0, =0x20000000
    VLDM    R0!, {S7-S10}

    // Compare round-trip
    VCMP.F32 S0, S7
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_7
    VCMP.F32 S1, S8
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_8
    VCMP.F32 S2, S9
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_9
    VCMP.F32 S3, S10
    VMRS    APSR_nzcv, FPSCR
    BNE     fail_10

    // --- All passed ---
    MOVS    R0, #0
    BKPT    #0x00
    B       .

fail_1:  MOVS R0, #1   ; BKPT #0x00 ; B .  
fail_2:  MOVS R0, #2   ; BKPT #0x00 ; B .
fail_3:  MOVS R0, #3   ; BKPT #0x00 ; B .
fail_4:  MOVS R0, #4   ; BKPT #0x00 ; B .
fail_5:  MOVS R0, #5   ; BKPT #0x00 ; B .
fail_6:  MOVS R0, #6   ; BKPT #0x00 ; B .
fail_7:  MOVS R0, #7   ; BKPT #0x00 ; B .
fail_8:  MOVS R0, #8   ; BKPT #0x00 ; B .
fail_9:  MOVS R0, #9   ; BKPT #0x00 ; B .
fail_10: MOVS R0, #10  ; BKPT #0x00 ; B .

.align 4
literal_0_0:    .word 0x00000000    // 0.0
literal_1_0:    .word 0x3F800000    // 1.0
literal_2_0:    .word 0x40000000    // 2.0
literal_3_0:    .word 0x40400000    // 3.0
literal_4_0:    .word 0x40800000    // 4.0
