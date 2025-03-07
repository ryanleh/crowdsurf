package lhe

import (
	"math"
	"slices"
	"testing"

	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

// ------- Tests -------

func randInstance[T m.Elem](
	scheme LHEType,
	bitsPer, rows, cols, pMod uint64,
	bench bool,
) (Client[T], Server[T], *m.Matrix[m.Elem32]) {
	if bitsPer > 63 || bitsPer%32 == 0 {
		panic("Unsupported entry bits")
	}

	// Generate random matrix
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, rows*numLimbs, cols, 0)

	// Truncate the last element of each entry to match `bitsPer`
	truncateMod := 1 << (bitsPer - (numLimbs-1)*32)
	for i := range rows * cols {
		matrix.Data()[(i+1)*numLimbs-1] %= m.Elem32(truncateMod)
	}

	// Build client / server objects
	var client Client[T]
	var server Server[T]
	ctx := crypto.NewContext[T](T(0).Bitlen(), cols, pMod)
	switch scheme {
	case Local:
		client = &LocalClient[T]{}
		server = MakeLocalServer[T](matrix, bitsPer)

	case Simple:
		client = &SimpleClient[T]{}
		server = MakeSimpleServer[T](matrix, bitsPer, ctx, &key, None, false, bench)

	case SimpleHybrid:
		client = &SimpleClient[T]{}
		server = MakeSimpleServer[T](matrix, bitsPer, ctx, &key, Hybrid, false, bench)

	default:
		panic("Invalid client type")
	}

	// Initialize the client
	client.Init(server.Hint())
	return client, server, matrix
}

// TODO: More iters here + smaller DB
func testLHEHelper[T m.Elem](
	t *testing.T,
	client Client[T],
	server Server[T],
	matrix *m.Matrix[m.Elem32],
	batchSize uint64,
) {
	defer client.Free()
	defer server.Free()
	dbInfo := client.DBInfo()
	server.SetBatch(batchSize)
	prg := rand.NewBufPRG(rand.NewPRG(&key))

	for range 1 {
		// Generate client queries
		indices := make([]uint64, batchSize)
		inputs := make([]*m.Matrix[T], batchSize)
		expected := make([][]m.Elem32, batchSize)
		numLimbs := uint64(math.Ceil(float64(dbInfo.BitsPer) / 32.0))
		for i := range inputs {
			indices[i] = prg.Uint64() % dbInfo.N
			inputs[i] = m.New[T](matrix.Cols(), 1)
			inputs[i].Data()[indices[i]%matrix.Cols()] = 1

			dataIdx := indices[i] * numLimbs
			expected[i] = matrix.Data()[dataIdx : dataIdx+numLimbs]
		}
		keys, queries := client.Query(inputs)

		// Answer queries
		answers := server.Answer(queries)
		results := client.Recover(keys, answers)

		// Check results
		for i := range results {
			rawResult := make([]m.Elem32, 0)
			index := dbInfo.Ne * (indices[i] / dbInfo.M)
			for j := range dbInfo.Ne {
				rawResult = append(rawResult, m.Elem32(results[i].Data()[index+j]))
			}
			result := dbInfo.ReconstructElem(rawResult)
			if !slices.Equal(result, expected[i]) {
				t.Fatalf("Failure @ %d: %v vs. %v", i, result, expected[i])
			}
		}
	}
}

func testLHE[T m.Elem](t *testing.T, bitsPer, pMod uint64) {
	dbRows := []uint64{13, 10, 512, 512}
	dbCols := []uint64{15, 200, 256, 512}
	batchSize := uint64(3)
	for i := range dbRows {
		// Test SimplePIR-based LHE
		c, s, m := randInstance[T](Simple, bitsPer, dbRows[i], dbCols[i], pMod, false)
		testLHEHelper[T](t, c, s, m, batchSize)

        c, s, m = randInstance[T](SimpleHybrid, bitsPer, dbRows[i], dbCols[i], pMod, false)
		testLHEHelper[T](t, c, s, m, batchSize)
	}
}

func TestSmallEntries32(t *testing.T) {
	testLHE[m.Elem32](t, 7, uint64(1<<8))
}

func TestSmallEntries64(t *testing.T) {
	testLHE[m.Elem64](t, 15, uint64(1<<16))
}

func TestLargeEntries32(t *testing.T) {
	testLHE[m.Elem32](t, 24, uint64(1<<8))
	testLHE[m.Elem32](t, 48, uint64(1<<8))
}

func TestLargeEntries64(t *testing.T) {
	testLHE[m.Elem64](t, 24, uint64(1<<16))
	testLHE[m.Elem64](t, 48, uint64(1<<16))
}

// ------- Latency Benches -------

func bench[T m.Elem](
	b *testing.B,
	batchSize int,
	rows, cols, bitsPer, pMod uint64,
	scheme LHEType,
	benchType int,
) {
	prg := rand.NewRandomBufPRG()

	// Generate random instance
	client, server, _ := randInstance[T](scheme, bitsPer, rows, cols, pMod, true)
	defer client.Free()
	defer server.Free()

	// Build dummy inputs
	inputs := make([]*m.Matrix[T], batchSize)
	for i := range inputs {
		inputs[i] = m.Rand[T](prg, cols, 1, pMod)
	}

	switch benchType {
	case 0:
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client.Query(inputs)
		}
	case 1:
		_, queries := client.Query(inputs)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			server.Answer(queries)
		}
	case 2:
		keys, queries := client.Query(inputs)
		answers := server.Answer(queries)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client.Recover(keys, answers)
		}
	}
}

var benchRows = uint64(4096)
var benchCols = uint64(4096)
var batchSize = 3

// Query
func BenchmarkSimpleQuery32(b *testing.B) {
	bench[m.Elem32](b, batchSize, benchRows, benchCols, 7, 1<<7, Simple, 0)
}

func BenchmarkSimpleHybridQuery32(b *testing.B) {
	bench[m.Elem32](b, batchSize, benchRows, benchCols, 7, 1<<7, SimpleHybrid, 0)
}

func BenchmarkSimpleQuery64(b *testing.B) {
	bench[m.Elem64](b, batchSize, benchRows, benchCols, 16, 1<<16, Simple, 0)
}

func BenchmarkSimpleHybridQuery64(b *testing.B) {
	bench[m.Elem64](b, batchSize, benchRows, benchCols, 16, 1<<16, SimpleHybrid, 0)
}

// Answer
func BenchmarkSimpleAnswer32(b *testing.B) {
	bench[m.Elem32](b, batchSize, benchRows, benchCols, 7, 1<<7, Simple, 1)
}

func BenchmarkSimpleHybridAnswer32(b *testing.B) {
	bench[m.Elem32](b, batchSize, benchRows, benchCols, 7, 1<<7, SimpleHybrid, 1)
}

func BenchmarkSimpleAnswer64(b *testing.B) {
	bench[m.Elem64](b, batchSize, benchRows, benchCols, 16, 1<<16, Simple, 1)
}

func BenchmarkSimpleHybridAnswer64(b *testing.B) {
	bench[m.Elem64](b, batchSize, benchRows, benchCols, 16, 1<<16, SimpleHybrid, 1)
}

// Recover
func BenchmarkSimpleRecover32(b *testing.B) {
	bench[m.Elem32](b, batchSize, benchRows, benchCols, 7, 1<<7, SimpleHybrid, 2)
}

func BenchmarkSimpleRecover64(b *testing.B) {
	bench[m.Elem64](b, batchSize, benchRows, benchCols, 16, 1<<16, SimpleHybrid, 2)
}
