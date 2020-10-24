package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var copyNamePattern = regexp.MustCompile(`^(?i)\s*(?:Copy (?:\d+ )?of )?((?U).*)(?:(?U)[-_ ()\d]+)?\s*((?U)\.[^\.]*)?\s*$`)

// Note that isCopyName must be transitive or links can be created and broken in a single traversal
func isCopyName(nameA, nameB string) bool {
	if len(nameA) == 0 || len(nameB) == 0 {
		return false
	}

	names := []string{
		nameA,
		nameB,
	}

	parts := make([][]string, 2)
	shortestBase := -1
	for i, name := range names {
		lastExt := ""
		for partExt := filepath.Ext(name); partExt != ""; partExt = filepath.Ext(name) {
			if lastExt == "" {
				lastExt = partExt
			}
			name = strings.TrimSuffix(name, partExt)
		}

		parts[i] = []string{
			name,
			lastExt,
		}

		if l := len(name); shortestBase == -1 || l < shortestBase {
			shortestBase = l
		}
	}

	if shortestBase != 0 && parts[0][0][0:shortestBase] == parts[1][0][0:shortestBase] && parts[0][1] == parts[1][1] {
		return true
	}

	for i := range names {
		m := copyNamePattern.FindStringSubmatch(names[i])
		if m == nil {
			return false
		}
		names[i] = strings.ToLower(fmt.Sprintf("%s%s", m[1], m[2]))
	}

	return names[0] == names[1]
}
