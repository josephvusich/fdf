package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type recordSet map[*fileRecord]struct{}

type fileTable struct {
	wd string

	db *db

	// 0 == quiet, -1 == error/not a terminal
	termWidth int

	options *options
	totals  *totals
}

func newFileTable(o *options, t *totals) *fileTable {
	return &fileTable{
		db:      newDB(),
		options: o,
		totals:  t,
	}
}

type checksum struct {
	size int64
	hash [ChecksumBlockSize]byte
}

type fileRecord struct {
	FilePath   string
	RelPath    string
	FoldedName string
	os.FileInfo

	HasChecksum    bool
	FailedChecksum error
	Checksum       checksum

	preserve       *bool
	PreserveReason string
}

func foldName(filePath string) string {
	return strings.ToLower(filepath.Base(filePath))
}

// Note that `p` is ignored if there is already a cached result
func (r *fileRecord) Preserve(p preservePatterns) bool {
	if r.preserve == nil {
		pattern, ok := p.Match(r.RelPath)
		r.preserve, r.PreserveReason = &ok, pattern
	}
	return *r.preserve
}

func (r *fileRecord) String() string {
	return fmt.Sprintf("%s: %t %X", r.FilePath, r.HasChecksum, r.Checksum)
}

func newFileRecord(path string, info os.FileInfo, relPath string) *fileRecord {
	return &fileRecord{
		FilePath:   path,
		RelPath:    relPath,
		FoldedName: foldName(path),
		FileInfo:   info,
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

type matchFlag uint

const (
	matchNothing  matchFlag = 0b0000000000000000                // default value, usually replaced with matchContent
	matchName     matchFlag = 0b0000000000000001                // case-insensitive
	matchSize     matchFlag = 0b0000000000000010                // implied by matchContent and matchHardlink
	matchContent            = 0b0000000000000100 | matchSize    // implied by matchHardlink
	matchHardlink           = 0b0000000000001000 | matchContent // used by --copy and for categorization
	matchCopyName           = 0b0000000000010000                // one filename must contain the other, e.g., "foo" and "foo copy (1)"
	fileIsIgnored matchFlag = 0b1000000000000000                // status returned for directories
	fileIsSkipped matchFlag = 0b0100000000000000                // file was excluded e.g., due to size requirements
	fileIsUnique  matchFlag = 0b0010000000000000                // no match found
)

func (m matchFlag) has(flag matchFlag) bool {
	return m&flag == flag
}

func (m matchFlag) Error() string {
	return fmt.Sprintf("MatchType<0b%b>", m)
}

func (t *fileTable) find(f string) (match *fileRecord, current *fileRecord, err error) {
	st, err := os.Stat(f)
	if err != nil {
		return nil, nil, err
	}

	if st.IsDir() {
		return nil, nil, fileIsIgnored
	}

	if st.Size() < t.options.MinSize() {
		return nil, nil, fileIsSkipped
	}

	current = newFileRecord(f, st, t.Rel(f))

	q := &query{}
	if t.options.MatchMode.has(matchName) {
		current.byName(q)
	}
	if t.options.MatchMode.has(matchSize) {
		current.bySize(q)
	}
	// Ignore checksums for now, as hardlinks can match content without the overhead of comparison

	// Query for any known files that match all desired fields (except content/checksum)
	candidates := t.db.query(q)

	// If copyname mode is active, filter down the candidate list
	if t.options.MatchMode.has(matchCopyName) {
		filtered := recordSet{}
		for other := range candidates {
			if isCopyName(current.FoldedName, other.FoldedName) {
				filtered[other] = struct{}{}
			}
		}
		candidates = filtered
	}

	// If there is a matching hardlink, skip further checking
	// Name is the only non-hardlink-included field
	for other := range candidates {
		if os.SameFile(current.FileInfo, other.FileInfo) {
			return other, current, t.options.MatchMode | matchHardlink
		}
	}

	// --copy is not interested in non-hardlinks
	if t.options.MatchMode.has(matchHardlink) {
		t.db.insert(current)
		return current, current, fileIsUnique
	}

	// If matching content is not important, return random valid match, if any
	if !t.options.MatchMode.has(matchContent) {
		for other := range candidates {
			return other, current, t.options.MatchMode
		}
	}

	// If we get here, we're matching content and no hardlink was found
	// First we check any existing checksum matches for full equality
	if current.HasChecksum {
		current.byChecksum(q)
		existingChecksums := t.db.query(q)

		for other := range existingChecksums {
			if equalFiles(current, other, t.options.SkipHeader) {
				return other, current, t.options.MatchMode
			}
		}
	}

	if other := t.checkCandidates(current, candidates); other != nil {
		return other, current, t.options.MatchMode
	}

	t.db.insert(current)
	return current, current, fileIsUnique
}

func (t *fileTable) checkCandidates(current *fileRecord, candidates recordSet) (other *fileRecord) {
	// If there were no checksum matches, we need to look at any otherwise-matching files with no checksum yet
	if len(candidates) == 0 {
		return nil
	}

	if err := t.Checksum(current, false); err != nil {
		// We might still find a hardlink match later, even without deep comparison
		return nil
	}

	// Already-checksummed files will have been found and eliminated via the index already
	// We only want to consider files that have not yet been checksummed
	for other := range candidates {
		if err := t.Checksum(other, true); err != nil {
			continue
		}

		if other.Checksum == current.Checksum && equalFiles(current, other, t.options.SkipHeader) {
			return other
		}
	}

	return nil
}
