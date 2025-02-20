package pbc

import (
	"golang.org/x/exp/maps"
	"math"
	"slices"
	"testing"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

func randInstance[T m.Elem](
	batchSize, bitsPer, rows, cols, pMod uint64,
	mode Mode,
) (*Server[T], *m.Matrix[m.Elem32]) {
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

	server := MakeServer[T](
		matrix,
		batchSize,
		pMod,
		bitsPer,
		prg.GenPRGKey(),
		batching.Balanced,
		mode,
		false,
	)
	return server, matrix
}

func testBatchPIR[T m.Elem](
	t *testing.T,
	client *Client[T],
	server *Server[T],
	matrix *m.Matrix[m.Elem32],
	N, bitsPer, pMod uint64,
) {
	defer client.Free()
	defer server.Free()
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	params := server.Params()

	// Generate client queries
	client.Init(params)

	indices := make([]uint64, params.BatchSize)
	expected := make(map[uint64][]m.Elem32, params.BatchSize)
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	for i := range params.BatchSize {
		indices[i] = prg.Uint64() % N
		dataIdx := indices[i] * numLimbs
		expected[indices[i]] = matrix.Data()[dataIdx : dataIdx+numLimbs]
	}
	keys, queries := client.Query(indices)

	// Answer queries
	answers := server.Answer(queries)
	results := client.Recover(keys, answers)

	// Check that we received the expected number of queries
	switch params.Mode {
	case Hash:
		bsFloat := float64(params.BatchSize)
		recovered := float64(len(results)) / bsFloat
		expected := 1.0 - math.Pow((1.0-1.0/bsFloat), bsFloat)
		if recovered < expected && math.Abs(recovered-expected) > 0.1 {
			t.Fatalf("Recovery error: %v%% vs. %v%%", 100.0*recovered, 100.0*expected)
		}
		//t.Logf("Recovery percentage: %v%%", 100.0*recovered)

	case Cuckoo:
		if len(results) != len(indices) {
			t.Fatalf("Recovery error: %v vs. %v", len(results), len(indices))
		}
	}

	// Check results
	for key, result := range results {
		if !slices.Equal(result, expected[key]) {
			t.Fatalf("Recovery error: %v vs. %v", result, expected[key])
		}
	}
}

func testBasicBatch[T m.Elem](t *testing.T, bitsPer, pMod uint64) {
	dbRows := []uint64{10, 512, 512}
	dbCols := []uint64{800, 256, 512}
	batchSize := uint64(32)
	for i := range dbRows {
		N := dbRows[i] * dbCols[i]
		// Test standard hash bucketing
		server, matrix := randInstance[T](batchSize, bitsPer, dbRows[i], dbCols[i], pMod, Hash)
		testBatchPIR[T](t, &Client[T]{}, server, matrix, N, bitsPer, pMod)

		// Test cuckoo hashing
        server, matrix = randInstance[T](batchSize, bitsPer, dbRows[i], dbCols[i], pMod, Cuckoo)
		testBatchPIR[T](t, &Client[T]{}, server, matrix, N, bitsPer, pMod)
	}
}

func TestBasicBatch32(t *testing.T) {
	testBasicBatch[m.Elem32](t, 8, uint64(1<<8))
	testBasicBatch[m.Elem32](t, 24, uint64(1<<8))
}

func testPBC(t *testing.T, mode Mode) {
	// Generate some random elements in a DB
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	N := uint64(512 * 512)
	batchSize := uint64(64)
	numLimbs := uint64(2)

	db := m.Rand[m.Elem32](prg, N*numLimbs, 1, 0)
	buckets, mapping := EncodeDB(db.Data(), numLimbs, batchSize, mode)

	// Check that each element appears the correct number of times,
	// and the mapping is correct
	for i := range N {
		keyChoices := mapping[uint64(i)]
		if uint64(len(keyChoices)) != mode.NumChoices() {
			t.Fail()
		}

		item := db.Data()[i*numLimbs : (i+1)*numLimbs]
		for bucket, bIndex := range keyChoices {
			idx := uint64(bIndex) * numLimbs
			result := buckets[bucket][idx : idx+numLimbs]
			if !slices.Equal(result, item) {
				t.Fatalf("PBC Failure: %v vs. %v", result, item)
			}
		}
	}

	iters := 10000
	recovered := 0.0
	for range iters {
		// Create a random query batch
		queries := make([]uint64, 0)
		for range batchSize {
			for {
				candidate := prg.Uint64() % N
				if !slices.Contains(queries, candidate) {
					queries = append(queries, candidate)
					break
				}
			}
		}

		// Generate a schedule and check that it's correct
		schedule := GenSchedule(queries, mode, int64(len(buckets)), prg)
		if mode == Cuckoo && (schedule == nil || uint64(len(schedule)) != batchSize) {
			t.Fatalf("Cuckoo Insertion Failed")
		}

		// Invert the schedule
		scheduleInv := make(map[uint64]uint32, len(schedule))
		for bucket, keys := range schedule {
			for _, key := range keys {
				scheduleInv[key] = bucket
			}
		}
		recovered += float64(len(scheduleInv)) / float64(len(queries))

		for _, key := range queries {
			if bucket, contains := scheduleInv[key]; contains {
				if !slices.Contains(maps.Keys(mapping[key]), bucket) {
					t.Fatalf("Invalid PBC Schedule: %v not in %v", bucket, maps.Keys(mapping[key]))
				}
			}
		}
	}

	// If using hash-based bucketing, assert we have received enough queries
	// (this is a lower-bound for a single query to each bucket so very loose)
	if mode == Hash {
		avgRecovered := recovered / float64(iters)
		bsFloat := float64(batchSize)
		avgExpected := 1.0 - math.Pow((1.0-1.0/bsFloat), bsFloat) - 0.01
		if avgRecovered < avgExpected {
			t.Fatalf("Poor scheduling: %0.2f%% vs. %0.2f%%", 100.0*avgRecovered, avgExpected*100.0)
		}
	}
}

func TestHash(t *testing.T) {
	testPBC(t, Hash)
}

func TestCuckoo(t *testing.T) {
	testPBC(t, Cuckoo)
}
