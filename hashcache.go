package main

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"time"
)

const (
	xattrVersion  = 1
	xattrDataSize = 42 // bytes before CRC32
	xattrSize     = xattrDataSize + 4
)

func tryLoadCachedHash(path string, info os.FileInfo, minCacheTime int64) (cs checksum, err error) {
	data, err := getXattr(path, xattrKey)
	if err != nil {
		if xattrNotFound(err) {
			return cs, nil
		}
		return cs, err
	}

	if len(data) != xattrSize {
		return cs, fmt.Errorf("xattr cache: unexpected size %d (expected %d)", len(data), xattrSize)
	}

	stored := binary.BigEndian.Uint32(data[xattrDataSize:])
	computed := crc32.ChecksumIEEE(data[:xattrDataSize])
	if stored != computed {
		return cs, fmt.Errorf("xattr cache: CRC32 mismatch (stored %08x, computed %08x)", stored, computed)
	}

	version := binary.BigEndian.Uint16(data[0:2])
	if version != xattrVersion {
		return cs, fmt.Errorf("xattr cache: version mismatch (stored %d, expected %d)", version, xattrVersion)
	}

	cacheTimestamp := int64(binary.BigEndian.Uint64(data[2:10]))
	if cacheTimestamp < minCacheTime {
		return cs, nil
	}

	cachedSize := int64(binary.BigEndian.Uint64(data[10:18]))
	cachedMtime := int64(binary.BigEndian.Uint64(data[18:26]))

	if cachedSize != info.Size() || cachedMtime != info.ModTime().UnixNano() {
		return cs, nil
	}

	cs.size = cachedSize
	copy(cs.hash[:], data[26:42])
	return cs, nil
}

func storeCachedHash(path string, info os.FileInfo, cs checksum) error {
	var data [xattrSize]byte

	binary.BigEndian.PutUint16(data[0:2], xattrVersion)
	binary.BigEndian.PutUint64(data[2:10], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(data[10:18], uint64(cs.size))
	binary.BigEndian.PutUint64(data[18:26], uint64(info.ModTime().UnixNano()))
	copy(data[26:42], cs.hash[:])

	binary.BigEndian.PutUint32(data[xattrDataSize:], crc32.ChecksumIEEE(data[:xattrDataSize]))

	return setXattr(path, xattrKey, data[:])
}
