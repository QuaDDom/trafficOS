# 0003 — Redis for expiring current-state snapshots

## Context
High-frequency vehicle positions should not become permanent database rows every frame. Independent readers may need a recent world image.

## Decision
Store one complete current snapshot in Redis every second with a ten-second TTL. Keep authority in the single-writer Go process and durable facts in PostgreSQL. Do not use distributed locks when there is only one writer.

## Alternatives
Persist all frames in SQL; Redis as authoritative database; no cross-process cache.

## Consequences
The cache is disposable and can be up to a second old. Redis failure degrades readiness but cannot mutate or corrupt signal control. SQL sampled state has separate, explicit freshness semantics.
