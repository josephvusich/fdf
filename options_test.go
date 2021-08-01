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

	tests := []struct {
		spec      string
		verb      verb
		expect    matchFlag
		comparers int
	}{
		{"content", VerbMakeLinks, matchContent | matchSize, 0},
		{"size", VerbMakeLinks, matchSize, 0},
		{"name", VerbMakeLinks, matchName, 0},

		{"content+name", VerbMakeLinks, matchContent | matchName, 0},
		{"size+name", VerbMakeLinks, matchSize | matchName, 0},
		{"name+content", VerbMakeLinks, matchName | matchContent, 0},
		{"name[0:3]+content", VerbMakeLinks, matchContent, 1},

		{"content+name", VerbSplitLinks, matchContent | matchName | matchHardlink, 0},
		{"size+name", VerbSplitLinks, matchSize | matchName | matchHardlink, 0},
		{"name+content", VerbSplitLinks, matchName | matchContent | matchHardlink, 0},
	}

	for _, t := range tests {
		var o options
		err := o.parseMatchSpec(t.spec, t.verb)
		assert.NoError(err)
		assert.Equal(t.expect, o.MatchMode)
	}
}
