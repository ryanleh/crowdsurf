package pbc

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/big"
	"slices"

	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

// Cuckoo hashing constants
const K float64 = 1.5
const D uint64 = 3
const MAX_CUCKOO_ITERS uint64 = 500

// Hash-based constants
const P uint64 = 2

// An enum representing which type of PBC to use
type Mode int

const (
	Hash Mode = iota
	Cuckoo
)

// Bucket -> bucket index
type KeyChoices map[uint32]uint32

// Params
type Params[T m.Elem] struct {
	BatchSize  uint64
	NumBuckets int64
	Mode       Mode
	Mapping    map[uint64]KeyChoices // TODO: This shouldn't be included
	LHEHints   []lhe.Hint[T]
}

// Secret
type Secret[T m.Elem] struct {
	Keys    []uint64
	Secrets []lhe.Secret[T]
}

// Query
type Query[T m.Elem] struct {
	Queries []lhe.Query[T]
}

func (q *Query[T]) Size() uint64 {
	size := uint64(0)
	for _, query := range q.Queries {
		size += query.Size()
	}
	return size
}

// Answer
type Answer[T m.Elem] struct {
	Answers []lhe.Answer[T]
}

func (a *Answer[T]) Size() uint64 {
	size := uint64(0)
	for _, answer := range a.Answers {
		size += answer.Size()
	}
	return size
}

/*
* Mode Impl
 */

func (m Mode) NumChoices() uint64 {
	switch m {
	case Hash:
		return 1
	case Cuckoo:
		return D
	default:
		panic("Invalid PBC Mode")
	}
}

func (m Mode) NumBuckets(batchSize uint64) uint64 {
	switch m {
	case Hash:
		return batchSize
	case Cuckoo:
		return uint64(math.Ceil(float64(batchSize) * K))
	default:
		panic("Invalid PBC Mode")
	}
}

func (m Mode) NumQueriesPer() uint64 {
	switch m {
	case Hash:
		return P
	case Cuckoo:
		return 1
	default:
		panic("Invalid PBC Mode")
	}
}

/*
*  Util Functions
 */

// Assumes `bigMod` is already set to the correct modulus
func getCandidate(buf []byte, bigNum, bigMod *big.Int) uint32 {
	digest := sha256.Sum256(buf)
	bigNum.SetBytes(digest[:])
	bigNum.Mod(bigNum, bigMod)
	return uint32(bigNum.Uint64())
}

func getBuckets(key uint64, numChoices uint64, bigNum, bigMod *big.Int) []uint32 {
	// Convert key to bytes
	buf := make([]byte, 9)
	binary.LittleEndian.PutUint64(buf, key)

	// Generate `numChoices` bucket choices for each index
	buckets := make([]uint32, 0)
	nonce := byte(0)
	for range numChoices {
		for {
			// Compute the next candidate bucket
			buf[8] = nonce
			candidate := getCandidate(buf, bigNum, bigMod)
			nonce += 1

			// Add if the candidate is new
			if !slices.Contains(buckets, candidate) {
				buckets = append(buckets, candidate)
				break
			}
		}
	}
	return buckets
}

func EncodeDB(items []m.Elem32, numLimbs, batchSize uint64, mode Mode) ([][]m.Elem32, map[uint64]KeyChoices) {
	N := uint64(len(items)) / numLimbs
	if uint64(len(items))%numLimbs != 0 {
		panic("Invalid data")
	}

	// Compute the number of buckets
	numBuckets := mode.NumBuckets(batchSize)
	numChoices := mode.NumChoices()

	mapping := make(map[uint64]KeyChoices, 0)
	buckets := make([][]m.Elem32, numBuckets)
	for i := range buckets {
		buckets[i] = make([]m.Elem32, 0)
	}

	bigNum := big.NewInt(0)
	bigMod := big.NewInt(int64(len(buckets)))
	for i := range N {
		// Get the set of buckets this entry is mapped too
		bucketsI := getBuckets(uint64(i), numChoices, bigNum, bigMod)

		// Add the entry to each of these buckets and record the index locations
		item := items[i*numLimbs : (i+1)*numLimbs]
		keyChoices := make(KeyChoices, numChoices)
		for _, candidate := range bucketsI {
			keyChoices[candidate] = uint32(len(buckets[candidate]) / int(numLimbs))
			buckets[candidate] = append(buckets[candidate], item...)
		}
		mapping[uint64(i)] = keyChoices
	}
	return buckets, mapping
}

func cuckooInsert(
	schedule map[uint32][]uint64,
	choices map[uint64][]uint32,
	key uint64,
	depth uint64,
	prg *rand.BufPRGReader,
) bool {
	if depth >= MAX_CUCKOO_ITERS {
		return false
	}

	// If any candidate buckets are empty, insert there.
	for _, bucket := range choices[key] {
		if _, contains := schedule[bucket]; !contains {
			schedule[bucket] = []uint64{key}
			return true
		}
	}

	// Otherwise, insert into a random bucket and attempt to evict/re-insert
	// the already existing element recursively
	replaceIdx := choices[key][prg.Uint64()%D]
	oldKey := schedule[replaceIdx][0]
	schedule[replaceIdx] = []uint64{key}
	return cuckooInsert(schedule, choices, oldKey, depth+1, prg)
}

// Returns a schedule of buckets
func GenSchedule(indices []uint64, mode Mode, numBuckets int64, prg *rand.BufPRGReader) map[uint32][]uint64 {
	// Get the possible bucket choices for each key
	numChoices := mode.NumChoices()
	choices := make(map[uint64][]uint32)
	bigNum := big.NewInt(0)
	bigMod := big.NewInt(numBuckets)
	for _, key := range indices {
		choices[key] = getBuckets(key, numChoices, bigNum, bigMod)
	}

	schedule := make(map[uint32][]uint64, 0)
	switch mode {
	case Hash:
		for _, key := range indices {
			// TODO: For now we only do single queries to each bucket. Queries
			// might be overwritten
			if qs, ok := schedule[choices[key][0]]; ok {
				// Add if length < R
				if uint64(len(qs)) < P {
					schedule[choices[key][0]] = append(schedule[choices[key][0]], key)
				}
			} else {
				schedule[choices[key][0]] = []uint64{key}
			}
		}
	case Cuckoo:
		// Do cuckoo hashing insertion following the approach of Angel et. al
		for _, key := range indices {
			if !cuckooInsert(schedule, choices, key, 0, prg) {
				return nil
			}
		}
	}
	return schedule
}
