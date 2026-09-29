# HTTP and WebSocket API

Interactive documentation is served at `/api/docs`; the OpenAPI document is `/api/openapi.json`. All resource responses are JSON. Mutations accept a JSON object. Bodies above 128 KiB are rejected. CORS permits only configured origins. A per-IP token bucket permits a burst of 60 requests and refills at 20/s. Browser controls are intentionally unauthenticated in this local demo; simulator ingestion requires `Authorization: Bearer <SIMULATOR_TOKEN>`.

| Method | Path | Purpose / input |
|---|---|---|
| GET | `/health/live` | Process liveness |
| GET | `/health/ready`, `/api/v1/health` | Dependency probes, optimizer/simulator status, storage mode |
| GET | `/api/v1/snapshot` | Complete authoritative world |
| GET | `/api/v1/maps` | Available district graphs, dimensions and speed limits |
| GET | `/api/v1/intersections` | Selected map’s configured intersections |
| GET | `/api/v1/intersections/{id}` | Intersection, lanes, phase, mode, version and last decision |
| GET | `/api/v1/intersections/{id}/state` | Same authoritative intersection snapshot |
| GET | `/api/v1/simulation/config` | Profile, speed, pause, mapId, rows, columns and epoch for telemetry producers |
| GET | `/api/v1/traffic/metrics` | Real traffic counters, HTTP rolling p50/p95 and WS clients |
| GET | `/api/v1/events?correlationId=…` | Up to 1,500 recent committed events, optional trace filter |
| GET | `/api/v1/decisions` | Up to 500 recent decisions |
| GET | `/api/v1/decisions/{id}` | A complete recent decision |
| GET | `/api/v1/requests` | 200 recent HTTP requests, sanitized headers and 2 KiB previews |
| GET | `/api/v1/database/{table}` | Latest 20 persisted rows from the fixed allowlist |
| POST | `/api/v1/simulation/profile` | `{ "profile": "RUSH_HOUR" }` |
| POST | `/api/v1/simulation/control` | `{ "paused": false, "speed": 2, "mode": "ADAPTIVE", "priorityMode": "PEOPLE" }`; priorityMode VEHICLES (default) or PEOPLE |
| POST | `/api/v1/simulation/map` | `{ "mapId": "avenida" }`; centro/barrio/avenida/ribera/parque/metropolis, new epoch, retains profile/speed/mode/pause |
| POST | `/api/v1/simulation/reset` | `{}`; new ephemeral epoch on the selected map; retain profile, controller mode, speed, pause and priority settings, and durable audit |
| POST | `/api/v1/vehicles` | `{ "type": "BUS", "occupants": 40, "road": 1, "direction": "E" }`; CAR 1–4, SUV 1–6, BUS 1–60; occupied entries return 409 |
| POST | `/api/v1/emergency-vehicles` | `{ "priority": 2 }`; required `Idempotency-Key` |
| POST | `/api/v1/incidents` | `{ "intersectionId": "INT-05", "type": "ROADWORK" }` |
| POST | `/api/v1/incidents/resolve` | `{ "intersectionId": "incident_<id>" }`; field refers to incident ID for this command |
| POST | `/api/v1/signals/commands` | `{ "intersectionId": "INT-05", "phase": "EW_YELLOW", "version": 1 }` |
| POST | `/api/v1/chaos` | Optimizer, intersection or telemetry latency control |
| POST | `/api/v1/comparisons` | `{ "seed": 42, "duration": 180, "profile": "NORMAL", "mapId": "centro", "priorityMode": "PEOPLE" }` |
| POST | `/api/v1/telemetry` | Authenticated producer packet below |
| GET | `/metrics` | Prometheus exposition |
| WS | `/ws` | Initial snapshot, committed events and live snapshots |

## Error envelope

```json
{
  "error": {
    "code": "INVALID_SIGNAL_TRANSITION",
    "message": "unsafe_transition: NS_GREEN -> EW_GREEN",
    "requestId": "req_…"
  }
}
```

Every request receives `X-Request-ID`. Validation returns 400, missing entities 404, occupied emergency entry or stale versions 409, unsafe signal transitions 422, throttling 429, unavailable persistence 503. A request that fails storage does not become a successful world mutation.

## Idempotency

Use a fresh random key for a new operation and reuse exactly the same body bytes and key when retrying. Scope is command kind + key. A SHA-256 fingerprint covers kind and body bytes. Reordering JSON fields with the same key is considered a different body and returns 409. Keys currently have no automatic expiry; production operation should define retention.

## Telemetry contract

```json
{
  "session": "sim_unique_producer_id",
  "sequence": 1,
  "epoch": "current_epoch_from_config",
  "frames": [
    { "dt": 0.2, "arrivals": [{ "road": 1, "direction": "E" }] }
  ]
}
```

Arrivals optionally include `type` (CAR/SUV/BUS) and `occupants`. Missing type defaults to CAR; zero/missing occupancy uses 2 for CAR/SUV and 35 for BUS. Nonzero occupancy must respect category capacity. The bundled producer always supplies category and occupancy.

Road indices are bounded by the selected map: 0..rows-1 for E/W and 0..columns-1 for N/S (0..9 in Metrópolis). Directions are N/S/E/W. Frames per packet: 1–10. Frame duration: greater than zero and at most 0.21s. Arrivals per frame: at most 100. A session owns ingestion until its heartbeat is absent for three seconds. Same-session old sequence numbers return the current config without applying again. Gaps return 409. Another active session returns 409.

The response returns profile, speed, paused, epoch, mapId, rows and columns. Epoch is optional for legacy clients; the bundled producer sends it. Frames carrying a stale epoch are acknowledged without mutating the new map. Simulator speed affects generation of the next packet. Consequently profile/pause/speed changes can take one in-flight frame to take effect. An API restart allows a new session or the existing producer's next packet; the new epoch resets its arrival seed.

## Chaos examples

```json
{ "type": "optimizer", "online": false }
```
```json
{ "type": "intersection", "intersectionId": "INT-05", "online": false }
```
```json
{ "type": "latency", "latencyMs": 800 }
```

The latency option delays real ingestion, not fabricated request timings. It accepts 0–1000ms. Comparison duration accepts 30–600s and one concurrent comparison per API process.

### Realtime event batches

Without the native NATS relay, each committed update sends a `snapshot` first, then an `events` envelope with `payload: Event[]` and the current `epoch`. Clients should discard batches from other epochs and deduplicate event IDs. The NATS relay still sends individual `event` envelopes; the web client accepts both shapes. Batching prevents a normal 100-intersection decision tick from overflowing the bounded WebSocket message queue. Slow consumers still reconnect to a full snapshot.
