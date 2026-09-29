# 0004 — Native WebSocket with bounded subscribers

## Context
Rendering vehicle movement and live phase changes via repeated HTTP polling would add overhead and delay. Slow clients must not block safety control.

## Decision
Use native browser WebSocket and coder/websocket in Go. Send a full snapshot on connection, then snapshots at accepted telemetry cadence plus committed event messages. Use bounded buffers, deadlines, pings and exponential client reconnection with jitter.

## Alternatives
Polling; SSE with separate commands; a large socket abstraction and custom session protocol.

## Consequences
Simple full-state recovery for a capped city. Snapshots use more bandwidth than deltas. Event history can contain gaps during disconnection; durable audit is separately inspectable. Renderer interpolation is visual only.
