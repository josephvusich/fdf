package main

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/minio/highwayhash"
	"github.com/stretchr/testify/require"
)

func BenchmarkChecksum_hwhContinuous(b *testing.B) {
	assert := require.New(b)
	buf := make([]byte, 2048*2048*100)
	rand.Read(buf)
	h, err := highwayhash.New128(hashKey)
	assert.NoError(err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Write(buf)
	}
	h.Sum(nil)
}

func BenchmarkChecksum_hwh(b *testing.B) {
	buf := make([]byte, 2048*2048*100)
	rand.Read(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hwhChecksum(bytes.NewReader(buf))
	}
}
