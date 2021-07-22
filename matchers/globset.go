package matchers

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
)

type Glob struct {
	Pattern string
	Include bool
}

func NewGlob(pattern string, include bool) (r *Glob, err error) {
	abs, err := filepath.Abs(pattern)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve \"%s\": %w", pattern, err)
	}
	if _, err := filepath.Match(abs, "foobar"); err != nil {
		return nil, err
	}
	return &Glob{
		Include: include,
		Pattern: abs,
	}, nil
}

func (g *Glob) Match(path string) bool {
	ok, err := zglob.Match(g.Pattern, path)
	if err != nil {
		panic(err)
	}
	return ok
}

type GlobSet struct {
	DefaultInclude bool
	rules          []*Glob
}

func (s *GlobSet) Add(r *Glob) {
	s.rules = append(s.rules, r)
}

type flagValue struct {
	gs      *GlobSet
	include bool
}

func (f *flagValue) Set(pattern string) error {
	r, err := NewGlob(pattern, f.include)
	if err != nil {
		return err
	}
	f.gs.Add(r)
	return nil
}

func (f *flagValue) String() string {
	if f.gs == nil {
		return ""
	}
	return f.gs.String()
}

// For integration with flag.Var
func (s *GlobSet) FlagValue(include bool) flag.Value {
	return &flagValue{
		gs:      s,
		include: include,
	}
}

func (s *GlobSet) String() string {
	elems := make([]string, 0, len(s.rules))
	for _, r := range s.rules {
		elems = append(elems, r.Pattern)
	}
	return strings.Join(elems, "\n")
}

// A later Glob overrides an earlier one relative to the order added.
// Default (before matching any rules) is the opposite of the first Glob type.
// E.g., a list beginning with Include has an implicit "Exclude all" base rule, and vice versa.
// Empty GlobSet returns DefaultInclude.
func (s *GlobSet) Includes(path string) bool {
	if len(s.rules) == 0 {
		return s.DefaultInclude
	}

	include := !s.rules[0].Include
	for _, r := range s.rules {
		if r.Match(path) {
			include = r.Include
		}
	}
	return include
}
