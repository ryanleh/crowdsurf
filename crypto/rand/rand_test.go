package rand

import (
	"bytes"
	//"log"
	"io"
	"testing"
)

func TestPRG(t *testing.T) {
	key := RandomPRGKey()
	prg := NewPRG(key)

	var buf [16]byte
	b, err := prg.Read(buf[:])
	if err != nil || b == 0 {
		t.Fail()
	}

	if buf[0] == 0 &&
		buf[1] == 0 &&
		buf[2] == 0 &&
		buf[3] == 0 &&
		buf[4] == 0 &&
		buf[5] == 0 &&
		buf[6] == 0 &&
		buf[7] == 0 &&
		buf[8] == 0 &&
		buf[9] == 0 &&
		buf[10] == 0 {
		t.Fatal("Bad randomness")
	}
	prg.Read(buf[:])

	var buf2 [16]byte
	prg2 := NewPRG(key)
	prg2.Read(buf2[:])
	prg2.Read(buf2[:])

	if !bytes.Equal(buf[:], buf2[:]) {
		t.Fail()
	}

	prg2.Read(buf2[:])

	if bytes.Equal(buf[:], buf2[:]) {
		t.Fail()
	}
}

/*
func TestFill(t *testing.T) {
  key := RandomPRGKey()
  prg := NewPRG(key)

  // 4 GB
  buf := make([]byte, 1024*1024*1024*4)
  prg.Read(buf[:])
}*/

func TestFillIO(t *testing.T) {
	key := RandomPRGKey()
	prg := NewPRG(key)

	// 4 GB
	buf := make([]byte, 1024*1024*1024*4)
	io.ReadFull(prg, buf[:])
}
