# Reproducible local benchmarks

These measurements predate the three-map routing and curved-turn implementation. They are historical baseline results, not performance measurements of the current traffic model. Re-run the commands below for current results.

Measured on **2026-09-19 (America/Buenos_Aires)** / 2026-09-20 UTC.

- CPU: AMD Ryzen 5 5500X3D, 6 physical cores / 12 logical processors.
- OS: Windows amd64.
- Runtime: Go 1.27.1, default GOMAXPROCS=12, benchmark bodies serial.
- The live development application was also running; these are local observations, not isolated laboratory claims.
- Raw output: [benchmark-results.txt](benchmark-results.txt).

| Benchmark | Iterations | Mean/op | Bytes/op | Allocations/op |
|---|---:|---:|---:|---:|
| Deterministic optimizer | 28,335,702 | 52.83 ns | 0 | 0 |
| World tick, NORMAL demand | 19,548 | 63,981 ns | 5,755 | 263 |
| Fixed + Adaptive, 180s | 8 | 143,146,850 ns | 9,226,713 | 493,927 |
| Domain event JSON encode | 459,130 | 2,442 ns | 480 | 5 |

The event encode benchmark produced approximately **409,500 encodings/sec** and 94.99 MB/s. This is one in-process JSON serialization kernel: **it excludes HTTP, simulation, PostgreSQL, the outbox, NATS and WebSocket delivery**. Its throughput cannot be used as an end-to-end events/sec claim. Benchmark loop timing provides mean latency, not p50/p95.

The world benchmark advances a changing populated simulation, clearing pending event/decision buffers between iterations. It includes domain event creation and lane measurement but excludes durable commit and transport. The comparison performs both modes from the same initial demand seed. Different environments and UI load can change wall-clock results without changing deterministic traffic outcomes.

```sh
go test '-run=^$' '-bench=.' -benchmem -count=1 \
  ./apps/api/internal/optimization \
  ./apps/api/internal/simulation \
  ./apps/api/internal/events
```

`make benchmark` runs the same benchmarks. Optimizer results are assigned to a package-level sink so the compiler cannot discard the operation.

## Not measured

The requested 10,000+ events/sec **durable end-to-end** target remains unverified. A proper future load test must specify event mix, database durability settings, outbox batch size, client count, message size, admitted/dropped traffic, p50/p95 end-to-end latency and machine configuration. The UI does not display the kernel result as system throughput.
