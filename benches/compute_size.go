package main

import (
    "fmt"
    "math"
	
	"github.com/ryanleh/secure-inference/batching"
    m "github.com/ryanleh/secure-inference/matrix"
)


// Helper script
func computeSizes() {
    size := uint64(1343345)
    //size := uint64(3040422)
    bits := uint64(4480)
    maxRows := uint64(4096)

    //rows, cols, pMod := batching.ApproxSquareDims[m.Elem32](size, bits)
    rows, cols, pMod := batching.ApproxRowConstraint[m.Elem32](size, bits, maxRows)

    fmt.Printf(
        "(%d, %d, %d) => DB has size %0.2f MB\n",
        rows, cols, pMod, float64(rows * 4 * cols) / (math.Pow(1024.0, 2.0)),
    )


//    // For each batch size (i.e. database), compute the matrix dimensions to
//    // reach that state size using SimplePIR
//    for i, batchSize := range bs {
//        //numLimbs := uint64(math.Ceil(float64(bits[i]) / 32.0))
//        //rows, cols, pMod := batching.ApproxSquareDims[T](sizes[i], bits[i])
//        //elemWidth := uint64(math.Ceil(float64(bits[i]) / math.Log2(float64(pMod))))
//
//        // Assuming 32-bit things
//        n := uint64(2048)
//        maxRows := uint64(math.Floor(float64(size) / (float64(n * 4) / math.Pow(1024.0, 2.0))))
//        rows, cols, pMod := batching.ApproxRowConstraint[m.Elem32](sizes[i], bits[i], maxRows)
//        fmt.Printf(
//            "Batch size %d: (%d, %d, %d) => %0.2f MB\n",
//            batchSize, rows, cols, pMod, float64(rows * 4 * n) / math.Pow(1024.0, 2.0),
//        )
//    }
}
