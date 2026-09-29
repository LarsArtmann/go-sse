#!/usr/bin/env bash
# Print the single golangci-lint version pinned in .github/workflows/ci.yml
# (with the leading "v"). Fails if zero or multiple distinct pins exist.
#
# Shared by scripts/verify.sh (local binary cross-check) and the flake-update
# workflow (nixpkgs binary cross-check), so the extraction regex lives in
# exactly one place.
set -euo pipefail
cd "$(dirname "$0")/.."

pin="$(sed -n 's/^[[:space:]]*version:[[:space:]]*\(v[0-9][0-9.]*\)[[:space:]]*$/\1/p' .github/workflows/ci.yml | sort -u)"
if [[ $(grep -c . <<<"$pin") -ne 1 ]]; then
	echo "FAIL: .github/workflows/ci.yml must pin exactly ONE golangci-lint version across jobs (found: $(echo "$pin" | tr '\n' ' '))" >&2
	exit 1
fi

echo "$pin"
