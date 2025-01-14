package main

import (
    "bufio"
    "log"
    "math"
    "time"
    "os"

    "github.com/ryanleh/secure-inference/crypto/rand"
    "github.com/ryanleh/secure-inference/lhe"
    m "github.com/ryanleh/secure-inference/matrix"
    "github.com/ryanleh/secure-inference/service"
)


var pirAddr = "0.0.0.0"
var hcAddr = "0.0.0.0"
var batchSize = uint64(3)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

func main() {
    // Initialize Client 
	log.Print("Initializing client...")
    start := time.Now()
	client := service.MakeClient(pirAddr, hcAddr, lhe.Simple, batchSize)
	//client := service.MakeClient(pirAddr, hcAddr, lhe.Local, batchSize)
	defer client.Free()
    log.Printf("\tTook: %0.2fs", time.Since(start).Seconds())

    // Generate client queries
	prg := rand.NewBufPRG(rand.NewPRG(&key))
    dbInfo := client.DBInfo()
    inputs := make([]*m.Matrix[m.Elem32], batchSize)
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

    buf := bufio.NewReader(os.Stdin)
	log.Println("Setup done, press any key to continue...")
	buf.ReadBytes('\n')

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

    log.Printf("Answer latency: %0.2fms (p: %0.2fms, h: %0.2fms)\n", avgTimeMS, avgPTimeMS, avgHTimeMS)
    log.Printf("PIR download: %0.2fMB", avgPDownMB)
    log.Printf("Hint download: %0.2fMB", avgHDownMB)
}
