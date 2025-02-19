package hint_compr

//import (
////	"encoding/binary"
//	m "github.com/ryanleh/secure-inference/matrix"
//	"github.com/ryanleh/secure-inference/crypto/rand"
////	"io"
////	"math"
////	"math/rand"
//	"testing"
//)
//
//var key = rand.PRGKey([16]byte{
//	100, 121, 60, 254, 76, 111, 7, 102, 199, 220, 220, 5, 95, 174, 252, 221,
//})
//
//// Multi-query correctness test
//func TestHintCompr(t *testing.T) {
//	prg := rand.NewBufPRG(rand.NewPRG(&key))
//    rows := []uint64{1024, 1024}
//    cols := []uint64{1024, 1024}
//    n := uint64(2048)
//    batchSize := []uint64{5, 5}
//
//    // Create server and client
//    server := NewServer(rows[0], cols[0])
//    client := NewClient(rows[0], cols[0], batchSize[0])
//
//    for i := range(rows) {
//        if i != 0 {
//            server.Reset(rows[i], cols[i])
//            client.Reset(rows[i], cols[i], batchSize[i])
//        }
//
//        // Initialize things
//        hint := m.Rand[m.Elem32](prg, rows[i], n, 0)
//        publicParams := server.SetHint(hint.Data())
//        client.RecvParams(publicParams)
//
//        // Generate a Query
//        keys := make([]*m.Matrix[m.Elem32], batchSize[i])
//        for j := range keys {
//            keys[j] = m.Gaussian[m.Elem32](prg, n, 1)
//        }
//        query := client.Query(keys)
//
//        // Answer query
//        server.SetQuery(query)
//        answer := server.Answer()
//
//        // Recover
//        result := client.Recover(answer, rows[i])
//        toCheck := m.NewFromRaw(result[0], rows[i], 1)
//        
//        if !toCheck.Equals(m.Mul(hint, keys[0])) {
//            t.Fail()
//        }
//    }
//}
