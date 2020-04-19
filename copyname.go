package main

import (
	"fmt"
	"regexp"
	"strings"
)

var copyNamePattern = regexp.MustCompile(`^(?i)(?:Copy (?:\d+ )?of )?((?U).*)(?:[-_ ()\d]+)?(\.[^\.]*)?$`)

// Note that isCopyName must be transitive or links can be created and broken in a single traversal
func isCopyName(nameA, nameB string) bool {
	names := []string{
		nameA,
		nameB,
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