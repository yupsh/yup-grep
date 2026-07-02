package main

import (
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-grep"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const name = "grep"

// Error is the sentinel error type for the yup-grep wrapper. It lets every
// error path this package raises be matched with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingPattern is returned when no PATTERN operand is supplied.
const ErrMissingPattern Error = "missing PATTERN operand"

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `grep [OPTIONS] PATTERN [FILE...]

Search for PATTERN in each FILE.
When no FILE is specified, read from standard input.
PATTERN is a fixed-string match by default.`

const (
	flagIgnoreCase  = "ignore-case"
	flagInvertMatch = "invert-match"
	flagLineNumber  = "line-number"
	flagCount       = "count"
	flagWord        = "word-regexp"
	flagLine        = "line-regexp"
	flagExtended    = "extended-regexp"
)

// buildVersion is the binary's build version threaded from main's ldflags
// target (`var version`) into the CLI. It is an alias, not a defined type:
// cli.Command.Version is a plain string and must be wired as the bare
// `version` identifier (no conversion) for --version to stay verifiably
// bound to the ldflags symbol.
type buildVersion = string

// run builds and executes the grep CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version buildVersion, args []string, stdin io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdin, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, name+": %v\n", err)
		return 1
	}
	return 0
}

func newApp(version buildVersion, stdin io.Reader, stdout io.Writer, fs afero.Fs) *cli.Command {
	// Replace urfave/cli's default --version/-v flag with a --version-only
	// flag, freeing the single-letter -v for command flags (e.g. grep -v)
	// while still exposing the injected build version. Done here rather than
	// in func init so construction stays explicit.
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
	return &cli.Command{
		Name:            name,
		Version:         version,
		Usage:           "search for patterns in files",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags:          flags(),
		Action:         action(stdin, stdout, fs),
	}
}

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: flagIgnoreCase, Aliases: []string{"i"}, Usage: "ignore case distinctions"},
		&cli.BoolFlag{Name: flagInvertMatch, Aliases: []string{"v"}, Usage: "select non-matching lines"},
		&cli.BoolFlag{Name: flagLineNumber, Aliases: []string{"n"}, Usage: "prefix each line with its line number"},
		&cli.BoolFlag{Name: flagCount, Aliases: []string{"c"}, Usage: "print only a count of matching lines"},
		&cli.BoolFlag{Name: flagWord, Aliases: []string{"w"}, Usage: "match only whole words"},
		&cli.BoolFlag{Name: flagLine, Aliases: []string{"x"}, Usage: "match only whole lines"},
		&cli.BoolFlag{
			Name:    flagExtended,
			Aliases: []string{"E"},
			Usage:   "interpret PATTERN as an extended regular expression",
		},
	}
}

func action(stdin io.Reader, stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		if c.NArg() == 0 {
			return ErrMissingPattern
		}
		pattern := c.Args().Get(0)
		cmd := command.Grep(pattern, options(c)...)
		_, err := gloo.Run(source(c, stdin, fs), gloo.ByteWriteTo(stdout), cmd)
		return err
	}
}

func source(c *cli.Command, stdin io.Reader, fs afero.Fs) any {
	if c.NArg() <= 1 {
		return gloo.ByteReaderSource([]io.Reader{stdin})
	}
	names := c.Args().Slice()[1:]
	files := make([]gloo.File, len(names))
	for i, name := range names {
		files[i] = gloo.File(name)
	}
	return gloo.ByteFileSource(fs, files)
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

func options(c *cli.Command) []any {
	var opts []any
	for _, fo := range flagOptions() {
		if c.Bool(fo.name) {
			opts = append(opts, fo.option)
		}
	}
	return opts
}
