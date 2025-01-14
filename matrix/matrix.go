package matrix

// #cgo CFLAGS: -O3 -march=native
// #include "matrix.h"
import "C"

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"

	// TODO: Remove dep (?)
	"github.com/ryanleh/secure-inference/crypto/lwe"
)

type Elem32 = C.Elem32
type Elem64 = C.Elem64

type Elem interface {
	Elem32 | Elem64
	Bitlen() uint64
}

type IoRandSource interface {
	io.Reader
	mrand.Source64
}

type Matrix[T Elem] struct {
	rows uint64
	cols uint64
	data []T
}

type MatrixSeeded[T Elem] struct {
	src  []IoRandSource
	rows []uint64
	cols uint64
}

// ----- Elem -----
func (Elem32) Bitlen() uint64 {
	return 32
}

func (Elem64) Bitlen() uint64 {
	return 64
}

// ----- Matrix -----
func (m *Matrix[T]) Copy() *Matrix[T] {
	out := &Matrix[T]{
		rows: m.rows,
		cols: m.cols,
		data: make([]T, len(m.data)),
	}

	copy(out.data[:], m.data[:])
	return out
}

func (m *Matrix[T]) Data() []T {
	return m.data
}

func (m *Matrix[T]) Rows() uint64 {
	return m.rows
}

func (m *Matrix[T]) Cols() uint64 {
	return m.cols
}

func (m *Matrix[T]) Size() uint64 {
	return m.rows * m.cols
}

func (m *Matrix[T]) AppendZeros(n uint64) {
	m.Concat(Zeros[T](n, m.Cols()))
}

func New[T Elem](rows uint64, cols uint64) *Matrix[T] {
	out := new(Matrix[T])
	out.rows = rows
	out.cols = cols
	out.data = make([]T, rows*cols)
	return out
}

func NewFromRaw[T Elem](data []T, rows uint64, cols uint64) *Matrix[T] {
	return &Matrix[T]{rows, cols, data}
}

// If mod is 0, then generate uniform random int of type T
func Rand[T Elem](src IoRandSource, rows uint64, cols uint64, mod uint64) *Matrix[T] {
	out := New[T](rows, cols)
	if mod == 0 {
		length := rows * cols
		if int(length) != len(out.data) {
			panic("Should not happen")
		}

		elemSz := T(0).Bitlen() / 8
		buf := make([]byte, elemSz*length)
		_, err := io.ReadFull(src, buf)
		if err != nil {
			panic("Randomness error")
		}

		for i := uint64(0); i < length; i++ {
			start := i * elemSz
			end := start + elemSz

			switch T(0).Bitlen() {
			case 32:
				out.data[i] = T(binary.LittleEndian.Uint32(buf[start:end]))
			case 64:
				out.data[i] = T(binary.LittleEndian.Uint64(buf[start:end]))
			default:
				panic("Shouldn't get here")
			}
		}
		return out
	}

	m := big.NewInt(int64(mod))
	for i := 0; i < len(out.data); i++ {
		v, err := rand.Int(src, m)
		if err != nil {
			panic("Randomness error")
		}
		out.data[i] = T(v.Uint64())
	}
	return out
}

// Elements in range [0, 1]
func Binary[T Elem](src IoRandSource, rows uint64, cols uint64) *Matrix[T] {
	out := Rand[T](src, rows, cols, 2)
	return out
}

// Elements in range [-1, 0, 1]
func Ternary[T Elem](src IoRandSource, rows uint64, cols uint64) *Matrix[T] {
	out := Rand[T](src, rows, cols, 3)
	out.SubConst(1)
	return out
}

func Zeros[T Elem](rows uint64, cols uint64) *Matrix[T] {
	out := New[T](rows, cols)
	for i := 0; i < len(out.data); i++ {
		out.data[i] = T(0)
	}
	return out
}

func (m *Matrix[T]) Get(i, j uint64) T {
	if i >= m.rows {
		panic("Too many rows!")
	}
	if j >= m.cols {
		panic("Too many cols!")
	}
	return m.data[i*m.cols+j]
}

func (m *Matrix[T]) Set(i, j uint64, val T) {
	if i >= m.rows {
		panic("Too many rows!")
	}
	if j >= m.cols {
		panic("Too many cols!")
	}
	m.data[i*m.cols+j] = T(val)
}

func (a *Matrix[T]) Concat(b *Matrix[T]) {
	if a.cols == 0 && a.rows == 0 {
		a.cols = b.cols
		a.rows = b.rows
		a.data = b.data
		return
	}

	if a.cols != b.cols {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}

	a.rows += b.rows
	a.data = append(a.data, b.data...)
}

func (m *Matrix[T]) DropLastrows(n uint64) {
	m.rows -= n
	m.data = m.data[:(m.rows * m.cols)]
}

func (m *Matrix[T]) GetRow(offset, num_rows uint64) *Matrix[T] {
	if offset+num_rows > m.rows {
		panic("Requesting too many rows")
	}

	m2 := New[T](num_rows, m.cols)
	m2.data = m.data[(offset * m.cols):((offset + num_rows) * m.cols)]
	return m2
}

func (m *Matrix[T]) RowsDeepCopy(offset, num_rows uint64) *Matrix[T] {
	if offset+num_rows > m.rows {
		panic("Requesting too many rows")
	}

	if offset+num_rows <= m.rows {
		m2 := New[T](num_rows, m.cols)
		copy(m2.data, m.data[(offset*m.cols):((offset+num_rows)*m.cols)])
		return m2
	}

	m2 := New[T](m.rows-offset, m.cols)
	copy(m2.data, m.data[(offset*m.cols):(m.rows)*m.cols])
	return m2
}

func (m *Matrix[T]) Equals(n *Matrix[T]) bool {
	if m.Cols() != n.Cols() {
		return false
	}
	if m.Rows() != n.Rows() {
		return false
	}

	for i := 0; i < len(m.data); i++ {
		if m.data[i] != n.data[i] {
			return false
		}
	}

	return true
}

func Gaussian[T Elem](src IoRandSource, rows, cols uint64) *Matrix[T] {
	out := New[T](rows, cols)
	samplef := lwe.GaussSample32
	switch T(0).Bitlen() {
	case 32:
		// Do nothing
	case 64:
		samplef = lwe.GaussSample64
	default:
		panic("Shouldn't get here")
	}

	for i := 0; i < len(out.data); i++ {
		out.data[i] = T(samplef(src))
	}
	return out
}

func (m *Matrix[T]) Make64() *Matrix[Elem64] {
	out := &Matrix[Elem64]{
		rows: m.rows,
		cols: m.cols,
		data: make([]Elem64, len(m.data)),
	}

	for i, val := range m.data {
		out.data[i] = Elem64(int(val))
	}

	return out
}

func (m *Matrix[T]) Make32() *Matrix[Elem32] {
	out := &Matrix[Elem32]{
		rows: m.rows,
		cols: m.cols,
		data: make([]Elem32, len(m.data)),
	}

	for i, val := range m.data {
		out.data[i] = Elem32(int(val))
	}

	return out
}

// ----- MatrixSeeded -----
func NewSeeded[T Elem](src []IoRandSource, rows []uint64, cols uint64) *MatrixSeeded[T] {
	out := new(MatrixSeeded[T])
	out.src = src
	out.rows = rows
	out.cols = cols
	return out
}
