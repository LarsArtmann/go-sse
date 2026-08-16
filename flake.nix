{
  description = "Server-Sent Events transport for Go (wire format, fan-out, replay)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      systems,
      treefmt-nix,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;
          buildGoModule = pkgs.buildGoModule.override { go = goPkg; };
          version = self.rev or self.dirtyRev or "dev";
          # Separate hashes: the two modules resolve different go.mod graphs
          # (ssetest replaces go-sse with a local path), so their vendored
          # module sets — and therefore FOD hashes — differ.
          vendorHash = "sha256-Gf8srGcQqteoGCUQSWcPrqZ+mSZKlmi8dkMobkkz464=";
          vendorHashSsetest = "sha256-TzgUuZw7DdKK4uSM/6wTU31yvMp8TyWtFp+1JP7l7Gg=";

          # go-sse is a pure library (no `main` package), so we do not publish a
          # binary `packages.default` or an overlay. Instead, buildGoModule is used
          # solely as a hermetic compile + test check: it vendors the public deps
          # (go-branded-id, go-error-family) via vendorHash, so `nix flake check`
          # and CI are fully reproducible with no network access.
          hermeticCheck = buildGoModule {
            pname = "go-sse";
            inherit version vendorHash;
            src = lib.fileset.toSource {
              root = ./.;
              fileset = lib.fileset.gitTracked ./.;
            };
            subPackages = [ "." ];
            proxyVendor = true;
            doCheck = true;
            env.GOEXPERIMENT = "jsonv2";

            meta = {
              description = "Server-Sent Events transport for Go";
              license = lib.licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
            };
          };

          # Same hermetic compile + test check for the separate ssetest/ module.
          # buildGoModule assumes the module at the source root, but ssetest is a
          # nested module with its own go.mod (`replace go-sse => ..`), so
          # preBuild cds into it and the package patterns resolve inside it. Its
          # vendored module set differs from the root module's, hence the
          # dedicated vendorHashSsetest.
          hermeticCheckSsetest = buildGoModule {
            pname = "go-sse-ssetest";
            inherit version;
            vendorHash = vendorHashSsetest;
            src = lib.fileset.toSource {
              root = ./.;
              fileset = lib.fileset.gitTracked ./.;
            };
            subPackages = [ "./..." ];
            proxyVendor = true;
            doCheck = true;
            env.GOEXPERIMENT = "jsonv2";
            preBuild = "cd ssetest";

            meta = {
              description = "Consumer test helpers for go-sse";
              license = lib.licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
            };
          };

          mkApp =
            name: runtimeInputs: text:
            let
              script = pkgs.writeShellApplication {
                inherit name runtimeInputs text;
              };
            in
            {
              type = "app";
              program = lib.getExe script;
              meta.description = "go-sse: ${name}";
            };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports.enable = true;
              golines.enable = true;
              nixfmt.enable = true;
            };
            settings = {
              formatter = {
                gofumpt.excludes = [ "*_templ.go" ];
                goimports.excludes = [ "*_templ.go" ];
                golines.excludes = [ "*_templ.go" ];
              };
            };
          };

          checks.format = config.treefmt.build.check self;
          checks.build = hermeticCheck;
          checks.build-ssetest = hermeticCheckSsetest;

          devShells.default = pkgs.mkShellNoCC {
            packages = [
              goPkg
              pkgs.golangci-lint
              pkgs.gopls
              pkgs.govulncheck
              pkgs.trash-cli
              pkgs.templ
            ];

            GOWORK = "off";
            GOTOOLCHAIN = "local";
            GOEXPERIMENT = "jsonv2";

            shellHook = ''
              echo "go-sse dev shell: $(go version)"
              echo "GOEXPERIMENT=$GOEXPERIMENT"
            '';
          };

          devShells.ci = pkgs.mkShellNoCC {
            packages = [
              goPkg
              pkgs.golangci-lint
            ];

            GOWORK = "off";
            GOTOOLCHAIN = "local";
            GOEXPERIMENT = "jsonv2";
          };

          apps = {
            test = mkApp "test" [ goPkg ] ''
              export GOEXPERIMENT=jsonv2
              go test ./... -count=1 "$@"
              (cd ssetest && GOWORK=off go test ./... -count=1)
            '';

            test-race = mkApp "test-race" [ goPkg ] ''
              export GOEXPERIMENT=jsonv2
              go test ./... -race -count=1 "$@"
              (cd ssetest && GOWORK=off go test ./... -race -count=1)
            '';

            build = mkApp "build" [ goPkg ] ''
              export GOEXPERIMENT=jsonv2
              go build ./...
              (cd ssetest && GOWORK=off go build ./...)
            '';

            vet = mkApp "vet" [ goPkg ] ''
              export GOEXPERIMENT=jsonv2
              go vet ./...
              (cd ssetest && GOWORK=off go vet ./...)
            '';

            lint = mkApp "lint" [ pkgs.golangci-lint ] ''
              export GOEXPERIMENT=jsonv2
              golangci-lint run ./...
              (cd ssetest && GOWORK=off golangci-lint run ./...)
            '';

            coverage = mkApp "coverage" [ goPkg ] ''
              export GOEXPERIMENT=jsonv2
              go test ./... -coverprofile=coverage.out -covermode=atomic "$@"
              go tool cover -func=coverage.out
              (cd ssetest && GOWORK=off go test ./... -coverprofile=../ssetest-coverage.out -covermode=atomic && go tool cover -func=../ssetest-coverage.out)
            '';

            coverage-gate = mkApp "coverage-gate" [ goPkg pkgs.bc ] ''
              export GOEXPERIMENT=jsonv2
              cov=$(go test . -count=1 -coverprofile=/tmp/sse-cov >/dev/null 2>&1 && go tool cover -func=/tmp/sse-cov | tail -1 | grep -oP '\d+\.\d+(?=%)')
              echo "library coverage: ''${cov}% (threshold: 90%)"
              if (( $(echo "$cov < 90" | bc -l) )); then
                echo "FAIL: library coverage ''${cov}% < 90%"
                exit 1
              fi
              echo "OK"
            '';

            clean =
              mkApp "clean"
                [
                  goPkg
                  pkgs.trash-cli
                ]
                ''
                  trash-put coverage.out 2>/dev/null || true
                  go clean -testcache
                '';
          };
        };
    };
}
