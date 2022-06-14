package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtectArgs(t *testing.T) {
	assert := require.New(t)

	args := []string{`fdf`, `-r`, `--protect`, `./a/**/*`, `--unprotect`, `a/**/bar`, `a`, `./b`}
	var o options
	dirs := o.ParseArgs(args)

	assert.True(o.Recursive)

	assert.Len(dirs, 2)
	assert.Equal("a", dirs[0])
	assert.Equal("./b", dirs[1])

	cases := map[string]bool{
		"./a/foo":     true,
		"./a/foo/bar": false,
		"./b/foo":     false,
		"./b/foo/bar": false,
	}

	for in, out := range cases {
		abs, err := filepath.Abs(in)
		assert.NoError(err)
		assert.Equal(out, o.Protect.Includes(abs), "expected Includes=%t for '%s'", out, in)
	}
}

func TestOptions_ParseArgs(t *testing.T) {
	assert := require.New(t)

	mockFileRecord := &fileRecord{
		FilePath:     "Path/To/File",
		FoldedName:   "FoldedName",
		FoldedParent: "FoldedParent",
	}

	tests := []struct {
		spec      string
		verb      verb
		expect    matchFlag
		comparers []string
	}{
		{"content", VerbMakeLinks, matchContent | matchSize, nil},
		{"size", VerbMakeLinks, matchSize, nil},
		{"name", VerbMakeLinks, matchName, nil},

		{"content+name", VerbMakeLinks, matchContent | matchName, nil},
		{"size+name", VerbMakeLinks, matchSize | matchName, nil},
		{"name+content", VerbMakeLinks, matchName | matchContent, nil},
		{"name[0:3]+content", VerbMakeLinks, matchContent, []string{"FoldedName"}},
		{"parent[0:3]+content", VerbMakeLinks, matchContent, []string{"FoldedParent"}},
		{"content+path", VerbMakeLinks, matchContent | matchParent | matchPathSuffix, []string{filepath.Join("Path", "To")}},

		{"content+name", VerbSplitLinks, matchContent | matchName | matchHardlink, nil},
		{"size+name", VerbSplitLinks, matchSize | matchName | matchHardlink, nil},
		{"name+content", VerbSplitLinks, matchName | matchContent | matchHardlink, nil},
	}

	for _, t := range tests {
		var o options
		err := o.parseMatchSpec(t.spec, t.verb)
		assert.NoError(err)
		assert.Equal(t.expect, o.MatchMode, t.spec)
		assert.Len(o.Comparers, len(t.comparers))
		for i, c := range o.Comparers {
			assert.Equal(t.comparers[i], c.HashFunc(mockFileRecord))
		}
	}
}
