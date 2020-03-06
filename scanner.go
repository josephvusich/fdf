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

	table *fileTable
	options
	totals
}

func newScanner() *scanner {
	return &scanner{
		table: newFileTable(),
	}
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
	if !f.options.Quiet {
		f.table.termWidth, _ = terminalWidth()
	}
	f.table.minSize = f.options.MinSize
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
		} else if err == noErrSkipped {
			fmt.Printf(" skipped\n")
			f.totals.Skipped.Add(current)
		} else if err != noErrNotDupe {
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
	noErrSkipped = errors.New("skipped")
	noErrNotDupe = errors.New("nothing")
)

func (f *scanner) execute(path string) (current *fileRecord, err error) {
	match, current, areLinked, err := f.table.find(path)

	if current != nil {
		f.totals.Files.Add(current)
	}

	if err != nil {
		return current, err
	}

	if match == current || match.FilePath == current.FilePath {
		f.totals.Unique.Add(current)
		return current, noErrNotDupe
	}

	comparison := "=="
	if areLinked {
		f.totals.Links.Add(current)
		if f.options.IgnoreExistingLinks {
			return current, noErrNotDupe
		}
		comparison = "<=>"
	} else {
		f.totals.Dupes.Add(current)
	}

	fmt.Printf("%s %s %s (%s)\n", match.RelPath, comparison, current.RelPath, humanize.IBytes(uint64(current.Size())))

	verb := f.options.Verb()
	if verb == VerbNone {
		return current, noErrNotDupe
	}

	f.Mutex.Destructive.RLock()
	defer f.Mutex.Destructive.RUnlock()

	// TODO handle uid and gid and perms
	switch verb {
	case VerbDelete:
		fmt.Printf("  delete( %s )", current.RelPath)
		if f.options.DryRun {
			return current, noErrSkipped
		}
		err = os.Remove(current.FilePath)
		if err == nil {
			if areLinked {
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
			if areLinked {
				return current, noErrNotDupe
			}
			x = "hardlink"
			a = os.Link
		} else if verb == VerbSplitLinks {
			if !areLinked {
				return current, noErrNotDupe
			}
			x = "copy"
			a = copyFile
		}
		fmt.Printf("  %s( %s => %s )", x, match.RelPath, current.RelPath)
		if f.options.DryRun {
			return current, noErrSkipped
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

			err = os.Rename(tmp, current.FilePath)
			if err == nil {
				switch verb {
				case VerbMakeLinks:
					f.totals.Dupes.Remove(current)
					f.totals.Links.Add(current)
				case VerbSplitLinks:
					f.totals.Links.Remove(current)
					f.totals.Dupes.Add(current)
				case VerbClone:
					if areLinked {
						f.totals.Links.Remove(current)
					} else {
						f.totals.Dupes.Remove(current)
					}
					f.totals.Cloned.Add(current)
				}
			}
			return current, err
		}
	}

	return current, noErrNotDupe
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
