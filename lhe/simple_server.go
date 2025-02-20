package lhe

import (
    mrand "math/rand"

	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	m "github.com/ryanleh/secure-inference/matrix"
	"github.com/ryanleh/secure-inference/matrix/gpu"
)

// TODO: Memory management when using a GPU can be improved
type SimpleServer[T m.Elem] struct {
	seed *rand.PRGKey
	mode Mode

	// Data
	db *DB

	// For slow queries / no hint compression
	hint *m.Matrix[T]

	// Context for crypto objects
	cryptoCtx *crypto.Context[T]

	// GPU stuff
	gpuCtx *gpu.Context[T]

    // Whether we're using hint compression
    //
    // TODO: This currently is only supported when running with the `service`
    // folder
    compressHint bool
}

func MakeSimpleServer[T m.Elem](
	matrix *m.Matrix[m.Elem32],
	dbElemBits uint64,
	cryptoCtx *crypto.Context[T],
	seed *rand.PRGKey,
	mode Mode,
    compressHint bool,
	bench bool, // TODO: Remove
) *SimpleServer[T] {
	params := cryptoCtx.Params

	// Encode the matrix into a DB and initialize the GPU context if available
	db := NewDB(matrix.Data(), dbElemBits, params.M, params.P, bench)
	//println("DB with size: ", db.Data.Rows(), ", ", db.Data.Cols(), "-- P = ", params.P)

	var gpuCtx *gpu.Context[T]
	if gpu.UseGPU() {
		gpuCtx = gpu.NewContext[T](db.Info.L, db.Info.M, params.N)
		gpuCtx.SetA(db.Data)
	}

	// Generate hint
	var hint *m.Matrix[T]
	if bench {
		// Generate a random hint if running benchmarks
        rng := mrand.New(mrand.NewSource(0))
		hint = m.Rand[T](rng, db.Info.L, params.N, 0)
	} else {
        prg := rand.NewBufPRG(rand.NewPRG(seed))
		if mode == None {
			matrixA := m.Rand[T](prg, db.Info.M, params.N, 0)
			if gpuCtx != nil {
				gpuCtx.SetB(matrixA, 0, true, true)
				hint = gpuCtx.GEMM()
			} else {
				hint = m.Mul(db.Data, matrixA)
			}
		} else {
			seeds, numA := GenASeeds[T](prg, db.Info, cryptoCtx.RingContext)
			hint = cryptoCtx.RingContext.ComputeHint(db.Data, seeds, numA)
		}
	}

	// If using a GPU, copy the DB to the device, otherwise squish the
	// DB representation
	if gpuCtx != nil {
		gpuCtx.Allocate(db.Info.L, db.Info.M, 1)
		gpuCtx.SetA(db.Data)
	} else {
		db.Squish()
	}

	return &SimpleServer[T]{
		seed,
		mode,
		db,
		hint,
		cryptoCtx,
		gpuCtx,
        compressHint,
	}
}

func (s *SimpleServer[T]) Free() {
	s.cryptoCtx.Free()
	if s.gpuCtx != nil {
		s.gpuCtx.Free()
	}
}

func (s *SimpleServer[T]) Hint() Hint[T] {
    hint := &SimpleHint[T]{
		Seed:   s.seed,
		Params: s.cryptoCtx.Params,
		DBInfo: s.db.Info,
		Mode:   s.mode,
        Hint:   s.hint,
        CompressHint: s.compressHint,
	}
	return hint
}

func (s *SimpleServer[T]) SetBatch(batch uint64) {
	if s.gpuCtx != nil {
		s.gpuCtx.SetBatch(batch)
	}
}

func (s *SimpleServer[T]) Answer(queries []Query[T]) []Answer[T] {
	var answers []Answer[T]

	// If using a GPU, perform a single matrix product
	//
	// TODO: Come up with a cleaner way to do this
	if s.gpuCtx != nil {
		answers = make([]Answer[T], 1)

		// Copy queries to GPU memory
		bGpuPtr := s.gpuCtx.GetHostData(1)
		for i := range queries {
			query := queries[i].(*SimpleQuery[T])
			if s.mode == Hybrid {
				// Extract CT LWE representation and modulus switch before copying
				for j := range query.FastQuery {
					s.cryptoCtx.RingContext.ExtractLWECtGPU(
						query.FastQuery[j],
						min(s.db.Info.M-uint64(j)*s.cryptoCtx.Params.N, s.cryptoCtx.Params.N),
						bGpuPtr,
						i*int(s.db.Info.M)+j*int(s.cryptoCtx.Params.N),
					)
				}
			} else {
				s.gpuCtx.SetB(query.Query, int(s.db.Info.M)*i, false, false)
			}
		}

		// Sync data and perform matrix computation
		s.gpuCtx.SyncDevice(1)
		answers[0] = &SimpleAnswer[T]{s.gpuCtx.GEMM()}
	} else {
		answers = make([]Answer[T], len(queries))
		for i := range queries {
			query := queries[i].(*SimpleQuery[T])
			ct := query.Query
			if s.mode == Hybrid {
				// Extract CT LWE representation and modulus switch
				tmpCT := m.New[T](0, 0)
				for j := range query.FastQuery {
					numSamples := min(s.db.Info.M-uint64(j)*s.cryptoCtx.Params.N, s.cryptoCtx.Params.N)
					tmpCT.Concat(s.cryptoCtx.RingContext.ExtractLWECt(query.FastQuery[j], numSamples))
				}

				// Pad the query to match the dimensions of the compressed DB if
				// applicable
				if s.db.Info.Squishing != 0 && s.db.Info.M%s.db.Info.Squishing != 0 {
					tmpCT.AppendZeros(s.db.Info.Squishing - (s.db.Info.M % s.db.Info.Squishing))
				}
				ct = tmpCT
			}

			// Compute the matrix product
			var answer SimpleAnswer[T]
			if s.db.Info.Squishing != 0 {
				answer = SimpleAnswer[T]{m.MulVecPacked(s.db.Data, ct)}
			} else {
				answer = SimpleAnswer[T]{m.MulVec(s.db.Data, ct)}
			}
			answers[i] = &answer
		}
	}
	return answers
}

func (s *SimpleServer[T]) DB() *DB {
	return s.db
}

func (s *SimpleServer[T]) StateSize() uint64 {
	panic("Unimplemented")
}
