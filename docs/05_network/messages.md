# Messages
status: LOCKED

## Scope
Defines canonical launch message families, direction, delivery semantics, and validation boundaries. Concrete protobuf field layouts are formally established by the canonical baseline in `proto/thinhthan/v1/*.proto` (`IMP-061`); subsequent wire schema evolution is strictly additive under `versioning.md` and field numbers may never be changed or reassigned once committed.

## Message ID Rules
`message_id` is an immutable numeric enum within a protocol major version.

Ranges:
```text
1..99      connection/session
100..199   input/movement
200..299   combat
300..399   world/entity replication
400..499   inventory/equipment/crafting
500..599   quest/reward/progression
600..699   social/party/guild
700..799   trade/auction
800..899   PvP/Guild War
900..999   diagnostics/control
```

Do not reuse retired IDs.

## Connection / Session
Launch messages:
```text
ID   Name
1    C2S_HELLO
2    S2C_HELLO_OK
3    S2C_ERROR
4    C2S_HEARTBEAT
5    S2C_HEARTBEAT
6    C2S_CHARACTER_ATTACH
7    S2C_CHARACTER_ATTACH_OK
8    S2C_SESSION_REPLACED
9    S2C_SERVER_DRAINING
10   C2S_CHARACTER_DETACH
11   S2C_CHARACTER_DETACH_OK
```

These validate authentication/session/protocol state before gameplay dispatch.

`C2S_CHARACTER_DETACH` (10) returns the attached character to `OFFLINE` on this session. Success is `S2C_CHARACTER_DETACH_OK` (11). Attaching another character without this detach is `CHARACTER_ALREADY_ACTIVE`.

## Input / Movement
```text
ID   Name
100  C2S_INPUT_STATE
101  C2S_JUMP
102  C2S_DROP_THROUGH
103  C2S_INTERACT
104  C2S_PORTAL_USE
105  S2C_TRANSFER_PREPARE
106  C2S_PRESENTATION_READY
107  S2C_MOVEMENT_CORRECTION
108  C2S_MOVEMENT_EDGE
109  C2S_CHANNEL_SWITCH
110  S2C_CHANNEL_SWITCH_RESULT
111  C2S_DUNGEON_ENTER_REQUEST
112  S2C_DUNGEON_ENTRY_STATE
113  C2S_DUNGEON_ENTRY_RESPOND
114  C2S_DUNGEON_LEAVE
115  S2C_DUNGEON_LEAVE_RESULT
116  S2C_INTERACT_RESULT
117  C2S_DUNGEON_ENTRY_CANCEL
```

`C2S_INTERACT` (103) is discrete and cannot be silently merged. Fields:
```text
interact_kind : TALK | PICKUP | CHEST | CAST | HOOK | KINDLE | COOK | BONFIRE_REST | QUEST_OBJECT | NPC_SERVICE
target_id     : string — target entity or interactive object ID (object ID patterns: ../02_world/maps_zones.md, ../02_world/world_rules.md)
operation_id  : UUID
recipe_id     : string, optional — required when interact_kind == COOK; must match a hearth recipe from crafting_catalog.md
public_boss_spawn_generation_id : UUID, optional — required only for CHEST on a public-boss Gilded Chest; ignored otherwise
service_id    : string, optional — required when interact_kind == NPC_SERVICE: set_checkpoint | travel
                (other NPC services use their dedicated messages: respec 513, crafting 404, enhancement 406,
                storage.account 418, auction 730..744; each carries npc_id and is range-checked like NPC_SERVICE)
service_param : string, optional — travel: destination safe-anchor map_id (../07_content/npc_shop_catalog.md § Travel);
                TALK: dialogue_option_id of the NPC's current dialogue node; empty otherwise
```
Fishing CAST/HOOK use `interact_kind`. `HOOK_WINDOW` accepts only `HOOK`. Kindling, hearth cook, and bonfire rest use `KINDLE` / `COOK` / `BONFIRE_REST`. When `interact_kind == COOK`, `recipe_id` determines the food crafted; `target_id` is the village hearth. When `interact_kind == BONFIRE_REST`, `target_id` is the active village bonfire; player must be within 6.0m and stationary to initiate rest.

Every `C2S_INTERACT` gets exactly one `S2C_INTERACT_RESULT` (116): `operation_id`, `interact_kind`, `target_id`, `status : SUCCESS | ERROR`, `error_code` (`errors.md`; e.g. `OUT_OF_RANGE`, `IN_COMBAT`, `INSUFFICIENT_CURRENCY`, `INVENTORY_FULL`, `CHEST_ELIGIBILITY_INVALID`, `NOT_DISCOVERED`), `granted : list of {item_id, quantity}` (pickup/chest/cook/fishing), `currency_delta : list of {currency_id, amount}` (travel fee, rest). A successful `travel` is followed by the normal transfer flow (`S2C_TRANSFER_PREPARE`); retry with the same `operation_id` returns the committed result and never charges twice.

### Dungeon Entry and Exit (111..117)
Rules: `../02_world/dungeons.md` (Entry / Membership, Re-entry / Cleanup).
- **C2S_DUNGEON_ENTER_REQUEST (111)**: `operation_id : UUID`, `dungeon_id : string`, `run_tag : NORMAL | ENDGAME_L60` (`../07_content/dungeon_catalog.md`), `entrance_id : string` (portal/entrance object the requester stands at). Sender is a partyless character or the party leader. If the sender is a snapshot member of a non-terminal instance of this `dungeon_id`, the request re-enters that instance and no prompt is created. A SOLO dungeon or a partyless sender creates the instance immediately. Otherwise the server creates one pending entry (`entry_id`, lifetime `30s`) and prompts every online party member on the same map instance as the sender; the sender counts as accepted. Rejections: `LEVEL_TOO_LOW`, `IN_COMBAT`, `STATE_CONFLICT` (pending entry exists, dead, transferring), `STORY_CHOICE_REQUIRED` (sender has an unresolved story branch for this dungeon; see 509), `PERMISSION_DENIED` (party member not leader), `OUT_OF_RANGE`.
- **S2C_DUNGEON_ENTRY_STATE (112)**: `entry_id : UUID`, `operation_id : UUID`, `dungeon_id`, `run_tag`, `requester_character_id`, `members : list of {character_id, display_name, response : PENDING | ACCEPTED | DECLINED | INELIGIBLE, error_code}`, `expires_in_ms : uint32`, `outcome : PENDING | CREATED | CANCELLED`, `error_code`. Sent to the requester and every prompted member on each change. The instance is created with the ACCEPTED members when every prompted member has responded or the entry expires (non-responders count as DECLINED); `CANCELLED` when the requester cancels, leaves the party, or becomes ineligible. `CREATED` is followed by `S2C_TRANSFER_PREPARE` for every accepted member. Membership snapshot = accepted members (`dungeons.md`).
- **C2S_DUNGEON_ENTRY_RESPOND (113)**: `operation_id : UUID`, `entry_id : UUID`, `decision : ACCEPT | DECLINE`. An ACCEPT is revalidated like the request (level, alive, not in combat, story branch); failure marks the member `INELIGIBLE` with its `error_code`.
- **C2S_DUNGEON_ENTRY_CANCEL (117)**: `operation_id : UUID`, `entry_id : UUID`. Requester only.
- **C2S_DUNGEON_LEAVE (114)**: `operation_id : UUID`, `mode : EXIT | ABANDON`. `EXIT` = voluntary exit; the member may re-enter with 111 before terminal lock. `ABANDON` removes completion eligibility and re-entry for that instance. Both transfer the character to the instance's return spawn. Rejected while `in_combat` with `IN_COMBAT` (EXIT only; ABANDON is always allowed).
- **S2C_DUNGEON_LEAVE_RESULT (115)**: `operation_id`, `status : SUCCESS | ERROR`, `mode`, `error_code`.

`C2S_CHANNEL_SWITCH` (109) payload: `target_channel_index`, `operation_id`. Cooldown 10s after a successful switch (`../02_world/world_rules.md`). Target at 18 players, or all 30 channels at 18, rejects with `MAP_CAPACITY_FULL`. Result is `S2C_CHANNEL_SWITCH_RESULT` (110).

`C2S_INPUT_STATE` delivery class: `REPLACEABLE_STATE` — replaceable/coalescible continuous held-state. It carries:
```text
input_flags     : bitmask of currently held directions/actions
client_seq      : monotonic client gameplay intent sequence
client_mono_ms  : uint64 — client monotonic milliseconds, ADVISORY ONLY
                  (see advisory clock rules below)
```

`C2S_MOVEMENT_EDGE` delivery class: `DISCRETE_INTENT` — **never coalesced, never merged**. It is the sole wire signal for movement-onset events required by mechanics such as Just Guard. Fields:
```text
edge_type      : enum PRESS | RELEASE | FLIP
direction      : enum LEFT | RIGHT
client_seq     : monotonic, shares the existing client gameplay intent sequence
client_mono_ms : uint64 — client monotonic milliseconds, ADVISORY ONLY
                 (see advisory clock rules below)
```

Every `C2S_MOVEMENT_EDGE` is validated individually. Sending one does not replace or suppress any pending `C2S_INPUT_STATE`.

### Advisory Clock Rules — `client_mono_ms`
`client_mono_ms` is present on both `C2S_INPUT_STATE` and `C2S_MOVEMENT_EDGE`. Its purpose is bounded latency compensation for timing-sensitive mechanics (e.g., Just Guard, whose detection window is defined in `../01_gameplay/combat.md`).

Server rules:
- The server uses `client_mono_ms` as an **advisory** clock only; it is never authoritative for game state or server time.
- Latency compensation (≤ 80 ms), `lag_ms`, `RTT_estimate` and the `STALE_INPUT` bound (`lag_ms > RTT_estimate + 80`) are canonical in `../01_gameplay/combat.md` § Just Guard; this file does not restate the formula.
- The client never sends a `just_guard` flag. Just Guard is detected and authorised server-side only.

Jump/drop/interact/portal are discrete commands and cannot be silently merged.

After the server authorizes a map/channel/instance destination it sends `S2C_TRANSFER_PREPARE` (`transfer_id`, destination `map_id` or `instance_id`, `content_revision`). The client downloads required Addressables per `../04_architecture/client_assets.md` and replies `C2S_PRESENTATION_READY` with that `transfer_id` only. The payload is not an asset list and not a chosen coordinate. Missing ready before `TRANSFER_BUDGET_*` in `../04_architecture/concurrency.md` is `TRANSFER_FAILED` plus source recovery.

Client coordinates may be included only as prediction context/diagnostic hints where explicitly defined; they never become authoritative world position.

## Combat
```text
ID   Name
200  C2S_SKILL_USE
201  C2S_BASIC_ATTACK
202  C2S_TARGET_INTENT
203  S2C_ACTION_STARTED
204  S2C_ACTION_REJECTED
205  S2C_STATUS_EVENT
206  S2C_DEATH
207  S2C_RESPAWN
208  C2S_RESPAWN_REQUEST
```

`S2C_COMBAT_EVENT` remains ID 304 in Replication. Client sends skill/target intent. Server owns action acceptance, hit, damage, healing, shield, status, cooldown/resource result, death, and respawn. `S2C_COMBAT_EVENT` includes `just_guard_window`, `just_guard_triggered`, `just_guard_hint`, and `beast_passive2_success`. Slow-mo juice derives from success flags only. The client never sends a presentation flag.

Positions on the wire are quantized to signed integer millimetres (`sint32 *_mm`, `0.001m` epsilon, `../04_architecture/physics_geometry_contract.md`). Enums used below: `facing : LEFT | RIGHT`; `damage_element : KIM | THUY | MOC | HOA | THO | PHYSICAL` (source: the skill's `damage_element` in `../07_content/class_skill_catalog.md`).

Combat field lists:
```text
C2S_SKILL_USE (200)       client_seq, client_mono_ms (advisory), skill_id, facing,
                          target_entity_id (uint64, 0 = none; SINGLE_TARGET requires it),
                          area_center_x_mm, area_center_y_mm (sint32; AREA_POSITION only, ignored otherwise;
                          clamped/rejected by cast range and collision per skills.md)
C2S_BASIC_ATTACK (201)    client_seq, client_mono_ms (advisory), facing, target_entity_id (0 = none).
                          Uses the equipped basic attack; the client never names a basic skill_id.
C2S_TARGET_INTENT (202)   client_seq, target_entity_id (0 = clear target)
S2C_ACTION_STARTED (203)  action_instance_id (uint64), source_entity_id, skill_id, client_seq (echo; 0 for
                          server-originated actions), server_tick, facing, target_entity_id,
                          area_center_x_mm, area_center_y_mm, cast_ms, cooldown_ends_at_tick, mp_after
S2C_ACTION_REJECTED (204) client_seq, request_message_id (200 | 201 | 202 | 208), skill_id,
                          error_code : COOLDOWN_ACTIVE | INSUFFICIENT_MP | SKILL_NOT_LEARNED | SKILL_LOADOUT_INVALID
                                     | TARGET_INVALID | OUT_OF_RANGE | INVALID_STATE | STALE_INPUT
S2C_STATUS_EVENT (205)    target_entity_id, source_entity_id, effect_id, status_kind (status_effects.md),
                          event : APPLIED | REFRESHED | STACK_CHANGED | EXPIRED | DISPELLED | CONSUMED,
                          stacks, expires_at_tick, server_tick
S2C_DEATH (206)           entity_id, killer_entity_id (0 = none), server_tick,
                          respawn_available_at_tick (0 when respawn is server-driven: dungeon, PvP, Guild War)
S2C_RESPAWN (207)         entity_id, map_id, checkpoint_id, x_mm, y_mm, hp_after, mp_after,
                          invulnerable_until_tick, server_tick
C2S_RESPAWN_REQUEST (208) operation_id
S2C_COMBAT_EVENT (304)    event_id (uint64, unique per partition), server_tick, action_instance_id,
                          source_entity_id, target_entity_id, skill_id (empty for effect ticks),
                          effect_id (DoT/HoT/shield/reflect source; empty for direct hits),
                          result_kind : DAMAGE | HEAL | SHIELD_GRANT | SHIELD_BREAK,
                          outcome : HIT | DODGED | INVULNERABLE (respawn protection; damage 0),
                          is_crit (bool), damage_element,
                          post_mitigation_damage, shield_absorbed, hp_damage, heal_amount, shield_amount
                          (int64 >= 0; combat.md § Damage terms),
                          target_hp_after, target_shield_after, killed (bool),
                          just_guard_window, just_guard_triggered, just_guard_hint,
                          beast_passive2_success, plus the secondary fields below
```
`C2S_RESPAWN_REQUEST` (208) is valid only for a `DEAD` character in the normal world when `server_tick >= respawn_available_at_tick` (`RESPAWN_DELAY`, `../01_gameplay/death_respawn.md`); success is `S2C_RESPAWN`, rejection is `S2C_ACTION_REJECTED` with `INVALID_STATE` (not dead, too early, or respawn is server-driven in dungeon/PvP/Guild War). A retry with the same `operation_id` after success re-sends the committed `S2C_RESPAWN`. A character that never sends 208 stays `DEAD`.


`S2C_COMBAT_EVENT` also carries the following secondary results generated at Global Effect Resolution Order stage 7 (ADR-0037). Each field is present only when the corresponding secondary result occurred in that event; absence means zero/not-triggered:

```text
reflect_damage_instance   -- integer >= 0; post-mitigation HP damage dealt to the attacker by
                             REFLECT; tagged NO_CRIT|NO_REFLECT|NO_LIFESTEAL|NO_PROC;
                             absent when REFLECT did not fire or melee range gate not met
lifesteal_heal_amount     -- integer >= 0; HP healed on the caster by LIFESTEAL after
                             LIFESTEAL_HPS_CAP throttle and all HEALING_RECEIVED
                             modifiers; absent when LIFESTEAL did not fire or heal
                             was zero after throttle
absorb_shield_amount      -- integer >= 0; HP shield granted to the caster by ABSORB;
                             reflects the final shield magnitude after shield pool cap and
                             the max(current_remaining, new_amount) reapplication rule;
                             absent when ABSORB did not fire or produced no shield change
```

The client renders these values as secondary visual feedback (floating numbers, VFX). The client **never** declares these values; it renders what the server sends. A missing field means the effect did not trigger; the client must not infer a non-zero value.

## Replication
```text
ID   Name
300  S2C_WORLD_BASELINE
301  S2C_ENTITY_SPAWN
302  S2C_ENTITY_DESPAWN
303  S2C_STATE_DELTA
304  S2C_COMBAT_EVENT     (see Combat section above)
305  S2C_ENCOUNTER_EVENT
306  C2S_BASELINE_ACK
```

Baseline and delta semantics are canonical in `synchronization.md`.

### S2C_ENCOUNTER_EVENT
Primary delivery vehicle for encounter-level events: boss phase transitions, telegraphed mechanics, and per-mechanic lifecycle. Delivery class: `AUTHORITATIVE_EVENT`.

Fields:
```text
encounter_id         : uint64 — server-assigned encounter instance identity; stable for the
                       lifetime of the encounter; matches the encounter_id in S2C_ENTITY_SPAWN
                       for the encounter entity
server_tick          : uint64 — authoritative server tick at event resolution
event_type           : enum
                         PHASE_TRANSITION       — boss changes combat phase
                         TELEGRAPH_START        — a telegraphed mechanic begins its
                                                  visual/warning window
                         TELEGRAPH_END          — telegraph window closes (attack fires or
                                                  was cancelled)
                         MECHANIC_TRIGGERED     — a named mechanic becomes active
                         MECHANIC_EXPIRED       — a named mechanic's active window ends
                         VULNERABILITY_OPEN     — a vulnerability window opens on the
                                                  source entity
                         VULNERABILITY_CLOSE    — the vulnerability window closes
mechanic_id          : uint32 — stable numeric ID of the named mechanic as registered in
                                content spec; 0 when event_type is PHASE_TRANSITION
source_entity_id     : uint64 — runtime entity ID of the entity generating the event
                                (e.g., the boss); references the entity lifecycle in
                                S2C_ENTITY_SPAWN / S2C_ENTITY_DESPAWN
affected_entity_ids  : repeated uint64 — runtime entity IDs of players or entities
                                         targeted or affected; empty list = all in AOI
phase_number         : uint32 — new phase number when event_type is PHASE_TRANSITION;
                                absent (0) for all other event types; Phase 2 is
                                expressed as a PHASE_TRANSITION with phase_number = 2
mechanic_payload     : bytes  — mechanic-specific protobuf sub-message, interpreted
                                according to mechanic_id; absent when mechanic_id = 0
                                or the mechanic carries no extra data
```

The client renders encounter events for telegraphs, VFX cues, phase UI transitions, and boss mechanic presentation. The client never declares encounter state; the server is authoritative.

`S2C_STATE_DELTA` replicates the current derived stat values for the five new stats (ADR-0037) whenever they change on a visible entity:

```text
stat_lifesteal        -- current effective LIFESTEAL ratio for the entity (post-PvP cap where applicable)
stat_reflect          -- current effective REFLECT ratio for the entity (post-PvP cap where applicable)
stat_absorb           -- current effective ABSORB ratio for the entity (post-PvP cap where applicable)
stat_heal_reduction   -- current effective HEAL_REDUCTION ratio for the entity (post-PvP cap where applicable)
stat_healing_received -- current effective HEALING_RECEIVED multiplier for the entity
```

These are derived projection values. The client uses them for UI display (e.g., stat panel, combat tooltips). The client never writes these fields back to the server. Fields absent from a delta message are unchanged from the last sent value.

## Durable Operations
Complete launch request/result pairs (ADR-0054):
```text
ID   Name
400  C2S_INVENTORY_MUTATE
401  S2C_INVENTORY_RESULT
402  C2S_LOADOUT_CHANGE
403  S2C_LOADOUT_RESULT
404  C2S_CRAFT
405  S2C_CRAFT_RESULT
406  C2S_ENHANCE
407  S2C_ENHANCE_RESULT
408  C2S_REWARD_CLAIM
409  S2C_REWARD_CLAIM_RESULT
410  C2S_BEAST_SET_ACTIVE
411  S2C_BEAST_SET_ACTIVE_RESULT
412  C2S_BEAST_EQUIP
413  S2C_BEAST_EQUIP_RESULT
414  C2S_BEAST_UNEQUIP
415  S2C_BEAST_UNEQUIP_RESULT
416  C2S_BEAST_FEED
417  S2C_BEAST_FEED_RESULT
418  C2S_ENTITLEMENT_CLAIM
419  S2C_ENTITLEMENT_CLAIM_RESULT
420  C2S_NPC_SHOP_BUY
421  S2C_NPC_SHOP_BUY_RESULT
422  C2S_COSMETIC_REDEEM
423  S2C_COSMETIC_REDEEM_RESULT
424  C2S_COSMETIC_EQUIP
425  S2C_COSMETIC_EQUIP_RESULT
426  C2S_NPC_SHOP_SELL
427  S2C_NPC_SHOP_SELL_RESULT
428  C2S_INVENTORY_EXPAND
429  S2C_INVENTORY_EXPAND_RESULT
430  C2S_BEAST_LEVEL_UP
431  S2C_BEAST_LEVEL_UP_RESULT
432  S2C_WALLET_STATE
433  S2C_INVENTORY_STATE
434  S2C_REWARD_CLAIMS_STATE
435  S2C_ENTITLEMENT_PANEL_STATE
436  S2C_BEAST_STATE
437  S2C_SOUL_STATE
438  S2C_COSMETIC_STATE
439..499  (reserved, unassigned)
```

Every result below carries `operation_id`, `status : SUCCESS | ERROR` and `error_code` (`NONE` on success; domain list in `errors.md`). A retry with the same `operation_id` returns the committed result.

- **C2S_INVENTORY_MUTATE (400)**: `operation_id`, `op : MOVE | SPLIT | MERGE | SORT | DISCARD | USE`, `item_instance_id` (all but SORT), `to_slot : uint32` (MOVE/SPLIT/MERGE; MOVE onto an occupied slot swaps), `quantity : uint32` (SPLIT/DISCARD: stack units, 0 = whole stack for DISCARD; USE: 1). Rules: `../03_systems/inventory.md`, `../03_systems/items.md` (Consumables, Transfer / Use / Discard, Bonus Books). `USE` covers every usable item: potions (shared cooldown groups), food buffs, `item.book.potential` / `item.book.skill` (grant unspent points; idempotent by `operation_id`). Beast food uses 416, never USE. Errors: `ITEM_NOT_FOUND`, `ITEM_LOCKED`, `INVENTORY_FULL`, `COOLDOWN_ACTIVE`, `INVALID_STATE` (dead, item not usable/discardable in this context), `STATE_CONFLICT`.
- **S2C_INVENTORY_RESULT (401)**: `op`, `inventory_revision : uint64`, `changed_slots : list of {slot, item_instance_id (empty = now empty), item_id, quantity}`, `use_effect : {potential_points_granted, skill_points_granted, cooldown_group, cooldown_ends_at_tick}` (USE only).
- **C2S_LOADOUT_CHANGE (402)**: `operation_id`, `kind : EQUIP | UNEQUIP | SWITCH_ACTIVE | SKILL_SET | SOUL_CONTRACT`, then per kind:
  - EQUIP: `loadout_id : loadout.primary | loadout.secondary_1 | loadout.secondary_2`, `slot_id` (`../03_systems/equipment.md` § Slots), `item_instance_id` (from CHARACTER_INVENTORY; a displaced item returns to inventory).
  - UNEQUIP: `loadout_id`, `slot_id` (requires inventory capacity; contracted Soul returns to Collection atomically).
  - SWITCH_ACTIVE: `loadout_id` (rules `equipment.md` § Loadout Switch).
  - SKILL_SET: `basic_skill_id` (one learned BASIC_ATTACK), `active_slots : 5 x string` (learned ACTIVE skill_id or empty; no duplicates) (`../01_gameplay/skills.md`).
  - SOUL_CONTRACT: `action : CREATE | REMOVE | MOVE | REPLACE`, `soul_instance_id`, `target_item_instance_id` (CREATE/MOVE/REPLACE: equipment in a loadout). REPLACE returns the soul currently on the target item to Collection in the same transaction; MOVE removes the soul from its current item. Rules `../03_systems/soul_contracts.md` § Contract, § Mutation.
  Errors: `ITEM_NOT_FOUND`, `ITEM_LOCKED`, `SLOT_MISMATCH`, `LEVEL_TOO_LOW`, `INVENTORY_FULL`, `IN_COMBAT`, `SKILL_NOT_LEARNED`, `SKILL_LOADOUT_INVALID`, `SOUL_CONTRACT_LIMIT_REACHED`, `STATE_CONFLICT`.
- **S2C_LOADOUT_RESULT (403)**: `kind`, `loadout_revision : uint64`, `active_loadout_id`, `changed_slots : list of {loadout_id, slot_id, item_instance_id}`, `skill_loadout : {basic_skill_id, active_slots[5]}` (SKILL_SET), `soul_contracts : list of {soul_instance_id, item_instance_id (empty = Collection)}` (SOUL_CONTRACT and UNEQUIP).
- **C2S_CRAFT (404)**: `operation_id`, `npc_id` (NPC with SERVICE(crafting) in range, or `cooking_hearth` via 103 COOK instead), `recipe_id`, `batch_quantity : uint32 (1..99)`. Rules `../03_systems/crafting.md`. Errors: `INSUFFICIENT_ITEM`, `INSUFFICIENT_CURRENCY`, `INVENTORY_FULL`, `LEVEL_TOO_LOW`, `OUT_OF_RANGE`.
- **S2C_CRAFT_RESULT (405)**: `recipe_id`, `batch_quantity`, `consumed : list of {item_id, quantity}`, `currency_delta : list of {currency_id, amount}`, `granted : list of {item_instance_id, item_id, quantity}`.
- **C2S_ENHANCE (406)**: `operation_id`, `npc_id` (SERVICE(enhancement) in range), `item_instance_id`, `target_level : uint32 (current + 1)`, `lucky_charm_item_instance_id` (optional), `insurance_item_instance_id` (optional). At most one of each (`crafting.md` § Lucky Charm, § Insurance). Errors: `ITEM_NOT_FOUND`, `ITEM_LOCKED`, `INSUFFICIENT_ITEM`, `INSUFFICIENT_CURRENCY`, `CHARM_INELIGIBLE`, `STATE_CONFLICT` (target_level != current + 1), `OUT_OF_RANGE`.
- **S2C_ENHANCE_RESULT (407)**: `item_instance_id`, `success : bool`, `level_before`, `level_after`, `final_rate_bp`, `pity_fail_count`, `consumed : list of {item_id, quantity}`, `currency_delta`.
- **C2S_REWARD_CLAIM (408)**: `operation_id`, `reward_claim_id : UUID`. Rules `../03_systems/reward_claims.md`. Errors: `INVENTORY_FULL`, `CURRENCY_CAP_EXCEEDED`, `EXPIRED`, `NOT_OWNER`, `STATE_CONFLICT` (already CLAIMED returns the committed result instead).
- **S2C_REWARD_CLAIM_RESULT (409)**: `reward_claim_id`, `granted : list of {item_instance_id, item_id, quantity}`, `currency_delta`.
- **C2S_NPC_SHOP_SELL (426)**: `operation_id`, `npc_id` (NPC with SHOP in range), `item_instance_id`, `quantity : uint32 (>=1)`. Price = the item's `sell_back_price` (`../07_content/npc_shop_catalog.md`, `../03_systems/economy.md` § NPC Sell-Back) × quantity; items without one reject `INVALID_STATE`. Errors also: `ITEM_LOCKED`, `CURRENCY_CAP_EXCEEDED`, `OUT_OF_RANGE`. Result 427: `item_id`, `quantity_sold`, `currency_delta`.
- **C2S_INVENTORY_EXPAND (428)**: `operation_id`, `expected_capacity : uint32` (current capacity; mismatch = `STATE_CONFLICT`). Buys the next `+10` step at the `../03_systems/inventory.md` price. Errors: `INSUFFICIENT_CURRENCY`, `CAPACITY_FULL` (already 120). Result 429: `capacity_after`, `currency_delta`.
- **C2S_BEAST_LEVEL_UP (430)**: `operation_id`, `beast_id`, `expected_level : uint32` (current level; mismatch = `STATE_CONFLICT`). Consumes the Linh Đan and `currency.common` cost of the next level (`../07_content/spirit_beast_catalog.md`; rules `../03_systems/spirit_beasts.md`). Errors: `BEAST_NOT_OWNED`, `INSUFFICIENT_ITEM`, `INSUFFICIENT_CURRENCY`, `LEVEL_TOO_LOW` (beast level would exceed character level), `CAPACITY_FULL` (beast level 60). Result 431: `beast_id`, `level_after`, `consumed`, `currency_delta`.

State pushes (`REPLACEABLE_STATE`, full snapshot each time; sent after `S2C_CHARACTER_ATTACH_OK` and after every committed change affecting them):
```text
432 S2C_WALLET_STATE             balances : list of {currency_id, amount, cap}; wallet_revision
433 S2C_INVENTORY_STATE          capacity, inventory_revision, slots : list of {slot, item_instance_id, item_id, quantity,
                                 effective_binding, enhancement_level, locked}; loadouts : 3 x {loadout_id, is_active,
                                 slots : list of {slot_id, item_instance_id}}; loadout_revision
434 S2C_REWARD_CLAIMS_STATE      claims : list of {reward_claim_id, source_type, source_reference, reward_slot, state, lines : list of {item_id | currency_id,
                                 quantity}, expires_at}; pending_count, cap (100)
435 S2C_ENTITLEMENT_PANEL_STATE  entitlements : list of {entitlement_id, product_id, entitlement_type, grant_state,
                                 season_number, claim_deadline_at, claimable_tier_ids, claimed_tier_ids (this character)}
436 S2C_BEAST_STATE              beasts : list of {beast_id, level, bond_points, daily_food_points_gained, is_active,
                                 equipment : list of {slot_id, item_instance_id}}
437 S2C_SOUL_STATE               souls : list of {soul_instance_id, soul_id, level, current_soul_exp,
                                 contracted_item_instance_id}; resonance : list of {soul_id, memory_resonance_count}
438 S2C_COSMETIC_STATE           owned : list of {cosmetic_id, scope : CHARACTER | ACCOUNT}; equipped : list of {slot, cosmetic_id}
```

- **C2S_NPC_SHOP_BUY (420)**: `operation_id : UUID`, `npc_id : string`, `offer_id : string`, `quantity : uint32 (>=1)`. Prices/limits: `../07_content/npc_shop_catalog.md`. Result 421: `operation_id`, `status`, `error_code` (`errors.md` domain list), `granted : list of {item_id, quantity}`.
- **C2S_COSMETIC_REDEEM (422)**: `operation_id : UUID`, `cosmetic_id : string`, `route : CURRENCY_SPECIAL | CURRENCY_COMMON | MATERIAL`. Rules: `../03_systems/cosmetics.md` (Material Redemption, Redemption Guardrails). Result 423: `operation_id`, `status`, `error_code`, `cosmetic_id`.
- **C2S_COSMETIC_EQUIP (424)**: `operation_id : UUID`, `slot : TITLE | TITLE_GLOW | FRAME | NAMEPLATE | APPEARANCE | WEAPON_TRAIL | AURA | CHARACTER_SHRINE | GUILD_STONE_INSCRIPTION` (character slots in `../03_systems/cosmetics.md` § Categories), `cosmetic_id : string` (empty = unequip). Result 425: `operation_id`, `status`, `error_code`, `slot`, `cosmetic_id`.

### Spirit Beast Message Contract (410..417)
Spirit Beast operations use dedicated messages (410..417). Do not reuse 408. All requests carry a client-generated UUID `operation_id` for idempotency.

- **C2S_BEAST_SET_ACTIVE (410)**: `operation_id : UUID`, `beast_id : string` (target companion to summon/activate; empty = deactivate the current beast, leaving `active_beast_count = 0`).
- **S2C_BEAST_SET_ACTIVE_RESULT (411)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `active_beast_id : string` (empty = none active), `error_code : NONE | BEAST_NOT_OWNED | IN_COMBAT`.
- **C2S_BEAST_EQUIP (412)**: `operation_id : UUID`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `item_instance_id : UUID` (piece in character inventory). Wire-to-data slot mapping: `COLLAR` → `vong_co`, `BARDING` → `ao_giap`, `ORB` → `linh_chau` (canonical slot IDs in `../06_data/data_model.md`).
- **S2C_BEAST_EQUIP_RESULT (413)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `equipped_item_instance_id : UUID`, `unequipped_item_instance_id : UUID (optional)`, `error_code : NONE | BEAST_NOT_OWNED | ITEM_NOT_FOUND | SLOT_MISMATCH | LEVEL_TOO_LOW`.
- **C2S_BEAST_UNEQUIP (414)**: `operation_id : UUID`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`.
- **S2C_BEAST_UNEQUIP_RESULT (415)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `unequipped_item_instance_id : UUID`, `error_code : NONE | BEAST_NOT_OWNED | SLOT_EMPTY | INVENTORY_FULL`.
- **C2S_BEAST_FEED (416)**: `operation_id : UUID`, `beast_id : string`, `food_item_id : string` (bond value per food and active/any-beast rule: `../03_systems/spirit_beasts.md`, values in `../07_content/item_catalog.md`), `quantity : uint32` (default 1).
- **S2C_BEAST_FEED_RESULT (417)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `beast_id : string`, `consumed_quantity : uint32`, `new_bond_points : uint32 (0..100)`, `daily_food_points_gained : uint32 (0..20)`, `error_code : NONE | BEAST_NOT_OWNED | INSUFFICIENT_ITEM | DAILY_FOOD_CAP_REACHED | MAX_BOND_REACHED`.

`C2S_ENTITLEMENT_CLAIM` (418): `operation_id`, `entitlement_id`, optional `reward_tier_id`. Result is `S2C_ENTITLEMENT_CLAIM_RESULT` (419). Cross-account receipt replay is `IAP_RECEIPT_ACCOUNT_MISMATCH`.
### Quest / Atlas (500..506)
Do not reuse 408 for atlas claim.
```text
ID   Name
500  C2S_QUEST_ACCEPT
501  S2C_QUEST_ACCEPT_RESULT
502  C2S_QUEST_TURN_IN
503  S2C_QUEST_UPDATE
504  C2S_ATLAS_CLAIM
505  S2C_ATLAS_CLAIM_RESULT
506  S2C_PROGRESSION_EVENT
507  C2S_QUEST_ABANDON
508  S2C_QUEST_ABANDON_RESULT
509  C2S_STORY_BRANCH_CHOOSE
510  S2C_STORY_BRANCH_RESULT
511  C2S_SKILL_UPGRADE
512  C2S_POTENTIAL_ALLOCATE
513  C2S_RESPEC
514  S2C_PROGRESSION_MUTATE_RESULT
515  S2C_PROGRESSION_STATE
516..599  (reserved, unassigned)
```

`C2S_ATLAS_CLAIM` (504): `operation_id`, `atlas_page_id`, `tier`. Acknowledge only: tier rewards auto-settle at promotion; the result returns the existing grant and sets `acknowledged_at` once; it never grants. `S2C_ATLAS_CLAIM_RESULT` (505): `operation_id`, `status`, `error_code` (`ATLAS_TIER_NOT_REACHED` when the tier is not reached; nothing changes), `atlas_page_id`, `tier`, `granted` (the settled grant). Rules `../03_systems/atlas.md`.

- **C2S_QUEST_ABANDON (507)**: `operation_id`, `quest_id`. Allowed for SIDE, DAILY and abandonable EVENT quests; MAIN rejects `INVALID_STATE` (`../02_world/quests.md` § Abandon). Result 508: `operation_id`, `status`, `error_code`, `quest_id`, `removed_items : list of {item_id, quantity}`; followed by `S2C_QUEST_UPDATE`.
- **C2S_STORY_BRANCH_CHOOSE (509)**: `operation_id`, `branch_flag` (e.g. `progression.story.a1.resolution`), `option` (one of the authored options in `../07_content/quest_catalog.md`). Allowed only while the owning MAIN quest is active and the flag is unset; the choice is permanent. Errors: `INVALID_STATE` (quest not active), `ALREADY_OWNED` (flag already set), `TARGET_INVALID` (unknown option). Result 510: `operation_id`, `status`, `error_code`, `branch_flag`, `option`.
- **C2S_SKILL_UPGRADE (511)**: `operation_id`, `skill_id`, `expected_level : uint32` (current level; mismatch = `STATE_CONFLICT`). Costs 1 unspent skill point (`../01_gameplay/progression.md` § Skill Points). Errors: `SKILL_NOT_LEARNED`, `SKILL_MAX_LEVEL`, `SKILL_POINTS_INSUFFICIENT`.
- **C2S_POTENTIAL_ALLOCATE (512)**: `operation_id`, `deltas : {str, vit, int, agi}` (uint32 each, sum >= 1). All-or-nothing against unspent points and the 60% per-stat cap (`../01_gameplay/stats.md`). Errors: `POTENTIAL_POINTS_INSUFFICIENT`, `POTENTIAL_CAP_EXCEEDED`.
- **C2S_RESPEC (513)**: `operation_id`, `npc_id` (NPC with SERVICE(respec) in range), `kind : SKILL | POTENTIAL`. Full reset of that kind at the `progression.md` § Respec price, refunding all spent points of that kind; learned skills stay learned and equipped at their base level (spent upgrade points are refunded). Errors: `INSUFFICIENT_CURRENCY`, `IN_COMBAT`, `INVALID_STATE` (active PvP or encounter build lock), `OUT_OF_RANGE`.
- **S2C_PROGRESSION_MUTATE_RESULT (514)**: `operation_id`, `request_message_id (511 | 512 | 513)`, `status`, `error_code`, followed by `S2C_PROGRESSION_STATE`.
- **S2C_PROGRESSION_STATE (515)** (`REPLACEABLE_STATE`, full snapshot after attach and every change): `level`, `current_exp`, `unspent_skill_points`, `unspent_potential_points`, `potential_allocated : {str, vit, int, agi}`, `potential_earned_total`, `skills : list of {skill_id, level}`, `skill_loadout : {basic_skill_id, active_slots[5]}`, `progression_revision : uint64`.

### Trade ID Reservation (700..729)
All trade message IDs must be registered here. A content spec MUST NOT use an ID in this sub-range without adding a row to this table.
```text
ID   Name                       Direction   Notes
700  C2S_TRADE_INVITE           C→S         initiate direct trade with target character
701  S2C_TRADE_INVITE           S→C         deliver trade invitation to target
702  C2S_TRADE_ACCEPT           C→S         accept pending trade invitation
703  C2S_TRADE_CANCEL           C→S         cancel or decline trade at any stage
704  S2C_TRADE_CANCELLED        S→C         trade cancelled; includes cancellation reason
705  C2S_TRADE_OFFER_UPDATE     C→S         add/remove items from own offer slot
706  S2C_TRADE_OFFER_STATE      S→C         both sides' offer state after any change
707  C2S_TRADE_CONFIRM          C→S         lock own offer, ready to finalise
708  C2S_TRADE_FINALISE         C→S         both confirmed — commit the exchange
709  S2C_TRADE_RESULT           S→C         authoritative trade outcome; items transferred
710..729  (reserved, unassigned)
```

Trade field lists (rules `../03_systems/trading_auction.md` § Direct Trade). Offered items stay in the owner's `CHARACTER_INVENTORY`, locked by the session (`ITEM_LOCKED` for other mutations), until settlement; there is no trade escrow row (`../06_data/data_model.md` § Auction / Trade):
```text
700 C2S_TRADE_INVITE        operation_id, target_character_id
701 S2C_TRADE_INVITE        trade_id, inviter_character_id, inviter_name, expires_in_seconds
702 C2S_TRADE_ACCEPT        operation_id, trade_id
703 C2S_TRADE_CANCEL        operation_id, trade_id (decline before accept = cancel)
704 S2C_TRADE_CANCELLED     trade_id, reason : DECLINED | CANCELLED | EXPIRED | INACTIVE_TIMEOUT | PARTNER_DISCONNECTED
                            | VALIDATION_FAILED, error_code
705 C2S_TRADE_OFFER_UPDATE  operation_id, trade_id, expected_revision, items : list of {item_instance_id, quantity}
                            (full replacement of own offer; max entries per trading_auction.md), common_amount (int64 >= 0)
706 S2C_TRADE_OFFER_STATE   trade_id, revision, state : OPEN | LOCKED | COMMITTING,
                            sides : 2 x {character_id, items : list of {item_instance_id, item_id, quantity, enhancement_level},
                            common_amount, confirmed (bool)}, fee_preview
707 C2S_TRADE_CONFIRM       operation_id, trade_id, expected_revision (locks own side; any offer change clears both)
708 C2S_TRADE_FINALISE      operation_id, trade_id, expected_revision (both confirmed)
709 S2C_TRADE_RESULT        trade_id, status : COMPLETED | ERROR, error_code, settlement_id,
                            received : list of {item_instance_id, item_id, quantity}, common_received, fee
```
Trade errors: `TRADE_ELIGIBILITY_LEVEL_REQUIRED`, `TRADE_ELIGIBILITY_AGE_REQUIRED`, `SAME_ACCOUNT_FORBIDDEN`, `TARGET_BLOCKED`, `OUT_OF_RANGE`, `IN_COMBAT`, `ITEM_LOCKED`, `STATE_CONFLICT` (revision mismatch), `TRADE_COMMON_BOTH_SIDES`, `TRADE_PRICE_FLOOR_NOT_MET`, `CURRENCY_CAP_EXCEEDED`, `INVENTORY_FULL`, `TRADE_PARTNER_DISCONNECTED`.

### Auction ID Reservation (730..759)
All auction message IDs must be registered here. A content spec MUST NOT use an ID in this sub-range without adding a row to this table.
```text
ID   Name                       Direction   Notes
730  C2S_AUCTION_LIST           C→S         create a new listing; carries operation_id
731  S2C_AUCTION_LIST_RESULT    S→C         listing accepted or rejected
732  C2S_AUCTION_BUY            C→S         purchase a FIXED_PRICE listing; carries operation_id
733  S2C_AUCTION_BUY_RESULT     S→C         purchase accepted or rejected
734  C2S_AUCTION_CANCEL_LISTING C→S         cancel own active listing; carries operation_id
735  S2C_AUCTION_CANCEL_RESULT  S→C         listing cancelled; item remains in escrow pending explicit reclaim (see trading_auction.md)
736  S2C_AUCTION_SOLD           S→C         notifies seller that listing was purchased
737  (retired unused)           —           retired; do not reuse
738  C2S_AUCTION_SEARCH         C→S         filters: item_id/category/tier/price range; page cursor (max 50 rows)
739  S2C_AUCTION_SEARCH_RESULT  S→C         listings page + next cursor
740  C2S_AUCTION_RECLAIM        C→S         reclaim cancelled/expired escrow asset; carries operation_id
741  S2C_AUCTION_RECLAIM_RESULT S→C         reclaim outcome (INVENTORY_FULL keeps asset in escrow)
742  C2S_AUCTION_PROCEEDS_CLAIM C→S         claim PENDING proceeds; carries operation_id
743  S2C_AUCTION_PROCEEDS_RESULT S→C        claim outcome (CURRENCY_CAP_EXCEEDED keeps it PENDING)
744  S2C_AUCTION_MY_STATE       S→C         own listings, escrow assets (with auto-claim deadline) and pending proceeds
745..759  (reserved, unassigned)
```

Auction field lists (rules `../03_systems/trading_auction.md` § Auction House; every result carries `operation_id`, `status`, `error_code`):
```text
730 C2S_AUCTION_LIST            operation_id, item_instance_id, quantity (whole lot), price_common (int64; total lot price)
731 S2C_AUCTION_LIST_RESULT     listing_id, listing_fee, expires_at
732 C2S_AUCTION_BUY             operation_id, listing_id, expected_price_common
733 S2C_AUCTION_BUY_RESULT      listing_id, price_common, received : {item_instance_id, item_id, quantity}
734 C2S_AUCTION_CANCEL_LISTING  operation_id, listing_id
735 S2C_AUCTION_CANCEL_RESULT   listing_id, escrow_asset_id
736 S2C_AUCTION_SOLD            listing_id, item_id, quantity, price_common, tax, proceeds_id, proceeds_amount
738 C2S_AUCTION_SEARCH          item_id, category, tier, min_price, max_price, sort : PRICE_ASC | PRICE_DESC | NEWEST,
                                page_cursor (opaque), page_size (1..50)
739 S2C_AUCTION_SEARCH_RESULT   rows : list of {listing_id, item_id, quantity, enhancement_level, price_common,
                                seller_display_name, expires_at}, next_page_cursor
740 C2S_AUCTION_RECLAIM         operation_id, escrow_asset_id
741 S2C_AUCTION_RECLAIM_RESULT  escrow_asset_id, received : {item_instance_id, item_id, quantity}
742 C2S_AUCTION_PROCEEDS_CLAIM  operation_id, proceeds_id
743 S2C_AUCTION_PROCEEDS_RESULT proceeds_id, amount_common
744 S2C_AUCTION_MY_STATE        listings : list of {listing_id, item_id, quantity, price_common, state, expires_at},
                                escrow_assets : list of {escrow_asset_id, item_id, quantity, reason : CANCELLED | EXPIRED,
                                auto_claim_at}, proceeds : list of {proceeds_id, amount_common, state}
```
Auction errors: `AH_ELIGIBILITY_LEVEL_REQUIRED`, `AH_ELIGIBILITY_AGE_REQUIRED`, `AH_PRICE_FLOOR_NOT_MET`, `SAME_ACCOUNT_FORBIDDEN`, `AUCTION_LISTING_NOT_ACTIVE`, `CAPACITY_FULL` (20 active listings), `ITEM_LOCKED`, `INSUFFICIENT_CURRENCY`, `INVENTORY_FULL`, `CURRENCY_CAP_EXCEEDED`, `STATE_CONFLICT` (expected price mismatch).

Every retriable value-affecting request carries a stable `operation_id`. Retry returns the committed outcome; it does not execute a second logical mutation or reroll RNG.

### Per-Operation Sub-limit Buckets
The `durable mutations` bucket is the aggregate ceiling for all durable operations. The following messages additionally belong to **stricter per-character sub-limits** that are checked and enforced before the aggregate bucket. Exact limits, rationale, and escalation rules are owned by `../07_security/rate_limits.md`; do not restate numbers here.

```text
Message                    Sub-limit bucket (see rate_limits.md)
------------------------------------------------------------------------
C2S_TRADE_INVITE   (700)   direct trade invitation sub-limit
C2S_AUCTION_LIST   (730)   auction listing creation sub-limit
C2S_AUCTION_BUY    (732)   auction purchase sub-limit
C2S_REWARD_CLAIM   (408)   reward claim sub-limit
social invite messages     party / guild / friend invite sub-limit
  (C2S_PARTY_INVITE 602, C2S_GUILD_INVITE 608, C2S_FRIEND_REQUEST 611)
```

Message handlers for these operations MUST apply the sub-limit check before the aggregate durable-mutations bucket. A sub-limit violation returns `RATE_LIMITED` with the specific operation name in the retry hint; see `../07_security/rate_limits.md` for escalation rules. These messages are NOT governed solely by the generic durable-mutations bucket.

## Social / Guild (600..699)
All social, party, friend, block, report, and guild message IDs must be registered here. All client requests carry a client-generated UUID `operation_id` for idempotency.

```text
ID   Name                       Direction   Notes
600  C2S_CHAT_SEND              C→S         send chat message across active channels
601  S2C_CHAT_MESSAGE           S→C         fanout chat delivery to recipients
602  C2S_PARTY_INVITE           C→S         party leader invites target character
603  S2C_PARTY_INVITE           S→C         party invitation delivery to target
604  C2S_PARTY_ACCEPT           C→S         accept pending party invitation
605  C2S_PARTY_LEAVE            C→S         voluntarily leave current party
606  C2S_PARTY_KICK             C→S         party leader kicks member
607  S2C_PARTY_STATE            S→C         party roster, leadership, and status sync
608  C2S_GUILD_INVITE           C→S         guild officer/leader invites character
609  S2C_GUILD_INVITE           S→C         guild invitation delivery to target
610  C2S_GUILD_ACCEPT           C→S         accept pending guild invitation
611  C2S_FRIEND_REQUEST         C→S         send mutual friend request
612  S2C_FRIEND_REQUEST         S→C         friend request delivery to target
613  C2S_FRIEND_ACCEPT          C→S         accept pending friend request
614  C2S_FRIEND_DECLINE         C→S         decline incoming friend request
615  C2S_FRIEND_REMOVE          C→S         remove established friendship
616  S2C_FRIEND_STATE           S→C         friend list and presence update
617  C2S_BLOCK_ADD              C→S         block character (auto-unfriends and severs invites)
618  C2S_BLOCK_REMOVE           C→S         unblock character
619  S2C_BLOCK_STATE            S→C         block list update
620  C2S_PARTY_DECLINE          C→S         decline incoming party invite
621  C2S_PARTY_INVITE_CANCEL    C→S         cancel outbound party invite
622  C2S_PARTY_LEADER_TRANSFER  C→S         transfer party leadership
623  C2S_GUILD_DECLINE          C→S         decline incoming guild invite
624  C2S_GUILD_LEAVE            C→S         leave current guild
625  C2S_GUILD_KICK             C→S         kick member from guild
626  C2S_GUILD_ROLE_UPDATE      C→S         promote or demote guild member
627  C2S_GUILD_LEADER_TRANSFER  C→S         transfer guild leadership
628  S2C_GUILD_STATE            S→C         guild membership, roles, and status sync
629  C2S_GUILD_STORAGE_DEPOSIT  C→S         deposit UNBOUND item into guild storage
630  C2S_GUILD_STORAGE_WITHDRAW C→S         withdraw item from guild storage into inventory
631  S2C_GUILD_STORAGE_STATE    S→C         guild storage contents and revision update
632  C2S_REPORT_PLAYER          C→S         submit player report (chat, botting, cheat)
633  S2C_REPORT_PLAYER_RESULT   S→C         report submission acknowledgement
634  C2S_PARTY_BOARD_POST       C→S         post LFG on safe-anchor party board (120s)
635  C2S_PARTY_BOARD_CANCEL     C→S         cancel own safe-anchor party board post
636  S2C_PARTY_BOARD_STATE      S→C         safe-anchor party board listings sync
637  C2S_GUILD_CREATE           C→S         create guild
638  C2S_GUILD_DISBAND          C→S         leader disbands guild
639  C2S_GUILD_APPLY            C→S         apply to a guild in APPLICATIONS mode
640  C2S_GUILD_APPLICATION_DECIDE C→S       accept/reject applicant
641  S2C_GUILD_APPLICATIONS     S→C         pending applications list
642  C2S_GUILD_MOTD_SET         C→S         set message of the day
643  C2S_GUILD_LEADERSHIP_CLAIM C→S         inactivity leadership claim
644  C2S_GUILD_STORAGE_MOVE     C→S         move item COMMON <-> RESERVE
645  C2S_GUILD_STORAGE_CLAIM_REQUEST C→S    request a Reserve item
646  C2S_GUILD_STORAGE_CLAIM_DECIDE  C→S    approve/reject/cancel/deliver Reserve claim
647  S2C_GUILD_STORAGE_CLAIMS   S→C         Reserve claim list
648  C2S_GUILD_BLESSING_VOTE    C→S         vote for a Blessing candidate
649  S2C_GUILD_RESULT           S→C         typed result for guild requests
650  C2S_GUILD_SETTINGS_SET     C→S         set guild settings (recruitment mode)
651  C2S_GUILD_INVITE_CANCEL    C→S         cancel a PENDING outbound guild invite
652  C2S_GUILD_APPLICATION_CANCEL C→S       applicant cancels own PENDING application
653..699 (reserved, unassigned)
```

### Social / Guild Payload Contracts

- **Chat (600..601)**:
  - `C2S_CHAT_SEND`: `channel : WORLD | LOCAL | PARTY | GUILD | WHISPER`, `target_character_id : UUID (optional, required for WHISPER)`, `message_text : string (1..240 graphemes; validated per social.md § Message Content)`. Rate limits: per-channel limits in `../03_systems/social.md` § Rate Limits.
  - `S2C_CHAT_MESSAGE`: `chat_message_id : UUID` (= `chat_messages.message_id`; referenced by reports), `channel`, `sender_character_id : UUID`, `sender_name : string`, `message_text : string`, `timestamp : int64`.
- **Friends & Blocks (611..619)**:
  - `C2S_FRIEND_REQUEST` (611): `operation_id : UUID`, `target_character_id : UUID`. Rate limit: social invite (10/60s). Error codes: `FRIEND_LIMIT_REACHED`, `TARGET_BLOCKED`, `ALREADY_FRIENDS`, `PENDING_REQUEST_EXISTS`.
  - `C2S_FRIEND_ACCEPT` (613): `operation_id : UUID`, `requester_character_id : UUID`. Mutual friendship created in `friends`.
  - `C2S_FRIEND_DECLINE` (614): `operation_id : UUID`, `requester_character_id : UUID`. Request marked DECLINED.
  - `C2S_FRIEND_REMOVE` (615): `operation_id : UUID`, `target_character_id : UUID`. Friendship severed atomically.
  - `S2C_FRIEND_STATE` (616): `friend_character_id : UUID`, `display_name : string`, `online_state : ONLINE | OFFLINE`, `zone_id : string`, `activity : WORLD | DUNGEON | PVP`.
  - `C2S_BLOCK_ADD` (617): `operation_id : UUID`, `target_character_id : UUID`. Inserts block, severs mutual friendship, cancels pending requests.
  - `C2S_BLOCK_REMOVE` (618): `operation_id : UUID`, `target_character_id : UUID`. Removes block.
  - `S2C_BLOCK_STATE` (619): `blocked_character_id : UUID`, `display_name : string`, `action : ADDED | REMOVED`.
- **Party (602..607, 620..622, 634..636)**:
  - `C2S_PARTY_INVITE` (602): `operation_id : UUID`, `target_character_id : UUID`. Sender must be leader, or partyless: a partyless sender's first invite atomically creates a party with the sender as leader (`party_revision = 1`) before the invite is issued; if that invite then ends without acceptance the one-member party remains valid until the leader leaves (`../03_systems/party.md`). Target online, partyless, unblocked. 60s TTL. Rate limit: social invite (10/60s).
  - `S2C_PARTY_INVITE` (603): `party_id : UUID`, `inviter_character_id : UUID`, `inviter_name : string`, `expires_in_seconds : uint32`.
  - `C2S_PARTY_ACCEPT` (604): `operation_id : UUID`, `party_id : UUID`, `inviter_character_id : UUID`. Validates capacity <= 5 at commit.
  - `C2S_PARTY_DECLINE` (620): `operation_id : UUID`, `party_id : UUID`, `inviter_character_id : UUID`.
  - `C2S_PARTY_INVITE_CANCEL` (621): `operation_id : UUID`, `target_character_id : UUID`.
  - `C2S_PARTY_LEAVE` (605): `operation_id : UUID`. Leaves party; lowest join_sequence inherits leader; disbands if last member.
  - `C2S_PARTY_KICK` (606): `operation_id : UUID`, `target_character_id : UUID`. Sender must be leader.
  - `C2S_PARTY_LEADER_TRANSFER` (622): `operation_id : UUID`, `target_character_id : UUID`. Sender must be leader; target must be party member.
  - `S2C_PARTY_STATE` (607): `party_id : UUID`, `party_revision : uint64`, `leader_character_id : UUID`, `members : list of {character_id, display_name, class_id, level, online_state, zone_id}`.
  - `C2S_PARTY_BOARD_POST` (634): `operation_id : UUID`, `dungeon_id : string`, `desired_size : uint32 (2..5)`, `note : string (<=40 chars)`. Allowed only in Safe Anchor, stationary, 120s TTL, 30s repost cooldown.
  - `C2S_PARTY_BOARD_CANCEL` (635): `operation_id : UUID`.
  - `S2C_PARTY_BOARD_STATE` (636): `entries : list of {post_id, poster_character_id, display_name, class_id, level, dungeon_id, desired_size, expires_in_seconds}`.
- **Guild (608..610, 623..631, 637..649)**:
  - `C2S_GUILD_INVITE` (608): `operation_id : UUID`, `target_character_id : UUID`. Caller role >= OFFICER; target level >= 10, guildless, unblocked. 10-minute invite lifetime (canonical: `../03_systems/guild.md` §Invitations). Rate limit: social invite (10/60s).
  - `S2C_GUILD_INVITE` (609): `guild_id : UUID`, `guild_name : string`, `inviter_name : string`, `expires_in_seconds : uint32`.
  - `C2S_GUILD_ACCEPT` (610): `operation_id : UUID`, `guild_id : UUID`. Checks member capacity (30..60).
  - `C2S_GUILD_DECLINE` (623): `operation_id : UUID`, `guild_id : UUID`.
  - `C2S_GUILD_LEAVE` (624): `operation_id : UUID`. Leader cannot leave without transfer or disband.
  - `C2S_GUILD_KICK` (625): `operation_id : UUID`, `target_character_id : UUID`. Caller outranks target.
  - `C2S_GUILD_ROLE_UPDATE` (626): `operation_id : UUID`, `target_character_id : UUID`, `new_role : guild.role.vice_leader | guild.role.officer | guild.role.member`. Caller must be LEADER, or VICE_LEADER for MEMBER->OFFICER and OFFICER->MEMBER (`../03_systems/guild.md` § Permissions).
  - `C2S_GUILD_LEADER_TRANSFER` (627): `operation_id : UUID`, `target_character_id : UUID`. Caller must be LEADER.
  - `S2C_GUILD_STATE` (628): `guild_id : UUID`, `guild_revision : uint64`, `guild_name : string`, `role : string` (receiver's role), `level : uint32`, `members_count : uint32`, `max_members : uint32`, `motd : string`, `recruitment_mode : CLOSED | APPLICATIONS`, `members : list of {character_id, display_name, class_id, level, role, online_state : ONLINE | OFFLINE, last_online_at (rounded per guild.md § Presence UI; offline only)}`.
  - `C2S_GUILD_STORAGE_DEPOSIT` (629): `operation_id : UUID`, `item_instance_id : UUID`, `quantity : uint32` (stack units; full stack if 0), `section : COMMON | RESERVE`. Caller has DEPOSIT permission for the section; item UNBOUND in inventory. Rate limit: durable mutation.
  - `C2S_GUILD_STORAGE_WITHDRAW` (630): `operation_id : UUID`, `item_instance_id : UUID`, `quantity : uint32`, `section : COMMON | RESERVE` (RESERVE: Leader/Vice only). Caller has WITHDRAW permission; inventory has space; same-account and 72h membership checks (`guild_storage.md`). Rate limit: durable mutation.
  - `S2C_GUILD_STORAGE_STATE` (631): `guild_id : UUID`, `storage_revision : uint64`, `items : list of {item_instance_id, item_id, quantity, section, deposited_by, deposited_at}`.
  - `C2S_GUILD_CREATE` (637): `operation_id : UUID`, `guild_name : string`. Rules and cost: `guild.md`.
  - `C2S_GUILD_DISBAND` (638): `operation_id : UUID`. LEADER only; Disband preconditions in `guild.md`.
  - `C2S_GUILD_APPLY` (639): `operation_id : UUID`, `guild_id : UUID`. Guild in `APPLICATIONS` mode.
  - `C2S_GUILD_APPLICATION_DECIDE` (640): `operation_id : UUID`, `applicant_character_id : UUID`, `decision : ACCEPT | REJECT`.
  - `S2C_GUILD_APPLICATIONS` (641): `guild_id : UUID`, `applications : list of {character_id, display_name, class_id, level, applied_at}`.
  - `C2S_GUILD_MOTD_SET` (642): `operation_id : UUID`, `motd : string` (text rules `../06_data/text.md`).
  - `C2S_GUILD_LEADERSHIP_CLAIM` (643): `operation_id : UUID`. Inactivity takeover rules in `guild.md` § Leader Inactivity.
  - `C2S_GUILD_STORAGE_MOVE` (644): `operation_id : UUID`, `item_instance_id : UUID`, `to_section : COMMON | RESERVE`. Leader/Vice/Officer.
  - `C2S_GUILD_STORAGE_CLAIM_REQUEST` (645): `operation_id : UUID`, `item_instance_id : UUID`, `quantity : uint32`. Reserve claim (`guild_storage.md`).
  - `C2S_GUILD_STORAGE_CLAIM_DECIDE` (646): `operation_id : UUID`, `claim_id : UUID`, `decision : APPROVE | REJECT | CANCEL | DELIVER`. Actors: APPROVE/REJECT = LEADER/VICE_LEADER; CANCEL = requester, LEADER or VICE_LEADER; DELIVER = requester only (delivers the APPROVED item into the requester's inventory; `INVENTORY_FULL` keeps it APPROVED).
  - `S2C_GUILD_STORAGE_CLAIMS` (647): `guild_id : UUID`, `storage_revision : uint64`, `claims : list of {claim_id, requester_character_id, item_instance_id, quantity, state, expires_at}`.
  - `C2S_GUILD_BLESSING_VOTE` (648): `operation_id : UUID`, `cycle_id : string`, `blessing_id : string` (`guild_progression.md`).
  - `C2S_GUILD_SETTINGS_SET` (650): `operation_id : UUID`, `recruitment_mode : CLOSED | APPLICATIONS`. LEADER only (`guild.md` § Recruitment Mode). Switching to CLOSED leaves PENDING applications to be decided or expire.
  - `C2S_GUILD_INVITE_CANCEL` (651): `operation_id : UUID`, `target_character_id : UUID`. Original inviter, LEADER or VICE_LEADER; invite -> CANCELLED.
  - `C2S_GUILD_APPLICATION_CANCEL` (652): `operation_id : UUID`, `guild_id : UUID`. Applicant only; application -> CANCELLED.
  - `C2S_GUILD_CREATE` / rename validation errors: `GUILD_NAME_INVALID` (text rules `../06_data/text.md`), `GUILD_NAME_TAKEN` (`name_key` collision).
  - `S2C_GUILD_RESULT` (649): `operation_id : UUID`, `request_message_id : uint32`, `status : SUCCESS | ERROR`, `error_code` (domain list in `errors.md`). Typed result for every guild request above, 608..627 and 650..652.
- **Player Report (632..633)**:
  - `C2S_REPORT_PLAYER` (632): `operation_id : UUID`, `target_character_id : UUID`, `reason : SPAM | HARASSMENT | HATE_OR_ABUSE | CHEATING | SCAM | INAPPROPRIATE_NAME | OTHER`, `chat_message_id : UUID (optional)`, `reporter_notes : string (optional, <= 200 graphemes)`. Canonical reasons and limit (10 submissions / 24h per account): `../03_systems/social.md` § Reports.
  - `S2C_REPORT_PLAYER_RESULT` (633): `operation_id : UUID`, `status : RECEIVED | RATE_LIMITED | INVALID_TARGET`.
## PvP / Guild War

### PvP ID Reservation (800..899)
All PvP and Guild War message IDs must be registered here. A content spec MUST NOT use an ID in this sub-range without adding a row to this table.
```text
ID   Name                       Direction   Notes
800  C2S_SPARRING_REQUEST       C→S         casual duel challenge; see below
801  C2S_SPARRING_ACCEPT        C→S         acceptance of sparring challenge
802  C2S_RANKED_QUEUE_JOIN      C→S         join ranked/arena queue
803  C2S_RANKED_QUEUE_LEAVE     C→S         leave ranked/arena queue
804  S2C_RANKED_QUEUE_UPDATE    S→C         queue position / MATCHED / ACCEPTING
805  C2S_MATCH_READY            C→S         ready-check response
806  S2C_MATCH_STATE            S→C         match lifecycle state
807  C2S_MATCH_SURRENDER        C→S         surrender after ACTIVE (pvp.md)
808  C2S_GUILD_WAR_QUEUE_JOIN   C→S         guild-war queue registration
809  C2S_GUILD_WAR_QUEUE_LEAVE  C→S         cancel guild-war queue
810  S2C_GUILD_WAR_STATE        S→C         guild-war match/queue state
811  S2C_SPARRING_CHALLENGE     S→C         challenge delivered to target: challenger, expires_in_seconds
812  C2S_SPARRING_DECLINE       C→S         target declines; carries operation_id
813  S2C_SPARRING_RESULT        S→C         to both: ACCEPTED | DECLINED | EXPIRED | REJECTED(+error_code)
814  C2S_DUEL_CHALLENGE         C→S         challenge a character to pvp.mode.duel
815  S2C_DUEL_CHALLENGE         S→C         challenge delivered to target
816  C2S_DUEL_RESPOND           C→S         target accepts or declines
817  C2S_DUEL_CANCEL            C→S         challenger cancels a pending challenge
818  S2C_DUEL_RESULT            S→C         to both: ACCEPTED | DECLINED | CANCELLED | EXPIRED | REJECTED(+error_code)
819..899  (reserved, unassigned)
```

Queue and duel payloads (every C2S carries `operation_id`; domain outcomes never use `S2C_ERROR`):
```text
802 C2S_RANKED_QUEUE_JOIN     operation_id, pvp_mode_id : pvp.mode.ranked_duel | pvp.mode.five_element_arena,
                              with_party (bool; Arena only: the party leader queues the whole party atomically,
                              party size 1..5, every member eligible per pvp.md § Eligibility)
803 C2S_RANKED_QUEUE_LEAVE    operation_id, pvp_mode_id (a party member leaving removes the whole party entry)
804 S2C_RANKED_QUEUE_UPDATE   pvp_mode_id, state : QUEUED | MATCHED | ACCEPTING | LEFT, queued_seconds,
                              pvp_match_id (from MATCHED), ready_deadline_ms (ACCEPTING), error_code
808 C2S_GUILD_WAR_QUEUE_JOIN  operation_id, roster_character_ids : exactly 10 distinct UUIDs (guild_war.md § Roster);
                              LEADER or VICE_LEADER only
809 C2S_GUILD_WAR_QUEUE_LEAVE operation_id
810 S2C_GUILD_WAR_STATE       guild_id, state (guild_war.md lifecycle), roster_character_ids, guild_war_match_id,
                              ready_deadline_ms, error_code
814 C2S_DUEL_CHALLENGE        operation_id, target_character_id
815 S2C_DUEL_CHALLENGE        challenge_id, challenger_character_id, challenger_name, expires_in_seconds (60)
816 C2S_DUEL_RESPOND          operation_id, challenge_id, decision : ACCEPT | DECLINE
817 C2S_DUEL_CANCEL           operation_id, challenge_id
818 S2C_DUEL_RESULT           challenge_id, outcome : ACCEPTED | DECLINED | CANCELLED | EXPIRED | REJECTED, error_code
```
Duel (`../03_systems/pvp.md` § Duel, § Eligibility): both characters level >= 10, online, not queued/matched/dead/transferring, `social.md` direct-interaction gate; at most one pending outbound and one pending inbound challenge per character. `ACCEPTED` creates a `pvp.mode.duel` match that follows the normal match lifecycle through 806 (`S2C_MATCH_STATE`) and 807 (surrender); `S2C_TRANSFER_PREPARE` moves both to `map.pvp.duel_court`. Queue rejections: `LEVEL_TOO_LOW`, `STATE_CONFLICT` (already queued/matched, dead, transferring, forbidden instance), `COOLDOWN_ACTIVE` (queue sanction or ready-check miss cooldown), `PERMISSION_DENIED` (not party leader / not LEADER or VICE_LEADER), `TARGET_INVALID` (roster member ineligible; `error_code` names the first failing rule).

`C2S_SPARRING_REQUEST` is sent by either player standing on the Sparring Ring platform to issue a casual duel challenge to a target character in the same channel (`pvp.mode.sparring`, ADR-0023). `C2S_SPARRING_ACCEPT` is the target's acceptance response. Both messages carry `target_character_id` and a stable `operation_id`; the server validates platform eligibility, channel co-location, and the `social.md` direct-interaction gate before creating the sparring match. Challenge lifetime is `60s`. The target receives `S2C_SPARRING_CHALLENGE` (811) and answers with `C2S_SPARRING_ACCEPT` (801) or `C2S_SPARRING_DECLINE` (812); both parties receive `S2C_SPARRING_RESULT` (813) for accept, decline, expiry or a validation rejection. `S2C_ERROR` is never used for these domain outcomes.

## Error Result Shape
Rejected requests use either:
- typed operation result with domain error, or
- `S2C_ERROR` for connection/protocol/session failures.

Canonical error taxonomy is in `errors.md`.

## Delivery Classes
Even though launch WebSocket transport is ordered/reliable, application messages are classified:

```text
REPLACEABLE_STATE
DISCRETE_INTENT
AUTHORITATIVE_EVENT
DURABLE_RESULT
CONTROL
```

Default class by message kind (a message listed explicitly elsewhere in this file keeps that class):
```text
IDs 1..11 (session), S2C_ERROR, heartbeat          -> CONTROL
C2S_INPUT_STATE, S2C_*_STATE, snapshots, deltas    -> REPLACEABLE_STATE
C2S_MOVEMENT_EDGE and every other C2S message      -> DISCRETE_INTENT
S2C_*_RESULT                                       -> DURABLE_RESULT
other S2C events (combat events, invites, notices) -> AUTHORITATIVE_EVENT
```

Rules:
- REPLACEABLE_STATE may supersede an older unsent/unapplied state,
- DISCRETE_INTENT is validated individually,
- AUTHORITATIVE_EVENT is ordered by server sequence,
- DURABLE_RESULT cannot be silently dropped,
- CONTROL has connection/session priority.

This classification is preserved if transport changes later.

## Validation
Every inbound message validates:
- legal connection/session phase,
- current session epoch,
- known message ID,
- payload size/schema,
- per-message rate limit,
- entity/account/character ownership,
- current simulation ownership where relevant,
- gameplay state/preconditions,
- sequence/idempotency rules.

Malformed payloads never reach gameplay handlers as partially trusted data.

## Invariants
- stable numeric message IDs,
- one message has one declared direction/class,
- movement held-state (C2S_INPUT_STATE) may coalesce; movement edges (C2S_MOVEMENT_EDGE) do not; discrete actions do not,
- client_mono_ms is advisory; server clamps compensation to 80 ms; edges implying implausible timing are rejected,
- just_guard is detected server-side only; the client never sends a just_guard flag,
- every message ID in every range must be registered in this document (ID, direction, fields) before use; unregistered IDs are `MESSAGE_UNKNOWN`,
- every durable result carries `operation_id`, `status` and `error_code`; state pushes (`S2C_*_STATE`) are full snapshots,
- durable mutation carries stable operation identity,
- server results are authoritative,
- transport reliability does not bypass stale/duplicate validation,
- C2S_TRADE_INVITE, C2S_AUCTION_LIST, C2S_AUCTION_BUY, C2S_REWARD_CLAIM, and social invite messages carry per-operation sub-limits defined in rate_limits.md; sub-limit check is applied before the aggregate durable-mutations bucket.
