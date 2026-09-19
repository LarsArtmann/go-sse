# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

With Nix (recommended — provides Go 1.27, golangci-lint, gopls, govulncheck):

    nix develop              # enter dev shell
    scripts/verify.sh        # one-command pre-push gate (fmt + vet + lint + test + flake check)
    nix run .#test-race      # tests with race detector
    nix run .#lint           # golangci-lint
    nix run .#vet            # go vet
    nix run .#coverage       # test + coverage report
    nix run .#coverage-gate  # fail under the coverage thresholds
    nix flake check          # full hermetic check

Without Nix, use raw `go` tooling:

    GOEXPERIMENT=jsonv2 go test ./... -race
    GOEXPERIMENT=jsonv2 go vet ./...
    golangci-lint run ./...

`GOEXPERIMENT=jsonv2` is required to build (transitive dependency via
`go-branded-id`). Without it, compilation fails. This environment variable
enables the `goexperiment.jsonv2` build tag. The `flake.nix` devShell sets
both `GOEXPERIMENT=jsonv2` and `GOWORK=off` automatically; in non-Nix
environments you must set them yourself.

If a parent `go.work` includes sibling projects with checksum conflicts,
also set `GOWORK=off` to isolate this module's dependency graph.

## Reporting Issues

Please use GitHub Issues to report bugs or request features.

## Release Checklist

The first releases fumbled the same steps; run this list top to bottom and
do not skip items. Both modules version independently: the root library tags
`vX.Y.Z`, the `ssetest` module tags `ssetest/vX.Y.Z`.

1. **Decide the version.** Check `git tag` and the CHANGELOG's `[Unreleased]`
   section. Cut CHANGELOG.md: rename `[Unreleased]` to the new version with
   today's date, and start a fresh `[Unreleased]` section.

2. **Refresh the living docs.** FEATURES.md and ROADMAP.md must reflect the
   release: new features move to DONE, shipped TODO items are removed (not
   left behind), stale plans are deleted or re-dated.

3. **Run the full hermetic gate.**

       scripts/verify.sh

   `nix flake check` is the part raw `go test` cannot replace: it pins the
   vendor hashes and builds both modules with the flake's own toolchain. The
   vendor hashes drift when a module's dependency graph changes (go.mod
   require/replace/go.sum edits) — source-only edits do not re-drift them
   (verified 2026-08-29). If the check fails with a hash mismatch, copy the
   hash from the error message into `flake.nix` and re-run.

4. **Burn a fuzz budget before tagging.** CI fuzzes each target for only
   1m; a release deserves a longer soak. Run every target (root + ssetest)
   for 5m each:

       nix develop -c bash -c 'go test . -fuzz=FuzzWriteEvent -fuzztime=5m; \
         go test . -fuzz=FuzzParseEventID -fuzztime=5m; \
         go test . -fuzz=FuzzKeyedLines -fuzztime=5m; \
         cd ssetest && GOWORK=off go test . -fuzz=FuzzReadEvents -fuzztime=5m && \
         go test . -fuzz=FuzzWriteReadRoundTrip -fuzztime=5m && \
         go test . -fuzz=FuzzSplitSSELines -fuzztime=5m'

   Any crasher found here blocks the release; minimize it, add it to the
   committed corpus as a regression test, fix, and re-run.

5. **Validate the tag in a worktree before touching the remote.** Tags that
   fail `pkg.go.dev` verification are painful to retract — the module proxy
   caches forever (use `go mod retract` only as a last resort).

       git worktree add ../go-sse-release vX.Y.Z
       cd ../go-sse-release
       GOWORK=off GOEXPERIMENT=jsonv2 go build ./...
       GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1
       cd - && git worktree remove ../go-sse-release

6. **Tag locally, then push.** Use a signed tag when a signing key is
   configured, otherwise a plain annotated tag — but never plan to "sign
   later": the module proxy caches the tag object forever, and a re-tagged
   unsigned-then-signed version is indistinguishable from a retraction.

       # signed (preferred when git config user.signingkey is set):
       git tag -s vX.Y.Z -m "vX.Y.Z: <one-line summary>"
       # otherwise annotated:
       git tag -a vX.Y.Z -m "vX.Y.Z: <one-line summary>"
       # ssetest changes only:
       git tag -a ssetest/vX.Y.Z -m "ssetest vX.Y.Z: <one-line summary>"

       git push origin master --follow-tags

   Never `git push --force` a release tag; if the tag is wrong, delete it
   locally AND on the remote and re-tag **before** anything fetches it.

7. **Verify the module proxy picked it up** (a few minutes after push) with
   the scripted consumer probe — it checks the version index, downloads the
   zip, and builds a from-scratch consumer module against the tag, failing
   loudly at each step (the hand-run version of this was fumbled twice):

       scripts/release-verify.sh vX.Y.Z
       scripts/release-verify.sh ssetest/vX.Y.Z

8. **Publish the GitHub release.** Stage the notes from the CHANGELOG entry:

       gh release create vX.Y.Z --title "vX.Y.Z" --notes-from-tag

9. **Post-release.** Close or update issues referenced in the CHANGELOG;
   re-run the CI workflow on the release tag if it did not trigger.
   Deliberately refresh two pins that must never float via `@latest`:

   - **govulncheck** — if a new stable release exists, bump the pinned
     version in `.github/workflows/ci.yml` (`go install …govulncheck@vX.Y.Z`).
     Same reasoning as the golangci-lint pin: reproducible CI over
     convenience.
   - **go-datastar's `ssetest` pin** — after a `ssetest/vX.Y.Z` release,
     bump the pin in [go-datastar](https://github.com/LarsArtmann/go-datastar)
     and tag `datastartest` (the pairing rule the v0.6.0/v0.3.0 releases
     followed). This belongs HERE, in go-sse's post-release steps — go-datastar
     has no way to notice a new ssetest tag on its own.
