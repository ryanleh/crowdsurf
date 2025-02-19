package lhe

import (
	"github.com/ryanleh/secure-inference/crypto"
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/crypto/rlwe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type SimpleClient[T m.Elem] struct {
	prg    *rand.BufPRGReader
	mode   Mode
	dbInfo *DBInfo

	// For slow queries / no hint compression
	seedA *rand.PRGKey
	hint  *m.Matrix[T]

	// For fast queries
	polysA []*rlwe.A

	// Context for crypto objects
	ctx *crypto.Context[T]

    // Whether we're using hint compression
    //
    // TODO: This currently is only supported when running with the `service`
    // folder
    compressHint bool
}

func (c *SimpleClient[T]) Init(h Hint[T]) {
	// Copy relevant fields
    
    hint := h.(*SimpleHint[T])
	c.seedA = hint.Seed
	c.dbInfo = hint.DBInfo
	c.mode = hint.Mode
	c.hint = hint.Hint
    c.compressHint = hint.CompressHint

	// Initialize crypto contexts
	c.ctx = crypto.NewContext[T](hint.Params.LogQ, hint.Params.M, hint.Params.P)

	// Generate A matrices
	if hint.Mode == Hybrid {
		prg := rand.NewBufPRG(rand.NewPRG(hint.Seed))
		seeds, numA := GenASeeds[T](prg, c.dbInfo, c.ctx.RingContext)
		c.polysA = make([]*rlwe.A, numA)
		key := c.ctx.RingContext.NewKey()
		for i := range c.polysA {
			seed := seeds[i*rlwe.SealSeedLength : (i+1)*rlwe.SealSeedLength]
			c.polysA[i] = rlwe.NewA(key, seed)
		}
	}

	// Initialize a new PRG for query generation
	c.prg = rand.NewRandomBufPRG()
}

func (c *SimpleClient[T]) Query(inputs []*m.Matrix[T]) ([]Secret[T], []Query[T]) {
	secrets := make([]Secret[T], len(inputs))
	queries := make([]Query[T], len(inputs))
	if c.mode == Hybrid {
		// Construct the queries
		for i := range inputs {
			// Sample secret key
			rlweSecret := c.ctx.RingContext.NewKey()
			innerSecret := c.ctx.RingContext.ExtractLWEKey(rlweSecret)
			secret := &SimpleSecret[T]{innerSecret, rlweSecret}

			// For each `a` polynomial, compute `a * s + e + delta * m`
			query := &SimpleQuery[T]{FastQuery: make([]CipherBlob, len(c.polysA))}
			for j, polyA := range c.polysA {
				ct := rlwe.NewCiphertext()
				defer ct.Free()

				// Extract data to embed in this ciphertext
				start := uint64(j) * c.ctx.Params.N
				end := min(uint64(j+1)*c.ctx.Params.N, inputs[i].Size())
				data := inputs[i].Data()[start:end]

				// Encrypt
				secret.rlweSecret.PreprocessEnc(polyA, ct)
				c.ctx.RingContext.EncryptPreprocessed(secret.rlweSecret, data, ct)
				query.FastQuery[j] = ct.StoreData()
			}
			secrets[i] = secret
			queries[i] = query
		}
	} else {
		for i := range inputs {
			// Sample secret key
			secret := &SimpleSecret[T]{innerSecret: m.Gaussian[T](c.prg, c.ctx.Params.N, 1)}

			// Compute `A * s + e + delta * m`
			query := &SimpleQuery[T]{}
			err := m.Gaussian[T](c.prg, c.dbInfo.M, 1)
			matrixAseeded := m.NewSeeded[T](
				[]m.IoRandSource{rand.NewBufPRG(rand.NewPRG(c.seedA))},
				[]uint64{c.dbInfo.M},
				c.ctx.Params.N,
			)
			query.Query = m.MulSeededLeft(matrixAseeded, secret.innerSecret)
			query.Query.Add(err)

			// NOTE: `inputs` is modified in-place + will share memory with `secrets`
			inputs[i].MulConst(T(c.ctx.Params.Delta))
			query.Query.Add(inputs[i])

			// Pad the query to match the dimensions of the compressed DB if
			// applicable
			if !c.dbInfo.GPU && c.dbInfo.Squishing != 0 && c.dbInfo.M%c.dbInfo.Squishing != 0 {
				query.Query.AppendZeros(c.dbInfo.Squishing - (c.dbInfo.M % c.dbInfo.Squishing))
			}
			secrets[i] = secret
			queries[i] = query
		}
	}

	return secrets, queries
}

func (c *SimpleClient[T]) DummyQuery(num uint64) ([]Secret[T], []Query[T]) {
	secrets := make([]Secret[T], num)
	queries := make([]Query[T], num)

	if c.mode == Hybrid {
		// Generate random seeds
		seeds := SampleSEALSeeds(c.prg, int(num))

		for i := range queries {
			seed := seeds[i*rlwe.SealSeedLength : (i+1)*rlwe.SealSeedLength]
			dummyCTs := c.ctx.RingContext.StoreRandomCTs(c.dbInfo.M, seed)
			secrets[i] = &SimpleSecret[T]{}
			queries[i] = &SimpleQuery[T]{FastQuery: dummyCTs}
		}
	} else {
		for i := range queries {
			query := &SimpleQuery[T]{
				Query: m.Rand[T](c.prg, c.dbInfo.M, 1, 0),
			}

			// Pad the query to match the dimensions of the compressed DB if
			// applicable
			//
			// TODO: Is this necessary?
			if !c.dbInfo.GPU && c.dbInfo.Squishing != 0 && c.dbInfo.M%c.dbInfo.Squishing != 0 {
				query.Query.AppendZeros(c.dbInfo.Squishing - (c.dbInfo.M % c.dbInfo.Squishing))
			}
			secrets[i] = &SimpleSecret[T]{}
			queries[i] = query
		}
	}
	return secrets, queries
}

func (c *SimpleClient[T]) Recover(secrets []Secret[T], answers []Answer[T]) []*m.Matrix[T] {
	results := make([]*m.Matrix[T], 0, len(answers))

	for i := range len(answers) {
		// Probably bad code where we type cast to the specific impl of the interface
		secret := secrets[i].(*SimpleSecret[T])

		// If this is a dummy query, skip this iteration
		if secret.innerSecret == nil && secret.rlweSecret == nil {
			continue
		}

        // TODO: E2E tests currently rerun with the same secret multiple times
        if !c.compressHint {
            defer secret.Free()
        }

		var answer *SimpleAnswer[T]
		if c.dbInfo.GPU {
			// TODO: Remove the copy / transpose here
			a := answers[0].(*SimpleAnswer[T]).Answer
			colCopy := m.New[T](a.Rows(), 1)
			for j := range a.Rows() {
				colCopy.Data()[j] = a.Get(j, uint64(i))
			}
			answer = &SimpleAnswer[T]{colCopy}
		} else {
			answer = answers[i].(*SimpleAnswer[T])
		}

        // TODO: Add a hint compression option for this
        var token *m.Matrix[T]
        if c.compressHint {
            token = secret.innerSecret
        } else {
            token = m.Mul(c.hint, secret.innerSecret)
        }

		// Subtract `H*s` from ciphertext
		ans := answer.Answer
		ans.Sub(token)

		// Round to recover final result
		//
		// TODO: Unify these
		var result *m.Matrix[T]
		if c.mode == Hybrid {
			c.ctx.RingContext.RoundLWEInplace(ans)
			result = m.NewFromRaw(ans.Data(), ans.Rows(), ans.Cols())
		} else {
			result = m.Zeros[T](ans.Rows(), 1)
			for row := uint64(0); row < ans.Rows(); row++ {
				noised := uint64(ans.Get(row, 0))
				denoised := c.ctx.Params.Round(noised)
				result.Set(row, 0, T(denoised%c.dbInfo.P))
			}
		}
		results = append(results, result)
	}
	return results
}

func (c *SimpleClient[T]) DBInfo() *DBInfo {
	return c.dbInfo
}

func (c *SimpleClient[T]) StateSize() uint64 {
	// Just returns the size of the hint
	if c.hint != nil {
		return c.hint.Size() * T(0).Bitlen() / 8
	}
	return 0
}

func (c *SimpleClient[T]) Free() {
	// Must call to free C++ memory
	c.ctx.Free()
	for i := range c.polysA {
		c.polysA[i].Free()
	}
}
