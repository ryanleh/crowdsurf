package dpir

import (
	"math/rand"

	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

type Client[T m.Elem] struct {
	// Parameters for the distribution + buckets
	Params[T]

	// LHE Clients for each bucket
	types      []PirType
	pirClients []interface{} // TODO
}

func (c *Client[T]) Init(params Params[T]) {
	c.types = params.Types
	hints := params.Hints

	// Initialize relevant fields
	c.Params = params
	c.pirClients = make([]interface{}, len(c.types))
	for i := range c.pirClients {
		// Set the correct type of LHE client for each bucket
		switch c.types[i] {
		case Local:
			client := &lhe.LocalClient[T]{}
			client.Init(hints[i].(lhe.Hint[T]))
			c.pirClients[i] = client
		case Simple, SimpleHybrid:
			client := &lhe.SimpleClient[T]{}
			client.Init(hints[i].(lhe.Hint[T]))
			c.pirClients[i] = client
		case PBC, PBCAngel:
			client := &pbc.Client[T]{}
			client.Init(hints[i].(*pbc.Params[T]))
			c.pirClients[i] = client
		default:
			panic("Invalid LHE client type")
		}
	}
}

// TODO: Need to free stuff
func (c *Client[T]) Query(indices []uint64) (*Secret[T], *Query[T]) {
    // Randomly choose which bucket to query by flipping a coin
    bucket := 0
    if rand.Float64() < c.Alpha {
        bucket = 1
    }

    queryIndices := []uint64{}
    if bucket == 0 {
        // Only allocate queries for indices that are in the popular subset
        for _, idx := range indices {
            if idx < c.Cutoff {
                queryIndices = append(queryIndices, idx)
            }
        }
    } else {
        queryIndices = indices
    }

    // Build queries for the bucket
	var secret *Secret[T]
	var query *Query[T]
    if c.types[bucket] == PBC || c.types[bucket] == PBCAngel {
        client := c.pirClients[bucket].(*pbc.Client[T])

        // Build query
        s, q := client.Query(queryIndices)
        secret = &Secret[T]{Bucket: bucket, BatchSecret: s}
        query = &Query[T]{Bucket: bucket, BatchQuery: q}
    } else {
        client := c.pirClients[bucket].(lhe.Client[T])
        dbInfo := client.DBInfo()

        // Translate indices to vector queries
        inputs := make([]*m.Matrix[T], len(queryIndices))
        for j, idx := range queryIndices {
            inputs[j] = m.New[T](dbInfo.M, 1)
            inputs[j].Set(uint64(idx%dbInfo.M), 0, 1)
        }

        // Build query
        s, q := client.Query(inputs)
        secret = &Secret[T]{Bucket: bucket, Keys: queryIndices, Secret: s}
        query = &Query[T]{Bucket: bucket, Query: q}

        // Generate dummy queries if needed
        s, q = client.DummyQuery(c.Load - uint64(len(inputs)))
        secret.Secret = append(secret.Secret, s...)
        query.Query = append(query.Query, q...)
    }
	return secret, query
}

func (c *Client[T]) Recover(secret *Secret[T], answer *Answer[T]) map[uint64][]m.Elem32 {
	results := make(map[uint64][]m.Elem32, c.Load)
    bucket := secret.Bucket
    pirType := c.Types[bucket]

    if pirType == PBC || pirType == PBCAngel {
        client := c.pirClients[bucket].(*pbc.Client[T])
        for key, val := range client.Recover(secret.BatchSecret, answer.BatchAnswer) {
            results[key] = val
        }
    } else {
        client := c.pirClients[bucket].(lhe.Client[T])
        recovered := client.Recover(secret.Secret, answer.Answer)
        dbInfo := client.DBInfo()
        for j, answer := range recovered {
            // Extract individual elements from each result
            //
            // TODO: For now just single elements instead of full LHE
            index := dbInfo.Ne * (secret.Keys[j] / dbInfo.M)

            // TODO: This is necessary due to typing stuff atm
            rawResult := make([]m.Elem32, 0)
            for k := range dbInfo.Ne {
                rawResult = append(rawResult, m.Elem32(answer.Data()[index+k]))
            }
            results[secret.Keys[j]] = dbInfo.ReconstructElem(rawResult)
        }
    }

	return results
}

func (c *Client[T]) StateSize() uint64 {
	size := uint64(0)
	for i, pirType := range c.types {
		if pirType == PBC || pirType == PBCAngel {
			client := c.pirClients[i].(*pbc.Client[T])
			size += client.StateSize()
		} else {
			client := c.pirClients[i].(lhe.Client[T])
			size += client.StateSize()
		}
	}
	return size
}

func (c *Client[T]) Free() {
	for i, pirType := range c.types {
		if pirType == PBC || pirType == PBCAngel {
			c.pirClients[i].(*pbc.Client[T]).Free()
		} else {
			c.pirClients[i].(lhe.Client[T]).Free()
		}
	}
	c.pirClients = nil
}
