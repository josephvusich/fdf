package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josephvusich/go-matchers"
)

type recordSet map[*fileRecord]struct{}

type fileTable struct {
	// The directory passed to scanner.Scan
	scanDir string

	// scanDir relative to the startup working directory
	relDir string

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
	// Absolute file path.
	FilePath string

	// File path relative to startup working directory.
	RelPath string

	// File path relative to the respective dir passed to scanner.Scan
	PathSuffix string

	// Lowercased filename for case-insensitive matching.
	FoldedName string

	// Lowercased parent directory basename.
	FoldedParent string

	os.FileInfo
	HasChecksum    bool
	FailedChecksum error
	Checksum       checksum

	// true/false indicates whether this file is protected from destructive operations.
	// nil if protection status has not yet been determined.
	protect *bool

	satisfiesKept *bool
}

func foldName(filePath string) string {
	return strings.ToLower(filepath.Base(filePath))
}

func (r *fileRecord) SatisfiesKept(k *matchers.RuleSet) bool {
	if r.satisfiesKept == nil {
		ok := k.Includes(r.FilePath)
		r.satisfiesKept = &ok
	}
	return *r.satisfiesKept
}

// Note that `p` is ignored if there is already a cached result
func (r *fileRecord) Protect(p *matchers.RuleSet) bool {
	if r.protect == nil {
		ok := p.Includes(r.FilePath)
		r.protect = &ok
	}
	return *r.protect
}

func (r *fileRecord) String() string {
	return fmt.Sprintf("%s: %t %X", r.FilePath, r.HasChecksum, r.Checksum)
}

func newFileRecord(path string, info os.FileInfo, relPath string, pathSuffix string) *fileRecord {
	return &fileRecord{
		FilePath:     path,
		RelPath:      relPath,
		PathSuffix:   pathSuffix,
		FoldedName:   foldName(path),
		FoldedParent: foldName(filepath.Base(filepath.Dir(path))),
		FileInfo:     info,
	}
}

// Rel returns absPath relative to the startup working directory,
// or absPath if filepath.Rel fails.
func (t *fileTable) Rel(absPath string) (rel string) {
	rel, err := filepath.Rel(t.scanDir, absPath)
	if err != nil {
		return absPath
	}
	return filepath.Join(t.relDir, rel)
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
	matchNothing    matchFlag = 0b0000000000000000                // default value, usually replaced with matchContent
	matchName       matchFlag = 0b0000000000000001                // case-insensitive
	matchSize       matchFlag = 0b0000000000000010                // implied by matchContent and matchHardlink
	matchContent              = 0b0000000000000100 | matchSize    // implied by matchHardlink
	matchHardlink             = 0b0000000000001000 | matchContent // used by --copy and for categorization
	matchCopyName             = 0b0000000000010000                // one filename must contain the other, e.g., "foo" and "foo copy (1)"
	matchParent               = 0b0000000000100000                // parent directory name (folded)
	matchPathSuffix           = 0b0000000001000000 | matchParent  // path relative to the directory passed to scanner.Scan
	matchNameSuffix           = 0b0000000010000000                // one filename must end with the other, e.g., "foo-fizz-buzz" and "fizz-buzz"
	fileIsUnique    matchFlag = 0b0010000000000000                // no match found
	fileIsSkipped   matchFlag = 0b0100000000000000                // file was excluded e.g., due to size requirements
	fileIsIgnored   matchFlag = 0b1000000000000000                // status returned for directories
)

func (m matchFlag) has(flag matchFlag) bool {
	return m&flag == flag
}

func (m matchFlag) Error() string {
	return fmt.Sprintf("MatchType<0b%b>", m)
}

func (t *fileTable) find(f, pathSuffix string) (match *fileRecord, current *fileRecord, err error) {
	if t.options.Exclude.Includes(f) {
		return nil, nil, fileIsIgnored
	}
	st, err := os.Stat(f)
	if err != nil {
		return nil, nil, err
	}
	return t.findStat(f, st, pathSuffix)
}

func (t *fileTable) findStat(f string, st os.FileInfo, pathSuffix string) (match *fileRecord, current *fileRecord, err error) {
	if st.IsDir() {
		return nil, nil, fileIsIgnored
	}

	if st.Size() < t.options.MinSize() {
		return nil, nil, fileIsSkipped
	}

	current = newFileRecord(f, st, t.Rel(f), pathSuffix)

	q := &query{}
	if t.options.MatchMode.has(matchName) {
		current.byName(q)
	}
	if t.options.MatchMode.has(matchParent) {
		current.byParent(q)
	}
	if t.options.MatchMode.has(matchPathSuffix) {
		current.byPathSuffix(q)
	}
	if t.options.MatchMode.has(matchSize) {
		current.bySize(q)
	}
	// Ignore checksums for now, as hardlinks can match content without the overhead of comparison

	// Query for any known files that match all desired fields (except content/checksum)
	candidates := t.db.query(q)

	// If the current file is protected, filter for unprotected candidates
	if current.Protect(&t.options.Protect) {
		filtered := recordSet{}
		for other := range candidates {
			if !other.Protect(&t.options.Protect) {
				filtered[other] = struct{}{}
			}
		}
		candidates = filtered
	}

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

	if t.options.MatchMode.has(matchNameSuffix) {
		filtered := recordSet{}
		for other := range candidates {
			if isNameSuffix(current.FoldedName, other.FoldedName) {
				filtered[other] = struct{}{}
			}
		}
		candidates = filtered
	}

	if len(t.options.Comparers) != 0 {
		filtered := recordSet{}
		for other := range candidates {
			allMatch := true
			for _, c := range t.options.Comparers {
				if !c.AreEqual(current, other) {
					allMatch = false
					break
				}
			}
			if allMatch {
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
			if equalFiles(current, other, t.options) {
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

		if other.Checksum == current.Checksum && equalFiles(current, other, t.options) {
			return other
		}
	}

	return nil
}
