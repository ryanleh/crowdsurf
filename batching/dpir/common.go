package dpir

import (
	"github.com/ryanleh/secure-inference/batching/pbc"
	"github.com/ryanleh/secure-inference/lhe"
	m "github.com/ryanleh/secure-inference/matrix"
)

// TODO: Everything involving batch codes is hacky atm
type PirType int

const (
	Simple PirType = iota
	SimpleHybrid
	Local
	PBC
	PBCAngel
)

// TODO: Temporary Params impl until we figure out something better
//
// Assumes that the DB is already ordered by popularity, just gives the bucket
// boundaries
type Params[T m.Elem] struct {
	Cutoff uint64
    Alpha  float64
	Load   uint64
	Types  []PirType
	Hints  []interface{} // TODO
}

// Secret
type Secret[T m.Elem] struct {
    Bucket      int
	Keys        []uint64
	Secret      []lhe.Secret[T]
	BatchSecret []*pbc.Secret[T]
}

// Query
type Query[T m.Elem] struct {
    Bucket     int
	Query      []lhe.Query[T]
	BatchQuery []*pbc.Query[T]
}

func (q *Query[T]) Size() uint64 {
	size := uint64(0)
	for _, query := range q.Query {
		size += query.Size()
	}
	for _, query := range q.BatchQuery {
		size += query.Size()
	}
	return size
}

// Answer
type Answer[T m.Elem] struct {
	Answer      []lhe.Answer[T]
	BatchAnswer []*pbc.Answer[T]
}

func (a *Answer[T]) Size() uint64 {
	size := uint64(0)
	for _, answer := range a.Answer {
		size += answer.Size()
	}
	for _, answer := range a.BatchAnswer {
		size += answer.Size()
	}
	return size
}
