#!/usr/bin/env bash
#
# Benchmarks gotpm against utpm on a handful of representative commands using
# hyperfine, and injects a summary table into README.md between the
# <!-- BENCH:RESULTS:START --> / <!-- BENCH:RESULTS:END --> markers.
#
# Requirements: hyperfine, jq, git, gotpm, utpm all on $PATH (override with
# $GOTPM_BIN / $UTPM_BIN env vars).
#
# All installs/uninstalls/lists run against sandboxed temp directories via
# $TYPST_PACKAGE_PATH / $XDG_DATA_HOME / $XDG_CACHE_HOME overrides - nothing
# touches your real ~/.local/share/typst (or platform equivalent).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FIXTURES_DIR="$SCRIPT_DIR/bench-fixtures"
LIB="$SCRIPT_DIR/bench-lib.sh"
README="$REPO_ROOT/README.md"
RESULTS_DIR="$REPO_ROOT/scripts/bench-results"

GOTPM_BIN="${GOTPM_BIN:-gotpm}"
UTPM_BIN="${UTPM_BIN:-utpm}"
export GOTPM_BIN UTPM_BIN

REMOTE_REPO="github.com/lilaq-project/lilaq"
REMOTE_TAG="v0.6.0"
REMOTE_URL="https://github.com/lilaq-project/lilaq"

# name:version pairs - the same real Typst Universe packages (pinned to their
# oldest published version) as scripts/bench-fixtures/update-imports.typ, also
# reused here to seed a realistic number of entries for the `list` benchmark.
PKG_SPECS=(
	cetz:0.0.1 tablex:0.0.4 codly:0.1.0 physica:0.7.5 showybox:0.1.1
	fletcher:0.1.1 cetz-plot:0.1.0 zebraw:0.1.0 glossarium:0.1.0 tidy:0.1.0
	oxifmt:0.2.0 codelst:0.0.3 pinit:0.1.0 equate:0.1.0 mitex:0.1.0
	curryst:0.1.0 hydra:0.1.0 touying:0.1.0
)

die() {
	echo "error: $*" >&2
	exit 1
}

check_prereqs() {
	command -v hyperfine >/dev/null 2>&1 || die "hyperfine not found. Install: https://github.com/sharkdp/hyperfine#installation"
	command -v jq >/dev/null 2>&1 || die "jq not found. Install: your package manager (apt/brew install jq)"
	command -v git >/dev/null 2>&1 || die "git not found on \$PATH"
	command -v "$GOTPM_BIN" >/dev/null 2>&1 || die "gotpm not found on \$PATH. Install: see README, or set \$GOTPM_BIN"
	command -v "$UTPM_BIN" >/dev/null 2>&1 || die "utpm not found on \$PATH. Install: cargo install utpm, or set \$UTPM_BIN"
}

TMPROOT="$(mktemp -d)"
cleanup() { rm -rf "$TMPROOT"; }
trap cleanup EXIT

run_bench() {
	local name=$1 out=$2 prep_g=$3 cmd_g=$4 prep_u=$5 cmd_u=$6
	echo "==> $name"
	hyperfine --shell bash \
		--export-json "$out" \
		--command-name gotpm --prepare "$prep_g" "$cmd_g" \
		--command-name utpm --prepare "$prep_u" "$cmd_u"
}

main() {
	check_prereqs
	mkdir -p "$RESULTS_DIR"

	local sbox_g="$TMPROOT/sandbox-gotpm"
	local sbox_u="$TMPROOT/sandbox-utpm"
	local work_g="$TMPROOT/work-gotpm"
	local work_u="$TMPROOT/work-utpm"

	# --- install (local) ---
	run_bench "install (local)" "$RESULTS_DIR/install-local.json" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_g\"'" \
		"TYPST_PACKAGE_PATH=$sbox_g/packages XDG_DATA_HOME=$sbox_g/data $GOTPM_BIN install '$FIXTURES_DIR/local-package' -n local" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_u\"'" \
		"cd '$FIXTURES_DIR/local-package' && TYPST_PACKAGE_PATH=$sbox_u/packages XDG_DATA_HOME=$sbox_u/data $UTPM_BIN project link"

	# --- install (remote git) ---
	run_bench "install (remote git)" "$RESULTS_DIR/install-remote.json" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_g\"'" \
		"TYPST_PACKAGE_PATH=$sbox_g/packages XDG_DATA_HOME=$sbox_g/data $GOTPM_BIN install -r $REMOTE_REPO -t $REMOTE_TAG -n preview" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_u\"'" \
		"TYPST_PACKAGE_PATH=$sbox_u/packages XDG_DATA_HOME=$sbox_u/data XDG_CACHE_HOME=$sbox_u/cache $UTPM_BIN packages install $REMOTE_URL -n preview"

	# --- uninstall ---
	run_bench "uninstall" "$RESULTS_DIR/uninstall.json" \
		"bash -c 'source \"$LIB\"; reset_and_install_gotpm \"$sbox_g\" \"$FIXTURES_DIR/local-package\"'" \
		"TYPST_PACKAGE_PATH=$sbox_g/packages XDG_DATA_HOME=$sbox_g/data $GOTPM_BIN uninstall bench-fixture -n local -V 0.1.0" \
		"bash -c 'source \"$LIB\"; reset_and_install_utpm \"$sbox_u\" \"$FIXTURES_DIR/local-package\"'" \
		"TYPST_PACKAGE_PATH=$sbox_u/packages XDG_DATA_HOME=$sbox_u/data $UTPM_BIN packages unlink @local/bench-fixture:0.1.0 -y"

	# --- list ---
	run_bench "list" "$RESULTS_DIR/list.json" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_g\"; seed_list_fixtures \"$sbox_g/packages\" ${PKG_SPECS[*]}'" \
		"TYPST_PACKAGE_PATH=$sbox_g/packages XDG_DATA_HOME=$sbox_g/data $GOTPM_BIN list" \
		"bash -c 'source \"$LIB\"; reset_dir \"$sbox_u\"; seed_list_fixtures \"$sbox_u/packages\" ${PKG_SPECS[*]}'" \
		"TYPST_PACKAGE_PATH=$sbox_u/packages XDG_DATA_HOME=$sbox_u/data $UTPM_BIN packages list -a"

	# --- bump (patch) ---
	run_bench "bump (patch)" "$RESULTS_DIR/bump.json" \
		"bash -c 'source \"$LIB\"; reset_work_copy \"$FIXTURES_DIR/local-package\" \"$work_g\"'" \
		"cd '$work_g' && $GOTPM_BIN bump patch" \
		"bash -c 'source \"$LIB\"; reset_work_copy \"$FIXTURES_DIR/local-package\" \"$work_u\"'" \
		"cd '$work_u' && $UTPM_BIN project bump 0.1.1"

	# --- update / sync ---
	run_bench "update (18 imports)" "$RESULTS_DIR/update.json" \
		"cp '$FIXTURES_DIR/update-imports.typ' '$work_g.typ'" \
		"$GOTPM_BIN update '$work_g.typ' --no-cache" \
		"cp '$FIXTURES_DIR/update-imports.typ' '$work_u.typ'" \
		"$UTPM_BIN project sync -f '$work_u.typ'"

	build_summary
}

# fmt_time <seconds> -> human string
fmt_time() {
	awk -v s="$1" 'BEGIN {
		if (s < 1) printf "%.0f ms", s * 1000
		else printf "%.2f s", s
	}'
}

build_summary() {
	local date os gotpm_ver utpm_ver
	date="$(date -u +%Y-%m-%d)"
	os="$(uname -srm)"
	gotpm_ver="$($GOTPM_BIN self version 2>&1 | awk -F'version=' '{print $2}' | awk '{print $1}')"
	utpm_ver="$($UTPM_BIN --version | awk '{print $2}')"

	{
		echo "| Command | gotpm | utpm | Result |"
		echo "| --- | --- | --- | --- |"
		summarize_row "Install (local)" "$RESULTS_DIR/install-local.json"
		summarize_row "Install (remote git)" "$RESULTS_DIR/install-remote.json"
		summarize_row "Uninstall" "$RESULTS_DIR/uninstall.json"
		summarize_row "List" "$RESULTS_DIR/list.json"
		summarize_row "Bump (patch)" "$RESULTS_DIR/bump.json"
		summarize_row "Update (18 imports)" "$RESULTS_DIR/update.json"
		echo ""
		echo "*Benchmarked $date on $os with gotpm $gotpm_ver and utpm $utpm_ver."
		echo "Results vary by machine and network conditions - run \`scripts/bench.sh\`"
		echo "yourself to reproduce. Whichever tool is faster for a given command is"
		echo "reported as-is.*"
	} >"$RESULTS_DIR/summary.md"

	echo
	echo "Summary written to $RESULTS_DIR/summary.md"
	cat "$RESULTS_DIR/summary.md"

	inject_readme "$RESULTS_DIR/summary.md"
}

summarize_row() {
	local label=$1 file=$2
	local mean_g mean_u
	mean_g="$(jq -r '.results[] | select(.command=="gotpm") | .mean' "$file")"
	mean_u="$(jq -r '.results[] | select(.command=="utpm") | .mean' "$file")"

	local result
	result="$(awk -v g="$mean_g" -v u="$mean_u" 'BEGIN {
		if (g < u) printf "gotpm %.1fx faster", u / g
		else if (u < g) printf "utpm %.1fx faster", g / u
		else printf "tie"
	}')"

	echo "| $label | $(fmt_time "$mean_g") | $(fmt_time "$mean_u") | $result |"
}

inject_readme() {
	local summary_file=$1
	local start="<!-- BENCH:RESULTS:START -->"
	local end="<!-- BENCH:RESULTS:END -->"

	grep -qF "$start" "$README" || die "README.md is missing the $start marker"
	grep -qF "$end" "$README" || die "README.md is missing the $end marker"

	local tmp
	tmp="$(mktemp)"
	awk -v start="$start" -v end="$end" -v summary="$summary_file" '
		BEGIN { in_block = 0 }
		$0 == start { print; while ((getline line < summary) > 0) print line; in_block = 1; next }
		$0 == end { in_block = 0 }
		!in_block { print }
	' "$README" >"$tmp"
	mv "$tmp" "$README"

	echo "README.md updated between $start and $end"
}

main "$@"
