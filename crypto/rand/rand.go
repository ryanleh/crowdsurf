// Code taken from: https://github.com/henrycg/prio/blob/master/utils/rand.go
/*

Copyright (c) 2016,2023 Henry Corrigan-Gibbs

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

*/

package rand

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"math/big"
	mrand "math/rand"
)

type PRGKey [aes.BlockSize]byte

const bufSize = 8192

func (r *BufPRGReader) MathRand() *mrand.Rand {
	return mrand.New(r)
}

// We use the AES-CTR to generate pseudo-random  numbers using a
// stream cipher. Go's native rand.Reader is extremely slow because
// it makes tons of system calls to generate a small number of
// pseudo-random bytes.
type PRGReader struct {
	Key   PRGKey
	ctr   uint64
	block cipher.Block
}

type BufPRGReader struct {
	mrand.Source64
	Key    PRGKey
	stream *bufio.Reader
}

func NewPRG(key *PRGKey) *PRGReader {
	out := new(PRGReader)
	out.Key = *key

	var err error

	out.block, err = aes.NewCipher(key[:])
	if err != nil {
		panic(err)
	}

	return out
}

func RandomPRGKey() *PRGKey {
	var key PRGKey
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}

	return &key
}

func RandomPRG() *PRGReader {
	return NewPRG(RandomPRGKey())
}

func (s *PRGReader) Read(p []byte) (int, error) {
	var buf [aes.BlockSize]byte

	for done := 0; done < len(p); done += aes.BlockSize {
		s.ctr += 1
		binary.BigEndian.PutUint64(buf[:], s.ctr)

		if len(p[done:]) >= aes.BlockSize {
			s.block.Encrypt(p[done:], buf[:])
		} else {
			s.block.Encrypt(buf[:], buf[:])
			copy(p[done:], buf[:])
		}
	}

	return len(p), nil
}

func NewBufPRG(prg *PRGReader) *BufPRGReader {
	out := new(BufPRGReader)
	out.Key = prg.Key
	out.stream = bufio.NewReaderSize(prg, bufSize)
	return out
}

func NewRandomBufPRG() *BufPRGReader {
	return NewBufPRG(NewPRG(RandomPRGKey()))
}

func (b *BufPRGReader) RandInt(mod *big.Int) *big.Int {
	out, err := rand.Int(b.stream, mod)
	if err != nil {
		// TODO: Replace this with non-absurd error handling.
		panic("Catastrophic randomness failure!")
	}
	return out
}

func (b *BufPRGReader) GenPRGKey() *PRGKey {
	var key PRGKey
	_, err := io.ReadFull(b, key[:])
	if err != nil {
		panic(err)
	}

	return &key
}

func (b *BufPRGReader) Read(p []byte) (int, error) {
	return b.stream.Read(p)
}

func (b *BufPRGReader) Int63() int64 {
	uout := b.Uint64()
	uout = uout % (1 << 63)
	return int64(uout)
}

func (b *BufPRGReader) Uint64() uint64 {
	var buf [8]byte

	read := 0
	for read < 8 {
		n, err := b.stream.Read(buf[read:8])
		if err != nil {
			panic("Should never get here")
		}
		read += n
	}

	return binary.LittleEndian.Uint64(buf[:])
}

func (b *BufPRGReader) Seed(int64) {
	panic("Should never call seed")
}
