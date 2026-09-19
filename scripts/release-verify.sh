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
set -euo pipefail

tag="${1:?usage: scripts/release-verify.sh <vX.Y.Z | ssetest/vX.Y.Z>}"

module="github.com/larsartmann/go-sse"
probe_import="sse \"github.com/larsartmann/go-sse\""
probe_body='_ = sse.Event{Event: "verify", Data: "release probe"}
	fmt.Fprintln(os.Stderr, "root library probe:", sse.ContentType)'

case "$tag" in
ssetest/*)
	module="github.com/larsartmann/go-sse/ssetest"
	probe_import="ssetest \"github.com/larsartmann/go-sse/ssetest\""
	probe_body='_ = ssetest.ReadEvents
	fmt.Fprintln(os.Stderr, "ssetest probe: ReadEvents resolved")'
	;;
v*) ;;
*)
	echo "FAIL: tag must look like vX.Y.Z or ssetest/vX.Y.Z (got: $tag)" >&2
	exit 2
	;;
esac

export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOPROXY="https://proxy.golang.org,direct"

echo "==> 1/3 proxy lists $module@$tag in its version index"
if ! go list -m -versions "$module" | tr ' ' '\n' | grep -qx "$tag"; then
	echo "FAIL: $tag not in go list -m -versions $module output yet" >&2
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
