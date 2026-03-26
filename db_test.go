package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDB_InsertAndQuery(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r := &fileRecord{
		FoldedName:   "test.txt",
		FoldedParent: "docs",
		PathSuffix:   "sub/docs",
		FileInfo:     &fakeStat{size: 1024},
	}
	d.insert(r)

	// Query by name
	q := &query{Name: "test.txt"}
	rs := d.query(q)
	assert.Contains(rs, r)

	// Query by size
	q = &query{Size: 1024}
	rs = d.query(q)
	assert.Contains(rs, r)

	// Query by name + size
	q = &query{Name: "test.txt", Size: 1024}
	rs = d.query(q)
	assert.Contains(rs, r)

	// Query miss — wrong name
	q = &query{Name: "other.txt"}
	rs = d.query(q)
	assert.NotContains(rs, r)

	// Query miss — wrong size
	q = &query{Size: 2048}
	rs = d.query(q)
	assert.NotContains(rs, r)
}

func TestDB_Remove(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r := &fileRecord{
		FoldedName: "remove-me.txt",
		FileInfo:   &fakeStat{size: 512},
	}
	d.insert(r)

	// Verify present
	q := &query{Name: "remove-me.txt"}
	assert.Contains(d.query(q), r)

	// Remove and verify absent
	d.remove(r)
	assert.NotContains(d.query(q), r)
}

func TestDB_MultipleRecords(t *testing.T) {
	assert := require.New(t)

	d := newDB()
	r1 := &fileRecord{
		FoldedName: "alpha.txt",
		FileInfo:   &fakeStat{size: 100},
	}
	r2 := &fileRecord{
		FoldedName: "beta.txt",
		FileInfo:   &fakeStat{size: 100},
	}
	d.insert(r1)
	d.insert(r2)

	// Query by size returns both
	q := &query{Size: 100}
	rs := d.query(q)
	assert.Contains(rs, r1)
	assert.Contains(rs, r2)

	// Query by name returns only matching
	q = &query{Name: "alpha.txt"}
	rs = d.query(q)
	assert.Contains(rs, r1)
	assert.NotContains(rs, r2)
}

func TestDB_QueryGenerators(t *testing.T) {
	assert := require.New(t)
	// The init() function pre-generates query key combinations from 5 index dimensions,
	// excluding invalid combos (checksum without size, pathSuffix without parent).
	assert.Len(queryGenerators, 17)
}
