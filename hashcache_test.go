package main

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func xattrSupported(t *testing.T, path string) {
	t.Helper()
	err := setXattr(path, xattrKey, []byte("test"))
	if err != nil {
		t.Skipf("xattrs not supported: %v", err)
	}
}

func TestHashCache_RoundTrip(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	cs := checksum{
		size: info.Size(),
		hash: [ChecksumBlockSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	assert.NoError(storeCachedHash(f, info, cs))

	loaded, err := tryLoadCachedHash(f, info)
	assert.NoError(err)
	assert.Equal(cs, loaded)
}

func TestHashCache_Miss_NoXattr(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello"), 0644))

	info, err := os.Stat(f)
	assert.NoError(err)

	loaded, err := tryLoadCachedHash(f, info)
	assert.Equal(checksum{}, loaded)
	// Error is platform-dependent (ENODATA on Linux, etc.) but should be non-nil or nil depending on
	// whether the filesystem reports "no such attribute" as an error. Either way, checksum should be zero.
}

func TestHashCache_Miss_SizeChanged(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	cs := checksum{
		size: info.Size(),
		hash: [ChecksumBlockSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	assert.NoError(storeCachedHash(f, info, cs))

	// Modify the file to change its size
	assert.NoError(os.WriteFile(f, []byte("hello world, bigger now"), 0644))

	info2, err := os.Stat(f)
	assert.NoError(err)

	loaded, err := tryLoadCachedHash(f, info2)
	assert.NoError(err)
	assert.Equal(checksum{}, loaded)
}

func TestHashCache_Miss_MtimeChanged(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	cs := checksum{
		size: info.Size(),
		hash: [ChecksumBlockSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	assert.NoError(storeCachedHash(f, info, cs))

	// Touch the file to change mtime without changing content/size
	newTime := info.ModTime().Add(time.Hour)
	assert.NoError(os.Chtimes(f, newTime, newTime))

	info2, err := os.Stat(f)
	assert.NoError(err)

	loaded, err := tryLoadCachedHash(f, info2)
	assert.NoError(err)
	assert.Equal(checksum{}, loaded)
}

func TestHashCache_Error_CRCMismatch(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	cs := checksum{
		size: info.Size(),
		hash: [ChecksumBlockSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	assert.NoError(storeCachedHash(f, info, cs))

	// Corrupt the xattr data
	data, err := getXattr(f, xattrKey)
	assert.NoError(err)
	data[26] ^= 0xFF // flip a hash byte
	assert.NoError(setXattr(f, xattrKey, data))

	loaded, err := tryLoadCachedHash(f, info)
	assert.Error(err)
	assert.Contains(err.Error(), "CRC32 mismatch")
	assert.Equal(checksum{}, loaded)
}

func TestHashCache_Error_VersionMismatch(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	cs := checksum{
		size: info.Size(),
		hash: [ChecksumBlockSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	assert.NoError(storeCachedHash(f, info, cs))

	// Change the version field and recompute CRC
	data, err := getXattr(f, xattrKey)
	assert.NoError(err)
	binary.BigEndian.PutUint16(data[0:2], 99)
	binary.BigEndian.PutUint32(data[xattrDataSize:], crc32.ChecksumIEEE(data[:xattrDataSize]))
	assert.NoError(setXattr(f, xattrKey, data))

	loaded, err := tryLoadCachedHash(f, info)
	assert.Error(err)
	assert.Contains(err.Error(), "version mismatch")
	assert.Equal(checksum{}, loaded)
}

func TestHashCache_Error_WrongLength(t *testing.T) {
	assert := require.New(t)

	dir := t.TempDir()
	f := filepath.Join(dir, "testfile")
	assert.NoError(os.WriteFile(f, []byte("hello world"), 0644))

	xattrSupported(t, f)

	info, err := os.Stat(f)
	assert.NoError(err)

	// Write truncated data
	assert.NoError(setXattr(f, xattrKey, []byte("too short")))

	loaded, err := tryLoadCachedHash(f, info)
	assert.Error(err)
	assert.Contains(err.Error(), "unexpected size")
	assert.Equal(checksum{}, loaded)
}
