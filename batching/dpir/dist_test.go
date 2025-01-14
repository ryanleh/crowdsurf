package dpir

import (
	"math"
	r "math/rand"
	"slices"
	"testing"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

// ------- Tests -------
func randInstance[T m.Elem](
	cutoff, load uint64,
    alpha float64,
	bitsPer, rows, cols, pMod uint64,
	types []PirType,
) (*Server[T], *m.Matrix[m.Elem32]) {
	if bitsPer > 63 || bitsPer%32 == 0 {
		panic("Unsupported entry bits")
	}

	// Generate a random matrix
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, rows*numLimbs, cols, 0)

	// Truncate the last element of each entry to match `bitsPer`
	truncateMod := 1 << (bitsPer - (numLimbs-1)*32)
	for i := range rows * cols {
		matrix.Data()[(i+1)*numLimbs-1] %= m.Elem32(truncateMod)
	}

	server := MakeServer[T](matrix, cutoff, alpha, load, bitsPer, pMod, types, batching.Balanced, &key, false)
	return server, matrix
}

func testBucketing[T m.Elem](
	t *testing.T,
	client *Client[T],
	server *Server[T],
	matrix *m.Matrix[m.Elem32],
	bitsPer, pMod uint64,
) {
	defer client.Free()
	defer server.Free()
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	params := server.Params()

    // Run query / answer a number of times
    //
    // TODO: Might need to make sure we free stuff here
	client.Init(params)
    iters := 50
    correct := 0.0 
    for range iters {
        // Generate client queries. Generate one less than the batch size to
        // test dummy queries
        numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
        indices := make([]uint64, params.Load-1)
        for i := range len(indices) {
            // Generate queries according to step distribution
            if r.Float64() < 1 - params.Alpha {
                indices[i] = prg.Uint64() % params.Cutoff
            } else {
                indices[i] = prg.Uint64() % (matrix.Size() / numLimbs)
            }
        }
        secret, query := client.Query(indices)

        // Answer queries
        answer := server.Answer(query)
        results := client.Recover(secret, answer)
        
        // Check results
        for index, result := range results {
            dataIdx := index * numLimbs
            expected := matrix.Data()[dataIdx : dataIdx+numLimbs]
            if !slices.Equal(result, expected) {
                t.Fatalf("Recovery error @ %v: %v vs. %v", index, result, expected)
            }
        }
        correct += float64(len(results) + 1)
    }

    percentCorrect := correct / float64(iters * int(params.Load))
    if percentCorrect < 0.88 {
        t.Fatalf("Error rate too high: %v vs. %v", percentCorrect, 0.9)
    }
}

func testBasicSplit[T m.Elem](t *testing.T, bitsPer, pMod uint64) {
	rows := []uint64{512, 256}
	cols := []uint64{512, 512}
	cutoffs := []uint64{ 26215, 13108 }
	loads := []uint64{10, 10}
    alpha := 0.1
	types := [][]PirType{
		{Simple, PBC},
		{SimpleHybrid, PBCAngel},
	}
	for i := range rows {
		server, matrix := randInstance[T](cutoffs[i], loads[i], alpha, bitsPer, rows[i], cols[i], pMod, types[i])
		testBucketing[T](t, &Client[T]{}, server, matrix, bitsPer, pMod)
	}
}

// Tests 1/10 of the database queried with probability 90%
func TestBasicSplit32(t *testing.T) {
	testBasicSplit[m.Elem32](t, 8, uint64(1<<8))
	testBasicSplit[m.Elem32](t, 24, uint64(1<<8))
	testBasicSplit[m.Elem32](t, 48, uint64(1<<8))
}

func TestBasicSplit64(t *testing.T) {
	testBasicSplit[m.Elem64](t, 16, uint64(1<<16))
	testBasicSplit[m.Elem64](t, 24, uint64(1<<16))
	testBasicSplit[m.Elem64](t, 48, uint64(1<<16))
}
