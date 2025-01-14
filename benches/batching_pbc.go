package main

import (
	"fmt"
	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
	"math"
	"testing"
)

func benchmarkPBC[T m.Elem]() {
	prg := rand.NewRandomBufPRG()

	for i := range bs {
		// Generate random matrix + bias
		numLimbs := uint64(math.Ceil(float64(bits[i]) / 32.0))
		//rows, cols, pMod := batching.ApproxSquareDims[T](sizes[i], bits[i])
		elemWidth := uint64(math.Ceil(float64(bits[i]) / math.Log2(float64(batchPMods[i]))))
		matrix := m.Rand[m.Elem32](prg, (batchRows[i]/elemWidth)*numLimbs, batchCols[i], 0)

		dbSizeGB := float64(bits[i]*sizes[i]) / 8.0 / math.Pow(1024.0, 3)
		fmt.Printf("DB with size: %0.2fGB\n", dbSizeGB)

		// Build client / server objects
		key := rand.RandomPRGKey()

		fmt.Print("Initializing server...")
		server := pbc.MakeServer[T](matrix, bs[i], batchPMods[i], bits[i], key, packing, hashMode, true)
		fmt.Println("Done.")

		// Initialize the client
		fmt.Print("Initializing client...")
		client := &pbc.Client[T]{}
		client.Init(server.Params())
		fmt.Println("Done.")

		defer client.Free()
		defer server.Free()

		// Generate and answer queries
		var answers []*pbc.Answer[T]
		var queries []*pbc.Query[T]
		var recovered int
		result := testing.Benchmark(func(b *testing.B) {
            recovered = 0
			// Answer queries
			for range b.N {
				b.StopTimer()

				// Generate queries
				indices := make([]uint64, bs[i])
				for j := range bs[i] {
					indices[j] = prg.Uint64() % sizes[i]
				}
				secrets, qs := client.Query(indices)
				queries = qs

				// Calculate fraction of inputs recovered successfully
				for _, secret := range secrets {
					recovered += len(secret.Keys)
				}

				b.StartTimer()
				answers = server.Answer(queries)
			}
		})

		avgRecovered := float64(recovered) / (float64(bs[i]) * float64(result.N))
		avgTimeSec := result.T.Seconds() / float64(result.N)
        queriesSec := (float64(bs[i]) * avgRecovered) / avgTimeSec
		fmt.Printf(
			"Queries-per-sec(%d, %0.2f, %d iters): %0.2f Q/s\n",
			bs[i], avgRecovered, result.N, queriesSec,
		)

		// Print communication info
		uploadBytes := uint64(0)
		for _, query := range queries {
			if query != nil {
				uploadBytes += query.Size()
			}
		}
		downloadBytes := uint64(0)
		for _, answer := range answers {
			if answer != nil {
				downloadBytes += answer.Size()
			}
		}
		fmt.Printf(
			"Upload: %0.2fMB, Download: %0.2fMB\n",
			float64(uploadBytes)/math.Pow(1024.0, 2),
			float64(downloadBytes)/math.Pow(1024.0, 2),
		)

		// Print storage information
		fmt.Printf(
			"Client Storage: %0.2fMB\n\n",
			float64(client.StateSize())/math.Pow(1024.0, 2),
		)
	}
}
