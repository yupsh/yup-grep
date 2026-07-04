#!/bin/sh
# Integration checks for yup-grep, run inside a Debian container against the GNU
# `grep` reference. LC_ALL=C pins byte-wise collation so case/word semantics match.
#
# parity_stdin INPUT PATTERN-ARGS...  — yup-grep reading stdin must match GNU `grep`.
# parity_file  PATTERN-ARGS...        — yup-grep reading a file operand must match GNU.
#
# grep exits 1 when no line matches (not an error); each helper captures stdout
# with `|| true` and compares stdout regardless of exit status.
set -eu
export LC_ALL=C

fails=0
sample='hello world
goodbye
Hello again
the end
hello'

parity_stdin() {
	in=$1
	shift
	ours=$(printf '%s\n' "$in" | yup-grep "$@" 2>/dev/null || true)
	gnu=$(printf '%s\n' "$in" | grep "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  grep %s < stdin\n' "$*"
	else
		printf 'FAIL  parity  grep %s < stdin\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

parity_file() {
	ours=$(yup-grep "$@" 2>/dev/null || true)
	gnu=$(grep "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  grep %s\n' "$*"
	else
		printf 'FAIL  parity  grep %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

# Basic fixed-string match (yup-grep's default mode == GNU `grep -F`).
parity_stdin "$sample" hello
# -i: case-insensitive match.
parity_stdin "$sample" -i hello
# -v: invert — select non-matching lines.
parity_stdin "$sample" -v hello
# -n: prefix each match with its 1-based line number.
parity_stdin "$sample" -n hello
# -c: print only the count of matching lines.
parity_stdin "$sample" -c hello
# No match: GNU exits 1 with empty stdout; yup-grep must produce the same stdout.
parity_stdin "$sample" missing
parity_stdin "$sample" -c missing
# -E: extended regexp with alternation.
parity_stdin "$sample" -E 'hello|end'
# -E with a metacharacter class.
parity_stdin "$sample" -E 'h.llo'
# -w: whole-word match (boundary semantics).
parity_stdin "$sample" -w the
# -x: whole-line match.
parity_stdin "$sample" -x hello
# Combined flags supplied separately (cli/v3 does not bundle short bool flags
# as `-in`; pass them as distinct tokens, which matches GNU's result).
parity_stdin "$sample" -i -n hello

# Fixed-string default: yup-grep matches literally by default (no regex mode),
# which equals GNU `grep -F`. yup-grep has no -F flag, so compare its bare
# default invocation against GNU's explicit -F: a literal '.' is not a wildcard.
fixed_in='a.c
abc'
ours=$(printf '%s\n' "$fixed_in" | yup-grep 'a.c' 2>/dev/null || true)
gnu=$(printf '%s\n' "$fixed_in" | grep -F 'a.c' 2>/dev/null || true)
if [ "$ours" = "$gnu" ]; then
	printf 'ok    parity  grep a.c (default fixed) == GNU grep -F a.c\n'
else
	printf 'FAIL  parity  grep a.c (default fixed)\n        gnu:  %s\n        ours: %s\n' "$gnu" "$ours"
	fails=$((fails + 1))
fi

# File operand: pattern + single FILE.
printf '%s\n' "$sample" > /tmp/in.txt
parity_file hello /tmp/in.txt
parity_file -n hello /tmp/in.txt

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
