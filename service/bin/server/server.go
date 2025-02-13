package main

import (
    "bufio"
    "log"
    "math"
    "os"

	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
    "github.com/ryanleh/secure-inference/service"
)

// Info per shard
var shard = 0
var rows = []uint64{1024}[shard]
var cols = []uint64{1024}[shard]
var pMod = []uint64{997}[shard]
var bitsPer = uint64(1)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

func randInstance() lhe.Server[m.Elem32] {
	// Generate random matrix
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, rows*numLimbs, cols, 0)

	// Build server objects
	ctx := crypto.NewContext[m.Elem32](m.Elem32(0).Bitlen(), cols, pMod)
	return lhe.MakeSimpleServer[m.Elem32](matrix, bitsPer, ctx, &key, lhe.Hybrid, true, true)
}

func main() {
    serverType := os.Args[1]
    switch serverType {
        case "pir":
            // Build a random instance
            pirServer := randInstance()
            server := service.StartServer(pirServer)
            defer server.StopServer()
        case "hint":
            elemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMod))))
            server := service.StartHCServer(rows * elemWidth, 2048)
            defer server.StopServer()
        default:
            panic("Invalid server type")
    }
	
    buf := bufio.NewReader(os.Stdin)
	log.Println("Press any button to kill server...")
	buf.ReadBytes('\n')
}
