# Benchmark baselines

Checked-in `go test -bench` baselines for regression comparison with
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). One file per
module per recording date. Refresh procedure and the comparison workflow live
in [CONTRIBUTING.md](../../CONTRIBUTING.md#benchmark-regression-tracking).

| File                        | Module   | Recorded    | Hardware note        |
| --------------------------- | -------- | ----------- | -------------------- |
| `2026-09-29-root.txt`       | root `sse` | 2026-09-29 | Lars's Linux workstation (count=6) |
| `2026-09-29-sseparse.txt`   | `sseparse` | 2026-09-29 | same run as above    |

Numbers are machine-relative — compare only against baselines recorded on the
same hardware, and re-record rather than panicking at a cross-machine delta.
