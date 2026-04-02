package main

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/minio/highwayhash"
)

func areHardlinked(r1, r2 *fileRecord) bool {
	if os.SameFile(r1.FileInfo, r2.FileInfo) {
		r1.everMatchedContent = true
		r2.everMatchedContent = true
		return true
	}
	return false
}

func equalFiles(r1, r2 *fileRecord, o *options) bool {
	f1, err := o.OpenFile(r1.FilePath)
	if err != nil {
		return false
	}
	defer f1.Close()

	f2, err := o.OpenFile(r2.FilePath)
	if err != nil {
		return false
	}
	defer f2.Close()

	var h1, h2 hash.Hash
	var reader1, reader2 io.Reader = f1, f2

	if r1.UnverifiedChecksum {
		h1, _ = highwayhash.New128(hashKeyFromSize(r1.Size()))
		reader1 = io.TeeReader(f1, h1)
	}
	if r2.UnverifiedChecksum {
		h2, _ = highwayhash.New128(hashKeyFromSize(r2.Size()))
		reader2 = io.TeeReader(f2, h2)
	}

	if equalReaders(reader1, reader2) {
		r1.everMatchedContent = true
		r2.everMatchedContent = true
		if h1 != nil {
			verifyCachedHash(r1, h1.Sum(nil), o)
		}
		if h2 != nil {
			verifyCachedHash(r2, h2.Sum(nil), o)
		}
		return true
	}
	return false
}

func verifyCachedHash(r *fileRecord, computed []byte, o *options) {
	r.UnverifiedChecksum = false

	if bytes.Equal(r.Checksum.hash[:], computed) {
		return
	}

	fmt.Printf("warning: %s: cached hash mismatch, correcting\n", r.FilePath)
	copy(r.Checksum.hash[:], computed)

	if o.XattrCache && !o.CacheReadonly {
		storeCachedHash(r.FilePath, r.FileInfo, r.Checksum)
	}
}

func equalReaders(f1, f2 io.Reader) bool {
	buf1 := make([]byte, 0xFFFFF)
	buf2 := make([]byte, 0xFFFFF)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		// io.Reader allows Read to return n > 0 alongside io.EOF. Neither
		// *os.File nor bufio.Reader (in the buffered path used here) emits
		// that pattern with the current reader chain, but compare the
		// trailing data anyway to stay correct under any future change.
		if err1 == io.EOF && err2 == io.EOF {
			return n1 == n2 && bytes.Equal(buf1[:n1], buf2[:n2])
		}

		if err1 != nil || err2 != nil {
			panic("unexpected read error")
		}

		if n1 != n2 {
			panic("unexpected read mismatch")
		}

		if !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false
		}
	}
}
