package gpu

import (
	"github.com/ryanleh/secure-inference/crypto/rand"
	"github.com/ryanleh/secure-inference/matrix"
	"testing"
)

func testGEMM[T matrix.Elem](t *testing.T, m, k, n uint64) {
	prg := rand.NewRandomBufPRG()

	for i := 0; i < 10; i++ {
		// Generate random matrices and compute expected product on CPU
		A := matrix.Rand[matrix.Elem32](prg, m, k, 0)
		B := matrix.Rand[T](prg, k, n, 0)
		C := matrix.Rand[T](prg, m, n, 0)

		// Compute the expected result (copy needed for types on matrix API)
		expected := matrix.Mul(A, B)
		expected.Add(C)

		// Now perform the same computation on the GPU
		//
		// Allocate GPU memory for the computation
		ctx := NewContext[T](m, k, n)
		defer ctx.Free()

		// Set A, B, C on the GPU
		ctx.SetA(A)
		ctx.SetB(B, 0, true, true)
		ctx.SetC(C)

		// Perform the calculation
		result := ctx.GEMM()

		// Check the result
		if !result.Equals(expected) {
			t.Fail()
		}

		// Do the same thing but change the batch size
		B = matrix.Rand[T](prg, k, n+100, 0)
		C = matrix.Rand[T](prg, m, n+100, 0)

		expected = matrix.Mul(A, B)
		expected.Add(C)

		ctx.SetBatch(n + 100)
		ctx.SetB(B, 0, true, true)
		ctx.SetC(C)

		result = ctx.GEMM()
		if !result.Equals(expected) {
			t.Fail()
		}
	}
}

func TestGEMM32(t *testing.T) {
	testGEMM[matrix.Elem32](t, 1024, 1024, 1024)
}

func TestGEMM64(t *testing.T) {
	testGEMM[matrix.Elem64](t, 1024, 1024, 1024)
}
