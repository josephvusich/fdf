package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/josephvusich/go-getopt"
	"github.com/josephvusich/go-matchers"
	"github.com/josephvusich/go-matchers/glob"
)

const fileBufferSize = 0x100000 // 1MB

type verb int

const (
	VerbNone verb = iota
	VerbClone
	VerbSplitLinks
	VerbMakeLinks
	VerbDelete
)

const (
	TimestampIgnore = "ignore"
	TimestampNewer  = "prefer-newer"
	TimestampOlder  = "prefer-older"
)

var (
	validTimestampFlags = map[string]struct{}{
		TimestampIgnore: {},
		TimestampNewer:  {},
		TimestampOlder:  {},
	}
)

func (v verb) PastTense() string {
	switch v {
	case VerbNone:
		return "skipped"
	case VerbClone:
		return "cloned"
	case VerbSplitLinks:
		return "copied"
	case VerbMakeLinks:
		return "hardlinked"
	case VerbDelete:
		return "deleted"
	}
	return fmt.Sprintf("unknown verb value %d", v)
}

type options struct {
	clone       bool
	splitLinks  bool
	makeLinks   bool
	deleteDupes bool

	MatchMode matchFlag

	Comparers []comparer
	Protect   matchers.RuleSet
	Exclude   matchers.RuleSet
	MustKeep  matchers.RuleSet

	TimestampBehavior string

	Recursive bool

	minSize    int64
	SkipHeader int64
	SkipFooter int64

	IgnoreExistingLinks bool
	CopyUnlinked        bool
	Quiet               bool
	Verbose             bool
	DryRun              bool

	JsonReport string
}

func keysToStringList(m map[string]struct{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// OpenFile returns a reader that follows options.SkipHeader
func (o *options) OpenFile(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if o.SkipHeader > 0 {
		if _, err = f.Seek(o.SkipHeader, io.SeekStart); err != nil {
			f.Close()
			return nil, err
		}
	}

	if o.SkipFooter == 0 {
		return f, nil
	}

	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return newLimitReadCloser(f, st.Size()-o.SkipFooter), nil
}

type limitReadCloser struct {
	io.Reader
	io.Closer
}

func newLimitReadCloser(f *os.File, n int64) *limitReadCloser {
	return &limitReadCloser{
		Reader: bufio.NewReaderSize(io.LimitReader(f, n), fileBufferSize),
		Closer: f,
	}
}

var matchFunc = regexp.MustCompile(`^([a-z]+)(?:\[([^\]]+)])?$`)

func (o *options) parseRange(rangePattern string, cmpFlag matchFlag, cmpFunc func(r *fileRecord) string) error {
	// non-indexed fields must use range matchers
	if cmpFlag == matchNothing && rangePattern == "" {
		rangePattern = ":"
	}

	if rangePattern != "" {
		cmp, err := newComparer(rangePattern, cmpFunc)
		if err != nil {
			return err
		}
		o.Comparers = append(o.Comparers, cmp)
	} else {
		o.MatchMode |= cmpFlag
	}
	return nil
}

// TODO add mod time
// does not modify options on error
func (o *options) parseMatchSpec(matchSpec string, v verb) (err error) {
	o.MatchMode = matchNothing
	if matchSpec == "" {
		matchSpec = "content"
	}
	modes := strings.Split(strings.ToLower(matchSpec), "+")
	for _, m := range modes {
		r := matchFunc.FindStringSubmatch(m)
		if r == nil {
			return fmt.Errorf("invalid field: %s", m)
		}

		switch r[1] {
		case "content":
			o.MatchMode |= matchContent
		case "name":
			if err := o.parseRange(r[2], matchName, func(r *fileRecord) string { return r.FoldedName }); err != nil {
				return err
			}
		case "parent":
			if err := o.parseRange(r[2], matchParent, func(r *fileRecord) string { return r.FoldedParent }); err != nil {
				return err
			}
		case "relpath":
			o.MatchMode |= matchPathSuffix
		case "path":
			if err := o.parseRange(r[2], matchNothing, func(r *fileRecord) string { return filepath.Dir(r.FilePath) }); err != nil {
				return err
			}
			// rely on path suffix match to narrow down possible matches pre-comparer
			o.MatchMode |= matchPathSuffix
		case "copyname":
			o.MatchMode |= matchCopyName
		case "namesuffix":
			o.MatchMode |= matchNameSuffix
		case "nameprefix":
			o.MatchMode |= matchNamePrefix
		case "size":
			o.MatchMode |= matchSize
		default:
			return fmt.Errorf("unknown matcher: %s", m)
		}
	}
	if o.MatchMode&matchCopyName != 0 || o.MatchMode&matchNameSuffix != 0 {
		if o.MatchMode&matchCopyName != 0 && o.MatchMode&matchNameSuffix != 0 {
			return errors.New("cannot specify both copyname and namesuffix for --match")
		}
		if o.MatchMode&matchName != 0 {
			return errors.New("cannot specify both name and copyname/namesuffix for --match")
		}
		if o.MatchMode&matchSize == 0 {
			return errors.New("--match copyname/namesuffix also require either size or content")
		}
	}
	if o.MatchMode == matchNothing {
		return errors.New("must specify at least one non-partial matcher")
	}
	if v == VerbSplitLinks && !o.CopyUnlinked {
		o.MatchMode |= matchHardlink
	}

	return nil
}

func (o *options) Verb() verb {
	switch true {
	case o.makeLinks:
		return VerbMakeLinks
	case o.clone:
		return VerbClone
	case o.splitLinks:
		return VerbSplitLinks
	case o.deleteDupes:
		return VerbDelete
	}
	return VerbNone
}

func (o *options) MinSize() int64 {
	if o.SkipHeader > 0 && o.SkipHeader+1 > o.minSize {
		return o.SkipHeader + 1
	}
	return o.minSize
}

func (o *options) ParseArgs(args []string) (dirs []string) {
	fs := getopt.NewFlagSet(args[0], flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr,
			"usage: fdf [--clone | --copy | --delete | --link] [-hqrtv]\n"+
				"        [-m FIELDS] [-z BYTES] [-n LENGTH]\n"+
				"        [--protect PATTERN] [--unprotect PATTERN] [directory ...]\n\n")
		fs.PrintDefaults()
	}
	badOptions := false

	o.Protect.DefaultInclude = false
	protect, unprotect := o.Protect.FlagValues(globMatcher)
	protectDir, unprotectDir := o.Protect.FlagValues(globMatcherFromDir)

	o.Exclude.DefaultInclude = false
	exclude, include := o.Exclude.FlagValues(globMatcher)
	excludeDir, includeDir := o.Exclude.FlagValues(globMatcherFromDir)

	o.MustKeep.DefaultInclude = true
	mustKeep, mustNotKeep := o.MustKeep.FlagValues(globMatcher)
	mustKeepDir, mustNotKeepDir := o.MustKeep.FlagValues(globMatcherFromDir)

	fs.BoolVar(&o.clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	fs.BoolVar(&o.splitLinks, "copy", false, "(verb) split existing hardlinks via copy\nmutually exclusive with --ignore-hardlinks")
	fs.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	fs.BoolVar(&o.makeLinks, "link", false, "(verb) hardlink duplicate files")
	fs.BoolVar(&o.deleteDupes, "delete", false, "(verb) delete duplicate files")
	fs.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	fs.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "ignore existing hardlinks\nmutually exclusive with --copy")
	fs.BoolVar(&o.CopyUnlinked, "copy-unlinked", false, "always copy over matching files even if not hardlinked")
	fs.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	fs.BoolVar(&o.Verbose, "verbose", false, "display additional details regarding protected paths")
	helpFlag := fs.Bool("help", false, "show this help screen and exit")
	fs.Int64Var(&o.minSize, "minimum-size", 1, "skip files smaller than `BYTES`, must be greater than the sum of --skip-header and --skip-footer")
	fs.Int64Var(&o.SkipHeader, "skip-header", 0, "skip `LENGTH` bytes at the beginning of each file when comparing")
	fs.Int64Var(&o.SkipFooter, "skip-footer", 0, "skip `LENGTH` bytes at the end of each file when comparing")
	fs.Var(exclude, "exclude", "exclude files matching `GLOB` from scanning")
	fs.Var(excludeDir, "exclude-dir", "exclude `DIR` from scanning, throws error if DIR does not exist")
	fs.Var(include, "include", "include `GLOB`, opposite of --exclude")
	fs.Var(includeDir, "include-dir", "include `DIR`, throws error if DIR does not exist")
	fs.Var(protect, "protect", "prevent files matching glob `PATTERN` from being modified or deleted\n"+
		"may appear more than once to support multiple patterns\n"+
		"rules are applied in the order specified")
	fs.Var(protect, "preserve", "(deprecated) alias for --protect `PATTERN`")
	fs.Var(protectDir, "protect-dir", "similar to --protect 'DIR/**/*', but throws error if `DIR` does not exist")
	fs.Var(unprotect, "unprotect", "remove files added by --protect\nmay appear more than once\nrules are applied in the order specified")
	fs.Var(unprotectDir, "unprotect-dir", "similar to --unprotect 'DIR/**/*', but throws error if `DIR` does not exist")
	fs.Var(mustKeep, "if-kept", "only remove files if the 'kept' file matches the provided `GLOB`")
	fs.Var(mustNotKeep, "if-not-kept", "only remove files if the 'kept' file does NOT match the provided `GLOB`")
	fs.Var(mustKeepDir, "if-kept-dir", "only remove files if the 'kept' file is a descendant of `DIR`")
	fs.Var(mustNotKeepDir, "if-not-kept-dir", "only remove files if the 'kept' file is NOT a descendant of `DIR`")
	fs.StringVar(&o.TimestampBehavior, "timestamps", TimestampOlder, "`MODE` must be one of "+keysToStringList(validTimestampFlags))
	matchSpec := fs.String("match", "", "Evaluate `FIELDS` to determine file equality, where valid fields are:\n"+
		"  name (case insensitive)\n"+
		"    range notation supported: name[offset:len,offset:len,...]\n"+
		"      name[0:-1] whole string\n"+
		"      name[0:-2] all except last character\n"+
		"      name[1:2]  second and third characters\n"+
		"      name[-1:1] last character\n"+
		"      name[-3:3] last 3 characters\n"+
		"  copyname (case insensitive)\n"+
		"    'foo.bar' == 'foo (1).bar' == 'Copy of foo.bar', also requires +size or +content\n"+
		"  namesuffix (case insensitive)\n"+
		"    one filename must end with the other, e.g.: 'foo-1.bar' and '1.bar'\n"+
		"  nameprefix (case insensitive)\n"+
		"    one filename must begin with the other, e.g., 'foo-1.bar' and 'foo.bar'\n"+
		"  parent (case insensitive name of immediate parent directory)\n"+
		"    range notation supported: see 'name' for examples\n"+
		"  path\n"+
		"    match parent directory path\n"+
		"  relpath\n"+
		"    match parent directory path relative to input dir(s)\n"+
		"  size\n"+
		"  content (default, also implies size)\n"+
		"specify multiple fields using '+', e.g.: name+content")
	allowNoContent := fs.Bool("ignore-content", false, "allow --match without 'content'")
	fs.StringVar(&o.JsonReport, "json-report", "", "on completion, dump JSON match data to `FILE`")

	fs.Alias("a", "clone")
	fs.Alias("c", "copy")
	fs.Alias("r", "recursive")
	fs.Alias("l", "link")
	fs.Alias("d", "delete")
	fs.Alias("q", "quiet")
	fs.Alias("v", "verbose")
	fs.Alias("t", "dry-run")
	fs.Alias("h", "ignore-hardlinks")
	fs.Alias("z", "minimum-size")
	fs.Alias("m", "match")
	fs.Alias("n", "skip-header")
	fs.Alias("p", "protect")

	if err := fs.Parse(args[1:]); err != nil {
		os.Exit(1)
	}

	var err error
	if o.Quiet && o.Verbose {
		fmt.Println("Invalid flag combination: --quiet and --verbose are mutually exclusive")
		badOptions = true
	}

	if o.CopyUnlinked && !o.splitLinks {
		fmt.Println("--copy-unlinked is only valid with --copy")
		badOptions = true
	}

	if _, ok := validTimestampFlags[o.TimestampBehavior]; !ok {
		fmt.Println("--timestamps must be one of:", keysToStringList(validTimestampFlags))
		badOptions = true
	}

	if err = o.parseMatchSpec(*matchSpec, o.Verb()); err != nil {
		fmt.Println("Invalid --match parameter:", err)
		badOptions = true
	}

	if o.MatchMode&matchContent != matchContent && !*allowNoContent && (o.Verb() != VerbNone && !o.DryRun) {
		fmt.Println("Must specify --ignore-content to use --match without 'content'")
		badOptions = true
	} else if o.MatchMode&matchContent == 1 && *allowNoContent {
		fmt.Println("--ignore-content specified, but --match contains 'content'")
		badOptions = true
	} else if o.DryRun && *allowNoContent {
		fmt.Println("--ignore-content is mutually exclusive with --dry-run")
		badOptions = true
	} else if o.Verb() == VerbNone && *allowNoContent {
		fmt.Println("--ignore-content specified without a verb")
		badOptions = true
	}

	if o.Verb() == VerbSplitLinks && o.IgnoreExistingLinks {
		fmt.Println("Invalid flag combination: --copy and --ignore-hardlinks are mutually exclusive")
		badOptions = true
	}

	if *helpFlag {
		fs.Usage()
		os.Exit(0)
	}

	if badOptions {
		os.Exit(1)
	}

	return fs.Args()
}

func globMatcher(pattern string) (matchers.Matcher, error) {
	abs, err := filepath.Abs(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve \"%s\": %w", pattern, err)
	}
	return glob.NewMatcher(abs)
}

func globMatcherFromDir(dir string) (matchers.Matcher, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve \"%s\": %w", dir, err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve \"%s\": %w", dir, err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	return glob.NewMatcher(filepath.Join(abs, "**", "*"))
}

func (o *options) globPattern() string {
	if o.Recursive {
		return "./**/*"
	}
	return "./*"
}
