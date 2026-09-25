# Task Queue
status: LOCKED

## Scope
Atomic implementation queue. A task becomes `DONE` only per `definition_of_done.md`; claim, review and merge follow `agent_execution_protocol.md` §5a.

Task state enum:
```text
NOT_STARTED
IN_PROGRESS
BLOCKED
DONE
```

Packets follow `../templates/task.md`; claim fields are written only by the coordinator. Gates are canonical in `audit_gates.md`; open contract conflicts in `known_blockers.md`.

- Bootstrap: a task may run before `IMP-068 = DONE` iff `IMP-068` is not in its transitive `depends_on` (`audit_gates.md` § Bootstrap Mode).
- Two-phase gate tasks: `IMP-000`, `IMP-061`, `IMP-003`, `IMP-004`, `IMP-005`, `IMP-065`, `IMP-068`.
- Path rules (owned paths, Unity test folders, shared registries, provenance fragments): `repository_layout.md` § Ownership Rules.
- Milestones: `milestones.md`; layers: `dependency_graph.md`; spec/ADR coverage: `spec_traceability.md`; execution waves: `wave_execution_prompts.md`. Packets below are grouped by domain only.

## Task Summary Table

| ID | Title | Status | Dependencies | Specs |
|---|---|---|---|---|
| `IMP-000` | M0 Bootstrap Gate & Toolchain Harness | `NOT_STARTED` | none | `../00_context/technology_versions.md`, `../00_context/constraints.md` |
| `IMP-001` | Stable IDs / Revisions | `NOT_STARTED` | IMP-000 | `../06_data/ids.md`, `../06_data/config.md` |
| `IMP-002` | Deterministic RNG Interface | `NOT_STARTED` | IMP-001 | `../04_architecture/concurrency.md`, `../06_data/config.md` |
| `IMP-003` | Content Compiler | `NOT_STARTED` | IMP-001, IMP-002 | `../01_gameplay/skills.md`, `../06_data/config.md` |
| `IMP-004` | Integration / Balance Activation Gate | `NOT_STARTED` | IMP-003 | `../01_gameplay/skills.md`, `../07_content/class_skill_catalog.md` |
| `IMP-005` | Operation Idempotency Primitive | `NOT_STARTED` | IMP-001 | `../06_data/database.md`, `../06_data/save_rules.md` |
| `IMP-006` | Account Auth, Session & Login Queue | `NOT_STARTED` | IMP-005, IMP-068, IMP-081, IMP-082, IMP-097 | `../04_architecture/authority.md`, `../06_data/data_model.md` |
| `IMP-007` | Currency Primitive | `NOT_STARTED` | IMP-005, IMP-068, IMP-082, IMP-097 | `../03_systems/README.md`, `../03_systems/economy.md` |
| `IMP-008` | Item Ownership Primitive | `NOT_STARTED` | IMP-005, IMP-068, IMP-082, IMP-097 | `../03_systems/items.md`, `../06_data/data_model.md` |
| `IMP-009` | Inventory / IAP Entitlement Panel | `NOT_STARTED` | IMP-008, IMP-066 | `../03_systems/inventory.md`, `../03_systems/account_storage.md` |
| `IMP-010` | Reward Claims | `NOT_STARTED` | IMP-005, IMP-007, IMP-008, IMP-009, IMP-011 | `../03_systems/reward_claims.md`, `../06_data/data_model.md` |
| `IMP-011` | Character Progression / Stats | `NOT_STARTED` | IMP-007, IMP-066, IMP-100 | `../01_gameplay/progression.md`, `../01_gameplay/stats.md` |
| `IMP-012` | Equipment / Loadout Core | `NOT_STARTED` | IMP-008, IMP-009, IMP-011 | `../03_systems/equipment.md`, `../07_content/equipment_catalog.md` |
| `IMP-013` | Movement / Collision / C2S_MOVEMENT_EDGE | `NOT_STARTED` | IMP-065, IMP-078, IMP-079, IMP-100 | `../01_gameplay/movement.md`, `../04_architecture/realtime_loop.md` |
| `IMP-014` | Combat Action State Machine | `NOT_STARTED` | IMP-011, IMP-013, IMP-081 | `../01_gameplay/combat.md`, `../01_gameplay/skills.md` |
| `IMP-015` | Skill Runtime / Geometry | `NOT_STARTED` | IMP-003, IMP-014 | `../01_gameplay/skills.md`, `../04_architecture/physics_geometry_contract.md` |
| `IMP-016` | Effects / Status / Shield Pipeline | `NOT_STARTED` | IMP-014, IMP-015 | `../01_gameplay/status_effects.md`, `../01_gameplay/combat.md` |
| `IMP-017` | Class Skills / Skill Levels | `NOT_STARTED` | IMP-015, IMP-016 | `../01_gameplay/classes.md`, `../07_content/class_skill_catalog.md` |
| `IMP-018` | Map / Transfer / Checkpoint Runtime | `NOT_STARTED` | IMP-013, IMP-066, IMP-100 | `../02_world/README.md`, `../02_world/maps_zones.md` |
| `IMP-019` | Spawn Runtime / Monster AI | `NOT_STARTED` | IMP-003, IMP-016, IMP-018 | `../02_world/spawning.md`, `../02_world/monsters.md` |
| `IMP-020` | Discovery / Progression Source Events | `NOT_STARTED` | IMP-005, IMP-011, IMP-018 | `../01_gameplay/README.md`, `../01_gameplay/core_loop.md` |
| `IMP-021` | Quest Runtime | `NOT_STARTED` | IMP-010, IMP-011, IMP-018, IMP-019 | `../02_world/quests.md`, `../07_content/quest_catalog.md` |
| `IMP-022` | Boss Runtime / WorldConsequence | `NOT_STARTED` | IMP-005, IMP-010, IMP-016, IMP-019 | `../02_world/bosses.md`, `../04_architecture/physics_geometry_contract.md` |
| `IMP-023` | Dungeon Runtime | `NOT_STARTED` | IMP-010, IMP-018, IMP-019, IMP-022 | `../02_world/dungeons.md`, `../04_architecture/physics_geometry_contract.md` |
| `IMP-024` | ENDGAME_L60 Variants | `NOT_STARTED` | IMP-023 | `../02_world/dungeons.md`, `../07_content/dungeon_catalog.md` |
| `IMP-025` | Spirit Surge | `NOT_STARTED` | IMP-010, IMP-019, IMP-021, IMP-080 | `../02_world/world_rules.md`, `../07_content/world_event_catalog.md` |
| `IMP-026` | Equipment Catalog Runtime Expansion | `NOT_STARTED` | IMP-003, IMP-012 | `../03_systems/equipment.md`, `../07_content/equipment_catalog.md` |
| `IMP-027` | Crafting / Enhancement | `NOT_STARTED` | IMP-007, IMP-008, IMP-009, IMP-026 | `../03_systems/crafting.md`, `../07_content/crafting_catalog.md` |
| `IMP-028` | NPC Services / Shops | `NOT_STARTED` | IMP-007, IMP-009, IMP-018, IMP-027 | `../02_world/npcs.md`, `../07_content/npc_shop_catalog.md` |
| `IMP-029` | Direct Trade | `NOT_STARTED` | IMP-007, IMP-008, IMP-009, IMP-011 | `../03_systems/trading_auction.md`, `../06_data/data_model.md` |
| `IMP-030` | Auction House | `NOT_STARTED` | IMP-005, IMP-007, IMP-008, IMP-009 | `../03_systems/trading_auction.md`, `../06_data/data_model.md` |
| `IMP-031` | Soul Contracts | `NOT_STARTED` | IMP-010, IMP-012, IMP-016 | `../03_systems/soul_contracts.md`, `../07_content/soul_catalog.md` |
| `IMP-032` | Spirit Meridian | `NOT_STARTED` | IMP-016, IMP-026 | `../03_systems/spirit_meridian.md`, `../07_content/build_catalog.md` |
| `IMP-033` | Formations | `NOT_STARTED` | IMP-016, IMP-026 | `../03_systems/formations.md`, `../07_content/build_catalog.md` |
| `IMP-034` | Friends / Block / Chat | `NOT_STARTED` | IMP-018, IMP-080, IMP-100 | `../03_systems/social.md`, `../05_network/messages.md` |
| `IMP-035` | Party | `NOT_STARTED` | IMP-018, IMP-080, IMP-100 | `../03_systems/party.md`, `../05_network/messages.md` |
| `IMP-036` | Guild Core / Progression | `NOT_STARTED` | IMP-007, IMP-034, IMP-100 | `../03_systems/guild.md`, `../03_systems/guild_progression.md` |
| `IMP-037` | Guild Storage | `NOT_STARTED` | IMP-008, IMP-009, IMP-036 | `../03_systems/guild_storage.md`, `../06_data/data_model.md` |
| `IMP-038` | Cosmetics | `NOT_STARTED` | IMP-007, IMP-008, IMP-036, IMP-100 | `../03_systems/cosmetics.md`, `../07_content/cosmetic_catalog.md` |
| `IMP-039` | PvP Build Snapshot / Transform | `NOT_STARTED` | IMP-012, IMP-017, IMP-031, IMP-032, IMP-033, IMP-057, IMP-058, IMP-059, IMP-060 | `../03_systems/pvp.md`, `../01_gameplay/combat.md` |
| `IMP-040` | Duel / Ranked Duel | `NOT_STARTED` | IMP-010, IMP-034, IMP-039 | `../03_systems/pvp.md`, `../05_network/messages.md` |
| `IMP-041` | Five Element Arena | `NOT_STARTED` | IMP-035, IMP-039, IMP-040 | `../03_systems/pvp.md`, `../05_network/messages.md` |
| `IMP-042` | Guild War | `NOT_STARTED` | IMP-010, IMP-036, IMP-039 | `../03_systems/guild_war.md`, `../05_network/messages.md` |
| `IMP-043` | Observability / Audit Coverage | `NOT_STARTED` | IMP-005, IMP-007, IMP-008, IMP-010, IMP-011, IMP-012, IMP-026, IMP-027, IMP-029, IMP-030, IMP-036, IMP-037, IMP-038, IMP-052, IMP-053, IMP-057, IMP-098, IMP-100 | `../08_scale_ops/observability.md`, `../08_scale_ops/caching.md` |
| `IMP-044` | Fault / Restart Matrix | `NOT_STARTED` | IMP-010, IMP-018, IMP-022, IMP-023, IMP-029, IMP-030, IMP-035, IMP-040, IMP-042, IMP-043, IMP-069 | `../09_testing/backend.md`, `../09_testing/strategy.md` |
| `IMP-045` | Security Abuse Suite | `NOT_STARTED` | IMP-013, IMP-014, IMP-018, IMP-028, IMP-029, IMP-030, IMP-034, IMP-035, IMP-036, IMP-040, IMP-053, IMP-069, IMP-100 | `../07_security/validation.md`, `../07_security/anti_cheat.md` |
| `IMP-046` | Load / Capacity Suite | `NOT_STARTED` | IMP-018, IMP-019, IMP-034, IMP-035, IMP-041, IMP-042, IMP-055, IMP-069 | `../08_scale_ops/capacity.md`, `../09_testing/load.md` |
| `IMP-047` | Migration / Backup / Restore Rehearsal | `NOT_STARTED` | IMP-005, IMP-043 | `../08_scale_ops/backup_recovery.md`, `../08_scale_ops/deployment.md` |
| `IMP-048` | Launch Candidate Gate | `NOT_STARTED` | IMP-044, IMP-045, IMP-046, IMP-047, IMP-096 | `definition_of_done.md`, `milestones.md` |
| `IMP-049` | TTK, Survivability, and Skill-Reach Re-verification | `NOT_STARTED` | IMP-004, IMP-026 | `../07_content/balance_validation.md`, `../07_content/class_skill_catalog.md` |
| `IMP-050` | Spirit Beast Passive Budget Compile Validation | `NOT_STARTED` | IMP-003, IMP-004 | `../03_systems/spirit_beasts.md`, `../07_content/spirit_beast_catalog.md` |
| `IMP-051` | Drop Table Coverage Invariant for Monster Roster Growth | `NOT_STARTED` | IMP-003, IMP-019 | `../07_content/drop_tables.md`, `../07_content/monster_catalog.md` |
| `IMP-052` | Seasons Infrastructure | `NOT_STARTED` | IMP-021, IMP-036, IMP-038, IMP-040, IMP-060, IMP-091, IMP-102 | `../03_systems/seasons.md`, `../03_systems/atlas.md` |
| `IMP-053` | IAP Receipt Verification & Entitlement Grants | `NOT_STARTED` | IMP-038, IMP-100 | `../03_systems/monetization.md`, `../03_systems/account_storage.md` |
| `IMP-054` | Anti-RMT Behavioral Signals | `NOT_STARTED` | IMP-001, IMP-007, IMP-029, IMP-030 | `../07_security/anti_cheat.md`, `../03_systems/trading_auction.md` |
| `IMP-055` | Entity Capacity Enforcement | `NOT_STARTED` | IMP-018, IMP-019, IMP-035 | `../02_world/maps_zones.md`, `../02_world/world_rules.md` |
| `IMP-056` | Retention & Erasure Engine | `NOT_STARTED` | IMP-043, IMP-094, IMP-100 | `../07_security/data_protection.md`, `../07_security/personal_data_register.md` |
| `IMP-057` | Linh Thú Companion Runtime | `NOT_STARTED` | IMP-008, IMP-010, IMP-016, IMP-019, IMP-100 | `../03_systems/spirit_beasts.md`, `../07_content/spirit_beast_catalog.md` |
| `IMP-058` | Folk Fishing Runtime | `NOT_STARTED` | IMP-002, IMP-008, IMP-010, IMP-018 | `../02_world/world_rules.md`, `../07_content/economy_catalog.md` |
| `IMP-059` | Hearth / Cooking / Bonfire Runtime | `NOT_STARTED` | IMP-003, IMP-007, IMP-008, IMP-018 | `../02_world/world_rules.md`, `../07_content/crafting_catalog.md` |
| `IMP-060` | Atlas Journal Runtime | `NOT_STARTED` | IMP-005, IMP-010, IMP-011, IMP-018 | `../03_systems/atlas.md`, `../07_content/atlas_catalog.md` |
| `IMP-061` | Protocol Buffers Schema & Multi-Language Codegen Harness | `NOT_STARTED` | IMP-000 | `../05_network/protocol.md`, `../05_network/messages.md` |
| `IMP-062` | Unity Geometry Exporter & Map Geometry Parity | `NOT_STARTED` | IMP-078, IMP-079 | `../01_gameplay/movement.md`, `../04_architecture/realtime_loop.md` |
| `IMP-063` | Addressables Asset Pipeline & Catalog Delivery | `NOT_STARTED` | IMP-000 | `../04_architecture/client_assets.md`, `../04_architecture/client.md` |
| `IMP-064` | Unity Bilingual Localization Pipeline (vi-VN / en-US) | `NOT_STARTED` | IMP-000 | `../04_architecture/client_localization.md`, `../06_data/text.md` |
| `IMP-065` | Unity Client Bootstrap, Session State & Network Transport | `NOT_STARTED` | IMP-061, IMP-100 | `../04_architecture/client.md`, `../04_architecture/client_experience_contract.md` |
| `IMP-066` | Unity Input Action Mapping & Core UI/HUD State Machine | `NOT_STARTED` | IMP-013, IMP-065 | `../04_architecture/client.md`, `../04_architecture/client_experience_contract.md` |
| `IMP-067` | Unity IL2CPP Player Build (Windows, Android) & Release Packaging | `NOT_STARTED` | IMP-020, IMP-024, IMP-025, IMP-028, IMP-041, IMP-042, IMP-076, IMP-084, IMP-085, IMP-086, IMP-087, IMP-088, IMP-089, IMP-090, IMP-093, IMP-099, IMP-103 | `../04_architecture/client.md`, `../04_architecture/client_assets.md` |
| `IMP-068` | Trusted CI, Post-Merge Guard & Foundation Exit | `NOT_STARTED` | IMP-004, IMP-005, IMP-061, IMP-063, IMP-064, IMP-083 | `audit_gates.md`, `agent_execution_protocol.md` |
| `IMP-069` | Server Composition Root and Lifecycle Wiring | `NOT_STARTED` | IMP-020, IMP-024, IMP-025, IMP-028, IMP-041, IMP-042, IMP-049, IMP-050, IMP-051, IMP-054, IMP-055, IMP-062, IMP-077, IMP-084, IMP-085, IMP-086, IMP-087, IMP-089, IMP-090, IMP-092, IMP-093, IMP-103 | `../04_architecture/backend.md`, `../04_architecture/service_boundaries.md` |
| `IMP-070` | Asset Provenance Register & Validator | `NOT_STARTED` | IMP-063, IMP-101 | `../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md` |
| `IMP-071` | Player Character & Class Art | `NOT_STARTED` | IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../01_gameplay/classes.md` |
| `IMP-072` | Normal-World Environment Art & Scenes | `NOT_STARTED` | IMP-062, IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../07_content/world_route_catalog.md` |
| `IMP-073` | UI, Item, Equipment & Skill VFX Art | `NOT_STARTED` | IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../07_content/class_skill_catalog.md` |
| `IMP-074` | Cosmetic Presentation Art | `NOT_STARTED` | IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../07_content/cosmetic_catalog.md` |
| `IMP-075` | SFX & Folklore BGM Production | `NOT_STARTED` | IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md` |
| `IMP-076` | Production Asset Coverage, Rights & Release Audit | `NOT_STARTED` | IMP-004, IMP-064, IMP-071, IMP-072, IMP-073, IMP-074, IMP-075, IMP-104, IMP-105 | `../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md` |
| `IMP-077` | Operator Admin API (auth, roles, two-person rule) | `NOT_STARTED` | IMP-006, IMP-043, IMP-094 | `../07_security/auth.md`, `../04_architecture/authority.md` |
| `IMP-078` | Server Geometry & Deterministic Collision Core | `NOT_STARTED` | IMP-003, IMP-068, IMP-098 | `../04_architecture/physics_geometry_contract.md`, `../01_gameplay/movement.md` |
| `IMP-079` | Sim Runtime: Fixed-Step Tick, AOI & Replication | `NOT_STARTED` | IMP-002, IMP-061, IMP-068, IMP-098 | `../04_architecture/realtime_loop.md`, `../04_architecture/concurrency.md` |
| `IMP-080` | Global Runtime (In-Process Single Writer) | `NOT_STARTED` | IMP-068, IMP-082, IMP-098 | `../04_architecture/service_boundaries.md`, `../04_architecture/concurrency.md` |
| `IMP-081` | Edge Listener, Framing & Heartbeat | `NOT_STARTED` | IMP-061, IMP-068, IMP-098 | `../05_network/protocol.md`, `../05_network/versioning.md` |
| `IMP-082` | Durable Command Queue & Backpressure | `NOT_STARTED` | IMP-005, IMP-068, IMP-098 | `../06_data/database.md`, `../06_data/save_rules.md` |
| `IMP-083` | Task-Graph & Architecture Conformance (Q0/Q4) | `NOT_STARTED` | IMP-000, IMP-061 | `architecture_conformance.md`, `repository_layout.md` |
| `IMP-084` | Death / Respawn | `NOT_STARTED` | IMP-014, IMP-016, IMP-018 | `../01_gameplay/death_respawn.md`, `../01_gameplay/combat.md` |
| `IMP-085` | Folklore Feats & Titles | `NOT_STARTED` | IMP-019, IMP-022, IMP-027, IMP-038, IMP-040, IMP-058 | `../03_systems/cosmetics.md`, `../07_content/cosmetic_catalog.md` |
| `IMP-086` | Chivalry Points & Titles | `NOT_STARTED` | IMP-023, IMP-034, IMP-038 | `../03_systems/social.md`, `../06_data/data_model.md` |
| `IMP-087` | Open Sparring Ring | `NOT_STARTED` | IMP-018, IMP-039 | `../03_systems/pvp.md`, `../02_world/world_rules.md` |
| `IMP-088` | Weapon Glow & Aura | `NOT_STARTED` | IMP-027, IMP-101 | `../03_systems/crafting.md`, `../04_architecture/client_assets.md` |
| `IMP-089` | Mystery Bounty | `NOT_STARTED` | IMP-021 | `../02_world/quests.md`, `../07_content/quest_catalog.md` |
| `IMP-090` | Progression Books | `NOT_STARTED` | IMP-009, IMP-011, IMP-021, IMP-023 | `../01_gameplay/progression.md`, `../03_systems/items.md` |
| `IMP-091` | Di Tích Relics & Boss Chest Ceremony | `NOT_STARTED` | IMP-010, IMP-022, IMP-023 | `../02_world/bosses.md`, `../06_data/data_model.md` |
| `IMP-092` | MA_AM Status | `NOT_STARTED` | IMP-016, IMP-022, IMP-057 | `../01_gameplay/status_effects.md`, `../01_gameplay/combat.md` |
| `IMP-093` | Guild Stone | `NOT_STARTED` | IMP-036, IMP-052, IMP-060 | `../03_systems/guild.md`, `../03_systems/seasons.md` |
| `IMP-094` | Chat Moderation & chat_messages | `NOT_STARTED` | IMP-034, IMP-080 | `../03_systems/social.md`, `../06_data/data_model.md` |
| `IMP-095` | Client Performance Budgets & Quality Presets (every PR) | `NOT_STARTED` | IMP-063, IMP-065, IMP-066, IMP-101 | `../04_architecture/client_performance.md`, `../04_architecture/client.md` |
| `IMP-096` | Android Device Performance on Firebase Test Lab | `NOT_STARTED` | IMP-067, IMP-095 | `../04_architecture/client_performance.md`, `audit_gates.md` |
| `IMP-097` | Aggregate Lock-Order Helper | `NOT_STARTED` | IMP-005, IMP-068, IMP-098 | `../06_data/database.md`, `../06_data/data_model.md` |
| `IMP-098` | Observability Core | `NOT_STARTED` | IMP-001, IMP-068 | `../08_scale_ops/observability.md`, `../04_architecture/backend.md` |
| `IMP-099` | Client Screens: Login, Queue, Loading, Settings, Credits | `NOT_STARTED` | IMP-064, IMP-065, IMP-066, IMP-095 | `../04_architecture/client_experience_contract.md`, `../04_architecture/client.md` |
| `IMP-100` | Character Lifecycle | `NOT_STARTED` | IMP-006 | `../01_gameplay/character.md`, `../06_data/data_model.md` |
| `IMP-101` | URP 2D Rendering & Lighting Setup | `NOT_STARTED` | IMP-000 | `../04_architecture/client.md`, `../04_architecture/client_assets.md` |
| `IMP-102` | Entitlement Claims & Store Client | `NOT_STARTED` | IMP-010, IMP-053, IMP-066 | `../03_systems/account_storage.md`, `../03_systems/monetization.md` |
| `IMP-103` | Account Deletion & Data Export API / Account UI | `NOT_STARTED` | IMP-056, IMP-066 | `../07_security/data_protection.md`, `../07_security/auth.md` |
| `IMP-104` | Monster, Boss & Spirit Beast Art | `NOT_STARTED` | IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../07_content/monster_catalog.md` |
| `IMP-105` | Dungeon, Finale & Competitive Environment Art | `NOT_STARTED` | IMP-062, IMP-063, IMP-070 | `../07_content/presentation_asset_manifest.md`, `../07_content/dungeon_catalog.md` |

## Topological Execution Order

```text
IMP-000 -> IMP-001 -> IMP-061 -> IMP-063 -> IMP-064 -> IMP-101 -> IMP-002 -> IMP-005 -> IMP-070 -> IMP-083 -> IMP-003 -> IMP-071
IMP-073 -> IMP-074 -> IMP-075 -> IMP-104 -> IMP-004 -> IMP-050 -> IMP-068 -> IMP-098 -> IMP-078 -> IMP-079 -> IMP-081 -> IMP-082
IMP-097 -> IMP-006 -> IMP-007 -> IMP-008 -> IMP-062 -> IMP-080 -> IMP-072 -> IMP-100 -> IMP-105 -> IMP-065 -> IMP-076 -> IMP-013
IMP-066 -> IMP-009 -> IMP-011 -> IMP-018 -> IMP-095 -> IMP-010 -> IMP-012 -> IMP-014 -> IMP-020 -> IMP-029 -> IMP-030 -> IMP-034
IMP-035 -> IMP-059 -> IMP-099 -> IMP-015 -> IMP-026 -> IMP-036 -> IMP-054 -> IMP-058 -> IMP-060 -> IMP-094 -> IMP-016 -> IMP-027
IMP-037 -> IMP-038 -> IMP-049 -> IMP-017 -> IMP-019 -> IMP-028 -> IMP-031 -> IMP-032 -> IMP-033 -> IMP-053 -> IMP-084 -> IMP-088
IMP-021 -> IMP-022 -> IMP-051 -> IMP-055 -> IMP-057 -> IMP-102 -> IMP-023 -> IMP-025 -> IMP-039 -> IMP-089 -> IMP-092 -> IMP-024
IMP-040 -> IMP-042 -> IMP-086 -> IMP-087 -> IMP-090 -> IMP-091 -> IMP-041 -> IMP-052 -> IMP-085 -> IMP-043 -> IMP-093 -> IMP-047
IMP-056 -> IMP-077 -> IMP-103 -> IMP-067 -> IMP-069 -> IMP-044 -> IMP-045 -> IMP-046 -> IMP-096 -> IMP-048
```

# Task Group — Contracts / Compiler

## `IMP-000` — M0 Bootstrap Gate & Toolchain Harness
id: IMP-000
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../00_context/technology_versions.md`, `../00_context/constraints.md`, `../00_context/glossary.md`, `../00_context/non_goals.md`, `../00_context/vision.md`, `../04_architecture/system_overview.md`, `../04_architecture/backend.md`, `repository_layout.md`, `architecture_conformance.md`, `../09_testing/test_and_release_evidence.md`, `audit_gates.md`, `agent_execution_protocol.md`, `engineering_conventions.md`]
adrs: [`0006-unity-go-postgresql-stack.md`, `0010-exact-technology-version-pinning.md`, `0040-world-consequence-durable-aggregate.md`, `0050-windows-only-ci-and-auto-merge.md`, `0052-single-launch-world.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: []
owned_paths: [`.editorconfig`, `.gitignore`, `.gitattributes`, `.github/pull_request_template.md`, `.github/workflows/verify.yml`, `scripts/verify.ps1`, `server/go.mod`, `server/go.sum`, `server/cmd/verify/`, `server/internal/conformance/gates/`, `server/internal/stackpin/`, `client/Packages/`, `client/ProjectSettings/`, `client/Assets/Plugins/Google.Protobuf/`, `client/Assets/Scripts/Core/ThinhThan.Core.asmdef`, `client/Assets/Scripts/Net/ThinhThan.Net.asmdef`, `client/Assets/Scripts/Systems/ThinhThan.Systems.asmdef`, `client/Assets/Scripts/UI/ThinhThan.UI.asmdef`, `client/Assets/Scripts/App/ThinhThan.App.asmdef`, `client/Assets/Tests/EditMode/ThinhThan.Tests.EditMode.asmdef`, `client/Assets/Tests/PlayMode/ThinhThan.Tests.PlayMode.asmdef`, `client/Assets/Tests/EditMode/AssemblyGraph/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [version matrix, docs-only baseline, repository layout]
contract_outputs: [native lockfiles, pinned project skeleton, Q0/Q1 verifier entrypoint]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Materialize the canonical versions from `../00_context/technology_versions.md` and the physical source tree from `repository_layout.md` into the native project lock points before dependent IMP tasks start.

## Acceptance
- materialized bootstrap paths strictly conform to `repository_layout.md`; `proto/`, migrations, generated outputs, and feature paths remain absent until their owning task,
- Unity `client/ProjectSettings/ProjectVersion.txt` (6000.6.1f1), `client/Packages/manifest.json`, and `client/Packages/packages-lock.json` match the matrix, including Addressables `2.11.2`,
- Go `server/go.mod` (module `thinhthan`, Go 1.27.1), `server/go.sum`, and CI use the pinned Go/direct-module versions,
- `scripts/verify.ps1` is PowerShell 7 (`pwsh` 7.6.6), runs unchanged on Linux and Windows, calls `server/cmd/verify` and accepts `-UnityResultsDir`; proto drift is owned by IMP-061,
- the seven asmdefs `ThinhThan.Core`, `ThinhThan.Net`, `ThinhThan.Systems`, `ThinhThan.UI`, `ThinhThan.App`, `ThinhThan.Tests.EditMode`, `ThinhThan.Tests.PlayMode` exist with acyclic references per `engineering_conventions.md` §2.2,
- `.github/pull_request_template.md` carries the PR report fields of `agent_execution_protocol.md` §4,
- `verify.yml` runs the Bootstrap Mode jobs `Q0-Q6 verify (Linux)` on `ubuntu-24.04` and `Q0-Q6 verify (Windows)` on `windows-2022` in parallel plus the `evidence manifest` job (ADR-0058); gates whose owner task is not `DONE` report `SKIP(bootstrap)` (`audit_gates.md`),
- every job's first step fails a fork PR with `external PRs not accepted` before checkout, cache or secrets (no job-level `if`); steps use `shell: pwsh`; Unity runs through the SHA-pinned GameCI actions in digest-pinned images with the licence from secrets and `client/Library` cached per OS,
- PostgreSQL 18.6: the Linux job uses the digest-pinned `postgres:18.6` service container, the Windows job the EDB binaries; both export `THINHTHAN_TEST_PG_DSN`; migrations and apply/down/apply remain owned by IMP-005/Q5,
- unlisted/floating/prerelease core dependency fails CI,
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/internal/stackpin/versions_test.go`: exact toolchain/dependency/action pins, image digests, runner labels (`ubuntu-24.04`, `windows-2022`; no `-latest`/self-hosted) and forbidden floating/unlisted dependencies.
- `server/internal/conformance/gates/workflow_test.go`: TestLinuxAndWindowsJobsRequired, TestForkGuardIsFirstStep, TestNoJobLevelIfOnRequiredJobs, TestSecretsOnlyAfterForkGuard.
- `server/internal/conformance/gates/gates_test.go`: initial Q0/Q1 wrapper-to-verifier wiring, bootstrap SKIP rules and fail-closed mutation fixtures.
- `client/Assets/Tests/EditMode/AssemblyGraph/AssemblyGraphTests.cs`: TestAsmdefReferencesAcyclic, TestProtocolReferencesNoProjectAssembly.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-000/"

## `IMP-001` — Stable IDs / Revisions
id: IMP-001
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../06_data/ids.md`, `../06_data/config.md`]
adrs: [`0001-content-revision-contract.md`, `0043-spirit-beast-instance-identity.md`]
depends_on: [IMP-000]
owned_paths: [`server/internal/core/id/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [stable-ID grammar, UUID rules, immutable content-grant namespace]
contract_outputs: [validated IDs, UUID v4/v5 primitives, content/schema revision values]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement stable entity/content/schema/operation IDs and content revision identity.

## Acceptance
- static IDs follow the canonical ASCII namespace rules from `../06_data/ids.md`,
- durable entity/operation IDs are server-generated RFC 4122 UUID v4 using `crypto/rand`,
- runtime entity IDs are owner/epoch-scoped uint64 values,
- revision included in compile/runtime diagnostic context,
- IDs are never inferred from display strings,
- malformed/zero UUID and conflicting operation-ID payload tests pass.

## Tests
- `server/internal/core/id/id_test.go`: `TestUUIDv4`, `TestUUIDv4Uniqueness`, `TestParseUUIDRejections`, `TestUUIDv5DeterministicContentGrant`, `TestValidateStaticContentID`, `TestRuntimeEntityID`, `TestContentRevisionDiagnosticContext`, `TestOperationPayloadConsistency`.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-001/"

## `IMP-002` — Deterministic RNG Interface
id: IMP-002
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/concurrency.md`, `../06_data/config.md`, `../09_testing/gameplay.md`]
adrs: [`0007-single-owner-fixed-step-simulation.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-001]
owned_paths: [`server/internal/core/rng/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [content revision, typed RNG context, server seed, ordered candidates]
contract_outputs: [deterministic PCG-64 streams and bounded weighted-selection results]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement server-owned deterministic seeded RNG abstraction for content rolls.

## Acceptance
- same revision+seed+inputs => same result,
- client cannot provide authoritative RNG result,
- gameplay RNG is Go `math/rand/v2` PCG-64; UUID/secrets remain `crypto/rand`,
- regression seed fixture exists.

## Tests
- `server/internal/core/rng/rng_test.go`: PCG-64 regression vector, same revision+seed+ordered inputs, independent context streams, invalid context inputs, weighted selection, and half-open range boundaries.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-002/"

## `IMP-003` — Content Compiler
id: IMP-003
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/skills.md`, `../06_data/config.md`, `../06_data/content_authoring_contract.md`, `../07_content/README.md`, `../07_content/item_catalog.md`, `../07_content/progression_route.md`, `../07_content/monster_catalog.md`, `../07_content/boss_catalog.md`, `../07_content/world_route_catalog.md`, `../07_content/dungeon_catalog.md`, `../07_content/class_skill_catalog.md`]
adrs: [`0001-content-revision-contract.md`, `0016-twelve-skill-pool-upgradeable-basics.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`, `0032-seven-channel-exp-source-portfolio.md`, `0033-skill-unlock-schedule-remap.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0047-skill-reach-budget-and-collider-aware-resolution.md`]
depends_on: [IMP-001, IMP-002]
owned_paths: [`server/cmd/compiler/`, `server/internal/config/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [24 Markdown catalogs, authoring schema, stable IDs, deterministic RNG, 45 primary skill geometries, typed secondary geometries]
contract_outputs: [CandidateSnapshot, content revision hash, compile diagnostics, compile report]
consumers_checked: [docs/01_gameplay/skills.md, docs/02_world/maps_zones.md, docs/02_world/monsters.md, docs/02_world/bosses.md, docs/02_world/dungeons.md, docs/03_systems/pvp.md, docs/03_systems/guild_war.md, docs/04_architecture/physics_geometry_contract.md, docs/06_data/config.md, docs/06_data/content_authoring_contract.md, docs/07_content/class_skill_catalog.md, docs/07_content/integration_validation.md, docs/09_testing/gameplay.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Compile static catalogs, finite expansions, references, enums, and shorthand.

## Acceptance
- all launch catalogs compile,
- 168 equipment IDs/52 portals/54 spawn groups expand deterministically,
- every monster/boss resolves exactly one size profile and every playable PvE space resolves exact bounds/layout profile,
- all 45 primary skill geometries and every secondary spatial effect compile from typed data,
- skill geometry values, tags, and effect shapes satisfy ADR-0047 before a snapshot is emitted,
- invalid reference rejects candidate.
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/cmd/compiler/compiler_test.go`: TestCompileAllCatalogs, TestEquipmentExpansion168, TestPortalExpansion52, TestEntitySizeProfileResolution, TestPlayableSpaceGeometryIndex, TestSkillGeometryRows45, TestSkillSecondaryGeometryCompile, TestSkillDisplacementTagConsistency, TestInvalidReferenceRejection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-003/"

## `IMP-004` — Integration / Balance Activation Gate
id: IMP-004
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/skills.md`, `../07_content/class_skill_catalog.md`, `../07_content/integration_validation.md`, `../07_content/balance_validation.md`, `../06_data/config.md`, `../06_data/content_authoring_contract.md`, `../07_content/progression_route.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0001-content-revision-contract.md`, `0016-twelve-skill-pool-upgradeable-basics.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`, `0032-seven-channel-exp-source-portfolio.md`, `0033-skill-unlock-schedule-remap.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0047-skill-reach-budget-and-collider-aware-resolution.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-003]
owned_paths: [`server/internal/config/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [CandidateSnapshot, integration rules, balance gates, prior active snapshot]
contract_outputs: [atomic activation decision, immutable active snapshot, rejection diagnostics]
consumers_checked: [docs/01_gameplay/skills.md, docs/02_world/maps_zones.md, docs/02_world/monsters.md, docs/02_world/bosses.md, docs/02_world/dungeons.md, docs/03_systems/pvp.md, docs/03_systems/guild_war.md, docs/04_architecture/physics_geometry_contract.md, docs/06_data/config.md, docs/06_data/content_authoring_contract.md, docs/07_content/class_skill_catalog.md, docs/07_content/integration_validation.md, docs/07_content/balance_validation.md, docs/09_testing/gameplay.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Execute `../07_content/integration_validation.md` + `balance_validation.md` before activation.

## Acceptance
- candidate activation is atomic,
- invalid revision leaves previous valid revision active,
- missing/mismatched size profile, bounds, topology, scene key or geometry export rejects activation,
- out-of-band skill reach, viewport envelope overflow, prose-only secondary geometry, or displacement tag/effect mismatch rejects activation,
- hard balance failure rejects activation.
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/internal/config/activation_test.go`: TestAtomicActivationLifecycle, TestInvalidRevisionPreservesPrevious, TestSpatialGeometryIntegrationRejection, TestSkillReachActivationRejection, TestSkillSecondaryGeometryRejection, TestSkillDisplacementTagRejection, TestHardBalanceFailureRejection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-004/"

## `IMP-061` — Protocol Buffers Schema & Multi-Language Codegen Harness
id: IMP-061
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../05_network/protocol.md`, `../05_network/messages.md`, `../05_network/errors.md`, `../05_network/protobuf_conventions.md`, `../05_network/synchronization.md`, `../05_network/versioning.md`, `repository_layout.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0050-windows-only-ci-and-auto-merge.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-000]
owned_paths: [`proto/thinhthan/v1/`, `proto/testdata/golden/`, `scripts/codegen.ps1`, `server/internal/protocol/v1/`, `server/internal/testing/protocol/`, `client/Assets/Scripts/Protocol/`, `client/Assets/Tests/EditMode/ProtocolParity/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [message registry, protocol/framing/version/error contracts, generator pins]
contract_outputs: [nine proto schemas, one canonical Go generated package, C# generated registry, binary parity fixtures, drift result]
consumers_checked: [AGENTS.md, docs/05_network/messages.md, docs/05_network/protobuf_conventions.md, docs/11_decisions/0008-client-network-transport-protocol.md, docs/10_implementation/repository_layout.md, docs/10_implementation/architecture_conformance.md, docs/10_implementation/engineering_conventions.md, server/internal/conformance/fences.go, server/internal/conformance/gates_test.go]

## Change
- Author canonical .proto definitions in proto/thinhthan/v1/ adhering to docs/05_network/messages.md and protocol conventions.
- Author the codegen script scripts/codegen.ps1 using pinned protoc 36.2 and protoc-gen-go v1.36.12.
- Target `server/internal/protocol/v1/` and `client/Assets/Scripts/Protocol/` with zero manual edits; never generate a duplicate flat Go package.
- Add codegen drift detection to verification harness.

## Acceptance
- Every message ID registered in `../05_network/messages.md` compiles deterministically across Go and C# targets,
- Regenerating protobuf output creates zero uncommitted git drift,
- Wire schema tests confirm one-to-one mapping between envelope and payloads.
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/internal/testing/protocol/registry_test.go`: TestMessageRegistryMapping, TestBinaryEncodingParity, TestCodegenDriftCheck.
- `client/Assets/Tests/EditMode/ProtocolParity/ProtocolParityTests.cs`: generated registry coverage and shared binary golden decode/encode parity.

generated_artifacts: [`server/internal/protocol/v1/*.pb.go`, `client/Assets/Scripts/Protocol/*.cs`]
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-061/"

## `IMP-062` — Unity Geometry Exporter & Map Geometry Parity
id: IMP-062
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/movement.md`, `../04_architecture/realtime_loop.md`, `../04_architecture/physics_geometry_contract.md`, `../03_systems/pvp.md`, `../03_systems/guild_war.md`, `../05_network/messages.md`, `../05_network/synchronization.md`, `../07_content/world_route_catalog.md`, `../07_content/dungeon_catalog.md`, `../09_testing/network.md`]
adrs: [`0005-skill-action-timing-geometry.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-078, IMP-079]
owned_paths: [`client/Assets/Scripts/Core/Geometry/`, `client/Assets/Tests/EditMode/GeometryExporter/`, `server/internal/sim/spatial/maps/`, `server/internal/sim/spatial/parity/`]
forbidden_paths: [`server/cmd/server/`, `server/internal/durable/`, `server/migrations/`]
contract_inputs: [tagged Unity geometry, space ID/kind, catalog bounds/layout profile, logical anchors, content revision, quantization contract]
contract_outputs: [deterministic geom JSON, Go collision data, cross-runtime golden vectors]
consumers_checked: [docs/02_world/maps_zones.md, docs/02_world/dungeons.md, docs/03_systems/pvp.md, docs/03_systems/guild_war.md, docs/06_data/config.md, docs/07_content/world_route_catalog.md, docs/07_content/dungeon_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement the Unity editor scene geometry exporter producing immutable `<space_id>.geom.json` in `server/internal/sim/spatial/maps/` using the IMP-078 schema.
- Validate every world/dungeon/finale/competitive space against exact bounds, topology and logical anchors.
- Deliver deterministic cross-language movement and collision golden vectors.

## Acceptance
- Exported map collision representation parses identically on server and client,
- Unity client prediction and Go authoritative simulation yield identical positions for golden input vectors,
- all declared playable spaces export with exact bounds and required layout topology; `1280x720` is never used as map bounds,
- PvP and Guild War mirror-parity checks pass within `0.001m`.

## Tests
- `server/internal/sim/spatial/parity/parity_test.go`: TestGeometryParity, TestAllPlayableSpaceBoundsProfiles, TestCompetitiveMirrorParity.
- `client/Assets/Tests/EditMode/GeometryExporter/GeometryExporterTests.cs`: quantization, stable export, invalid collider rejection, Go golden parity.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-062/"

## `IMP-063` — Addressables Asset Pipeline & Catalog Delivery
id: IMP-063
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client_assets.md`, `../04_architecture/client.md`, `../04_architecture/client_experience_contract.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/presentation_asset_manifest.md`, `../02_world/world_rules.md`, `repository_layout.md`, `../04_architecture/client_performance.md`]
adrs: [`0014-unity-addressables-asset-delivery.md`, `0035-spawn-density-increase.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0050-windows-only-ci-and-auto-merge.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-000]
owned_paths: [`client/Assets/AddressableAssetsData/`, `client/Assets/Scripts/Core/Assets/`, `client/Assets/Tests/EditMode/AddressablesValidation/`]
forbidden_paths: [`server/`]
contract_inputs: [presentation asset manifest, stable asset keys, Addressables pins]
contract_outputs: [built catalog/groups, key-resolution report, immutable bundle identity]
consumers_checked: [docs/04_architecture/client_assets.md, docs/04_architecture/client_experience_contract.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/presentation_asset_manifest.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Configure Unity Addressables 2.11.2 project settings, asset groups, and bundle delivery schemas.
- Implement asset manifest validation verifying stable keys for launch equipment, monsters, skills, and cosmetics.
- Author CI bundle build and catalog verification scripts.

## Acceptance
- Addressables groups build deterministically with pinned bundle settings,
- Missing or unmapped asset keys fail validation before candidate build,
- gameplay sprites validate 2x texture size, `100 PPU`, Bottom Center pivot, canonical cell/silhouette, compression class and Transform scale `(1,1,1)` (ADR-0055),
- every playable scene key resolves without requiring a monolithic map bitmap,
- Content catalog asset references resolve 100% against declared Addressables keys.

## Tests
- `client/Assets/Tests/EditMode/AddressablesValidation/AddressablesValidationTests.cs`: TestCatalogAssetKeyResolution, TestAddressableGroupBudgets, TestCanonicalSpriteImportProfiles, TestPlayableSceneKeyCoverage.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-063/"

## `IMP-101` — URP 2D Rendering & Lighting Setup
id: IMP-101
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client.md`, `../04_architecture/client_assets.md`, `../04_architecture/client_performance.md`, `../07_content/presentation_asset_manifest.md`, `../02_world/world_rules.md`]
adrs: [`0035-spawn-density-increase.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-000]
owned_paths: [`client/Assets/Settings/Rendering/`, `client/Assets/Scripts/Core/Rendering/`, `client/Assets/Tests/EditMode/RenderingSetup/`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [ADR-0056 lighting rules, quality preset budgets]
contract_outputs: [2D Renderer asset, Sprite-Lit materials, day/night driver, contact shadows]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Configure the URP 2D Renderer per ADR-0056: Sprite-Lit materials, per-map Global Light2D day/night driver, runtime contact-shadow component for every actor profile, point Light2D budget per quality preset (`client_performance.md`).

## Acceptance
- 2D Renderer asset and Sprite-Lit materials are the only gameplay sprite path,
- Global Light2D interpolates day/night per map,
- active point Light2D count never exceeds the preset budget (`LOW` 4, `MEDIUM` 8, `HIGH` 16),
- every actor profile has a contact shadow; UI imports at 200 PPU.

## Tests
- `client/Assets/Tests/EditMode/RenderingSetup/RenderingSetupTests.cs`: TestRendererAsset, TestSpriteLitMaterials, TestDayNightInterpolation, TestPresetLightBudget, TestContactShadowPerActorProfile, TestUiImport200Ppu.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-101/"

## `IMP-064` — Unity Bilingual Localization Pipeline (vi-VN / en-US)
id: IMP-064
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client_localization.md`, `../06_data/text.md`]
adrs: [`0015-unity-localization.md`]
depends_on: [IMP-000]
owned_paths: [`client/Assets/Localization/Settings/`, `client/Assets/Localization/Tables/Core/`, `client/Assets/Scripts/Core/Localization/`, `client/Assets/Tests/EditMode/LocalizationValidation/`]
forbidden_paths: [`server/`]
contract_inputs: [stable localization keys, vi-VN/en-US text, typed Smart String arguments]
contract_outputs: [bilingual string/asset tables, locale validation report]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Configure Unity Localization 1.5.12 with vi-VN (default) and en-US locales per ADR-0015.
- Ingest canonical localized strings from content catalogs into String Tables.
- Implement missing-translation gate rejecting incomplete locale entries.

## Acceptance
- All launch content keys exist in both vi-VN and en-US tables,
- Missing or fallback keys fail localization validation,
- Runtime locale switching preserves formatted currency and entity variables without allocation leaks.

## Tests
- `client/Assets/Tests/EditMode/LocalizationValidation/LocalizationValidationTests.cs`: TestBilingualKeyParity, TestNoMissingTranslations.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-064/"

## `IMP-083` — Task-Graph & Architecture Conformance (Q0/Q4)
id: IMP-083
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`architecture_conformance.md`, `repository_layout.md`, `audit_gates.md`, `task_queue.md`, `../templates/task.md`, `README.md`, `dependency_graph.md`, `spec_traceability.md`, `wave_execution_prompts.md`, `../README.md`, `../templates/adr.md`, `../templates/spec.md`]
adrs: [`0044-launch-topology-single-binary-role-modes.md`, `0050-windows-only-ci-and-auto-merge.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`]
depends_on: [IMP-000, IMP-061]
owned_paths: [`server/internal/conformance/taskgraph/`, `server/internal/conformance/architecture/`]
forbidden_paths: [`server/cmd/server/`, `server/internal/sim/`]
contract_inputs: [task packets, repository layout, spec Requirement IDs tables, Go import graph]
contract_outputs: [Q0 task-graph verdicts, Q4 architecture verdicts]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement Q0 task-graph checks (DAG, links, states, transitions, claim fields, owned/forbidden overlap ordered by `depends_on`, test paths inside owned paths, requirement-ID coverage, control-file diff rules) and Q4 fences (import direction, one production main, generated-code boundary, forbidden dependencies/schema) from `architecture_conformance.md`.

## Acceptance
- every rule has a failing mutation fixture,
- each requirement ID `[A-Z]{2,6}-\d{3}` in a spec "Requirement IDs" table appears in one packet's `## Acceptance` and the same packet's `## Tests`,
- import fences equal `architecture_conformance.md`; a forbidden import fails Q4.

## Tests
- `server/internal/conformance/taskgraph/taskgraph_test.go`: TestDagAcyclic, TestDanglingRefs, TestOwnedForbiddenOverlap, TestTestPathsOwned, TestRequirementIdCoverage, TestControlFileDiffRules.
- `server/internal/conformance/architecture/architecture_test.go`: TestImportDirection, TestOneProductionMain, TestForbiddenDependencies, TestGeneratedBoundary.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-083/"

## `IMP-068` — Trusted CI, Post-Merge Guard & Foundation Exit
id: IMP-068
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`audit_gates.md`, `agent_execution_protocol.md`, `known_blockers.md`, `../00_context/technology_versions.md`, `../07_security/external_integrations.md`, `../08_scale_ops/deployment.md`, `../09_testing/test_and_release_evidence.md`]
adrs: [`0010-exact-technology-version-pinning.md`, `0045-ci-evidence-without-self-referential-sha.md`, `0050-windows-only-ci-and-auto-merge.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-004, IMP-005, IMP-061, IMP-063, IMP-064, IMP-083]
owned_paths: [`.github/workflows/verify.yml`, `.github/workflows/post_merge_guard.yml`, `server/internal/conformance/ratchet/`, `server/internal/conformance/trusted/`]
forbidden_paths: [`server/cmd/server/`, `server/internal/sim/`, `server/internal/global/`, `server/internal/edge/`]
contract_inputs: [Owner Setup, Q0-Q6 implementations, DONE packets, GitHub API state]
contract_outputs: [trusted required check, post-merge guard, derived gate ratchet, Owner Setup evidence, runtime-unblock decision]
consumers_checked: [docs/10_implementation/README.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/task_queue.md]

## Change
- Read and evidence Owner Setup (public repository settings, outside-collaborator approval, `main` ruleset, both App installations, required secret names, Test Lab project) with `gh api`; never change repository settings.
- Finalize `verify.yml` as the trusted `pull_request_target` workflow (ADR-0058): jobs `Q0-Q6 verify (Linux)` (`ubuntu-24.04`) and `Q0-Q6 verify (Windows)` (`windows-2022`) each fail a fork PR in their first step, build the verifier from `main` and run it on the PR head checked out into a separate directory; no secret is exposed before the fork guard.
- Build the post-merge guard: both verify jobs on every push to `main`; a non-infrastructure failure makes the merge-guard App token open `revert/<sha>` and set dependents `BLOCKED`; a revert commit or infrastructure failure sets `AUTO_MERGE_FROZEN=true` and opens `OPS-xxx`.
- Derive the gate ratchet automatically on the base branch from the verifier gate list plus tests named in `DONE` packets.
- Verify, never resolve, contract blockers: an open `BLK-xxx` fails Gate A and leaves this task `BLOCKED`.

## Acceptance
- Gates A-D in `audit_gates.md` pass without required skips; Owner Setup JSON (repo, ruleset, App installations, secret names) is referenced by the evidence manifest,
- `policy-review` is required on every PR and accepted only from the App; no workflow job has that name,
- a PR that edits the verifier is still judged by the verifier built from `main`,
- removing a ratchet entry or adding a skip without an ADR already on `main` fails,
- a failing `main` push yields a revert PR; a failing revert or infrastructure failure freezes auto-merge instead,
- Unity EditMode executes in both OS jobs; PlayMode becomes required after IMP-065 (bootstrap rule),
- a fork PR fails both required checks before any checkout or secret use; the trusted workflow never checks out fork code with secrets,
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/internal/conformance/ratchet/ratchet_test.go`: TestRatchetDerivedFromDonePackets, TestRatchetDecreaseNeedsAdrOnMain, TestSkipWithoutAdrFails.
- `server/internal/conformance/trusted/trusted_test.go`: TestVerifierBuiltFromBase, TestOwnerSetupEvidenceSchema, TestGuardOpensRevertPr, TestRevertOrInfraFailureFreezes, TestOpenBlkFailsGateA, TestForkPrFailsBeforeCheckout, TestBothOsJobsRequired.

generated_artifacts: [`verify-report.json`]
cleanup_obligations: [Remove temporary codegen/build/migration workspaces; leave zero generated drift.]
evidence_location: "docs/10_implementation/evidence/IMP-068/"

# Task Group — Runtime Cores

## `IMP-098` — Observability Core
id: IMP-098
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../08_scale_ops/observability.md`, `../04_architecture/backend.md`]
adrs: [`0010-exact-technology-version-pinning.md`, `0040-world-consequence-durable-aggregate.md`]
depends_on: [IMP-001, IMP-068]
owned_paths: [`server/internal/observability/core/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/cmd/server/`]
contract_inputs: [operation/source/revision context, typed events]
contract_outputs: [structured logger, bounded metric registry, correlation context]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `log/slog` structured logging, a bounded-cardinality metrics registry, correlation context (operation/source/revision/retry) and secret redaction with non-blocking sinks. IMP-043 builds audit coverage and dashboards on top.

## Acceptance
- correlation context propagates operation/source/revision across calls,
- secrets and raw credentials are redacted,
- metric registration rejects unbounded label sets,
- a failing sink never blocks or changes the caller.

## Tests
- `server/internal/observability/core/core_test.go`: TestCorrelationContext, TestSecretRedaction, TestBoundedCardinality, TestSinkFailureNonBlocking.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-098/"

## `IMP-082` — Durable Command Queue & Backpressure
id: IMP-082
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../06_data/database.md`, `../06_data/save_rules.md`, `../04_architecture/concurrency.md`, `../08_scale_ops/capacity.md`]
adrs: [`0011-postgresql-relational-persistence.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-005, IMP-068, IMP-098]
owned_paths: [`server/internal/durable/queue/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/sim/`]
contract_inputs: [typed durable commands with operation ID and aggregate key]
contract_outputs: [ordered commit results, backpressure errors]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the bounded durable command queue between Sim/Global/Edge and PostgreSQL: per-aggregate ordering, ack after commit, typed backpressure, retry with the same operation ID (IMP-005).

## Acceptance
- commands for one aggregate key commit in submission order,
- a full queue returns a typed backpressure error; callers never block the simulation tick,
- ack is sent only after commit; a crash retry reuses the operation ID and settles once,
- queue/pool limits come from `capacity.md`.

## Tests
- `server/internal/durable/queue/queue_test.go`: TestPerAggregateOrdering, TestBackpressureWhenFull, TestAckAfterCommit, TestCrashRetrySettlesOnce.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-082/"

## `IMP-097` — Aggregate Lock-Order Helper
id: IMP-097
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../06_data/database.md`, `../06_data/data_model.md`]
adrs: [`0040-world-consequence-durable-aggregate.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-005, IMP-068, IMP-098]
owned_paths: [`server/internal/durable/lockorder/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/sim/`]
contract_inputs: [aggregate lock order from database.md]
contract_outputs: [ordered lock acquisition helper]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the helper every multi-aggregate transaction uses to lock rows in the canonical aggregate order of `../06_data/database.md`.

## Acceptance
- multi-aggregate transactions lock in `database.md` order with a deterministic key sort inside one aggregate type,
- out-of-order acquisition fails with a typed error in test builds,
- concurrent transfer fixture runs 1,000 iterations on PostgreSQL 18.6 without deadlock.

## Tests
- `server/internal/durable/lockorder/lockorder_test.go`: TestCanonicalOrder, TestOutOfOrderRejected, TestNoDeadlockConcurrentTransfers.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-097/"

## `IMP-081` — Edge Listener, Framing & Heartbeat
id: IMP-081
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../05_network/protocol.md`, `../05_network/versioning.md`, `../05_network/errors.md`, `../07_security/rate_limits.md`, `../04_architecture/service_boundaries.md`, `../00_context/technology_versions.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0038-discrete-movement-edge-input-message.md`, `0044-launch-topology-single-binary-role-modes.md`, `0051-first-party-username-password-login.md`, `0053-durable-contract-reconciliation.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-061, IMP-068, IMP-098]
owned_paths: [`server/internal/edge/listener/`, `server/internal/edge/heartbeat/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/sim/`, `server/internal/durable/`]
contract_inputs: [TLS/WSS connections, envelope frames, protocol version]
contract_outputs: [validated frames, per-session RTT samples, disconnect events]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the HTTPS/WSS listener (`github.com/coder/websocket` at the pinned version), envelope framing, protocol-version handshake, size/rate limits, drain, and the application heartbeat of `protocol.md` § Heartbeat (5 s interval, lost after 15 s) with per-session RTT samples consumed by IMP-014.

## Acceptance
- `C2S_HEARTBEAT`/`S2C_HEARTBEAT` follow `protocol.md`; a connection without valid traffic for 15 s is lost,
- RTT sample = server receive time of `C2S_HEARTBEAT` - `echo_server_ms` (when non-zero), published per session,
- oversize, unknown or incompatible-version frames are rejected with the `errors.md` code,
- the listener never mutates gameplay state; drain stops accepting before shutdown.

## Tests
- `server/internal/edge/listener/listener_test.go`: TestVersionHandshake, TestFrameSizeLimit, TestUnknownMessageRejected, TestDrainClosesAccept.
- `server/internal/edge/heartbeat/heartbeat_test.go`: TestHeartbeatTimeout15s, TestRttSampleFromEcho.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-081/"

## `IMP-080` — Global Runtime (In-Process Single Writer)
id: IMP-080
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/service_boundaries.md`, `../04_architecture/concurrency.md`, `../04_architecture/system_overview.md`, `../04_architecture/backend.md`]
adrs: [`0040-world-consequence-durable-aggregate.md`, `0044-launch-topology-single-binary-role-modes.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-068, IMP-082, IMP-098]
owned_paths: [`server/internal/global/runtime/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/sim/`]
contract_inputs: [typed Global commands, Durable interfaces]
contract_outputs: [serialized Global state transitions, restart rebuild]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the in-process single-writer Global actor (party, WORLD chat, matchmaking, Spirit Surge, boss generation hosts) with a bounded mailbox and typed Durable interfaces.

## Acceptance
- exactly one writer goroutine; every mutation passes the mailbox,
- a full mailbox returns backpressure and never blocks simulation ticks,
- restart rebuilds Global state from Durable only; no Redis/Kafka/NATS or `global_leader_lease`.

## Tests
- `server/internal/global/runtime/runtime_test.go`: TestSingleWriterSerialization, TestMailboxBackpressure, TestRestartRebuildFromDurable.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-080/"

## `IMP-079` — Sim Runtime: Fixed-Step Tick, AOI & Replication
id: IMP-079
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/realtime_loop.md`, `../04_architecture/concurrency.md`, `../04_architecture/authority.md`, `../05_network/synchronization.md`, `../08_scale_ops/capacity.md`]
adrs: [`0007-single-owner-fixed-step-simulation.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-002, IMP-061, IMP-068, IMP-098]
owned_paths: [`server/internal/sim/runtime/`, `server/internal/sim/aoi/`, `server/internal/sim/replication/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/durable/`, `server/internal/edge/`]
contract_inputs: [validated intents, seeded RNG streams, content revision]
contract_outputs: [ordered tick phases, AOI interest sets, snapshot/delta stream]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the single-owner 20 Hz map-instance actor (input queue, ordered tick phases, RNG streams, tick-budget metrics), the AOI grid and the snapshot/delta replication of `synchronization.md`, exposing typed ports to Edge and Durable without importing them.

## Acceptance
- fixed 50 ms step; p95 tick < 35 ms on the capacity fixture; an overrun is measured and never skips ordered phases,
- one goroutine owns each map instance; inputs apply in (tick, receive order),
- AOI enter/leave and snapshot/delta sequences are identical for a seeded replay,
- sim imports no pgx/SQL or `edge` package (Q4).

## Tests
- `server/internal/sim/runtime/runtime_test.go`: TestFixedStepTickOrder, TestSingleOwnerMailbox, TestTickOverrunAccounting, TestSeededReplayDeterminism.
- `server/internal/sim/aoi/aoi_test.go`: TestAoiEnterLeave, TestInterestSetBoundaries.
- `server/internal/sim/replication/replication_test.go`: TestSnapshotDeltaSequence, TestDeliveryClassRouting.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-079/"

## `IMP-078` — Server Geometry & Deterministic Collision Core
id: IMP-078
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/physics_geometry_contract.md`, `../01_gameplay/movement.md`, `../04_architecture/realtime_loop.md`, `../06_data/config.md`]
adrs: [`0005-skill-action-timing-geometry.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-003, IMP-068, IMP-098]
owned_paths: [`server/internal/sim/spatial/geometry/`, `server/internal/sim/spatial/collision/`]
forbidden_paths: [`server/migrations/`, `client/`, `server/internal/durable/`]
contract_inputs: [geom JSON schema, quantization contract, compiled bounds/layout profiles]
contract_outputs: [immutable parsed geometry, deterministic collision queries]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the `<space_id>.geom.json` schema, parser and types plus deterministic collision queries (sweeps, ground, slopes, steps, one-way ledges, walls, epsilon quantization) per `physics_geometry_contract.md`. IMP-013 movement and IMP-062 exporter parity consume it.

## Acceptance
- malformed, unquantized or out-of-bounds geometry is rejected with a path-specific error,
- collision queries are bitwise repeatable over 10,000 fuzzed vectors,
- slopes, steps and one-way drops follow the contract transitions,
- `1280x720` is never used as map bounds.

## Tests
- `server/internal/sim/spatial/geometry/geometry_test.go`: TestParseValidGeometry, TestRejectMalformedGeometry, TestQuantizationEpsilon.
- `server/internal/sim/spatial/collision/collision_test.go`: TestSlopeStepTransitions, TestOneWayLedgeDrop, TestDeterministicSweepVectors.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-078/"

# Task Group — Persistence / Core Character

## `IMP-005` — Operation Idempotency Primitive
id: IMP-005
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../06_data/database.md`, `../06_data/save_rules.md`, `../06_data/data_model.md`, `../06_data/migrations.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0011-postgresql-relational-persistence.md`, `0040-world-consequence-durable-aggregate.md`, `0048-character-update-timestamp.md`, `0053-durable-contract-reconciliation.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-001]
owned_paths: [`server/internal/durable/idempotency/`, `server/internal/durable/db/`, `server/internal/durable/schema/`, `server/migrations/`, `server/cmd/migrate/`, `server/internal/testing/pgtest/`]
forbidden_paths: [`server/internal/sim/`, `client/Assets/Scripts/`]
contract_inputs: [operation ID, canonical payload hash, transaction callback, schema contract]
contract_outputs: [single committed operation record, replayable result, baseline migrations]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement PostgreSQL-backed stable operation dedupe/result replay under `../06_data/database.md` and `save_rules.md`, the baseline migration `server/migrations/000001_baseline.{up,down}.sql` with the complete launch schema of `../06_data/physical_schema_contract.md` (ADR-0048, ADR-0053), and the PostgreSQL 18.6 test harness (`server/internal/testing/pgtest/`) that uses `THINHTHAN_TEST_PG_DSN` when set (Linux CI `postgres:18.6` service container) and otherwise starts the EDB Windows binaries and exports it (ADR-0058).

IMP-005 is the only migration owner. No other packet adds a migration; a later schema change is a spec change whose new numbered pair is owned by a new packet.

## Acceptance
- duplicate/retry/restart settlement commits at most once,
- commit-before-response retry reconstructs the same outcome,
- same operation ID with conflicting payload is rejected,
- real PostgreSQL UNIQUE/transaction behavior is covered.
- baseline 000001 applies, rolls back and re-applies on PostgreSQL 18.6; committed migration files are immutable (hash-checked),
- per-constraint schema snapshot: every table, column, key, check, foreign key and index of `physical_schema_contract.md` matches the migrated catalog, one assertion per constraint,
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `server/internal/durable/idempotency/idempotency_test.go`: TestOperationDeduplication, TestCommitBeforeResponseRetry, TestConflictingPayloadRejection, TestPostgresUniqueConstraint.
- `server/internal/durable/schema/schema_snapshot_test.go`: TestBaselineApplyDownApply, TestPerConstraintSnapshot, TestMigrationsImmutable.
- `server/internal/testing/pgtest/pgtest_test.go`: TestUsesPresetDsn, TestStartsEdbBinariesWhenDsnUnset.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-005/"

## `IMP-006` — Account Auth, Session & Login Queue
id: IMP-006
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/authority.md`, `../06_data/data_model.md`, `../07_security/auth.md`, `../07_security/session.md`, `../07_security/external_integrations.md`, `../07_security/rate_limits.md`, `../05_network/errors.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0009-account-session-credentials.md`, `0030-one-account-one-live-session.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-005, IMP-068, IMP-081, IMP-082, IMP-097]
owned_paths: [`server/cmd/server/`, `server/internal/durable/account/`, `server/internal/edge/auth/`, `server/internal/edge/session/`, `server/internal/edge/router/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [verified provider credential or gameplay ticket, account/character intents, current epoch]
contract_outputs: [canonical account rows, session epoch, attach/detach result, durable-intent router, minimal server main]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement the `password` provider (register/login, Argon2id, no reset) per `../07_security/auth.md` § Password Provider, federated providers, gameplay tickets and refresh rotation.
Implement one-account-one-live-session epochs, attach/detach, `SESSION_REPLACED` and the login queue.
Implement the Edge durable-intent router (`server/internal/edge/router/`): a registry mapping each durable C2S message ID to a handler interface; later tasks register their handler from their own package without editing the router.
Create the minimal session-capable `server/cmd/server` entry point; IMP-069 finalizes composition.

## Acceptance
- login queue at `WORLD_CCU_CAP` per `../07_security/session.md` § Login Queue (FIFO, `SERVER_OVERLOADED.retry_after_ms`, reconnect bypass),
- password register/login follow `auth.md` validation, uniqueness, generic `AUTH_INVALID` and Argon2id rehash rules,
- a second login replaces the old session (`SESSION_REPLACED`); stale epochs cannot act; one account controls at most one live character,
- router rejects unknown or duplicate durable intent IDs.

## Tests
- `server/internal/edge/auth/password_test.go`: TestRegisterValidation, TestUsernameEmailKeyUniqueness, TestLoginGenericFailure, TestArgon2idParamsAndRehash, TestPasswordChangeRevokesOtherSessions.
- `server/internal/edge/auth/auth_test.go`: federated provider verification, gameplay-ticket single use/TTL, refresh rotation/reuse, provider-link uniqueness.
- `server/internal/edge/session/login_queue_test.go`: TestFifoAdmission, TestReconnectBypassesQueue, TestAdmissionWindowExpiry.
- `server/internal/edge/session/session_test.go`: attach/detach, stale epoch rejection, `SESSION_REPLACED`, reconnect authority, one account/one character live.
- `server/internal/edge/router/router_test.go`: TestHandlerRegistryUniqueIds, TestUnknownDurableIntentRejected.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-006/"

## `IMP-100` — Character Lifecycle
id: IMP-100
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/character.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`, `../06_data/text.md`, `../05_network/errors.md`]
adrs: [`0013-canonical-unicode-text-normalization.md`, `0029-character-resource-isolation.md`, `0030-one-account-one-live-session.md`, `0048-character-update-timestamp.md`]
depends_on: [IMP-006]
owned_paths: [`server/internal/durable/character/`, `server/internal/edge/character/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [authenticated account, create/select intents, account status]
contract_outputs: [canonical character rows, character list/select result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement character create/list/select: max three, class permanence, no deletion, normalized display-name uniqueness and `characters.updated_at` (ADR-0048). Creation is rejected while the account is `SUSPENDED_PAYMENT_RECONCILIATION` (`../03_systems/monetization.md`).

## Acceptance
- account cannot exceed three live characters; a 4th create is rejected and no slot product exists,
- characters are permanent; normalized display-name uniqueness never replaces immutable `character_id`,
- `updated_at` changes on every character-row mutation,
- a suspended account cannot create a character; existing characters stay playable.

## Tests
- `server/internal/durable/character/character_test.go`: TestMaxThreeCharactersPerAccount, TestFourthCharacterRejected, TestCharacterPermanenceNoDeletion, TestNormalizedNameKeyUniqueness, TestUpdatedAtMaintained, TestSuspendedAccountCannotCreate.
- `server/internal/edge/character/character_handler_test.go`: TestCreateListSelectFlow, TestSelectForeignCharacterRejected.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-100/"

## `IMP-007` — Currency Primitive
id: IMP-007
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/README.md`, `../03_systems/economy.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0029-character-resource-isolation.md`]
depends_on: [IMP-005, IMP-068, IMP-082, IMP-097]
owned_paths: [`server/internal/durable/currency/`]
forbidden_paths: [`server/internal/sim/`, `client/Assets/Scripts/`]
contract_inputs: [character ID, currency ID, signed delta, operation ID, cap policy]
contract_outputs: [atomic balance mutation or typed rejection, auditable committed result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement scoped balances/caps/atomic debit-credit/audit for common/bound/special.

## Acceptance
- caps, concurrent debit, rollback, retry tests pass.

## Tests
- `server/internal/durable/currency/currency_test.go`: TestAtomicDebitCredit, TestCurrencyCapsEnforcement, TestConcurrentDebitRollback, TestAuditLogBalance.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-007/"

## `IMP-008` — Item Ownership Primitive
id: IMP-008
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/items.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0029-character-resource-isolation.md`, `0043-spirit-beast-instance-identity.md`]
depends_on: [IMP-005, IMP-068, IMP-082, IMP-097]
owned_paths: [`server/internal/durable/items/`]
forbidden_paths: [`server/internal/sim/`, `client/Assets/Scripts/`]
contract_inputs: [item definition, instance/location/binding intent, operation ID]
contract_outputs: [atomic item instance/location ownership result with monotonic binding]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement item definitions/instances/binding/source binding override and exactly-one current `item_locations` ownership context.

## Acceptance
- one item instance has exactly one current ownership/location context even under concurrent transfer,
- inventory/equipped/storage/trade/Auction/claim custody cannot coexist for the same item instance,
- binding never loosens,
- bound-purchase laundering regression passes.

## Tests
- `server/internal/durable/items/items_test.go`: TestItemInstanceCreation, TestSingleItemLocationConstraint, TestCharacterBoundOwnership, TestAccountScopedAccess.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-008/"

## `IMP-009` — Inventory / IAP Entitlement Panel
id: IMP-009
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/inventory.md`, `../03_systems/account_storage.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0029-character-resource-isolation.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-008, IMP-066]
owned_paths: [`server/internal/durable/inventory/`, `client/Assets/Scripts/Systems/Inventory/`, `client/Assets/Scripts/UI/Inventory/`, `client/Assets/Tests/PlayMode/InventoryPanel/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [authoritative item/entitlement snapshots and inventory intents]
contract_outputs: [inventory mutations, IAP panel projection, capacity/rejection UI state]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement inventory capacity/stacking/expansion. `account_storage.md` is the IAP entitlement panel only (ADR-0029): it is not a gameplay item vault and does not introduce an `ACCOUNT_STORAGE` item location.

## Acceptance
- full inventory, split/merge, and entitlement-panel (non-vault) tests pass; no item instance may occupy account storage.

## Tests
- `server/internal/durable/inventory/inventory_test.go`: TestInventoryCapacityStacking, TestInventoryExpansionLimits, TestIAPEntitlementPanelAccess.
- `client/Assets/Tests/PlayMode/InventoryPanel/InventoryPanelTests.cs`: authoritative inventory snapshot/delta, claim overflow, IAP panel separation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-009/"

## `IMP-010` — Reward Claims
id: IMP-010
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/reward_claims.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0012-reward-claim-item-materialization.md`]
depends_on: [IMP-005, IMP-007, IMP-008, IMP-009, IMP-011]
owned_paths: [`server/internal/durable/reward/`, `client/Assets/Scripts/Systems/Rewards/`, `client/Assets/Scripts/UI/Rewards/`, `client/Assets/Tests/PlayMode/RewardClaimUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [committed reward choice, source identity, independent reward slots, operation ID]
contract_outputs: [materialized item/currency delivery or persistent Reward Claim with replayable result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement persistent overflow/recovery claims and compatible currency aggregation.

## Acceptance
- independent reward slots settle without sibling rollback/reroll.

## Tests
- `server/internal/durable/reward/reward_test.go`: TestRewardClaimMaterialization, TestPersistentOverflowClaims, TestCompatibleCurrencyAggregation.
- `client/Assets/Tests/PlayMode/RewardClaimUi/RewardClaimUiTests.cs`: claim list, retry, overflow materialization, and authoritative rejection/result states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-010/"

## `IMP-011` — Character Progression / Stats
id: IMP-011
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/progression.md`, `../01_gameplay/stats.md`]
adrs: [`0031-exp-scale-x100-and-corrected-act-budgets.md`, `0032-seven-channel-exp-source-portfolio.md`, `0033-skill-unlock-schedule-remap.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`]
depends_on: [IMP-007, IMP-066, IMP-100]
owned_paths: [`server/internal/sim/progression/`, `server/internal/durable/progression/`, `client/Assets/Scripts/Systems/Progression/`, `client/Assets/Scripts/UI/Progression/`, `client/Assets/Tests/EditMode/ProgressionPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [authoritative EXP/stat sources, level state, build contributions, content revision]
contract_outputs: [persisted progression, deterministic final stats, client progression projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement level/EXP, skill/potential points, allocation/respec, final stat pipeline.

## Acceptance
- formula vectors and Level-60 stop pass exactly.

## Tests
- `server/internal/sim/progression/progression_test.go`: TestLevelEXPCurve, TestPotentialPointAllocation, TestSkillPointBudget59, TestStatPipelineResolution.
- `client/Assets/Tests/EditMode/ProgressionPresentation/ProgressionPresentationTests.cs`: level/EXP/potential/skill-point authoritative projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-011/"

## `IMP-012` — Equipment / Loadout Core
id: IMP-012
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/equipment.md`, `../07_content/equipment_catalog.md`]
adrs: [`0021-hardcore-enhancement-rate-curve.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`]
depends_on: [IMP-008, IMP-009, IMP-011]
owned_paths: [`server/internal/sim/equipment/`, `server/internal/durable/equipment/`, `client/Assets/Scripts/Systems/Equipment/`, `client/Assets/Scripts/UI/Equipment/`, `client/Assets/Tests/PlayMode/EquipmentUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [owned item instances, equipment definitions, loadout intent, current build lock]
contract_outputs: [atomic equipment/loadout state, derived stat contribution, authoritative UI result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 14 slots, 3 loadouts, ACTIVE/SUPPORT selection, persistent rolls/enhancement state.

## Acceptance
- one-instance-one-slot/loadout contribution tests pass.

## Tests
- `server/internal/sim/equipment/equipment_test.go`: TestFourteenEquipmentSlots, TestThreeLoadoutSwitching, TestEnhancementSuccessCurve, TestLuckyCharmProtection.
- `client/Assets/Tests/PlayMode/EquipmentUi/EquipmentUiTests.cs`: equip/loadout rejection and authoritative stat refresh.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-012/"

## `IMP-065` — Unity Client Bootstrap, Session State & Network Transport
id: IMP-065
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client.md`, `../04_architecture/client_experience_contract.md`, `../05_network/protocol.md`, `../05_network/errors.md`, `../05_network/reconnect.md`, `../05_network/synchronization.md`, `../05_network/versioning.md`, `../07_security/auth.md`, `../07_security/session.md`, `../04_architecture/client_performance.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0030-one-account-one-live-session.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`, `0054-wire-message-completion.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-061, IMP-100]
owned_paths: [`client/Assets/Scripts/Net/`, `client/Assets/Scripts/Core/Session/`, `client/Assets/Scripts/Systems/Character/`, `client/Assets/Scripts/UI/Character/`, `client/Assets/Tests/PlayMode/Harness/`, `client/Assets/Tests/PlayMode/SessionTransport/`, `client/Assets/Tests/PlayMode/CharacterLifecycleClient/`]
forbidden_paths: [`server/internal/`, `server/migrations/`, `server/cmd/compiler/`, `server/cmd/migrate/`, `server/cmd/server/`]
contract_inputs: [HTTPS/WSS endpoints, generated messages, credentials, session/reconnect baseline]
contract_outputs: [Unity transport, session/character-select FSM, baseline/delta/reconnect handling]
consumers_checked: [docs/04_architecture/client_experience_contract.md, docs/04_architecture/physics_geometry_contract.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement Unity network transport layer over WSS using Google.Protobuf generated messages.
- Implement client session state machine: DISCONNECTED, CONNECTING, AUTHENTICATING, IN_WORLD, TRANSFERRING, RECONNECTING.
- Implement heartbeats, sequence validation, transfer freeze/presentation ready handshake, and SESSION_REPLACED handling.
- Provide the shared PlayMode harness (`client/Assets/Tests/PlayMode/Harness/`): fake server and network emulator (latency, jitter, loss) used by later PlayMode tests.

## Acceptance
- Client successfully establishes WSS connection and completes authentication handshake,
- Reconnect and map transfer state transitions match docs/05_network/reconnect.md,
- Session replaced message cleanly shuts down connection without unhandled exceptions.
- bootstrap configures the `1280x720` logical reference surface without treating it as a map size.
- remote interpolation/extrapolation/correction parameters equal `client_performance.md` § Network Smoothness,
- two-phase gate task (`agent_execution_protocol.md` §5a): the implementation PR merges with status `IN_PROGRESS`; a follow-up status PR sets `DONE` with evidence from the post-merge `main` run.

## Tests
- `client/Assets/Tests/PlayMode/SessionTransport/SessionTransportTests.cs`: TestConnectAuthFlow, TestReconnectResume, TestSessionReplacedHandling.
- `client/Assets/Tests/PlayMode/CharacterLifecycleClient/CharacterLifecycleClientTests.cs`: creation/select/attach/detach/session-replaced UI states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-065/"

# Task Group — Realtime Combat / Movement

## `IMP-013` — Movement / Collision / C2S_MOVEMENT_EDGE
id: IMP-013
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/movement.md`, `../04_architecture/realtime_loop.md`, `../04_architecture/physics_geometry_contract.md`, `../05_network/messages.md`, `../05_network/synchronization.md`, `../09_testing/network.md`]
adrs: [`0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-065, IMP-078, IMP-079, IMP-100]
owned_paths: [`server/internal/sim/movement/`, `client/Assets/Scripts/Systems/Movement/`, `client/Assets/Tests/PlayMode/MovementPrediction/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [PRESS/RELEASE/FLIP and jump/drop intents, authoritative geometry, tick state]
contract_outputs: [authoritative transform/velocity, corrections, movement-edge stream, client reconciliation]
consumers_checked: [docs/02_world/maps_zones.md, docs/02_world/monsters.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/monster_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement idle/run/jump/fall/double-jump/one-way drop/knockback on the IMP-078 collision core and the `C2S_MOVEMENT_EDGE` message (message ID 108, `DISCRETE_INTENT` delivery class) per ADR-0038.

**`C2S_MOVEMENT_EDGE` is load-bearing for Just Guard (IMP-014).** ADR-0038 documents the prior defect where the network-layer coalescing rule in `05_network/` made Just Guard permanently non-functional; the coalescing rule must not suppress `C2S_MOVEMENT_EDGE` on the authoritative server path.

Additional scope:
- message ID 108 reserved; must not be reused by any other message type,
- advisory `client_mono_ms` is carried to IMP-014 and never used for authoritative movement timing,
- `STALE_INPUT` anti-cheat rejection emitted and logged for movement-edge messages arriving outside the acceptance window.

## Acceptance
- movement prediction never overrides server position,
- blocked geometry tests pass,
- `CHARACTER` collider is exactly `0.8m x 1.8m` at scale `(1,1,1)` and is independent of the `64x96px` silhouette,
- `C2S_MOVEMENT_EDGE` message ID 108 parses and routes correctly,
- `STALE_INPUT` is emitted and logged for out-of-window movement-edge inputs,
- regression test confirms the network coalescing rule cannot suppress `C2S_MOVEMENT_EDGE` on the authoritative server path (ADR-0038 defect class).

## Tests
- `server/internal/sim/movement/movement_test.go`: TestIdleRunJumpFallTransitions, TestCharacterReferenceCollider, TestDoubleJumpOneWayDrop, TestDiscreteMovementEdgeMessage, TestKnockbackCollision, TestStaleInputRejected, TestCoalescingNeverDropsMovementEdge.
- `client/Assets/Tests/PlayMode/MovementPrediction/MovementPredictionTests.cs`: prediction/reconciliation and PRESS/RELEASE/FLIP dispatch.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-013/"

## `IMP-014` — Combat Action State Machine
id: IMP-014
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/combat.md`, `../01_gameplay/skills.md`, `../04_architecture/realtime_loop.md`, `../09_testing/gameplay.md`, `../05_network/protocol.md`]
adrs: [`0018-combat-target-caps-and-skill-scaling.md`, `0026-just-guard-and-ma-am-status.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0047-skill-reach-budget-and-collider-aware-resolution.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-011, IMP-013, IMP-081]
owned_paths: [`server/internal/sim/combat/`, `client/Assets/Scripts/Systems/Combat/`, `client/Assets/Tests/PlayMode/CombatPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [validated combat intent, actor stats/state, tick time, target candidates]
contract_outputs: [action phase transitions, combat/death events, deterministic rejection]
consumers_checked: [docs/01_gameplay/combat.md, docs/01_gameplay/skills.md, docs/07_content/class_skill_catalog.md, docs/09_testing/gameplay.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement STARTUP->ACTIVE->RECOVERY, action IDs, interruption, `in_combat`, and death-state entry (respawn is IMP-084).
Implement Just Guard (ADR-0034, ADR-0038) and its latency model per `../01_gameplay/combat.md`: edge trigger from `C2S_MOVEMENT_EDGE`, streak, per-session `RTT_estimate` = EWMA(alpha 1/8) of IMP-081 heartbeat RTT samples (200 ms before the first sample), 80 ms compensation clamp.

## Acceptance
- replay/interrupt/combat-lock tests pass,
- target eligibility uses authoritative shape/hurtbox intersection and stable tie-breaking; sprite/pivot/client distance never decides a hit.
- Just Guard fires on the server in a synthetic round trip with simulated latency within the 80 ms clamp and does not fire beyond it,
- Just Guard streak follows ADR-0034; a coalesced or missing movement edge never produces a true positive,
- `RTT_estimate` follows the EWMA rule and 200 ms default; client-declared latency is never trusted.

## Tests
- `server/internal/sim/combat/combat_test.go`: TestStartupActiveRecoveryTiming, TestActionInterruptionRules, TestInCombatStateLifecycle, TestTargetCapsEnforcement, TestClosestHurtboxRangeBoundary, TestSpriteBoundsCannotCreateHit, TestJustGuardWithinClampFires, TestJustGuardBeyondClampRejected, TestJustGuardStreak, TestRttEwmaLatencyModel.
- `client/Assets/Tests/PlayMode/CombatPresentation/CombatPresentationTests.cs`: action start/reject/interrupt/death authoritative presentation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-014/"

## `IMP-015` — Skill Runtime / Geometry
id: IMP-015
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/skills.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/class_skill_catalog.md`]
adrs: [`0005-skill-action-timing-geometry.md`, `0016-twelve-skill-pool-upgradeable-basics.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0047-skill-reach-budget-and-collider-aware-resolution.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-003, IMP-014]
owned_paths: [`server/internal/sim/skills/`, `client/Assets/Scripts/Systems/Skills/`, `client/Assets/Scripts/UI/Skills/`, `client/Assets/Tests/PlayMode/SkillUi/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`]
contract_inputs: [compiled skill definition, action context, typed primary/secondary geometry, authoritative collider profiles, targets]
contract_outputs: [cost/cooldown/projectile/hit/effect requests and client skill presentation]
consumers_checked: [docs/01_gameplay/combat.md, docs/01_gameplay/skills.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/class_skill_catalog.md, docs/07_content/integration_validation.md, docs/07_content/balance_validation.md, docs/07_content/presentation_asset_manifest.md, docs/09_testing/gameplay.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement execution/targeting/tags/timing-speed/typed geometry/cost/cooldown from skill data.

## Acceptance
- all 45 basic/active launch definitions compile,
- ADR-0005 timing/geometry regressions pass,
- hitboxes/projectiles use typed world-space geometry and never sprite silhouette bounds,
- every authoritative shape starts from `caster_anchor_y + 0.9m`, never an animation socket,
- ADR-0047 role/envelope and exact-hurtbox boundary regressions pass for all collider profiles,
- `MOVE_CONTACT_LINE`, `BARRIER_POSITION`, every secondary spatial effect, and displacement tag/effect parity resolve exactly as catalogued,
- Bộc Bộ ember trail remains presentation-only and cannot create a second zone/hit/status result,
- accepted ON_START interruption consumes once,
- combat rolls use Go `math/rand/v2` PCG-64.

## Tests
- `server/internal/sim/skills/skills_test.go`: TestSkillTargetingGeometry, TestSkillOriginY, TestSkillReachRoleBands45, TestColliderBoundaryIntersectionProfiles, TestMoveContactLineCollisionSweep, TestBarrierPositionPlacementAndLifetime, TestSecondarySpatialEffects, TestDisplacementTagEffectParity, TestBocBoTrailPresentationOnly, TestSkillCooldownSpeedScaling, TestSkillResourceDeduction, TestProjectileResolution.
- `client/Assets/Tests/PlayMode/SkillUi/SkillUiTests.cs`: cooldown/cost/targeting presentation from server events; TestSkillTelegraphsUseResolvedGeometry verifies all 45 telegraphs plus Lưu Bộ sweep/Sơn Bích barrier while never changing authoritative reach.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-015/"

## `IMP-016` — Effects / Status / Shield Pipeline
id: IMP-016
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/status_effects.md`, `../01_gameplay/combat.md`]
adrs: [`0002-effect-value-shield-contract.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-014, IMP-015]
owned_paths: [`server/internal/sim/effects/`, `client/Assets/Scripts/Systems/Effects/`, `client/Assets/Tests/PlayMode/EffectPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [typed effect requests, source/target stats, current statuses/shields]
contract_outputs: [ordered damage/heal/status/shield results, recursion-safe combat events]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement damage/heal/status/buff/debuff/shield/displacement/resource effects and recursion safety.

## Acceptance
- `../09_testing/gameplay.md` combat/status tests pass.

## Tests
- `server/internal/sim/effects/effects_test.go`: TestDamageResolutionOrder, TestShieldAbsorptionLifecycle, TestStatusEffectStacking, TestRecursionSafetyCap.
- `client/Assets/Tests/PlayMode/EffectPresentation/EffectPresentationTests.cs`: status/shield/heal/secondary-result rendering and pooling.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-016/"

## `IMP-017` — Class Skills / Skill Levels
id: IMP-017
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/classes.md`, `../07_content/class_skill_catalog.md`]
adrs: [`0016-twelve-skill-pool-upgradeable-basics.md`, `0033-skill-unlock-schedule-remap.md`]
depends_on: [IMP-015, IMP-016]
owned_paths: [`server/internal/sim/classes/`, `client/Assets/Scripts/Systems/Classes/`, `client/Assets/Tests/EditMode/ClassPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [compiled five-class skill kits, unlock schedule, allocated skill points]
contract_outputs: [validated learned skill levels, numeric outcomes, class presentation data]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement all five class kits and Level1..12 / Level1..6 scaling under ADR-0016.

## Acceptance
- 60 upgradeable skills (4 basic + 5 active + 3 passive per class),
- basic and active skills scale 1..12; passive skills scale 1..6,
- basic attack cooldown scales to class speed caps (0.2s..0.5s) with status proc scaling,
- only `skill.kim.basic.vo_song_kiem` carries PENETRATE at ratio 0.15; other basic_4 skills have neither the tag nor a ratio,
- every upgrade changes a numeric outcome,
- synthetic class combat fixtures pass.

## Tests
- `server/internal/sim/classes/classes_test.go`: TestFiveClassKitsResolution, TestBasicAttackScalingLevel12, TestActiveSkillScalingLevel12, TestPassiveScalingLevel6.
- `client/Assets/Tests/EditMode/ClassPresentation/ClassPresentationTests.cs`: five class kits and skill-level data projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-017/"

## `IMP-084` — Death / Respawn
id: IMP-084
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/death_respawn.md`, `../01_gameplay/combat.md`, `../01_gameplay/status_effects.md`, `../02_world/maps_zones.md`]
adrs: [`0007-single-owner-fixed-step-simulation.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-014, IMP-016, IMP-018]
owned_paths: [`server/internal/sim/death/`, `client/Assets/Scripts/Systems/Death/`, `client/Assets/Tests/PlayMode/DeathPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`]
contract_inputs: [HP <= 0 combat result, checkpoint, status state]
contract_outputs: [DEAD/RESPAWNING/ACTIVE transitions, respawn placement, invulnerability window]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `death_respawn.md`: atomic death steps, dead-state restrictions, status/cooldown/summon cleanup, `RESPAWN_DELAY` 3 s, checkpoint respawn at 40% HP/MP, checkpoint fallback and 3 s respawn invulnerability.

## Acceptance
- death processing is atomic and idempotent,
- dead characters cannot move, attack, interact or be healed; UI/chat and respawn request stay available,
- respawn after 3 s at the marked checkpoint with `floor(MAX * 0.40)` HP/MP; an invalid checkpoint falls back to `checkpoint.lang_da.dinh_lang`; client coordinates are never accepted,
- during the 3 s invulnerability incoming hostile damage and harmful statuses are ignored and outgoing damage is 0.

## Tests
- `server/internal/sim/death/death_test.go`: TestDeathAtomicSteps, TestDeathIdempotent, TestDeadStateRestrictions, TestRespawnDelay3s, TestRespawnHpMp40Percent, TestInvalidCheckpointFallback, TestRespawnInvulnerability3s, TestOutgoingDamageZeroWhileInvulnerable.
- `client/Assets/Tests/PlayMode/DeathPresentation/DeathPresentationTests.cs`: dead/respawning/active presentation from server events.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-084/"

## `IMP-066` — Unity Input Action Mapping & Core UI/HUD State Machine
id: IMP-066
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client.md`, `../04_architecture/client_experience_contract.md`, `../01_gameplay/movement.md`, `../05_network/messages.md`, `../05_network/synchronization.md`, `../04_architecture/client_performance.md`]
adrs: [`0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-013, IMP-065]
owned_paths: [`client/Assets/Scripts/UI/CoreHud/`, `client/Assets/Scripts/UI/StateMachine/`, `client/Assets/Scripts/Core/Input/`, `client/Assets/Tests/PlayMode/InputHudStateMachine/`]
forbidden_paths: [`server/`]
contract_inputs: [keyboard/gamepad/touch actions and authoritative replication/combat events]
contract_outputs: [wire intents, core UI FSM, HUD projection, accessibility-safe controls]
consumers_checked: [docs/04_architecture/client_experience_contract.md, docs/04_architecture/physics_geometry_contract.md, client/ProjectSettings/ProjectSettings.asset, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement Unity Input System action maps for keyboard, mouse, and touch overlays.
- Wire C2S_MOVEMENT_EDGE (108) discrete intent message dispatch.
- Implement core HUD state machine displaying HP, MP, action skill cooldowns, hotbar, and active buffs.
- Map the stop action to wire `RELEASE` per `../04_architecture/client_experience_contract.md`; do not add a `STOP` enum.

## Acceptance
- Input actions dispatch discrete edge messages matching server timing guardrails,
- HUD reflects authoritative server combat events and status updates without desync,
- HUD/camera layout passes `1280x720`, 16:9, 21:9 and safe-area fixtures without changing world-space scale,
- Action state prevents duplicate trigger events during startup/recovery windows.

## Tests
- `client/Assets/Tests/PlayMode/InputHudStateMachine/InputHudStateMachineTests.cs`: TestInputEdgeDispatch, TestCooldownDisplaySync, TestBuffDisplayUpdate.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-066/"

# Task Group — Client Performance / Screens

## `IMP-095` — Client Performance Budgets & Quality Presets (every PR)
id: IMP-095
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client_performance.md`, `../04_architecture/client.md`, `../07_content/presentation_asset_manifest.md`, `../09_testing/load.md`, `engineering_conventions.md`, `audit_gates.md`]
adrs: [`0050-windows-only-ci-and-auto-merge.md`, `0052-single-launch-world.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-063, IMP-065, IMP-066, IMP-101]
owned_paths: [`client/Assets/Scripts/Core/Performance/`, `client/ProjectSettings/QualitySettings.asset`, `client/Assets/Scenes/Perf/`, `client/Assets/Tests/PlayMode/Performance/`, `client/Assets/Tests/EditMode/PerformanceBudgets/`]
forbidden_paths: [`server/`]
contract_inputs: [client_performance.md targets, Addressables catalog, IMP-065 network emulator]
contract_outputs: [quality presets, battery saver, hotspot scene, every-PR client-performance gate]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
- Implement the 5-second first-launch benchmark that selects `LOW`/`MEDIUM`/`HIGH`; presets change only render scale, point Light2D budget, particle budget, parallax `L3` and bloom. Implement the battery-saver 30 FPS cap.
- Build the hotspot scene `client/Assets/Scenes/Perf/` (18 players + 42 `AI_CLASS_NAMED_MECHANIC` monsters + skill VFX, `../09_testing/load.md` scenario 11) from Addressables keys, so production art is measured as it lands.
- Run PlayMode performance tests (category `Performance`) with `FrameTimingManager`/`ProfilerRecorder` on the GitHub-hosted Linux CI job (no GPU, ADR-0058): CPU timing with `-batchmode -nographics`, draw/memory measurements under xvfb + Mesa llvmpipe; network-smoothness tests use the IMP-065 emulator. These form the every-PR client-performance gate once this task is `DONE`; GPU frame time is never measured in CI.

## Acceptance
- PERF-001: the benchmark selects a preset; switching presets changes presentation only (colliders, hitboxes and telegraphs identical),
- PERF-002: desktop CPU budget in the hotspot scene for 5 min on the Linux job (`-nographics`): main-thread CPU time excluding GPU/present waits p95 <= 8 ms, p99 <= 12 ms, no frame > 33 ms,
- PERF-004: 0 bytes managed GC allocation per frame in steady gameplay in the hotspot scene,
- PERF-005 (desktop): resident memory <= 2.5 GB,
- PERF-006: batches <= 150 and SetPass calls <= 60 on `LOW`; texture memory within `presentation_asset_manifest.md` §1; active point Light2D and particle counts <= preset budget,
- PERF-009: local input -> first visual response <= 1 rendered frame; server-confirmed result shown <= RTT + 50 ms,
- PERF-010: interpolation delay 2 snapshot intervals adaptive 150..300 ms, extrapolation <= 250 ms, correction smoothed over 100 ms when <= 0.5 m else snapped,
- PERF-011: full-quality conditions (RTT <= 150 ms, jitter <= 30 ms, loss <= 2%) give <= 1 correction > 0.5 m per minute; degraded conditions (<= 300 ms, <= 60 ms, <= 5%) show the network indicator with no desync,
- PERF-013: battery saver caps FPS at 30 on every tier,
- performance tests run only in the Linux job and never read GPU frame time; the gate never reports a skipped pass once this task is `DONE`.

## Tests
- `client/Assets/Tests/PlayMode/Performance/QualityPresetTests.cs`: TestBenchmarkSelectsPreset (PERF-001), TestPresetsPresentationOnly (PERF-001), TestBatterySaverCaps30 (PERF-013).
- `client/Assets/Tests/PlayMode/Performance/HotspotFrameTests.cs`: TestDesktopCpuBudget (PERF-002), TestZeroGcPerFrame (PERF-004), TestDesktopResidentMemory (PERF-005), TestBatchesSetPassLightsParticles (PERF-006), TestPerformanceCategoryLinuxOnlyNoGpuTiming.
- `client/Assets/Tests/PlayMode/Performance/InputLatencyTests.cs`: TestFirstVisualResponseOneFrame (PERF-009), TestConfirmedResultWithinRttPlus50 (PERF-009).
- `client/Assets/Tests/PlayMode/Performance/NetworkSmoothnessTests.cs`: TestInterpolationExtrapolationCorrection (PERF-010), TestFullQualityAndDegradedConditions (PERF-011).
- `client/Assets/Tests/EditMode/PerformanceBudgets/TextureBudgetTests.cs`: TestTextureMemoryBudgets (PERF-006).

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-095/"

## `IMP-099` — Client Screens: Login, Queue, Loading, Settings, Credits
id: IMP-099
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client_experience_contract.md`, `../04_architecture/client.md`, `../04_architecture/client_performance.md`, `../04_architecture/client_localization.md`, `../07_security/auth.md`, `../07_security/session.md`]
adrs: [`0015-unity-localization.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-064, IMP-065, IMP-066, IMP-095]
owned_paths: [`client/Assets/Scripts/UI/Screens/`, `client/Assets/Tests/PlayMode/Screens/`]
forbidden_paths: [`server/`]
contract_inputs: [auth/session results, quality preset API, localization tables]
contract_outputs: [login/register, login-queue, loading/transfer progress, settings and credits screens]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement login/register (password and federated), login-queue position with `retry_after_ms`, loading/transfer progress, settings (quality preset, battery saver, locale) and the credits screen resolving `asset.ui.credits.third_party_assets`.

## Acceptance
- PERF-008: every wait > 0.5 s shows progress; no frozen frame > 100 ms while loading (async Addressables, incremental instantiation),
- `AUTH_INVALID` is shown generically; `SERVER_OVERLOADED` shows the queue and retries after `retry_after_ms`,
- settings persist preset and battery saver through IMP-095; every string is a localization key in vi-VN and en-US.

## Tests
- `client/Assets/Tests/PlayMode/Screens/ScreensTests.cs`: TestLoginRegisterFlow, TestLoginQueueDisplay, TestSettingsPresetAndBatterySaver, TestCreditsKeyResolves.
- `client/Assets/Tests/PlayMode/Screens/LoadingProgressTests.cs`: TestProgressShownAfterHalfSecond (PERF-008), TestNoFrozenFrameOver100ms (PERF-008).

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-099/"

## `IMP-096` — Android Device Performance on Firebase Test Lab
id: IMP-096
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client_performance.md`, `audit_gates.md`, `../00_context/technology_versions.md`, `../09_testing/test_and_release_evidence.md`]
adrs: [`0050-windows-only-ci-and-auto-merge.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-067, IMP-095]
owned_paths: [`.github/workflows/device_perf.yml`, `scripts/device_perf.ps1`, `client/Assets/Scripts/Core/PerformanceDevice/`, `server/internal/conformance/deviceperf/`]
forbidden_paths: [`server/internal/sim/`, `server/internal/durable/`, `proto/`]
contract_inputs: [IL2CPP Android hotspot build, Owner Setup device models, Test Lab quota]
contract_outputs: [device-perf results manifest, PASSED/FAILED/DEFERRED(quota) verdict]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
- Add the Unity game-loop entry (`client/Assets/Scripts/Core/PerformanceDevice/`) that runs the hotspot scene in an IL2CPP Android build and writes frame timings, memory and CPU samples to the game-loop results file.
- Add the scheduled `device-perf` workflow on `main`: at most one run per UTC day, only when client code/assets changed since the last device run, plus the launch-candidate run; it calls `gcloud firebase test android run --type game-loop` (gcloud pinned in `technology_versions.md`) on the ANDROID_MIN and ANDROID_REC models from Owner Setup, downloads results and gates them. The workflow runs on `ubuntu-24.04` (ADR-0058); no device is attached to the runner.
- Exhausted Test Lab quota reports `DEFERRED(quota)` and retries the next day; the workflow is never a PR check and never opens `OPS-xxx` for quota.

## Acceptance
- PERF-003: ANDROID_MIN p95 <= 33.3 ms, p99 <= 45 ms, <= 1 hitch per 5 min; ANDROID_REC p95 <= 16.7 ms, p99 <= 25 ms, <= 1 hitch per 5 min,
- PERF-005 (device): ANDROID_MIN resident memory <= 1.3 GB,
- PERF-007 (device): ANDROID_MIN cold start <= 10 s, login to in-world <= 8 s, transfers <= 3 s / 6 s, reconnect resume <= 5 s,
- PERF-012: 30-minute launch-candidate runs: ANDROID_REC p50 >= 45 FPS for the whole run; ANDROID_MIN average CPU <= 50% and no p95 regression > 10% between minute 1 and minute 30,
- cadence and quota: <= 1 scheduled run per day, skipped when no client path changed; `DEFERRED(quota)` never fails a PR; the launch-candidate run waits until quota allows and never skips.

## Tests
- `server/internal/conformance/deviceperf/deviceperf_test.go`: TestAndroidFramePacingGate (PERF-003), TestAndroidMemoryGate (PERF-005), TestAndroidLoadTimeGate (PERF-007), TestSustainedProxyGate (PERF-012), TestDailyCadenceOnlyOnClientChange, TestQuotaExhaustedIsDeferred, TestLaunchCandidateWaitsForPass.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-096/"

# Task Group — World / Spawns / Quests

## `IMP-018` — Map / Transfer / Checkpoint Runtime
id: IMP-018
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/README.md`, `../02_world/maps_zones.md`, `../02_world/world_rules.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/world_route_catalog.md`]
adrs: [`0020-map-channel-capacity-contract.md`, `0035-spawn-density-increase.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-013, IMP-066, IMP-100]
owned_paths: [`server/internal/sim/world/`, `server/internal/durable/world/`, `client/Assets/Scripts/Systems/World/`, `client/Assets/Tests/PlayMode/WorldTransferPresentation/`]
forbidden_paths: [`server/migrations/`, `client/Assets/Scripts/UI/`]
contract_inputs: [map/portal graph, geometry, channel state, transfer/checkpoint intent]
contract_outputs: [authoritative map/channel placement, checkpoint persistence, transfer/recovery result]
consumers_checked: [docs/02_world/maps_zones.md, docs/04_architecture/client_experience_contract.md, docs/04_architecture/physics_geometry_contract.md, docs/06_data/config.md, docs/07_content/world_route_catalog.md, docs/07_content/map_spawn_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement normal-world map instances, entry spawns, portals, checkpoints, transfer failure/reconnect fallback.

## Acceptance
- 24-map/52-portal route compile and anti-softlock tests pass,
- all 24 maps load exact, mutually distinct width-height spans and distinct topology profiles; camera scrolls inside map bounds instead of treating `1280x720` as map size,
- every release-scope destination resolves a compatible Addressables dependency set,
- missing/download-failed presentation assets cannot cause client-selected transfer fallback or authoritative state mutation.

## Tests
- `server/internal/sim/world/world_test.go`: TestMapInstanceLifecycle, TestTwentyFourMapBoundsAndDistinctTopologies, TestCheckpointTransferHandshake, TestTransferTimeoutFallback, TestPortalTransition.
- `client/Assets/Tests/PlayMode/WorldTransferPresentation/WorldTransferPresentationTests.cs`: preload/ready/failure/recovery and channel-switch UI.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-018/"

## `IMP-019` — Spawn Runtime / Monster AI
id: IMP-019
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/spawning.md`, `../02_world/monsters.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/monster_catalog.md`, `../07_content/map_spawn_catalog.md`]
adrs: [`0003-spawn-selector-anchor-contract.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`, `0035-spawn-density-increase.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-003, IMP-016, IMP-018]
owned_paths: [`server/internal/sim/spawning/`, `client/Assets/Scripts/Systems/Monsters/`, `client/Assets/Tests/PlayMode/MonsterPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [spawn groups, monster definitions, channel state, deterministic RNG]
contract_outputs: [bounded monster population, AI state/events, spawn/despawn presentation]
consumers_checked: [docs/02_world/monsters.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/monster_catalog.md, docs/07_content/map_spawn_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 54 persistent groups, pool selectors, respawn/leash/basic AI.

## Acceptance
- population/restart/reward-once tests pass; spawn-pool rolls use Go `math/rand/v2` PCG-64,
- every spawned monster uses its compiled canonical size profile; sprite/Transform scale cannot change authoritative collision.

## Tests
- `server/internal/sim/spawning/spawning_test.go`: TestFiftyFourSpawnGroups, TestMonsterSizeProfileResolution, TestPoolSelectorWeights, TestMonsterLeashRespawnLifecycle, TestSimpleAIBehavior.
- `client/Assets/Tests/PlayMode/MonsterPresentation/MonsterPresentationTests.cs`: spawn/despawn/AI-state interpolation without client authority.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-019/"

## `IMP-020` — Discovery / Progression Source Events
id: IMP-020
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/README.md`, `../01_gameplay/core_loop.md`, `../07_content/world_route_catalog.md`]
adrs: [`0025-peak-moments-and-progression-books.md`]
depends_on: [IMP-005, IMP-011, IMP-018]
owned_paths: [`server/internal/sim/discovery/`, `server/internal/durable/discovery/`, `client/Assets/Scripts/UI/Discovery/`, `client/Assets/Tests/PlayMode/DiscoveryPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [first-discovery/progression source event, character state, operation ID]
contract_outputs: [once-only durable progress/reward result and client discovery event]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 24 character first-discovery rewards and progression-source operation identities.

## Acceptance
- 0.8% discovery budget/act (safe-anchor 0.2% + three FIELD 0.2% each) and retry tests pass.

## Tests
- `server/internal/sim/discovery/discovery_test.go`: TestTwentyFourFirstDiscoveryEvents, TestProgressionSourceOperationIDs, TestIdempotentDiscoveryGrants.
- `client/Assets/Tests/PlayMode/DiscoveryPresentation/DiscoveryPresentationTests.cs`: first-discovery result and duplicate suppression.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-020/"

## `IMP-021` — Quest Runtime
id: IMP-021
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/quests.md`, `../07_content/quest_catalog.md`]
adrs: [`0025-peak-moments-and-progression-books.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`]
depends_on: [IMP-010, IMP-011, IMP-018, IMP-019]
owned_paths: [`server/internal/sim/quests/`, `server/internal/durable/quests/`, `client/Assets/Scripts/Systems/Quests/`, `client/Assets/Scripts/UI/Quests/`, `client/Assets/Tests/PlayMode/QuestUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [typed world/combat/interaction events, quest definitions, character quest state]
contract_outputs: [durable objective transitions, completion/reward result, quest UI projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement MAIN/SIDE/Daily/Event objective state/prerequisites/rewards.

## Acceptance
- 24 MAIN chain,12 SIDE, Daily board and no-public-boss-gate tests pass.

## Tests
- `server/internal/sim/quests/quests_test.go`: TestMainSideDailyQuestStates, TestPrerequisiteValidation, TestQuestObjectiveProgress, TestQuestRewardSettlement.
- `client/Assets/Tests/PlayMode/QuestUi/QuestUiTests.cs`: objective delta, completion, rejection, and tracker limits.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-021/"

## `IMP-089` — Mystery Bounty
id: IMP-089
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/quests.md`, `../07_content/quest_catalog.md`]
adrs: [`0027-world-liveliness-mystery-bounty-capacity.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`]
depends_on: [IMP-021]
owned_paths: [`server/internal/sim/quests/bounty/`, `client/Assets/Scripts/UI/Quests/Bounty/`, `client/Assets/Tests/PlayMode/BountyUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [DAILY bounty set, character position/NPC interaction]
contract_outputs: [mystery reveal, settlement with EXP bonus and bound currency]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `quests.md` § Mystery Bounty: 1 of the 6 DAILY choices hidden until area/NPC reveal, variable-ratio selection, +15% EXP and the MYSTERY `currency.bound` grant.

## Acceptance
- exactly one of six DAILY choices is MYSTERY and stays `???` until the character reaches its area or starter NPC,
- completion grants `floor(floor(bounty_set_exp / 3) * 1.15)` EXP and the `quest_catalog.md` MYSTERY bound amount,
- settlement is per bounty and order-independent; retries settle once.

## Tests
- `server/internal/sim/quests/bounty/bounty_test.go`: TestSixDailyChoicesOneMystery, TestHiddenUntilAreaOrNpc, TestMysteryExpBonus115, TestMysteryBoundCurrencyGrant, TestOrderIndependentSettlement.
- `client/Assets/Tests/PlayMode/BountyUi/BountyUiTests.cs`: hidden/revealed/completed states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-089/"

## `IMP-090` — Progression Books
id: IMP-090
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/progression.md`, `../03_systems/items.md`, `../07_content/item_catalog.md`, `../02_world/quests.md`]
adrs: [`0025-peak-moments-and-progression-books.md`, `0033-skill-unlock-schedule-remap.md`, `0043-spirit-beast-instance-identity.md`]
depends_on: [IMP-009, IMP-011, IMP-021, IMP-023]
owned_paths: [`server/internal/sim/books/`, `client/Assets/Scripts/UI/Books/`, `client/Assets/Tests/PlayMode/BooksUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [eligible MAIN/FIRST_CLEAR/level-milestone sources, character level]
contract_outputs: [book grants, atomic consumption to potential/skill points]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `progression.md` § Bonus Books: `item.book.potential`/`item.book.skill` grants from eligible sources from Level 25 and atomic consumption into allocation points.

## Acceptance
- books come only from eligible MAIN quests, dungeon FIRST_CLEAR or level-milestone rewards per the schedule table,
- consumption is atomic and grants exactly the listed points once,
- books are character-bound and cannot be traded or auctioned.

## Tests
- `server/internal/sim/books/books_test.go`: TestBookScheduleFromLevel25, TestConsumeAtomicallyGrantsPoints, TestBooksNotTradable, TestOnlyEligibleSourcesGrant.
- `client/Assets/Tests/PlayMode/BooksUi/BooksUiTests.cs`: grant/consume/rejection states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-090/"

## `IMP-025` — Spirit Surge
id: IMP-025
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/world_rules.md`, `../07_content/world_event_catalog.md`]
adrs: [`0027-world-liveliness-mystery-bounty-capacity.md`, `0035-spawn-density-increase.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-010, IMP-019, IMP-021, IMP-080]
owned_paths: [`server/internal/global/spirit_surge/`, `server/internal/sim/world/surge/`, `client/Assets/Scripts/Systems/WorldEvents/`, `client/Assets/Scripts/UI/WorldEvents/`, `client/Assets/Tests/PlayMode/SpiritSurgePresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`]
contract_inputs: [UTC hour, world-event catalog, content revision, live partition list]
contract_outputs: [single-writer Spirit Surge selection and typed activate/deactivate commands]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement deterministic hourly event selection/waves/contribution/rewards.

## Acceptance
- world-event regression suite passes.

## Tests
- `server/internal/global/spirit_surge/spirit_surge_test.go`: TestHourlySurgeEventSelection, TestWaveContributionTracking, TestSurgeRewardDistribution.
- `client/Assets/Tests/PlayMode/SpiritSurgePresentation/SpiritSurgePresentationTests.cs`: global activation/deactivation and region UI.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-025/"

## `IMP-055` — Entity Capacity Enforcement
id: IMP-055
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/maps_zones.md`, `../02_world/world_rules.md`, `../04_architecture/realtime_loop.md`, `../05_network/synchronization.md`, `../08_scale_ops/capacity.md`]
adrs: [`0020-map-channel-capacity-contract.md`, `0035-spawn-density-increase.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0052-single-launch-world.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-018, IMP-019, IMP-035]
owned_paths: [`server/internal/sim/spatial/capacity/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [channel entity set, spawn request, per-client AOI candidates and priority]
contract_outputs: [80-entity admission decision, 40-entity AOI projection, hotspot metrics]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement `MAX_ENTITIES_PER_CHANNEL = 80` spawn rejection on the hot path: exactly 80 entities are permitted; a spawn that would become the 81st is rejected with a logged reason; no entity is silently over-allocated or queued indefinitely.
- Implement `MAX_ENTITIES_IN_AOI_PER_CLIENT = 40` AOI shedding logic on the hot path: when the server-side AOI set for a client exceeds 40 entities, the server sheds the lowest-priority entities from that client's update stream until the count is within threshold.
- Implement the named hotspot benchmark: 42 `AI_CLASS_NAMED_MECHANIC` monsters plus 18 players in sustained combat in one channel, measuring per-tick server cost against the performance budget declared in `../08_scale_ops/`. This benchmark is a **release gate for M10** (IMP-046 and IMP-048 depend on it passing).

## Acceptance
- Channel spawn request that would exceed `MAX_ENTITIES_PER_CHANNEL = 80` is rejected server-side with a structured log entry; no entity is silently over-committed.
- Spawn at exactly 80 entities is permitted; the 81st entity is rejected.
- AOI shedding activates when entity count in a client's AOI exceeds 40; shed entities receive no state updates on that client until the count drops back within threshold.
- AOI shedding is server-side; no client-provided entity count or priority hint is trusted.
- Named hotspot benchmark (42 NAMED_MECHANIC + 18 players, plus projectiles/transients up to the 80-entity cap) passes p95 tick < 35 ms in a single-channel harness owned by this task; the 10k CCU run of the same scenario belongs to IMP-046.

## Tests
- `server/internal/sim/spatial/capacity/capacity_test.go`: entity cap, AOI shedding, hotspot fixture, and forged-priority rejection.
- `server/internal/sim/spatial/capacity/capacity_test.go`: Unit: spawn rejection at boundary values (80 → permit, 81 → reject).
- `server/internal/sim/spatial/capacity/capacity_test.go`: Unit: AOI shedding at boundary values (40 → full updates, 41 → shed lowest-priority entity).
- `server/internal/sim/spatial/capacity/capacity_test.go`: Integration: hotspot benchmark — 42 NAMED_MECHANIC + 18 players at max density, per-tick cost measured and compared to budget.
- `server/internal/sim/spatial/capacity/hotspot_bench_test.go`: Benchmark — single-channel hotspot at 80 entities; result attached to IMP-055 evidence.
- `server/internal/sim/spatial/capacity/capacity_test.go`: Regression: spawn rejection cannot be bypassed by a client-sent entity count or priority override.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-055/"

# Task Group — Economy / Equipment Progression

## `IMP-058` — Folk Fishing Runtime
id: IMP-058
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/world_rules.md`, `../07_content/economy_catalog.md`]
adrs: [`0024-fishing-cooking-feats-titles-boss-chest-ceremony.md`, `0035-spawn-density-increase.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-002, IMP-008, IMP-010, IMP-018]
owned_paths: [`server/internal/sim/fishing/`, `server/internal/durable/fishing/`, `client/Assets/Scripts/Systems/LifeSkills/Fishing/`, `client/Assets/Scripts/UI/LifeSkills/Fishing/`, `client/Assets/Tests/PlayMode/FishingPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [CAST/HOOK intent, fishing table, character/day/cast identity, RNG]
contract_outputs: [authoritative fishing state/result, durable counters/reward, client presentation]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement folk fishing state machine `IDLE -> CASTING -> HOOK_WINDOW -> RESOLVING -> IDLE`.
- `C2S_INTERACT` (103) `interact_kind=CAST` starts CASTING; `HOOK_WINDOW` accepts only `interact_kind=HOOK`.
- Catch rolls use Go `math/rand/v2` PCG-64 with key `fishing.<character_id>.<utc_date>.<cast_sequence>`.

## Acceptance
- CAST/HOOK only via `C2S_INTERACT` `interact_kind`; HOOK outside `HOOK_WINDOW` is rejected.
- Catch IDs = `COMMON_CATCH ∪ RARE_CATCH ∪` IDs of the active seasonal table (at most one extra `SEASONAL_CATCH`); default table remains closed to `COMMON_CATCH ∪ RARE_CATCH`.
- RNG is PCG-64; client cannot supply the roll.
- 51st successful catch in a UTC day is rejected.

## Tests
- `server/internal/sim/fishing/fishing_test.go`: Unit: CAST then HOOK in window succeeds; HOOK without CAST fails; CAST during `HOOK_WINDOW` fails.
- `server/internal/sim/fishing/fishing_test.go`: Integration: seeded PCG-64 catch table; bait consumed once; retry cannot reroll.
- `server/internal/sim/fishing/fishing_test.go`: Regression: `interact_kind` other than HOOK is rejected in `HOOK_WINDOW`.
- `client/Assets/Tests/PlayMode/FishingPresentation/FishingPresentationTests.cs`: CAST/HOOK windows, result, daily counter projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-058/"

## `IMP-059` — Hearth / Cooking / Bonfire Runtime
id: IMP-059
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/world_rules.md`, `../07_content/crafting_catalog.md`]
adrs: [`0023-engagement-loops-weapon-glow-bonfire-chivalry-chests-sparring.md`, `0024-fishing-cooking-feats-titles-boss-chest-ceremony.md`, `0035-spawn-density-increase.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-003, IMP-007, IMP-008, IMP-018]
owned_paths: [`server/internal/sim/cooking/`, `server/internal/durable/cooking/`, `client/Assets/Scripts/Systems/LifeSkills/Cooking/`, `client/Assets/Scripts/UI/LifeSkills/Cooking/`, `client/Assets/Tests/PlayMode/CookingBonfirePresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [KINDLE/COOK/BONFIRE_REST intent, hearth recipes, inventory/state]
contract_outputs: [atomic cooking/rest result, durable counters/items, timed buff presentation]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement `KINDLE`, `COOK`, and `BONFIRE_REST` via `C2S_INTERACT` `interact_kind`.
- Each `recipe.food.*` hearth recipe (all 5 rows) grants `extra_output = 1 item.material.cui_lua_trai`. Cooking is the kindling faucet; do not add shop rows or prices.

## Acceptance
- `KINDLE` consumes one `item.material.cui_lua_trai` at an inactive bonfire per `world_rules.md`.
- `COOK` of any `recipe.food.*` yields the dish plus 1 `item.material.cui_lua_trai`.
- `BONFIRE_REST` ticks rest EXP and Linh Thú bond per `world_rules.md`.
- No shop faucet for kindling.

## Tests
- `server/internal/sim/cooking/cooking_test.go`: Unit: all 5 food recipes emit `extra_output` `cui_lua_trai`.
- `server/internal/sim/cooking/cooking_test.go`: Integration: `KINDLE`/`COOK`/`BONFIRE_REST` `interact_kind`; full inventory routes extra_output to Reward Claims.
- `server/internal/sim/cooking/cooking_test.go`: Regression: kindling cannot be purchased from an NPC shop.
- `client/Assets/Tests/PlayMode/CookingBonfirePresentation/CookingBonfirePresentationTests.cs`: kindle/cook/rest/buff states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-059/"

# Task Group — Dungeons / Bosses

## `IMP-022` — Boss Runtime / WorldConsequence
id: IMP-022
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/bosses.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/boss_catalog.md`, `../07_content/world_route_catalog.md`, `../07_content/dungeon_catalog.md`]
adrs: [`0031-exp-scale-x100-and-corrected-act-budgets.md`, `0040-world-consequence-durable-aggregate.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0053-durable-contract-reconciliation.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-005, IMP-010, IMP-016, IMP-019]
owned_paths: [`server/internal/sim/bosses/`, `server/internal/durable/worldconsequence/`, `client/Assets/Scripts/Systems/Bosses/`, `client/Assets/Scripts/UI/Bosses/`, `client/Assets/Tests/PlayMode/BossPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [boss generation/combat events, eligibility, world key, operation ID]
contract_outputs: [boss lifecycle, WorldConsequence mutation, chest/reward result, client events]
consumers_checked: [docs/02_world/bosses.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/boss_catalog.md, docs/07_content/world_route_catalog.md, docs/07_content/dungeon_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement INSTANCED/PUBLIC lifecycle, PARTY/PUBLIC scaling, participation, boss mechanics/add ownership, and the `WriteWorldConsequence` durable command (ADR-0040).

**`WorldConsequence` is required by ADR-0040.** The prior defect: the original acceptance criteria (eight bosses compile, HP formula, anti-duplication) could all pass while the durable-write path was absent, leaving boss relics and world-state records vulnerable to loss on partition restart. `WriteWorldConsequence` must be part of this task.

Additional scope:
- `WriteWorldConsequence` durable command backed by two persistent tables as defined in ADR-0040,
- durable command uses the operation idempotency primitive from IMP-005,
- partition-start recovery path reads unacknowledged `WorldConsequence` rows and blocks new player acceptance until recovery completes,
- no boss relic or world-state row may use `map_instance_id` as a durable foreign key (`map_instance_id` is a runtime identity per `02_world/world_rules.md` and is invalid after partition restart).

## Acceptance
- eight bosses compile,
- every boss resolves the exact declared space and `BOSS_LARGE|WORLD_BOSS` profile,
- HP formula/scaling tests pass,
- public generation anti-duplication passes,
- `WriteWorldConsequence` commits before boss outcome is reported to clients,
- simulated server restart after boss kill but before client acknowledgement recovers the consequence row and delivers the correct result without duplication,
- partition-start recovery blocks new player acceptance until all pending `WorldConsequence` rows are resolved,
- no boss relic or world-state row keys on `map_instance_id`; a migration test verifies any pre-existing such row is rejected or migrated to a stable key.

## Tests
- `server/internal/sim/bosses/bosses_test.go`: TestInstancedPublicBossLifecycle, TestBossSpaceAndSizeProfiles, TestPartyPublicScalingResolution, TestWriteWorldConsequenceDurableCommand.
- `client/Assets/Tests/PlayMode/BossPresentation/BossPresentationTests.cs`: telegraph, generation ID, chest, and world-consequence rendering.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-022/"

## `IMP-091` — Di Tích Relics & Boss Chest Ceremony
id: IMP-091
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/bosses.md`, `../06_data/data_model.md`, `../06_data/save_rules.md`, `../07_security/validation.md`]
adrs: [`0024-fishing-cooking-feats-titles-boss-chest-ceremony.md`, `0040-world-consequence-durable-aggregate.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-010, IMP-022, IMP-023]
owned_paths: [`server/internal/sim/bosses/relics/`, `server/internal/sim/bosses/chest/`, `client/Assets/Scripts/Systems/Bosses/Relics/`, `client/Assets/Tests/PlayMode/RelicChestPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [DEFEATED boss transitions, generation-scoped contribution records]
contract_outputs: [relic/marker state, buff, personal chest settlements]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement launch Di Tích relics and markers and the Gilded Chest ceremony of `bosses.md`: 60-minute channel relic with `buff.di_tich.<boss_id>`, restart restore, region marker, 3-minute chest with generation-scoped eligibility and personal loot, unclaimed rewards to Reward Claims.

## Acceptance
- `DEFEATED -> COOLDOWN` spawns `relic.boss.<boss_id>` for 60 min in that map instance; the channel buff is non-stacking and ends on despawn or map exit,
- an active relic is restored after restart with remaining duration (>= 1 s); the marker records last defeat time, participants and active state,
- the chest stays interactable 3 min; eligibility requires a non-expired contribution with the chest generation ID; one claim never depletes another; unclaimed rewards go to Reward Claims.

## Tests
- `server/internal/sim/bosses/relics/relics_test.go`: TestRelicSpawnOnDefeated, TestChannelBuffNonStacking, TestRelicRestoreAfterRestart, TestDiTichMarkerUpdate.
- `server/internal/sim/bosses/chest/chest_test.go`: TestChestThreeMinuteWindow, TestGenerationScopedEligibility, TestPersonalLootNoDepletion, TestUnclaimedToRewardClaims.
- `client/Assets/Tests/PlayMode/RelicChestPresentation/RelicChestPresentationTests.cs`: relic, marker and chest ceremony presentation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-091/"

## `IMP-092` — MA_AM Status
id: IMP-092
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../01_gameplay/status_effects.md`, `../01_gameplay/combat.md`, `../03_systems/spirit_beasts.md`, `../07_content/boss_catalog.md`]
adrs: [`0026-just-guard-and-ma-am-status.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0043-spirit-beast-instance-identity.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-016, IMP-022, IMP-057]
owned_paths: [`server/internal/sim/effects/maam/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/`]
contract_inputs: [MA_AM applications, BURN/POISON ticks, active Linh Thú]
contract_outputs: [stack state, consumption and Passive-2 eligibility check]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `status_effects.md` § MA_AM: `NEGATIVE|MA_AM` stacks to 3, consumed on the next BURN/POISON tick with the Linh Thú Passive-2 eligibility check; only ELITE, boss and explicit folklore content may apply it.

## Acceptance
- stacks cap at 3; at 3 the next BURN/POISON tick triggers the Passive-2 check and removes all stacks regardless of result,
- HARD_CONTROL and non-BURN/POISON DOT ticks never trigger,
- content activation rejects MA_AM on normal field monsters.

## Tests
- `server/internal/sim/effects/maam/maam_test.go`: TestMaAmStacksTo3, TestConsumedOnBurnPoisonTick, TestHardControlDoesNotTrigger, TestOnlyEliteBossFolkloreAppliers.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-092/"

## `IMP-023` — Dungeon Runtime
id: IMP-023
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/dungeons.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/dungeon_catalog.md`, `../07_content/encounter_catalog.md`]
adrs: [`0004-party-dungeon-scaling-reward-slots.md`, `0036-seasons-as-launch-infrastructure.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-010, IMP-018, IMP-019, IMP-022]
owned_paths: [`server/internal/sim/dungeons/`, `server/internal/durable/dungeons/`, `client/Assets/Scripts/Systems/Dungeons/`, `client/Assets/Scripts/UI/Dungeons/`, `client/Assets/Tests/PlayMode/DungeonPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [entry request, party snapshot, dungeon/encounter definitions, operation IDs]
contract_outputs: [owned instance/stage state, scaling, clear settlement, reconnect/client result]
consumers_checked: [docs/02_world/dungeons.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/dungeon_catalog.md, docs/07_content/encounter_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement five NORMAL dungeons/stages/checkpoints/scaling/repeat+first-clear settlement.

## Acceptance
- 1p/5p tests and five first-progression-clear EXP values pass,
- five exact dungeon bounds/layout profiles load; mandatory stages, checkpoints, branches and boss areas remain legally connected.

## Tests
- `server/internal/sim/dungeons/dungeons_test.go`: TestFiveNormalDungeonsLifecycle, TestDungeonBoundsAndTopologyProfiles, TestPartyDungeonScalingRewardSlots, TestFirstClearVsRepeatSettlement.
- `client/Assets/Tests/PlayMode/DungeonPresentation/DungeonPresentationTests.cs`: stage/party scaling/reconnect/result states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-023/"

## `IMP-024` — ENDGAME_L60 Variants
id: IMP-024
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/dungeons.md`, `../07_content/dungeon_catalog.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0004-party-dungeon-scaling-reward-slots.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-023]
owned_paths: [`server/internal/sim/dungeons/endgame/`, `client/Assets/Scripts/Systems/Dungeons/Endgame/`, `client/Assets/Tests/PlayMode/EndgameDungeonPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`, `client/Assets/Scripts/UI/`]
contract_inputs: [Lv60 variant definitions, base dungeon runtime, participant state]
contract_outputs: [deterministic endgame mechanics, repeat settlement, variant presentation]
consumers_checked: [docs/02_world/dungeons.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/dungeon_catalog.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement tagged run selection, fixed Lv60 stats, five mechanic remixes, combined `.endgame` reward settlement.

## Acceptance
- no double repeat reward; one-member boss149800 HP before party scaling,
- ENDGAME_L60 reuses the owning dungeon's exact bounds/layout profile without an unversioned geometry fork.

## Tests
- `server/internal/sim/dungeons/endgame/dungeons_endgame_test.go`: TestEndgameL60StatScaling, TestFiveMechanicRemixes, TestEndgameRewardDistribution.
- `client/Assets/Tests/PlayMode/EndgameDungeonPresentation/EndgameDungeonPresentationTests.cs`: Lv60 variant identity/mechanics/reward presentation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-024/"

## `IMP-057` — Linh Thú Companion Runtime
id: IMP-057
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/spirit_beasts.md`, `../07_content/spirit_beast_catalog.md`]
adrs: [`0019-spirit-beast-companion-system.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0043-spirit-beast-instance-identity.md`]
depends_on: [IMP-008, IMP-010, IMP-016, IMP-019, IMP-100]
owned_paths: [`server/internal/sim/beasts/`, `server/internal/durable/beasts/`, `client/Assets/Scripts/Systems/Beasts/`, `client/Assets/Scripts/UI/Beasts/`, `client/Assets/Tests/PlayMode/SpiritBeastPresentation/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [beast instances, active selection, combat event, passive definitions]
contract_outputs: [durable beast state, bounded passive proc result, companion presentation]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement Linh Thú companion runtime for all 10 launch beasts. Identity is composite `(character_id, beast_id)` with no new UUID (ADR-0043). `character_beasts` PK = `(character_id, beast_id)`; `beast_equipment_locations` PK = `(character_id, beast_id, slot_id)` FK → `character_beasts`.
- Implement messages 410–417: `C2S_BEAST_SET_ACTIVE`/`S2C_BEAST_SET_ACTIVE_RESULT`, `C2S_BEAST_EQUIP`/`S2C_BEAST_EQUIP_RESULT`, `C2S_BEAST_UNEQUIP`/`S2C_BEAST_UNEQUIP_RESULT`, `C2S_BEAST_FEED`/`S2C_BEAST_FEED_RESULT`. Payloads carry `operation_id`, `beast_id`, and `slot_id`/`item_instance_id` as needed.
- Equipped `BEAST_EQUIPMENT` occupies `BEAST_EQUIPMENT_SLOT` only; unequipped `BEAST_EQUIPMENT` lives in `CHARACTER_INVENTORY`.
- Distinct from IMP-050 (compile-time passive budget only).

## Acceptance
- 10 launch beasts persist and reconstruct on reconnect; PK is `(character_id, beast_id)`.
- Active beast swap during a PvP match is rejected; PREPARING snapshot includes active `beast_id`.
- Messages 410–417 parse and settle once per `operation_id`.
- 18 `item.beast_eq.*` equip/unequip without an account item vault.

## Tests
- `server/internal/sim/beasts/beasts_test.go`: Unit: composite PK uniqueness; second grant of the same `beast_id` is idempotent.
- `server/internal/sim/beasts/beasts_test.go`: Integration: 410–417 round-trip; unequipped beast_eq in `CHARACTER_INVENTORY`; equipped in `BEAST_EQUIPMENT_SLOT`.
- `server/internal/sim/beasts/beasts_test.go`: Regression: IMP-050 compile pass is not a substitute for this runtime.
- `client/Assets/Tests/PlayMode/SpiritBeastPresentation/SpiritBeastPresentationTests.cs`: summon/passive/proc/reconnect identity projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-057/"

## `IMP-060` — Atlas Journal Runtime
id: IMP-060
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/atlas.md`, `../07_content/atlas_catalog.md`]
adrs: [`0028-atlas-soft-pity-guild-stone-morning-market.md`, `0042-atlas-roster-expansion-104-pages.md`]
depends_on: [IMP-005, IMP-010, IMP-011, IMP-018]
owned_paths: [`server/internal/sim/atlas/`, `server/internal/durable/atlas/`, `client/Assets/Scripts/Systems/Atlas/`, `client/Assets/Scripts/UI/Atlas/`, `client/Assets/Tests/PlayMode/AtlasUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [Atlas source event, page definition, character page state, operation ID]
contract_outputs: [Seen/Studied/Mastered transition, tier grant result, Atlas UI projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement Atlas journal runtime (Seen/Studied/Mastered) on `character_atlas`. Distinct from IMP-050 (beast compile) and IMP-052 (seasons infrastructure).
- Implement `C2S_ATLAS_CLAIM` (504) / `S2C_ATLAS_CLAIM_RESULT` (505): `operation_id`, `atlas_page_id`, `tier`. If auto-settled already, return existing (idempotent). Do not reuse 408.
- `S2C_PROGRESSION_EVENT` (506) for atlas tier-up.

## Acceptance
- Launch atlas pages follow `../03_systems/atlas.md` and `../07_content/atlas_catalog.md`; no world-domain duplicate may be introduced.
- Claim is idempotent; auto-settled returns existing.
- Atlas tier-up grants LIFE_SKILL EXP per the character's current act from the progression-route table.

## Tests
- `server/internal/sim/atlas/atlas_test.go`: Unit: duplicate 504 returns the existing row.
- `server/internal/sim/atlas/atlas_test.go`: Integration: `MONSTER_KILLED` / `FISH_CAUGHT` / `CHEST_OPENED` / `DISH_COOKED` / `BOSS_DEFEATED` promote tiers once.
- `server/internal/sim/atlas/atlas_test.go`: Regression: IMP-052 seasonal chapters reuse this runtime and do not fork a second atlas store.
- `client/Assets/Tests/PlayMode/AtlasUi/AtlasUiTests.cs`: Seen/Studied/Mastered and reward-tier claim states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-060/"

# Task Group — Economy / Items / NPC Services

## `IMP-026` — Equipment Catalog Runtime Expansion
id: IMP-026
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/equipment.md`, `../07_content/equipment_catalog.md`]
adrs: [`0021-hardcore-enhancement-rate-curve.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`]
depends_on: [IMP-003, IMP-012]
owned_paths: [`server/internal/config/equipment/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [equipment source tables, expansion rules, stat/set constraints]
contract_outputs: [168 validated immutable equipment definitions and client-consumable data]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 12 sets/168 definitions/fixed stats/secondary rolls/elements/set thresholds/support signatures.

## Acceptance
- generated IDs/stats/roll persistence tests pass.

## Tests
- `server/internal/config/equipment/equipment_expansion_test.go`: TestTwelveSetsExpansion, TestOneHundredSixtyEightDefinitions, TestSecondaryRollThresholds.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-026/"

## `IMP-088` — Weapon Glow & Aura
id: IMP-088
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/crafting.md`, `../04_architecture/client_assets.md`, `../04_architecture/client_performance.md`, `../07_content/presentation_asset_manifest.md`]
adrs: [`0023-engagement-loops-weapon-glow-bonfire-chivalry-chests-sparring.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-027, IMP-101]
owned_paths: [`client/Assets/Scripts/Systems/WeaponGlow/`, `client/Assets/Tests/EditMode/WeaponGlow/`]
forbidden_paths: [`server/`]
contract_inputs: [authoritative enhancement level]
contract_outputs: [client glow/aura presentation tier]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `crafting.md` § Visual Prestige: glow/aura tiers +8..+16 from the authoritative enhancement level, presentation only, within the preset particle/light budget.

## Acceptance
- tiers +8..+9, +10..+11, +12..+13, +14..+15 and +16 map to the five listed presentations,
- glow never changes stats or hit data and follows the authoritative level only,
- glow respects the active preset particle and light budgets.

## Tests
- `client/Assets/Tests/EditMode/WeaponGlow/WeaponGlowTests.cs`: TestGlowTierFromEnhancementLevel, TestPresentationOnlyNoStats, TestPresetParticleBudget.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-088/"

## `IMP-027` — Crafting / Enhancement
id: IMP-027
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/crafting.md`, `../07_content/crafting_catalog.md`]
adrs: [`0022-four-tier-lucky-charm-system.md`]
depends_on: [IMP-007, IMP-008, IMP-009, IMP-026]
owned_paths: [`server/internal/sim/crafting/`, `server/internal/durable/crafting/`, `client/Assets/Scripts/Systems/Crafting/`, `client/Assets/Scripts/UI/Crafting/`, `client/Assets/Tests/PlayMode/CraftingUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [recipe/enhancement intent, owned inputs, currency, protection item, RNG]
contract_outputs: [atomic craft/enhancement state, committed roll, authoritative UI outcome]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 168 guaranteed recipes, Lucky/Insurance, +0..+16 transition/cost tables.

## Acceptance
- exact transition and expected-cost regression vectors pass.

## Tests
- `server/internal/sim/crafting/crafting_test.go`: TestOneHundredSixtyEightRecipes, TestGuaranteedCraftingSettlement, TestEnhancementPlusZeroToSixteen.
- `client/Assets/Tests/PlayMode/CraftingUi/CraftingUiTests.cs`: recipe/enhancement/charm authoritative outcomes.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-027/"

## `IMP-028` — NPC Services / Shops
id: IMP-028
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../02_world/npcs.md`, `../07_content/npc_shop_catalog.md`]
adrs: [`0028-atlas-soft-pity-guild-stone-morning-market.md`]
depends_on: [IMP-007, IMP-009, IMP-018, IMP-027]
owned_paths: [`server/internal/sim/shops/`, `server/internal/durable/shops/`, `client/Assets/Scripts/Systems/NpcServices/`, `client/Assets/Scripts/UI/Shops/`, `client/Assets/Tests/PlayMode/ShopUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [NPC/interact context, service/shop catalog, owned currency/items]
contract_outputs: [validated NPC transaction/service result and shop UI projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement recovery shop, bound utility shop, crafting/enhancement, storage/auction/travel/respec service gates.

## Acceptance
- price/access/bound-output/travel safety tests pass.

## Tests
- `server/internal/sim/shops/shops_test.go`: TestRecoveryShopPurchases, TestBoundUtilityShopLimits, TestServiceGateValidation.
- `client/Assets/Tests/PlayMode/ShopUi/ShopUiTests.cs`: NPC range/service/price/rejection presentation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-028/"

## `IMP-029` — Direct Trade
id: IMP-029
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/trading_auction.md`, `../06_data/data_model.md`]
adrs: [`0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`]
depends_on: [IMP-007, IMP-008, IMP-009, IMP-011]
owned_paths: [`server/internal/sim/trade/`, `server/internal/durable/trade/`, `client/Assets/Scripts/Systems/Trade/`, `client/Assets/Scripts/UI/Trade/`, `client/Assets/Tests/PlayMode/TradeUi/`]
forbidden_paths: [`server/migrations/`, `proto/`, `server/internal/protocol/v1/`, `client/Assets/Scripts/Protocol/`]
contract_inputs: [two participants, proximity/eligibility, offers, locks, operation ID]
contract_outputs: [atomic trade settlement or whole-trade rejection, client trade FSM]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement same-map/range two-player atomic trade.

## Acceptance
- disconnect/cap/ownership/concurrent mutation rejects whole trade safely.

## Tests
- `server/internal/sim/trade/trade_test.go`: TestSameMapDistanceFourMetersCheck, TestTwoPlayerAtomicExchange, TestTradeEscrowSettlement.
- `client/Assets/Tests/PlayMode/TradeUi/TradeUiTests.cs`: offer/lock/confirm/cancel/disconnect state machine.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-029/"

## `IMP-030` — Auction House
id: IMP-030
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/trading_auction.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0028-atlas-soft-pity-guild-stone-morning-market.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`]
depends_on: [IMP-005, IMP-007, IMP-008, IMP-009]
owned_paths: [`server/internal/durable/auction/`, `client/Assets/Scripts/Systems/Auction/`, `client/Assets/Scripts/UI/Auction/`, `client/Assets/Tests/PlayMode/AuctionUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [listing/buy/cancel intent, escrow item, buyer/seller balances, operation ID]
contract_outputs: [fixed-price listing state, atomic sale/proceeds result, Auction UI projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement listing escrow, fixed-price purchase, fee/tax, pending seller proceeds.

## Acceptance
- buyer debit+tax+item transfer+SOLD+seller credit/proceeds is duplication-safe.

## Tests
- `server/internal/durable/auction/auction_test.go`: TestFixedPriceListingEscrow, TestFixedPricePurchaseSettlement, TestTaxDeductionAndProceedsEscrow.
- `client/Assets/Tests/PlayMode/AuctionUi/AuctionUiTests.cs`: fixed-price list/buy/cancel/proceeds state machine.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-030/"

# Task Group — Build Systems

## `IMP-031` — Soul Contracts
id: IMP-031
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/soul_contracts.md`, `../07_content/soul_catalog.md`]
adrs: [`0012-reward-claim-item-materialization.md`]
depends_on: [IMP-010, IMP-012, IMP-016]
owned_paths: [`server/internal/sim/souls/`, `server/internal/durable/souls/`, `client/Assets/Scripts/Systems/Souls/`, `client/Assets/Scripts/UI/Souls/`, `client/Assets/Tests/PlayMode/SoulUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [Soul instances/EXP intent, active/support selection, build state]
contract_outputs: [durable Soul progression/selection and combat contribution projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement collection/contract limits/Soul EXP/level effects.

## Acceptance
- 25-Soul, duplicate-instance, transfer-lock and EXP-source tests pass.

## Tests
- `server/internal/sim/souls/souls_test.go`: TestSoulCollectionLimits, TestSoulContractEquipEffects, TestSoulEXPGrowthPipeline.
- `client/Assets/Tests/PlayMode/SoulUi/SoulUiTests.cs`: contract/EXP/active-support projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-031/"

## `IMP-032` — Spirit Meridian
id: IMP-032
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/spirit_meridian.md`, `../07_content/build_catalog.md`]
adrs: [`0016-twelve-skill-pool-upgradeable-basics.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`]
depends_on: [IMP-016, IMP-026]
owned_paths: [`server/internal/sim/meridian/`, `server/internal/durable/meridian/`, `client/Assets/Scripts/Systems/Meridian/`, `client/Assets/Scripts/UI/Meridian/`, `client/Assets/Tests/EditMode/MeridianUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [selected build entries, elements, resonance catalog, build lock]
contract_outputs: [deterministic active resonance, durable selection, UI witness]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement eight-BASIC-slot relation matcher and max-three selected effects.

## Acceptance
- all15 resonances have legal selected witnesses.

## Tests
- `server/internal/sim/meridian/meridian_test.go`: TestEightBasicSlotRelationMatcher, TestMaxThreeSelectedMeridianEffects, TestStatBonusAggregation.
- `client/Assets/Tests/EditMode/MeridianUi/MeridianUiTests.cs`: resonance reachability and authoritative activation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-032/"

## `IMP-033` — Formations
id: IMP-033
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/formations.md`, `../07_content/build_catalog.md`]
adrs: [`0016-twelve-skill-pool-upgradeable-basics.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`]
depends_on: [IMP-016, IMP-026]
owned_paths: [`server/internal/sim/formations/`, `server/internal/durable/formations/`, `client/Assets/Scripts/Systems/Formations/`, `client/Assets/Scripts/UI/Formations/`, `client/Assets/Tests/EditMode/FormationUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [six selected ADVANCED slots, formation catalog, build lock]
contract_outputs: [single winning Formation, durable selection, effect/UI projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement six-ADVANCED-slot matcher and one active winner.

## Acceptance
- all 12 Formations have legal selected-winner witnesses.

## Tests
- `server/internal/sim/formations/formations_test.go`: TestSixAdvancedSlotMatcher, TestSingleActiveFormationWinner, TestPartyBuffApplication.
- `client/Assets/Tests/EditMode/FormationUi/FormationUiTests.cs`: formation matching and authoritative activation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-033/"

# Task Group — Social / Guild

## `IMP-034` — Friends / Block / Chat
id: IMP-034
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/social.md`, `../05_network/messages.md`]
adrs: [`0013-canonical-unicode-text-normalization.md`]
depends_on: [IMP-018, IMP-080, IMP-100]
owned_paths: [`server/internal/global/social/`, `server/internal/durable/social/`, `client/Assets/Scripts/Systems/Social/`, `client/Assets/Scripts/UI/Social/`, `client/Assets/Tests/PlayMode/SocialChatUi/`]
forbidden_paths: [`server/internal/sim/combat/`, `server/migrations/`]
contract_inputs: [authenticated social/chat intent, text policy, block/mute/friend state]
contract_outputs: [durable social graph mutation or in-process chat fanout/rejection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement social graph and LOCAL/WORLD/GUILD/party-compatible messaging gates.

## Acceptance
- block/direct-interaction/range/level tests pass.

## Tests
- `server/internal/global/social/social_test.go`: TestSocialGraphFriendBlock, TestChatRateLimitingChannels, TestCanonicalTextNormalization.
- `client/Assets/Tests/PlayMode/SocialChatUi/SocialChatUiTests.cs`: friend/block/mute/chat channel/filter states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-034/"

## `IMP-094` — Chat Moderation & chat_messages
id: IMP-094
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/social.md`, `../06_data/data_model.md`, `../07_security/data_protection.md`, `../07_security/personal_data_register.md`]
adrs: [`0013-canonical-unicode-text-normalization.md`, `0051-first-party-username-password-login.md`]
depends_on: [IMP-034, IMP-080]
owned_paths: [`server/internal/durable/chat/`, `server/internal/global/moderation/`]
forbidden_paths: [`server/migrations/`, `server/internal/sim/`, `client/`]
contract_inputs: [accepted chat messages, reports, moderation actions]
contract_outputs: [chat_messages rows, mute/restriction state, report cases]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `social.md` § Moderation: `chat_messages` persistence with 90-day retention markers, automated unsafe/spam rejection without punitive sanctions, reports, and operator-issued restrictions enforced on send.

## Acceptance
- accepted messages persist to `chat_messages` with the retention deadline consumed by IMP-056,
- automated filtering may reject but never sanctions; sanctions come only from explicit moderation actions,
- a restricted character cannot send in the restricted channels; reports create a case record.

## Tests
- `server/internal/durable/chat/chat_test.go`: TestChatPersistedToChatMessages, TestNinetyDayRetentionDeadline.
- `server/internal/global/moderation/moderation_test.go`: TestAutomatedFilterRejectsNoSanction, TestRestrictionEnforcedOnSend, TestReportCreatesCase.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-094/"

## `IMP-035` — Party
id: IMP-035
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/party.md`, `../05_network/messages.md`]
adrs: [`0004-party-dungeon-scaling-reward-slots.md`]
depends_on: [IMP-018, IMP-080, IMP-100]
owned_paths: [`server/internal/global/party/`, `client/Assets/Scripts/Systems/Party/`, `client/Assets/Scripts/UI/Party/`, `client/Assets/Tests/PlayMode/PartyUi/`]
forbidden_paths: [`server/internal/sim/combat/`, `server/migrations/`]
contract_inputs: [party intent, live sessions/presence, leader/member state]
contract_outputs: [in-process party snapshot/events, invite/join/leave/restart result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement max5 membership/leader/invites/disconnect/dungeon snapshot.

## Acceptance
- concurrent join/leave/reward-range fixtures pass.

## Tests
- `server/internal/global/party/party_test.go`: TestMaxFivePartyMembership, TestLeaderPromotionAndTransfer, TestDisconnectTimeoutGracePeriod.
- `client/Assets/Tests/PlayMode/PartyUi/PartyUiTests.cs`: invite/membership/leader/restart dissolution states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-035/"

## `IMP-036` — Guild Core / Progression
id: IMP-036
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/guild.md`, `../03_systems/guild_progression.md`, `../06_data/physical_schema_contract.md`]
adrs: [`0028-atlas-soft-pity-guild-stone-morning-market.md`]
depends_on: [IMP-007, IMP-034, IMP-100]
owned_paths: [`server/internal/durable/guild/`, `server/internal/global/guild/`, `client/Assets/Scripts/Systems/Guild/`, `client/Assets/Scripts/UI/Guild/`, `client/Assets/Tests/PlayMode/GuildUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [guild intent, actor permissions, currency/progression state]
contract_outputs: [durable guild aggregate mutation, in-process guild fanout, UI result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement roles/lifecycle/create/join/capacity/Guild EXP/Ritual/Blessing.

## Acceptance
- permission, one-guild, Ritual and disband guards pass.

## Tests
- `server/internal/durable/guild/guild_test.go`: TestGuildCreateJoinRoles, TestGuildEXPRitualBlessing, TestGuildCapacityCaps.
- `client/Assets/Tests/PlayMode/GuildUi/GuildUiTests.cs`: membership/role/progression/permission states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-036/"

## `IMP-037` — Guild Storage
id: IMP-037
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/guild_storage.md`, `../06_data/data_model.md`, `../07_security/anti_cheat.md`, `../05_network/messages.md`]
adrs: [`0029-character-resource-isolation.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0049-guild-storage-same-account-transfer-prohibition.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-008, IMP-009, IMP-036]
owned_paths: [`server/internal/durable/guild_storage/`, `client/Assets/Scripts/Systems/GuildStorage/`, `client/Assets/Scripts/UI/GuildStorage/`, `client/Assets/Tests/PlayMode/GuildStorageUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [guild storage intent, permission, item state, operation ID]
contract_outputs: [reservation/claim/deposit/withdraw settlement or durable rejection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement COMMON/RESERVE storage and pending/approved/expired claims.

## Acceptance
- reservation/expiry/full-inventory/restart tests pass,
- withdraw or Reserve delivery of an item deposited by a different character of the same account is rejected with `GUILD_STORAGE_SAME_ACCOUNT`; same-character withdrawal is allowed,
- withdrawing another character's deposit before 72h membership is rejected with `GUILD_MEMBERSHIP_TOO_NEW`,
- storage rows persist `depositor_character_id`/`depositor_account_id`, and cross-character withdrawals update `item_partner_counts`.

## Tests
- `server/internal/durable/guild_storage/guild_storage_test.go`: TestCommonReserveStoragePartitions, TestStorageClaimApprovalFlow, TestClaimExpirationSettlement, TestSameAccountWithdrawRejected, TestSameAccountReserveDeliveryRejected, TestMembershipAgeGate, TestItemPartnerCountsUpdated.
- `client/Assets/Tests/PlayMode/GuildStorageUi/GuildStorageUiTests.cs`: deposit/withdraw/claim rejection and retry states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-037/"

## `IMP-038` — Cosmetics
id: IMP-038
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/cosmetics.md`, `../07_content/cosmetic_catalog.md`]
adrs: [`0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-007, IMP-008, IMP-036, IMP-100]
owned_paths: [`server/internal/durable/cosmetics/`, `client/Assets/Scripts/Systems/Cosmetics/`, `client/Assets/Scripts/UI/Cosmetics/`, `client/Assets/Tests/PlayMode/CosmeticsUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [cosmetic entitlement/equip intent, account/character scope, catalog]
contract_outputs: [durable entitlement/equip state and no-power client projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement play-earned CHARACTER-scoped cosmetics, GUILD crest/banner/shrine, IAP account-entitled store cosmetics, and seasonal free/paid/atlas-title cosmetics. Equip and material/special redemption. Launch stable ID counts: `TITLE_PLAY=125`, `PROFILE_FRAME_PLAY=3`, `APPEARANCE_PLAY=4`, `GUILD=3`, `PLAY_PLUS_GUILD=135`, `COMMON_SINKS=20`, `SPECIAL_CURRENCY_SINKS=20`, `SEASONAL_ATLAS_TITLES=60`, `SEASON_FREE=18`, `SEASON_PAID=18`, `IAP_STORE_IDS=13`, `TOTAL_STABLE_COSMETIC_IDS=284`.

## Acceptance
- `TOTAL_STABLE_COSMETIC_IDS = 284` with the breakdown above; play cosmetics are CHARACTER-scoped; IAP store cosmetics are account-entitled (`account_cosmetic_entitlements`) and equippable on any character of the account; no-power and duplicate/no-double-consume tests pass. Do not treat 18 or 135 as an unexplained launch total.

## Tests
- `server/internal/durable/cosmetics/cosmetics_test.go`: TestPlayEarnedCharacterCosmetics, TestIAPAccountEntitledWardrobe, TestCosmeticEquipValidation.
- `client/Assets/Tests/PlayMode/CosmeticsUi/CosmeticsUiTests.cs`: entitlement/equip/preview/account-vs-character scope.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-038/"

## `IMP-086` — Chivalry Points & Titles
id: IMP-086
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/social.md`, `../06_data/data_model.md`, `../07_content/cosmetic_catalog.md`]
adrs: [`0023-engagement-loops-weapon-glow-bonfire-chivalry-chests-sparring.md`, `0029-character-resource-isolation.md`]
depends_on: [IMP-023, IMP-034, IMP-038]
owned_paths: [`server/internal/durable/chivalry/`, `client/Assets/Scripts/UI/Chivalry/`, `client/Assets/Tests/PlayMode/ChivalryUi/`]
forbidden_paths: [`server/migrations/`, `server/internal/sim/`]
contract_inputs: [qualifying dungeon completion settlements]
contract_outputs: [chivalry UTC-day and lifetime counters, milestone title grants]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `social.md` § Chivalry System: 15 points per qualifying completion, clamped daily cap 100, lifetime total, milestone titles at 500/2,000/5,000; non-spendable score.

## Acceptance
- grants clamp to the 100/day cap exactly (e.g. 90 -> +10),
- duplicate completion settlement never increments either counter twice,
- milestones grant their title cosmetic once and nothing else; points cannot be debited, exchanged or transferred.

## Tests
- `server/internal/durable/chivalry/chivalry_test.go`: TestFifteenPerQualifyingClear, TestDailyCapClamp100, TestDuplicateSettlementNoDoubleCount, TestMilestoneTitles, TestNotSpendable.
- `client/Assets/Tests/PlayMode/ChivalryUi/ChivalryUiTests.cs`: counter and title projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-086/"

## `IMP-085` — Folklore Feats & Titles
id: IMP-085
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/cosmetics.md`, `../07_content/cosmetic_catalog.md`]
adrs: [`0024-fishing-cooking-feats-titles-boss-chest-ceremony.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-019, IMP-022, IMP-027, IMP-038, IMP-040, IMP-058]
owned_paths: [`server/internal/durable/feats/`, `client/Assets/Scripts/UI/Feats/`, `client/Assets/Tests/PlayMode/FeatsUi/`]
forbidden_paths: [`server/migrations/`, `server/internal/sim/`]
contract_inputs: [MONSTER_KILLED, BOSS_DEFEATED, FISH_CAUGHT, ENHANCEMENT_COMPLETED, PVP_SEASON_SETTLED events]
contract_outputs: [feat counters, title entitlements]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `cosmetics.md` § Folklore Feats Tracking for every feat in `cosmetic_catalog.md`: counter models, persistence, idempotent milestone grants of title cosmetics.

## Acceptance
- every catalog feat counts only its listed event filter and grants its title once,
- grants are idempotent per the `cosmetics.md` key; counters survive restart and season boundaries,
- titles grant zero stats, multipliers or hidden perks.

## Tests
- `server/internal/durable/feats/feats_test.go`: TestCatalogFeatCounters, TestFeatGrantIdempotent, TestCountersSurviveSeasonAndRestart, TestTitleGrantsNoStats.
- `client/Assets/Tests/PlayMode/FeatsUi/FeatsUiTests.cs`: progress and unlocked-title states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-085/"

## `IMP-052` — Seasons Infrastructure
id: IMP-052
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/seasons.md`, `../03_systems/atlas.md`, `../02_world/bosses.md`]
adrs: [`0036-seasons-as-launch-infrastructure.md`, `0040-world-consequence-durable-aggregate.md`, `0042-atlas-roster-expansion-104-pages.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-021, IMP-036, IMP-038, IMP-040, IMP-060, IMP-091, IMP-102]
owned_paths: [`server/internal/durable/seasons/`, `server/internal/global/seasons/`, `client/Assets/Scripts/Systems/Seasons/`, `client/Assets/Scripts/UI/Seasons/`, `client/Assets/Tests/PlayMode/SeasonsUi/`]
forbidden_paths: [`server/internal/sim/`]
contract_inputs: [UTC season, season config, Atlas/guild/cosmetic/IAP state]
contract_outputs: [season availability/progression/grants and free/paid client track]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement restart-safe `season_number` derivation: `floor((server_utc_seconds - 1799020800) / 4838400)` using `SEASON_EPOCH_UTC_SECONDS` from `seasons.md`. No season-state row is stored; a restart within a season period must produce the same `season_number`.
- Implement `season_region_index = season_number mod 6` and `featured_zone_id` lookup from the six-entry rotation table in `seasons.md`.
- Implement seasonal Atlas chapter availability gate: 10 pages per season under namespace `atlas.page.season.<season_region_index>.<page_key>`, using existing `character_atlas` table from IMP-060. Pages are permanent once unlocked; never removed at season boundary.
- Implement free seasonal cosmetic track (ungated; no paid entitlement required): season-0 maps `reward_tier.season.0.free.title` → `cosmetic.title.season.0.lang_da_ky_ghe`, `reward_tier.season.0.free.frame` → `cosmetic.frame.season.0`, `reward_tier.season.0.free.shrine` → `cosmetic.shrine.season.0`. Cosmetic grant idempotency key: `season.<season_number>.<cosmetic_id>.<character_id>`.
- Implement paid season cosmetic tiers only (no paid power): season-0 maps `reward_tier.season.0.paid.title` → `cosmetic.title.season.0.paid.dem_lang`, `reward_tier.season.0.paid.frame` → `cosmetic.frame.season.0.paid`, `reward_tier.season.0.paid.emote` → `cosmetic.emote.season.0.paid.chap_tay`. Paid access check through the IMP-102 claim path of the IMP-053 grant: `ACCOUNT_SCOPED_ACCESS` entitlement for `product.service.season_track.<season_id>` with `grant_state = GRANTED` for the requesting `account_id`. This check covers all characters on the account; it does not gate the free track.
- Implement per-character seasonal progression record survival across reconnect and restart (backed by existing `character_atlas` and cosmetic entitlement tables).

## Acceptance
- `season_number` derivation is stateless and restart-safe: service restart within a season returns the same value.
- Seasonal Atlas pages activate only when the character is in or visiting the featured region; they are inaccessible from non-featured regions in the same season.
- Atlas page progress is permanent: reconnect and restart tests confirm no seasonal page record is removed at or after a season boundary.
- Free-track cosmetics grant without a paid entitlement; paid tiers require `ACCOUNT_SCOPED_ACCESS`.
- Cosmetic grants are idempotent: replaying the same `(cosmetic_id, character_id, season_number)` tuple is a safe no-op that returns the existing record.
- Paid track access check returns the correct result for both GRANTED and non-GRANTED `ACCOUNT_SCOPED_ACCESS` entitlements; a second character on the same purchasing account may claim their own paid cosmetic tier without depleting the entitlement.
- No seasonal reward grants a combat stat; paid track adds cosmetic tiers only; content activation rejects any seasonal product that violates `power_granting = false`.
- Anti-FOMO: no streak mechanic or progress decay introduced at any season boundary.
- seasonal Di Tích relics of `../02_world/bosses.md` § Seasonal Di Tich Relics reuse the IMP-091 relic runtime with key `(map_id, channel_id, relic_id)` and write no region marker.

## Tests
- `server/internal/durable/seasons/seasons_test.go`: Unit: `season_number` derivation against known UTC timestamp vectors including boundary values.
- `server/internal/durable/seasons/seasons_test.go`: Unit: `season_region_index = season_number mod 6` for all six indices.
- `server/internal/durable/seasons/seasons_test.go`: Unit: cosmetic grant idempotency key uniqueness across characters and seasons.
- `server/internal/durable/seasons/seasons_test.go`: Integration: Atlas page unlock triggers (`MONSTER_KILLED`, `FISH_CAUGHT`, `CHEST_OPENED`, `DISH_COOKED`, `BOSS_DEFEATED`) produce seasonal page record only when character is in the featured region.
- `server/internal/durable/seasons/seasons_test.go`: Integration: paid track access check returns correct result for GRANTED and non-GRANTED entitlements; second-character claim from same purchase succeeds.
- `server/internal/durable/seasons/seasons_test.go`: Regression: simulated restart at `SEASON_EPOCH_UTC_SECONDS` boundary does not double-advance `season_number`.
- `server/internal/durable/seasons/seasons_test.go`: Regression: cosmetic grant with duplicate operation key is a safe no-op.
- `server/internal/durable/seasons/seasons_test.go`: Regression: seasonal Atlas record and cosmetic entitlements are fully restored after disconnect/restart.
- `client/Assets/Tests/PlayMode/SeasonsUi/SeasonsUiTests.cs`: season derivation, free/paid track, Atlas states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-052/"

## `IMP-093` — Guild Stone
id: IMP-093
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/guild.md`, `../03_systems/seasons.md`]
adrs: [`0028-atlas-soft-pity-guild-stone-morning-market.md`, `0036-seasons-as-launch-infrastructure.md`]
depends_on: [IMP-036, IMP-052, IMP-060]
owned_paths: [`server/internal/durable/guild/stone/`, `client/Assets/Scripts/UI/GuildStone/`, `client/Assets/Tests/PlayMode/GuildStoneUi/`]
forbidden_paths: [`server/migrations/`, `server/internal/sim/`]
contract_inputs: [guild membership history, seasonal Atlas T3 masteries]
contract_outputs: [permanent Guild Stone inscriptions and counts]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `guild.md` § Guild Stone: one stone per Safe Anchor, category completion (including the seasonal category, `>= 30` member T3 masteries in the season), permanent inscriptions and counts; no currency, stats or character cosmetic.

## Acceptance
- every Safe Anchor exposes `object.guild_stone.<map_id>`,
- a seasonal category completes at >= 30 distinct character-page T3 masteries by members at mastery time and adds +1 once per cycle,
- inscriptions are permanent and grant nothing else; repeat cycles add another line.
- the Guild Stone count never decreases at a season boundary.

## Tests
- `server/internal/durable/guild/stone/stone_test.go`: TestStonePerSafeAnchor, TestSeasonalCategoryThreshold30, TestInscriptionPermanentNoRewards, TestRepeatCycleAddsLine.
- `client/Assets/Tests/PlayMode/GuildStoneUi/GuildStoneUiTests.cs`: inscription display states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-093/"

## `IMP-053` — IAP Receipt Verification & Entitlement Grants
id: IMP-053
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/monetization.md`, `../03_systems/account_storage.md`, `../03_systems/cosmetics.md`, `../06_data/data_model.md`, `../06_data/physical_schema_contract.md`, `../07_security/external_integrations.md`]
adrs: [`0029-character-resource-isolation.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0051-first-party-username-password-login.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-038, IMP-100]
owned_paths: [`server/internal/durable/monetization/`, `server/internal/edge/iap/`]
forbidden_paths: [`server/internal/sim/`, `client/`, `server/migrations/`]
contract_inputs: [platform receipt/webhook, provider response, account/product, operation ID]
contract_outputs: [verified entitlement state, deduped notification, refund/suspension result]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Store catalog from `monetization.md` § Store Structure: 13 `cosmetic.iap.*` `DIRECT_ACCOUNT_COSMETIC` products, bundle `product.cosmetic.bundle.nguoi_hung_lang_da` (three grants) and `product.service.season_track.<season_number>` (`ACCOUNT_SCOPED_ACCESS`). No mounts, no character-slot product.
- Receipt verification: opaque receipt -> platform confirmation -> entitlement row; a receipt binds permanently to the first account (`IAP_RECEIPT_ACCOUNT_MISMATCH` + security anomaly otherwise).
- Grant state machine `PENDING -> GRANTED | REFUNDED`, `GRANTED -> REFUNDED | REFUNDED_CONSUMED` with the per-type refund rules: `DIRECT_ACCOUNT_COSMETIC` revokes and deletes the account cosmetic row (`REFUNDED_CONSUMED` plus slot reset if it was ever equipped); `ACCOUNT_SCOPED_ACCESS` revokes every claimed tier cosmetic on all characters (`REFUNDED_CONSUMED` if any tier was claimed); `ONE_SHOT` keeps materialized items.
- `account_refund_consumed_events` ledger and rolling 180-day `iap_refund_consumed_score`; at 2 the account becomes `SUSPENDED_PAYMENT_RECONCILIATION` (blocks IAP, character creation, Ranked PvP).
- At most one live season track per `(account, season_number)`; a second verified receipt is not granted and is flagged for platform refund.

## Acceptance
- no entitlement row exists without platform confirmation; a forged receipt is rejected first,
- the same receipt from another account returns `IAP_RECEIPT_ACCOUNT_MISMATCH`, creates nothing and emits the anomaly signal,
- store grants write `account_cosmetic_entitlements` for `cosmetic.iap.*`; no `item.cosmetic.*` character item is created; the bundle grants exactly its three cosmetics,
- each refund path produces the state and revocation listed in `monetization.md` for its entitlement type,
- suspension happens at the 2nd `REFUNDED_CONSUMED` inside 180 days; one event does not suspend,
- duplicate receipt on a GRANTED entitlement returns the existing record; a second track receipt for the same season is not granted,
- any product with `power_granting`, `tradable`, `durability` or `rent` = true is rejected at content activation.

## Tests
- `server/internal/durable/monetization/monetization_test.go`: TestGrantStateMachine, TestDirectCosmeticRefundRevokes, TestEquippedCosmeticRefundConsumed, TestSeasonTrackRefundRevokesAllCharacters, TestOneShotRefundKeepsItems, TestSuspensionThreshold180Days, TestOneTrackPerSeason, TestDuplicateReceiptNoop, TestPowerGrantingProductRejected.
- `server/internal/edge/iap/iap_test.go`: TestForgedReceiptRejected, TestCrossAccountReceiptMismatch, TestBundleGrantsThree.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-053/"

## `IMP-102` — Entitlement Claims & Store Client
id: IMP-102
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/account_storage.md`, `../03_systems/monetization.md`, `../03_systems/seasons.md`, `../03_systems/cosmetics.md`]
adrs: [`0029-character-resource-isolation.md`, `0036-seasons-as-launch-infrastructure.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-010, IMP-053, IMP-066]
owned_paths: [`server/internal/durable/monetization/claims/`, `client/Assets/Scripts/Systems/Store/`, `client/Assets/Scripts/UI/Store/`, `client/Assets/Tests/PlayMode/StoreUi/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [GRANTED entitlements, selected character, reward tiers]
contract_outputs: [per-character claims, store/panel UI states]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement the entitlement claim path: `ONE_SHOT` materialization through Reward Claims and `ACCOUNT_SCOPED_ACCESS` per-character tier claims keyed `account_entitlement_id.character_id.reward_tier_id` until `claim_deadline_at` (season end + 14 days); implement the store/entitlement panel UI showing per-character ownership before purchase and never trusting receipts.

## Acceptance
- the composite key prevents a second claim by the same character; another character claims its own instance; the access entitlement is never depleted,
- claims after `claim_deadline_at` are rejected; a tier already owned from an earlier cycle is a no-op,
- a `ONE_SHOT` claim materializes once to the selected character,
- store UI states come only from server results.

## Tests
- `server/internal/durable/monetization/claims/claims_test.go`: TestSeasonTrackCompositeClaimKey, TestSecondCharacterClaimsOwnInstance, TestClaimDeadline, TestAlreadyOwnedTierNoop, TestOneShotClaimOnce.
- `client/Assets/Tests/PlayMode/StoreUi/StoreUiTests.cs`: catalog/purchase/claim/refund/suspension states without trusting receipts.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-102/"

# Task Group — Competitive Modes

## `IMP-039` — PvP Build Snapshot / Transform
id: IMP-039
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/pvp.md`, `../01_gameplay/combat.md`]
adrs: [`0017-percentage-dodge-stat-no-active-dodge.md`, `0034-just-guard-edge-trigger-streak.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-012, IMP-017, IMP-031, IMP-032, IMP-033, IMP-057, IMP-058, IMP-059, IMP-060]
owned_paths: [`server/internal/sim/pvp/`, `server/internal/durable/pvp/`, `client/Assets/Scripts/Systems/Pvp/`, `client/Assets/Scripts/UI/Pvp/`, `client/Assets/Tests/PlayMode/PvpBuildUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [authoritative build state, mode transform, active beast, preparation lock]
contract_outputs: [immutable PvP snapshot, temporary transformed stats, locked client projection]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement competitive build lock, reference-vector transform/caps, control DR. PREPARING snapshot includes active Linh Thú.

## Acceptance
- transformed state is temporary and mutation stays locked through match; Linh Thú swap during match is rejected.

## Tests
- `server/internal/sim/pvp/pvp_test.go`: TestPVPBuildSnapshotNormalization, TestReferenceVectorTransform, TestControlDiminishingReturns.
- `client/Assets/Tests/PlayMode/PvpBuildUi/PvpBuildUiTests.cs`: preparing snapshot/lock/transform projection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-039/"

## `IMP-040` — Duel / Ranked Duel
id: IMP-040
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/pvp.md`, `../05_network/messages.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-010, IMP-034, IMP-039]
owned_paths: [`server/internal/sim/pvp/duel/`, `server/internal/durable/pvp/duel/`, `server/internal/global/matchmaking/duel/`, `client/Assets/Scripts/Systems/Pvp/Duel/`, `client/Assets/Scripts/UI/Pvp/Duel/`, `client/Assets/Tests/PlayMode/DuelUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [queue/duel intent, snapshot, opponent/result, season/rating state]
contract_outputs: [duel instance/result, idempotent rating/reward settlement, UI FSM]
consumers_checked: [docs/03_systems/pvp.md, docs/04_architecture/physics_geometry_contract.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement duel/Bo3 lifecycle/MMR/season/reconnect/surrender/reward eligibility.

## Acceptance
- rating/reward/VOID/abandon/first5-bound-per-day tests pass,
- `map.pvp.duel_court` bounds/topology and `0.001m` mirror parity pass.

## Tests
- `server/internal/sim/pvp/duel/duel_test.go`: TestDuelBo3Lifecycle, TestDuelCourtGeometryMirrorParity, TestRatingSettlementIdempotency, TestDisconnectSurrenderResolution.
- `client/Assets/Tests/PlayMode/DuelUi/DuelUiTests.cs`: queue/accept/active/result/void state machine.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-040/"

## `IMP-087` — Open Sparring Ring
id: IMP-087
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/pvp.md`, `../02_world/world_rules.md`, `../05_network/messages.md`]
adrs: [`0023-engagement-loops-weapon-glow-bonfire-chivalry-chests-sparring.md`, `0035-spawn-density-increase.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-018, IMP-039]
owned_paths: [`server/internal/sim/pvp/sparring/`, `client/Assets/Scripts/Systems/Pvp/Sparring/`, `client/Assets/Tests/PlayMode/SparringPresentation/`]
forbidden_paths: [`server/migrations/`, `server/internal/durable/`]
contract_inputs: [C2S_SPARRING_REQUEST/ACCEPT, ring occupancy]
contract_outputs: [zero-stake duel lifecycle, HP/MP restore]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `pvp.md` § Open Sparring Ring: request/accept on the `sparring_ring.<map_id>` platform in the same channel, consumables disabled, end at HP 1 with full HP/MP restore.

## Acceptance
- both combatants must stand on the ring in the same channel; either may request, the target accepts,
- consumables are disabled; the duel ends when either HP reaches 1 and both are restored to 100% HP/MP,
- sparring causes no death, item, currency or rating change.

## Tests
- `server/internal/sim/pvp/sparring/sparring_test.go`: TestRequestAcceptOnRing, TestConsumablesDisabled, TestEndsAtOneHpAndRestores, TestZeroStake.
- `client/Assets/Tests/PlayMode/SparringPresentation/SparringPresentationTests.cs`: request/active/victory presentation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-087/"

## `IMP-041` — Five Element Arena
id: IMP-041
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/pvp.md`, `../05_network/messages.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-035, IMP-039, IMP-040]
owned_paths: [`server/internal/sim/pvp/arena/`, `server/internal/durable/pvp/arena/`, `server/internal/global/matchmaking/arena/`, `client/Assets/Scripts/Systems/Pvp/Arena/`, `client/Assets/Scripts/UI/Pvp/Arena/`, `client/Assets/Tests/PlayMode/ArenaUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [party queue, build snapshots, arena rules/objectives]
contract_outputs: [arena instance/score/result settlement and client match states]
consumers_checked: [docs/03_systems/pvp.md, docs/04_architecture/physics_geometry_contract.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement5v5 altars/attunements/Harmony Pulse/overtime/score.

## Acceptance
- objective-first scoring and team/friendly-fire tests pass,
- `map.pvp.five_element_arena` bounds, tri-altar topology and `0.001m` mirror parity pass.

## Tests
- `server/internal/sim/pvp/arena/arena_test.go`: TestFiveElementAltarAttunements, TestArenaGeometryMirrorParity, TestHarmonyPulseCapture, TestOvertimeScoreResolution.
- `client/Assets/Tests/PlayMode/ArenaUi/ArenaUiTests.cs`: party queue, score, reconnect, and result states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-041/"

## `IMP-042` — Guild War
id: IMP-042
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/guild_war.md`, `../05_network/messages.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`]
depends_on: [IMP-010, IMP-036, IMP-039]
owned_paths: [`server/internal/sim/guild_war/`, `server/internal/durable/guild_war/`, `server/internal/global/matchmaking/guild_war/`, `client/Assets/Scripts/Systems/GuildWar/`, `client/Assets/Scripts/UI/GuildWar/`, `client/Assets/Tests/PlayMode/GuildWarUi/`]
forbidden_paths: [`server/migrations/`]
contract_inputs: [guild roster/queue, build snapshots, war objectives, weekly state]
contract_outputs: [Guild War instance/result/rating/reward settlement and client states]
consumers_checked: [docs/03_systems/guild_war.md, docs/04_architecture/physics_geometry_contract.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement 10v10 roster/matchmaking/five seals/rating/weekly progression/reward caps.

## Acceptance
- no Blessing advantage, no territory, and the bound cap survives guild change,
- `map.guild_war.five_seal_conflict` bounds, braided-front topology and `0.001m` mirror parity pass.

## Tests
- `server/internal/sim/guild_war/guild_war_test.go`: TestTenVersusTenRosterMatchmaking, TestGuildWarGeometryMirrorParity, TestFiveSealsCaptureMechanics, TestWeeklyRatingProgression.
- `client/Assets/Tests/PlayMode/GuildWarUi/GuildWarUiTests.cs`: roster/accept/objective/result/void states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-042/"

# Task Group — Hardening / Release

## `IMP-043` — Observability / Audit Coverage
id: IMP-043
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../08_scale_ops/observability.md`, `../08_scale_ops/caching.md`, `../09_testing/backend.md`]
adrs: [`0010-exact-technology-version-pinning.md`]
depends_on: [IMP-005, IMP-007, IMP-008, IMP-010, IMP-011, IMP-012, IMP-026, IMP-027, IMP-029, IMP-030, IMP-036, IMP-037, IMP-038, IMP-052, IMP-053, IMP-057, IMP-098, IMP-100]
owned_paths: [`server/internal/observability/audit/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [typed runtime/durable/security events with operation/source/revision context]
contract_outputs: [bounded metrics, structured logs, traces, audit records, dashboards]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Build audit events, the required metric/alert registry and the seven mandatory launch dashboards on the IMP-098 core. Instrument realtime, network, PostgreSQL, economy/integrity, AOI shedding, WorldConsequence load, Spirit Surge coordination, anti-RMT queries, and movement-edge delivery without changing gameplay authority.

## Acceptance
- operation/source/revision/retry/invariant reason is visible for every critical value or authority failure,
- the canonical correlation context propagates across Edge, Sim, Global, and Durable boundaries when applicable,
- every mandatory metric, threshold, dashboard, and alert class in `observability.md` has an executable registration test,
- telemetry failure never changes gameplay outcome or blocks the fixed-step loop.

## Tests
- `server/internal/observability/audit/audit_test.go`: TestCrossBoundaryTraceContext, TestErrorInvariantReasonLogging, TestRequiredMetricAndAlertRegistry, TestSevenDashboardCoverage, TestTelemetryFailureDoesNotChangeAuthority.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-043/"

## `IMP-069` — Server Composition Root and Lifecycle Wiring
id: IMP-069
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/backend.md`, `../04_architecture/service_boundaries.md`, `../04_architecture/system_overview.md`, `../04_architecture/authority.md`, `../04_architecture/realtime_loop.md`, `../05_network/reconnect.md`, `../08_scale_ops/deployment.md`, `architecture_conformance.md`]
adrs: [`0007-single-owner-fixed-step-simulation.md`, `0030-one-account-one-live-session.md`, `0038-discrete-movement-edge-input-message.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0040-world-consequence-durable-aggregate.md`, `0044-launch-topology-single-binary-role-modes.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-020, IMP-024, IMP-025, IMP-028, IMP-041, IMP-042, IMP-049, IMP-050, IMP-051, IMP-054, IMP-055, IMP-062, IMP-077, IMP-084, IMP-085, IMP-086, IMP-087, IMP-089, IMP-090, IMP-092, IMP-093, IMP-103]
owned_paths: [`server/cmd/server/`, `server/internal/app/`]
forbidden_paths: [`server/cmd/edge/`, `server/cmd/sim/`, `server/cmd/durable/`, `server/cmd/global/`]
contract_inputs: [typed Edge/Sim/Durable/Global packages, validated config/content/schema compatibility, lifecycle signals]
contract_outputs: [one thinhthan-server composition root, startup readiness, graceful drain/shutdown, checkpoint-backed restart]
consumers_checked: [docs/04_architecture/backend.md, docs/04_architecture/service_boundaries.md, docs/08_scale_ops/deployment.md, docs/10_implementation/architecture_conformance.md]

## Change
Wire Edge, Sim, Durable, and Global into one production Go process through explicit dependency injection. Start Durable/config validation before Global/Sim and accept sessions only after readiness succeeds. On termination, stop accepting new sessions, drain in-flight work within canonical budgets, persist required durable work, and restore simulation state from the latest valid checkpoint after process restart.

## Acceptance
- the build produces exactly one production binary named `thinhthan-server`; there is no production `--role` or `SERVER_ROLE` split,
- every registered handler, simulation system, durable repository, and Global coordinator required by completed dependencies is reachable from the composition root,
- startup fails closed on database, schema, protocol, or content-revision incompatibility and never reports ready early,
- graceful shutdown stops new sessions, performs bounded drain/persistence, and exits deterministically,
- full-process restart restores checkpoint-backed state while ordinary parties are intentionally not restored,
- subsystem calls remain in-process typed calls or bounded queues; no internal network RPC, Redis, Kafka, NATS, or leader lease is introduced.

## Tests
- `server/internal/app/app_test.go`: TestCompositionGraphComplete, TestStartupOrderAndReadiness, TestFailClosedCompatibility, TestGracefulDrain, TestCheckpointRestartAndPartyDrop.
- `server/cmd/server/main_test.go`: TestSingleProductionBinary, TestNoRoleSplit, TestSignalDrivenShutdown.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned composition registrations, role-specific entry points, or lifecycle fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-069/"

## `IMP-044` — Fault / Restart Matrix
id: IMP-044
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../09_testing/backend.md`, `../09_testing/strategy.md`]
adrs: [`0011-postgresql-relational-persistence.md`]
depends_on: [IMP-010, IMP-018, IMP-022, IMP-023, IMP-029, IMP-030, IMP-035, IMP-040, IMP-042, IMP-043, IMP-069]
owned_paths: [`server/internal/testing/fault/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [mandatory fault points, committed state, restart/reconnect policy]
contract_outputs: [deterministic recovery outcomes and no-loss/no-duplication fault report]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Inject restart/disconnect/timeouts at transaction/state boundaries.

## Acceptance
- no duplicate/lost committed persistent state across mandatory matrix.

## Tests
- `server/internal/testing/fault/fault_test.go`: TestRestartAtTransactionBoundary, TestTimeoutRollbackConsistency, TestDisconnectionReconnection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-044/"

## `IMP-045` — Security Abuse Suite
id: IMP-045
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_security/validation.md`, `../07_security/anti_cheat.md`, `../07_security/rate_limits.md`, `../07_security/auth.md`, `../07_security/session.md`, `../07_security/external_integrations.md`, `../09_testing/network.md`]
adrs: [`0008-client-network-transport-protocol.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0051-first-party-username-password-login.md`, `0052-single-launch-world.md`, `0053-durable-contract-reconciliation.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-013, IMP-014, IMP-018, IMP-028, IMP-029, IMP-030, IMP-034, IMP-035, IMP-036, IMP-040, IMP-053, IMP-069, IMP-100]
owned_paths: [`server/internal/edge/security/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [authenticated external requests, security limits, ownership state, approved limiter]
contract_outputs: [bounded rejection/closure, zero unauthorized mutation, security audit evidence]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement the security abuse suite against `../07_security/external_integrations.md` (no SERVER_ROLE env contract; Go standard-library token bucket). Authorization/replay/rate/size/forged-ownership/value tests must pass with zero security blocker.

## Acceptance
- all authentication, authorization, replay, rate, size, overflow, and forged-ownership cases in the owning security/network specs reject safely,
- no rejected request mutates durable or simulation state,
- secrets/tokens are redacted and security-sensitive failures emit bounded audit events,
- the approved rate limiter is deterministic under concurrent limit-boundary tests,
- security blocker count is zero.

## Tests
- `server/internal/edge/security/security_test.go`: TestAuthorizationTokenValidation, TestMessageReplayRejection, TestRateLimitTierEnforcement.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-045/"

## `IMP-046` — Load / Capacity Suite
id: IMP-046
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../08_scale_ops/capacity.md`, `../09_testing/load.md`]
adrs: [`0020-map-channel-capacity-contract.md`, `0035-spawn-density-increase.md`, `0039-entity-capacity-model-and-ai-budget-classes.md`, `0052-single-launch-world.md`]
depends_on: [IMP-018, IMP-019, IMP-034, IMP-035, IMP-041, IMP-042, IMP-055, IMP-069]
owned_paths: [`deploy/load/`, `server/internal/testing/load/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [production fingerprint, synthetic users, channel/instance scenarios, budgets]
contract_outputs: [load metrics/report, capacity pass/fail, reproducible environment fingerprint]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Build reproducible component, partition, and full-service load scenarios with active gameplay-equivalent clients and production-fingerprint capture. Exercise every mandatory scenario in `../09_testing/load.md`, measure `MAX_PARTITIONS_PER_PROCESS`, re-measure saturated-AOI bandwidth, and record Go/PostgreSQL/network stability plus correctness results.

## Acceptance
- ramp, 60-minute 10k hold, six-hour soak, reconnect/deploy/fault, scheduler, Spirit Surge, boss/dungeon, and database-degradation scenarios meet canonical thresholds,
- hotspot 42 `AI_CLASS_NAMED_MECHANIC` + 18 players keeps p95 tick below 35 ms; entity cap permits 80 and rejects the 81st,
- saturated 40-entity AOI bandwidth is measured and the release budget is updated from evidence rather than assumed,
- numeric `MAX_PARTITIONS_PER_PROCESS`, measurement timestamp, hardware fingerprint, traffic mix, seed, revisions, and infrastructure shape are recorded,
- no authority/value duplication, unbounded queue/goroutine/memory growth, pool exhaustion without backpressure, OOM, or panic loop occurs,
- the capacity plan retains at least 30% peak headroom or records a demonstrated safe scaling path.

## Tests
- `server/internal/testing/load/load_test.go`: TestMandatoryScenarioManifest, TestTenThousandActivePlayerFingerprint, TestHotspotFortyTwoNamedPlusEighteen, TestEntityCapEightyPermitEightyOneReject, TestChannelCapacityEighteen, TestSLOAndCorrectnessFailClosed, TestCapacityMeasurementMetadata.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-046/"

## `IMP-047` — Migration / Backup / Restore Rehearsal
id: IMP-047
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../08_scale_ops/backup_recovery.md`, `../08_scale_ops/deployment.md`, `../06_data/migrations.md`]
adrs: [`0011-postgresql-relational-persistence.md`, `0048-character-update-timestamp.md`, `0052-single-launch-world.md`]
depends_on: [IMP-005, IMP-043]
owned_paths: [`server/internal/testing/migration/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [migration set, representative snapshot, backup artifact, recovery targets]
contract_outputs: [apply/down/apply and restore evidence with schema/data integrity]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Automate fresh/apply/down/apply and supported-version migration rehearsals, continuous-backup/PITR verification, and an isolated restore drill that follows `backup_recovery.md`, including WorldConsequence, expiry, escrow, proceeds, Reward Claim, content/schema, and idempotency reconciliation.

## Acceptance
- fresh and representative supported snapshots migrate with invariant checks and restart-safe bounded backfills,
- destructive production down-migration is never used where forward-fix or PITR is required,
- verified restore meets the RPO <= 5 minutes and RTO <= 60 minutes targets and records measured duration,
- absent/corrupt WorldConsequence data fails restore; zero rows pass only for a proven pre-defeat snapshot,
- restored-as-of time is used to re-evaluate relic/cosmetic expiry and external side effects reconcile by operation/audit ID,
- post-restore login, character, inventory, economy, escrow, Reward Claim, content-revision, and schema-compatibility smoke tests pass.

## Tests
- `server/internal/testing/migration/migration_test.go`: TestRepresentativeSnapshotMigration, TestRollbackIntegrityRehearsal, TestSchemaChecksumDeterministic, TestPITRRestoreReconciliation, TestWorldConsequenceRestoreGuard, TestRestoredAsOfExpiryEvaluation.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-047/"

## `IMP-048` — Launch Candidate Gate
id: IMP-048
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`definition_of_done.md`, `milestones.md`, `audit_gates.md`, `../08_scale_ops/deployment.md`, `../08_scale_ops/sharding.md`, `../09_testing/strategy.md`, `../09_testing/test_and_release_evidence.md`, `../04_architecture/client_performance.md`]
adrs: [`0050-windows-only-ci-and-auto-merge.md`, `0052-single-launch-world.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-044, IMP-045, IMP-046, IMP-047, IMP-096]
owned_paths: [`docs/10_implementation/release/`, `server/internal/testing/release/`, `deploy/prod/`]
forbidden_paths: [`server/cmd/server/`]
contract_inputs: [server/client/content/schema artifacts and all task/milestone evidence]
contract_outputs: [immutable release manifest, one-process deployment, rollback/forward-fix runbooks]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Assemble the immutable server/client/content/schema release manifest, the static server binary artifact and `deploy/prod/thinhthan-server.service` (single process, ADR-0052), rollback/forward-fix runbooks, and reproducible M10 CI evidence.

## Acceptance
- every M10 requirement in `milestones.md` and `definition_of_done.md` passes from reproducible CI/release evidence,
- deployment runs only `thinhthan-server`; no role-specific server workload, Redis/Kafka/NATS, or `global_leader_lease`,
- manifest binds server artifact, client builds, Addressables catalog, protocol range, schema migration set, content revision, and checksums,
- schema-coupled content uses a compensating migration or forward-only fix plan rather than unsafe binary rollback,
- rollback/restore rehearsal succeeds and all known data-loss, duplication, progression-softlock, economy-exploit, and security-blocker counts are zero.
- PERF-003: ANDROID_MIN and ANDROID_REC frame pacing passes in an IMP-096 device run on the release commit; the gate waits for Test Lab quota and never skips,
- PERF-012: the 30-minute sustained proxies pass on Test Lab for the release commit.

## Tests
- `server/internal/testing/release/release_test.go`: TestMilestoneTenAllGatesPass, TestZeroKnownSoftlocks, TestReleaseManifestHashConsistency, TestOneProcessDeploymentManifest, TestReleaseCommitDevicePerfPassed (PERF-003, PERF-012), TestDeferredQuotaWaitsNeverSkips.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-048/"

## `IMP-049` — TTK, Survivability, and Skill-Reach Re-verification
id: IMP-049
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/balance_validation.md`, `../07_content/class_skill_catalog.md`, `../01_gameplay/combat.md`, `../01_gameplay/skills.md`]
adrs: [`0016-twelve-skill-pool-upgradeable-basics.md`, `0031-exp-scale-x100-and-corrected-act-budgets.md`, `0032-seven-channel-exp-source-portfolio.md`, `0034-just-guard-edge-trigger-streak.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0038-discrete-movement-edge-input-message.md`, `0047-skill-reach-budget-and-collider-aware-resolution.md`, `0054-wire-message-completion.md`]
depends_on: [IMP-004, IMP-026]
owned_paths: [`server/internal/config/validation/balance/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [compiled builds/content, deterministic combat fixtures, balance windows, 45 launch skill geometries, ADR-0046 collider profiles]
contract_outputs: [TTK/survivability/reach report and activation-gate pass/fail]
consumers_checked: [docs/01_gameplay/combat.md, docs/01_gameplay/skills.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/class_skill_catalog.md, docs/07_content/integration_validation.md, docs/07_content/balance_validation.md, docs/09_testing/gameplay.md, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Re-verify all existing TTK windows and the heavy-hit survivability window against current equipment roll magnitudes now that the four new combat stats (LIFESTEAL, REFLECT, ABSORB, HEAL_REDUCTION) are present in `../07_content/balance_validation.md`. Also emit and gate the complete ADR-0047 skill-reach/camera-readability matrix. This is a release gate: every window and reach rule must remain inside its declared guardrail. If a combat window falls outside, the fix is to adjust equipment secondary roll magnitudes — never the guardrail value.

## Acceptance
- tooling run against the pinned reference character (balance_validation.md synthetic Lv60) produces TTK results for all window brackets,
- every existing TTK window falls within its guardrail,
- heavy-hit survivability window falls within its guardrail,
- reject rules for LIFESTEAL / REFLECT / ABSORB / HEAL_REDUCTION pass at cap values,
- exactly 45 launch primary geometries pass their role bands and envelope ceilings,
- ranged-basic/melee separation ratio is at least `2.50` and reference-camera horizontal telegraph margin is at least `1.8m`,
- exact-edge hit and positive-separation miss fixtures pass for every authoritative entity collider profile,
- if any window is outside a guardrail, equipment secondary roll magnitudes are adjusted and the run is repeated until all windows pass — guardrails are never adjusted,
- tooling run evidence is attached to the activation artifact for IMP-004.

## Tests
- `server/internal/config/validation/balance/balance_test.go`: TestTTKWindowsAgainstFourNewStats, TestHeavyHitSurvivabilityGuardrail, TestSecondaryRollBounds, TestSkillReachMatrix45, TestSkillReachSeparationRatio, TestSkillCameraReadabilityMargin, TestSkillColliderBoundaryProfiles.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-049/"

## `IMP-050` — Spirit Beast Passive Budget Compile Validation
id: IMP-050
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../03_systems/spirit_beasts.md`, `../07_content/spirit_beast_catalog.md`]
adrs: [`0019-spirit-beast-companion-system.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0043-spirit-beast-instance-identity.md`]
depends_on: [IMP-003, IMP-004]
owned_paths: [`server/internal/config/validation/beast/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [compiled spirit-beast definitions and passive budget rules]
contract_outputs: [deterministic passive-budget diagnostics and compile rejection/pass]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
Implement the compile-time evaluation for all Spirit Beast passive budgets (Rules A–D declared in `../03_systems/spirit_beasts.md`) against the pinned reference stats in `../01_gameplay/stats.md`. Evaluation must run at content-activation time alongside the flat-stat budget check.

## Acceptance
- pipeline evaluates every beast Passive 1 against Rules A / B / C / D at Lv60 for all 10 launch beasts,
- a missing reference stat value in stats.md is a compile error that blocks activation,
- any beast whose Passive 1 exceeds its applicable ceiling at Lv60 is rejected with the offending rule and value cited,
- ICD ladder check rejects any Passive 2 ladder where two tiers compile to the same effective ICD value (clamp to [45s, 90s]) — OBJ-SBB-003 class defect,
- Kill/Assist Resource Restore payload check rejects any instance where payload_pct > 0.03 or payload includes a banned modifier,
- all 10 launch beasts pass without rejection.

## Tests
- `server/internal/config/validation/beast/beast_budget_test.go`: TestBeastPassiveBudgetRulesAD, TestTenLaunchBeastsPassiveValidation, TestResourceRestoreCaps.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-050/"

## `IMP-051` — Drop Table Coverage Invariant for Monster Roster Growth
id: IMP-051
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/drop_tables.md`, `../07_content/monster_catalog.md`]
adrs: [`0031-exp-scale-x100-and-corrected-act-budgets.md`, `0035-spawn-density-increase.md`]
depends_on: [IMP-003, IMP-019]
owned_paths: [`server/internal/config/validation/drop/`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [monster roster and drop-table catalog]
contract_outputs: [coverage diagnostics rejecting every monster without a drop table]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
The monster roster expanded from 35 to 58 entries; all 23 new monsters received drop tables in the same content commit. Record and enforce the invariant that any future monster roster addition must land with a complete drop table entry in the same content commit.

## Acceptance
- content compiler rejects a monster definition that has no corresponding drop table entry in drop_tables.md,
- all 23 roster additions (entries 36–58) pass this check against the current drop_tables.md,
- a synthetic test monster definition without a drop table entry causes compile rejection,
- the constraint is noted in the content pipeline documentation.

## Tests
- `server/internal/config/validation/drop/drop_coverage_test.go`: TestMonsterRosterDropTableCoverage, TestTwentyThreeRosterAdditionsCheck, TestSyntheticMissingDropRejection.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-051/"

## `IMP-054` — Anti-RMT Behavioral Signals
id: IMP-054
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_security/anti_cheat.md`, `../03_systems/trading_auction.md`]
adrs: [`0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0053-durable-contract-reconciliation.md`]
depends_on: [IMP-001, IMP-007, IMP-029, IMP-030]
owned_paths: [`server/internal/durable/anti_rmt/`]
forbidden_paths: [`server/migrations/`, `server/internal/sim/`, `client/Assets/Scripts/UI/`]
contract_inputs: [durable economy events, character level/age, rolling-window definitions]
contract_outputs: [persistent anti-RMT aggregates, eligibility decisions, audit signals]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement two rolling-window economy signals on a **persistent aggregation table** (not in-memory): signal definitions, rolling window durations, and per-account accumulation as specified in ADR-0041 and `anti_cheat.md`. The table must survive server restart without losing rolling-window history.
- Implement level-10 trade gate: characters below Level 10 cannot initiate or receive direct trades or post Auction House listings; rejection includes a client-visible reason code.
- Implement 24-hour new-character trade gate: `character.age_hours >= 24` is required to initiate or receive direct trades or post Auction House listings; rejection includes a client-visible reason code. This is character age, not account age.
- Implement signal emission on all qualifying economy events: direct trade (both parties), Auction House listing, Auction sale proceeds settlement, and any NPC transaction classified as a qualifying transfer in ADR-0041.

## Acceptance
- Level-10 trade gate: server rejects trade initiation from or to a character below Level 10 with a reason code; a Level 9 character's trade request is rejected, a Level 10 character's request is permitted.
- 24-hour gate: server rejects trade initiation from characters whose `character.age_hours < 24`; a 23-hour-old character is rejected, a 25-hour-old character is permitted.
- Rolling-window signal accumulation survives simulated server restart: post-restart rolling totals are consistent with pre-restart history (no lost rows, no reset to zero).
- Signal emission fires on every qualifying economy event; missing signal emission for a direct trade or Auction sale is a test failure.
- Trade gates are enforced server-side; a client-provided character level or character age is never trusted.

## Tests
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: rolling-window arithmetic, restart persistence, trade gates, and forged age/level rejection.
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: Unit: rolling-window accumulation arithmetic for both signals over a multi-day fixture.
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: Integration: level-10 gate enforced on direct trade and Auction House endpoints (boundary: Level 9 rejects, Level 10 permits).
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: Integration: 24-hour gate enforced on `character.age_hours` (boundary: 23h rejects, 25h permits); account age is not the predicate.
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: Regression: rolling-window accumulation table persists across simulated restart; totals are unchanged after restart.
- `server/internal/durable/anti_rmt/anti_rmt_test.go`: Regression: trade gate cannot be bypassed by a forged or client-declared character level or character age.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-054/"

## `IMP-056` — Retention & Erasure Engine
id: IMP-056
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_security/data_protection.md`, `../07_security/personal_data_register.md`, `../06_data/data_model.md`]
adrs: [`0011-postgresql-relational-persistence.md`, `0051-first-party-username-password-login.md`]
depends_on: [IMP-043, IMP-094, IMP-100]
owned_paths: [`server/internal/durable/privacy/`]
forbidden_paths: [`server/internal/sim/`, `client/`, `server/migrations/`]
contract_inputs: [account request, personal-data register, retention/legal-hold policy]
contract_outputs: [schema-valid export or safe deletion/anonymization with no orphaned value]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement per-table data retention schedules as defined in `../07_security/data_protection.md`. Automated deletion or anonymization jobs run on schedule for each table.
- Implement data-subject deletion request flow: removes personally identifiable data from all tables listed in `data_protection.md` while preserving audit records required by legal hold; applies anonymization where full deletion is not permissible. Account assets (escrow, pending rewards, guild storage, auction listings) are resolved or cancelled before PII removal is finalized.
- Implement in-app account deletion (`POST /api/v1/account/delete`, re-auth, 7-day `PENDING_DELETION` cancel window, erasure within 15 days) and its client screen.
- Implement the erasure ledger in backup storage and the restore-time replay (`../08_scale_ops/backup_recovery.md`).
- Implement data-subject export for categories A, B, D summary, E summary and F (`../07_security/data_protection.md` § Access and Portability); game-state export is optional courtesy data.

## Acceptance
- Retention jobs execute on schedule and delete or anonymize rows past their retention deadline; a test verifies rows are absent after the deadline elapses.
- Data-subject deletion removes all PII fields from tables listed in `data_protection.md`; preserved audit records do not contain PII beyond what is required by legal hold.
- Deletion of an account does not orphan escrow, pending rewards, guild storage, or auction listings — all owned assets are resolved or cancelled before deletion is finalized; test verifies no orphaned rows remain.
- Data-subject export contains exactly the categories listed in `data_protection.md` and passes schema validation.

## Tests
- `server/internal/durable/privacy/privacy_test.go`: TestRetentionDeadlineSelection, TestDeletionRemovesPiiAllTables, TestAssetsResolvedBeforeErasure, TestAuctionListingCancelledBeforeErasure, TestExportSchemaCategories, TestLegalHoldPreserved, TestErasureLedgerReplayOnRestore.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-056/"

## `IMP-103` — Account Deletion & Data Export API / Account UI
id: IMP-103
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_security/data_protection.md`, `../07_security/auth.md`, `../08_scale_ops/backup_recovery.md`, `../04_architecture/client_experience_contract.md`]
adrs: [`0051-first-party-username-password-login.md`]
depends_on: [IMP-056, IMP-066]
owned_paths: [`server/internal/edge/account/`, `client/Assets/Scripts/UI/Account/`, `client/Assets/Tests/PlayMode/AccountUi/`, `deploy/prod/runbooks/data_subject_requests.md`]
forbidden_paths: [`server/internal/sim/`, `server/migrations/`]
contract_inputs: [authenticated account request, re-auth proof]
contract_outputs: [PENDING_DELETION state, export package, account UI states]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
Implement `POST /api/v1/account/delete` (re-auth, 7-day `PENDING_DELETION`, cancel by login, erasure within 15 days through IMP-056), the data-subject export request endpoint, the client account screen and the operator runbook for data-subject requests.

## Acceptance
- deletion requires re-auth and revokes sessions; login during the window cancels; erasure runs by day 7 and never after day 15,
- export returns exactly the `data_protection.md` categories,
- the runbook covers receipt, acknowledgement and fulfilment windows and is reviewed before M10.

## Tests
- `server/internal/edge/account/account_test.go`: TestDeleteRequiresReauth, TestPendingDeletionCancelOnLogin, TestErasureWithin15Days, TestExportRequest.
- `client/Assets/Tests/PlayMode/AccountUi/AccountUiTests.cs`: delete/cancel/export request states.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-103/"

## `IMP-077` — Operator Admin API (auth, roles, two-person rule)
id: IMP-077
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_security/auth.md`, `../04_architecture/authority.md`, `../07_security/validation.md`, `../06_data/data_model.md`, `../08_scale_ops/observability.md`]
adrs: [`0009-account-session-credentials.md`, `0041-anti-rmt-trade-gates-and-iap-entitlement-integrity.md`, `0051-first-party-username-password-login.md`]
depends_on: [IMP-006, IMP-043, IMP-094]
owned_paths: [`server/internal/edge/admin/`, `server/internal/durable/operator/`]
forbidden_paths: [`server/internal/sim/`, `client/`]
contract_inputs: [operator credentials + TOTP, typed admin requests with reason/ticket]
contract_outputs: [audited moderation/economy operations, account status changes, operator management]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Implement `operators` storage, password + TOTP login, 8 h / 30 min operator sessions, and the private-network admin HTTPS listener.
- Implement roles SUPPORT / MODERATOR / ECONOMY / ADMIN, account status changes (mute, suspend, BANNED with session revocation), economy grant/rollback with the two-person rule, and audit writes.

## Acceptance
- the admin API is not reachable on the public listener,
- every call without a valid operator session and TOTP-verified login is rejected,
- each role can call only its operations; ECONOMY grants above the threshold stay pending until a second operator approves,
- every operation is idempotent by `operation_id` and writes `audit_events` with operator, reason and ticket,
- setting BANNED revokes all player session families of the account.

## Tests
- `server/internal/edge/admin/admin_test.go`: TestPublicListenerRejectsAdmin, TestTotpRequired, TestRolePermissions, TestTwoPersonRule, TestBanRevokesSessions, TestAuditWritten.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-077/"

## `IMP-067` — Unity IL2CPP Player Build (Windows, Android) & Release Packaging
id: IMP-067
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../04_architecture/client.md`, `../04_architecture/client_assets.md`, `../04_architecture/client_localization.md`, `../04_architecture/client_experience_contract.md`, `../04_architecture/physics_geometry_contract.md`, `../07_content/presentation_asset_manifest.md`, `../07_content/world_route_catalog.md`, `../07_content/dungeon_catalog.md`, `../03_systems/pvp.md`, `../03_systems/guild_war.md`, `../05_network/versioning.md`, `../08_scale_ops/deployment.md`, `../00_context/technology_versions.md`, `../04_architecture/client_performance.md`]
adrs: [`0006-unity-go-postgresql-stack.md`, `0010-exact-technology-version-pinning.md`, `0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0050-windows-only-ci-and-auto-merge.md`, `0052-single-launch-world.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-020, IMP-024, IMP-025, IMP-028, IMP-041, IMP-042, IMP-076, IMP-084, IMP-085, IMP-086, IMP-087, IMP-088, IMP-089, IMP-090, IMP-093, IMP-099, IMP-103]
owned_paths: [`client/BuildProfiles/`, `client/Assets/Scenes/Bootstrap/`, `client/Assets/Scripts/App/`, `client/Assets/Tests/PlayMode/AppComposition/`, `scripts/verify_client_build.ps1`]
forbidden_paths: [`server/`]
contract_inputs: [validated Unity project, Addressables/localization/protobuf artifacts, version gates]
contract_outputs: [Windows/Android IL2CPP builds (outside the repository), checksums, symbols, release packages]
consumers_checked: [docs/02_world/maps_zones.md, docs/02_world/dungeons.md, docs/03_systems/pvp.md, docs/03_systems/guild_war.md, docs/04_architecture/client_experience_contract.md, docs/04_architecture/physics_geometry_contract.md, docs/07_content/world_route_catalog.md, docs/07_content/dungeon_catalog.md, docs/07_content/presentation_asset_manifest.md, client/ProjectSettings/ProjectSettings.asset, docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md]

## Change
- Author IL2CPP player builds for Windows and Android from `client/BuildProfiles/` with `game-ci/unity-builder` (ADR-0058): Windows in the `windows-6000.6.1f1-windows-il2cpp` image on the `windows-2022` job, Android in the `ubuntu-6000.6.1f1-android` image on the `ubuntu-24.04` job; build output goes to a directory outside the repository (runner temp), never into the tree.
- Enforce .NET Standard 2.1 API profile and exact assembly dependencies.
- Wire the production bootstrap scene and the `ThinhThan.App` composition root (`client/Assets/Scripts/App/`) so every completed client feature is reachable without test-only setup.
- Deliver release packaging validating bundle integrity, symbols, and executable hashes.

## Acceptance
- Clean player build succeeds with IL2CPP scripting backend without stripping errors,
- default Windows player opens resizable windowed at `1280x720`; native/fullscreen options preserve the canonical camera scale,
- build contains 24 distinct world scenes, five dungeon scenes, finale and three competitive scenes with matching Addressable keys and geometry exports,
- the production bootstrap scene reaches login, character selection, world HUD, and every feature UI supplied by its dependencies,
- Release package includes Addressables catalogs, localization tables, protobuf assemblies and accessible third-party asset credits generated by IMP-076; no placeholder or unapproved-source file ships,
- Build checksums and binary artifacts are recorded in release manifest.
- PERF-007 (desktop): PlayMode tests in category `Performance` on the Linux job (llvmpipe, ADR-0058), cold start to login <= 6 s, login to in-world <= 8 s, same-region transfer <= 3 s, new-region transfer <= 6 s, reconnect resume <= 5 s,
- no build output or cache is committed; `git status` is clean after the build.

## Tests
- `scripts/verify_client_build.ps1`: clean IL2CPP Windows (Windows job) and Android (Linux job) smoke builds and package verification (ADR-0058).
- `client/Assets/Tests/PlayMode/AppComposition/AppCompositionTests.cs`: production bootstrap, reference viewport/camera, 33 playable-scene registrations, and feature-registration coverage.
- `client/Assets/Tests/PlayMode/AppComposition/LoadTimeTests.cs`: TestColdStartDesktop, TestLoginToWorld, TestMapTransferTimes, TestReconnectResumeTime (PERF-007).

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-067/"

# Task Group — Production Presentation Assets

## `IMP-070` — Asset Provenance Register & Validator
id: IMP-070
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md`, `../04_architecture/physics_geometry_contract.md`, `repository_layout.md`]
adrs: [`0014-unity-addressables-asset-delivery.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0050-windows-only-ci-and-auto-merge.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`, `0058-public-repo-github-hosted-linux-and-windows-runners.md`]
depends_on: [IMP-063, IMP-101]
owned_paths: [`client/Assets/Art/Provenance/asset_source_register.json`, `client/Assets/Art/Provenance/register.schema.json`, `client/Assets/Scripts/Core/Assets/Editor/AssetProduction/`, `client/Assets/Scenes/Review/`, `client/Assets/Tests/EditMode/AssetProvenance/`, `client/Assets/Tests/EditMode/CutoutQualityGate/`, `client/Assets/Tests/EditMode/VolumeDepthGate/`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [Addressables key database, canonical source-register fields and source policy]
contract_outputs: [machine-readable source register, deterministic provenance/hash validator]
consumers_checked: [docs/07_content/presentation_asset_manifest.md, docs/04_architecture/client_assets.md, docs/10_implementation/definition_of_done.md]

## Change
- Create the versioned source register/fragment schema and Editor validation that maps each included media file to one register row, checks normalized path uniqueness, Addressable key validity, SHA-256, allowed source kind/license, required metadata and approval state.
- Produce a deterministic credits text from approved `CC-BY-4.0` rows and font notice from `OFL-1.1` rows. Keep source/rights decisions in the register; do not move gameplay authority into Addressables.

## Acceptance
- Implement the Volume & Depth Gate (`presentation_asset_manifest.md` §3.6) in the same Editor validator, with one failing fixture per rule (flat fill, narrow value range, bottom-lit form, no edge separation, background layer contrast inversion).
- Implement the Cutout Quality Gate validator (`../07_content/presentation_asset_manifest.md` §3.2) as an Editor tool that measures every alpha texture and writes a per-file report; every threshold has a failing negative fixture (baked checkerboard, halo band > 2 px, magenta fringe, white/black fringe, undilated transparent RGB, stray speck < 64 px, binary jagged edge, interior hole, wrong 2x size).
- Clean empty register passes foundation tests; a production file without a row fails the release mode of the validator.
- Invalid/missing license URL, source URL, generation record, hash, attribution or approval produces a path-specific error; a third-party input to AI generation cannot be hidden.
- Validator uses pinned Unity/project tooling only; no new runtime/package dependency.
- `client/Assets/Scenes/Review/` hosts the Visual Review scenes (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%); the Linux CI job renders them under xvfb + Mesa llvmpipe (`renderer=llvmpipe`, ADR-0058) and uploads artifact `visual-review`; screenshots are review artifacts referenced by the evidence manifest, never committed or used as evidence.

## Tests
- `client/Assets/Tests/EditMode/CutoutQualityGate/CutoutQualityGateTests.cs`: one passing clean sprite and one failing fixture per §3.2 rule.
- `client/Assets/Tests/EditMode/VolumeDepthGate/VolumeDepthGateTests.cs`: one passing sprite/layer set and one failing fixture per §3.6 rule; asset_class scoping per §3.1a.
- `client/Assets/Tests/EditMode/AssetProvenance/AssetProvenanceTests.cs`: valid AI/CC0/CC-BY/OFL rows, missing row, duplicate path or invalid key, changed file hash, disallowed license, pending/rejected record, missing attribution/font notice and generated-input provenance.

generated_artifacts: []
cleanup_obligations: [Remove test-only asset imports and temporary credits output.]
evidence_location: "docs/10_implementation/evidence/IMP-070/"

## `IMP-071` — Player Character & Class Art
id: IMP-071
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../01_gameplay/classes.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/Actors/Players/`, `client/Assets/Tests/EditMode/PlayerArtCoverage/`, `client/Assets/Art/Provenance/fragments/actors_players.json`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [five class identities, launch monster/boss/beast rosters, canonical size profiles and source policy]
contract_outputs: [production actor sprites/animations, Addressable mappings, approved provenance rows]
consumers_checked: [docs/07_content/monster_catalog.md, docs/07_content/boss_catalog.md, docs/07_content/spirit_beast_catalog.md, docs/04_architecture/physics_geometry_contract.md, docs/10_implementation/task_queue.md]

## Change
- Create original or free-licensed art and produce final 2D sheets/rigs and gameplay-state animations for all five classes.
- Import with canonical cell, PPU, pivot and size profile; integrate stable Addressable keys without changing authoritative collider dimensions. Record each file and source in `client/Assets/Art/Provenance/fragments/actors_players.json`.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every class/player actor ID resolves to an intentional visual (including explicit shared variants); no placeholder/default-tool sprite remains.
- Idle/move/attack/hit/defeat and other states required by the owning runtime/UI contract are present; animation timing does not assert server gameplay results.
- Class skill silhouettes remain legible at `1280x720` and mobile layout; Vietnamese folklore silhouette/identity is reviewed.

## Tests
- `client/Assets/Tests/EditMode/PlayerArtCoverage/PlayerArtCoverageTests.cs`: class-to-key coverage, required clips, import scale/cell/pivot, provenance/hash and no-placeholder checks.

generated_artifacts: []
cleanup_obligations: [Remove unused source imports and superseded placeholders from release groups.]
evidence_location: "docs/10_implementation/evidence/IMP-071/"

## `IMP-104` — Monster, Boss & Spirit Beast Art
id: IMP-104
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../07_content/monster_catalog.md`, `../07_content/boss_catalog.md`, `../07_content/spirit_beast_catalog.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0031-exp-scale-x100-and-corrected-act-budgets.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/Actors/Creatures/`, `client/Assets/Tests/EditMode/CreatureArtCoverage/`, `client/Assets/Art/Provenance/fragments/actors_creatures.json`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [monster/boss/Spirit Beast rosters, size profiles, art direction]
contract_outputs: [final creature sheets/animations, Addressable keys, provenance fragment]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
- Create original or free-licensed art and produce final 2D sheets/rigs and gameplay-state animations for every release monster, boss and Spirit Beast ID.
- Import with canonical cell, PPU, pivot and size profile; record each file and source in `client/Assets/Art/Provenance/fragments/actors_creatures.json`.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every monster, boss and Spirit Beast ID resolves to an intentional visual (including explicit shared variants); no placeholder sprite remains.
- Idle/move/attack/hit/defeat states required by the runtime/UI contract are present; animation timing does not assert server results.
- Boss/elite telegraphs remain legible at `1280x720` and mobile layout; Vietnamese folklore silhouette/identity is reviewed.

## Tests
- `client/Assets/Tests/EditMode/CreatureArtCoverage/CreatureArtCoverageTests.cs`: roster-to-key coverage, required clips, import scale/cell/pivot, shared-variant mapping, provenance/hash and no-placeholder checks.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-104/"

## `IMP-072` — Normal-World Environment Art & Scenes
id: IMP-072
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../07_content/world_route_catalog.md`, `../02_world/maps_zones.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-062, IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/World/`, `client/Assets/Scenes/World/`, `client/Assets/Tests/EditMode/WorldArtCoverage/`, `client/Assets/Art/Provenance/fragments/world.json`]
forbidden_paths: [`server/internal/sim/spatial/maps/`, `proto/`]
contract_inputs: [locked scene roster/topology, exported geometry, size/viewport contract, source policy]
contract_outputs: [authored production scenes, tile/prop/parallax sets, approved provenance rows]
consumers_checked: [docs/07_content/world_route_catalog.md, docs/07_content/dungeon_catalog.md, docs/03_systems/pvp.md, docs/03_systems/guild_war.md, docs/04_architecture/physics_geometry_contract.md]

## Change
- Create or source licensed tiles, props, architecture and parallax layers; compose all 24 normal-world scenes according to their distinct topology profiles.
- Keep visual layers separate from authoritative collision/geometry export; no giant one-sprite map. Record every shipped image/source in `client/Assets/Art/Provenance/fragments/world.json`.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every scene has a unique stable Addressable key and visual identity; different map shape/size/branches/vertical tiers match catalogs and exported geometry.
- Foreground/parallax and telegraph contrast remain readable; atlas/bundle size and region unload budgets pass.
- Visual editing does not silently alter canonical collision, spawn anchors or server geometry. Any needed geometry change returns to its owning spec/task.

## Tests
- `client/Assets/Tests/EditMode/WorldArtCoverage/WorldArtCoverageTests.cs`: 24-scene roster, Addressable keys, dimensions/topology/geometry consistency, no single-bitmap substitutes, bundle budget and provenance checks.

generated_artifacts: []
cleanup_obligations: [Remove scene placeholder layers and unused imported art.]
evidence_location: "docs/10_implementation/evidence/IMP-072/"

## `IMP-105` — Dungeon, Finale & Competitive Environment Art
id: IMP-105
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../07_content/dungeon_catalog.md`, `../03_systems/pvp.md`, `../03_systems/guild_war.md`, `../04_architecture/physics_geometry_contract.md`]
adrs: [`0036-seasons-as-launch-infrastructure.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-062, IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/Instances/`, `client/Assets/Scenes/Dungeons/`, `client/Assets/Scenes/Finale/`, `client/Assets/Scenes/Competitive/`, `client/Assets/Tests/EditMode/InstanceArtCoverage/`, `client/Assets/Art/Provenance/fragments/instances.json`]
forbidden_paths: [`server/internal/sim/spatial/maps/`, `proto/`]
contract_inputs: [dungeon/finale/competitive space profiles, exported geometry]
contract_outputs: [final instance scenes, Addressable keys, provenance fragment]
consumers_checked: [docs/10_implementation/milestones.md, docs/10_implementation/dependency_graph.md, docs/10_implementation/spec_traceability.md]

## Change
- Compose the five dungeon scenes, the finale and the three competitive scenes with licensed/original tiles, props and parallax layers matching their topology profiles.
- Keep visual layers separate from geometry export; record every file and source in `client/Assets/Art/Provenance/fragments/instances.json`.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every instance scene has a unique stable Addressable key; shape/size/branches/vertical tiers match catalogs and exported geometry; competitive scenes keep mirror parity.
- Telegraph contrast remains readable; atlas/bundle size budgets pass; visual editing never alters collision, anchors or server geometry.

## Tests
- `client/Assets/Tests/EditMode/InstanceArtCoverage/InstanceArtCoverageTests.cs`: nine-scene roster, Addressable keys, geometry consistency, mirror parity, bundle budget and provenance checks.

generated_artifacts: []
cleanup_obligations: [Ensure zero orphaned files or test fixtures.]
evidence_location: "docs/10_implementation/evidence/IMP-105/"

## `IMP-073` — UI, Item, Equipment & Skill VFX Art
id: IMP-073
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../07_content/class_skill_catalog.md`, `../07_content/item_catalog.md`, `../07_content/equipment_catalog.md`, `../04_architecture/client_localization.md`]
adrs: [`0016-twelve-skill-pool-upgradeable-basics.md`, `0037-reflect-lifesteal-absorb-heal-reduction-stats.md`, `0046-reference-viewport-entity-scale-and-map-geometry.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/UI/`, `client/Assets/Art/Items/`, `client/Assets/Art/VFX/`, `client/Assets/Tests/EditMode/InterfaceArtCoverage/`, `client/Assets/Art/Provenance/fragments/interface.json`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [release UI states, skill/item/equipment IDs, presentation size budgets, source policy]
contract_outputs: [production UI/font/icon art and skill telegraphs/VFX, approved provenance rows]
consumers_checked: [docs/07_content/class_skill_catalog.md, docs/07_content/item_catalog.md, docs/07_content/equipment_catalog.md, docs/04_architecture/client_localization.md]

## Change
- Agent creates or sources free-licensed font/UI art, item/equipment icons, skill VFX and telegraphs, using stable IDs or explicit shared-art mappings.
- Deliver both required locales' glyph coverage and PC/mobile presentation variants without encoding gameplay outcomes in visual data.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every release UI control/state and relevant item/equipment/skill ID resolves; no tofu, unlabelled placeholder icon or missing telegraph.
- VFX shape/timing visually communicates the canonical skill geometry but cannot change hitboxes, duration or target selection.
- Small-screen contrast/readability and color-independent dangerous telegraphs pass visual review.

## Tests
- `client/Assets/Tests/EditMode/InterfaceArtCoverage/InterfaceArtCoverageTests.cs`: catalog/UI-to-key coverage, font glyph coverage, VFX mapping, mobile readability fixtures, import/bundle budgets and provenance checks.

generated_artifacts: []
cleanup_obligations: [Remove temporary icons, fonts and unused VFX materials.]
evidence_location: "docs/10_implementation/evidence/IMP-073/"

## `IMP-074` — Cosmetic Presentation Art
id: IMP-074
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../07_content/cosmetic_catalog.md`, `../03_systems/cosmetics.md`]
adrs: [`0053-durable-contract-reconciliation.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-063, IMP-070]
owned_paths: [`client/Assets/Art/Cosmetics/`, `client/Assets/Tests/EditMode/CosmeticArtCoverage/`, `client/Assets/Art/Provenance/fragments/cosmetics.json`, `client/Assets/Art/Provenance/cultural_review.md`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [launch cosmetic entitlements and equip slots, cultural-review rule, source policy]
contract_outputs: [production cosmetic visuals or explicit text/shared-art mappings, approved provenance rows]
consumers_checked: [docs/07_content/cosmetic_catalog.md, docs/03_systems/cosmetics.md, docs/00_context/constraints.md]

## Change
- Agent creates or sources final skin/frame/title/shrine and other cosmetic presentation for every release cosmetic ID. Text-only titles and intentional shared art use explicit mappings rather than unnecessary duplicate files.
- Record cultural/reference review by cosmetic ID in `client/Assets/Art/Provenance/cultural_review.md` for entries named by `cosmetic_catalog.md`; enter every shipped media file into the source fragment.

## Acceptance
- Every shipped texture is finished at its exact 2x size, passes the Cutout Quality Gate and the Volume & Depth Gate with zero violations, follows the art direction in `presentation_asset_manifest.md` §3.5, and has Visual Review screenshots (1280x720, 1920x1080, 2400x1080; day/night; 100%/200%) approved by a different agent; screenshots are captured in the `client/Assets/Scenes/Review/` scenes (IMP-070) on the Linux CI job (llvmpipe) and attached as review artifacts, never committed or used as evidence (`presentation_asset_manifest.md` §3.1–3.3).
- Every cosmetic ID renders the correct entitlement presentation and never changes gameplay collider/stats/equipment identity.
- Culturally sensitive concepts have review evidence before production acceptance; no recognizable borrowed trademark, religious insignia or unlicensed reference.
- No placeholder cosmetic ships; free-license/AI-tool rights and attribution are complete.

## Tests
- `client/Assets/Tests/EditMode/CosmeticArtCoverage/CosmeticArtCoverageTests.cs`: full ID-to-key/text/shared mapping, equip-slot preview, non-power invariant, cultural-review evidence and provenance coverage.

generated_artifacts: []
cleanup_obligations: [Remove rejected designs and superseded previews from release groups.]
evidence_location: "docs/10_implementation/evidence/IMP-074/"

## `IMP-075` — SFX & Folklore BGM Production
id: IMP-075
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md`, `../00_context/constraints.md`]
adrs: [`0014-unity-addressables-asset-delivery.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`]
depends_on: [IMP-063, IMP-070]
owned_paths: [`client/Assets/Audio/`, `client/Assets/Tests/EditMode/AudioAssetCoverage/`, `client/Assets/Art/Provenance/fragments/audio.json`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [release map/action/UI cue inventory, audio group budgets, source policy]
contract_outputs: [production SFX/BGM and cue mappings, approved provenance rows]
consumers_checked: [docs/07_content/presentation_asset_manifest.md, docs/04_architecture/client_assets.md, docs/00_context/constraints.md]

## Change
- Agent creates original audio or sources free-licensed recordings/music, edits clean loops and action/UI SFX, and maps every required release cue to an Addressable key.
- Record composer/performer/source/derivative details for recordings and generated audio; traditional instrument inspiration does not authorize copying a modern performance or arrangement.

## Acceptance
- All release map BGM and action/UI feedback cues resolve; BGM loops cleanly and streams under group budget, SFX do not clip or mask critical combat feedback.
- No unlicensed recording, placeholder beep or generic borrowed soundtrack ships; attribution is complete where required.

## Tests
- `client/Assets/Tests/EditMode/AudioAssetCoverage/AudioAssetCoverageTests.cs`: cue/key coverage, loop and clip import settings, bundle/streaming budgets, source hash/license and no-placeholder checks.

generated_artifacts: []
cleanup_obligations: [Remove raw trial recordings and unused audio exports from release groups.]
evidence_location: "docs/10_implementation/evidence/IMP-075/"

## `IMP-076` — Production Asset Coverage, Rights & Release Audit
id: IMP-076
status: NOT_STARTED
claimed_by: ""
branch: ""
claimed_at: ""
blocked_by: ""

specs: [`../07_content/presentation_asset_manifest.md`, `../04_architecture/client_assets.md`, `definition_of_done.md`]
adrs: [`0014-unity-addressables-asset-delivery.md`, `0045-ci-evidence-without-self-referential-sha.md`, `0055-2x-texture-authoring-and-cutout-quality-gate.md`, `0056-volumetric-art-direction-and-2d-lighting.md`, `0057-bootstrap-trusted-ci-evidence-identity-and-merge-mechanics.md`]
depends_on: [IMP-004, IMP-064, IMP-071, IMP-072, IMP-073, IMP-074, IMP-075, IMP-104, IMP-105]
owned_paths: [`client/Assets/Scripts/Core/Assets/Editor/AssetProduction/ReleaseAssetAudit.cs`, `client/Assets/Tests/EditMode/ReleaseAssetAudit/`, `client/Assets/Notices/THIRD_PARTY_ASSETS.txt`, `client/Assets/Art/Provenance/asset_source_register.json`, `client/Assets/Art/Provenance/asset_rights_review.md`]
forbidden_paths: [`server/`, `proto/`]
contract_inputs: [validated release catalogs, completed asset groups/scenes, source register]
contract_outputs: [complete release coverage report, reviewed rights report, packaged credits notice]
consumers_checked: [docs/07_content/presentation_asset_manifest.md, docs/04_architecture/client_assets.md, docs/04_architecture/client_localization.md, docs/10_implementation/milestones.md, docs/10_implementation/definition_of_done.md]

## Change
- Walk all release catalog IDs, feature UI states and Addressable dependencies; reject missing/placeholder/unapproved assets, bad hashes, over-budget groups and unsupported glyphs.
- Merge the seven provenance fragments in `client/Assets/Art/Provenance/fragments/` deterministically into the release register; review original source pages/tool terms and cultural-review evidence, generate deterministic attribution notice, then make the result available to IMP-067 packaging.

## Acceptance
- Re-run the Cutout Quality Gate and the Volume & Depth Gate on the full release set and verify Visual Review evidence exists for every entity/UI surface; any violation or missing review blocks release.
- A clean checkout reproduces the complete key/file/provenance report; every source is `APPROVED` with commercial-modification/distribution rights and correct packaged attribution.
- All 33 playable scenes, actors, cosmetics, skills, UI, items and audio required by release scope resolve; no shortcut suppresses missing coverage.
- `THIRD_PARTY_ASSETS.txt` has stable Core-UI Addressable key `asset.ui.credits.third_party_assets`, is included in the player package and accessible through the credits UI wired by IMP-067. Any failed rights or asset gate blocks IMP-067 and M10.

## Tests
- `client/Assets/Tests/EditMode/ReleaseAssetAudit/ReleaseAssetAuditTests.cs`: full release walk, negative missing-key/placeholder/hash/rights/credits/budget cases and deterministic notice output.

generated_artifacts: [`client/Assets/Notices/THIRD_PARTY_ASSETS.txt`]
cleanup_obligations: [Commit the deterministic notice, remove temporary audit outputs, and confirm no rejected files remain in build groups.]
evidence_location: "docs/10_implementation/evidence/IMP-076/"

# Design Decisions

## Souls Do Not Source the Four New Combat Stats at Launch
The four new combat stats (LIFESTEAL, REFLECT, ABSORB, HEAL_REDUCTION) are sourced from equipment secondary rolls, Spirit Meridian, and Formations only. Verified realistic cap reachability from those three sources is 49–68% of cap, which is healthy. Adding Soul sources would require redoing the cap-reachability arithmetic for all four stats and would widen the balance surface for negligible gain. This is a closed decision: Souls do not source these stats at launch, and no task is open to add them. Any future proposal to add Soul sources requires a new ADR and full cap-reachability re-verification.

# Queue Rules
- Bootstrap: a task runs before `IMP-068 = DONE` only if `IMP-068` is not in its transitive `depends_on`; IMP-068 verifies, never resolves, contract blockers.
- The coordinator claims ready tasks lowest wave first, then lowest ID (`agent_execution_protocol.md`).
- One task may span multiple files/components but must have one testable behavior boundary.
- Do not mark DONE from documentation alone or from a green local `go test`.
- A spec gap or contradiction found during implementation becomes a `BLK-xxx` entry; the task is set `BLOCKED` and the `spec-owner` resolves it in a spec-change PR.
- A bug fix adds a regression to the owning task/test suite; it does not create an undocumented exception.
- `BLOCKED` tasks return to `NOT_STARTED` only after the named canonical blocker is resolved and regression coverage exists.
- `DONE` requires the exact task tests on disk plus CI evidence per ADR-0057 (`source_tree_hash`, API-verified `ci_run_id` + `run_attempt`).
