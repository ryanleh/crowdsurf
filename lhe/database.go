package lhe

import (
	"math"
	"math/big"

	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/matrix/gpu"
)

// TODO: Map elements to the range [-p/2, p/2] to control noise growth
// TODO: Add batching for dPIR

// Stores DB metadata
type DBInfo struct {
	N       uint64 // number of DB entries
	BitsPer uint64 // bits per entry
	L       uint64 // DB height
	M       uint64 // DB width

	P  uint64 // Plaintext modulus
	Ne uint64 // Number of Z_p elems per DB entry

	gpu bool // Whether or not we're using a GPU

	// For in-memory db compression
	Squishing uint64
	Cols      uint64
}

// Struct implementing an LWE-compatible database. Underlying data entries may
// be packed into multiple Zp elements.
type DB struct {
	Info *DBInfo
	Data *m.Matrix[m.Elem32]
}

// Create a new DB from `data`
func NewDB(data []m.Elem32, bitsPer uint64, cols uint64, pMod uint64) *DB {
    if pMod == 0 {
        panic("Invalid Pmod")
    }

	// Create DB info
	dbInfo := newDBInfo(uint64(len(data)), bitsPer, cols, pMod)
	db := &DB{
		Info: dbInfo,
		Data: m.Zeros[m.Elem32](dbInfo.L, dbInfo.M),
	}

	// Pack data into `db`
    //
    // TODO: If working w/ local DB and bitsper > 32 we need to repack in
    // column-order layout
	if float64(bitsPer) > math.Log2(float64(pMod)) {
		// If a DB element takes multiple Zp elements to represent we first
		// encode the full value into a bignum and then tightly pack.
		//
		// TODO: Could speed things up by separating out the case when we don't
		// need big nums
		p := big.NewInt(0).SetUint64(dbInfo.P)
		val := big.NewInt(0)
		adder := big.NewInt(0)
		rem := big.NewInt(0)

		// If `bitsPer` > 32 then the data consists of multiple limbs. Assumes
		// big-endian limb-ordering.
		numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
		for i := range db.Info.N {
			// First encode the value into a bignum
			val.SetUint64(uint64(data[i*numLimbs]))
			for j := range uint64(numLimbs - 1) {
				// Shift current values up appropiately before adding next limb
				val.Lsh(val, min(32, uint(bitsPer-32*(j+1))))
				adder.SetUint64(uint64(data[i*numLimbs+j+1]))
				val.Add(val, adder)
			}

			// Pack bignum into multiple Zp elements in little-endian ordering
			for j := range dbInfo.Ne {
				val.DivMod(val, p, rem)
				entry := m.Elem32(uint32(rem.Uint64()))
				db.Data.Set((i/dbInfo.M)*dbInfo.Ne+j, i%dbInfo.M, entry)
			}
		}
	} else {
		p := m.Elem32(dbInfo.P)
		for i := uint64(0); i < uint64(len(data)); i++ {
			for j := uint64(0); j < dbInfo.Ne; j++ {
				// Compute the current DB entry
				entry := data[i]
				for range j {
					entry /= p
				}
				entry %= p

				// Set the corresponding entry in the DB
				db.Data.Set(
					(i/dbInfo.M)*dbInfo.Ne+j,
					i%dbInfo.M,
					entry,
				)
			}
		}
	}

	return db
}

// Create a random database
func RandomDB(num, bitsPer uint64, cols uint64, pMod uint64, prg *rand.BufPRGReader) *DB {
	dbInfo := newDBInfo(num, bitsPer, cols, pMod)
	db := &DB{
		Info: dbInfo,
		Data: m.Rand[m.Elem32](prg, dbInfo.L, dbInfo.M, min(pMod, 1<<bitsPer)),
	}

	// Zero out any overflow columns
	row := dbInfo.L - 1
	for i := num; i < dbInfo.L*dbInfo.M; i++ {
		col := i % dbInfo.M
		db.Data.Set(row, col, 0)
	}

	return db
}

// Copy the database
func (db *DB) Copy() *DB {
	return &DB{
		Info: db.Info,
		Data: db.Data.Copy(),
	}
}

// Compress the database in-place to increase memory-bandwidth
func (db *DB) Squish() {
	// Only compress if parameters are compatible
	if db.Data.CanSquish(db.Info.P) {
		db.Info.Squishing = db.Data.SquishRatio()
		db.Info.Cols = db.Data.Cols()
		db.Data.Squish()
	}
}

/*
   ===== Helper Funcs =====
*/

func newDBInfo(num, bitsPer, cols, pMod uint64) *DBInfo {

	info := &DBInfo{
		BitsPer: bitsPer,
		M:       cols,
		P:       pMod,
		gpu:     gpu.UseGPU(),
	}

	// Compute the number of distinct DB entries
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	if num%numLimbs != 0 {
		panic("Invalid data")
	}
	info.N = num / numLimbs

	// Compute the number of Z_p elements per entry
	var totalElems uint64
    if pMod == 0 {
        info.Ne = numLimbs
        totalElems = info.N
    } else if float64(bitsPer) <= math.Log2(float64(pMod)) {
		info.Ne = 1
		totalElems = info.N
	} else {
		info.Ne = uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMod))))
		totalElems = info.N * info.Ne
	}

	// Set the # of DB rows based on compression amount
	//
	// Entries consisting of multiple elems are stacked _vertically_
	info.L = uint64(math.Ceil(float64(totalElems) / float64(info.M)))
	if info.L%info.Ne != 0 {
		info.L += info.Ne - (info.L % info.Ne)
	}

	return info
}

// Reconstruct an element decomposed into multiple Z_p elements
func (Info *DBInfo) ReconstructElem(vals []m.Elem32) []m.Elem32 {
	var result []m.Elem32
    if Info.P == 0 {
        return vals
    } else if float64(Info.BitsPer) > math.Log2(float64(Info.P)) {
		// If a DB element takes multiple Zp elements to represent we first
		// extract the full value and then repack it in 32-bit chunks. Assumes
		// little-endian ordering of Zp elements.
		p := big.NewInt(0).SetUint64(Info.P)
		val := big.NewInt(0).SetUint64(uint64(vals[len(vals)-1]))
		adder := big.NewInt(0)
		numLimbs := uint64(math.Ceil(float64(Info.BitsPer) / 32.0))

		// Get full value
		for i := range int(Info.Ne - 1) {
			val.Mul(val, p)
			adder.SetUint64(uint64(vals[len(vals)-2-i]))
			val.Add(val, adder)
		}

		// Decompose final result into 32-bit ints. This is a bit messy due to
		// the way that the big package represents numbers
		result = make([]m.Elem32, numLimbs)
		finalBits := Info.BitsPer - (numLimbs-1)*32
		result[numLimbs-1] = m.Elem32(val.Uint64() & ((1 << finalBits) - 1))
		val.Rsh(val, uint(finalBits))
		for i := range numLimbs - 1 {
			result[numLimbs-2-i] = m.Elem32(val.Uint64() & ((1 << 32) - 1))
			val.Rsh(val, 32)
		}
	} else {
		result = make([]m.Elem32, 1)
		p := m.Elem32(Info.P)
		result[0] = vals[0]
		for i := range len(vals) - 1 {
			result[0] *= p
			result[0] += vals[i+1]
		}
	}
	return result
}
