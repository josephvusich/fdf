package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRange(t *testing.T) {
	assert := assert.New(t)

	subject := "fooBarFizzBuzz"
	cases := map[string]string{
		":":    "fooBarFizzBuzz",
		"0:-1": "fooBarFizzBuzz",
		"0:-2": "fooBarFizzBuz",
		"6:-1": "FizzBuzz",
		"3:3":  "Bar",
		"-4:4": "Buzz",
	}

	for inputs, expect := range cases {
		cmp, err := newComparer(inputs, func(r *fileRecord) string { return r.FoldedName })
		assert.NoError(err)
		assert.Len(cmp.ranges, 1)
		assert.Equal(expect, getRange(subject, cmp.ranges[0][0], cmp.ranges[0][1]), "failed [%s]", inputs)
	}
}
