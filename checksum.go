package main

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/minio/highwayhash"
)

const ChecksumBlockSize = 16

func hashKeyFromSize(size int64) []byte {
	key := make([]byte, 32)
	binary.BigEndian.PutUint64(key, uint64(size))
	return key
}

// updateDB is false if the file being checksummed has not yet been added to the DB
func (t *fileTable) Checksum(r *fileRecord, updateDB bool) error {
	if r.HasChecksum {
		return nil
	}

	if r.FailedChecksum != nil {
		return r.FailedChecksum
	}

	if t.options.XattrCache {
		cached, err := tryLoadCachedHash(r.FilePath, r.FileInfo, t.options.CacheMinTime)
		if err != nil {
			fmt.Printf("warning: %s: %s\n", r.RelPath, err)
			t.options.XattrCache = false
		} else if cached.size != 0 {
			r.Checksum = cached
			r.HasChecksum = true
			if updateDB {
				t.db.insert(r)
			}
			return nil
		}
	}

	t.progress(r.RelPath, false)

	f, err := t.options.OpenFile(r.FilePath)
	if err != nil {
		r.FailedChecksum = err
		t.totals.Errors.Add(r)
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return err
	}
	defer f.Close()

	b, err := hwhChecksum(f, r.Size()-t.options.SkipHeader-t.options.SkipFooter)
	if err != nil {
		r.FailedChecksum = err
		t.totals.Errors.Add(r)
		fmt.Printf("%s: %s\n", r.RelPath, err)
		return err
	}

	r.Checksum.size = r.Size()
	copy(r.Checksum.hash[:], b)
	r.HasChecksum = true

	if t.options.XattrCache && !t.options.CacheReadonly {
		if err := storeCachedHash(r.FilePath, r.FileInfo, r.Checksum); err != nil {
			fmt.Printf("warning: %s: %s\n", r.RelPath, err)
			t.options.XattrCache = false
		}
	}

	if updateDB {
		// Update indexes with new checksum
		t.db.insert(r)
	}
	return nil
}

func hwhChecksum(r io.Reader, size int64) ([]byte, error) {
	h, err := highwayhash.New128(hashKeyFromSize(size))
	if err != nil {
		return nil, err
	}

	n, err := io.Copy(h, r)
	if err != nil {
		return nil, err
	}

	if n != size {
		return nil, fmt.Errorf("expected %d bytes, got %d", size, n)
	}

	return h.Sum(nil), nil
}
