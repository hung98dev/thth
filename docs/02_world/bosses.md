# Bosses
status: LOCKED

## Scope
Defines boss identity, phases, scaling, reset, participation, reward eligibility, loot, respawn, reconnect, and recovery.

## Identity / Modes
`boss_id`, `boss_instance_id`, `encounter_instance_id`. Modes PUBLIC and INSTANCED.

Every launch boss definition also declares one stable `space_id` and one `size_profile` in `../07_content/boss_catalog.md`. Collider dimensions come from `../04_architecture/physics_geometry_contract.md`; runtime never infers them from boss mode or art.

PUBLIC standalone spawns also receive a world-scoped stable:
```text
public_boss_spawn_generation_id
```
All channel copies representing the same logical spawn generation share this ID for reward settlement.

## States / Phases
```text
INACTIVE -> SPAWNING -> READY -> ACTIVE -> DEFEATED -> COOLDOWN
ACTIVE -> RESETTING -> READY
```
Recommended `2-3` phases, `2-4` core attacks/phase, one signature mechanic. Mechanics before HP inflation.

## Telegraph
An attack capable of roughly `>35%` appropriately geared max HP requires readable startup/area/pattern/counterplay.

## Dynamic PUBLIC Scaling
Participant count is sampled at encounter activation, each phase transition, and every `15s` while ACTIVE.
```text
n_effective = clamp(n, 1, 18)
hp_multiplier = 1 + 0.35 * (n_effective - 1)
damage_multiplier = 1 + min(0.0125 * (n_effective - 1), 0.30)
```
When max HP changes from resampling, preserve current HP percentage; scaling never heals by percentage manipulation. Reward quantity does not scale linearly with participant count.

INSTANCED dungeon bosses use the owning dungeon's PARTY encounter scaling unless the boss definition explicitly opts out.

## Targeting / Control
Default target score = recent damage threat + proximity + mechanic priority; threat decays after `8s`. Boss hard CC converts to STAGGER contribution; SLOW `50%` normal magnitude; BURN/POISON normal unless override.

Stagger meter (server-owned, per boss instance):
```text
contribution(hard CC application) = authored_duration_ms * weight
  weight: STUN 1.0, FREEZE 1.0, AIRBORNE 1.0, ROOT 0.5
  forced displacement (KNOCKBACK/PULL) = flat 400
  SLOW / non-control statuses       = 0
initial_threshold  = 6000 * stagger_scale        (6 s of full-weight hard CC)
stagger_scale      = PARTY/PUBLIC hp_multiplier of the current scaling sample, clamped to 1.0..4.0
decay              = -500 per second after 3.0 s without a contribution (never below 0)
meter >= threshold -> STAGGERED for 2 s: boss cannot start actions, active channel is cancelled, meter resets to 0
after each stagger  threshold = initial_threshold * (1 + 0.25 * stagger_count), capped at 2.0 * initial_threshold
UNSTAGGERABLE      -> contributions are discarded while it is active
reset (Reset below) -> meter 0, stagger_count 0
```
Hard CC applied to a boss never produces the status itself; it only feeds the meter. `stagger progress = meter / threshold` is the value used by content such as `boss.ho_tinh` `UY_SON`.

## Reset
Reset after no valid participant in combat area for `10s`, dungeon wipe, or explicit failure. Clear hazards/summons and restore initial state; no rewards.

## Participation
Track valid damage, effective support, mechanics. Eligibility normally requires `>=30s` active time and `>=3%` boss-max-HP-equivalent contribution or configured major mechanic threshold. Presence alone is insufficient.

## Loot / EXP
PERSONAL loot. Configured EXP/currency/material/equipment/quest/cosmetic rewards. Boss kill EXP is granted only by PUBLIC bosses (monster level adjustment unless fixed); INSTANCED bosses grant no kill EXP (`../07_content/boss_catalog.md`). Earned item bundles that cannot fit inventory use `../03_systems/reward_claims.md`.

## Victory Ceremony & Gold Chest Claim
When a major boss is defeated, the server triggers an in-channel victory celebration under ADR-0024:
1. The boss collapses and a massive **Gilded Dragon Chest** (`chest.world_boss.<boss_id>`) appears at the arena center amidst festive firecrackers and celebratory effects.
2. The chest remains interactable for 3 minutes before despawning.
3. Every eligible participant (meeting the contribution threshold above) interacts with the chest to claim their personal reward settlement.
4. **Personal Loot Invariant**: Interacting with the chest executes that specific player's personal drop table roll. One player claiming never depletes another player's loot. Unclaimed rewards upon chest despawn safely transfer to Reward Claims (`../03_systems/reward_claims.md`).

## Boss Aftermath — Di Tich (He Qua The Gioi)
Defeating a major boss leaves a 60-minute world consequence in that map/channel:

- **Spawn**: Upon `DEFEATED -> COOLDOWN`, the server spawns `relic.boss.<boss_id>` at the arena center in that defeated `map_instance_id` for 60 minutes. It is visible only to characters in that map instance; each channel copy may independently have its own relic.
- **Channel-wide Buff**: Every character in the same `map_instance_id` (same channel) receives `buff.di_tich.<boss_id>` while the relic is active:
  ```
  +5% monster EXP in this map
  CHEST_SPOTTED range 12m -> 13m (integer)
  if map has_water = true: fishing.catch.default rare weight 100 -> 110 bp (world_rules.md)
  ```
  Buff is passive, non-stacking, and expires when relic despawns or character leaves the map.
- **Persistence**: Relic state is persisted in authoritative server storage and survives server restart. On restart the server restores any active relic with its remaining duration (clamped to at least 1 second). Separately, a permanent **Di Tich marker** (`marker.di_tich.<boss_id>`) is written to the owning region's safe-anchor world state on every `DEFEATED` transition and updated on each subsequent defeat. The marker records: last-defeated UTC timestamp, last-defeated participant count, and whether a relic is currently active in any channel of that region. Clients display the marker at the safe anchor regardless of which channel the character occupies. No reward is granted by the relic itself beyond the buff; relic and marker together serve as persistent social proof of what players accomplished.
- **Authority**: Server owns relic spawn/despawn and buff application; client cannot forge relic.

## Seasonal Di Tich Relics
While the matching `season_region_index` is active, seasonal relics reuse the same 60-minute witness, channel-wide `buff.di_tich.*` effects, persistence, and authority as launch Di Tích. Atlas pages point at these relic IDs. They are not extra power. Duration 60 minutes; witness range 60m. Durable key `(map_id, channel_id, relic_id)` in `../06_data/data_model.md` `world_consequence_relics` (`source_id` = the dungeon or monster that spawned it). Seasonal relics apply `buff.di_tich.season`, whose payload equals the launch `buff.di_tich.*` payload; they write no region marker. Spawn trigger: dungeon completion spawns the relic on the listed map in the channel the party entered the dungeon from; a monster kill spawns it on the kill's channel; an already-active relic with the same key is not refreshed.

| relic_id | season_region_index | spawn after | map |
|---|---:|---|---|
| `relic.season.0.quy_xuan` | 0 | `dungeon.dinh_lang_bo_hoang` | `map.lang_da.go_ma` |
| `relic.season.0.hon_dau` | 0 | kill `monster.lang_da.hon_gao` | `map.lang_da.go_ma` |
| `relic.season.1.moc_mieu` | 1 | `dungeon.mieu_ba_trong_rung` | `map.rung_u_minh.mieu_bo_hoang` |
| `relic.season.1.moc_tinh` | 1 | kill `monster.rung_u_minh.moc_tinh` | `map.rung_u_minh.rung_sau` |
| `relic.season.2.xom_chim` | 2 | `dungeon.xom_chim` | `map.ben_nuoc_den.ben_do_cu` |
| `relic.season.2.ma_da_gia` | 2 | kill `monster.ben_nuoc_den.ma_da_gia` | `map.ben_nuoc_den.bai_lau` |
| `relic.season.3.hang` | 3 | `dungeon.hang_ma_tranh` | `map.deo_may.khe_da` |
| `relic.season.3.ho_ve` | 3 | kill `monster.deo_may.ho_tinh_ve` | `map.deo_may.khe_da` |
| `relic.season.4.den` | 4 | `dungeon.den_tran` | `map.thanh_co.den_tran` |
| `relic.season.4.thach` | 4 | kill `monster.thanh_co.thach_ve` | `map.thanh_co.duong_da` |
| `relic.season.5.cong` | 5 | `boss.than_trung` | `map.nui_thieng.cong_co` |
| `relic.season.5.linh_ve` | 5 | kill `monster.nui_thieng.linh_ve` | `map.nui_thieng.rung_may` |

## Reward Slots
Canonical boss reward slots:
```text
REPEAT      -> every eligible encounter/generation reward settlement
FIRST_CLEAR -> once per character for the configured boss definition
DAILY_FIRST -> once per character per UTC day when explicitly configured
```

`FIRST_CLEAR` is lifetime-style content progression and never resets automatically at UTC midnight. `DAILY_FIRST` is a separate optional accelerator. Neither is an entry lockout.

Configured guaranteed first-clear collectibles such as a Boss Soul use `FIRST_CLEAR`.

## PUBLIC Cross-Channel Settlement
For standalone PUBLIC bosses, one character may receive the same generation-scoped repeat reward at most once per:
```text
character_id + public_boss_spawn_generation_id + reward_slot
```
Killing another channel copy from the same generation grants no duplicate reward. Contribution is still tracked independently per encounter instance.

A lifetime `FIRST_CLEAR` key is additionally scoped by:
```text
character_id + boss_id + FIRST_CLEAR
```
so changing channel/generation can never duplicate it.

## Generation ID Rotation Atomicity
When a new PUBLIC boss spawn begins, the server atomically assigns a new `public_boss_spawn_generation_id` as part of the spawn transaction. The prior generation ID is retired in the same transaction: all contribution records keyed to the previous `public_boss_spawn_generation_id` are marked expired server-side at that moment. No contribution record earned against generation N is valid for a chest claim against generation N+1.

The **Gilded Chest** entity produced by a spawn carries the `public_boss_spawn_generation_id` of the spawn that created it. The chest interaction payload echoes this ID to the server. At commit time the server re-validates that the interacting character holds a non-expired contribution record whose `public_boss_spawn_generation_id` matches the chest's ID. Chest eligibility is therefore scoped to exactly one generation; a character cannot use a contribution earned in a prior or concurrent generation to claim from a different generation's chest.

Channel copies representing the same logical spawn generation already share the same `public_boss_spawn_generation_id` under the PUBLIC Cross-Channel Settlement rules above. A new spawn on any channel in the same region causes all channels to retire the old generation ID together; this is coordinated by the Ephemeral Global subsystem, which owns `public_boss_spawn_generation_id` (`../04_architecture/service_boundaries.md`). Per-character eligibility is persisted in `boss_chest_eligibility` (`../06_data/data_model.md`).

## Respawn
Standalone PUBLIC bosses use random delay `30m..45m` after logical generation completion/reset cleanup unless scheduled-event owned. Server coordinates generation identity across channels. Spirit Surge bosses follow world-event lifecycle.

PUBLIC bosses are optional open-world encounters by default. A MAIN story quest must not require waiting for a standalone PUBLIC generation unless the content provides an always-available instanced story equivalent.

## Reconnect / Restart
Participation retained `120s` after disconnect. Restart does not reconstruct active combat; incomplete encounter fails, committed rewards remain, next spawn derives from authoritative server time.

## Reward Idempotency
INSTANCED repeat slot:
```text
character_id + encounter_instance_id + reward_slot
```
PUBLIC repeat slot additionally enforces the generation-level key above.

FIRST_CLEAR/DAILY_FIRST use their own canonical scopes and are independent of repeat reward settlement.

## Invariants
```text
PUBLIC scaling n <= 18
PUBLIC participant resample <= 15s cadence
HP percentage preserved on scaling
one PUBLIC logical generation repeat reward per character/reward slot across channels
FIRST_CLEAR != DAILY_FIRST
MAIN progression does not wait for standalone PUBLIC respawn by default
last hit != ownership
hard CC -> stagger by default
generation ID rotation is atomic with spawn; prior-generation contribution records are expired at rotation
Gilded Chest carries its generation's public_boss_spawn_generation_id; server re-validates at commit time
chest eligibility is scoped to one generation only; contribution from generation N does not authorise a claim against generation N+1
```
