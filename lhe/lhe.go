package lhe

import (
	m "github.com/ryanleh/secure-inference/matrix"
)

//
// NOTE: All of the schemes in this folder can support LHE (instead of just
// PIR), but not all currently implement this
//

// An enum representing the different LHE implementations
type LHEType int

const (
	Simple LHEType = iota
	SimpleHybrid
	Local
)

// The interface for an LHE client
type Client[T m.Elem] interface {
	// Initialize an LHE client using a hint
	Init(Hint[T])

	// Generate an LHE query
	Query([]*m.Matrix[T]) ([]Secret[T], []Query[T])

	// Generate dummy LHE queries
	DummyQuery(uint64) ([]Secret[T], []Query[T])

	// Decrypt an LHE answer
	Recover([]Secret[T], []Answer[T]) []*m.Matrix[T]

	// Get database info
	DBInfo() *DBInfo

	// Get the client state size
	StateSize() uint64

	// Free the client
	Free()
}

// The interface for an LHE server
type Server[T m.Elem] interface {
	// Produce a hint to initialize a client
	Hint() Hint[T]

	// Set the batch size of the Answer routine
	SetBatch(batch uint64)

	// Answer an LHE Query
	Answer([]Query[T]) []Answer[T]

	// Get the raw DB
	DB() *DB

	// Get the server state size
	StateSize() uint64

	// Free the server
	Free()
}

// We can't really do what we'd like type-wise with the interfaces above
// because of Go's lack of complexity to the type system (we'd really like an
// interface with associated types). The above code basically just uses
// `interface{}` for all input/output types since these types will differ
// depending on the exact scheme used. To get a tiny bit more control here, we
// define the 'dummy' interfaces below which the various types will implement.
type Hint[T m.Elem] interface {
	hint()
}

type Secret[T m.Elem] interface {
	secret()
}

type Query[T m.Elem] interface {
	query()
	Size() uint64
}

type Answer[T m.Elem] interface {
	answer()
	Size() uint64
}
