package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"github.com/minio/highwayhash"
)

const ChecksumBlockSize = 16

// 32 bytes of random hash key
var hashKey []byte

func init() {
	hashKey = make([]byte, 32)
	rand.Read(hashKey)

	h, err := highwayhash.New128(hashKey)
	if err != nil {
		panic(err)
	}

	if h.Size() != ChecksumBlockSize {
		panic("unexpected block size")
	}
}

func (t *fileTable) Checksum(r *fileRecord) bool {
	if r.HasChecksum {
		return true
	}

	if r.FailedChecksum {
		return false
	}

	t.progress(r.RelPath, false)

	f, err := os.Open(r.FilePath)
	if err != nil {
		r.FailedChecksum = true
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return false
	}
	defer f.Close()

	b, err := hwhChecksum(f)
	if err != nil {
		r.FailedChecksum = true
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return false
	}

	r.Checksum.size = r.Size()
	copy(r.Checksum.hash[:], b)
	r.HasChecksum = true
	return true
}

func hwhChecksum(r io.Reader) ([]byte, error) {
	h, err := highwayhash.New128(hashKey)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(h, r)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
