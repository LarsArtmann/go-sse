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
    nix run .#test-race      # tests with race detector
    nix run .#lint           # golangci-lint
    nix run .#vet            # go vet
    nix run .#coverage       # test + coverage report
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
