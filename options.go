package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"rsc.io/getopt"
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

	Recursive bool

	minSize    int64
	SkipHeader int64

	IgnoreExistingLinks bool
	Quiet               bool
	DryRun              bool
	Help                bool
}

// TODO add mod time
func parseMatchSpec(matchSpec string, v verb) (f matchFlag, err error) {
	if matchSpec == "" {
		matchSpec = "content"
	}
	modes := strings.Split(matchSpec, "+")
	for _, m := range modes {
		switch m {
		case "content":
			f |= matchContent
		case "name":
			f |= matchName
		case "size":
			f |= matchSize
		default:
			return f, fmt.Errorf("invalid field: %s", m)
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

func (o *options) ParseArgs() {
	flag.BoolVar(&o.clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	flag.BoolVar(&o.splitLinks, "copy", false, "(verb) split existing links via copy")
	flag.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	flag.BoolVar(&o.makeLinks, "link", false, "(verb) hardlink duplicate files")
	flag.BoolVar(&o.deleteDupes, "delete", false, "(verb) delete duplicate files")
	flag.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	flag.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "don't show existing hardlinks")
	flag.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	flag.BoolVar(&o.Help, "help", false, "show this help screen and exit")
	flag.Int64Var(&o.minSize, "minimum-size", 1, "skip files smaller than <int> bytes")
	flag.Int64Var(&o.SkipHeader, "skip-header", 0, "skip <int> header bytes when comparing content, implies --minimum-size N+1")
	matchSpec := flag.String("match", "content", "< content | name+content | name+size | name | size >")

	getopt.Alias("a", "clone")
	getopt.Alias("c", "copy")
	getopt.Alias("r", "recursive")
	getopt.Alias("l", "link")
	getopt.Alias("d", "delete")
	getopt.Alias("q", "quiet")
	getopt.Alias("t", "dry-run")
	getopt.Alias("h", "ignore-hardlinks")
	getopt.Alias("z", "minimum-size")
	getopt.Alias("m", "match")
	getopt.Alias("n", "skip-header")

	if err := getopt.CommandLine.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	var err error
	if o.MatchMode, err = parseMatchSpec(*matchSpec, o.Verb()); err != nil {
		fmt.Println("Invalid --match parameter:", err)
		o.Help = true
	}

	if o.Help {
		fmt.Println("Latest version can be found at https://github.com/josephvusich/fdf")
		flag.Usage()
		os.Exit(1)
	}
}

func (o *options) globPattern() string {
	if o.Recursive {
		return "./**/*"
	}
	return "./*"
}
