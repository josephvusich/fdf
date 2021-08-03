// +build !no_clone

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanner_Clone(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.clone = true
		scanner.options.Recursive = true
		scanner.options.MatchMode = matchContent

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(7), scanner.totals.Unique.count)
		assert.Equal(uint64(33), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(15), scanner.totals.Cloned.count)
		assert.Equal(uint64(40), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(40), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)
	})
}
