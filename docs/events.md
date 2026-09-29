# Event contracts

```json
{
  "id": "evt_<epoch>_<sequence>",
  "type": "signal.phase.changed",
  "timestamp": "2026-09-20T00:00:00Z",
  "source": "signal-controller",
  "correlationId": "corr_<epoch>_<sequence>",
  "payload": { "intersectionId": "INT-05", "from": "NS_GREEN", "phase": "NS_YELLOW", "version": 2 }
}
```

The example illustrates the contract; it is not a recorded event or a performance result.

| Family | Emitted facts |
|---|---|
| Traffic | `traffic.vehicle.entered`, `traffic.vehicle.left`, `traffic.vehicle.turned`, `traffic.queue.threshold_exceeded` |
| Optimizer | `optimizer.calculation.started`, `optimizer.decision.created`, `optimizer.unavailable`, `optimizer.recovered` |
| Signals | `signal.phase.change.requested`, `signal.phase.changed`, `signal.transition.rejected`, `intersection.status.changed` |
| Emergency | `emergency.vehicle.detected`, `emergency.route.calculated`, `emergency.corridor.created`, `emergency.reservation.released`, `emergency.corridor.completed` |
| Incidents | `incident.created`, `incident.resolved` |
| System | `fallback.activated`, `adaptive_control.restored`, `system.node.status.changed`, `system.telemetry.latency.changed` |
| Simulation | `simulation.profile.changed`, `simulation.control.changed`, `simulation.reset`, `simulation.map.changed`, `simulation.priority.changed` |

At each optimizer evaluation, queues of four or more vehicles trigger a threshold observation. It is a repeated observation, not an edge-triggered alert. The threshold event, calculation start, decision and resulting phase request/change share a correlation ID. A HOLD or EXTEND result correctly has no immediate phase transition. Subsequent yellow/clearance transitions are separate timed operations.

Mutation requests use the generated HTTP request ID as their correlation ID. Long-lived corridor release/completion events use the corridor ID; the corridor-created payload links it back to the initiating request.

## NATS and delivery

Subject: `trafficos.events`. JetStream stream: `TRAFFICOS`. Stream retention: 24 hours or 128 MiB, whichever fills first. Duplicate window: ten minutes. PostgreSQL outbox entries are marked published after JetStream acknowledgement. A crash after acknowledgement but before marking can replay an ID.

The browser keeps 350 recent events and deduplicates by ID. This live window is not a complete audit export. The API keeps 1,500 events and 500 decisions. PostgreSQL is the durable source for audit records; the UI database view reads it directly through allowlisted queries.

## WebSocket

Connect to `/ws`. Messages are tagged envelopes:

```ts
type Message =
  | { type: 'snapshot'; payload: World; timestamp: string }
  | { type: 'event'; payload: DomainEvent };
```

Snapshot subscription and initial state are established under the controller lock. Each client has a bounded channel of 32 messages. The server pings every 15s and closes slow clients. The browser reconnects with jitter and a 0.5–15s exponential delay; a new connection starts with a complete snapshot. Events can lag a newer snapshot because the outbox is asynchronous.
