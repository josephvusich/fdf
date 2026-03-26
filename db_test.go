package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDB_InsertAndQuery(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r := &fileRecord{
		FilePath:     "/docs/test.txt",
		FoldedName:   "test.txt",
		FoldedParent: "docs",
		PathSuffix:   "sub/docs",
		FileInfo:     &fakeStat{size: 1024},
	}
	d.insert(r)

	// Query by name
	q := &query{fields: queryName, Name: "test.txt"}
	rs := d.query(q)
	assert.Contains(rs, r)

	// Query by size
	q = &query{fields: querySize, Size: 1024}
	rs = d.query(q)
	assert.Contains(rs, r)

	// Query by name + size
	q = &query{fields: queryName | querySize, Name: "test.txt", Size: 1024}
	rs = d.query(q)
	assert.Contains(rs, r)

	// Query miss — wrong name
	q = &query{fields: queryName, Name: "other.txt"}
	rs = d.query(q)
	assert.NotContains(rs, r)

	// Query miss — wrong size
	q = &query{fields: querySize, Size: 2048}
	rs = d.query(q)
	assert.NotContains(rs, r)
}

func TestDB_Remove(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r := &fileRecord{
		FilePath:   "/tmp/remove-me.txt",
		FoldedName: "remove-me.txt",
		FileInfo:   &fakeStat{size: 512},
	}
	d.insert(r)

	// Verify present
	q := &query{fields: queryName, Name: "remove-me.txt"}
	assert.Contains(d.query(q), r)

	// Remove and verify absent
	d.remove(r)
	assert.NotContains(d.query(q), r)
}

func TestDB_MultipleRecords(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r1 := &fileRecord{
		FilePath:   "/tmp/alpha.txt",
		FoldedName: "alpha.txt",
		FileInfo:   &fakeStat{size: 100},
	}
	r2 := &fileRecord{
		FilePath:   "/tmp/beta.txt",
		FoldedName: "beta.txt",
		FileInfo:   &fakeStat{size: 100},
	}
	d.insert(r1)
	d.insert(r2)

	// Query by size returns both
	q := &query{fields: querySize, Size: 100}
	rs := d.query(q)
	assert.Contains(rs, r1)
	assert.Contains(rs, r2)

	// Query by name returns only matching
	q = &query{fields: queryName, Name: "alpha.txt"}
	rs = d.query(q)
	assert.Contains(rs, r1)
	assert.NotContains(rs, r2)
}

func TestDB_QuerySizeZero(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r1 := &fileRecord{
		FilePath:   "/tmp/empty1.txt",
		FoldedName: "empty1.txt",
		FileInfo:   &fakeStat{size: 0},
	}
	r2 := &fileRecord{
		FilePath:   "/tmp/empty2.txt",
		FoldedName: "empty2.txt",
		FileInfo:   &fakeStat{size: 0},
	}
	r3 := &fileRecord{
		FilePath:   "/tmp/notempty.txt",
		FoldedName: "notempty.txt",
		FileInfo:   &fakeStat{size: 10},
	}
	d.insert(r1)
	d.insert(r2)
	d.insert(r3)

	// Query by size=0 finds both empty files
	q := &query{fields: querySize, Size: 0}
	rs := d.query(q)
	assert.Contains(rs, r1)
	assert.Contains(rs, r2)
	assert.NotContains(rs, r3)
}

func TestDB_ChecksumReinsert(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r := &fileRecord{
		FilePath:   "/tmp/checksummed.txt",
		FoldedName: "checksummed.txt",
		FileInfo:   &fakeStat{size: 256},
	}
	d.insert(r)

	// Not findable by checksum yet
	cs := checksum{size: 256, hash: [ChecksumBlockSize]byte{1, 2, 3}}
	q := &query{fields: queryChecksum, Checksum: cs}
	assert.NotContains(d.query(q), r)

	// Add checksum and re-insert (upsert)
	r.HasChecksum = true
	r.Checksum = cs
	d.insert(r)

	// Now findable by checksum
	assert.Contains(d.query(q), r)
}

func TestDB_AllRecords(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r1 := &fileRecord{
		FilePath:   "/tmp/one.txt",
		FoldedName: "one.txt",
		FileInfo:   &fakeStat{size: 10},
	}
	r2 := &fileRecord{
		FilePath:   "/tmp/two.txt",
		FoldedName: "two.txt",
		FileInfo:   &fakeStat{size: 20},
	}
	d.insert(r1)
	d.insert(r2)

	all := d.allRecords()
	assert.Len(all, 2)
	assert.Contains(all, r1)
	assert.Contains(all, r2)
}
