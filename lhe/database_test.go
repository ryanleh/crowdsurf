package lhe

import (
	"math"
	"slices"
	"testing"

	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
)

func (db *DB) getElem(i uint64) []m.Elem32 {
	// Extract the relevant elements corresponding to index `i`
	vals := make([]m.Elem32, db.Info.Ne)
	row := db.Info.Ne * (i / db.Info.M)
	col := i % db.Data.Cols()
	for j := range db.Info.Ne {
		vals[j] = db.Data.Get(row+j, col)
	}
	return db.Info.ReconstructElem(vals)
}

func testDBInit(t *testing.T, data []m.Elem32, bitsPer, cols, pMod uint64) *DB {
	db := NewDB(data, bitsPer, cols, pMod, false)

	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	for i := range uint64(len(data)) / numLimbs {
		result := db.getElem(i)
		expected := data[i*numLimbs : (i+1)*numLimbs]
		if !slices.Equal(result, expected) {
			t.Fatalf("DB Packing failure (%v): %v != %v", i, result, expected)
		}
	}

	return db
}

// Each database entry is ~1 Zp element
func TestDBMediumEntries(t *testing.T) {
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	matrix := m.Rand[m.Elem32](prg, 1024, 1024, 1<<9)
	db := testDBInit(t, matrix.Data(), 9, matrix.Cols(), 1<<10)
	if db.Info.Ne != 1 {
		t.Fail()
	}

}

// Each database entry is >1 Zp elements
func TestDBLargeEntries(t *testing.T) {
	prg := rand.NewBufPRG(rand.NewPRG(&key))

	// Check a non-power-of-two number of entry bits
	matrix := m.Rand[m.Elem32](prg, 512*3, 512, 0)
	testDBInit(t, matrix.Data(), 96, matrix.Cols(), 1<<10)

	// 128-byte entries
	matrix = m.Rand[m.Elem32](prg, 512*32, 512, 0)
	testDBInit(t, matrix.Data(), 1024, matrix.Cols(), 1<<10)
}
