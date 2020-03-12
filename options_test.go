package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptions_ParseArgs(t *testing.T) {
	assert := require.New(t)

	tests := []struct {
		spec   string
		verb   verb
		expect matchFlag
	}{
		{"content", VerbMakeLinks, matchContent | matchSize},
		{"size", VerbMakeLinks, matchSize},
		{"name", VerbMakeLinks, matchName},

		{"content+name", VerbMakeLinks, matchContent | matchName},
		{"size+name", VerbMakeLinks, matchSize | matchName},
		{"name+content", VerbMakeLinks, matchName | matchContent},

		{"content+name", VerbSplitLinks, matchContent | matchName | matchHardlink},
		{"size+name", VerbSplitLinks, matchSize | matchName | matchHardlink},
		{"name+content", VerbSplitLinks, matchName | matchContent | matchHardlink},
	}

	for _, t := range tests {
		mf, err := parseMatchSpec(t.spec, t.verb)
		assert.NoError(err)
		assert.Equal(t.expect, mf)
	}
}
