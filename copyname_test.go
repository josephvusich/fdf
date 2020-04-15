package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsCopyName(t *testing.T) {
	assert := require.New(t)

	positive := [][]string{
		{
			"foo.bar",
			"Copy of foo.bar",
			"foo (1).bar",
		},
		{
			"foo",
			"Copy of foo",
			"foo-01",
		},
	}

	negative := []string{
		"foo.bar",
		"foo",
		".bar",
		"foo.",
		"foo.abc",
		"Copy of foo.xyz",
		"bar.foo",
		"f_o.bar",
	}

	for _, c := range positive {
		for i, x := range c {
			for j := i + 1; j < len(c); j++ {
				assert.True(isCopyName(x, c[j]), "%s and %s should be copy names", x, c[j])
				assert.True(isCopyName(c[j], x), "%s and %s should be copy names", c[j], x)
			}
		}
	}

	for i, x := range negative {
		for j := i + 1; j < len(negative); j++ {
			assert.False(isCopyName(x, negative[j]), "%s and %s should not be copy names", x, negative[j])
			assert.False(isCopyName(negative[j], x), "%s and %s should not be copy names", negative[j], x)
		}
	}
}
