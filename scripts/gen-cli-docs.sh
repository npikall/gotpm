#!/usr/bin/env bash
# Generate the CLI help snippets included by docs/reference.md.
#
# Each snippet is the output of "gotpm help <command>", de-indented and stripped
# of the padding fang adds. The snippets are build output, not source: they are
# gitignored and written fresh by "task docs:build" and "task docs:serve", which
# run this script before zensical.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="$root/docs/includes/cli"

# Commands with no help of their own to document.
skipped="help|completion"

tmp="$(mktemp -d)"
bin="$tmp/gotpm"
trap 'rm -rf "$tmp"' EXIT
go build -o "$bin" "$root"

# strip removes the two-space indent fang applies to every line, the trailing
# padding it uses to reach a fixed width, and the blank lines around the whole
# block, so the snippet sits flush inside a fenced code block.
strip() {
	sed -e 's/[[:space:]]*$//' -e 's/^  //' |
		cat -s |
		awk 'NF {found = 1} found {print}' |
		awk '{lines[NR] = $0} END {last = NR; while (last > 0 && lines[last] == "") last--; for (i = 1; i <= last; i++) print lines[i]}'
}

# commands lists every subcommand the binary documents, so a command added
# later shows up in the docs without this script being edited.
commands() {
	"$bin" help |
		sed -n '/^  COMMANDS/,/^  FLAGS/p' |
		awk '{print $1}' |
		grep -Ev "^($skipped)$" |
		grep -E '^[a-z][a-z-]*$'
}

# A snippet left over from a renamed or deleted command would keep being
# included by the docs, so the directory starts empty on every run.
rm -rf "$out"
mkdir -p "$out"

count=0
while read -r cmd; do
	count=$((count + 1))
	"$bin" help "$cmd" | strip >"$out/$cmd.txt"
done < <(commands)

if [[ $count -eq 0 ]]; then
	echo "no commands found in 'gotpm help' output" >&2
	exit 1
fi

echo "wrote $count snippets to docs/includes/cli/"
