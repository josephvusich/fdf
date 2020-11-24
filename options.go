package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josephvusich/go-getopt"
	"github.com/mattn/go-zglob"
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

	Preserve preservePatterns

	Recursive bool

	minSize    int64
	SkipHeader int64

	IgnoreExistingLinks bool
	Quiet               bool
	Verbose             bool
	DryRun              bool
	Help                bool
}

type preservePatterns map[string]struct{}

func (p preservePatterns) Set(str string) error {
	abs, err := filepath.Abs(str)
	if err != nil {
		return fmt.Errorf("unable to resolve \"%s\": %w", str, err)
	}
	p[abs] = struct{}{}
	return nil
}

func (p preservePatterns) String() string {
	if p != nil {
		elems := make([]string, 0, len(p))
		for x := range p {
			elems = append(elems, x)
		}
		return strings.Join(elems, "\n")
	}
	return ""
}

func (p preservePatterns) Validate() error {
	for x := range p {
		if _, err := filepath.Match(x, "foobar"); err != nil {
			return err
		}
	}
	return nil
}

func (p preservePatterns) Match(path string) (pattern string, ok bool) {
	for x := range p {
		ok, err := zglob.Match(x, path)
		if err != nil {
			// Should have been caught in Validate()
			panic(err)
		}
		if ok {
			return x, true
		}
	}
	return "", false
}

// TODO add mod time
func parseMatchSpec(matchSpec string, v verb) (f matchFlag, err error) {
	if matchSpec == "" {
		matchSpec = "content"
	}
	modes := strings.Split(strings.ToLower(matchSpec), "+")
	for _, m := range modes {
		switch m {
		case "content":
			f |= matchContent
		case "name":
			f |= matchName
		case "copyname":
			f |= matchCopyName
		case "size":
			f |= matchSize
		default:
			return f, fmt.Errorf("invalid field: %s", m)
		}
	}
	if f&matchCopyName != 0 {
		if f&matchName != 0 {
			return f, errors.New("cannot specify both name and copyname for --match")
		}
		if f&matchSize == 0 {
			return f, errors.New("--match copyname also requires either size or content")
		}
	}
	if v == VerbSplitLinks {
		f |= matchHardlink
	}
	return
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
	o.Preserve = make(preservePatterns)
	flag.BoolVar(&o.clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	flag.BoolVar(&o.splitLinks, "copy", false, "(verb) split existing hardlinks via copy\nmutually exclusive with --ignore-hardlinks")
	flag.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	flag.BoolVar(&o.makeLinks, "link", false, "(verb) hardlink duplicate files")
	flag.BoolVar(&o.deleteDupes, "delete", false, "(verb) delete duplicate files")
	flag.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	flag.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "ignore existing hardlinks\nmutually exclusive with --copy")
	flag.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	flag.BoolVar(&o.Verbose, "verbose", false, "display additional details regarding preserved paths")
	flag.BoolVar(&o.Help, "help", false, "show this help screen and exit")
	flag.Int64Var(&o.minSize, "minimum-size", 1, "skip files smaller than `BYTES`")
	flag.Int64Var(&o.SkipHeader, "skip-header", 0, "skip `LENGTH` bytes at the beginning of each file when comparing\nimplies --minimum-size LENGTH+1")
	flag.Var(o.Preserve, "preserve", "prevent files matching glob `PATTERN` from being modified or deleted\nmay appear more than once to support multiple patterns")
	matchSpec := flag.String("match", "", "Evaluate `FIELDS` to determine file equality, where valid fields are:\n  name (case insensitive)\n  copyname (e.g., 'foo.bar' == 'foo (1).bar' == 'Copy of foo.bar', must specify +size or +content)\n  size\n  content (default, also implies size)\nspecify multiple fields using '+', e.g.: name+content")

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
	getopt.Alias("p", "preserve")

	if err := getopt.CommandLine.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	var err error
	if o.Quiet && o.Verbose {
		fmt.Println("Invalid flag combination: --quiet and --verbose are mutually exclusive")
		o.Help = true
	}

	if o.MatchMode, err = parseMatchSpec(*matchSpec, o.Verb()); err != nil {
		fmt.Println("Invalid --match parameter:", err)
		o.Help = true
	}

	if err = o.Preserve.Validate(); err != nil {
		fmt.Println("Invalid --preserve pattern:", err)
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
