# Go vendor hashes for the two buildGoModule checks (flake.nix imports this).
#
# They are independent: the root module and `ssetest/` resolve different
# go.mod graphs (ssetest is its own module), so their vendored module sets —
# and therefore FOD hashes — differ. `sseparse/` has ZERO third-party
# requires (guarded by TestModuleStaysZeroDependency), so its check carries
# no vendorHash at all.
#
# Recompute after ANY go.mod/go.sum change or a change to the replaced parent
# tree: set the value to lib.fakeHash, run `nix build`, copy the hash
# `nix` reports. `buildflow -s nix-hash-fix --fix` does the same job.
{
  # `nix build .#checks.x86_64-linux.build`
  root = "sha256-t65CSNftAmmJ5UXk9T8Npz/iM2D4BxdjqovA3hLNfEo=";
  # `nix build .#checks.x86_64-linux.build-ssetest`
  ssetest = "sha256-EIHJSU3VhrS9fNEVH/1COR3vb92hWVsmaJRqbZ20Ew8=";
}
