# 0001 — Modular monolith and one authoritative writer

## Context
Nine intersections require consistent transitions and auditable operations. Splitting every domain into a service would introduce ordering, network failure and distributed transaction problems without a scaling requirement.

## Decision
Use Go + chi in a modular monolith. A single application mutex serializes commands; changes apply to a clone and publish only after audit commit. Keep signal safety, optimizer and world logic independently testable. Deploy the demand producer separately to exercise a real ingestion boundary.

## Alternatives
One microservice per intersection; Redis distributed locks; optimistic-only SQL updates; one giant HTTP handler containing the model.

## Consequences
Simple safety invariants and rollback. One writer instance per simulation; throughput is bounded by audit transaction latency. Future partitioning must establish explicit city ownership before scaling writers.
