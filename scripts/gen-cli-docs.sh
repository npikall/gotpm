#!/usr/bin/env bash
# Regenerate the CLI help snippets included by docs/reference.md.
#
# Each snippet is the output of "gotpm help <command>", de-indented and stripped
# of the padding fang adds. Run "task docs:cli" after changing a command's help
# text; "--check" fails instead of writing, which is what CI wants.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="$root/docs/includes/cli"

# Commands with no help of their own to document.
skipped="help|completion"

check=false
[[ ${1:-} == "--check" ]] && check=true

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

mkdir -p "$out"

status=0
count=0
while read -r cmd; do
	count=$((count + 1))
	file="$out/$cmd.txt"
	if $check; then
		if ! "$bin" help "$cmd" | strip | diff -q - "$file" >/dev/null 2>&1; then
			echo "out of date: docs/includes/cli/$cmd.txt" >&2
			status=1
		fi
	else
		"$bin" help "$cmd" | strip >"$file"
	fi
done < <(commands)

if [[ $count -eq 0 ]]; then
	echo "no commands found in 'gotpm help' output" >&2
	exit 1
fi

if $check; then
	# A snippet whose command is gone would keep being included by the docs.
	for file in "$out"/*.txt; do
		cmd="$(basename "$file" .txt)"
		if ! commands | grep -qx "$cmd"; then
			echo "orphaned: docs/includes/cli/$cmd.txt has no command" >&2
			status=1
		fi
	done
	[[ $status -eq 0 ]] && echo "CLI help snippets are up to date."
	exit $status
fi

echo "wrote $count snippets to docs/includes/cli/"
