package main

import (
	"fmt"
	"regexp"
	"strings"
)

var copyNamePattern = regexp.MustCompile(`^(?i)\s*(?:Copy (?:\d+ )?of )?(.*?)(?:[-_ ][()\d]+?(?:x\d+)?)?\s*(\.[^\.]*?)?\s*$`)

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

func isNameSuffix(nameA, nameB string) bool {
	nameA = strings.ToLower(nameA)
	nameB = strings.ToLower(nameB)

	if len(nameA) == len(nameB) {
		return nameA == nameB
	}

	if len(nameA) < len(nameB) {
		return strings.HasSuffix(nameB, nameA)
	}

	return strings.HasSuffix(nameA, nameB)
}
