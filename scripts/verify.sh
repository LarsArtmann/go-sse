#!/usr/bin/env bash
# One-command pre-push verification gate for go-sse.
#
# Usage:
#   scripts/verify.sh          full gate: fmt + vet + lint + test + nix flake check
#   scripts/verify.sh --fast   skip `nix flake check` (~2 min) for quick iteration
#
# Every go/golangci-lint invocation sets GOWORK=off explicitly, so the script
# works with or without direnv (the .envrc exports the same value). Tools
# missing outside `nix develop` are skipped with a note rather than failing
# the gate.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOWORK=off

# A caller-exported GOCACHE pointing at a nonexistent or unwritable path
# (e.g. a full cache mount) fails every go invocation cryptically. Same
# fallback as the coverage-gate flake app: degrade loudly to a temp dir
# instead of dying mid-gate.
if [[ -n "${GOCACHE:-}" ]] && ! mkdir -p "${GOCACHE}" 2>/dev/null; then
	echo "GOCACHE='${GOCACHE}' is not writable; falling back to a temp dir" >&2
	fallback="$(mktemp -d)"
	export GOCACHE="$fallback"
fi

echo "==> treefmt (formatting check)"
if command -v treefmt >/dev/null 2>&1; then
	treefmt --fail-on-change
	echo "    formatting clean"
else
	echo "    treefmt not found; skipping (nix develop provides it)"
fi

echo "==> go mod tidy (drift check)"
# Module files must be tidy-clean: a go.mod/go.sum edit without a follow-up
# tidy has repeatedly drifted this repo (buildflow's "go line changed 9
# times" warning). The snapshots make this check read-only — a dirty result
# is shown and the tree restored, never silently kept. (Note: plain `go mod
# tidy` does not revert hand-raised go directives — buildflow's normalize
# step does — but it does pin requires and sums, and the directive-equality
# check below closes the together-bump trap.)
tidy_tmp="$(mktemp -d)"
trap 'rm -rf "$tidy_tmp"' EXIT
for mod in . ssetest sseparse; do
	(
	cd "$mod"
	name="$(echo "$mod" | tr '/' '_')"
	snap_mod="$tidy_tmp/$name.go.mod"
	snap_sum="$tidy_tmp/$name.go.sum"
	cp go.mod "$snap_mod"
	had_sum=0
	if [[ -f go.sum ]]; then
		cp go.sum "$snap_sum"
		had_sum=1
	fi
	if ! go mod tidy >/dev/null; then
		echo "FAIL: go mod tidy errored in $mod" >&2
		exit 1
	fi
	drift=0
	cmp -s "$snap_mod" go.mod || drift=1
	if (( had_sum )); then
		cmp -s "$snap_sum" go.sum || drift=1
	elif [[ -f go.sum ]]; then
		drift=1 # tidy created a go.sum where none belonged
	fi
	if (( drift )); then
		diff -u "$snap_mod" go.mod >&2 || true
		if [[ -f go.sum ]] && (( ! had_sum )); then
			echo "--- unexpected new file: go.sum ---" >&2
		elif (( had_sum )); then
			diff -u "$snap_sum" go.sum >&2 || true
		fi
		echo "FAIL: '$mod' go.mod/go.sum is not tidy-clean — run 'go mod tidy' in $mod and commit the result (tree restored)." >&2
		cp "$snap_mod" go.mod
		if (( had_sum )); then cp "$snap_sum" go.sum; else rm -f go.sum; fi
		exit 1
	fi
	) || exit 1
done
rm -rf "$tidy_tmp"
trap - EXIT
# All three go.mod files must declare the SAME go directive: a lagging
# nested module is exactly what reddened master on 2026-09-18 (root bumped,
# ssetest left behind).
directive_drift="$(grep -h '^go ' go.mod ssetest/go.mod sseparse/go.mod | sort -u)"
if [[ $(grep -c . <<<"$directive_drift") -ne 1 ]]; then
	echo "FAIL: module go directives diverged — bump all three go.mod files together (found: $(echo "$directive_drift" | tr '\n' ' '))" >&2
	exit 1
fi
echo "    tidy clean in all three modules; go directives aligned ($(echo "$directive_drift" | tr -d '
'))"

echo "==> go vet"
go vet ./...
(cd ssetest && go vet ./...)
(cd sseparse && go vet ./...)

echo "==> golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
	# The CI pin must match the local binary (nixpkgs `pkgs.golangci-lint` —
	# the same package the devShell and `nix run .#lint` use). A 2.12/2.13
	# skew once let a goconst counting difference pass locally and fail CI on
	# every push; this cross-check closes that class. When nixpkgs bumps the
	# package, this fails until the `version:` lines in ci.yml follow. The
	# flake-update workflow runs the same check against ITS nixpkgs so the
	# skew cannot even reach master (scripts/golangci-pin.sh is shared).
	pin="$(scripts/golangci-pin.sh)"
	local_version="$(golangci-lint version --short 2>/dev/null)"
	if [[ "v$local_version" != "$pin" ]]; then
		echo "FAIL: golangci-lint version skew — ci.yml pins $pin, local (nixpkgs) binary is v$local_version." >&2
		echo "      Bump every 'version:' line in .github/workflows/ci.yml to v$local_version." >&2
		exit 1
	fi
	echo "    pin matches local binary ($pin)"

	golangci-lint run ./...
	(cd ssetest && golangci-lint run ./...)
	(cd sseparse && golangci-lint run ./...)
	echo "    lint clean"
else
	echo "    golangci-lint not found; skipping (nix develop provides it)"
fi

echo "==> go test (race)"
go test ./... -race -count=1
(cd ssetest && go test ./... -race -count=1)
(cd sseparse && go test ./... -race -count=1)

echo "==> bench smoke (a moved or broken benchmark must fail the gate, not rot silently)"
go test -run '^$' -bench=. -benchtime=1x -count=1 ./...
(cd ssetest && go test -run '^$' -bench=. -benchtime=1x -count=1)
(cd sseparse && go test -run '^$' -bench=. -benchtime=1x -count=1)

if [[ "${1:-}" != "--fast" ]]; then
	echo "==> nix flake check (hermetic gate: builds + tests + vendor hashes)"
	nix flake check
else
	echo "==> skipping nix flake check (--fast)"
fi

echo ""
echo "ALL CHECKS PASSED"
