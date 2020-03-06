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

// updateDB is false if the file being checksummed has not yet been added to the DB
func (t *fileTable) Checksum(r *fileRecord, updateDB bool) error {
	if r.HasChecksum {
		return nil
	}

	if r.FailedChecksum != nil {
		return r.FailedChecksum
	}

	t.progress(r.RelPath, false)

	f, err := os.Open(r.FilePath)
	if err != nil {
		r.FailedChecksum = err
		t.totals.Errors.Add(r)
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return err
	}
	defer f.Close()

	if t.options.SkipHeader > 0 {
		if _, err = f.Seek(t.options.SkipHeader, io.SeekStart); err != nil {
			r.FailedChecksum = err
			t.totals.Errors.Add(r)
			fmt.Printf("%s: %s\n", r.RelPath, err)
			return err
		}
	}

	b, err := hwhChecksum(f)
	if err != nil {
		r.FailedChecksum = err
		t.totals.Errors.Add(r)
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return err
	}

	r.Checksum.size = r.Size()
	copy(r.Checksum.hash[:], b)
	r.HasChecksum = true

	if updateDB {
		// Update indexes with new checksum
		t.db.insert(r)
	}
	return nil
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
