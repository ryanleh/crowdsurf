package lhe

import (
    "unsafe"
	
	m "github.com/ryanleh/secure-inference/matrix"
)

/*
* Structs
 */
type LocalHint[T m.Elem] struct {
	DB *m.Matrix[T]
    BitsPer uint64
}

type LocalSecret[T m.Elem] struct {
	query *m.Matrix[T]
}

type Empty[T m.Elem] struct{}

// Implement dummy interfaces for relevant structs
func (h *LocalHint[T]) hint()     {}
func (h *LocalSecret[T]) secret() {}
func (e *Empty[T]) query()        {}
func (e *Empty[T]) answer()       {}
func (e *Empty[T]) Size() uint64 {
	return 0
}

/*
* Client
 */
type LocalClient[T m.Elem] struct {
	db *m.Matrix[T]
    bitsPer uint64
}

func (c *LocalClient[T]) Init(h Hint[T]) {
	hint := h.(*LocalHint[T])
	c.db = hint.DB
    c.bitsPer = hint.BitsPer
}

func (c *LocalClient[T]) Query(inputs []*m.Matrix[T]) ([]Secret[T], []Query[T]) {
	secrets := make([]Secret[T], len(inputs))
	queries := make([]Query[T], len(inputs))
	for i, input := range inputs {
		secrets[i] = &LocalSecret[T]{input}
		queries[i] = &Empty[T]{}
	}
	return secrets, queries
}

func (c *LocalClient[T]) DummyQuery(num uint64) ([]Secret[T], []Query[T]) {
	secrets := make([]Secret[T], num)
	queries := make([]Query[T], num)
	return secrets, queries
}

func (c *LocalClient[T]) Recover(secrets []Secret[T], answers []Answer[T]) []*m.Matrix[T] {
	results := make([]*m.Matrix[T], 0, len(secrets))
	for i := range secrets {
		if secrets[i] != nil {
			secret := secrets[i].(*LocalSecret[T])
			result := m.Mul(c.db, secret.query)
			results = append(results, result)
		}
	}
	return results
}

func (c *LocalClient[T]) DBInfo() *DBInfo {
    return newDBInfo(c.db.Size(), c.bitsPer, c.db.Cols(), 0)
}

func (c *LocalClient[T]) StateSize() uint64 {
	// Just returns the size of the matrix
	return c.db.Size() * 4
}

func (c *LocalClient[T]) Free() {}

/*
* Server
 */

type LocalServer[T m.Elem] struct {
	db *m.Matrix[T]
    bitsPer uint64
}

func MakeLocalServer[T m.Elem](matrix *m.Matrix[m.Elem32], bitsPer uint64) *LocalServer[T] {
    // TODO: Tmp
    switch T(0).Bitlen() {
    case 32:
        return &LocalServer[T]{(*m.Matrix[T])(unsafe.Pointer(matrix)), bitsPer}
    case 64:
        panic("Unimplemented")
    }
    return nil
}

func (s *LocalServer[T]) Hint() Hint[T] {
    return &LocalHint[T]{DB: s.db, BitsPer: s.bitsPer}
}

func (s *LocalServer[T]) SetBatch(batch uint64) {}

func (s *LocalServer[T]) Answer(queries []Query[T]) []Answer[T] {
	answers := make([]Answer[T], len(queries))
	for i := range queries {
		answers[i] = &Empty[T]{}
	}
	return answers
}

func (s *LocalServer[T]) DB() *DB {
	return nil
}

func (s *LocalServer[T]) StateSize() uint64 {
	panic("Unimplemented")
}

func (s *LocalServer[T]) Free() {}
