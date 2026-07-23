# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

Run the following commands to set up your development environment:

    GOEXPERIMENT=jsonv2 go test ./... -race
    GOEXPERIMENT=jsonv2 go vet ./...
    golangci-lint run ./...

`GOEXPERIMENT=jsonv2` is required to build (transitive dependency via
`go-branded-id`). Without it, compilation fails.

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
