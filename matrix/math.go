package matrix

// #cgo CFLAGS: -O3 -march=native
// #include "matrix.h"
import "C"

import (
	"fmt"
	"io"
	"unsafe"
)

func (a *Matrix[T]) Add(b *Matrix[T]) {
	if (a.cols != b.cols) || (a.rows != b.rows) {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] += b.data[i]
	}
}

func (a *Matrix[T]) AddWithMismatch(b *Matrix[T]) {
	if a.cols != b.cols {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}

	if a.rows < b.rows {
		a.Concat(Zeros[T](b.rows-a.rows, a.cols))
	}

	for i := uint64(0); i < b.cols*b.rows; i++ {
		a.data[i] += b.data[i]
	}
}

func (a *Matrix[T]) MulConst(val T) {
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] *= val
	}
}

func (a *Matrix[T]) ModConst(val T) {
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] %= val
	}
}

func (a *Matrix[T]) AddConst(val T) {
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] += val
	}
}

func (a *Matrix[T]) AddAt(i, j uint64, val T) {
	if (i >= a.rows) || (j >= a.cols) {
		panic("Out of bounds")
	}
	a.Set(i, j, a.Get(i, j)+val)
}

func (a *Matrix[T]) Sub(b *Matrix[T]) {
	if (a.cols != b.cols) || (a.rows != b.rows) {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] -= b.data[i]
	}
}

func (a *Matrix[T]) SubConst(val T) {
	for i := uint64(0); i < a.cols*a.rows; i++ {
		a.data[i] -= val
	}
}

func Mul[S Elem, T Elem](a *Matrix[S], b *Matrix[T]) *Matrix[T] {
	if b.cols == 1 {
		return MulVec(a, b)
	}
	if a.cols != b.rows {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}

	out := Zeros[T](a.rows, b.cols)

	arows := C.size_t(a.rows)
	acols := C.size_t(a.cols)
	bcols := C.size_t(b.cols)

	outPtr := unsafe.Pointer(&out.data[0])
	aPtr := unsafe.Pointer(&a.data[0])
	bPtr := unsafe.Pointer(&b.data[0])

	if S(0).Bitlen() == T(0).Bitlen() {
		switch T(0).Bitlen() {
		case 32:
			C.matMul32((*Elem32)(outPtr), (*Elem32)(aPtr), (*Elem32)(bPtr), arows, acols, bcols)
		case 64:
			C.matMul64((*Elem64)(outPtr), (*Elem64)(aPtr), (*Elem64)(bPtr), arows, acols, bcols)
		default:
			panic("Shouldn't get here")
		}
	} else if S(0).Bitlen() == 32 {
		C.matMul32by64((*Elem64)(outPtr), (*Elem32)(aPtr), (*Elem64)(bPtr), arows, acols, bcols)
	} else {
		panic("Unreachable")
	}
	return out
}

func MulSeededLeft[T Elem](a *MatrixSeeded[T], b *Matrix[T]) *Matrix[T] {
	if len(a.src) != len(a.rows) {
		panic("Bad input")
	}

	aRows := uint64(0)
	for _, rows := range a.rows {
		aRows += rows
	}

	if a.cols != b.rows {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", aRows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}

	out := Zeros[T](aRows, b.cols)

	elemSz := T(0).Bitlen() / 8
	curRows := uint64(0)
	bPtr := unsafe.Pointer(&b.data[0])

	ch := make(chan bool)
	for i := range a.rows {
		go func(it int, curRowsIn uint64) {
			buf := make([]byte, elemSz*a.cols*a.rows[it])
			bufPtr := unsafe.Pointer(&buf[0])
			outPtr := unsafe.Pointer(&out.data[curRowsIn*b.cols])

			_, err := io.ReadFull(a.src[it], buf)
			if err != nil {
				panic("Randomness error")
			}

			switch T(0).Bitlen() {
			case 32:
				C.randMatMul32((*Elem32)(outPtr), (*C.uint8_t)(bufPtr), (*Elem32)(bPtr), C.size_t(a.rows[it]), C.size_t(a.cols), C.size_t(b.cols))
			case 64:
				C.randMatMul64((*Elem64)(outPtr), (*C.uint8_t)(bufPtr), (*Elem64)(bPtr), C.size_t(a.rows[it]), C.size_t(a.cols), C.size_t(b.cols))
			default:
				panic("Shouldn't get here")
			}

			ch <- true
		}(i, curRows)
		curRows += a.rows[i]
	}

	for i := 0; i < len(a.rows); i++ {
		b := <-ch
		if !b {
			panic("Should not happen")
		}
	}

	return out
}

func MulVec[S Elem, T Elem](a *Matrix[S], b *Matrix[T]) *Matrix[T] {
	if (a.cols != b.rows) && (a.cols+1 != b.rows) && (a.cols+2 != b.rows) { // do not require exact match because of DB compression
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		panic("Dimension mismatch")
	}
	if b.cols != 1 {
		panic("Second argument is not a vector")
	}

	out := New[T](a.rows, 1)
	arows := C.size_t(a.rows)
	acols := C.size_t(a.cols)

	outPtr := unsafe.Pointer(&out.data[0])
	aPtr := unsafe.Pointer(&a.data[0])
	bPtr := unsafe.Pointer(&b.data[0])

	if S(0).Bitlen() == T(0).Bitlen() {
		switch T(0).Bitlen() {
		case 32:
			C.matMulVec32((*Elem32)(outPtr), (*Elem32)(aPtr), (*Elem32)(bPtr), arows, acols)
		case 64:
			C.matMulVec64((*Elem64)(outPtr), (*Elem64)(aPtr), (*Elem64)(bPtr), arows, acols)
		default:
			panic("Shouldn't get here")
		}
	} else if S(0).Bitlen() == 32 {
		C.matMul32Vec64((*Elem64)(outPtr), (*Elem32)(aPtr), (*Elem64)(bPtr), arows, acols)
	} else {
		panic("Unreachable")
	}
	return out
}

func MulVecPacked[S Elem, T Elem](a *Matrix[S], b *Matrix[T]) *Matrix[T] {
	if a.cols*a.SquishRatio() != b.rows {
		fmt.Printf("%d-by-%d vs. %d-by-%d\n", a.rows, a.cols, b.rows, b.cols)
		fmt.Printf("Want %v == %v", a.cols*a.SquishRatio(), b.rows)
		panic("Dimension mismatch")
	}
	if b.cols != 1 {
		panic("Second argument is not a vector")
	}

	out := New[T](a.rows+8, 1)
	arows := C.size_t(a.rows)
	acols := C.size_t(a.cols)

	outPtr := unsafe.Pointer(&out.data[0])
	aPtr := unsafe.Pointer(&a.data[0])
	bPtr := unsafe.Pointer(&b.data[0])

	if S(0).Bitlen() == T(0).Bitlen() {
		switch T(0).Bitlen() {
		case 32:
			C.matMulVecPacked32((*Elem32)(outPtr), (*Elem32)(aPtr), (*Elem32)(bPtr), arows, acols)
		case 64:
			C.matMulVecPacked64((*Elem64)(outPtr), (*Elem64)(aPtr), (*Elem64)(bPtr), arows, acols)
		default:
			panic("Shouldn't get here")
		}
	} else if S(0).Bitlen() == 32 {
		C.matMul32VecPacked64((*Elem64)(outPtr), (*Elem32)(aPtr), (*Elem64)(bPtr), arows, acols)
	} else {
		panic("Unreachable")
	}
	out.DropLastrows(8)

	return out
}

func (m *Matrix[T]) Round(round_to uint64, mod uint64) {
	for i := uint64(0); i < m.rows*m.cols; i++ {
		v := (uint64(m.data[i]) + round_to/2) / round_to
		m.data[i] = T(v % mod)
	}
}

func (m *Matrix[T]) ReduceMod(p uint64) {
	mod := T(p)
	for i := 0; i < len(m.data); i++ {
		m.data[i] = m.data[i] % mod
	}
}

func (m *Matrix[T]) ShiftDown(n int) {
	for i := 0; i < len(m.data); i++ {
		m.data[i] = (m.data[i] >> n)
	}
}
