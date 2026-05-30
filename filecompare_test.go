package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

// TestEqualFiles_SkipHeaderAndFooter_FooterLeak verifies that with both
// --skip-header and --skip-footer set, the footer region is excluded from
// comparison. The LimitReader limit must subtract SkipHeader as well — the
// bytes consumed by the seek do not reduce st.Size() — otherwise the reader
// reads SkipHeader extra bytes from the footer region.
func TestEqualFiles_SkipHeaderAndFooter_FooterLeak(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()

	// File layout (20 bytes): [header:5][body:10][footer:5]
	// With SkipHeader=5 and SkipFooter=5, only the 10-byte body should be compared.
	header := []byte("HHHHH")
	body := bytes.Repeat([]byte("B"), 10)

	file1Data := append(append(append([]byte{}, header...), body...), []byte("XXXXX")...)
	file2Data := append(append(append([]byte{}, header...), body...), []byte("YYYYY")...)

	f1 := filepath.Join(dir, "file1")
	f2 := filepath.Join(dir, "file2")
	assert.NoError(os.WriteFile(f1, file1Data, 0644))
	assert.NoError(os.WriteFile(f2, file2Data, 0644))

	st1, err := os.Stat(f1)
	assert.NoError(err)
	st2, err := os.Stat(f2)
	assert.NoError(err)

	r1 := newFileRecord(f1, st1, f1, "")
	r2 := newFileRecord(f2, st2, f2, "")

	o := &options{SkipHeader: 5, SkipFooter: 5}

	// Files differ only in the footer, which should be skipped.
	assert.True(equalFiles(r1, r2, o))
}
