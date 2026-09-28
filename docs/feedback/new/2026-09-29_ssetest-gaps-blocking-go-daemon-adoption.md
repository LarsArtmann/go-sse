# ssetest: what's missing for go-daemon adoption

> **Processed 2026-09-29** — all four asks implemented (module named `sseparse`
> instead of the suggested `ssetestcore`): (1) zero-dependency parser core
> module `github.com/larsartmann/go-sse/sseparse`, ssetest re-exports it;
> (2) corpus exported as data via `sseparse.Corpus()`/`MustCorpus` backed by
> `sseparse/testdata/wpt_format_corpus.json`; (3) reader API promoted to a
> first-class README section; (4) cap documented on `ReadEvents`, boundary
> pinned by tests, `WithMaxLineBytes` added. go.mod alignment handled with a
> nuance: sseparse starts at `go 1.27.1` (json/v2 user); ssetest stays at
> `go 1.27` — post-split its own code no longer needs 1.27.1 and `go mod tidy`
> reverts any forced parity. Pending: the two release tags (see TODO_LIST).

2026-09-29. Feedback from the go-daemon side
(github.com/LarsArtmann/go-daemon, private mechanism library for unix-socket
daemons, go 1.27.1). go-daemon ships its own framing-only SSE parser
(`ParseSSEData`, sse.go:36) and does NOT use ssetest today. Verified against
ssetest at go-sse v0.6.0 (ssetest module, `go 1.27`).

## Problem

go-daemon wants an SSE spec oracle for its parser tests but cannot take
ssetest as-is. Three gaps and one trivial.

## 1. The go-sse require is test-only but costs every consumer

ssetest's non-test code never imports go-sse. `rg -l 'larsartmann/go-sse'
ssetest --type go` hits only `*_test.go` files plus doc.go's example comment.
Yet ssetest/go.mod carries `require github.com/larsartmann/go-sse v0.6.0` as a
direct require, so every consumer's module graph also pulls go-sse v0.6.0,
go-branded-id, and go-error-family.

For a public consumer that is noise. For go-daemon it is disqualifying: the
repo is private (GOPRIVATE), and consumers pin it in hermetic nix builds via
vendorHash. Three extra modules in the graph, resolved over SSH and re-hashed
on every bump, for zero compiled code.

**Ask**: split the parser + assertion core (reader.go, event.go, assert.go)
into its own module, leaving the HTTP collection helpers and e2e tests in
ssetest. `go get .../ssetestcore` then costs one module with nothing
go-sse-shaped in the graph.

## 2. No exportable golden corpus

The WPT conformance vectors are pinned internally
(wpt_format_corpus_test.go) but not consumable by other parsers. The only
sane use of ssetest from go-daemon is as a spec oracle, but asserting
go-daemon's parser through ssetest's parser couples two private test suites:
a go-sse regression would fail go-daemon CI as a confusing diff.

**Ask**: export the corpus as data. Wire-format input plus expected dispatch,
serialized (JSON expectations in testdata, go:embed-able), so a third-party
parser can assert against the same vectors without running ssetest code in
the loop. The framing-only projection is the valuable part: go-daemon
surfaces only data payloads (`emit func(string)`, sse.go:36), so vectors
marking which frames dispatch, and which are comments/id/retry-only, are
exactly what its tests need.

## 3. Reader API exists but is positioned as an afterthought

Five of five Collect helpers require an `http.Handler` (collect.go:25,48,74,97,129).
`ReadEvents`/`NewStreamReader` (reader.go:50, reader.go:157) are precisely
what non-HTTP consumers need, but the README lists them last as "Parse SSE
from any io.Reader yourself". go-daemon parses socket streams, not handlers.

**Ask**: promote the reader API to a first-class README section. Event being
ssetest-owned (event.go:24, not an alias of go-sse types) is the right call;
keep it that way.

## 4. Fixed 1 MiB line cap, untested boundary

reader.go:16: `maxLineBytes = 1024 * 1024`, hardwired into newSSEScanner
(reader.go:276). Not configurable; at the cap bufio returns ErrTooLong and no
corpus vector pins that behavior. go-daemon's parser takes a maxEventBytes
parameter (sse.go:36) and its suite includes a 512 KiB event
(sse_test.go:181). For framing parsers the size axis is first-class.

**Ask**: document the cap on ReadEvents, pin the over-cap behavior with a
vector, consider a WithMaxLineBytes option.

## Trivial

- ssetest/go.mod says `go 1.27`; go-sse/go.mod says `go 1.27.1`. Align.

## What would flip adoption

In order: (1) core module split removes the dependency argument entirely;
(2) corpus export makes ssetest the shared spec oracle across my
SSE-touching repos instead of a runtime oracle; (3) and (4) are polish. With
1+2, go-daemon's sse_test.go (10 hand-rolled framing tests today) migrates
to corpus vectors, and project-discovery-daemon inherits the conformance
guarantee through it.

---

💘 Generated with Crush (GLM), from the go-daemon adoption review, 2026-09-29.
