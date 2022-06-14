package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileTable_Find(t *testing.T) {
	assert := require.New(t)

	scanner := newScanner()
	scanner.options.MatchMode = matchSize

	m, c1, err := scanner.table.findStat("foobar", &fakeStat{size: 1024}, "")
	assert.Equal(c1, m)
	assert.Equal("foobar", c1.FoldedName)
	assert.Equal(fileIsUnique, err)

	m, c2, err := scanner.table.findStat("fooBuzz", &fakeStat{size: 1024}, "")
	assert.Equal(c1, m)
	assert.Equal("foobuzz", c2.FoldedName)
	assert.Equal(matchSize, err)

	m, c3, err := scanner.table.findStat("oofbar", &fakeStat{size: 1024}, "")
	assert.Equal(c1, m)
	assert.Equal("oofbar", c3.FoldedName)
	assert.Equal(matchSize, err)
}

func TestFileTable_FindComparer(t *testing.T) {
	assert := require.New(t)

	scanner := newScanner()
	scanner.options.MatchMode = matchSize
	cmp, err := newComparer("1:1,-3:3", func(r *fileRecord) string { return r.FoldedName })
	assert.NoError(err)
	scanner.options.Comparers = []comparer{cmp}

	m, c1, err := scanner.table.findStat("foobar", &fakeStat{size: 1024}, "")
	assert.Equal(c1, m)
	assert.Equal("foobar", c1.FoldedName)
	assert.Equal(fileIsUnique, err)

	m, c2, err := scanner.table.findStat("fooBuzz", &fakeStat{size: 1024}, "")
	assert.Equal(c2, m)
	assert.Equal("foobuzz", c2.FoldedName)
	assert.Equal(fileIsUnique, err)

	m, c3, err := scanner.table.findStat("oofbar", &fakeStat{size: 1024}, "")
	assert.Equal(c1, m)
	assert.Equal("oofbar", c3.FoldedName)
	assert.Equal(matchSize, err)
}

type fakeStat struct {
	os.FileInfo

	isDir bool
	size  int64
}

func (s *fakeStat) IsDir() bool {
	return s.isDir
}

func (s *fakeStat) Size() int64 {
	return s.size
}
