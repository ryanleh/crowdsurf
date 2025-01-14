package dpir

import (
	"math"

	"github.com/ryanleh/secure-inference/batching"
	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type Server[T m.Elem] struct {
	cutoff     uint64
    alpha      float64
	load       uint64
	types      []PirType
	pirServers []interface{}
}

// TODO: Depending on params type, may need to refactor things here
func MakeServer[T m.Elem](
	matrix *m.Matrix[m.Elem32],
	cutoff uint64,
    alpha float64,
    load uint64,
	bitsPer uint64,
	pMod uint64,
	types []PirType,
	packing batching.Packing,
	seed *rand.PRGKey,
	bench bool, // TODO: Remove
) *Server[T] {
	// PRG for creating seeds
	prg := rand.NewBufPRG(rand.NewPRG(seed))

    // Copy into two buckets
	numLimbs := uint64(math.Ceil(float64(bitsPer) / 32.0))
	buckets := make([][]m.Elem32, 2)
    buckets[0] = matrix.Data()[:cutoff*numLimbs] // Popular
    buckets[1] = matrix.Data()                   // Full

    // Approximate square dimensions for 
    pop_rows, pop_cols, pop_p := batching.ApproxSquareDims[T](cutoff, bitsPer)
    rows := []uint64{pop_rows, matrix.Rows()}
    cols := []uint64{pop_cols, matrix.Cols()}
    pMods := []uint64{pop_p, pMod}
	
    // Initialize a PIR Server for each bucket
	pirServers := make([]interface{}, len(buckets))
	for i := range buckets {
		ctx := crypto.NewContext[T](T(0).Bitlen(), cols[i], pMods[i])
		elemWidth := uint64(math.Ceil(float64(bitsPer) / math.Log2(float64(pMods[i]))))
		matrix := m.NewFromRaw(buckets[i], (rows[i]*numLimbs)/elemWidth, cols[i])
		switch types[i] {
		case Local:
			server := lhe.MakeLocalServer[T](matrix, bitsPer)
			server.SetBatch(load)
			pirServers[i] = server

		case Simple:
			server := lhe.MakeSimpleServer[T](
				matrix,
				bitsPer,
				ctx,
				prg.GenPRGKey(),
				lhe.None,
				bench,
			)
			server.SetBatch(load)
			pirServers[i] = server

		case SimpleHybrid:
			server := lhe.MakeSimpleServer[T](
				matrix,
				bitsPer,
				ctx,
				prg.GenPRGKey(),
				lhe.Hybrid,
				bench,
			)
			server.SetBatch(load)
			pirServers[i] = server

		case PBC:
			server := pbc.MakeServer[T](
				matrix,
				load,
				ctx.Params.P,
				bitsPer,
				prg.GenPRGKey(),
				packing,
				pbc.Hash,
				bench,
			)
			pirServers[i] = server

		case PBCAngel:
			server := pbc.MakeServer[T](
				matrix,
				load,
				ctx.Params.P,
				bitsPer,
				prg.GenPRGKey(),
				packing,
				pbc.Cuckoo,
				bench,
			)
			pirServers[i] = server

		default:
			panic("Invalid LHE type")
		}
	}

	return &Server[T]{cutoff, alpha, load, types, pirServers}
}

func (s *Server[T]) Params() Params[T] {
	hints := make([]interface{}, len(s.pirServers))
	for i, pirType := range s.types {
		if pirType == PBC || pirType == PBCAngel {
			server := s.pirServers[i].(*pbc.Server[T])
			hints[i] = server.Params()
		} else {
			server := s.pirServers[i].(lhe.Server[T])
			hints[i] = server.Hint()
		}
	}

	return Params[T]{
		Cutoff: s.cutoff,
        Alpha:  s.alpha,
		Load:   s.load,
		Types:   s.types,
		Hints:   hints,
	}
}

func (s *Server[T]) Answer(query *Query[T]) *Answer[T] {
	var answer *Answer[T]
    bucket := query.Bucket
    pirType := s.types[bucket]
    if pirType == PBC || pirType == PBCAngel {
        server := s.pirServers[bucket].(*pbc.Server[T])
        answer = &Answer[T]{
            BatchAnswer: server.Answer(query.BatchQuery),
        }
    } else {
        server := s.pirServers[bucket].(lhe.Server[T])
        answer = &Answer[T]{
            Answer: server.Answer(query.Query),
        }
    }
	return answer
}

func (s *Server[T]) StateSize() uint64 {
	panic("Unimplemented")
}

func (s *Server[T]) Free() {
	for i, pirType := range s.types {
		if pirType == PBC || pirType == PBCAngel {
			s.pirServers[i].(*pbc.Server[T]).Free()
		} else {
			s.pirServers[i].(lhe.Server[T]).Free()
		}
	}
	s.pirServers = nil
}
