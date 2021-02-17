package matchers

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobSet(t *testing.T) {
	assert := require.New(t)

	gs := GlobSet{DefaultInclude: false}

	g, err := NewGlob("./a/**/*", true)
	assert.NoError(err)
	gs.Add(g)
	g, err = NewGlob("./a/**/bar", false)
	assert.NoError(err)
	gs.Add(g)

	cases := map[string]bool{
		"./a/foo":     true,
		"./a/foo/bar": false,
		"./b/foo":     false,
		"./b/foo/bar": false,
	}

	for in, out := range cases {
		abs, err := filepath.Abs(in)
		assert.NoError(err)
		assert.Equal(out, gs.Includes(abs), "expected Includes=%t for '%s'", out, in)
	}
}
