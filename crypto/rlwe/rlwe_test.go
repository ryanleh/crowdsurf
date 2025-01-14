package rlwe

import (
	"encoding/binary"
	m "github.com/ryanleh/secure-inference/matrix"
	"io"
	"math"
	"math/rand"
	"testing"
)

func TestContext(t *testing.T) {
	ctx := NewContext[m.Elem32](0, 4096, false)
	defer ctx.Free()
	if ctx.N() <= 100 {
		t.FailNow()
	}
}

func TestPlaintext(t *testing.T) {
	ctx := NewContext[m.Elem32](0, 4096, false)
	defer ctx.Free()
	pt := NewPlaintext()
	defer pt.Free()
}

func TestCiphertext(t *testing.T) {
	ct := NewCiphertext()
	defer ct.Free()
}

func BenchmarkModSwitch(b *testing.B) {
	rng := rand.New(rand.NewSource(99))
	ctx := NewContext[m.Elem32](0, 4096, false)
	defer ctx.Free()

	ct := NewCiphertext()
	defer ct.Free()

	pt := m.Rand[m.Elem32](rng, ctx.N(), 1, ctx.P())

	// Generate A matrix
	key := ctx.NewKey()
	A := NewA(key, make([]uint64, 8))
	defer key.Free()

	// Encrypt the plaintext
	key.PreprocessEnc(A, ct)
	ctx.EncryptPreprocessed(key, pt.Data(), ct)
	ctSer := ct.StoreData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.ExtractLWECt(ctSer, ctx.N())
	}
}

func testHybrid[T m.Elem](t *testing.T, ctx *Context[T], rows, cols, pMod uint64) {
	rng := rand.New(rand.NewSource(99))

	// Build kernel
	kernel := m.Rand[m.Elem32](rng, rows, cols, pMod)

	// Sample seeds for A poly
	num := int(math.Ceil(float64(cols) / float64(ctx.N())))
	seeds := make([]uint64, SealSeedLength*num)
	buf := make([]byte, 8)
	for i := range seeds {
		io.ReadFull(rng, buf)
		seeds[i] = binary.LittleEndian.Uint64(buf[:])
	}

	// Compute hint
	hint := ctx.ComputeHint(kernel, seeds, num)

	// Define and encrypt an input
	input := m.Rand[T](rng, cols, 1, pMod)
	cts := make([]*Ciphertext, num)
	for i := range cts {
		cts[i] = NewCiphertext()
		defer cts[i].Free()
	}

	key := ctx.NewKey()
	defer key.Free()
	for i := range num {
		// Build A
		seed := seeds[i*SealSeedLength : (i+1)*SealSeedLength]
		a := NewA(key, seed)
		defer a.Free()

		// Grab relevant input
		start := uint64(i) * ctx.N()
		end := min(uint64(i+1)*ctx.N(), input.Size())
		data := input.Data()[start:end]

		// Encrypt
		key.PreprocessEnc(a, cts[i])
		ctx.EncryptPreprocessed(key, data, cts[i])
	}

	// Modulus switch the ciphertext and perform evaluation
	resultCT := m.New[T](0, 0)
	for i := range num {
		numSamples := min(cols-uint64(i)*ctx.N(), ctx.N())
		resultCT.Concat(ctx.ExtractLWECt(cts[i].StoreData(), numSamples))
	}

	resultCT = m.MulVec(kernel, resultCT)

	// Attempt to decrypt the result and check that we're correct
	token := m.Mul(hint, ctx.ExtractLWEKey(key))
	resultCT.Sub(token)
	ctx.RoundLWEInplace(resultCT)

	expected := m.MulVec(kernel, input)
	expected.ReduceMod(pMod)
	if !expected.Equals(resultCT) {
		t.Fail()
	}
}

func TestHybrid32(t *testing.T) {
	pMod := uint64(1 << 7)
	ctx := NewContext[m.Elem32](pMod, 2048, true)
	testHybrid[m.Elem32](t, ctx, ctx.N(), ctx.N(), pMod)
	testHybrid[m.Elem32](t, ctx, ctx.N(), 2*ctx.N(), pMod)
	testHybrid[m.Elem32](t, ctx, ctx.N(), 2*ctx.N()+5, pMod)
}

func TestHybrid64(t *testing.T) {
	pMod := uint64(1 << 15)
	ctx := NewContext[m.Elem64](pMod, 4096, true)
	testHybrid[m.Elem64](t, ctx, ctx.N(), ctx.N(), pMod)
	testHybrid[m.Elem64](t, ctx, ctx.N(), 2*ctx.N(), pMod)
	testHybrid[m.Elem64](t, ctx, ctx.N(), 2*ctx.N()+5, pMod)
}
