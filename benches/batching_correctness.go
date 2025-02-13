package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

func benchmarkDpirCorrectness[T m.Elem]() {
	prg := rand.NewRandomBufPRG()

	for i := range bs {
        fmt.Printf("---------\nBatch size %d\n---------\n", bs[i])

        // Generate a random instancej
        numLimbs := uint64(math.Ceil(float64(bits[i]) / 32.0))
        rows, cols, pMod := batching.ApproxSquareDims[T](sizes[i], bits[i])
        elemWidth := uint64(math.Ceil(float64(bits[i]) / math.Log2(float64(pMod))))
        matrix := m.Rand[m.Elem32](prg, (rows/elemWidth)*numLimbs, cols, 0)

        // Print DB size
        dbSizeGB := float64(bits[i]*sizes[i]) / 8.0 / math.Pow(1024.0, 3)
        fmt.Printf("DB with size: %0.2fGB\n\n", dbSizeGB)

        // Test PIR speed over the entire database
        // 
        // NOTE: Hardcoded server type for brevity
        ctx := crypto.NewContext[T](T(0).Bitlen(), cols, pMod)
        server := lhe.MakeSimpleServer[T](matrix, bits[i], ctx, prg.GenPRGKey(), lhe.None, false, true)
        server.SetBatch(bs[i])

        // Initialize the client
        client := &lhe.SimpleClient[T]{}
        client.Init(server.Hint())

        // Benchmark the first bucket
        _, query := client.DummyQuery(bs[i])
            
        fullResult := testing.Benchmark(func(b *testing.B) {
            // Answer queries
            b.ResetTimer()
            for range b.N {
                server.Answer(query)
            }
        })

        // Stats
        fullSec := fullResult.T.Seconds() / float64(fullResult.N)
        fmt.Printf("Full Query (%d iters): %f s\n", fullResult.N, fullSec)

        server.Free()
        client.Free()

        // Now we run a smaller instance corresponding to each popular bucket
        for j, avgErr := range avgCaseErrs {
            // Extract the smaller DB
            popRows, popCols, popPMod := batching.ApproxSquareDims[T](cutoffs[i][j], bits[i])
            popMatrix := m.NewFromRaw(matrix.Data()[:cutoffs[i][j] * numLimbs], popRows, popCols)

            // Initialize the server
            //
            // NOTE: Hardcoded server type for brevity
            ctx = crypto.NewContext[T](T(0).Bitlen(), popCols, popPMod)
            server := lhe.MakeSimpleServer[T](popMatrix, bits[i], ctx, prg.GenPRGKey(), lhe.None, false, true)

            // Initialize the client
            client := &lhe.SimpleClient[T]{}
            client.Init(server.Hint())

            // Benchmark the bucket
            _, popQuery := client.DummyQuery(bs[i])
            
            popResult := testing.Benchmark(func(b *testing.B) {
                // Answer queries
                b.ResetTimer()
                for range b.N {
                    server.Answer(popQuery)
                }
            })

            // Compute expected queries-per-second
            popSec := popResult.T.Seconds() / float64(popResult.N)
            avgTimeSec := alpha * popSec + (1 - alpha) * fullSec
            diff := fullSec / avgTimeSec

            fmt.Printf(
                "Avg-case error %f (%d iters): %f s (%0.2fx)\n",
                avgErr, popResult.N, avgTimeSec, diff,
            )

            client.Free()
            server.Free()
        }
        fmt.Println("");
	}
}
