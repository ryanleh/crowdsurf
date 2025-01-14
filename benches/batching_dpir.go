package main

import (
	"fmt"
	"math"
	"testing"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/crypto"
//	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

func benchmarkDPIR[T m.Elem]() {
	prg := rand.NewRandomBufPRG()

	for i := range bs {
        fmt.Printf("---------\nBatch size %d\n---------\n", bs[i])

        // Generate a random instancej
        numLimbs := uint64(math.Ceil(float64(bits[i]) / 32.0))
        elemWidth := uint64(math.Ceil(float64(bits[i]) / math.Log2(float64(batchPMods[i]))))
        matrix := m.Rand[m.Elem32](prg, (batchRows[i]/elemWidth)*numLimbs, batchCols[i], 0)

        // Print DB size
        dbSizeGB := float64(bits[i]*sizes[i]) / 8.0 / math.Pow(1024.0, 3)
        fmt.Printf("DB with size: %0.2fGB\n\n", dbSizeGB)

        // Test PIR speed over the entire database
        // 
        // NOTE: Hardcoded server type for brevity
        ctx := crypto.NewContext[T](T(0).Bitlen(), batchCols[i], batchPMods[i])
        server := pbc.MakeServer[T](
            matrix,
            bs[i],
            ctx.Params.P,
            bits[i],
            prg.GenPRGKey(),
            packing,
            pbc.Hash,
            true,
        )

        // Initialize the client
        client := &pbc.Client[T]{}
        client.Init(server.Params())
        fullState := client.StateSize()

        // Benchmark the first bucket
        _, fullQueries := client.Query([]uint64{})
           
		var fullAnswers []*pbc.Answer[T]
        fullResult := testing.Benchmark(func(b *testing.B) {
            // Answer queries
            b.ResetTimer()
            for range b.N {
                fullAnswers = server.Answer(fullQueries)
            }
        })

        // Stats
        fullSec := fullResult.T.Seconds() / float64(fullResult.N)
        fmt.Printf("Full Query (%d iters): %f s\n", fullResult.N, fullSec)

		fullUploadBytes := uint64(0)
		for _, query := range fullQueries {
			if query != nil {
				fullUploadBytes += query.Size()
			}
		}
		
		fullDownBytes := uint64(0)
        for _, answer := range fullAnswers {
			if answer != nil {
				fullDownBytes += answer.Size()
			}
        }
        server.Free()
        client.Free()

        // Now we do the smaller bucket
        avgErr := avgCaseErrs[0]

        // Extract the smaller DB
        popRows, popCols, popPMod := batching.ApproxSquareDims[T](cutoffs[i][0], bits[i])
        popMatrix := m.NewFromRaw(matrix.Data()[:cutoffs[i][0] * numLimbs], popRows, popCols)

        // Initialize the server
        //
        // NOTE: Hardcoded server type for brevity
        ctx = crypto.NewContext[T](T(0).Bitlen(), popCols, popPMod)
        //server = lhe.MakeSimpleServer[T](popMatrix, bits[i], ctx, prg.GenPRGKey(), lhe.None, true)
        server = pbc.MakeServer[T](
            popMatrix,
            bs[i],
            ctx.Params.P,
            bits[i],
            prg.GenPRGKey(),
            packing,
            pbc.Hash,
            true,
        )

        // Initialize the client
        client = &pbc.Client[T]{}
        client.Init(server.Params())

        // Benchmark the bucket
        _, popQueries := client.Query([]uint64{})
       
        var popAnswers []*pbc.Answer[T]
        popResult := testing.Benchmark(func(b *testing.B) {
            // Answer queries
            b.ResetTimer()
            for range b.N {
                popAnswers = server.Answer(popQueries)
            }
        })

        // Compute expected queries-per-second
        popSec := popResult.T.Seconds() / float64(popResult.N)
        fmt.Printf("Pop Query (%d iters): %f s\n", popResult.N, popSec)

        avgTimeSec := alpha * popSec + (1 - alpha) * fullSec
        queriesSec := (float64(bs[i]) * (1 - avgErr)) / avgTimeSec
        fmt.Printf("Queries-per-sec: %0.2f s\n", queriesSec)

        client.Free()
        server.Free()

		// Print communication info
		uploadBytes := uint64(0)
		for _, query := range popQueries {
			if query != nil {
				uploadBytes += query.Size()
			}
		}
        avgUploadBytes := alpha * float64(uploadBytes) + (1 - alpha) * float64(fullUploadBytes)
		
        downloadBytes := uint64(0)
		for _, answer := range popAnswers {
			if answer != nil {
				downloadBytes += answer.Size()
			}
		}
        avgDownloadBytes := alpha * float64(downloadBytes) + (1 - alpha) * float64(fullDownBytes)
		fmt.Printf(
			"Upload: %0.2fMB, Download: %0.2fMB\n",
			float64(avgUploadBytes)/math.Pow(1024.0, 2),
			float64(avgDownloadBytes)/math.Pow(1024.0, 2),
		)

		// Print storage information
		fmt.Printf(
			"Client Storage: %0.2fMB\n\n",
			float64(fullState + client.StateSize())/math.Pow(1024.0, 2),
		)

	}
}