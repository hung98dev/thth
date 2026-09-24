# Load Tests
status: LOCKED

## Scope
Defines production-like traffic models and pass thresholds for the 10,000+ CCU launch target.

Canonical thresholds: `../08_scale_ops/capacity.md`. Client-side frame, memory and load-time targets for the scenario 11 hotspot scene are canonical in `../04_architecture/client_performance.md`.

## Test Layers
### Component
Benchmark serialization, content lookup, collision/hit resolution, PostgreSQL transaction paths, and reward/economy operations.

### Partition
Exercise one simulation partition (one map channel) at:
- normal load,
- 17-player pre-threshold,
- 18-player hard cap (soft threshold and hard cap are now equal),
- public-boss/event density.


### Full-Service
Run Edge + simulation + durable domain + PostgreSQL with >=10,000 concurrent authenticated gameplay-equivalent clients.

## Player Mix
A full-service scenario must include a realistic mix rather than idle sockets only:
- normal field movement/combat,
- town/social idle/chat,
- dungeon/instance combat,
- public boss/event participants,
- crafting/inventory/equipment operations,
- Auction/shop reads/writes,
- reconnecting clients.

Exact percentages are test-scenario config and are recorded with results.

## Mandatory Scenarios
1. ramp 0 -> 10k CCU,
2. hold 10k for >=60 minutes,
3. soak representative peak for >=6 hours,
4. 18-player channel saturation (hard cap) with peak combat and multi-channel map load,
5. public boss participant burst,
6. simultaneous dungeon creation/completion burst,
7. reconnect storm,
8. PostgreSQL latency degradation,
9. daily/weekly scheduler boundary,
10. maintenance restart under load (ADR-0052): pass = zero lost or duplicated committed durable operations and >= 99% of drained clients reconnected within 5 minutes after the new process reports ready,
11. **canonical hotspot benchmark**: 42 `AI_CLASS_NAMED_MECHANIC` monsters plus **18 players** in sustained combat in one channel; pass criterion: p95 tick runtime < 35ms. A run with 42 passive-class monsters must not substitute for this scenario — the mechanic class specifically exercises AI budget; the test must record entity class distribution in results,
12. **Spirit Surge worst-case**: 3 concurrent Spirit Surge regions each on a fully populated (18-player) map channel; pass criterion: p95 tick < 35ms across all three affected partitions simultaneously; records per-partition tick distribution,
13. **AOI bandwidth measurement**: per-client bytes/sec with `MAX_ENTITIES_IN_AOI_PER_CLIENT = 40` saturated (all 40 slots occupied with moving, attacking entities); records p50/p95/p99 server-to-client bytes/sec per connection; result is required input for re-validating the `server -> client p95 <= 25 KiB/s` budget before the 10k gate (see `../08_scale_ops/capacity.md`).
14. **login queue**: offered load above `WORLD_CCU_CAP`; pass = no connected session is dropped for capacity, queued logins are admitted in FIFO order, and `SERVER_OVERLOADED.retry_after_ms` is honoured,
15. **map fill**: 540 players across the 30 channels of one map, then a 541st entry attempt; pass = entry rejected with `MAP_CAPACITY_FULL`, no channel exceeds 18,
16. **settlement burst**: 1,000 auction purchases and 1,000 Reward Claims within 60 s; pass = no duplicate/lost settlement, p95 commit < 150 ms,
17. **partition capacity measurement**: raise co-hosted partitions until p95 tick >= 35 ms; record `MAX_PARTITIONS_PER_PROCESS` with `measured_at` (`../08_scale_ops/capacity.md`),
18. **entity cap with transients**: hotspot channel filled to exactly 80 entities including projectiles/transient entities (ADR-0039); pass = p95 tick < 35 ms and the 81st spawn rejected.

## Pass Thresholds
At supported peak:
```text
tick p95 <35ms
tick p99 <50ms
command queue wait p95 <25ms
durable mutation p95 <=150ms
durable mutation p99 <=500ms
client->server p95 sustained <=5 KiB/s
server->client p95 sustained <=25 KiB/s  [placeholder — must be re-measured via scenario 13 before 10k gate; see capacity.md]
unbounded queue/goroutine/memory growth = none
authority duplication = zero
value duplication = zero
```

Release capacity also requires planned 30% headroom or demonstrated horizontal scaling capacity beyond expected peak.

## PostgreSQL Capture
Record:
- transactions/sec,
- pool wait/usage,
- query p95/p99,
- slow queries,
- row/lock waits,
- deadlocks/serialization retries,
- WAL/checkpoint/I/O,
- CPU/memory,
- table/index growth during soak.

## Go Runtime Capture
Record:
- CPU,
- heap/RSS,
- allocation rate,
- GC pause/time,
- goroutine count,
- queue depths,
- partition count/entities,
- tick phase timings.

Goroutine/heap counts must return to a stable band after client/instance churn.

## Network Capture
Record:
- active WSS,
- connect/handshake rate,
- bytes/sec,
- baseline/resync rate,
- reconnect rate,
- slow-consumer disconnects,
- per-message-family volume.

## Failure Criteria
Immediate failure:
- dual character/partition authority,
- duplicate reward/currency/item from retry,
- process OOM/panic loop,
- DB pool exhaustion without backpressure,
- sustained tick overrun,
- unbounded backlog,
- correctness degradation used to meet throughput.

## Reproducibility
Every run records:
- server/client load-tool build,
- content revision,
- DB schema version,
- infrastructure shape,
- dataset seed,
- traffic mix,
- test duration,
- fault injection schedule.

Performance claims without this context are not release evidence.

## Invariants
- 10k test means active gameplay-equivalent workload, not 10k idle sockets.
- Correctness is a load-test pass condition.
- 18-player channel hotspot is explicitly tested.
- Canonical hotspot benchmark (42 named-mechanic monsters + 18 players) is a mandatory scenario.
- Spirit Surge 3-region worst-case is a mandatory scenario.
- AOI bandwidth measurement scenario must run before the 10k gate; its result gates the server->client budget figure.
- Soak verifies memory/goroutine/DB stability.
- Production release keeps capacity headroom.

