package main

import (
	"fmt"
	"strconv"
	"strings"
)

type comparer interface {
	AreEqual(a, b *fileRecord) bool
}

type nameComparer struct {
	ranges [][2]int
}

func (c *nameComparer) AreEqual(a, b *fileRecord) bool {
	for _, r := range c.ranges {
		if getRange(a.FoldedName, r[0], r[1]) != getRange(b.FoldedName, r[0], r[1]) {
			return false
		}
	}
	return true
}

func getRange(s string, offset, length int) (partial string) {
	if length == 0 {
		return ""
	}
	defer func() {
		if r := recover(); r != nil {
			partial = ""
		}
	}()
	if offset < 0 {
		offset += len(s)
		for offset < 0 {
			offset++
			if length > 0 {
				length--
			}
		}
	}
	if length < 0 {
		length += 1 + len(s)
	} else {
		length += offset
	}
	partial = s[offset:length]
	return partial
}

func newNameComparer(input string) (*nameComparer, error) {
	c := &nameComparer{}

	for _, r := range strings.Split(input, ",") {
		values := strings.Split(r, ":")
		if len(values) != 2 {
			return nil, fmt.Errorf("invalid range spec: %s", r)
		}

		var nums [2]int
		for i, v := range values {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid range spec `%s`: %w", r, err)
			}
			nums[i] = n
		}

		c.ranges = append(c.ranges, nums)
	}
	return c, nil
}
