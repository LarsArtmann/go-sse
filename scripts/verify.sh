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
if [[ -n ${GOCACHE:-} ]] && ! mkdir -p "${GOCACHE}" 2>/dev/null; then
  echo "GOCACHE='${GOCACHE}' is not writable; falling back to a temp dir" >&2
  fallback="$(mktemp -d)"
  export GOCACHE="$fallback"
fi

echo "==> GOEXPERIMENT regression guard"
# GOEXPERIMENT=jsonv2 was removed from every live surface in 1c0fc85
# (2026-09-29): encoding/json/v2 is a go1.27-gated stable API, so the flag is
# unnecessary on the 1.27 toolchain and one stray export or build tag
# resurrects the directive/toolchain split this repo just dug out of. The
# gate must-find-nothing, so any reappearance fails here instead of master.
# (--exclude=verify.sh: this guard's own text is the one legit mention.)
if grep -rn --exclude=verify.sh "GOEXPERIMENT" flake.nix scripts/ .github/workflows/ 2>/dev/null; then
  echo "FAIL: GOEXPERIMENT resurfaced (matches above) — removed everywhere on 2026-09-29; do not re-add it." >&2
  exit 1
fi
echo "    no GOEXPERIMENT references"

echo "==> bare-shfmt fingerprint guard"
# buildflow's formatter step runs bare `shfmt -w`, which re-indents Bash
# arithmetic negation differently from treefmt's canonical style — exactly
# the drift that reddened master twice (2026-10-03, 2026-10-11) as a
# "formatting" daemon commit. CI's treefmt check is the primary net; this
# grep is the cheap second one that fires before anything expensive runs.
# (--exclude=verify.sh: the pattern text below is the one legit match.)
if grep -rnE --include='*.sh' --exclude=verify.sh '\(\(![^ ]|\(\(!$' scripts/ 2>/dev/null; then
  echo "FAIL: bare-shfmt fingerprint in scripts/ (an arithmetic negation lost its space after '(', above)." >&2
  echo "      Canonical style keeps '((! x))'; run 'nix fmt' and commit its output — never bare 'shfmt -w'." >&2
  exit 1
fi
echo "    no bare-shfmt fingerprints"

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
    if ((had_sum)); then
      cmp -s "$snap_sum" go.sum || drift=1
    elif [[ -f go.sum ]]; then
      drift=1 # tidy created a go.sum where none belonged
    fi
    if ((drift)); then
      diff -u "$snap_mod" go.mod >&2 || true
      if [[ -f go.sum ]] && ((! had_sum)); then
        echo "--- unexpected new file: go.sum ---" >&2
      elif ((had_sum)); then
        diff -u "$snap_sum" go.sum >&2 || true
      fi
      echo "FAIL: '$mod' go.mod/go.sum is not tidy-clean — run 'go mod tidy' in $mod and commit the result (tree restored)." >&2
      cp "$snap_mod" go.mod
      if ((had_sum)); then cp "$snap_sum" go.sum; else rm -f go.sum; fi
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

echo "==> AGENTS-facts assertion (documented values must match the tree)"
# AGENTS.md documents load-bearing facts no compiler checks: the coverage
# ratchet baseline and the go-directive policy. Both went stale silently
# once (thresholds 90/95/95 for ten days; the 1.27.1-floor claim outliving
# the minor-form reversal), so the gate now compares the claims against the
# tree. Only the machine-checkable claim sentences are pinned — historical
# mentions of other values in AGENTS.md stay legitimate.
lib_thr="$(awk '$1 == "library" { print $2 }' docs/benchmarks/coverage-baseline.txt)"
ssetest_thr="$(awk '$1 == "ssetest" { print $2 }' docs/benchmarks/coverage-baseline.txt)"
sseparse_thr="$(awk '$1 == "sseparse" { print $2 }' docs/benchmarks/coverage-baseline.txt)"
directive="$(echo "$directive_drift" | head -1)"
facts_ok=1
if [[ -z $lib_thr || -z $ssetest_thr || -z $sseparse_thr ]]; then
  echo "FAIL: could not read coverage baselines from docs/benchmarks/coverage-baseline.txt — expected '<module> <pct>' lines for library/ssetest/sseparse." >&2
  facts_ok=0
else
  if ! grep -qF "thresholds (library ${lib_thr}%, ssetest ${ssetest_thr}%, sseparse ${sseparse_thr}%)" AGENTS.md; then
    echo "FAIL: AGENTS.md's coverage-threshold claim is stale — the ratchet baseline is ${lib_thr}/${ssetest_thr}/${sseparse_thr}." >&2
    echo "      Fix: update the 'nix run .#coverage-gate' line in AGENTS.md to '(library ${lib_thr}%, ssetest ${ssetest_thr}%, sseparse ${sseparse_thr}%)'." >&2
    facts_ok=0
  fi
  if ! grep -qF "${lib_thr}/${ssetest_thr}/${sseparse_thr} thresholds" AGENTS.md; then
    echo "FAIL: AGENTS.md's CI-gotcha threshold claim is stale — the ratchet baseline is ${lib_thr}/${ssetest_thr}/${sseparse_thr}." >&2
    echo "      Fix: update the CI-gotcha bullet in AGENTS.md to 'the flake app's ${lib_thr}/${ssetest_thr}/${sseparse_thr} thresholds'." >&2
    facts_ok=0
  fi
fi
directive="${directive#"go "}"
policy_claim='minor form (`go '"${directive}"'`), all three modules identical'
if ! grep -qF "$policy_claim" AGENTS.md; then
  echo "FAIL: AGENTS.md's go-directive-policy claim is stale — the tree declares '${directive}' in all three go.mod files." >&2
  echo "      Fix: update AGENTS.md's 'Go directive policy' line to the tree's '${directive}' (and only then ask whether the policy itself changed)." >&2
  facts_ok=0
fi
if ((! facts_ok)); then exit 1; fi
echo "    threshold (${lib_thr}/${ssetest_thr}/${sseparse_thr}) and directive (${directive}) claims match the tree"

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

if [[ ${1:-} != "--fast" ]]; then
  echo "==> nix flake check (hermetic gate: builds + tests + vendor hashes)"
  nix flake check
else
  echo "==> skipping nix flake check (--fast)"
fi

echo ""
echo "ALL CHECKS PASSED"
