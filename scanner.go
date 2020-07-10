package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
)

type scanner struct {
	Mutex struct {
		// For removing filesystem files prior to linking/cloning, get read lock
		// Termination of process requires write lock
		Destructive sync.RWMutex
	}

	table   *fileTable
	options options
	totals  totals
}

func newScanner() *scanner {
	s := &scanner{}
	s.table = newFileTable(&s.options, &s.totals)
	return s
}

// Don't display warnings for these dotfiles
var silentSkip = map[string]struct{}{
	".DS_Store":               {},
	".DocumentRevisions-V100": {},
	".Spotlight-V100":         {},
	".TemporaryItems":         {},
	".Trashes":                {},
	".fseventsd":              {},
}

func (f *scanner) Scan() (err error) {
	f.table.wd, err = os.Getwd()
	if err != nil {
		return err
	}
	if f.options.MatchMode == 0 {
		return errors.New("MatchMode not specified in options")
	}
	if !f.options.Quiet {
		f.table.termWidth, _ = terminalWidth()
	}
	f.totals.Start()

	return filepath.Walk(f.table.wd, func(path string, info os.FileInfo, inErr error) error {
		base := filepath.Base(path)
		typ := info.Mode()

		if base[0] == '.' || inErr != nil {
			_, silent := silentSkip[base]
			if !silent {
				if inErr != nil {
					fmt.Printf("%s: %s\n", path, inErr)
					return nil
				}
				if !f.options.Quiet {
					fmt.Printf("%s: skipping dot-prefix\n", path)
				}
			}
			if inErr == nil && typ.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		f.table.progress(path, true)

		// Avoid hogging too many resources
		time.Sleep(0)

		if !f.options.Recursive && typ.IsDir() && path != f.table.wd {
			return filepath.SkipDir
		}

		if typ&os.ModeSymlink != 0 {
			if typ.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		current, err := f.execute(path)
		if err == nil {
			fmt.Printf(" success\n")
			f.totals.Processed.Add(current)
		} else if err == noErrDryRun || err == fileIsSkipped {
			if err == noErrDryRun {
				fmt.Printf(" skipped\n")
			}
			f.totals.Skipped.Add(current)
		} else if err != fileIsIgnored {
			f.totals.Errors.Add(current)
			if current != nil {
				fmt.Printf(" %s: %s\n", current.RelPath, err)
			} else {
				fmt.Println(err)
			}
		}

		return nil
	})
}

var (
	// Used as a special status for dry-runs
	// Files skipped for other reasons should use fileIsSkipped
	// Unlike fileIsSkipped, noErrDryRun displays the filepath along with "skipped"
	noErrDryRun = errors.New("skipped")
)

func (f *scanner) execute(path string) (current *fileRecord, err error) {
	match, current, err := f.table.find(path)

	if current != nil {
		f.totals.Files.Add(current)
	}

	m, ok := err.(matchFlag)
	if !ok || m == fileIsIgnored || m == fileIsSkipped {
		return current, err
	}

	if m == fileIsUnique || match == current || match.FilePath == current.FilePath {
		f.totals.Unique.Add(current)
		return current, fileIsIgnored
	}

	comparison := "=="
	if m.has(matchHardlink) {
		f.totals.Links.Add(current)
		if f.options.IgnoreExistingLinks {
			return current, fileIsIgnored
		}
		comparison = "<=>"
	} else {
		f.totals.Dupes.Add(current)
	}

	if f.options.Verbose || !current.Preserve(f.options.Preserve) || !match.Preserve(f.options.Preserve) {
		fmt.Printf("%s %s %s (%s)\n", match.RelPath, comparison, current.RelPath, humanize.IBytes(uint64(current.Size())))
	}

	verb := f.options.Verb()
	if verb == VerbNone {
		return current, fileIsIgnored
	}

	if current.Preserve(f.options.Preserve) {
		if f.options.Verbose {
			fmt.Printf("  preserve( %s ) matched\n", current.PreserveReason)
			fmt.Printf("    skip( %s ) preserved\n", current.RelPath)
		}
		if match.Preserve(f.options.Preserve) {
			if f.options.Verbose {
				if current.PreserveReason != match.PreserveReason {
					fmt.Printf("  preserve( %s ) matched\n", match.PreserveReason)
				}
				fmt.Printf("    skip( %s ) preserved\n", match.RelPath)
			}
			return current, fileIsSkipped
		}
		f.table.db.remove(match)
		f.table.db.insert(current)
		match, current = current, match
	}

	f.Mutex.Destructive.RLock()
	defer f.Mutex.Destructive.RUnlock()

	// TODO handle uid and gid and perms
	switch verb {
	case VerbDelete:
		fmt.Printf("  delete( %s )", current.RelPath)
		if f.options.DryRun {
			return current, noErrDryRun
		}
		err = os.Remove(current.FilePath)
		if err == nil {
			if m.has(matchHardlink) {
				f.totals.Links.Remove(current)
			} else {
				f.totals.Dupes.Remove(current)
			}
		}
		return current, err
	case VerbClone, VerbMakeLinks, VerbSplitLinks:
		x := "clone"
		a := cloneFile
		if verb == VerbMakeLinks {
			if m.has(matchHardlink) {
				return current, fileIsIgnored
			}
			x = "hardlink"
			a = os.Link
		} else if verb == VerbSplitLinks {
			if !m.has(matchHardlink) {
				return current, fileIsIgnored
			}
			x = "copy"
			a = copyFile
		}
		fmt.Printf("  %s( %s => %s )", x, match.RelPath, current.RelPath)
		if f.options.DryRun {
			return current, noErrDryRun
		}
		for retry := 0; retry < 3; retry++ {
			tmp, err := tempName(current)
			if err != nil {
				return current, err
			}

			if err = a(match.FilePath, tmp); err != nil {
				if errors.Is(err, syscall.EEXIST) {
					continue
				}
				os.Remove(tmp)
				return current, fmt.Errorf("%s: %w", f.table.Rel(tmp), err)
			}

			if err = os.Rename(tmp, current.FilePath); err == nil {
				switch verb {
				case VerbMakeLinks:
					f.totals.Dupes.Remove(current)
					f.totals.Links.Add(match)
				case VerbSplitLinks:
					f.totals.Links.Remove(current)
					f.totals.Dupes.Add(current)
				case VerbClone:
					if m.has(matchHardlink) {
						f.totals.Links.Remove(current)
					} else {
						f.totals.Dupes.Remove(current)
					}
					f.totals.Cloned.Add(match)
				}
			}
			return current, err
		}
	}

	return current, fileIsIgnored
}

func tempName(r *fileRecord) (string, error) {
	dir := filepath.Dir(r.FilePath)
	if dir == "" {
		dir = "."
	}
	f, err := ioutil.TempFile(dir, ".fdf-")
	if err != nil {
		return "", err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	return name, nil
}

func (f *scanner) Exit(code int) {
	f.Mutex.Destructive.Lock()
	os.Exit(code)
}

type totals struct {
	Started time.Time

	Files  total
	Unique total
	Dupes  total
	Cloned total
	Links  total

	Processed total
	Skipped   total
	Errors    total
}

type total struct {
	count uint64
	size  uint64
}

func (t *totals) PrettyFormat(v verb) string {
	lines := []string{
		fmt.Sprintf("%s elapsed", t.End()),
	}

	for _, x := range []struct {
		total
		suffix string
	}{
		{t.Files, "scanned"},
		{t.Unique, "unique"},
		{t.Links, "as hardlinks"},
		{t.Cloned, "as clones"},
		{t.Dupes, "duplicated"},
		{},
		{t.Processed, fmt.Sprintf("%s successfully", v.PastTense())},
		{t.Skipped, "skipped"},
		{t.Errors, "had errors"},
	} {
		if x.count != 0 {
			lines = append(lines, fmt.Sprintf("%s %s", x.String(), x.suffix))
		} else if x.suffix == "" {
			lines = append(lines, "")
		}
	}
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func (t *total) String() string {
	count, size := t.Get()
	return fmt.Sprintf("%d files (%s)", count, humanize.IBytes(size))
}

func (t *totals) Start() {
	t.Started = time.Now()
}

func (t *totals) End() time.Duration {
	return time.Since(t.Started)
}

func (t *total) Add(r *fileRecord) {
	atomic.AddUint64(&t.count, 1)
	if r != nil && r.Size() > 0 {
		atomic.AddUint64(&t.size, uint64(r.Size()))
	}
}

func (t *total) Remove(r *fileRecord) {
	atomic.AddUint64(&t.count, uSubtract(1))
	if r != nil && r.Size() > 0 {
		atomic.AddUint64(&t.size, uSubtract(uint64(r.Size())))
	}
}

func uSubtract(positive uint64) (negative uint64) {
	return ^(positive - 1)
}

func (t *total) Get() (count, size uint64) {
	return atomic.LoadUint64(&t.count), atomic.LoadUint64(&t.size)
}
