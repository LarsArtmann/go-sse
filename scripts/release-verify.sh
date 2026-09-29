#!/usr/bin/env bash
# Post-push release probe: verify the module proxy serves a freshly tagged
# release AND that a from-scratch consumer (fresh temp module, no replace
# directives, proxy-only resolution) can resolve, build, and import it.
#
# This encodes CONTRIBUTING.md release-checklist step 6 as a script: the
# hand-run version was fumbled per release (wrong function signatures,
# masked exit codes — 18-25 report d2). Every command here runs under
# set -euo pipefail with no fallbacks: a failure reddens the script.
#
# Usage:
#   scripts/release-verify.sh vX.Y.Z            # root library tag
#   scripts/release-verify.sh ssetest/vX.Y.Z    # ssetest module tag
#   scripts/release-verify.sh sseparse/vX.Y.Z   # sseparse module tag
set -euo pipefail

tag="${1:?usage: scripts/release-verify.sh <vX.Y.Z | ssetest/vX.Y.Z | sseparse/vX.Y.Z>}"

module="github.com/larsartmann/go-sse"
probe_import="sse \"github.com/larsartmann/go-sse\""
probe_body='_ = sse.Event{Event: "verify", Data: "release probe"}
	fmt.Fprintln(os.Stderr, "root library probe:", sse.ContentType)'

case "$tag" in
sseparse/*)
	module="github.com/larsartmann/go-sse/sseparse"
	probe_import="sseparse \"github.com/larsartmann/go-sse/sseparse\"
	\"strings\""
	probe_body='events, err := sseparse.ReadEvents(strings.NewReader("data: hi\n\n"))
	if err != nil || len(events) != 1 || len(events[0].DataLines) != 1 || events[0].DataLines[0] != "hi" {
		fmt.Fprintf(os.Stderr, "sseparse probe FAILED: err=%v events=%v\n", err, events)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "sseparse probe: parsed %d event(s), data=%q\n", len(events), events[0].Data())'
	;;
ssetest/*)
	module="github.com/larsartmann/go-sse/ssetest"
	probe_import="ssetest \"github.com/larsartmann/go-sse/ssetest\""
	probe_body='_ = ssetest.ReadEvents
	fmt.Fprintln(os.Stderr, "ssetest probe: ReadEvents resolved")'
	;;
v*) ;;
*)
	echo "FAIL: tag must look like vX.Y.Z, ssetest/vX.Y.Z, or sseparse/vX.Y.Z (got: $tag)" >&2
	exit 2
	;;
esac

export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOPROXY="https://proxy.golang.org,direct"
# The ambient shell may carry an older toolchain pinned via GOTOOLCHAIN=local;
# go list would then die with "go.mod requires go >= 1.27" and the failure
# would masquerade as a proxy lag below.
export GOTOOLCHAIN=auto

echo "==> 1/3 proxy serves $module@$tag (version-specific fetch; the @v/list index lags for brand-new modules)"
if ! GOPROXY="https://proxy.golang.org" go list -m -json "$module@$tag" >/dev/null 2>&1; then
	echo "FAIL: the proxy does not serve $module@$tag yet" >&2
	echo "      (the proxy can lag a few minutes behind the push; retry)" >&2
	exit 1
fi

echo "==> 2/3 proxy serves the module zip"
go mod download "$module@$tag"

echo "==> 3/3 scratch consumer resolves, builds, and imports it"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

cd "$scratch"
go mod init example.com/release-probe >/dev/null
go get "$module@$tag" >/dev/null

cat >main.go <<EOF
package main

import (
	"fmt"
	"os"

	$probe_import
)

func main() {
	$probe_body
}
EOF

go build ./...
go vet ./...

echo ""
echo "RELEASE PROBE PASSED: $module@$tag is consumable from scratch"
