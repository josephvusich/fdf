package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// eofWithDataReader is an io.Reader that returns data alongside io.EOF on the
// final read, which is permitted by the io.Reader contract.
type eofWithDataReader struct {
	data   []byte
	offset int
}

func (r *eofWithDataReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	if r.offset >= len(r.data) {
		return n, io.EOF // data and EOF together — valid per io.Reader contract
	}
	return n, nil
}

// TestEqualReaders_EOFWithData_DifferentFinalChunk ensures equalReaders
// compares data returned alongside io.EOF. Two readers that differ only in
// their final bytes — delivered in the same Read as io.EOF — must be reported
// as not equal.
func TestEqualReaders_EOFWithData_DifferentFinalChunk(t *testing.T) {
	assert := require.New(t)

	data1 := []byte("identical prefix, different end: A")
	data2 := []byte("identical prefix, different end: B")

	r1 := &eofWithDataReader{data: data1}
	r2 := &eofWithDataReader{data: data2}

	assert.False(equalReaders(r1, r2))
}

// TestEqualReaders_EOFWithData_IdenticalContent verifies that two identical
// eofWithDataReaders are still reported as equal (sanity check).
func TestEqualReaders_EOFWithData_IdenticalContent(t *testing.T) {
	assert := require.New(t)

	data := []byte("exactly the same content")

	r1 := &eofWithDataReader{data: data}
	r2 := &eofWithDataReader{data: data}

	assert.True(equalReaders(r1, r2))
}

// TestEqualReaders_EOFWithData_MultiChunkDifferentFinal exercises the same
// EOF-with-data path across multiple read rounds, ensuring the final-chunk
// comparison still catches differences after a long identical prefix.
func TestEqualReaders_EOFWithData_MultiChunkDifferentFinal(t *testing.T) {
	assert := require.New(t)

	// Build data larger than the internal buffer (0xFFFFF) so there is at
	// least one full read round before the final data+EOF read.
	prefix := bytes.Repeat([]byte("x"), 0xFFFFF+100)

	data1 := append(append([]byte{}, prefix...), 'A')
	data2 := append(append([]byte{}, prefix...), 'B')

	r1 := &eofWithDataReader{data: data1}
	r2 := &eofWithDataReader{data: data2}

	assert.False(equalReaders(r1, r2))
}
