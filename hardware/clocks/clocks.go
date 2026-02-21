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
	MariaCycles = 4

	// We know for sure there are four cycles per CPU cycle. So how many cycles when the CPU is running
	// slower?
	//
	// 		1.79Mhz / 1.194182Mhz = 1.50019
	//
	// 	So,
	//
	// 		4 * 1.50019 = 6.00076
	//
	MariaCycles_for_SlowMemory = 6
)

const (
	NTSC_MARIA  = NTSC_for_TIA * MariaCycles_for_SlowMemory // 7.16Mhz
	PAL_MARIA   = PAL_for_TIA * MariaCycles_for_SlowMemory
	PAL60_MARIA = PAL60_for_TIA * MariaCycles_for_SlowMemory
	PAL_M_MARIA = PAL_M_for_TIA * MariaCycles_for_SlowMemory
	SECAM_MARIA = SECAM_for_TIA * MariaCycles_for_SlowMemory
)

const (
	NTSC  = NTSC_MARIA / MariaCycles // 1.79MHz
	PAL   = PAL_MARIA / MariaCycles
	PAL60 = PAL60_MARIA / MariaCycles
	PAL_M = PAL_M_MARIA / MariaCycles
	SECAM = SECAM_MARIA / MariaCycles
)
