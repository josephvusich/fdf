package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/josephvusich/go-getopt"
	"github.com/josephvusich/go-matchers"
	"github.com/josephvusich/go-matchers/glob"
)

type verb int

const (
	VerbNone verb = iota
	VerbClone
	VerbSplitLinks
	VerbMakeLinks
	VerbDelete
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

	Recursive bool

	minSize    int64
	SkipHeader int64

	IgnoreExistingLinks bool
	Quiet               bool
	Verbose             bool
	DryRun              bool
	Help                bool
}

var matchFunc = regexp.MustCompile(`^([a-z]+)(?:\[([^\]]+)])?$`)

func (o *options) parseRange(rangePattern string, cmpFlag matchFlag, cmpFunc func(r *fileRecord) string) error {
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
		case "copyname":
			o.MatchMode |= matchCopyName
		case "size":
			o.MatchMode |= matchSize
		default:
			return fmt.Errorf("unknown matcher: %s", m)
		}
	}
	if o.MatchMode&matchCopyName != 0 {
		if o.MatchMode&matchName != 0 {
			return errors.New("cannot specify both name and copyname for --match")
		}
		if o.MatchMode&matchSize == 0 {
			return errors.New("--match copyname also requires either size or content")
		}
	}
	if o.MatchMode == matchNothing {
		return errors.New("must specify at least one non-partial matcher")
	}
	if v == VerbSplitLinks {
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

	o.Protect.DefaultInclude = false
	protect, unprotect := o.Protect.FlagValues(func(pattern string) (matchers.Matcher, error) {
		abs, err := filepath.Abs(pattern)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve \"%s\": %w", pattern, err)
		}
		return glob.NewMatcher(abs)
	})
	fs.BoolVar(&o.clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	fs.BoolVar(&o.splitLinks, "copy", false, "(verb) split existing hardlinks via copy\nmutually exclusive with --ignore-hardlinks")
	fs.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	fs.BoolVar(&o.makeLinks, "link", false, "(verb) hardlink duplicate files")
	fs.BoolVar(&o.deleteDupes, "delete", false, "(verb) delete duplicate files")
	fs.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	fs.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "ignore existing hardlinks\nmutually exclusive with --copy")
	fs.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	fs.BoolVar(&o.Verbose, "verbose", false, "display additional details regarding protected paths")
	fs.BoolVar(&o.Help, "help", false, "show this help screen and exit")
	fs.Int64Var(&o.minSize, "minimum-size", 1, "skip files smaller than `BYTES`")
	fs.Int64Var(&o.SkipHeader, "skip-header", 0, "skip `LENGTH` bytes at the beginning of each file when comparing\nimplies --minimum-size LENGTH+1")
	fs.Var(protect, "protect", "prevent files matching glob `PATTERN` from being modified or deleted\nmay appear more than once to support multiple patterns\nrules are applied in the order specified")
	fs.Var(protect, "preserve", "(deprecated) alias for --protect `PATTERN`")
	fs.Var(unprotect, "unprotect", "remove files added by --protect\nmay appear more than once\nrules are applied in the order specified")
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
		"  parent (case insensitive name of immediate parent directory)\n"+
		"    range notation supported: see 'name' for examples\n"+
		"  size\n"+
		"  content (default, also implies size)\n"+
		"specify multiple fields using '+', e.g.: name+content")

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
		o.Help = true
	}

	if err = o.parseMatchSpec(*matchSpec, o.Verb()); err != nil {
		fmt.Println("Invalid --match parameter:", err)
		o.Help = true
	}

	if o.Verb() == VerbSplitLinks && o.IgnoreExistingLinks {
		fmt.Println("Invalid flag combination: --copy and --ignore-hardlinks are mutually exclusive")
		o.Help = true
	}

	if o.Help {
		fmt.Println("Latest version can be found at https://github.com/josephvusich/fdf")
		fs.Usage()
		os.Exit(1)
	}

	return fs.Args()
}

func (o *options) globPattern() string {
	if o.Recursive {
		return "./**/*"
	}
	return "./*"
}
