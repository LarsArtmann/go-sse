# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

With Nix (recommended — provides Go 1.26, golangci-lint, gopls, govulncheck):

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

4. **Validate the tag in a worktree before touching the remote.** Tags that
   fail `pkg.go.dev` verification are painful to retract — the module proxy
   caches forever (use `go mod retract` only as a last resort).

       git worktree add ../go-sse-release vX.Y.Z
       cd ../go-sse-release
       GOWORK=off GOEXPERIMENT=jsonv2 go build ./...
       GOWORK=off GOEXPERIMENT=jsonv2 go test ./... -race -count=1
       cd - && git worktree remove ../go-sse-release

5. **Tag locally, then push.** Annotated tags for releases:

       git tag -a vX.Y.Z -m "vX.Y.Z: <one-line summary>"
       # ssetest changes only:
       git tag -a ssetest/vX.Y.Z -m "ssetest vX.Y.Z: <one-line summary>"

       git push origin master --follow-tags

   Never `git push --force` a release tag; if the tag is wrong, delete it
   locally AND on the remote and re-tag **before** anything fetches it.

6. **Verify the module proxy picked it up** (a few minutes after push):

       go list -m -versions github.com/larsartmann/go-sse
       go list -m -versions github.com/larsartmann/go-sse/ssetest
       GOPROXY=https://proxy.golang.org GOWORK=off GOEXPERIMENT=jsonv2 \
         go mod download github.com/larsartmann/go-sse@vX.Y.Z

7. **Publish the GitHub release.** Stage the notes from the CHANGELOG entry:

       gh release create vX.Y.Z --title "vX.Y.Z" --notes-from-tag

8. **Post-release.** Close or update issues referenced in the CHANGELOG;
   re-run the CI workflow on the release tag if it did not trigger.
