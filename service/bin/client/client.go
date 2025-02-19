package main

import (
	"flag"
    "log"
    "math"
    "time"

    "github.com/ryanleh/secure-inference/crypto/rand"
    "github.com/ryanleh/secure-inference/lhe"
    m "github.com/ryanleh/secure-inference/matrix"
    "github.com/ryanleh/secure-inference/service"
)


var pirAddr = "0.0.0.0"
var hcAddr = "0.0.0.0"

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

func main() {
    rows := flag.Uint64("rows", 1024, "# of rows")
	cols := flag.Uint64("cols", 1024, "# of cols")
	pMod := flag.Uint64("p", 512, "Plaintext modulus")
	bitsPer := flag.Uint64("bits", 4480, "Bits per database element")
    batchSize := flag.Uint64("batch_size", 3, "Number of queries to make")
    hintTimeMs := flag.Float64("hint_ms", 0.0, "Hint time")
    flag.Parse()

    // Initialize Client 
	log.Print("Initializing client...")
    start := time.Now()
	client := service.MakeClient(
        pirAddr, hcAddr, lhe.SimpleHybrid, *rows, *cols, *pMod, *bitsPer, *batchSize,
    )
	defer client.Free()
    log.Printf("\tTook: %0.2fs", time.Since(start).Seconds())

    // Generate client queries
	prg := rand.NewBufPRG(rand.NewPRG(&key))
    dbInfo := client.DBInfo()
    inputs := make([]*m.Matrix[m.Elem32], *batchSize)
    for i := range inputs {
        index := prg.Uint64() % dbInfo.N
        inputs[i] = m.New[m.Elem32](dbInfo.M, 1)
        inputs[i].Data()[index%dbInfo.M] = 1
    }

    // Register queries
    start = time.Now()
    keys := client.Query(inputs)
	log.Printf("Registering Queries: %dms\n", time.Since(start).Milliseconds())

    // Reset communication
    pComm, hComm := client.GetConns() 
    pComm.Reset()
    hComm.Reset()

	log.Println("Setup done.")
    time.Sleep(1 * time.Second)

    // Get answers
    iters := 15
    totalTime := 0.0
    totalPTime := 0.0
    totalHTime := 0.0
    for range iters {
        start = time.Now()
        pTime, hTime, _ := client.Answer(keys)
        totalTime += float64(time.Since(start).Milliseconds())
        totalPTime += pTime
        totalHTime += hTime
    }

    _, pDown := pComm.GetCounts()
    _, hDown := hComm.GetCounts()

    // Get average stats
    avgTimeMS := totalTime / float64(iters)
    avgPTimeMS := totalPTime / float64(iters)
    avgHTimeMS := totalHTime / float64(iters)
    avgPDownMB := (float64(pDown) / float64(iters)) / math.Pow(1024.0, 2)
    avgHDownMB := (float64(hDown) / float64(iters)) / math.Pow(1024.0, 2)

    // Now ask the PIR server to compute it's total batch capacity
    var batchCapacity uint64
    if *hintTimeMs != 0 {
        log.Printf("Getting batch capacity for avg. hint time %0.2fms", *hintTimeMs)
        batchCapacity = client.GetBatchCapacity(*hintTimeMs, avgPTimeMS)
    } else {
        log.Printf("Getting batch capacity for avg. hint time %0.2fms", avgHTimeMS)
        batchCapacity = client.GetBatchCapacity(avgHTimeMS, avgPTimeMS)
    }

    log.Printf("Answer latency: %0.2fms (p: %0.2fms, h: %0.2fms)\n", avgTimeMS, avgPTimeMS, avgHTimeMS)
    log.Printf("PIR download: %0.2fMB", avgPDownMB)
    log.Printf("Hint download: %0.2fMB", avgHDownMB)
    log.Printf("Batch Capacity: %d", batchCapacity)
}
