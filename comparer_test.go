package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetRange(t *testing.T) {
	assert := assert.New(t)

	subject := "fooBarFizzBuzz"
	cases := map[string]string{
		"fooBarFizzBuzz": "0:-1",
		"fooBarFizzBuz":  "0:-2",
		"FizzBuzz":       "6:-1",
		"Bar":            "3:3",
		"Buzz":           "-4:4",
	}

	for expect, inputs := range cases {
		cmp, err := newNameComparer(inputs)
		assert.NoError(err)
		assert.Len(cmp.ranges, 1)
		assert.Equal(expect, getRange(subject, cmp.ranges[0][0], cmp.ranges[0][1]), "failed %d:%d", inputs[0], inputs[1])
	}
}
