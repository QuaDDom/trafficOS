# Architecture and ownership

## Authority boundary

The independent simulator owns random demand generation. Its default seed is 42. Every 200ms it submits 1–10 small frames, according to the speed returned by the preceding API response. The packet is reused exactly on retry. Arrivals follow independent Poisson processes for twelve directional boundary lanes in Centro/Barrio and fourteen in Avenida; the central east/west avenue receives 1.5× demand. Rates are **per entry lane**, not city-wide: 100/400/900/1800 vehicles per hour for LOW/NORMAL/RUSH_HOUR/GRIDLOCK.

The API owns vehicle motion, positions, queues, controllers, incidents, emergency reservations, simulation time and measured counters. It accepts frame durations up to 0.21s, limits packet size and arrivals, and requires a private simulator bearer token. Control commands never rely on client validation.

World mutation is deterministic given an initial state and frame stream. Wall-clock timestamps, generated epoch identifiers and measured execution durations are deliberately excluded from claims of byte-for-byte deterministic audit output. Traffic outcomes are deterministic.

## Single-writer transaction

1. Parse and validate the transport envelope.
2. Acquire the application controller mutex.
3. Look up the idempotency key when supplied.
4. Clone the current world, apply domain operations, collect events and decisions.
5. Commit PostgreSQL audit, decision records, outbox records and response replay atomically.
6. Publish the new world reference and a WebSocket snapshot.
7. The outbox worker separately publishes to NATS and marks acknowledgements.

Database failure leaves the previous world and counters intact. No retry is attempted inside the domain critical section. The external simulator retries the same packet with exponential backoff. NATS publication is at-least-once; message IDs, JetStream deduplication and client event-ID deduplication handle replay. There is no exactly-once claim across the network.

Core signal safety remains synchronous; NATS carries committed facts for observers, not safety-critical commands. A slow WebSocket consumer gets disconnected after its bounded buffer fills and reconnects to a complete snapshot. WebSocket writes and pings have deadlines.

## Persistence

PostgreSQL tables: intersections, lanes, signal_states, signal_transitions, traffic_snapshots, optimization_decisions, events, incidents, emergency_vehicles, emergency_corridors, emergency_reservations, idempotency_keys, event_outbox and schema_migrations.

Stable relational identifiers and timestamps provide indexed lookups; JSONB preserves the full audit payload. There is no arbitrary SQL execution endpoint. Database exploration uses a hardcoded table allowlist and LIMIT 20.

Redis holds `trafficos:current`, refreshed every wall-clock second with a 10-second TTL. It is a disposable cache for secondary readers, never a lock service or an independent authority. The API's own reads use its consistent in-process world. SQL signal states and aggregate traffic snapshots are sampled every ten simulation seconds. Event/decision records and mutations commit immediately.

The API starts a new simulation epoch after restart. SQL audit survives; transient traffic does not resume. Completed emergency vehicle position rows retain their last recorded sample. Corridor/reservation completion is the authoritative emergency lifecycle record.

## Safety and traffic model

The shared map catalog defines nodes, connected street segments, boundary entries and building lots. Ordinary routes use breadth-first search to a seeded destination, with sampled quadratic curves for left/right turns. Movement enforces length-aware headway (half the combined vehicle lengths + 0.6m; 3.8m for two cars), gradual acceleration and distance-based braking. A conservative one-vehicle junction reservation prevents conflicting turns; cars wait upstream when the downstream exit is occupied. Red/yellow stops apply before entry, while committed vehicles clear the junction. Incidents reduce approach speed to 3m/s. This is a kinematic model without a full physical collision solver or lane changes.

Changing maps creates a fresh epoch through the same transactional command path. The producer receives updated dimensions and epoch; packets bearing an old epoch are acknowledged without advancing the new world. The renderer consumes the authoritative geometry and memoizes static scenery. Current SQL geometry is replaced at the checkpoint while historical events remain available.

Occupancy priority is optional (`priorityMode=PEOPLE`, default `VEHICLES`) and only affects adaptive decisions. Each axis pressure is `2*queuedPeople + personWait/8 + .25*(people-queuedPeople)`, where detection covers the next approach within 40m, excludes committed crossings and weights the waiting time on this approach by occupancy. Existing hysteresis (1.4× + 2), minimum green, maximum green and alternating phase sequence remain. Emergency override and safe/fixed control bypass the adaptive rule. Synthetic demand mixes 63% CAR, 25% SUV and 12% BUS; occupants include everyone on board and remain fixed for each journey. There are no bus schedules or stops.

Metrics conserve admitted/completed/active people, sum person-seconds of waiting, and divide by all admitted people including unfinished journeys. Comparisons reuse identical mixed arrivals and occupancies in both runs; the chosen priority only affects ADAPTIVE. Manual fleet injection validates category, occupancy and map entry, commits audit transactionally and supports idempotency.

The emergency scheduler computes ETA from route distance / free-flow emergency speed, continually revises windows while delayed, and releases after crossing. It never truncates mandatory safety phases. The arbitration comparator supports competing axes, though the public MVP spawns all ambulances along the central eastbound hospital route.

## Failure semantics

| Failure | Behavior |
|---|---|
| Optimizer disabled | Safe fixed cycles; audit fallback and recovery |
| Intersection offline | Local simulated safe controller continues; status reflects loss of supervisory connection |
| Artificial telemetry latency | Delays real requests; simulator's effective wall-clock rate decreases |
| PostgreSQL unavailable | Mutations rejected; no uncommitted state or events published |
| NATS unavailable after startup | Outbox remains pending; relay retries 0.1–5s exponentially; snapshots continue |
| Redis unavailable | Current-state cache stale/absent; simulation continues; readiness degrades |
| WebSocket failure | Exponential reconnect, jitter and full-state resynchronization |
| Simulator absent | No new world steps; health marks simulator offline after 3s |

Liveness checks the process. Readiness checks configured PostgreSQL, Redis and NATS. Simulator and optimizer are reported separately because their controlled failure does not make HTTP unreadable.
