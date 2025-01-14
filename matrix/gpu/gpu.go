package gpu

// For this code to work, the internal CUDA code needs to be built via:
// ```
//  cd secure-inference/matrix/gpu/cuda
//  cmake -S . -B build
//  cmake --build build -j 8
// ```

// #cgo CXXFLAGS: -std=c++17 -Wall -Werror -O3 -march=native
// #cgo LDFLAGS: -L${SRCDIR}/cuda/build -Wl,-rpath,${SRCDIR}/cuda/build -lgpu
// #include "cuda/ffi.h"
import "C"

import (
	m "github.com/ryanleh/secure-inference/matrix"
	"unsafe"
)

type Context[T m.Elem] struct {
	ctx32   *C.GPUMul32_t
	ctx64   *C.GPUMul64_t
	outRows uint64
	outCols uint64
}

func NewContext[T m.Elem](m, k, n uint64) *Context[T] {
	ctx := &Context[T]{
		outRows: m,
		outCols: n,
	}
	switch T(0).Bitlen() {
	case 32:
		ctx.ctx32 = C.mul_new_32(C.uint64_t(m), C.uint64_t(k), C.uint64_t(n))
	case 64:
		ctx.ctx64 = C.mul_new_64(C.uint64_t(m), C.uint64_t(k), C.uint64_t(n))
	default:
		panic("Unreachable")
	}
	return ctx
}

func (ctx *Context[T]) Free() {
	switch T(0).Bitlen() {
	case 32:
		C.mul_free_32(ctx.ctx32)
	case 64:
		C.mul_free_64(ctx.ctx64)
	default:
		panic("Unreachable")
	}
}

func (ctx *Context[T]) Allocate(m, k, n uint64) {
	ctx.outRows = m
	ctx.outCols = n
	switch T(0).Bitlen() {
	case 32:
		C.allocate_32(ctx.ctx32, C.uint64_t(m), C.uint64_t(k), C.uint64_t(n))
	case 64:
		C.allocate_64(ctx.ctx64, C.uint64_t(m), C.uint64_t(k), C.uint64_t(n))
	default:
		panic("Unreachable")
	}
}

func (ctx *Context[T]) SetBatch(n uint64) {
	ctx.outCols = n
	switch T(0).Bitlen() {
	case 32:
		C.set_batch_32(ctx.ctx32, C.uint64_t(n))
	case 64:
		C.set_batch_64(ctx.ctx64, C.uint64_t(n))
	default:
		panic("Unreachable")
	}
}

// Sets the database on the GPU
//
// This is a different function since the DB is always represented with 32-bit
// uints
func (ctx *Context[T]) SetA(src *m.Matrix[m.Elem32]) {
	hostPtr := ctx.GetHostData(0)
	for i, val := range src.Data() {
		*(*m.Elem32)(unsafe.Add(hostPtr, i*4)) = val
	}
	ctx.SyncDevice(0)
}

// Sets the B matrix on the GPU. This is stored in a column-major format
func (ctx *Context[T]) SetB(src *m.Matrix[T], offset int, transpose bool, sync bool) {
	hostPtr := ctx.GetHostData(1)
	bytes := (int)(T(0).Bitlen() / 8)

	if transpose {
		for i := uint64(0); i < src.Rows(); i++ {
			for j := uint64(0); j < src.Cols(); j++ {
				idx := int(j*src.Rows() + i)
				*(*T)(unsafe.Add(hostPtr, (offset+idx)*bytes)) = src.Get(i, j)
			}
		}
	} else {
		for i, val := range src.Data() {
			*(*T)(unsafe.Add(hostPtr, (offset+i)*bytes)) = val
		}
	}

	if sync {
		ctx.SyncDevice(1)
	}
}

func (ctx *Context[T]) SetC(src *m.Matrix[T]) {
	hostPtr := ctx.GetHostData(2)
	bytes := (int)(T(0).Bitlen() / 8)

	for i, val := range src.Data() {
		*(*T)(unsafe.Add(hostPtr, i*bytes)) = val
	}
	ctx.SyncDevice(2)
}

func (ctx *Context[T]) SyncDevice(index int) {
	switch T(0).Bitlen() {
	case 32:
		C.sync_device_32(ctx.ctx32, C.int(index))
	case 64:
		C.sync_device_64(ctx.ctx64, C.int(index))
	default:
		panic("Unreachable")
	}
}

func (ctx *Context[T]) GEMM() *m.Matrix[T] {
	// Run GEMM
	switch T(0).Bitlen() {
	case 32:
		C.gemm_32(ctx.ctx32)
	case 64:
		C.gemm_64(ctx.ctx64)
	default:
		panic("Unreachable")
	}

	// Copy result to a new matrix value
	//
	// TODO: Could do this without copying if a bottleneck
	hostPtr := ctx.GetHostData(3)
	bytes := (int)(T(0).Bitlen() / 8)
	data := make([]T, ctx.outRows*ctx.outCols)
	for i := range data {
		data[i] = *(*T)(unsafe.Add(hostPtr, i*bytes))
	}
	return m.NewFromRaw[T](data, ctx.outRows, ctx.outCols)
}

func (ctx *Context[T]) GetHostData(index int) unsafe.Pointer {
	if index == 0 {
		switch T(0).Bitlen() {
		case 32:
			return unsafe.Pointer(C.get_host_a_32(ctx.ctx32))
		case 64:
			return unsafe.Pointer(C.get_host_a_64(ctx.ctx64))
		default:
			panic("Unreachable")
		}
	} else {
		switch T(0).Bitlen() {
		case 32:
			return unsafe.Pointer(C.get_host_data_32(ctx.ctx32, C.int(index)))
		case 64:
			return unsafe.Pointer(C.get_host_data_64(ctx.ctx64, C.int(index)))
		default:
			panic("Unreachable")
		}
	}
}

func UseGPU() bool {
	return (bool)(C.use_gpu())
}
