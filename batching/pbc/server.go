package pbc

import (
	"math"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type Server[T m.Elem] struct {
	lheServers []lhe.Server[T]
	batchSize  uint64
	buckets    [][]m.Elem32
	mapping    map[uint64]KeyChoices
	mode       Mode
}

func MakeServer[T m.Elem](
	matrix *m.Matrix[m.Elem32],
	batchSize, pMod, bitsPer uint64,
	seed *rand.PRGKey,
	packing batching.Packing,
	mode Mode,
	bench bool, // TODO: Remove
) *Server[T] {
	// PRG for creating seeds
	prg := rand.NewBufPRG(rand.NewPRG(seed))

	// Encode the database into buckets
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	buckets, mapping := EncodeDB(matrix.Data(), numLimbs, batchSize, mode)

	// Get the re-mapped bucket parameters
	bucketSizes := make([]uint64, len(buckets))
	for i := range buckets {
		bucketSizes[i] = uint64(len(buckets[i]))
	}
	rows, cols, pMods := batching.PackingDims[T](bucketSizes, bitsPer, matrix.Rows(), matrix.Cols(), pMod, packing)

	// Initialize an LHE server for each bucket
	//
	// TODO: Make this general if we need
	servers := make([]lhe.Server[T], len(buckets))

	for i := range servers {
		// Create the LHE server
		servers[i] = lhe.MakeSimpleServer[T](
			m.NewFromRaw(buckets[i], rows[i], cols[i]),
			bitsPer,
			crypto.NewContext[T](T(0).Bitlen(), cols[i], pMods[i]),
			prg.GenPRGKey(),
			lhe.Hybrid,
			bench,
		)
	}

	return &Server[T]{servers, batchSize, buckets, mapping, mode}
}

func (s *Server[T]) Params() *Params[T] {
	hints := make([]lhe.Hint[T], len(s.lheServers))
	for i, server := range s.lheServers {
		hints[i] = server.Hint()
	}
	params := &Params[T]{
		BatchSize:  s.batchSize,
		NumBuckets: int64(len(s.buckets)),
		Mapping:    s.mapping,
		LHEHints:   hints,
	}

	if s.mode == Cuckoo {
		params.Mapping = s.mapping
	}
	return params
}

func (s *Server[T]) SetBatch(batch uint64) {
	for _, server := range s.lheServers {
		server.SetBatch(batch)
	}
}

// TODO: Make this multi-client
func (s *Server[T]) Answer(queries []*Query[T]) []*Answer[T] {
	answers := make([]*Answer[T], len(s.buckets))
	for i, query := range queries {
		answers[i] = &Answer[T]{s.lheServers[i].Answer(query.Queries)}
	}
	return answers
}

func (s *Server[T]) StateSize() uint64 {
	panic("Unimplemented")
}

func (s *Server[T]) Free() {
	for _, server := range s.lheServers {
		server.Free()
	}
	s.lheServers = nil
}
