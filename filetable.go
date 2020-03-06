package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type fileTable struct {
	bySize     map[int64][]*fileRecord
	byChecksum map[checksum][]*fileRecord
	wd         string

	// -1 == quiet
	termWidth int

	options *options
}

func (t *fileTable) MatchesBySize(r *fileRecord) []*fileRecord {
	return t.bySize[r.Size()]
}

// First pass avoids checksumming files if a matching checksum is already found
// If false, one new file (if any) will be checksummed and returned. Repeat until empty list
// This allows lazy checksumming for the first file of a given size, or if all files of a
// given size are hardlinks already
func (t *fileTable) MatchesByChecksum(r *fileRecord, firstPass bool) []*fileRecord {
	if !t.Checksum(r) {
		return nil
	}

	if firstPass {
		return t.byChecksum[r.Checksum]
	}

	sameSize := t.MatchesBySize(r)
	for _, other := range sameSize {
		if !other.HasChecksum {
			if !t.Checksum(other) {
				continue
			}

			t.byChecksum[other.Checksum] = append(t.byChecksum[other.Checksum], other)

			if r.Checksum == other.Checksum {
				return []*fileRecord{other}
			}
		}
	}

	return nil
}

func newFileTable(o *options) *fileTable {
	return &fileTable{
		bySize:     map[int64][]*fileRecord{},
		byChecksum: map[checksum][]*fileRecord{},
		options:    o,
	}
}

type checksum struct {
	size int64
	hash [ChecksumBlockSize]byte
}

type fileRecord struct {
	FilePath string
	RelPath  string
	os.FileInfo

	HasChecksum    bool
	FailedChecksum bool
	Checksum       checksum
}

func (r *fileRecord) String() string {
	return fmt.Sprintf("%s: %t %X", r.FilePath, r.HasChecksum, r.Checksum)
}

func newFileRecord(path string, info os.FileInfo, relPath string) *fileRecord {
	return &fileRecord{
		FilePath: path,
		RelPath:  relPath,
		FileInfo: info,
	}
}

func (t *fileTable) Rel(absPath string) (rel string) {
	rel, err := filepath.Rel(t.wd, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

const truncFill = " ... "

func (t *fileTable) progress(s string, makeRelPath bool) {
	if t.termWidth <= 0 {
		return
	}

	if makeRelPath {
		s = t.Rel(s)
	}

	if t.termWidth > len(truncFill)+2 && len(s) >= t.termWidth {
		chunkSize := (t.termWidth - len(truncFill) - 1) >> 1
		s = s[:chunkSize] + truncFill + s[len(s)-chunkSize:]
	}

	fmt.Printf("\033[2K%s\r", s)
}

func (t *fileTable) find(f string) (match *fileRecord, current *fileRecord, areLinked bool, err error) {
	st, err := os.Stat(f)
	if err != nil {
		return nil, nil, false, err
	}

	if st.IsDir() {
		return nil, nil, false, noErrNotDupe
	}

	if st.Size() < t.options.MinSize() {
		return nil, nil, false, noErrNotDupe
	}

	current = newFileRecord(f, st, t.Rel(f))

	if sameSize := t.MatchesBySize(current); len(sameSize) != 0 {

		for _, other := range sameSize {
			if os.SameFile(current.FileInfo, other.FileInfo) {
				return other, current, true, nil
			}
		}

		for firstPass := true; ; firstPass = false {
			sameChecksum := t.MatchesByChecksum(current, firstPass)

			if len(sameChecksum) == 0 && !firstPass {
				break
			}

			for _, other := range sameChecksum {
				if equalFiles(current, other, t.options.SkipHeader) {
					return other, current, false, nil
				}
			}
		}
	}

	t.insert(current)
	return current, current, false, nil
}

// Does not do duplicate checking
func (t *fileTable) insert(r *fileRecord) {
	t.bySize[r.Size()] = append(t.bySize[r.Size()], r)
	if r.HasChecksum {
		t.byChecksum[r.Checksum] = append(t.byChecksum[r.Checksum], r)
	}
}
