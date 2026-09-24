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
```

`C2S_INTERACT` (103) is discrete and cannot be silently merged. Fields:
```text
interact_kind : TALK | PICKUP | CHEST | CAST | HOOK | KINDLE | COOK | BONFIRE_REST | QUEST_OBJECT | NPC_SERVICE
target_id     : string — target entity or interactive object ID (e.g. bonfire.<zone_id>, cooking_hearth.<zone_id>)
operation_id  : UUID
recipe_id     : string, optional — required when interact_kind == COOK; must match a hearth recipe from crafting_catalog.md
public_boss_spawn_generation_id : UUID, optional — required only for CHEST on a public-boss Gilded Chest; ignored otherwise
```
Fishing CAST/HOOK use `interact_kind`. `HOOK_WINDOW` accepts only `HOOK`. Kindling, hearth cook, and bonfire rest use `KINDLE` / `COOK` / `BONFIRE_REST`. When `interact_kind == COOK`, `recipe_id` determines the food crafted; `target_id` is the village hearth. When `interact_kind == BONFIRE_REST`, `target_id` is the active village bonfire; player must be within 6.0m and stationary to initiate rest.

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
```

`S2C_COMBAT_EVENT` remains ID 304 in Replication. Client sends skill/target intent. Server owns action acceptance, hit, damage, healing, shield, status, cooldown/resource result, death, and respawn. `S2C_COMBAT_EVENT` includes `just_guard_window`, `just_guard_triggered`, `just_guard_hint`, and `beast_passive2_success`. Slow-mo juice derives from success flags only. The client never sends a presentation flag.


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
```

- **C2S_NPC_SHOP_BUY (420)**: `operation_id : UUID`, `npc_id : string`, `offer_id : string`, `quantity : uint32 (>=1)`. Prices/limits: `../07_content/npc_shop_catalog.md`. Result 421: `operation_id`, `status`, `error_code` (`errors.md` domain list), `granted : list of {item_id, quantity}`.
- **C2S_COSMETIC_REDEEM (422)**: `operation_id : UUID`, `cosmetic_id : string`, `route : CURRENCY_SPECIAL | CURRENCY_COMMON | MATERIAL`. Rules: `../03_systems/cosmetics.md` (Material Redemption, Redemption Guardrails). Result 423: `operation_id`, `status`, `error_code`, `cosmetic_id`.
- **C2S_COSMETIC_EQUIP (424)**: `operation_id : UUID`, `slot : enum` (cosmetic slots in `cosmetics.md` § Categories), `cosmetic_id : string` (empty = unequip). Result 425: `operation_id`, `status`, `error_code`, `slot`, `cosmetic_id`.

### Spirit Beast Message Contract (410..417)
Spirit Beast operations use dedicated messages (410..417). Do not reuse 408. All requests carry a client-generated UUID `operation_id` for idempotency.

- **C2S_BEAST_SET_ACTIVE (410)**: `operation_id : UUID`, `beast_id : string` (target companion to summon/activate).
- **S2C_BEAST_SET_ACTIVE_RESULT (411)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `active_beast_id : string`, `error_code : NONE | BEAST_NOT_OWNED | IN_COMBAT`.
- **C2S_BEAST_EQUIP (412)**: `operation_id : UUID`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `item_instance_id : UUID` (piece in character inventory). Wire-to-data slot mapping: `COLLAR` → `vong_co`, `BARDING` → `ao_giap`, `ORB` → `linh_chau` (canonical slot IDs in `../06_data/data_model.md`).
- **S2C_BEAST_EQUIP_RESULT (413)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `equipped_item_instance_id : UUID`, `unequipped_item_instance_id : UUID (optional)`, `error_code : NONE | BEAST_NOT_OWNED | ITEM_NOT_FOUND | SLOT_MISMATCH | LEVEL_TOO_LOW`.
- **C2S_BEAST_UNEQUIP (414)**: `operation_id : UUID`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`.
- **S2C_BEAST_UNEQUIP_RESULT (415)**: `operation_id : UUID`, `status : SUCCESS | ERROR`, `beast_id : string`, `slot_id : enum COLLAR | BARDING | ORB`, `unequipped_item_instance_id : UUID`, `error_code : NONE | BEAST_NOT_OWNED | SLOT_EMPTY | INVENTORY_FULL`.
- **C2S_BEAST_FEED (416)**: `operation_id : UUID`, `beast_id : string`, `food_item_id : string` (`item.consumable.food.ca_bong_kho` +5, `ca_chep_nuong` +8, `tom_nuong` +10), `quantity : uint32` (default 1).
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
```

`C2S_ATLAS_CLAIM` (504): `operation_id`, `atlas_page_id`, `tier`. If the tier is already auto-settled, return the existing grant (idempotent).

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
650..699 (reserved, unassigned)
```

### Social / Guild Payload Contracts

- **Chat (600..601)**:
  - `C2S_CHAT_SEND`: `channel : WORLD | LOCAL | PARTY | GUILD | WHISPER`, `target_character_id : UUID (optional, required for WHISPER)`, `message_text : string (1..200 graphemes; validated per social.md)`. Rate limit: chat bucket (5/10s burst, 20/60s sustained).
  - `S2C_CHAT_MESSAGE`: `channel`, `sender_character_id : UUID`, `sender_name : string`, `message_text : string`, `timestamp : int64`.
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
  - `C2S_PARTY_INVITE` (602): `operation_id : UUID`, `target_character_id : UUID`. Sender must be leader; target online, partyless, unblocked. 60s TTL. Rate limit: social invite (10/60s).
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
  - `C2S_GUILD_ROLE_UPDATE` (626): `operation_id : UUID`, `target_character_id : UUID`, `new_role : guild.role.vice_leader | guild.role.officer | guild.role.member`. Caller must be LEADER (or VICE_LEADER for MEMBER->OFFICER).
  - `C2S_GUILD_LEADER_TRANSFER` (627): `operation_id : UUID`, `target_character_id : UUID`. Caller must be LEADER.
  - `S2C_GUILD_STATE` (628): `guild_id : UUID`, `guild_name : string`, `role : string`, `level : uint32`, `members_count : uint32`, `max_members : uint32`, `motd : string`.
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
  - `C2S_GUILD_STORAGE_CLAIM_DECIDE` (646): `operation_id : UUID`, `claim_id : UUID`, `decision : APPROVE | REJECT | CANCEL | DELIVER`.
  - `S2C_GUILD_STORAGE_CLAIMS` (647): `guild_id : UUID`, `storage_revision : uint64`, `claims : list of {claim_id, requester_character_id, item_instance_id, quantity, state, expires_at}`.
  - `C2S_GUILD_BLESSING_VOTE` (648): `operation_id : UUID`, `cycle_id : string`, `blessing_id : string` (`guild_progression.md`).
  - `S2C_GUILD_RESULT` (649): `operation_id : UUID`, `request_message_id : uint32`, `status : SUCCESS | ERROR`, `error_code` (domain list in `errors.md`). Typed result for every guild request above and 608..627.
- **Player Report (632..633)**:
  - `C2S_REPORT_PLAYER` (632): `operation_id : UUID`, `target_character_id : UUID`, `category : CHAT_ABUSE | BOT_AUTOMATION | EXPLOIT | HARASSMENT`, `evidence_chat_snippet : string (optional, <= 500 chars)`. Rate limit: 5 / 600s per character.
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
814..899  (reserved, unassigned)
```

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
- all message IDs in trade (700..729), auction (730..759), and PvP (800..899) sub-ranges must be registered in the reservation tables in this document before use,
- durable mutation carries stable operation identity,
- server results are authoritative,
- transport reliability does not bypass stale/duplicate validation,
- C2S_TRADE_INVITE, C2S_AUCTION_LIST, C2S_AUCTION_BUY, C2S_REWARD_CLAIM, and social invite messages carry per-operation sub-limits defined in rate_limits.md; sub-limit check is applied before the aggregate durable-mutations bucket.
