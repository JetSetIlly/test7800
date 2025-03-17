package clocks

const Mhz = 1000000

const (
	NTSC_for_TIA  = 1.193182 * Mhz
	PAL_for_TIA   = 1.182298 * Mhz
	PAL60_for_TIA = 1.182298 * Mhz
	PAL_M_for_TIA = 1.191870 * Mhz
	SECAM_for_TIA = 1.187500 * Mhz
)

const (
	MariaCycles_for_TIA = 6
	MariaCycles         = 4
)

const (
	NTSC_MARIA  = NTSC_for_TIA * MariaCycles_for_TIA // 7.16Mhz
	PAL_MARIA   = PAL_for_TIA * MariaCycles_for_TIA
	PAL60_MARIA = PAL60_for_TIA * MariaCycles_for_TIA
	PAL_M_MARIA = PAL_M_for_TIA * MariaCycles_for_TIA
	SECAM_MARIA = SECAM_for_TIA * MariaCycles_for_TIA
)

const (
	NTSC  = NTSC_MARIA / 4 // 1.79MHz
	PAL   = PAL_MARIA / 4
	PAL60 = PAL60_MARIA / 4
	PAL_M = PAL_M_MARIA / 4
	SECAM = SECAM_MARIA / 4
)
