package matrix

// #cgo CFLAGS: -O3 -march=native
// #include "matrix.h"
import "C"

import "log"

const squishBasis32 = C.BASIS_32
const squishRatio32 = C.COMPRESSION_32

const squishBasis64 = C.BASIS_64
const squishRatio64 = C.COMPRESSION_64

// Compresses the matrix to store it in 'packed' form.
// Specifically, this method squishes the matrix by representing each
// group of 'delta' consecutive values as a single database Element,
// where each value uses 'basis' bits.
func (m *Matrix[T]) Squish() {
	basis := m.SquishBasis()
	delta := m.SquishRatio()

	n := Zeros[T](m.rows, (m.cols+delta-1)/delta)

	for i := uint64(0); i < n.rows; i++ {
		for j := uint64(0); j < n.cols; j++ {
			for k := uint64(0); k < delta; k++ {
				if delta*j+k < m.cols {
					val := m.Get(i, delta*j+k)
					if val >= (1 << m.SquishBasis()) {
						log.Fatalf("Database entry %v too large to squish (%v, %v)", val, i, delta*j +k)
					}
					n.data[i*n.cols+j] += (val << (k * basis))
				}
			}
		}
	}

	m.cols = n.cols
	m.rows = n.rows
	m.data = n.data
}

func (m *Matrix[T]) SquishBasis() uint64 {
	switch T(0).Bitlen() {
	case 32:
		return squishBasis32
	case 64:
		return squishBasis64
	default:
		panic("Shouldn't get here")
	}
}

func (m *Matrix[T]) SquishRatio() uint64 {
	switch T(0).Bitlen() {
	case 32:
		return squishRatio32
	case 64:
		return squishRatio64
	default:
		panic("Shouldn't get here")
	}
}

func (m *Matrix[T]) CanSquish(pMod uint64) bool {
	return !(pMod > (1 << m.SquishBasis()))
}
