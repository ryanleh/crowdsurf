package lhe

import (
	"encoding/binary"
	"io"
	"math"
)

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/crypto/rlwe"
	m "github.com/ryanleh/secure-inference/matrix"
)

// Opaque type for passing SEAL ciphertextx
type CipherBlob = []byte

// An enum representing which optimizations a client is using
type Mode int

const (
	None Mode = iota // No optimizations
	Hybrid
)

// Hint
type SimpleHint[T m.Elem] struct {
	Seed         *rand.PRGKey
	Params       *crypto.Params
	DBInfo       *DBInfo
	Hint         *m.Matrix[T]
	Mode         Mode
    CompressHint bool
}

// Secret
type SimpleSecret[T m.Elem] struct {
	innerSecret *m.Matrix[T] // Regev secret key or decryption helper
	rlweSecret  *rlwe.Key    // RLWE-encoding of secret key
}

func (s *SimpleSecret[T]) GetInner() *m.Matrix[T] {
    return s.innerSecret
}

func (s *SimpleSecret[T]) SetInner(m *m.Matrix[T]) {
    s.innerSecret = m
}

func (s *SimpleSecret[T]) Free() {
	if s.rlweSecret != nil {
		s.rlweSecret.Free()
	}
}

// Query
type SimpleQuery[T m.Elem] struct {
	Query     *m.Matrix[T]
	FastQuery []CipherBlob
}

func (q *SimpleQuery[T]) Size() uint64 {
	size := uint64(0)
	if q.Query != nil {
		size += (T(0).Bitlen() * q.Query.Size()) / 8
	}
	if q.FastQuery != nil {
		for _, query := range q.FastQuery {
			size += uint64(len(query))
		}
	}
	return size
}

// Answer
type SimpleAnswer[T m.Elem] struct {
	Answer *m.Matrix[T]
}

func (a *SimpleAnswer[T]) Size() uint64 {
	size := uint64(0)
	if a.Answer != nil {
		size += (T(0).Bitlen() * a.Answer.Size()) / 8
	}
	return size
}

// Implement dummy interfaces for relevant structs
func (h *SimpleHint[T]) hint()     {}
func (s *SimpleSecret[T]) secret() {}
func (q *SimpleQuery[T]) query()   {}
func (a *SimpleAnswer[T]) answer() {}

/*
* Util functions
 */

// Sample a random SEAL seed
func SampleSEALSeeds(prg *rand.BufPRGReader, num int) []uint64 {
	// SEAL uses 512-bit seeds
	seed := make([]uint64, rlwe.SealSeedLength*num)
	buf := make([]byte, 8) // Holds one 64-bit value
	for i := range seed {
		io.ReadFull(prg, buf)
		seed[i] = binary.LittleEndian.Uint64(buf[:])
	}
	return seed
}

// Generate seeds for each A polynomial
func GenASeeds[T m.Elem](prg *rand.BufPRGReader, dbInfo *DBInfo, ctx *rlwe.Context[T]) ([]uint64, int) {
	// Determine how many distinct A polynomials we need to generate
	numA := int(math.Ceil(float64(dbInfo.M) / float64(ctx.N())))

	// Generate seeds for each polynomial: SEAL uses 512-bit seeds
	return SampleSEALSeeds(prg, numA), numA
}
