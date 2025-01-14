package batching

import (
	"math"
	"sort"

	"github.com/ryanleh/secure-inference/crypto"
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/matrix/gpu"
)

// TODO: Create an abstraction that captures both dPIR and batch codes

// An enum representing the different ways to pack buckets
type Packing int

const (
	Balanced Packing = iota // Balance comm. and storage overheads
	Comm                    // Keep comm. unchanged
	Storage                 // Keep storage unchanged
)

// Helper struct for sorting bucket sizes
type PermBucket struct {
	Entries uint64
	Idx     uint64
}

// Get the set of plaintext moduli valid for the given architecture and modulus
// size
//
// TODO: Some of this should probably be in `crypto/`
func getOptions[T m.Elem]() ([]uint64, map[uint64]uint64) {
	// TODO: Some of this should probably be in `crypto/`
	//
	// Find the largest pMod which supports square dimensions
	pMods := crypto.PMod32
	options := crypto.PModOptions32
	if T(0).Bitlen() == 64 {
		pMods = crypto.PMod64
		options = crypto.PModOptions64
	} else if gpu.UseGPU() {
		options = crypto.PModOptions64
	}
	return options, pMods
}

// Approximate square dimensions + plaintext modulus for the given set of elements.
// Returns rows, cols, pMod
func ApproxSquareDims[T m.Elem](entries, bitsPer uint64) (uint64, uint64, uint64) {
	// Find the largest pMod which supports square dimensions
	options, pMods := getOptions[T]()

	for _, cutoff := range options {
		pMod := pMods[cutoff]
		elemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMod))))
		size := entries * elemWidth

		// Remap the bucket into a matrix
		rows := uint64(math.Floor(math.Sqrt(float64(size))))
		for rows%elemWidth != 0 {
			rows += 1
		}
		cols := uint64(math.Ceil(float64(size) / float64(rows)))
		if cols <= cutoff {
			return rows, cols, pMod
		}
	}
	panic("Invalid LWE parameters (?)")
}

// Approximate square dimensions + plaintext modulus for the given set of elements where
// the columns are constrainted by a given amount
func ApproxColConstraint[T m.Elem](entries, bitsPer, maxCols uint64) (uint64, uint64, uint64) {
	// If square dimensions satisfy the constraint then we're done
	rows, cols, pMod := ApproxSquareDims[T](entries, bitsPer)
	if cols <= maxCols {
		return rows, cols, pMod
	}

	// Find the largest pMod which supports the constraints
	options, pMods := getOptions[T]()

	cols = maxCols
	for _, cutoff := range options {
		if cols <= cutoff {
			pMod := pMods[cutoff]
			elemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMod))))
			size := entries * elemWidth

			rows = uint64(math.Ceil(float64(size) / float64(cols)))
			for rows%elemWidth != 0 {
				rows += 1
			}
			return rows, cols, pMod
		}
	}
	panic("Invalid LWE parameters (?)")
}

// Approximate square dimensions + plaintext modulus for the given set of elements where
// the rows are constrainted by a given amount
func ApproxRowConstraint[T m.Elem](entries, bitsPer, maxRows uint64) (uint64, uint64, uint64) {
	// If square dimensions satisfy the constraint then we're done
	rows, cols, pMod := ApproxSquareDims[T](entries, bitsPer)
	if rows <= maxRows {
		return rows, cols, pMod
	}

	// Find the largest pMod which supports the constraints
	options, pMods := getOptions[T]()

	for _, cutoff := range options {
		pMod := pMods[cutoff]
		elemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMod))))
		size := entries * elemWidth

		rows = maxRows
		for rows%elemWidth != 0 {
			rows -= 1
		}
		cols := uint64(math.Ceil(float64(size) / float64(rows)))
		if cols <= cutoff {
			return rows, cols, pMod
		}
	}
	panic("Invalid LWE parameters (?)")
}

// Computes matrix / plaintext parameters for a given set of buckets to balance
// communication and storage.
//
// The output corresponds to parameters _after_ DB encoding, so you might need to
// multiply by `numLimbs / elemWidth` if you're allocating based on this
//
// TODO: Clean up this API at some point (e.g. have some form of config file we
// pass around)
func PackingDims[T m.Elem](
	sizes []uint64,
	bitsPer, origRows, origCols, origP uint64,
	method Packing,
) ([]uint64, []uint64, []uint64) {
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))

	// We assumes `sizes` refers to un-encoded DB. Get the number of rows after theoretically encoding the
	// original matrix into Zp Elements.
	origElemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(origP))))
	origRows = (origRows / numLimbs) * origElemWidth

	// Sort the buckets by size. This is for the greedy solver we have below.
	sortedSizes := make([]PermBucket, len(sizes))
	for i := range uint64(len(sizes)) {
		sortedSizes[i] = PermBucket{Entries: sizes[i] / numLimbs, Idx: i}
	}
	sort.Slice(sortedSizes, func(i, j int) bool { return sortedSizes[i].Entries < sortedSizes[j].Entries })

	rows := make([]uint64, len(sizes))
	cols := make([]uint64, len(sizes))
	pMods := make([]uint64, len(sizes))
	free := uint64(0)
	for _, permSize := range sortedSizes {
		// Get the size of the bucket after encoding into Zp elements
		i := permSize.Idx

		switch method {
		case Balanced:
			// To balance, each bucket should be square
			rows[i], cols[i], pMods[i] = ApproxSquareDims[T](permSize.Entries, bitsPer)
		case Comm:
			// To keep communication unchanged, the total number of columns
			// should not exceed origCols. We use a simple greedy algorithm to
			// satisfy this.
			//
			// Maximum columns for this bucket is its own allocation + whatever
			// previous buckets didn't use
			maxCols := uint64(math.Floor(float64(origCols)/float64(len(sizes)))) + free
			rows[i], cols[i], pMods[i] = ApproxColConstraint[T](permSize.Entries, bitsPer, maxCols)
			free = maxCols - cols[i]
		case Storage:
			// To keep communication unchanged, the total number of rows
			// should not exceed origRows. We use a simple greedy algorithm to
			// satisfy this.
			//
			// Maximum rows for this bucket is its own allocation + whatever
			// previous buckets didn't use
			maxRows := uint64(math.Floor(float64(origRows)/float64(len(sizes)))) + free
			rows[i], cols[i], pMods[i] = ApproxRowConstraint[T](permSize.Entries, bitsPer, maxRows)
			free = maxRows - rows[i]
		}
	}
	return rows, cols, pMods
}
