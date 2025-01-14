package matrix

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

func (m *Matrix[T]) Print() {
	fmt.Printf("%d-by-%d matrix:\n", m.rows, m.cols)
	for i := uint64(0); i < m.rows; i++ {
		for j := uint64(0); j < m.cols; j++ {
			fmt.Printf("%d ", m.data[i*m.cols+j])
		}
		fmt.Printf("\n")
	}
}

func (m *Matrix[T]) PrintStart() {
	fmt.Printf("%d-by-%d matrix:\n", m.rows, m.cols)
	for i := uint64(0); i < 2; i++ {
		for j := uint64(0); j < 2; j++ {
			fmt.Printf("%d ", m.data[i*m.cols+j])
		}
		fmt.Printf("\n")
	}
}

func (m Matrix[T]) GobEncode() ([]byte, error) {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	err1 := encoder.Encode(m.rows)
	err2 := encoder.Encode(m.cols)
	err3 := encoder.Encode(m.data)

	if err1 != nil || err2 != nil || err3 != nil {
		panic("Gob encoding error")
	}

	return buf.Bytes(), nil
}

func (m *Matrix[T]) GobDecode(buf []byte) error {
	b := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(b)
	err1 := decoder.Decode(&m.rows)
	err2 := decoder.Decode(&m.cols)

	m.data = make([]T, m.rows*m.cols)
	err3 := decoder.Decode(&m.data)

	if err1 != nil || err2 != nil || err3 != nil {
		panic("Gob decoding error")
	}

	return nil
}

func (m *Matrix[T]) WriteToFile(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return m.WriteToFileDescriptor(f)
}
func (m *Matrix[T]) WriteToFileDescriptor(f *os.File) error {
	_, err := fmt.Fprintf(f, "%d,%d\n", m.rows, m.cols)
	if err != nil {
		return err
	}

	if m.rows*m.cols != uint64(len(m.data)) {
		panic("Rows/cols do not match data size")
	}

	for _, elem := range m.data {
		_, err = fmt.Fprintf(f, "%d,", elem)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Matrix[T]) ReadFromFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return m.ReadFromFileDescriptor(f)
}

func (m *Matrix[T]) ReadFromFileDescriptor(f *os.File) error {
	_, err := fmt.Fscanf(f, "%d,%d\n", &m.rows, &m.cols)
	if err != nil {
		return err
	}

	m.data = make([]T, m.rows*m.cols)
	for i := uint64(0); i < m.rows*m.cols; i++ {
		_, err = fmt.Fscanf(f, "%d,", &m.data[i])
		if err != nil {
			return err
		}
	}

	return nil
}
