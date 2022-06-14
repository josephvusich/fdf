package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattn/go-zglob"
	"github.com/stretchr/testify/require"
)

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
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `--protect`, `./b/**/*`, `-m`, `content`, `-z`, `0`}))
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
		// Phase 1: Hardlink
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
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `content+parent`, `-z`, `0`}))
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
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `path+content`, `-z`, `0`}))
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
		assert.Empty(scanner.options.ParseArgs([]string{`fdf`, `-rd`, `-m`, `relpath+content`, `-z`, `0`}))
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
