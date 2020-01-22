package main

import (
	"flag"
	"fmt"
	"os"

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
	Clone       bool
	SplitLinks  bool
	MakeLinks   bool
	DeleteDupes bool

	Recursive bool
	MinSize   int64

	IgnoreExistingLinks bool
	Quiet               bool
	DryRun              bool
	Help                bool
}

func (o *options) Verb() verb {
	switch true {
	case o.MakeLinks:
		return VerbMakeLinks
	case o.Clone:
		return VerbClone
	case o.SplitLinks:
		return VerbSplitLinks
	case o.DeleteDupes:
		return VerbDelete
	}
	return VerbNone
}

func (o *options) ParseArgs() {
	flag.BoolVar(&o.Clone, "clone", false, "(verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)")
	flag.BoolVar(&o.SplitLinks, "copy", false, "(verb) split existing links via copy")
	flag.BoolVar(&o.Recursive, "recursive", false, "traverse subdirectories")
	flag.BoolVar(&o.MakeLinks, "link", false, "(verb) hardlink duplicate files")
	flag.BoolVar(&o.DeleteDupes, "delete", false, "(verb) delete duplicate files")
	flag.BoolVar(&o.DryRun, "dry-run", false, "don't actually do anything, just show what would be done")
	flag.BoolVar(&o.IgnoreExistingLinks, "ignore-hardlinks", false, "don't show existing hardlinks")
	flag.BoolVar(&o.Quiet, "quiet", false, "don't display current filename during scanning")
	flag.BoolVar(&o.Help, "help", false, "show this help screen and exit")
	flag.Int64Var(&o.MinSize, "minimum-size", 1, "ignore files smaller than <int> bytes")

	getopt.Alias("a", "clone")
	getopt.Alias("c", "copy")
	getopt.Alias("r", "recursive")
	getopt.Alias("l", "link")
	getopt.Alias("d", "delete")
	getopt.Alias("q", "quiet")
	getopt.Alias("t", "dry-run")
	getopt.Alias("h", "ignore-hardlinks")
	getopt.Alias("z", "minimum-size")

	if err := getopt.CommandLine.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
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
