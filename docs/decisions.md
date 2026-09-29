# Engineering decisions

| Decision | Why | Cost |
|---|---|---|
| Go + chi | Standard `net/http`, context cancellation, middleware and httptest interoperate directly | More explicit transport code than a large framework |
| Modular monolith | One domain transaction and ownership boundary; straightforward interview narrative | Cannot scale independent modules without extracting interfaces |
| Single-writer controller + versions | Eliminates signal races without distributed locks | One authority per city; database latency limits tick throughput |
| Separate demand producer | Demonstrates real ingestion, retries, sequencing and heartbeat independent of rendering | Vehicle motion stays inside the API rather than an external physics engine |
| PostgreSQL transactional outbox | Audit and publication intent commit together | Relay may duplicate deliveries and incurs additional writes |
| NATS JetStream | Simple durable event stream with message ID deduplication | Live UI subscription is ephemeral; SQL is durable history |
| Redis ephemeral snapshot | Disposable cross-process read cache with explicit TTL | Extra dependency; not required for controller correctness |
| Full snapshots at 5Hz | Simple reconnect and server authority for a small city | More bandwidth than delta encoding; capped world size |
| Deterministic pressure optimizer | Explainable, fast and reproducible; can expose every decision | Not calibrated or globally optimal |
| Native UI components and small hooks | No large UI/state library for a single control surface | Some accessibility and formatting primitives are maintained locally |
| Procedural city and instanced cars | No external art downloads; batches vehicle drawing | Buildings remain a simplified visual model |

The simulator and optimizer share demand generation code only for reproducible experiments. The UI never imports that generator. Browser animations interpolate received positions, not authoritative simulation physics.

An intentionally explicit memory mode is available for environments without Docker. It must never be confused with the production-storage path or described as PostgreSQL/NATS verification.
