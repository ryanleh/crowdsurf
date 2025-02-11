package main

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
	"math"
	"testing"
)

func benchmarkQuery[T m.Elem]() testing.BenchmarkResult {
	// Generate matrix
	prg := rand.NewRandomBufPRG()
	numLimbs := uint64(math.Ceil(float64(*bitsPer) / 32.0))
	matrix := m.New[m.Elem32](*rows*numLimbs, *cols)

	// Build client / server objects
	ctx := crypto.NewContext[T](T(0).Bitlen(), *cols, *pMod)
	key := rand.RandomPRGKey()
	client := &lhe.SimpleClient[T]{}
	server := lhe.MakeSimpleServer(matrix, *bitsPer, ctx, key, mode, true)
	defer client.Free()
	defer server.Free()

	// Initialize the client
	client.Init(server.Hint())

	// Benchmark queries
	result := testing.Benchmark(func(b *testing.B) {
		// Generate client query
		input := []*m.Matrix[T]{m.Rand[T](prg, matrix.Cols(), 1, *pMod)}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
            secrets, _ := client.Query(input)
            for i := range secrets {
                secret := secrets[i].(*lhe.SimpleSecret[T])
                defer secret.Free()
            }

		}
	})
	return result
}
