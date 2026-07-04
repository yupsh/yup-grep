// Command yup-grep is the CLI wrapper around github.com/gloo-foo/cmd-grep.
package main

import (
	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-grep"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name            = "grep"
	flagIgnoreCase  = "ignore-case"
	flagInvertMatch = "invert-match"
	flagLineNumber  = "line-number"
	flagCount       = "count"
	flagWord        = "word-regexp"
	flagLine        = "line-regexp"
	flagExtended    = "extended-regexp"
)

// Error is the package's sentinel error type, so every emitted error path is
// comparable with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingPattern is emitted when no PATTERN operand is supplied.
const ErrMissingPattern Error = "missing PATTERN operand"

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `grep [OPTIONS] PATTERN

Search standard input for PATTERN, writing matching lines to
standard output. PATTERN is a fixed-string match by default.`

// spec declares the grep wrapper: a stdin filter whose first operand is the
// pattern and whose flags configure the match.
var spec = clix.Spec{
	Name:     name,
	Summary:  "search for patterns in files",
	Synopsis: synopsis,
	Build:    build,
	Flags: []urf.Flag{
		&urf.BoolFlag{Name: flagIgnoreCase, Aliases: []string{"i"}, Usage: "ignore case distinctions"},
		&urf.BoolFlag{Name: flagInvertMatch, Aliases: []string{"v"}, Usage: "select non-matching lines"},
		&urf.BoolFlag{Name: flagLineNumber, Aliases: []string{"n"}, Usage: "prefix each line with its line number"},
		&urf.BoolFlag{Name: flagCount, Aliases: []string{"c"}, Usage: "print only a count of matching lines"},
		&urf.BoolFlag{Name: flagWord, Aliases: []string{"w"}, Usage: "match only whole words"},
		&urf.BoolFlag{Name: flagLine, Aliases: []string{"x"}, Usage: "match only whole lines"},
		&urf.BoolFlag{
			Name:    flagExtended,
			Aliases: []string{"E"},
			Usage:   "interpret PATTERN as an extended regular expression",
		},
	},
}

// build maps the invocation to grep's pipeline: standard input feeds grep, whose
// first operand is the pattern. A bare invocation is a usage error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	operands := inv.Args.Args().Slice()
	if len(operands) == 0 {
		return nil, nil, ErrMissingPattern
	}
	pattern := command.GrepPattern(operands[0])
	return clix.Stdin(inv.Stdin), command.Grep(pattern, options(inv.Args)...), nil
}

// flagOption pairs a CLI flag name with the library option it enables.
type flagOption struct {
	option any
	name   string
}

func flagOptions() []flagOption {
	return []flagOption{
		{name: flagIgnoreCase, option: command.GrepIgnoreCase},
		{name: flagInvertMatch, option: command.GrepInvert},
		{name: flagLineNumber, option: command.GrepLineNumbers},
		{name: flagCount, option: command.GrepCount},
		{name: flagWord, option: command.GrepWord},
		{name: flagLine, option: command.GrepWholeLine},
		{name: flagExtended, option: command.GrepExtended},
	}
}

// options folds the parsed flags into grep's option values.
func options(c *urf.Command) []any {
	var opts []any
	for _, fo := range flagOptions() {
		if c.Bool(fo.name) {
			opts = append(opts, fo.option)
		}
	}
	return opts
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
