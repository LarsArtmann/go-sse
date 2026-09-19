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

echo "==> treefmt (formatting check)"
if command -v treefmt >/dev/null 2>&1; then
	treefmt --fail-on-change
	echo "    formatting clean"
else
	echo "    treefmt not found; skipping (nix develop provides it)"
fi

echo "==> go vet"
go vet ./...
(cd ssetest && go vet ./...)

echo "==> golangci-lint"
if command -v golangci-lint >/dev/null 2>&1; then
	# The CI pin must match the local binary (nixpkgs `pkgs.golangci-lint` —
	# the same package the devShell and `nix run .#lint` use). A 2.12/2.13
	# skew once let a goconst counting difference pass locally and fail CI on
	# every push; this cross-check closes that class. When nixpkgs bumps the
	# package, this fails until the `version:` lines in ci.yml follow.
	pin="$(sed -n 's/^[[:space:]]*version:[[:space:]]*\(v[0-9][0-9.]*\)[[:space:]]*$/\1/p' .github/workflows/ci.yml | sort -u)"
	if [[ $(grep -c . <<<"$pin") -ne 1 ]]; then
		echo "FAIL: .github/workflows/ci.yml must pin exactly ONE golangci-lint version across jobs (found: $(echo "$pin" | tr '\n' ' '))" >&2
		exit 1
	fi
	local_version="$(golangci-lint version --short 2>/dev/null)"
	if [[ "v$local_version" != "$pin" ]]; then
		echo "FAIL: golangci-lint version skew — ci.yml pins $pin, local (nixpkgs) binary is v$local_version." >&2
		echo "      Bump every 'version:' line in .github/workflows/ci.yml to v$local_version." >&2
		exit 1
	fi
	echo "    pin matches local binary ($pin)"

	golangci-lint run ./...
	(cd ssetest && golangci-lint run ./...)
	echo "    lint clean"
else
	echo "    golangci-lint not found; skipping (nix develop provides it)"
fi

echo "==> go test (race)"
go test ./... -race -count=1
(cd ssetest && go test ./... -race -count=1)

if [[ "${1:-}" != "--fast" ]]; then
	echo "==> nix flake check (hermetic gate: builds + tests + vendor hashes)"
	nix flake check
else
	echo "==> skipping nix flake check (--fast)"
fi

echo ""
echo "ALL CHECKS PASSED"
