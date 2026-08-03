#!/usr/bin/env bash
# Sourced by bench.sh, and by hyperfine's --prepare commands (via `bash -c
# "source bench-lib.sh; <func> <args>"`) to reset sandbox state between runs.
set -euo pipefail

# reset_dir <root>
# Wipes and recreates a sandbox root with the three subdirectories gotpm/utpm
# need pointed at via env vars: packages ($TYPST_PACKAGE_PATH), data
# ($XDG_DATA_HOME, for each tool's own scratch/cache files), cache
# ($XDG_CACHE_HOME, where utpm's `packages install` actually writes).
reset_dir() {
	local root=$1
	rm -rf "$root"
	mkdir -p "$root/packages" "$root/data" "$root/cache"
}

# seed_list_fixtures <packages-dir> <name:version> [<name:version> ...]
# Creates minimal fake @preview/<name>/<version> package directories directly
# on disk (no install needed) so `list` has a realistic number of entries to
# enumerate.
seed_list_fixtures() {
	local pkgdir=$1
	shift
	mkdir -p "$pkgdir/preview"
	for spec in "$@"; do
		local name=${spec%%:*}
		local ver=${spec##*:}
		local dir="$pkgdir/preview/$name/$ver"
		mkdir -p "$dir"
		printf '[package]\nname = "%s"\nversion = "%s"\nentrypoint = "lib.typ"\n' "$name" "$ver" >"$dir/typst.toml"
		printf '#let greet(name) = [Hello #name]\n' >"$dir/lib.typ"
	done
}

# reset_and_install_gotpm <sandbox-root> <fixture-dir>
# Prepares an uninstall benchmark: fresh sandbox, package installed via gotpm.
# Both tools exit 0 even on some internal errors, so we verify the expected
# on-disk result directly instead of trusting the exit code.
reset_and_install_gotpm() {
	local root=$1 fixture=$2
	reset_dir "$root"
	TYPST_PACKAGE_PATH="$root/packages" XDG_DATA_HOME="$root/data" \
		"$GOTPM_BIN" install "$fixture" -n local >/dev/null
	[ -d "$root/packages/local/bench-fixture/0.1.0" ] || {
		echo "reset_and_install_gotpm: install did not produce expected package dir" >&2
		exit 1
	}
}

# reset_and_install_utpm <sandbox-root> <fixture-dir>
reset_and_install_utpm() {
	local root=$1 fixture=$2
	reset_dir "$root"
	(cd "$fixture" && TYPST_PACKAGE_PATH="$root/packages" XDG_DATA_HOME="$root/data" \
		"$UTPM_BIN" project link >/dev/null)
	[ -d "$root/packages/local/bench-fixture/0.1.0" ] || {
		echo "reset_and_install_utpm: link did not produce expected package dir" >&2
		exit 1
	}
}

# reset_work_copy <fixture-dir> <work-dir>
# Prepares a bump benchmark: fresh writable copy of the fixture package.
reset_work_copy() {
	local fixture=$1 work=$2
	rm -rf "$work"
	cp -r "$fixture" "$work"
}
