#!/usr/bin/env bash
# Boot each example server, verify it serves real SSE bytes, then kill it.
#
# A no-browser smoke check: CI's examples job only compiles them, so a dead
# handler, a wrong port, or a startup panic would ship unnoticed. Run locally
# with `nix develop -c scripts/smoke-examples.sh` or in CI as-is (needs go +
# curl on PATH). Each example is booted on a smoke port (18080/18765/18766 —
# the 1-prefix avoids colliding with the defaults while something else on
# the dev machine holds 8080) via the PORT env the examples honor.
set -euo pipefail
cd "$(dirname "$0")/.."

export GOWORK=off
export GOEXPERIMENT=jsonv2

bin_dir="$(mktemp -d)"
pids=()

cleanup() {
	for pid in "${pids[@]:-}"; do
		kill "$pid" 2>/dev/null || true
	done
	rm -rf "$bin_dir"
}
trap cleanup EXIT

# start_example <name> <go package> <port>
start_example() {
	local name="$1" pkg="$2" port="$3"

	go build -o "$bin_dir/$name" "$pkg"
	PORT="$port" "$bin_dir/$name" >/dev/null 2>&1 &
	pids+=("$!")

	for _ in $(seq 1 20); do
		if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
			return 0
		fi
		sleep 1
	done

	echo "FAIL: $name never listened on :$port within 20s" >&2
	exit 1
}

# sse_bytes <url> <max-seconds>: succeed when the stream yields a frame line.
# curl exits 28 when --max-time cuts a healthy infinite stream, so its exit
# code is deliberately not the verdict — the captured bytes are.
sse_bytes() {
	local url="$1" secs="$2" out
	out="$(curl -sN --max-time "$secs" "$url" 2>/dev/null || true)"
	grep -qEm1 '^(event|data|:)' <<<"$out"
}

http_ok() {
	curl -sf -o /dev/null "$1"
}

echo "==> example (root, :18080)"
start_example example-root ./example 18080
sse_bytes "http://127.0.0.1:18080/events" 5 \
	|| { echo "FAIL: root example /events served no SSE frame" >&2; exit 1; }
curl -sf -X POST "http://127.0.0.1:18080/broadcast?msg=smoke-probe" >/dev/null \
	|| { echo "FAIL: root example /broadcast rejected the probe" >&2; exit 1; }
echo "    root example OK"

echo "==> example/datastar (:18765)"
start_example example-datastar ./example/datastar 18765
http_ok "http://127.0.0.1:18765/" \
	|| { echo "FAIL: datastar example index not served" >&2; exit 1; }
sse_bytes "http://127.0.0.1:18765/events" 8 \
	|| { echo "FAIL: datastar example /events served no SSE frame" >&2; exit 1; }
echo "    datastar example OK"

echo "==> example/htmx (:18766)"
start_example example-htmx ./example/htmx 18766
http_ok "http://127.0.0.1:18766/" \
	|| { echo "FAIL: htmx example index not served" >&2; exit 1; }
http_ok "http://127.0.0.1:18766/sse-container" \
	|| { echo "FAIL: htmx example /sse-container fragment not served" >&2; exit 1; }
sse_bytes "http://127.0.0.1:18766/events" 8 \
	|| { echo "FAIL: htmx example /events served no SSE frame" >&2; exit 1; }
echo "    htmx example OK"

echo ""
echo "ALL EXAMPLE SMOKE CHECKS PASSED"
