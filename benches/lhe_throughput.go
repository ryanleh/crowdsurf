package main

import (
	"fmt"
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
	"math"
	"testing"
)

func benchmarkThroughput[T m.Elem]() {
	// Generate random matrix
	prg := rand.NewRandomBufPRG()
	numLimbs := uint64(math.Ceil(float64(*bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, *rows*numLimbs, *cols, *pMod)

	// Build client / server objects
	ctx := crypto.NewContext[T](T(0).Bitlen(), *cols, *pMod)
	key := rand.RandomPRGKey()
	client := &lhe.SimpleClient[T]{}
	server := lhe.MakeSimpleServer(matrix, *bitsPer, ctx, key, mode, true)
	defer client.Free()
	defer server.Free()

	dbSizeGB := math.Log2(float64(*pMod)) * float64(server.DB().Data.Size()) / 8.0 / math.Pow(1024.0, 3)
	fmt.Printf("DB with size: %0.2fGB\n", dbSizeGB)

	// Initialize the client
	client.Init(server.Hint())

	for _, k := range ks {
		// Generate client queries
		inputs := make([]*m.Matrix[T], k)
		for j := range k {
			inputs[j] = m.Rand[T](prg, matrix.Cols(), 1, *pMod)
		}
		keys, queries := client.Query(inputs)
		for i := range keys {
			key := keys[i].(*lhe.SimpleSecret[T])
			key.Free()
		}

		// Set the batch size
		server.SetBatch(k)

		result := testing.Benchmark(func(b *testing.B) {
			// Answer queries
			b.ResetTimer()
			for range b.N {
				server.Answer(queries)
			}
		})
		avgTimeSec := result.T.Seconds() / float64(result.N)
		tputGBperS := (dbSizeGB * float64(k)) / avgTimeSec
		fmt.Printf(
			"Throughput(%d x %d, %d, %d, %d): %0.2f GB/s\n",
			*rows, *cols, k, *pMod, result.N, tputGBperS,
		)
	}
}
