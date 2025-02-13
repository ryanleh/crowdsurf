package main

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/matrix/gpu"
	"math"
	"testing"
)

func benchmarkPreprocessing[T m.Elem]() testing.BenchmarkResult {
	// Generate random DB
	prg := rand.NewRandomBufPRG()
	numLimbs := uint64(math.Ceil(float64(*bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, *rows*numLimbs, *cols, 0)
	ctx := crypto.NewContext[T](T(0).Bitlen(), *cols, *pMod)
	db := lhe.NewDB(matrix.Data(), *bitsPer, ctx.Params.M, ctx.Params.P)
    
	var result testing.BenchmarkResult
	if mode == lhe.Hybrid {
		// Generate A polynomial seeds
		seeds, numA := lhe.GenASeeds[T](prg, db.Info, ctx.RingContext)

		// Compute the hint
		result = testing.Benchmark(func(b *testing.B) {
			b.ResetTimer()
			for range b.N {
				ctx.RingContext.ComputeHint(db.Data, seeds, numA)
			}
		})
	} else {
		A := m.Rand[T](prg, db.Info.M, ctx.Params.N, 0)

		// Initialize the GPU context if available
		var gpuCtx *gpu.Context[T]
		if gpu.UseGPU() {
			gpuCtx = gpu.NewContext[T](db.Info.L, db.Info.M, ctx.Params.N)
		}

		// Compute the hint
		result = testing.Benchmark(func(b *testing.B) {
			if gpuCtx != nil {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					gpuCtx.SetA(db.Data)
					gpuCtx.SetB(A, 0, true, true)
					gpuCtx.GEMM()
				}
			} else {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					m.Mul(db.Data, A)
				}
			}
		})
	}
	return result
}
