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
