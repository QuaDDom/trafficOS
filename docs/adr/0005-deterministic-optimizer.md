# 0005 — Explainable deterministic control before machine learning

## Context
The portfolio must expose why decisions happened and compare controllers honestly. There is no calibrated real-world training dataset.

## Decision
Use queue pressure, waiting pressure and incoming traffic with explicit minimum/maximum green constraints. Run FIXED and ADAPTIVE against an identical seeded Poisson demand stream. Persist the input, action, rule parameters and measured computation time.

## Alternatives
Reinforcement learning; hardcoded favorable comparisons; random phase timing.

## Consequences
Reproducible traffic outcomes and inspectable rules. The algorithm is not globally optimal and can underperform fixed cycles. Waiting metrics include unfinished vehicles; admission rejections remain visible. ML requires a separate evaluation and safety case before introduction.
