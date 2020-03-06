package main

import (
	"bytes"
	"io"
	"os"
)

func equalFiles(r1, r2 *fileRecord, skipHeader int64) bool {
	f1, err := os.Open(r1.FilePath)
	if err != nil {
		return false
	}
	defer f1.Close()

	f2, err := os.Open(r2.FilePath)
	if err != nil {
		return false
	}
	defer f2.Close()

	if skipHeader > 0 {
		for _, f := range []*os.File{f1, f2} {
			if _, err = f.Seek(skipHeader, io.SeekStart); err != nil {
				return false
			}
		}
	}

	return equalReaders(f1, f2)
}

func equalReaders(f1, f2 io.Reader) bool {
	buf1 := make([]byte, 0xFFFFF)
	buf2 := make([]byte, 0xFFFFF)

	for {
		n1, err1 := f1.Read(buf1)
		n2, err2 := f2.Read(buf2)

		if err1 == io.EOF && err2 == io.EOF {
			return true
		}

		if err1 != nil || err2 != nil {
			panic("unexpected read error")
		}

		if n1 != n2 {
			panic("unexpected read mismatch")
		}

		if !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false
		}
	}
}
