package rlwe

// For this code to work, SEAL needs to be built via:
// ```
//  cd secure-inference/external/SEAL
//  cmake -S . -B build -DCMAKE_INSTALL_PREFIX=$PWD/build
//  cmake --build build -j 8
//  cmake --install build
// ```

// #cgo CXXFLAGS: -std=c++20 -I/opt/homebrew/include -I${SRCDIR}/../../external/SEAL/build/include/SEAL-4.1 -Wall -Werror -O3 -march=native
// #cgo LDFLAGS: -L/opt/homebrew/lib -L${SRCDIR}/../../external/SEAL/build/lib -lseal-4.1 -lgmp
// #include "ffi.h"
import "C"

import (
	"math"
	"unsafe"

	m "github.com/ryanleh/secure-inference/matrix"
)

const SealSeedLength = 8

type Context[T m.Elem] struct {
	ctx *C.context_t
}

type Ciphertext C.ciphertext_t

type Plaintext C.plaintext_t

type A C.a_t

type Key struct {
	key *C.skey_t
}

/*
 * ======== Context ========
 */

func NewContext[T m.Elem](pMod, n uint64, mod_switch bool) *Context[T] {
	// Default plaintext modulus
	//
	// TODO: Move this to params
	if pMod == 0 {
		pMod = 0
		pMod = 95640378
	}

	return &Context[T]{
		ctx: C.ctx_new(
			C.uint64_t(pMod),
			C.uint64_t(n),
			C.uint64_t(T(0).Bitlen()),
			C.bool(mod_switch),
		),
	}
}

func (ctx *Context[T]) Free() {
	C.ctx_free(ctx.ctx)
}

func (ctx *Context[T]) N() uint64 {
	return uint64(C.ctx_n(ctx.ctx))
}

func (ctx *Context[T]) P() uint64 {
	return uint64(C.ctx_p(ctx.ctx))
}

/*
 * ======== Secret Key ========
 */

func (ctx *Context[T]) NewKey() *Key {
	return &Key{
		key: C.key_new(ctx.ctx),
	}
}

func (key *Key) Free() {
	C.key_free(key.key)
}

func (key *Key) Size() int {
	return int(C.key_size(key.key))
}

/*
 * ======== Ciphertext ========
 */

func NewCiphertext() *Ciphertext {
	return (*Ciphertext)(C.ct_new())
}

func (ct *Ciphertext) Free() {
	C.ct_free((*C.ciphertext_t)(ct))
}
func (ct *Ciphertext) Size() int {
	return int(C.ct_size((*C.ciphertext_t)(ct)))
}

/*
 * ======== A Poly ========
 */

func NewA(key *Key, seed []uint64) *A {
	if len(seed) != 8 {
		panic("SEAL seed must be 512-bits")
	}
	return (*A)(C.a_new(key.key, (*C.uint64_t)(&seed[0])))
}

func (a *A) Free() {
	C.a_free((*C.a_t)(a))
}

/*
 * ======== Plaintext ========
 */

func NewPlaintext() *Plaintext {
	return (*Plaintext)(C.pt_new())
}

func (pt *Plaintext) Set(vals []uint64) {
	C.pt_set_64((*C.plaintext_t)(pt), (*C.uint64_t)(&vals[0]), C.size_t(len(vals)))
}

// Work-around to set plaintext from genericly-typed values
func GenericSet[T m.Elem](pt *Plaintext, vals []T) {
	switch T(0).Bitlen() {
	case 32:
		C.pt_set_32(
			(*C.plaintext_t)(pt),
			(*C.uint32_t)(unsafe.Pointer(&vals[0])),
			C.size_t(len(vals)),
		)
	case 64:
		C.pt_set_64(
			(*C.plaintext_t)(pt),
			(*C.uint64_t)(unsafe.Pointer(&vals[0])),
			C.size_t(len(vals)),
		)
	default:
		panic("Unreachable")
	}
}

func (pt *Plaintext) Free() {
	C.pt_free((*C.plaintext_t)(pt))
}

/*
 * ======== Hybrid Mode ========
 */

// This function computes a product of the form `H_i = D * a_i`, where the
// various `i`s are stacked vertically. This is slightly tricky since
// we know how to _right-multiplication_ of a polynomial by a matrix over a
// negacyclic ring, but not left. To get around this we instead compute
// `(a_i)^T * D^T` and then transpose the final product: observing that the
// "transpose" operator on a polynomial here corresponds to a substitution
// operation.
func (ctx *Context[T]) ComputeHint(matrix *m.Matrix[m.Elem32], seeds []uint64, numSeeds int) *m.Matrix[T] {
	hint := m.New[T](matrix.Rows(), ctx.N())

	switch T(0).Bitlen() {
	case 32:
		C.mul_matrix_As_32(
			ctx.ctx,
			(*C.uint32_t)(unsafe.Pointer(&matrix.Data()[0])), //TODO
			(*C.uint64_t)(&seeds[0]),
			(*C.uint32_t)(unsafe.Pointer(&hint.Data()[0])),
			(C.uint64_t)(matrix.Rows()),
			(C.uint64_t)(matrix.Cols()),
			(C.uint64_t)(numSeeds),
		)
	case 64:
		C.mul_matrix_As_64(
			ctx.ctx,
			(*C.uint32_t)(unsafe.Pointer(&matrix.Data()[0])), //TODO
			(*C.uint64_t)(&seeds[0]),
			(*C.uint64_t)(unsafe.Pointer(&hint.Data()[0])),
			(C.uint64_t)(matrix.Rows()),
			(C.uint64_t)(matrix.Cols()),
			(C.uint64_t)(numSeeds),
		)
	default:
		panic("Unreachable")
	}
	return hint
}

func (ctx *Context[T]) ExtractLWEKey(key *Key) *m.Matrix[T] {
	skLWE := m.New[T](ctx.N(), 1)

	switch T(0).Bitlen() {
	case 32:
		C.key_extract_lwe_32(
			ctx.ctx,
			(*C.skey_t)(key.key),
			(*C.uint32_t)(unsafe.Pointer(&skLWE.Data()[0])),
		)
	case 64:
		C.key_extract_lwe_64(
			ctx.ctx,
			(*C.skey_t)(key.key),
			(*C.uint64_t)(unsafe.Pointer(&skLWE.Data()[0])),
		)
	default:
		panic("Unreachable")
	}
	return skLWE
}

func (ctx *Context[T]) ExtractLWECt(in []byte, samples uint64) *m.Matrix[T] {
	ctLWE := m.New[T](samples, 1)

	switch T(0).Bitlen() {
	case 32:
		C.ct_extract_lwe_32(
			ctx.ctx,
			(*C.uint8_t)(&in[0]),
			C.size_t(len(in)),
			(C.uint64_t)(samples),
			(*C.uint32_t)(unsafe.Pointer(&ctLWE.Data()[0])),
		)
	case 64:
		C.ct_extract_lwe_64(
			ctx.ctx,
			(*C.uint8_t)(&in[0]),
			C.size_t(len(in)),
			(C.uint64_t)(samples),
			(*C.uint64_t)(unsafe.Pointer(&ctLWE.Data()[0])),
		)
	default:
		panic("Unreachable")
	}
	return ctLWE
}

func (ctx *Context[T]) ExtractLWECtGPU(in []byte, samples uint64, out unsafe.Pointer, offset int) {
	switch T(0).Bitlen() {
	case 32:
		C.ct_extract_lwe_32(
			ctx.ctx,
			(*C.uint8_t)(&in[0]),
			C.size_t(len(in)),
			(C.uint64_t)(samples),
			(*C.uint32_t)(unsafe.Add(out, offset*4)),
		)
	case 64:
		C.ct_extract_lwe_64(
			ctx.ctx,
			(*C.uint8_t)(&in[0]),
			C.size_t(len(in)),
			(C.uint64_t)(samples),
			(*C.uint64_t)(unsafe.Add(out, offset*8)),
		)
	default:
		panic("Unreachable")
	}
}

func (key *Key) PreprocessEnc(a *A, ct *Ciphertext) {
	C.key_preprocess_enc(key.key, (*C.a_t)(a), (*C.ciphertext_t)(ct))
}

func (ctx *Context[T]) EncryptPreprocessed(key *Key, input []T, ct *Ciphertext) {
	pt := NewPlaintext()
	defer pt.Free()
	GenericSet[T](pt, input)
	C.key_enc_preprocessed(key.key, (*C.plaintext_t)(pt), (*C.ciphertext_t)(ct))

	// Truncate to the size of input
	if uint64(len(input)) < ctx.N() {
		ctx.TruncateCT(ct, len(input))
	}
}

func (ctx *Context[T]) TruncateCT(ct *Ciphertext, size int) {
	C.truncate_ct(ctx.ctx, (*C.ciphertext_t)(ct), C.size_t(size))
}

// TODO: This is messy and should be rewritten
func (ctx *Context[T]) StoreRandomCTs(samples uint64, seed []uint64) [][]byte {
	if len(seed) != 8 {
		panic("SEAL seed must be 512-bits")
	}
	size := uint64(C.dummy_ct_size(ctx.ctx))
	num := uint64(math.Ceil(float64(samples) / float64(ctx.N())))
	buf := make([]byte, size*num)
	actualSizes := make([]uint64, num)
	C.store_dummy_cts(
		ctx.ctx,
		(*C.uint64_t)(&seed[0]),
		(C.size_t)(samples),
		(*C.uint8_t)(&buf[0]),
		(*C.size_t)(&actualSizes[0]),
	)

	results := make([][]byte, num)
	for i := range results {
		results[i] = buf[i*int(size) : i*int(size)+int(actualSizes[i])]
	}
	return results
}

func (ct *Ciphertext) StoreData() []byte {
	size := int(C.ct_data_size((*C.ciphertext_t)(ct)))
	out := make([]byte, size)
	actualSize := uint64(0)
	C.ct_store_data(
		(*C.ciphertext_t)(ct),
		(*C.uint8_t)(&out[0]),
		C.size_t(len(out)),
		(*C.size_t)(&actualSize),
	)
	return out[:actualSize]
}

func (ctx *Context[T]) RoundLWEInplace(noisyResult *m.Matrix[T]) {
	switch T(0).Bitlen() {
	case 32:
		C.round_lwe_32(
			ctx.ctx,
			(*C.uint32_t)(unsafe.Pointer(&noisyResult.Data()[0])),
			(C.size_t)(noisyResult.Size()),
		)
	case 64:
		C.round_lwe_64(
			ctx.ctx,
			(*C.uint64_t)(unsafe.Pointer(&noisyResult.Data()[0])),
			(C.size_t)(noisyResult.Size()),
		)
	default:
		panic("Unreachable")
	}
}

func (ctx *Context[T]) LiftPlainInplace(plain *m.Matrix[T]) {
	switch T(0).Bitlen() {
	case 32:
		C.lift_lwe_32(
			ctx.ctx,
			(*C.uint32_t)(unsafe.Pointer(&plain.Data()[0])),
			(C.size_t)(plain.Size()),
		)
	case 64:
		C.lift_lwe_64(
			ctx.ctx,
			(*C.uint64_t)(unsafe.Pointer(&plain.Data()[0])),
			(C.size_t)(plain.Size()),
		)
	default:
		panic("Unreachable")
	}
}
