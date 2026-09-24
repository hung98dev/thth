# Sharding
status: LOCKED

## Scope
Defines runtime partitioning/routing and the launch durable-data sharding decision.

## Runtime Partitioning
Simulation scales horizontally by ownership partitions:
```text
normal world -> map_id + channel_id
instanced content -> instance_id
```

Exactly one simulation owner mutates each partition.

Partitions may move between simulation owners inside the single world process through explicit ownership/transfer rules (ADR-0044); entities are never concurrently active in two owners. At launch partitions never migrate across processes — a stopped world process recovers its partitions from PostgreSQL checkpoints on restart.

## Normal World Channels
Channels are capacity partitions of one logical world, not separate progression worlds.

Persistent character/economy/guild state is shared through the durable domain regardless of channel.

Public-boss generation semantics remain logical across channels where owning boss specs require it.

## Instance Placement
Dungeon/finale/PvP/Guild-War instances receive one in-process simulation owner for their runtime lifetime.

Placement considers:
- active entity/tick load,
- instance count,
- memory,
- locality constraints where needed.

Do not split one small launch instance across multiple simulation owners.

## Routing
Edge/session routing maintains:
```text
character_id -> session_epoch -> simulation_owner_id + ownership_epoch
```

Ownership epoch prevents stale source/destination packets from creating dual authority after transfer/reconnect.

## PostgreSQL
Launch uses one logical PostgreSQL database/cluster for canonical durable gameplay state.

Do **not** horizontally shard PostgreSQL at launch solely because service target is 10k CCU.

Before DB sharding, first use measured optimizations:
- correct indexes/query plans,
- bounded connection pools,
- transaction/lock reduction,
- read replicas for explicitly safe read-only workloads if later required,
- archival/partitioning of high-volume append-only history where appropriate.

A future durable-data shard key/migration requires an ADR because it affects transactions and cross-aggregate invariants.

## Hotspots
Runtime hotspot mitigation order:
1. enforce map channel hard caps,
2. create/route additional normal-map channels under world rules,
3. rebalance whole partitions between in-process simulation owners,
4. reduce non-critical replication cost,
5. admit new logins through the `WORLD_CCU_CAP` login queue (ADR-0052); no additional worlds.

Do not dynamically split a combat partition mid-tick.

## Cross-Partition Operations
Character transfer is an explicit handoff.

Cross-player persistent operations such as trade/Auction/guild mutation are resolved in the durable domain, not by directly locking two simulation processes together.

## Invariants
- Runtime shard = simulation ownership partition.
- Channels remain one logical world.
- One partition has one mutable owner.
- PostgreSQL is not sharded at launch.
- Cross-partition durable value flows through durable domain.
- Future DB sharding requires measured need + ADR.
