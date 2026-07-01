package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		files      map[string]string
		name       string
		version    string
		stdin      string
		wantOut    string
		wantErrSub string
		args       []string
		wantCode   int
	}{
		{
			name:    "basic match from stdin",
			args:    []string{"grep", "hello"},
			stdin:   "hello world\ngoodbye\nhello again\n",
			wantOut: "hello world\nhello again\n",
		},
		{
			name:    "ignore case",
			args:    []string{"grep", "-i", "HELLO"},
			stdin:   "Hello world\ngoodbye\nhELLo\n",
			wantOut: "Hello world\nhELLo\n",
		},
		{
			// Guards the version-flag collision fix: -v must still mean
			// invert-match (select non-matching lines), not print version.
			name:    "invert match via -v alias",
			args:    []string{"grep", "-v", "b"},
			stdin:   "a\nb\nc\n",
			wantOut: "a\nc\n",
		},
		{
			name:    "line numbers",
			args:    []string{"grep", "-n", "hello"},
			stdin:   "foo\nhello\nbar\nhello world\n",
			wantOut: "2:hello\n4:hello world\n",
		},
		{
			name:    "count",
			args:    []string{"grep", "-c", "hello"},
			stdin:   "foo\nhello\nbar\nhello world\n",
			wantOut: "2\n",
		},
		{
			name:    "word match",
			args:    []string{"grep", "-w", "he"},
			stdin:   "he\nhello\nthe\n",
			wantOut: "he\n",
		},
		{
			name:    "whole line match",
			args:    []string{"grep", "-x", "hello"},
			stdin:   "hello\nhello world\n",
			wantOut: "hello\n",
		},
		{
			name:    "extended regexp",
			args:    []string{"grep", "-E", "he.lo"},
			stdin:   "hello\nworld\n",
			wantOut: "hello\n",
		},
		{
			name:    "single file source",
			args:    []string{"grep", "hello", "/in.txt"},
			files:   map[string]string{"/in.txt": "hello\nworld\n"},
			wantOut: "hello\n",
		},
		{
			name: "multiple file sources",
			args: []string{"grep", "hello", "/a.txt", "/b.txt"},
			files: map[string]string{
				"/a.txt": "hello a\nskip\n",
				"/b.txt": "hello b\nnope\n",
			},
			wantOut: "hello a\nhello b\n",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"grep", "--version"},
			wantOut: "grep version 1.2.3\n",
		},
		{
			name:       "missing pattern errors",
			args:       []string{"grep"},
			wantCode:   1,
			wantErrSub: "grep: missing PATTERN operand",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"grep", "--nope", "hello"},
			wantCode:   1,
			wantErrSub: "grep:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for path, content := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(tc.stdin), &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
