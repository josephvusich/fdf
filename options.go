package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/josephvusich/fdf/matchers"
	"github.com/josephvusich/go-getopt"
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
	Protect   matchers.GlobSet

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

// TODO add mod time
// does not modify options on error
func (o *options) parseMatchSpec(matchSpec string, v verb) (err error) {
	var f matchFlag
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
			f |= matchContent
		case "name":
			if r[2] != "" {
				cmp, err := newComparer(r[2], func(r *fileRecord) string { return r.FoldedName })
				if err != nil {
					return err
				}
				o.Comparers = append(o.Comparers, cmp)
			} else {
				f |= matchName
			}
		case "parent":
			if r[2] != "" {
				cmp, err := newComparer(r[2], func(r *fileRecord) string { return r.FoldedParent })
				if err != nil {
					return err
				}
				o.Comparers = append(o.Comparers, cmp)
			} else {
				f |= matchParent
			}
		case "copyname":
			f |= matchCopyName
		case "size":
			f |= matchSize
		default:
			return fmt.Errorf("unknown matcher: %s", m)
		}
	}
	if f&matchCopyName != 0 {
		if f&matchName != 0 {
			return errors.New("cannot specify both name and copyname for --match")
		}
		if f&matchSize == 0 {
			return errors.New("--match copyname also requires either size or content")
		}
	}
	if f == 0 {
		return errors.New("must specify at least one non-partial matcher")
	}
	if v == VerbSplitLinks {
		f |= matchHardlink
	}

	o.MatchMode = f
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

func (o *options) ParseArgs() (dirs []string) {
	o.Protect.DefaultInclude = false
	flag.BoolVar(&o.clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	flag.BoolVar(&o.splitLinks, "copy", false, "(verb) split existing hardlinks via copy\nmutually exclusive with --ignore-hardlinks")
	flag.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	flag.BoolVar(&o.makeLinks, "link", false, "(verb) hardlink duplicate files")
	flag.BoolVar(&o.deleteDupes, "delete", false, "(verb) delete duplicate files")
	flag.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	flag.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "ignore existing hardlinks\nmutually exclusive with --copy")
	flag.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	flag.BoolVar(&o.Verbose, "verbose", false, "display additional details regarding protected paths")
	flag.BoolVar(&o.Help, "help", false, "show this help screen and exit")
	flag.Int64Var(&o.minSize, "minimum-size", 1, "skip files smaller than `BYTES`")
	flag.Int64Var(&o.SkipHeader, "skip-header", 0, "skip `LENGTH` bytes at the beginning of each file when comparing\nimplies --minimum-size LENGTH+1")
	flag.Var(o.Protect.FlagValue(true), "protect", "prevent files matching glob `PATTERN` from being modified or deleted\nmay appear more than once to support multiple patterns\nrules are applied in the order specified")
	flag.Var(o.Protect.FlagValue(true), "preserve", "(deprecated) alias for --protect `PATTERN`")
	flag.Var(o.Protect.FlagValue(false), "unprotect", "remove files added by --protect\nmay appear more than once\nrules are applied in the order specified")
	matchSpec := flag.String("match", "", "Evaluate `FIELDS` to determine file equality, where valid fields are:\n"+
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

	getopt.Alias("a", "clone")
	getopt.Alias("c", "copy")
	getopt.Alias("r", "recursive")
	getopt.Alias("l", "link")
	getopt.Alias("d", "delete")
	getopt.Alias("q", "quiet")
	getopt.Alias("v", "verbose")
	getopt.Alias("t", "dry-run")
	getopt.Alias("h", "ignore-hardlinks")
	getopt.Alias("z", "minimum-size")
	getopt.Alias("m", "match")
	getopt.Alias("n", "skip-header")
	getopt.Alias("p", "protect")

	if err := getopt.CommandLine.Parse(os.Args[1:]); err != nil {
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
		flag.Usage()
		os.Exit(1)
	}

	return getopt.CommandLine.Args()
}

func (o *options) globPattern() string {
	if o.Recursive {
		return "./**/*"
	}
	return "./*"
}
