package crypto

// NOTE: All parameters chosen assuming gaussian secrets
//
// TODO: May need to be rerun as might have slightly changed
var secretDims = map[uint64]uint64{
	32: 2048,
	64: 4096,
}

// This is hardcoded to 3.2 because SEAL is hardcoded to this stddev. Changing
// this would require recompiling SEAL to use gaussian noise vs. a centered
// binomial.
const errorStdDev = float64(3.2)

// The following maps are for choosing plaintext modulus for the _query_ LWE
// and RLWE schemes based on how many samples there are.
//
// TODO: These parameters needs to be re-run (they're for 64-bit moduli so
// slightly off)
var pMod32 = map[uint64]uint64{
	1 << 7:  3675,
	1 << 8:  3090,
	1 << 9:  2599,
	1 << 10: 2185,
	1 << 11: 1837,
	1 << 12: 1545,
	1 << 13: 1410,
	1 << 14: 1186,
	1 << 15: 997,
	1 << 16: 838,
	1 << 17: 705,
	1 << 18: 593,
	1 << 19: 498,
	1 << 20: 419,
}

// TODO: Do this correctly.
var PModOptions32 = []uint64{
	1 << 15,
	1 << 16,
	1 << 17,
	1 << 18,
	1 << 19,
	1 << 20,
}
var PModOptions64 = []uint64{
	1 << 12,
	1 << 13,
	1 << 14,
	1 << 15,
	1 << 16,
	1 << 17,
	1 << 18,
	1 << 19,
	1 << 20,
}
var PMod32 = pMod32
var PMod64 = pMod64

var pMod64 = map[uint64]uint64{
	//	1 << 7:  240909673,  TODO: Noise analysis seems to be off here
	//	1 << 8:  202580081,
	//	1 << 9:  170348863,
	//	1 << 10: 143245749,
	//	1 << 11: 120454836,
	1 << 12: 101290040,
	1 << 13: 92459714,
	1 << 14: 77749042,
	1 << 15: 65378890,
	1 << 16: 54976875,
	1 << 17: 46229857,
	1 << 18: 38874521,
	1 << 19: 32689445,
	1 << 20: 27488437,
}
