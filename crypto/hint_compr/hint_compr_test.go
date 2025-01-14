package hint_compr

import (
//	"encoding/binary"
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/crypto/rand"
//	"io"
//	"math"
//	"math/rand"
	"testing"
)

var key = rand.PRGKey([16]byte{
	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
})

//// Basic single-query correctness test
//func TestHintCompr(t *testing.T) {
//	prg := rand.NewBufPRG(rand.NewPRG(&key))
//    rows := uint64(1024)
//    cols := uint64(1024)
//    n := uint64(2048)
//    batchSize := uint64(1)
//
//    // Identity hint
//    hint := m.Rand[m.Elem32](prg, rows, n, 0)
//
//    // Create server and set params
//    server := NewServer(rows, cols)
//    publicParams := server.SetHint(hint.Data())
//
//    // Initialize client
//    client := NewClient(rows, cols, batchSize)
//    client.RecvParams(publicParams)
//
//    // Generate a Query
//    key := m.Gaussian[m.Elem32](prg, n, 1)
//    query := client.Query([][]m.Elem32{key.Data()})
//
//    // Answer query
//    server.SetQuery(query)
//    answer := server.Answer()
//
//    // Recover
//    result := client.Recover(answer)
//    final := m.NewFromRaw(result[0], rows, 1)
//    
//    if !final.Equals(m.Mul(hint, key)) {
//        t.Fail()
//    }
//}

// Multi-query correctness test
func TestHintCompr(t *testing.T) {
	prg := rand.NewBufPRG(rand.NewPRG(&key))
    rows := uint64(1024)
    cols := uint64(1024)
    n := uint64(2048)
    batchSize := uint64(5)

    // Identity hint
    hint := m.Rand[m.Elem32](prg, rows, n, 0)

    // Create server and set params
    server := NewServer(rows, cols)
    publicParams := server.SetHint(hint.Data())

    // Initialize client
    client := NewClient(rows, cols, batchSize)
    client.RecvParams(publicParams)

    // Generate a Query
    keys := make([]*m.Matrix[m.Elem32], batchSize)
    for i := range keys {
        keys[i] = m.Gaussian[m.Elem32](prg, n, 1)
    }
    query := client.Query(keys)

    // Answer query
    server.SetQuery(query)
    answer := server.Answer()

    // Recover
    result := client.Recover(answer)
    toCheck := m.NewFromRaw(result[0], rows, 1)
    
    if !toCheck.Equals(m.Mul(hint, keys[0])) {
        t.Fail()
    }
}
