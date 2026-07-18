package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtectArgs(t *testing.T) {
	assert := require.New(t)

	args := []string{`fdf`, `-r`, `--protect`, `./a/**/*`, `--unprotect`, `a/**/bar`, `a`, `./b`}
	var o options
	dirs := o.ParseArgs(args)

	assert.True(o.Recursive)

	assert.Len(dirs, 2)
	assert.Equal("a", dirs[0])
	assert.Equal("./b", dirs[1])

	cases := map[string]bool{
		"./a/foo":     true,
		"./a/foo/bar": false,
		"./b/foo":     false,
		"./b/foo/bar": false,
	}

	for in, out := range cases {
		abs, err := filepath.Abs(in)
		assert.NoError(err)
		assert.Equal(out, o.Protect.Includes(abs), "expected Includes=%t for '%s'", out, in)
	}
}

func TestIsPathAncestor(t *testing.T) {
	assert := require.New(t)

	base := t.TempDir()
	root := filepath.VolumeName(base) + string(filepath.Separator)

	cases := []struct {
		ancestor   string
		descendant string
		expect     bool
	}{
		{base, base, false},
		{base, filepath.Join(base, "a"), true},
		{base, filepath.Join(base, "a", "b", "c"), true},
		{filepath.Join(base, "a"), filepath.Join(base, "ab"), false},
		{filepath.Join(base, "a", "b"), filepath.Join(base, "a"), false},
		{root, base, true},
	}

	for _, c := range cases {
		assert.Equal(c.expect, isPathAncestor(c.ancestor, c.descendant), "isPathAncestor(%q, %q)", c.ancestor, c.descendant)
	}
}

func TestResolveScanDirs(t *testing.T) {
	wd := t.TempDir()
	outside := t.TempDir()

	df := func(name, value string, seed bool) dirFlagArg {
		return dirFlagArg{flagName: name, value: value, seedScan: seed}
	}

	tests := []struct {
		name        string
		positionals []string
		dirFlags    []dirFlagArg
		recursive   bool
		expectDirs  []string
		expectErrs  []string
	}{
		{
			name:       "auto-fill in flag order",
			dirFlags:   []dirFlagArg{df("protect-dir", "a", true), df("if-kept-dir", "b", true)},
			expectDirs: []string{"a", "b"},
		},
		{
			name:       "dedup by absolute path, first form wins",
			dirFlags:   []dirFlagArg{df("protect-dir", "a", true), df("unprotect-dir", "./a", true), df("if-kept-dir", filepath.Join(wd, "a"), true)},
			expectDirs: []string{"a"},
		},
		{
			name:     "exclude-dir never seeds",
			dirFlags: []dirFlagArg{df("exclude-dir", "a", false)},
		},
		{
			name:       "exclude-dir validated against cwd fallback",
			dirFlags:   []dirFlagArg{df("exclude-dir", outside, false)},
			expectErrs: []string{"--exclude-dir", outside},
		},
		{
			name: "cwd fallback with no flags",
		},
		{
			name:       "recursive prunes nested auto-filled roots",
			dirFlags:   []dirFlagArg{df("protect-dir", "a", true), df("if-kept-dir", filepath.Join("a", "sub"), true)},
			recursive:  true,
			expectDirs: []string{"a"},
		},
		{
			name:       "recursive prune with ancestor later in flag order",
			dirFlags:   []dirFlagArg{df("if-kept-dir", filepath.Join("a", "sub"), true), df("protect-dir", "a", true)},
			recursive:  true,
			expectDirs: []string{"a"},
		},
		{
			name:       "no prune when non-recursive",
			dirFlags:   []dirFlagArg{df("if-kept-dir", filepath.Join("a", "sub"), true), df("protect-dir", "a", true)},
			expectDirs: []string{filepath.Join("a", "sub"), "a"},
		},
		{
			name:        "positionals suppress auto-fill",
			positionals: []string{"a"},
			dirFlags:    []dirFlagArg{df("protect-dir", filepath.Join("a", "x"), true)},
			expectDirs:  []string{"a"},
		},
		{
			name:        "validation passes on equal",
			positionals: []string{"a"},
			dirFlags:    []dirFlagArg{df("if-kept-dir", "a", true)},
			expectDirs:  []string{"a"},
		},
		{
			name:        "validation passes on descendant flag",
			positionals: []string{"a"},
			dirFlags:    []dirFlagArg{df("if-kept-dir", filepath.Join("a", "b", "c"), true)},
			expectDirs:  []string{"a"},
		},
		{
			name:        "validation passes on ancestor flag",
			positionals: []string{filepath.Join("a", "b")},
			dirFlags:    []dirFlagArg{df("protect-dir", "a", true)},
			expectDirs:  []string{filepath.Join("a", "b")},
		},
		{
			name:        "validation fails on disjoint flag",
			positionals: []string{"a"},
			dirFlags:    []dirFlagArg{df("if-kept-dir", "b", true)},
			expectErrs:  []string{`--if-kept-dir "b"`, "scanned: a"},
		},
		{
			name:        "multiple violations reported together",
			positionals: []string{"a"},
			dirFlags:    []dirFlagArg{df("if-kept-dir", "b", true), df("protect-dir", "c", true)},
			expectErrs:  []string{"--if-kept-dir", "--protect-dir"},
		},
		{
			name:        "relative positional matches absolute flag",
			positionals: []string{"sub"},
			dirFlags:    []dirFlagArg{df("protect-dir", filepath.Join(wd, "sub"), true)},
			expectDirs:  []string{"sub"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := require.New(t)
			dirs, err := resolveScanDirs(tc.positionals, tc.dirFlags, wd, tc.recursive)
			if len(tc.expectErrs) > 0 {
				assert.Error(err)
				for _, want := range tc.expectErrs {
					assert.Contains(err.Error(), want)
				}
				return
			}
			assert.NoError(err)
			assert.Equal(tc.expectDirs, dirs)
		})
	}
}

func TestParseArgsAutoFillDirs(t *testing.T) {
	setup := func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(t, os.MkdirAll(filepath.Join("a", "sub"), 0o755))
		require.NoError(t, os.MkdirAll("b", 0o755))
	}

	t.Run("auto-fill from -dir flags", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "-r", "--unprotect-dir", "a", "--if-kept-dir", "b"})

		assert.True(o.Recursive)
		assert.Equal([]string{"a", "b"}, dirs)

		absA, err := filepath.Abs(filepath.Join("a", "foo"))
		assert.NoError(err)
		absB, err := filepath.Abs(filepath.Join("b", "foo"))
		assert.NoError(err)
		assert.False(o.Protect.Includes(absA))
		assert.True(o.Protect.Includes(absB))
	})

	t.Run("dedup across flags", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "--protect-dir", "a", "--unprotect-dir", "./a"})
		assert.Equal([]string{"a"}, dirs)
	})

	t.Run("recursive prunes nested roots", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "-r", "--protect-dir", "a", "--if-kept-dir", filepath.Join("a", "sub")})
		assert.Equal([]string{"a"}, dirs)
	})

	t.Run("non-recursive keeps nested roots", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "--protect-dir", "a", "--if-kept-dir", filepath.Join("a", "sub")})
		assert.Equal([]string{"a", filepath.Join("a", "sub")}, dirs)
	})

	t.Run("exclude-dir does not seed", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "--exclude-dir", "a"})
		assert.Empty(dirs)
	})

	t.Run("positionals win verbatim", func(t *testing.T) {
		assert := require.New(t)
		setup(t)

		var o options
		dirs := o.ParseArgs([]string{"fdf", "--protect-dir", "a", "a", "b"})
		assert.Equal([]string{"a", "b"}, dirs)
	})
}

func TestOptions_ParseArgs(t *testing.T) {
	assert := require.New(t)

	mockFileRecord := &fileRecord{
		FilePath:     "Path/To/File",
		FoldedName:   "FoldedName",
		FoldedParent: "FoldedParent",
	}

	tests := []struct {
		spec      string
		verb      verb
		expect    matchFlag
		comparers []string
	}{
		{"content", VerbMakeLinks, matchContent | matchSize, nil},
		{"size", VerbMakeLinks, matchSize, nil},
		{"name", VerbMakeLinks, matchName, nil},

		{"content+name", VerbMakeLinks, matchContent | matchName, nil},
		{"size+name", VerbMakeLinks, matchSize | matchName, nil},
		{"name+content", VerbMakeLinks, matchName | matchContent, nil},
		{"name[0:3]+content", VerbMakeLinks, matchContent, []string{"FoldedName"}},
		{"parent[0:3]+content", VerbMakeLinks, matchContent, []string{"FoldedParent"}},
		{"content+path", VerbMakeLinks, matchContent | matchParent | matchPathSuffix, []string{filepath.Join("Path", "To")}},

		{"content+name", VerbSplitLinks, matchContent | matchName | matchHardlink, nil},
		{"size+name", VerbSplitLinks, matchSize | matchName | matchHardlink, nil},
		{"name+content", VerbSplitLinks, matchName | matchContent | matchHardlink, nil},
	}

	for _, t := range tests {
		var o options
		err := o.parseMatchSpec(t.spec, t.verb)
		assert.NoError(err)
		assert.Equal(t.expect, o.MatchMode, t.spec)
		assert.Len(o.Comparers, len(t.comparers))
		for i, c := range o.Comparers {
			assert.Equal(t.comparers[i], c.HashFunc(mockFileRecord))
		}
	}
}
