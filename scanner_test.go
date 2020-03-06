package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanner_NoVerb(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.Recursive = true
		scanner.options.minSize = 1

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(14), scanner.totals.Files.count)
		assert.Equal(uint64(58), scanner.totals.Files.size)
		assert.Equal(uint64(4), scanner.totals.Unique.count)
		assert.Equal(uint64(18), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(10), scanner.totals.Dupes.count)
		assert.Equal(uint64(40), scanner.totals.Dupes.size)
		assert.Equal(uint64(0), scanner.totals.Processed.count)
		assert.Equal(uint64(0), scanner.totals.Processed.size)
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
		scanner.options.MakeLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(20), scanner.totals.Files.count)
		assert.Equal(uint64(58), scanner.totals.Files.size)
		assert.Equal(uint64(5), scanner.totals.Unique.count)
		assert.Equal(uint64(18), scanner.totals.Unique.size)
		assert.Equal(uint64(15), scanner.totals.Links.count)
		assert.Equal(uint64(40), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(40), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)

		// Validate hardlink behavior
		func() {
			f, err := os.OpenFile(filepath.Join(l.dirs[0], "bar1.f"), os.O_WRONLY, 0666)
			assert.NoError(err)
			defer f.Close()
			_, err = f.Seek(0, io.SeekStart)
			assert.NoError(err)
			assert.NoError(f.Truncate(0))
			_, err = f.WriteString("hello world")
			assert.NoError(err)
			l.content["bar1"] = "hello world"
			l.content["bar2"] = "hello world"
			l.content["bar3"] = "hello world"
		}()
		validate(l)

		// Phase 2: Copy
		scanner = newScanner()
		scanner.options.SplitLinks = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(20), scanner.totals.Files.count)
		assert.Equal(uint64(100), scanner.totals.Files.size)
		assert.Equal(uint64(5), scanner.totals.Unique.count)
		assert.Equal(uint64(25), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(15), scanner.totals.Dupes.count)
		assert.Equal(uint64(75), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(75), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)

		// Validate hardlink split
		for i := 0; i <= 1; i++ {
			func(i int) {
				f, err := os.OpenFile(filepath.Join(l.dirs[i], "bar1.f"), os.O_WRONLY, 0666)
				assert.NoError(err)
				defer f.Close()
				_, err = f.Seek(0, io.SeekStart)
				assert.NoError(err)
				assert.NoError(f.Truncate(0))
				_, err = f.WriteString("goodbye world")
				assert.NoError(err)
			}(i)
		}
		l.content["bar1"] = "goodbye world"
		validate(l)
	})
}

func TestScanner_Clone(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		// Phase 1: Hardlink
		scanner := newScanner()
		scanner.options.Clone = true
		scanner.options.Recursive = true

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(20), scanner.totals.Files.count)
		assert.Equal(uint64(58), scanner.totals.Files.size)
		assert.Equal(uint64(5), scanner.totals.Unique.count)
		assert.Equal(uint64(18), scanner.totals.Unique.size)
		assert.Equal(uint64(0), scanner.totals.Links.count)
		assert.Equal(uint64(0), scanner.totals.Links.size)
		assert.Equal(uint64(15), scanner.totals.Cloned.count)
		assert.Equal(uint64(40), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(15), scanner.totals.Processed.count)
		assert.Equal(uint64(40), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		validate(l)
	})
}

func TestScanner_SkipHeader(t *testing.T) {
	assert := require.New(t)
	setupTest(assert, func(l *testLayout, validate func(*testLayout)) {
		scanner := newScanner()
		scanner.options.MakeLinks = true
		scanner.options.Recursive = true
		scanner.options.SkipHeader = 3

		assert.NoError(scanner.Scan())
		fmt.Println(scanner.totals.PrettyFormat(scanner.options.Verb()))
		assert.Equal(uint64(14), scanner.totals.Files.count)
		assert.Equal(uint64(58), scanner.totals.Files.size)
		assert.Equal(uint64(2), scanner.totals.Unique.count)
		assert.Equal(uint64(9), scanner.totals.Unique.size)
		assert.Equal(uint64(12), scanner.totals.Links.count)
		assert.Equal(uint64(49), scanner.totals.Links.size)
		assert.Equal(uint64(0), scanner.totals.Cloned.count)
		assert.Equal(uint64(0), scanner.totals.Cloned.size)
		assert.Equal(uint64(0), scanner.totals.Dupes.count)
		assert.Equal(uint64(0), scanner.totals.Dupes.size)
		assert.Equal(uint64(12), scanner.totals.Processed.count)
		assert.Equal(uint64(49), scanner.totals.Processed.size)
		assert.Equal(uint64(0), scanner.totals.Errors.count)
		assert.Equal(uint64(0), scanner.totals.Errors.size)
		l.content["foo1"] = l.content["bar1"]
		l.content["foo2"] = l.content["bar1"]
		l.content["foo3"] = l.content["bar1"]
		l.different[1] = l.different[0]
		validate(l)
	})
}

type testLayout struct {
	dirs []string

	// Duplicated per dirList[n]
	content map[string]string

	// dirList[n]/different == different[n]
	different []string
}

func setupTest(assert *require.Assertions, f func(l *testLayout, validate func(*testLayout))) {
	dir, err := ioutil.TempDir("", "fdftest")
	assert.NoError(err)
	defer os.RemoveAll(dir)
	assert.NoError(os.Chdir(dir))

	l := &testLayout{
		dirs: []string{
			"./a",
			"./b",
		},
		content: map[string]string{
			"bar1":   "bar\n",
			"bar2":   "bar\n",
			"bar3":   "bar\n",
			"foo1":   "foo\n",
			"foo2":   "foo\n",
			"foo3":   "foo\n",
			"empty1": "",
			"empty2": "",
			"empty3": "",
		},
		different: []string{
			"fizz\n",
			"buzz\n",
		},
	}

	for i, d := range l.dirs {
		assert.NoError(os.Mkdir(d, 0777))
		for f, c := range l.content {
			assert.NoError(ioutil.WriteFile(filepath.Join(d, fmt.Sprintf("%s.f", f)), []byte(c), 0666))
		}
		assert.NoError(ioutil.WriteFile(filepath.Join(d, "different.f"), []byte(l.different[i]), 0666))
	}

	f(l, func(l *testLayout) {
		g, err := filepath.Glob("./**/*.f")
		assert.NoError(err)
		assert.Len(g, (len(l.dirs)*len(l.content))+len(l.different))

		for i, d := range l.dirs {
			for f, c := range l.content {
				b, err := ioutil.ReadFile(filepath.Join(d, fmt.Sprintf("%s.f", f)))
				assert.NoError(err)
				assert.Equalf(c, string(b), "%s", f)
			}
			b, err := ioutil.ReadFile(filepath.Join(d, "different.f"))
			assert.NoError(err)
			assert.Equal(l.different[i], string(b))
		}
	})
}
