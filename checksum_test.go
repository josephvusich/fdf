package main

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/minio/highwayhash"
	"github.com/stretchr/testify/require"
)

func TestHwhChecksum_Deterministic(t *testing.T) {
	assert := require.New(t)
	data := []byte("deterministic input data")

	h1, err := hwhChecksum(bytes.NewReader(data))
	assert.NoError(err)
	h2, err := hwhChecksum(bytes.NewReader(data))
	assert.NoError(err)

	assert.Equal(h1, h2)
}

func TestHwhChecksum_DifferentInputs(t *testing.T) {
	assert := require.New(t)

	h1, err := hwhChecksum(bytes.NewReader([]byte("input A")))
	assert.NoError(err)
	h2, err := hwhChecksum(bytes.NewReader([]byte("input B")))
	assert.NoError(err)

	assert.NotEqual(h1, h2)
}

func TestHwhChecksum_EmptyInput(t *testing.T) {
	assert := require.New(t)

	h, err := hwhChecksum(bytes.NewReader(nil))
	assert.NoError(err)
	assert.Len(h, ChecksumBlockSize)
}

func TestHwhChecksum_ResultSize(t *testing.T) {
	assert := require.New(t)

	h, err := hwhChecksum(bytes.NewReader([]byte("some data")))
	assert.NoError(err)
	assert.Len(h, ChecksumBlockSize)
}

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
