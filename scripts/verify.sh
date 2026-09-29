#!/usr/bin/env bash
# One-command pre-push verification gate for go-sse.
#
# Usage:
#   scripts/verify.sh          full gate: fmt + vet + lint + test + nix flake check
#   scripts/verify.sh --fast   skip `nix flake check` (~2 min) for quick iteration
#
# Every go/golangci-lint invocation sets GOWORK=off and GOEXPERIMENT=jsonv2
# explicitly, so the script works with or without direnv (the .envrc exports
# the same values). Tools missing outside `nix develop` are skipped with a
# note rather than failing the gate.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOEXPERIMENT=jsonv2
export GOWORK=off

# A caller-exported GOCACHE pointing at a nonexistent or unwritable path
# (e.g. a full cache mount) fails every go invocation cryptically. Same
# fallback as the coverage-gate flake app: degrade loudly to a temp dir
# instead of dying mid-gate.
if [[ -n "${GOCACHE:-}" ]] && ! mkdir -p "${GOCACHE}" 2>/dev/null; then
	echo "GOCACHE='${GOCACHE}' is not writable; falling back to a temp dir" >&2
	export GOCACHE="$(mktemp -d)"
fi

echo "==> treefmt (formatting check)"
if command -v treefmt >/dev/null 2>&1; then
	treefmt --fail-on-change
	echo "    formatting clean"
else
	echo "    treefmt not found; skipping (nix develop provides it)"
fi

echo "==> go mod tidy (drift check)"
# A go.mod edit without a follow-up tidy has flipflopped go directives in
# this repo repeatedly (the 2026-09-18 red master; buildflow's "go line
# changed 9 times" warning). tidy must be a no-op on any committed tree;
# the snapshots make this check read-only — a dirty result is shown and
# the tree restored, never silently kept.
tidy_tmp="$(mktemp -d)"
trap 'rm -rf "$tidy_tmp"' EXIT
for mod in . ssetest sseparse; do
	(
	cd "$mod"
	cp go.mod "$tidy_tmp/go.mod"
	if [[ -f go.sum ]]; then cp go.sum "$tidy_tmp/go.sum"; fi
	if ! go mod tidy >/dev/null; then
		echo "FAIL: go mod tidy errored in $mod" >&2
		exit 1
	fi
	if ! cmp -s "$tidy_tmp/go.mod" go.mod || { [[ -f $tidy_tmp/go.sum ]] && ! cmp -s "$tidy_tmp/go.sum" go.sum; }; then
		diff -u "$tidy_tmp/go.mod" go.mod >&2 || true
		if [[ -f $tidy_tmp/go.sum ]]; then diff -u "$tidy_tmp/go.sum" go.sum >&2 || true; fi
		echo "FAIL: '$mod' go.mod/go.sum is not tidy-clean — run 'go mod tidy' in $mod and commit the result (tree restored)." >&2
		cp "$tidy_tmp/go.mod" go.mod
		if [[ -f $tidy_tmp/go.sum ]]; then cp "$tidy_tmp/go.sum" go.sum; fi
		exit 1
	fi
	) || exit 1
done
rm -rf "$tidy_tmp"
trap - EXIT
echo "    tidy clean in all three modules"

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
