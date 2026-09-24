# Capacity
status: LOCKED

## Scope
Defines launch service capacity targets and release gates for the Unity + Go + PostgreSQL stack.

## Service Target
Launch architecture must support:
```text
10,000+ concurrent authenticated players
```
without changing gameplay authority or introducing a second logical world.

Admission (ADR-0052): `WORLD_CCU_CAP` (runtime config) is set to the CCU the 10k release gate measured on production hardware. A measured cap below 10,000 fails the release gate; the fix is performance work or hardware, never a second world. When attached sessions reach the cap, new logins enter the login queue (`../07_security/session.md` § Login Queue); connected players are never removed for capacity.

Normal map channels keep the canonical:
```text
channels_per_map         = 30
soft threshold / channel = 18 players
hard cap / channel       = 18 players   (reduced from 20; ADR-0035 resolved)
total map capacity       = 540 players  (30 × 18)
```
from world rules, ADR-0020, and ADR-0035. Instanced activities scale independently by instance partitions.

## Headroom
Normal production planning keeps at least:
```text
30% spare capacity
```
at the expected peak for Edge, Simulation, application workers, and PostgreSQL connection/compute budgets.

A service is not considered 10k-ready merely because it survives exactly 10,000 synthetic sockets with no gameplay load.

## Realtime SLOs
Server-side targets under supported peak load:
```text
simulation tick = 20 Hz / 50ms fixed step
tick runtime p95 < 35ms
tick runtime p99 < 50ms
simulation command queue wait p95 < 25ms
no sustained catch-up loop
```

Transport RTT is network/location dependent and is measured separately from server processing.

## Durable SLOs
For ordinary online PostgreSQL-backed gameplay mutations under supported peak:
```text
p95 <= 150ms
p99 <= 500ms
```
excluding intentionally long external/provider operations.

Value correctness/idempotency has priority over returning a false fast success.

## Bandwidth Budget
Release load tests target, per actively playing client:
```text
client -> server p95 <= 5 KiB/s sustained
server -> client p95 <= 25 KiB/s sustained
```

Short combat/spawn bursts may exceed these values. AOI and delta replication must prevent sustained map-wide fanout.

**Note:** The `server -> client p95 <= 25 KiB/s` figure predates the density increase and was not derived against `MAX_ENTITIES_IN_AOI_PER_CLIENT = 40`. It must be re-measured before the 10k CCU gate using a load scenario that saturates the AOI cap per client (see `../09_testing/load.md`). Until re-measured, the 25 KiB/s figure is a placeholder bound, not a validated release target.

## PostgreSQL Capacity
PostgreSQL connections are pooled by backend processes, never one connection per player.

Release test records:
- total active/idle pool connections,
- transaction throughput,
- lock wait/deadlock rate,
- slow-query distribution,
- WAL/checkpoint pressure,
- DB CPU/memory/I/O headroom.

The world process's configured pool maximum plus ops/migration tooling must remain below the database connection budget with reserved operational headroom.

## Process Capacity
Do not hard-code a players-per-process number; the single world process is sized by the measured `WORLD_CCU_CAP` and `MAX_PARTITIONS_PER_PROCESS` (ADR-0052).

Benchmark each deployable role on production-like hardware using:
- Edge connections,
- active simulation partitions/entities,
- world boss/event density,
- dungeon/PvP instances,
- durable transaction mix,
- replication bandwidth.

Partition placement inside the process uses measured safe capacity, not theoretical goroutine counts. There is no autoscaling of worlds (ADR-0052).

`MAX_PARTITIONS_PER_PROCESS` is a **benchmark-derived release gate**: the maximum number of World Simulation partitions that may be co-hosted in one Go process while satisfying the p95 tick SLO under sustained load. It must be measured before the 10k CCU gate on production-like hardware and recorded in deployment configuration. At 10k CCU with 18 players per channel there are approximately 556 active normal-map channels; all of them plus active instances run in the single world process (ADR-0052), so the release gate requires measured `MAX_PARTITIONS_PER_PROCESS >= 556 + peak concurrent instances`. `WORLD_CCU_CAP` is set from the measured result; logins above it wait in the login queue.

**CI enforcement:** The 10k load scenario fails if deployment configuration does not carry a numeric `MAX_PARTITIONS_PER_PROCESS` value together with a `measured_at` timestamp on production-like hardware. An absent or placeholder value is a test-blocking defect, not a warning. No interim ceiling is assumed; admission control must refuse to exceed an unset value rather than substituting infinity.

## Hotspot Tests

Release tests include:
- 18-player full channel (hard cap) with peak combat,
- **42 AI_CLASS_NAMED_MECHANIC monsters plus 18 players in sustained combat in one channel; p95 tick runtime must remain under 35ms** (this is the canonical entity capacity benchmark validating MAX_ENTITIES_PER_CHANNEL = 80 and the current spawn density; player count updated from 20 to match the revised hard cap),
- multi-channel map fill toward 540 players,
- public boss with maximum effective participants,
- Spirit Surge on populated map with all 3 concurrent regions active,
- simultaneous dungeon creation/completion burst,
- Auction/reward settlement burst,
- reconnect storm,
- daily/weekly boundary jobs.


## 10k Release Gate
A candidate is 10k-ready only when a production-like soak at >=10,000 CCU-equivalent:
- satisfies tick/durable SLOs,
- has >=30% planned headroom or documented safe scaling path,
- creates no unbounded queue/goroutine/memory growth,
- maintains PostgreSQL pool/lock health,
- preserves reward/economy idempotency,
- completes without authority duplication.

## Invariants
- 10k CCU is a measured release gate.
- Channel hard cap = 18; map hard cap = 540 (30 x 18).
- MAX_ENTITIES_PER_CHANNEL = 80 (release gate; benchmarked before 10k CCU gate).
- MAX_PARTITIONS_PER_PROCESS is benchmark-derived (release gate; must be measured and recorded before 10k CCU gate).
- No DB connection per player.
- No capacity optimization may weaken server authority.
- Headroom is planned, not consumed as normal operating target.

