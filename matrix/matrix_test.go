package matrix

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/ryanleh/secure-inference/crypto/rand"
)

func testGob[U Elem](t *testing.T) {
	rand := rand.NewRandomBufPRG()
	m := Rand[U](rand, 5, 5, 0)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	dec := gob.NewDecoder(&buf)
	var n Matrix[U]
	err = dec.Decode(&n)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if !m.Equals(&n) {
		m.Print()
		n.Print()
		t.Fail()
	}
}

func TestGob32(t *testing.T) {
	testGob[Elem32](t)
}

func TestGob64(t *testing.T) {
	testGob[Elem64](t)
}

func testToFile[U Elem](t *testing.T, fn string) {
	rand := rand.NewRandomBufPRG()

	m := Rand[U](rand, 5, 5, 0)
	err := m.WriteToFile(fn)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	n := new(Matrix[U])
	err = n.ReadFromFile(fn)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if !m.Equals(n) {
		m.Print()
		n.Print()
		t.Fail()
	}
}

func TestToFile32(t *testing.T) {
	testToFile[Elem32](t, "test32.log")
}

func TestToFile64(t *testing.T) {
	testToFile[Elem64](t, "test64.log")
}

func testAdd[U Elem](t *testing.T, r1 uint64, c1 uint64) {
	rand := rand.NewRandomBufPRG()

	m := Rand[U](rand, r1, c1, 0)
	z := Zeros[U](r1, c1)

	if !z.Equals(z) {
		t.Fail()
	}

	z.Add(m)
	if !z.Equals(m) {
		t.Fail()
	}
}

func TestAdd32(t *testing.T) {
	testAdd[Elem32](t, 2, 2)
}

func TestAdd64(t *testing.T) {
	testAdd[Elem64](t, 72, 110)
}

func testMul[U Elem](t *testing.T, r1 uint64, c1 uint64, r2 uint64, c2 uint64) {
	// First, test regular multiplication
	rand1 := rand.NewRandomBufPRG()

	m1 := Rand[U](rand1, r1, c1, 0)
	m2 := Rand[U](rand1, r2, c2, 0)
	z := Zeros[U](r2, c2)
	zout := Zeros[U](r1, c2)

	z2 := Mul(m1, z)
	if !z2.Equals(zout) {
		t.Fail()
	}

	out := Mul(m1, m2)
	res := Zeros[U](r1, c2)
	for i := uint64(0); i < r1; i++ {
		for j := uint64(0); j < c2; j++ {
			tmp := U(0)
			for k := uint64(0); k < c1; k++ {
				tmp += (U(m1.Get(i, k)) * U(m2.Get(k, j)))
			}
			res.Set(i, j, tmp)
		}
	}

	if !out.Equals(res) {
		fmt.Println("Failed regular")
		t.Fail()
	}

	// Test left-seeded multiplication
	key := rand.RandomPRGKey()
	rand2 := rand.NewBufPRG(rand.NewPRG(key))
	rand3 := rand.NewBufPRG(rand.NewPRG(key))
	m3 := Rand[U](rand2, r1, c1, 0)
	m4 := &MatrixSeeded[U]{
		src:  []IoRandSource{rand3},
		rows: []uint64{r1},
		cols: c1,
	}

	z3 := Mul(m3, m2)
	z4 := MulSeededLeft(m4, m2)

	if !z3.Equals(z4) {
		fmt.Println("Failed left-seeded")
		z3.Print()
		z4.Print()
		t.Fail()
	}

	// Test left-seeded multiplication with multiple keys
	key2 := rand.RandomPRGKey()
	rand6 := rand.NewBufPRG(rand.NewPRG(key))
	rand7 := rand.NewBufPRG(rand.NewPRG(key2))
	rand8 := rand.NewBufPRG(rand.NewPRG(key2))
	m3.Concat(Rand[U](rand7, r1, c1, 0))
	m7 := &MatrixSeeded[U]{
		src:  []IoRandSource{rand6, rand8},
		rows: []uint64{r1, r1},
		cols: c1,
	}

	z7 := Mul(m3, m2)
	z8 := MulSeededLeft(m7, m2)

	if !z7.Equals(z8) {
		fmt.Println("Failed seeded with multiple keys")
		t.Fail()
	}
}

func TestMul32(t *testing.T) {
	testMul[Elem32](t, 2, 8, 8, 7)
}

func TestMul64(t *testing.T) {
	testMul[Elem64](t, 2, 8, 8, 7)
}

func TestMulVec32(t *testing.T) {
	testMul[Elem32](t, 60, 83, 83, 1)
}

func TestMulVec64(t *testing.T) {
	testMul[Elem64](t, 60, 83, 83, 1)
}

func testGauss[U Elem](t *testing.T, r1 uint64, c1 uint64) {
	rand := rand.NewRandomBufPRG()
	Gaussian[U](rand, r1, c1)
}

func TestGauss32(t *testing.T) {
	testGauss[Elem32](t, 2, 8)
}

func TestGauss64(t *testing.T) {
	testGauss[Elem64](t, 2, 8)
}

func testMulPacked[U Elem](t *testing.T, r1 uint64, c1 uint64) {
	rand := rand.NewRandomBufPRG()

	m2 := Rand[U](rand, c1, 1, 0)
	m1 := Rand[U](rand, r1, c1, 1<<m2.SquishBasis())

	res1 := Mul(m1, m2)
	m1.Squish()

	newCols := m1.Cols() * m1.SquishRatio()
	m2.AppendZeros(newCols - m2.Rows())

	res2 := MulVecPacked(m1, m2)

	if !res1.Equals(res2) {
		t.Fail()
	}
}

func TestMulVecPacked32(t *testing.T) {
	testMulPacked[Elem32](t, 8, 13)
}

func TestMulPacked64(t *testing.T) {
	testMulPacked[Elem64](t, 8, 13)
}

func TestMulVecPackedBig32(t *testing.T) {
	testMulPacked[Elem32](t, 812, 1391)
}

func TestMulPackedBig64(t *testing.T) {
	testMulPacked[Elem64](t, 810, 132)
}
