package pbc

import (
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type Client[T m.Elem] struct {
	lheClients []lhe.Client[T]
	batchSize  uint64
	numBuckets int64
	mapping    map[uint64]KeyChoices
	mode       Mode
	prg        *rand.BufPRGReader
}

func (c *Client[T]) Init(params *Params[T]) {
	// Copy relevant fields
	c.mapping = params.Mapping
	c.batchSize = params.BatchSize
	c.numBuckets = params.NumBuckets
	c.mode = params.Mode

	// Initialize each LHE scheme
	//
	// TODO: Make this general if we need
	c.lheClients = make([]lhe.Client[T], c.numBuckets)
	for i := range c.lheClients {
		c.lheClients[i] = &lhe.SimpleClient[T]{}
		c.lheClients[i].Init(params.LHEHints[i])
	}

	// Initialize a PRG for query generation.
	//
	// TODO: Might want to make this seeded for testing
	c.prg = rand.NewRandomBufPRG()
}

// TODO: Need to free stuff
func (c *Client[T]) Query(indices []uint64) ([]*Secret[T], []*Query[T]) {
	// First, generate a schedule for the given batch
	//
	// The schedule maps bucket -> key
	schedule := GenSchedule(indices, c.mode, c.numBuckets, c.prg)
	if schedule == nil {
		panic("Cuckoo Insertion Error")
	}

	// Build query for each bucket
	queriesPer := c.mode.NumQueriesPer()
	secrets := make([]*Secret[T], c.numBuckets)
	queries := make([]*Query[T], c.numBuckets)
	for i := range uint32(c.numBuckets) {
		if keys, ok := schedule[i]; ok {
			inputs := make([]*m.Matrix[T], len(keys))
			for j, key := range keys {
				// Build the input for this bucket
				cols := c.lheClients[i].DBInfo().M
				input := m.New[T](cols, 1)
				input.Set(uint64(c.mapping[key][i])%cols, 0, 1)
				inputs[j] = input
			}
			// Compute the query
			s, q := c.lheClients[i].Query(inputs)
			secrets[i] = &Secret[T]{keys, s}
			queries[i] = &Query[T]{q}

			// Generate dummy queries if needed
			remaining := queriesPer - uint64(len(inputs))
			if remaining > 0 {
				s, q := c.lheClients[i].DummyQuery(remaining)
				secrets[i].Secrets = append(secrets[i].Secrets, s...)
				queries[i].Queries = append(queries[i].Queries, q...)
			}
		} else {
			s, q := c.lheClients[i].DummyQuery(queriesPer)
			secrets[i] = &Secret[T]{nil, s}
			queries[i] = &Query[T]{q}
		}
	}

	return secrets, queries
}

func (c *Client[T]) Recover(secrets []*Secret[T], answers []*Answer[T]) map[uint64][]m.Elem32 {
	results := make(map[uint64][]m.Elem32, c.batchSize)
	for i := range uint32(c.numBuckets) {
		recovered := c.lheClients[i].Recover(secrets[i].Secrets, answers[i].Answers)

		for j, answer := range recovered {
			// Extract the exact part we want
			//
			// TODO: For now just a single element
			dbInfo := c.lheClients[i].DBInfo()

			// TODO: This is necessary due to typing stuff atm
			rawResult := make([]m.Elem32, 0)
			index := dbInfo.Ne * (uint64(c.mapping[secrets[i].Keys[j]][i]) / dbInfo.M)
			for j := range dbInfo.Ne {
				rawResult = append(rawResult, m.Elem32(answer.Data()[index+j]))
			}
			results[secrets[i].Keys[j]] = dbInfo.ReconstructElem(rawResult)
		}
	}
	return results
}

func (c *Client[T]) StateSize() uint64 {
	size := uint64(0)
	for i := range c.lheClients {
		size += c.lheClients[i].StateSize()
	}
	return size
}

func (c *Client[T]) Free() {
	for i := range c.lheClients {
		c.lheClients[i].Free()
	}
	c.lheClients = nil
}
