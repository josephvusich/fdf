package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/josephvusich/go-matchers"
	"github.com/josephvusich/go-matchers/glob"
	"github.com/mattn/go-zglob"
	"github.com/stretchr/testify/require"
)

func TestScanner_SelectAndSwap(t *testing.T) {
	assert := require.New(t)

	s := newScanner()
	s.options.Protect = matchers.RuleSet{}
	m, err := glob.NewMatcher("/foo_001")
	assert.NoError(err)
	m2, err := glob.NewMatcher("/foo")
	assert.NoError(err)
	s.options.Protect.Add(m, true)
	s.options.MustKeep.Add(m2, true)

	current := newFileRecord("/foo", nil, "foo", "")
	match := newFileRecord("/foo_001", nil, "foo_001", "")

	swapMatch, err := s.selectAndSwap(current, match, matchCopyName)
	assert.ErrorIs(err, fileIsSkipped)

	s.options.MustKeep = matchers.RuleSet{DefaultInclude: true}
	current.satisfiesKept = nil
	match.satisfiesKept = nil

	swapMatch, err = s.selectAndSwap(current, match, matchCopyName)
	assert.NoError(err)
	assert.False(swapMatch)

	s.options.Protect = matchers.RuleSet{}
	current.protect = nil
	match.protect = nil

	swapMatch, err = s.selectAndSwap(current, match, matchContent)
	assert.NoError(err)
	assert.True(swapMatch)

	current.protect = nil
	current.satisfiesKept = nil
	match.protect = nil
	match.satisfiesKept = nil
	s.options.MustKeep.Add(m, true)

	swapMatch, err = s.selectAndSwap(current, match, matchContent)
	assert.NoError(err)
	assert.False(swapMatch)
}

func TestScanner_SelectAndSwapMustKeep(t *testing.T) {
	assert := require.New(t)

	s := newScanner()
	s.options.Protect = matchers.RuleSet{}
	m, err := glob.NewMatcher("/foo_001")
	assert.NoError(err)
	s.options.Protect.Add(m, true)

	current := newFileRecord("/foo", nil, "foo", "")
	match := newFileRecord("/foo_001", nil, "foo_001", "")

	swapMatch, err := s.selectAndSwap(current, match, matchCopyName)
	assert.NoError(err)
	assert.False(swapMatch)

	s.options.Protect = matchers.RuleSet{}
	current.protect = nil
	match.protect = nil

	swapMatch, err = s.selectAndSwap(current, match, matchContent)
	assert.NoError(err)
	assert.True(swapMatch)
}

func TestScanner_NoVerb(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.MatchMode = matchContent
		scanner.options.Recursive = true
		scanner.options.minSize = 1

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(16), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(6), scanner.totals.Unique.count)
		assert.Equal(uint64(33), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(10), scanner.totals.Dupes.count)
		assert.Equal(uint64(40), scanner.totals.Dupes.size)
		assert.Equal(uint64(0), scanner.totals.Processed.count)
		assert.Equal(uint64(0), scanner.totals.Processed.size)
		assert.Equal(uint64(6), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)
	})
}

func TestScanner_LinkUnlink(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.makeLinks = true
		scanner.options.Recursive = true
		scanner.options.MatchMode = matchContent

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(7), scanner.totals.Unique.count)
		assert.Equal(uint64(33), scanner.totals.Unique.size)
		assert.Equal(uint64(15), scanner.totals.Links.count)
		assert.Equal(uint64(40), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(40), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)

		// Validate hardlink behavior
		func() {
			f, err := os.OpenFile(filepath.Join(l.dirs[0], "bar"), os.O_WRONLY, 0666)
			assert.NoError(err)
			defer f.Close()
			_, err = f.Seek(0, io.SeekStart)
			assert.NoError(err)
			assert.NoError(f.Truncate(0))
			_, err = f.WriteString("hello world")
			assert.NoError(err)
			l.content["bar"] = "hello world"
			l.content["bar2"] = "hello world"
			l.content["bar3"] = "hello world"
		}()
		validate(l)

		// Phase 2: Copy
		scanner = newScanner()
		scanner.options.splitLinks = true
		scanner.options.Recursive = true
		scanner.options.MatchMode = matchContent

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(115), scanner.totals.Files.size)
		assert.Equal(uint64(7), scanner.totals.Unique.count)
		assert.Equal(uint64(40), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(15), scanner.totals.Dupes.count)
		assert.Equal(uint64(75), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(75), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)

		// Validate hardlink split
		for i := 0; i < len(l.dirs); i++ {
			func(i int) {
				f, err := os.OpenFile(filepath.Join(l.dirs[i], "bar"), os.O_WRONLY, 0666)
				assert.NoError(err)
				defer f.Close()
				_, err = f.Seek(0, io.SeekStart)
				assert.NoError(err)
				assert.NoError(f.Truncate(0))
				_, err = f.WriteString("goodbye world")
				assert.NoError(err)
			}(i)
		}
		l.content["bar"] = "goodbye world"
		validate(l)
	})
}

func TestScanner_Timestamps(t *testing.T) {
	assert := require.New(t)

	now := time.Now()
	newTs := now.Add(time.Hour)
	oldTs := now.Add(-time.Hour)

	testTimestamps := func(aTime, bTime time.Time, mode string, expectA bool) {
		l := &testLayout{
			dirs: []string{
				"./d",
			},
			content: map[string]string{
				"a": "foobar",
				"b": "foobar",
			},
			diffContent: nil,
			diffSize:    nil,
		}

		setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
			scanner := newScanner()
			assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rdv`, `--timestamps`, mode}))

			assert.NoError(os.Chtimes("d/a", aTime, aTime))
			assert.NoError(os.Chtimes("d/b", bTime, bTime))

			assert.NoError(scanner.Scan())
			l.contentOverride = true
			l.content = map[string]string{}
			if expectA {
				l.content["d/a"] = "foobar"
			} else {
				l.content["d/b"] = "foobar"
			}
			validate(l)
		})
	}

	testTimestamps(newTs, oldTs, "prefer-newer", true)
	testTimestamps(newTs, oldTs, "prefer-older", false)
	testTimestamps(oldTs, newTs, "prefer-newer", false)
	testTimestamps(oldTs, newTs, "prefer-older", true)
}

func TestScanner_Delete(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.deleteDupes = true
		scanner.options.Recursive = true
		scanner.options.MatchMode = matchContent

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(7), scanner.totals.Unique.count)
		assert.Equal(uint64(33), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(40), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"a/bar":         "bar\n",
			"a/diffContent": "fizz\n",
			"a/diffSize":    "foobar\n",
			"a/empty":       "",
			"a/foo":         "foo\n",
			"b/diffContent": "buzz\n",
			"b/diffSize":    "foobar2\n",
		}
		validate(l)
	})
}

func TestScanner_DeleteProtect(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `--protect`, `./b/**/*`, `-m`, `content`, `-z`, `0`, `--timestamps=ignore`}))
		assert.True(scanner.options.deleteDupes)
		assert.True(scanner.options.Recursive)
		assert.Equal(matchContent, scanner.options.MatchMode)

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(13), scanner.totals.Unique.count)
		assert.Equal(uint64(49), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(9), scanner.totals.Processed.count)
		assert.Equal(uint64(24), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"a/diffContent": "fizz\n",
			"a/diffSize":    "foobar\n",
			"b/bar":         "bar\n",
			"b/bar2":        "bar\n",
			"b/bar3":        "bar\n",
			"b/empty":       "",
			"b/empty2":      "",
			"b/empty3":      "",
			"b/foo":         "foo\n",
			"b/foo2":        "foo\n",
			"b/foo3":        "foo\n",
			"b/diffContent": "buzz\n",
			"b/diffSize":    "foobar2\n",
		}
		validate(l)
	})
}

func TestScanner_NameSize(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.MatchMode = matchName | matchSize
		scanner.options.makeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(12), scanner.totals.Unique.count)
		assert.Equal(uint64(44), scanner.totals.Unique.size)
		assert.Equal(uint64(10), scanner.totals.Links.count)
		assert.Equal(uint64(29), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(10), scanner.totals.Processed.count)
		assert.Equal(uint64(29), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.diffContent[1] = l.diffContent[0]
		validate(l)
	})
}

func TestScanner_NameContent(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.MatchMode = matchName | matchContent
		scanner.options.makeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(13), scanner.totals.Unique.count)
		assert.Equal(uint64(49), scanner.totals.Unique.size)
		assert.Equal(uint64(9), scanner.totals.Links.count)
		assert.Equal(uint64(24), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(9), scanner.totals.Processed.count)
		assert.Equal(uint64(24), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)
	})
}

func TestScanner_CopyNameContent(t *testing.T) {
	assert := require.New(t)

	l := &testLayout{
		dirs: []string{
			"./a",
			"./b",
		},
		content: map[string]string{
			"bar":         "bar\n",
			"Copy of bar": "bar\n",
			"bar (1)":     "bar\n",
			"bar-01":      "bar\n",
			"foo":         "bar\n",
			"bar.foo":     "bar\n",
		},
		diffContent: []string{
			"fizz\n",
			"buzz\n",
		},
		diffSize: []string{
			"foobar\n",
			"foobar2\n",
		},
	}

	setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.MatchMode = matchCopyName | matchContent
		scanner.options.makeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(16), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(7), scanner.totals.Unique.count)
		assert.Equal(uint64(37), scanner.totals.Unique.size)
		assert.Equal(uint64(9), scanner.totals.Links.count)
		assert.Equal(uint64(36), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(9), scanner.totals.Processed.count)
		assert.Equal(uint64(36), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)
	})
}

func TestScanner_NameSuffix(t *testing.T) {
	assert := require.New(t)

	l := &testLayout{
		dirs: []string{
			"./a",
			"./b",
		},
		content: map[string]string{
			"bar":         "bar\n",
			"Copy of bar": "bar\n",
			"bar (1)":     "bar\n",
			"bar-01":      "bar\n",
			"foo":         "bar\n",
			"bar.foo":     "bar\n",
		},
		diffContent: []string{
			"fizz\n",
			"buzz\n",
		},
		diffSize: []string{
			"foobar\n",
			"foobar2\n",
		},
	}

	setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.MatchMode = matchNameSuffix | matchContent
		scanner.options.deleteDupes = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(16), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(8), scanner.totals.Unique.count)
		assert.Equal(uint64(41), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(8), scanner.totals.Processed.count)
		assert.Equal(uint64(32), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"a/bar":         "bar\n",
			"a/bar (1)":     "bar\n",
			"a/bar.foo":     "bar\n",
			"a/bar-01":      "bar\n",
			"a/diffContent": "fizz\n",
			"a/diffSize":    "foobar\n",
			"b/diffContent": "buzz\n",
			"b/diffSize":    "foobar2\n",
		}
		validate(l)
	})
}

func TestScanner_NameOnly(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.MatchMode = matchName
		scanner.options.makeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(11), scanner.totals.Unique.count)
		// this might change by one if the other file is found first
		assert.Equal(uint64(36), scanner.totals.Unique.size)
		assert.Equal(uint64(11), scanner.totals.Links.count)
		assert.Equal(uint64(36), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(11), scanner.totals.Processed.count)
		// this will be one larger than Unique size because b/diffSize becomes smaller
		assert.Equal(uint64(37), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.diffContent[1] = l.diffContent[0]
		l.diffSize[1] = l.diffSize[0]
		validate(l)
	})
}

func TestScanner_SizeOnly(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.MatchMode = matchSize
		scanner.options.makeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(22), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(5), scanner.totals.Unique.count)
		assert.Equal(uint64(24), scanner.totals.Unique.size)
		assert.Equal(uint64(17), scanner.totals.Links.count)
		assert.Equal(uint64(49), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(17), scanner.totals.Processed.count)
		assert.Equal(uint64(49), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.content["foo"] = l.content["bar"]
		l.content["foo2"] = l.content["bar"]
		l.content["foo3"] = l.content["bar"]
		l.diffContent[1] = l.diffContent[0]
		validate(l)
	})
}

func TestScanner_SkipHeader(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.MatchMode = matchContent
		scanner.options.makeLinks = true
		scanner.options.Recursive = true
		scanner.options.SkipHeader = 3

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(16), scanner.totals.Files.count)
		assert.Equal(uint64(73), scanner.totals.Files.size)
		assert.Equal(uint64(4), scanner.totals.Unique.count)
		assert.Equal(uint64(24), scanner.totals.Unique.size)
		assert.Equal(uint64(12), scanner.totals.Links.count)
		assert.Equal(uint64(49), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(12), scanner.totals.Processed.count)
		assert.Equal(uint64(49), scanner.totals.Processed.size)
		assert.Equal(uint64(6), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.content["foo"] = l.content["bar"]
		l.content["foo2"] = l.content["bar"]
		l.content["foo3"] = l.content["bar"]
		l.diffContent[1] = l.diffContent[0]
		validate(l)
	})
}

func TestScanner_Parent(t *testing.T) {
	assert := require.New(t)
	l := &testLayout{
		dirs: []string{
			"./foo/a",
			"./foo/b",
			"./bar/a",
			"./bar/b",
		},
		content: map[string]string{
			"fizz1": "fizz",
			"fizz2": "fizz",
			"buzz":  "buzz",
		},
		diffContent: nil,
		diffSize:    nil,
	}
	setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `content+parent`, `-z`, `0`, `--timestamps=ignore`}))
		assert.True(scanner.options.deleteDupes)
		assert.True(scanner.options.Recursive)
		assert.Equal(matchContent|matchParent, scanner.options.MatchMode)

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(12), scanner.totals.Files.count)
		assert.Equal(uint64(48), scanner.totals.Files.size)
		assert.Equal(uint64(4), scanner.totals.Unique.count)
		assert.Equal(uint64(16), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(8), scanner.totals.Processed.count)
		assert.Equal(uint64(32), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"bar/a/fizz1": "fizz",
			"bar/a/buzz":  "buzz",
			"bar/b/fizz1": "fizz",
			"bar/b/buzz":  "buzz",
		}
		validate(l)
	})
}

func TestScanner_Path(t *testing.T) {
	assert := require.New(t)
	l := &testLayout{
		dirs: []string{
			"./foo/a",
			"./foo/b",
			"./bar/a",
			"./bar/b",
		},
		content: map[string]string{
			"fizz1": "fizz",
			"fizz2": "fizz",
			"buzz":  "buzz",
		},
		diffContent: nil,
		diffSize:    nil,
	}
	setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `path+content`, `-z`, `0`, `--timestamps=ignore`}))
		assert.True(scanner.options.deleteDupes)
		assert.True(scanner.options.Recursive)
		assert.Equal(matchContent|matchParent|matchPathSuffix, scanner.options.MatchMode)

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(12), scanner.totals.Files.count)
		assert.Equal(uint64(48), scanner.totals.Files.size)
		assert.Equal(uint64(8), scanner.totals.Unique.count)
		assert.Equal(uint64(32), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(4), scanner.totals.Processed.count)
		assert.Equal(uint64(16), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"foo/a/fizz1": "fizz",
			"foo/a/buzz":  "buzz",
			"foo/b/fizz1": "fizz",
			"foo/b/buzz":  "buzz",
			"bar/a/fizz1": "fizz",
			"bar/a/buzz":  "buzz",
			"bar/b/fizz1": "fizz",
			"bar/b/buzz":  "buzz",
		}
		validate(l)
	})
}

func TestScanner_PathSuffix(t *testing.T) {
	assert := require.New(t)
	l := &testLayout{
		dirs: []string{
			"./foo/a",
			"./foo/b",
			"./bar/a",
			"./bar/b",
		},
		content: map[string]string{
			"fizz1": "fizz",
			"fizz2": "fizz",
			"buzz":  "buzz",
		},
		diffContent: nil,
		diffSize:    nil,
	}
	setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `relpath+content`, `-z`, `0`, `--timestamps=ignore`}))
		assert.True(scanner.options.deleteDupes)
		assert.True(scanner.options.Recursive)
		assert.Equal(matchContent|matchParent|matchPathSuffix, scanner.options.MatchMode)

		assert.NoError(scanner.Scan("./foo", "./bar"))
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(12), scanner.totals.Files.count)
		assert.Equal(uint64(48), scanner.totals.Files.size)
		assert.Equal(uint64(4), scanner.totals.Unique.count)
		assert.Equal(uint64(16), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(8), scanner.totals.Processed.count)
		assert.Equal(uint64(32), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Skipped.count)
		assert.Equal(uint64(0), scanner.totals.Skipped.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.contentOverride = true
		l.content = map[string]string{
			"foo/a/fizz1": "fizz",
			"foo/a/buzz":  "buzz",
			"foo/b/fizz1": "fizz",
			"foo/b/buzz":  "buzz",
		}
		validate(l)
	})
}

type testLayout struct {
	dirs []string

	// Duplicated per dirList[n]
	content map[string]string

	// Used for certain cases, such as delete, that need to override the default layout
	// If this is set, content keys are relative paths and all other fields are ignored
	contentOverride bool

	// dirList[n]/different == different[n]
	diffContent []string

	// dirList[n]/diffsize == diffsize[n]
	diffSize []string
}

func setupTest(assert *require.Assertions, f func(l *testLayout, validate func(*testLayout))) {
	l := &testLayout{
		dirs: []string{
			"./a",
			"./b",
		},
		content: map[string]string{
			"bar":    "bar\n",
			"bar2":   "bar\n",
			"bar3":   "bar\n",
			"foo":    "foo\n",
			"foo2":   "foo\n",
			"foo3":   "foo\n",
			"empty":  "",
			"empty2": "",
			"empty3": "",
		},
		diffContent: []string{
			"fizz\n",
			"buzz\n",
		},
		diffSize: []string{
			"foobar\n",
			"foobar2\n",
		},
	}
	setupTestLayout(assert, l, f)
}

func setupTestLayout(assert *require.Assertions, l *testLayout, f func(l *testLayout, validate func(*testLayout))) {
	dir, err := ioutil.TempDir("", "fdftest")
	assert.NoError(err)
	defer os.RemoveAll(dir)
	assert.NoError(os.Chdir(dir))

	for i, d := range l.dirs {
		assert.NoError(os.MkdirAll(d, 0777))
		for f, c := range l.content {
			assert.NoError(ioutil.WriteFile(filepath.Join(d, fmt.Sprintf("%s", f)), []byte(c), 0666))
		}
		if len(l.diffContent) != 0 {
			assert.NoError(ioutil.WriteFile(filepath.Join(d, "diffContent"), []byte(l.diffContent[i]), 0666))
		}
		if len(l.diffSize) != 0 {
			assert.NoError(ioutil.WriteFile(filepath.Join(d, "diffSize"), []byte(l.diffSize[i]), 0666))
		}
	}

	f(l, func(l *testLayout) {
		glob, err := zglob.Glob("./**/*")
		var g []string
		for _, x := range glob {
			st, err := os.Stat(x)
			assert.NoError(err)
			if !st.IsDir() {
				g = append(g, x)
			}
		}
		assert.NoError(err)

		if l.contentOverride {
			assert.Len(g, len(l.content), "wrong number of files")

			for f, c := range l.content {
				b, err := ioutil.ReadFile(f)
				assert.NoError(err)
				assert.Equalf(c, string(b), "%s", f)
			}
		} else {
			assert.Len(g, (len(l.dirs)*len(l.content))+len(l.diffContent)+len(l.diffSize))

			for i, d := range l.dirs {
				for f, c := range l.content {
					b, err := ioutil.ReadFile(filepath.Join(d, f))
					assert.NoError(err)
					assert.Equalf(c, string(b), "%s", f)
				}
				if len(l.diffContent) != 0 {
					b, err := ioutil.ReadFile(filepath.Join(d, "diffContent"))
					assert.NoError(err)
					assert.Equal(l.diffContent[i], string(b))
				}
				if len(l.diffSize) != 0 {
					b, err := ioutil.ReadFile(filepath.Join(d, "diffSize"))
					assert.NoError(err)
					assert.Equal(l.diffSize[i], string(b))
				}
			}
		}
	})
}

func TestScanner_Symlinks(t *testing.T) {
	t.Run("file symlinks are ignored", func(t *testing.T) {
		assert := require.New(t)
		l := &testLayout{
			dirs: []string{"./a"},
			content: map[string]string{
				"foo": "foo\n",
			},
		}
		setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
			target, err := filepath.Abs(filepath.Join(".", "a", "foo"))
			assert.NoError(err)
			assert.NoError(os.Symlink(target, filepath.Join(".", "a", "link")))

			scanner := newScanner()
			scanner.options.MatchMode = matchContent
			scanner.options.Recursive = true

			assert.NoError(scanner.Scan())
			assert.Equal(uint64(1), scanner.totals.Files.count)
			assert.Equal(uint64(0), scanner.totals.Dupes.count)
		})
	})

	t.Run("directory symlinks are not traversed", func(t *testing.T) {
		assert := require.New(t)
		l := &testLayout{
			dirs: []string{"./a"},
			content: map[string]string{
				"foo": "foo\n",
			},
		}
		setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
			target, err := filepath.Abs(filepath.Join(".", "a"))
			assert.NoError(err)
			assert.NoError(os.Symlink(target, filepath.Join(".", "b")))

			scanner := newScanner()
			scanner.options.MatchMode = matchContent
			scanner.options.Recursive = true

			assert.NoError(scanner.Scan())
			assert.Equal(uint64(1), scanner.totals.Files.count)
			assert.Equal(uint64(0), scanner.totals.Dupes.count)
		})
	})

	t.Run("delete ignores symlinks", func(t *testing.T) {
		assert := require.New(t)
		l := &testLayout{
			dirs: []string{"./a", "./b"},
			content: map[string]string{
				"foo": "foo\n",
			},
		}
		setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
			target, err := filepath.Abs(filepath.Join(".", "a", "foo"))
			assert.NoError(err)
			linkPath := filepath.Join(".", "a", "link")
			assert.NoError(os.Symlink(target, linkPath))

			scanner := newScanner()
			scanner.options.deleteDupes = true
			scanner.options.MatchMode = matchContent
			scanner.options.Recursive = true

			assert.NoError(scanner.Scan())
			assert.Equal(uint64(2), scanner.totals.Files.count)
			assert.Equal(uint64(1), scanner.totals.Processed.count)

			// Symlink still exists
			linfo, err := os.Lstat(linkPath)
			assert.NoError(err)
			assert.NotZero(linfo.Mode() & os.ModeSymlink)
		})
	})

	t.Run("hardlink ignores symlinks", func(t *testing.T) {
		assert := require.New(t)
		l := &testLayout{
			dirs: []string{"./a", "./b"},
			content: map[string]string{
				"foo": "foo\n",
			},
		}
		setupTestLayout(assert, l, func(l *testLayout, validate func(*testLayout)) {
			target, err := filepath.Abs(filepath.Join(".", "a", "foo"))
			assert.NoError(err)
			linkPath := filepath.Join(".", "a", "link")
			assert.NoError(os.Symlink(target, linkPath))

			scanner := newScanner()
			scanner.options.makeLinks = true
			scanner.options.MatchMode = matchContent
			scanner.options.Recursive = true

			assert.NoError(scanner.Scan())
			assert.Equal(uint64(2), scanner.totals.Files.count)
			assert.Equal(uint64(1), scanner.totals.Processed.count)

			// Symlink still exists and is still a symlink
			linfo, err := os.Lstat(linkPath)
			assert.NoError(err)
			assert.NotZero(linfo.Mode() & os.ModeSymlink)

			// Real files are now hardlinked
			aInfo, err := os.Stat(filepath.Join(".", "a", "foo"))
			assert.NoError(err)
			bInfo, err := os.Stat(filepath.Join(".", "b", "foo"))
			assert.NoError(err)
			assert.True(os.SameFile(aInfo, bInfo))
		})
	})
}

func TestUSubtract(t *testing.T) {
	assert := require.New(t)

	// 5 - 1 = 4
	var x uint64 = 5
	atomic.AddUint64(&x, uSubtract(1))
	assert.Equal(uint64(4), x)

	// n - n = 0
	x = 42
	atomic.AddUint64(&x, uSubtract(42))
	assert.Equal(uint64(0), x)

	// uSubtract(0) is identity (adding 0)
	assert.Equal(uint64(0), uSubtract(0))
}

func TestTotal_AddRemoveGet(t *testing.T) {
	assert := require.New(t)

	var tot total

	r1 := &fileRecord{FileInfo: &fakeStat{size: 100}}
	r2 := &fileRecord{FileInfo: &fakeStat{size: 200}}

	tot.Add(r1)
	tot.Add(r2)
	count, size := tot.Get()
	assert.Equal(uint64(2), count)
	assert.Equal(uint64(300), size)

	tot.Remove(r1)
	count, size = tot.Get()
	assert.Equal(uint64(1), count)
	assert.Equal(uint64(200), size)
}
