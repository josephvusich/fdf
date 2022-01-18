package main

import (
	"fmt"
	"strconv"
	"strings"
)

type comparer interface {
	AreEqual(a, b *fileRecord) bool
	HashFunc(*fileRecord) interface{}
}

type propertyComparer struct {
	getter func(*fileRecord) string
	ranges [][2]int
}

func (c *propertyComparer) AreEqual(a, b *fileRecord) bool {
	for _, r := range c.ranges {
		if getRange(c.getter(a), r[0], r[1]) != getRange(c.getter(b), r[0], r[1]) {
			return false
		}
	}
	return true
}

func (c *propertyComparer) HashFunc(r *fileRecord) interface{} {
	return c.getter(r)
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

func newComparer(input string, getter func(*fileRecord) string) (*propertyComparer, error) {
	c := &propertyComparer{
		getter: getter,
	}

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
