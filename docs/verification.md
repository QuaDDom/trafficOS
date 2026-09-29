# Verification record

Executed locally on Windows, 19–20 September 2026. Node 24.13.1, Go 1.27.1 and a portable LLVM compiler were used. No Docker installation was available.

## Passed

- `go test ./...` and `go vet ./...`.
- `go test -race ./...` with CGO and Clang enabled.
- Real PostgreSQL 17.11 integration: `go test -tags=integration -v ./apps/api/internal/storage`. This exercised migration, audit/outbox persistence, decisions, idempotency, transactional rollback and replacement of current map geometry while retaining audit history against a local database on port 5433.
- Actual HTTP/WebSocket flow tests in the transport suite, including validation and propagation.
- `npm test`: eight tests covering formatting, bounded event deduplication, vehicle heading/selection, conflicting signal phases completed routes and variable-distance stop lines.
- `npm run build`: production compilation, TypeScript checking and static page generation.
- Browser checks against the running API, independent simulator and Next.js client: live vehicle motion and metrics, comparison, optimizer failure/recovery, request inspection and emergency creation.
- Urban redesign: boarding an active vehicle, passenger speed/signal/route instruments, exterior follow camera, completed-journey state, boarding another journey and Escape back to the map.
- Three-map revision: switched Centro → Avenida → Barrio in the compiled client, boarded a turning route on Avenida, checked automatic return to orbit on map change, inspected road surfaces and confirmed no captured browser console errors.
- Desktop, intermediate-width and 390px mobile layout inspection, including the map selector and passenger view. Captures are in `docs/screenshots/`.

Map tests additionally cover graph connectivity, route geometry staying within roads, monotonic path distances, traffic conservation, exclusive junction occupancy, turns completing on all three maps, emergency corridor release, stale-epoch telemetry and transactional map-switch rollback.

## Vehicle model revision

The procedural fleet now has sedan, hatchback, SUV and ambulance geometry. Production build and all eight frontend tests passed. A geometry check constructed every variant and verified finite vertex positions and bounding boxes. That revision used 31 instanced mesh batches; alloy wheel detail is restricted to vehicles within 60 metres of the camera. This is a geometry/render structure check, not a measured FPS benchmark. The compiled application was inspected in passenger and exterior follow views with live traffic; no browser console errors were captured.

## Scope of the running preview

The UI at `localhost:3000` uses explicit memory mode. Vehicle positions, signal phases and statistics come from the real Go controller through WebSocket snapshots. Audit history lasts only for the current process. PostgreSQL, Redis and NATS are correctly shown as disabled in this preview. The separate PostgreSQL integration test does not imply that the preview uses PostgreSQL.

## Not executed here

The complete Docker Compose deployment, Redis and NATS JetStream integration, and Grafana/Prometheus container startup have not been verified locally. The full tagged transport integration test is authored and compiled, but requires all three external services. CI is configured to run that path; no CI result is claimed in this record.

Benchmark measurements and their limits are documented separately in [benchmarks.md](benchmarks.md). Passenger mode follows curved routes and has a stylized cockpit overlay; it does not add manual steering, a physical collision solver or lane changes.

## Occupancy priority and documentation — 21 September 2026

- Added actual CAR/SUV/BUS arrivals with seeded, bounded occupancy. Ambulances retain emergency precedence. The expanded procedural fleet uses 38 instanced mesh batches; bus wheels and cameras use bus dimensions.
- `go test ./...`, `go test -race ./...` and `go vet ./...` passed. After consolidating immediate admission counts, simulation/application/transport race tests were rerun and passed.
- Tests cover the decision reversal between eight cars and one 40-person bus, a low-occupancy bus, signal safety limits, weighted waiting, person conservation, mixed fleet progress on all maps, emergency/fallback precedence, command validation, idempotency and rollback.
- `npm test` (8 tests) and the final production build passed. Browser checks confirmed mixed fleet counters, priority activation, suspension in fixed mode, successful manual creation of a 40-person bus and rejection at an occupied entry.
- The `/docs` guide was inspected at 390px and 1440px, including section navigation and signal explanations. The mobility panel was checked at 390px without horizontal overflow. No browser console errors were captured during this review.
- This is a simplified occupancy-weighted heuristic with synthetic passenger counts, not a calibrated deployment or an implementation of the complete OCC-MP algorithm. No network-wide performance improvement is asserted. Buses currently have no stops, timetables or boarding/alighting model.

## Brand identity — 21 September 2026

- Production build passed with the new `/brand` page and Open Graph image route.
- Verified 11 SVG/PNG pairs for dimensions and alpha channel; logo SVGs contain paths without text or external fonts. ZIP contents were checked against original files.
- Downloadable assets returned HTTP 200. Root metadata includes the SVG favicon, Apple icon and social cover.
- Inspected the brand page at 390px and 1440px, plus app and documentation headers at 390px. No horizontal overflow or captured browser console errors. Screenshot: `docs/screenshots/brand-desktop.png`.
- Canonical artwork and all exports regenerate with `npm run brand:build`; the frontend loads the same exported SVGs provided in the kit.

## City scale, traffic flow and readable UI — 23 September 2026

- Six maps: the new Ribera, Parque and Metrópolis have 25, 49 and 100 intersections. Tests cover connectivity, boundary entries, fleet conservation, signal safety and routes on all maps.
- Fixed exit clearance that mistook an opposite-lane bus for a blocked exit. Oriented vehicle footprints now include width, with a conservative distance rejection before the detailed overlap test. A seeded nine-minute mixed-fleet regression checks continued trip completion.
- Arrival validation accepts every boundary street of the selected map. Out-of-map arrivals remain rejected. Current-approach waits and a 40 m detection range drive lane demand. Reset retains experiment settings.
- Local event delivery batches a commit after its snapshot; regression tests cover a 100-intersection tick without subscriber overflow. Native NATS delivery remains individual-event based and was not integration-tested in this session.
- `go test ./... -timeout 300s`, `go test -race ./... -timeout 300s` and `go vet ./...` passed. The simulation suite took 38.4 s normally and 233.5 s with the race detector on this machine.
- `npm test` passed all 10 tests; the production build passed.
- UI now uses paper/asphalt contrast, larger readable labels, Spanish primary controls, an explanatory junction inspector, expandable map/fleet controls and a collapsed technical console. Removed the blocking introduction and icon-only rail. The inspector weights approach wait by detected vehicle count and hides obsolete adaptive decisions in other modes.
- Browser review caught and fixed duplicate React keys between city scenery and signals when changing to a large map. Metrópolis rendered successfully after correction, including focusing INT-100. Mobile layout and the priority panel were checked at 390 px with no horizontal overflow.
- Preview uses memory storage. No new claim is made about Docker integrations or real-world traffic calibration.
