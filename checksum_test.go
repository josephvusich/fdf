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
	size := int64(len(data))

	h1, err := hwhChecksum(bytes.NewReader(data), size)
	assert.NoError(err)
	h2, err := hwhChecksum(bytes.NewReader(data), size)
	assert.NoError(err)

	assert.Equal(h1, h2)
}

func TestHwhChecksum_DifferentInputs(t *testing.T) {
	assert := require.New(t)

	h1, err := hwhChecksum(bytes.NewReader([]byte("input A")), 7)
	assert.NoError(err)
	h2, err := hwhChecksum(bytes.NewReader([]byte("input B")), 7)
	assert.NoError(err)

	assert.NotEqual(h1, h2)
}

func TestHwhChecksum_EmptyInput(t *testing.T) {
	assert := require.New(t)

	h, err := hwhChecksum(bytes.NewReader(nil), 0)
	assert.NoError(err)
	assert.Len(h, ChecksumBlockSize)
}

func TestHwhChecksum_ResultSize(t *testing.T) {
	assert := require.New(t)

	h, err := hwhChecksum(bytes.NewReader([]byte("some data")), 9)
	assert.NoError(err)
	assert.Len(h, ChecksumBlockSize)
}

func BenchmarkChecksum_hwhContinuous(b *testing.B) {
	assert := require.New(b)
	buf := make([]byte, 2048*2048*100)
	rand.Read(buf)
	h, err := highwayhash.New128(hashKeyFromSize(int64(len(buf))))
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
	size := int64(len(buf))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hwhChecksum(bytes.NewReader(buf), size)
	}
}
