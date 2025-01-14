package service

import (
    "math"
    "slices"
	"testing"
	"time"
)

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

func randInstance(pirType lhe.LHEType, bitsPer, rows, cols, pMod uint64) (lhe.Server[m.Elem32], *m.Matrix[m.Elem32]) {
	if bitsPer > 63 || bitsPer%32 == 0 {
		panic("Unsupported entry bits")
	}

	// Generate random matrix
	prg := rand.NewBufPRG(rand.NewPRG(&key))
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	matrix := m.Rand[m.Elem32](prg, rows*numLimbs, cols, 0)

	// Truncate the last element of each entry to match `bitsPer`
	truncateMod := 1 << (bitsPer - (numLimbs-1)*32)
	for i := range rows * cols {
		matrix.Data()[(i+1)*numLimbs-1] %= m.Elem32(truncateMod)
	}

	// Build client / server objects
	var server lhe.Server[m.Elem32]
	ctx := crypto.NewContext[m.Elem32](m.Elem32(0).Bitlen(), cols, pMod)
    switch pirType {
    case lhe.Local:
        server = lhe.MakeLocalServer[m.Elem32](matrix, bitsPer, ctx)
    case lhe.Simple:
        server = lhe.MakeSimpleServer[m.Elem32](matrix, bitsPer, ctx, &key, lhe.None, false) // TODO
    case lhe.SimpleHybrid:
        server = lhe.MakeSimpleServer[m.Elem32](matrix, bitsPer, ctx, &key, lhe.Hybrid, false) // TODO
    default:
        panic("Unreachable")
    }
	return server, matrix
}


// TODO: More iters here + smaller DB
func testE2E(
	t *testing.T,
    pirType   lhe.LHEType,
	pirServer lhe.Server[m.Elem32],
	matrix *m.Matrix[m.Elem32],
	batchSize uint64,
) {
    // Setup server and client
	server := StartServer(pirServer)
    dbInfo := server.DBInfo()
	defer server.StopServer()
	
    hcServer := StartHCServer(dbInfo.L, 2048)
	defer hcServer.StopServer()

	time.Sleep(2 * time.Second) // Give server a second to start

	client := MakeClient("0.0.0.0", "0.0.0.0", pirType, batchSize)
	defer client.Free()

	server.SetBatch(batchSize)
	prg := rand.NewBufPRG(rand.NewPRG(&key))

    // Generate client queries
    indices := make([]uint64, batchSize)
    inputs := make([]*m.Matrix[m.Elem32], batchSize)
    expected := make([][]m.Elem32, batchSize)
    numLimbs := uint64(math.Ceil(float64(dbInfo.BitsPer) / 32.0))
    for i := range inputs {
        indices[i] = prg.Uint64() % dbInfo.N
        inputs[i] = m.New[m.Elem32](matrix.Cols(), 1)
        inputs[i].Data()[indices[i]%matrix.Cols()] = 1

        dataIdx := indices[i] * numLimbs
        expected[i] = matrix.Data()[dataIdx : dataIdx+numLimbs]
    }

    keys := client.Query(inputs)
    results := client.Answer(keys)

    // Check results
    for i := range results {
        rawResult := make([]m.Elem32, 0)
        index := dbInfo.Ne * (indices[i] / dbInfo.M)
        for j := range dbInfo.Ne {
            rawResult = append(rawResult, m.Elem32(results[i].Data()[index+j]))
        }
        result := dbInfo.ReconstructElem(rawResult)
        if !slices.Equal(result, expected[i]) {
            t.Fatalf("Failure @ %d: %v vs. %v", i, result, expected)
        }
    }
}

func TestSmallModSquare(t *testing.T) {
    rows := uint64(1024)
    cols := uint64(1024)
    bitsPer := uint64(46)
	batchSize := uint64(1)
    pirType := lhe.Simple
    s, m := randInstance(pirType, bitsPer, rows, cols, 1 << 9)
    testE2E(t, pirType, s, m, batchSize)
}
