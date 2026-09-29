# 0002 — NATS JetStream with a transactional outbox

## Context
Publishing an event before a SQL commit exposes state that may roll back. Publishing after commit without recording intent can lose a message if the process crashes.

## Decision
Persist the event and outbox row in one PostgreSQL transaction. A bounded worker publishes to JetStream, acknowledges publication in SQL and retries with exponential backoff. Carry stable message and correlation IDs.

## Alternatives
Direct NATS publish; dual-write SQL/NATS; Kafka; database-only observation.

## Consequences
At-least-once delivery with deduplication, no exactly-once claim. The browser's ephemeral subscription can miss disconnected events; SQL remains durable audit. The controller does not depend on async event order for safety. Outbox retention is an explicit future operational task.
