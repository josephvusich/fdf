package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
