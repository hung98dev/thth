# Data Model
status: LOCKED

## Scope
Defines canonical durable aggregate families, ownership relations, lifecycle boundaries and storage invariants. This is a logical relational model; one logical family may use multiple physical tables.

Decision: ../11_decisions/0011-postgresql-relational-persistence.md.
ID rules: ids.md.

# Clock Authority
The operational clock is Go UTC (`time.Now().UTC()`), set by the application before writing to PostgreSQL. `DEFAULT NOW()` and `NOW()` in SQL are used for non-critical defaults; server-set timestamps (passed as parameters from Go) are authoritative for any timestamp that participates in ordering, expiry, or consistency checks. ADR-0048 governs `characters.updated_at`.

# Aggregate Rules
A durable aggregate has a canonical UUID root where it represents an entity, explicit owner/scope, lifecycle/state, monotonic revision where concurrent edits require it, DB constraints for local invariants, and operation/audit identity for value-changing mutations.

Cross-aggregate changes are transactional only when one invariant requires atomicity.

# Account / Authentication
Logical families:
~~~
accounts
account_identities
account_password_credentials
auth_session_families
auth_refresh_credentials
auth_revocations
account_cosmetic_entitlements
account_iap_entitlements
~~~

Relations:
- account -> at most 3 characters,
- account -> many provider identities,
- provider_id + provider_subject is UNIQUE,
- account -> account-scoped IAP cosmetics (`account_cosmetic_entitlements`, equippable on any character of that account) and account IAP entitlement records (`account_iap_entitlements`).

### accounts schema
```text
account_id                 UUID PRIMARY KEY
status                     VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'
                           -- CHECK (status IN ('ACTIVE', 'SUSPENDED_PAYMENT_RECONCILIATION', 'BANNED', 'PENDING_DELETION', 'TOMBSTONE_ERASED'))
iap_refund_consumed_score  INTEGER NOT NULL DEFAULT 0
last_refund_consumed_at    TIMESTAMPTZ NULL
deletion_requested_at      TIMESTAMPTZ NULL
created_at                 TIMESTAMPTZ NOT NULL
```
`PENDING_DELETION` = player requested deletion; login is allowed only to cancel it; erasure runs after 7 days (`../07_security/data_protection.md`). `deletion_requested_at TIMESTAMPTZ NULL` records the request.

`TOMBSTONE_ACCOUNT_ID = '00000000-0000-0000-0000-000000000001'` is the reserved non-personal tombstone account owner for anonymized characters after data erasure.

When `iap_refund_consumed_score >= 2` within rolling 180 days (evaluated from `account_refund_consumed_events`), `status` automatically transitions to `SUSPENDED_PAYMENT_RECONCILIATION`.

### account_password_credentials schema (ADR-0051)
```text
account_id         UUID PRIMARY KEY REFERENCES accounts(account_id)
username_key       VARCHAR(20) NOT NULL UNIQUE
email              VARCHAR(254) NOT NULL
email_key          VARCHAR(254) NOT NULL UNIQUE
password_hash      TEXT NOT NULL          -- Argon2id PHC string
params_version     SMALLINT NOT NULL
created_at         TIMESTAMPTZ NOT NULL
updated_at         TIMESTAMPTZ NOT NULL
```
At most one row per account. Validation rules: `../07_security/auth.md` § Password Provider.

### account_refund_consumed_events schema
Immutable event ledger supporting the rolling 180-day refund window and audit reconciliation:
```text
event_id            UUID PRIMARY KEY
account_id          UUID NOT NULL REFERENCES accounts(account_id)
entitlement_id      UUID NOT NULL REFERENCES account_iap_entitlements(entitlement_id)
occurred_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()

INDEX (account_id, occurred_at)
```
`accounts.iap_refund_consumed_score` is the derived count of events where `account_id = $1 AND occurred_at >= NOW() - INTERVAL '180 days'`.

There is no gameplay item vault or `account_storage` container (ADR-0029). `../03_systems/account_storage.md` is the IAP entitlement panel.

### account_iap_entitlements schema
```text
entitlement_id       -- stable UUID; one per purchase event; primary key
account_id           -- FK -> accounts
product_id           -- references store catalog in monetization.md
entitlement_type     -- discriminator; CHECK (entitlement_type IN ('ONE_SHOT', 'ACCOUNT_SCOPED_ACCESS', 'DIRECT_ACCOUNT_COSMETIC'))
grant_state          -- PENDING | GRANTED | REFUNDED | REFUNDED_CONSUMED
platform_receipt     -- opaque platform verification token; UNIQUE across all accounts
granted_at           -- UTC timestamp of GRANTED transition
season_number        -- INT NULL; required (NOT NULL) when product_id is a season track, else NULL
claim_deadline_at    -- TIMESTAMPTZ NULL; season track only = season end + 14 days

UNIQUE (entitlement_id, account_id, entitlement_type)  -- composite FK target for account_entitlement_claims
UNIQUE (account_id, season_number) WHERE season_number IS NOT NULL AND grant_state IN ('PENDING','GRANTED')  -- one live season track per account per season
```

The `entitlement_type` discriminator is mandatory and enforced at the DB layer by a `CHECK` constraint on `account_iap_entitlements`:
- `ONE_SHOT` — consumed by the first character to claim it; the one-shot claim rule in `../03_systems/account_storage.md` applies; no per-character claim rows are created.
- `ACCOUNT_SCOPED_ACCESS` — never consumed by a claim; the access entitlement remains GRANTED for its full validity period. Per-character reward claims are tracked in `account_entitlement_claims` (see below). The one-shot claim rule does NOT apply to this type.
- `DIRECT_ACCOUNT_COSMETIC` — store cosmetic purchases (`product.cosmetic.*`) granting account-wide wardrobe unlocks in `account_cosmetic_entitlements`. Equippable directly by all characters on the account. On refund: if unequipped/not consumed -> `REFUNDED`, deletes row from `account_cosmetic_entitlements`; if active/used in session -> `REFUNDED_CONSUMED`, deletes row, un-equips on session sync, logs `IAP_REFUND_CONSUMED` audit event, and increments `accounts.iap_refund_consumed_score`.

### account_cosmetic_entitlements schema (DIRECT_ACCOUNT_COSMETIC)
Stores account-wide wardrobe unlocks for direct store cosmetics (`cosmetic.iap.*`).
```text
account_id           UUID NOT NULL REFERENCES accounts(account_id)
cosmetic_id          TEXT NOT NULL -- references cosmetic.iap.* in cosmetic_catalog.md
entitlement_id       UUID NOT NULL REFERENCES account_iap_entitlements(entitlement_id)
granted_at           TIMESTAMPTZ NOT NULL

PRIMARY KEY (account_id, cosmetic_id, entitlement_id)
FOREIGN KEY (account_id) REFERENCES accounts(account_id)
FOREIGN KEY (entitlement_id) REFERENCES account_iap_entitlements(entitlement_id)
```

An account owns a store cosmetic while **any** row for `(account_id, cosmetic_id)` exists. A bundle and a standalone purchase of the same cosmetic each keep their own row; refunding one purchase deletes only its rows, so the other purchase keeps the unlock (ADR-0053).

An `ACCOUNT_SCOPED_ACCESS` entitlement MUST NOT be stored or processed using the one-shot claim path. Enforcement uses a **denormalized discriminator column on the claim row backed by a composite foreign key** (see `account_entitlement_claims` below). A PostgreSQL `CHECK` constraint cannot reference another table's columns, so application-layer guards alone are insufficient; the composite FK mechanism enforces this invariant at commit time. The composite FKs and the `UNIQUE (entitlement_id, account_id, entitlement_type)` constraint on `account_iap_entitlements` are migration-versioned alongside these tables.

### account_entitlement_claims (ACCOUNT_SCOPED_ACCESS only)
Tracks which reward tiers each character has already claimed under a given `ACCOUNT_SCOPED_ACCESS` entitlement. This is the persistence layer for the composite idempotency key defined by `../03_systems/monetization.md`.

```text
account_entitlement_id   -- composite FK component (see below)
account_id               -- denormalized from account_iap_entitlements; composite FK component
entitlement_type         -- denormalized discriminator; CHECK (entitlement_type = 'ACCOUNT_SCOPED_ACCESS')
character_id             -- composite FK component (see below)
reward_tier_id           -- reward tier within the season track
claimed_at               -- UTC timestamp of successful claim

FOREIGN KEY (account_entitlement_id, account_id, entitlement_type)
    REFERENCES account_iap_entitlements(entitlement_id, account_id, entitlement_type)
    -- enforces: (a) entitlement belongs to the correct account;
    --           (b) entitlement_type = ACCOUNT_SCOPED_ACCESS
    --           (the CHECK on this row forces the FK to match only ACCOUNT_SCOPED_ACCESS rows in parent)

FOREIGN KEY (character_id, account_id)
    REFERENCES characters(character_id, account_id)
    -- enforces: character belongs to the same account as the entitlement
    -- requires UNIQUE (character_id, account_id) on characters; trivially unique since character_id
    --   is the PK, but must be declared as an explicit composite unique constraint in the migration

UNIQUE (account_entitlement_id, character_id, reward_tier_id)
```

The three-column composite FK `(account_entitlement_id, account_id, entitlement_type)` targets the explicit `UNIQUE (entitlement_id, account_id, entitlement_type)` constraint on `account_iap_entitlements`. Because this row carries `CHECK (entitlement_type = 'ACCOUNT_SCOPED_ACCESS')`, the FK can only reference parent rows where `entitlement_type = 'ACCOUNT_SCOPED_ACCESS'`, making it structurally impossible at the DB layer to insert a claim row against a `ONE_SHOT` entitlement. The `(character_id, account_id)` FK enforces that the claiming character belongs to the same account that owns the entitlement. Both constraints fire at commit time; application validation does not replace them.

The UNIQUE constraint on `(account_entitlement_id, character_id, reward_tier_id)` is the DB-level enforcement of the composite idempotency key. A character cannot claim the same `reward_tier_id` twice; another character on the same account can insert its own row for the same `account_entitlement_id + reward_tier_id` pair.

Secrets store only verifier/hash material required by security specs.

# Character
Root includes character_id, account_id, display/normalized name, class_id, lifecycle, appearance (fixed class default at creation), `created_at`, and `updated_at`. Both timestamps are server-owned `timestamptz`; `updated_at` starts at creation and advances on every committed update to the `characters` row. Changes only to child/projection rows do not advance it. When migration `000003` is implemented, existing rows receive its transaction timestamp because their earlier edit times cannot be reconstructed.

Owned projections include progression, potential allocation, skills, currencies, progression flags, discoveries/first-clears, checkpoint, cosmetic selection, fishing UTC-date catch count (`world_rules.md` daily cap 50), and chivalry lifetime plus utc-day counters.

### Character Progression Column Types
`current_exp` stores the character's **absolute cumulative total earned EXP** across all levels, not a per-level-segment residual. The character's current level is derived by finding the largest `L` in `[1..60]` such that `cumulative_exp_to_reach(L) <= current_exp`, where `cumulative_exp_to_reach(1) = 0` and `cumulative_exp_to_reach(L) = sum(exp_required(i) for i=1..L-1)` for `L in [2..60]`. Column type: `integer` (PostgreSQL signed 32-bit; maximum ~2,147,483,647). The lifetime cumulative cap at the maximum supported level 60 under `exp_required(L) = 10000 * L * L` is exactly 702,100,000, within the 32-bit range. If a future level cap increase raises the lifetime cumulative cap above 2,147,483,647, a migration to `bigint` is required before that cap activates. Any formula change that alters the interpretation of stored `current_exp` requires an explicit data migration per `config.md` § Versioning — see `migrations.md`.


Created characters are permanent; character deletion is not supported. Character display name and canonical `name_key` are separate from UUID identity and are computed only by `text.md`. The same display/key split applies to guild names.
# Inventory / Item Ownership
item_instances owns intrinsic persistent item state:
~~~
item_instance_id
item_id
quantity
effective_binding
generated/roll state
enhancement/equipment state where applicable
definition/content provenance needed for interpretation
created_at
~~~

Exactly one item_locations row exists for every live owned item instance.

Canonical location kinds are CHARACTER_INVENTORY, EQUIPPED, BEAST_EQUIPMENT_SLOT, GUILD_STORAGE, TRADE_ESCROW and AUCTION_ESCROW.

The location row carries only fields legal for its kind, such as character/guild/listing/trade reference, inventory slot, or loadout/slot. `GUILD_STORAGE` rows also carry `depositor_character_id` and `depositor_account_id` (ADR-0049 same-account check and item-transfer signal). No account-storage slot exists.


DB checks/FKs plus transactional validation enforce:
- one item instance -> one current context,
- binding never loosens,
- slot uniqueness inside a container,
- equipped item belongs to the same character,
- escrow item is not simultaneously in inventory.

Do not clone an item into independent inventory + escrow rows.

# Enhancement Pity
Per equipment instance targeting +13..+16, persist one record per `item_instance_id + target_level`: `pity_fail_count` (0..9); derive `pity_bonus_bp` (0..500) from that count. Reset only on success at that target. Pity state transfers with the item and is never an independently mutable balance. See `crafting.md`.

# Inventory / Loadouts
Logical roots:
~~~
character_inventories(character_id, capacity, revision)
character_loadouts(character_id, loadout_index, role, revision)
~~~

Slot occupancy is derived from/validated with item_locations. Exactly three equipment loadouts and canonical equipment slots follow system specs.


# Souls / Builds
Persist owned Soul instances, Soul EXP/level, contract placements, Meridian/loadout configuration, Formation selection and the inputs needed to reconstruct the build snapshot.

Static effects remain immutable content IDs; do not copy full static definitions into every player row.

# Derived Combat Stat Projection — New Stats (ADR-0037)
`LIFESTEAL`, `REFLECT`, `ABSORB`, `HEAL_REDUCTION`, and `HEALING_RECEIVED` are **derived stats only**. They appear in the authoritative derived character stat projection computed at runtime from equipment secondary rolls, set bonuses, Soul effects, Spirit Meridian resonances, and Formation bonuses.

**No new persisted row exists for these stats.** An implementer must not create a `character_lifesteal`, `character_reflect`, or equivalent table. The derivation uses the same inputs that already persist:
- `item_instances` (generated/roll state, set membership)
- `character_loadouts` / `item_locations` (EQUIPPED items)
- Soul contract placements
- Meridian configuration
- Formation selection

The projection is recomputed from those inputs whenever the build changes (equip/unequip, enhance, Soul contract, respec, Meridian/Formation change, PvP snapshot). The resulting values are transmitted to the client via `S2C_STATE_DELTA`; they are not stored in a dedicated persistence row.

`HEALING_RECEIVED` is promoted to a first-class base stat (default `1.00`, floor `0.00`) under ADR-0037. It consolidates all previous `target_healing_received_multiplier` writes. This stat is still derived-only: its value is computed from active modifiers and no standalone persisted column is created for it.


# Spirit Beasts (Linh Thú)
Persist owned Linh Thú, level, bond_points, active state, and 3 equipment slot references under ADR-0019 / ADR-0043. No `beast_instance_id` UUID. Linh Thú levels advance only by the material/common transition table; no EXP accumulator exists:
~~~
character_beasts PK (character_id, beast_id) plus level, bond_points, is_active, updated_at
beast_equipment_locations PK (character_id, beast_id, slot_id) plus item_instance_id
FK beast_equipment_locations -> character_beasts
~~~
Constraints enforce at most one active beast per character, beast level <= character level, exactly 3 valid slot IDs (`vong_co`, `ao_giap`, `linh_chau`), and one beast equipment item in at most one beast slot. Unequipped BEAST_EQUIPMENT lives in CHARACTER_INVENTORY; equipped uses BEAST_EQUIPMENT_SLOT.
# Quests / Progression
Persist active/completed quest state, daily window/choice keys, story progression, map discovery, first-clear flags, bonus book flags (`progression.book.potential.<level>`, `progression.book.skill.<level>`) and one-time reward operation references.

One-time completion/reward uses UNIQUE owner + stable identity so retry cannot regrant.

# Atlas
Persist atlas pages per character:
```
character_atlas(character_id, atlas_page_id, tier, seen_count, completed_at, reward_operation_id)
atlas_milestones(character_id, milestone_id, completed_at)
```
Atlas rewards are idempotent per page tier.

# Reward Claims
Root reward_claims stores claim ID, owner character, source, reward_slot, state and timestamps.

Claim value is represented by typed child lines for item/currency/other explicitly supported types. Random choices are finalized before rows commit.

A pending item/equipment claim stores an immutable finalized item-creation line and **does not** reference a materialized `item_instance_id`. On successful claim, the transaction creates/merges the exact item value into CHARACTER_INVENTORY and marks the claim CLAIMED. REWARD_CLAIM is not an `item_locations` kind. See ADR-0012.

Currency-overflow aggregation has an append-only contribution ledger with UNIQUE:
~~~
source_reward_operation_id + owner_character_id + reward_slot
~~~

Do not use one mutable opaque JSON reward blob as the only authoritative value representation.

# Auction / Trade
Auction uses auction_listings, auction_proceeds and item location AUCTION_ESCROW.

Purchase atomically changes listing state, buyer balance, item location, tax sink and seller proceeds.

Direct trade uses trade root/participants/offers plus TRADE_ESCROW custody when required. Final settlement is atomic.

## Economy Aggregation Fields (Anti-Cheat Support)
The behavioural anomaly signals defined in `../07_security/anti_cheat.md` (net 7-day common outflow per account; 30-day trade-partner concentration per character) require rolling-window aggregation over settled economy records. Every settled record in the two families below MUST carry the following fields to make those queries bounded:

### auction_proceeds (required fields)
```text
proceeds_id               -- UUID, primary key
settled_at                -- UTC timestamp of settlement (indexed; see below)
seller_character_id       -- FK -> characters
seller_account_id         -- FK -> accounts
buyer_character_id        -- FK -> characters
buyer_account_id          -- FK -> accounts
proceeds_amount           -- common credited to seller after tax
listing_id                -- FK -> auction_listings (audit linkage)
```

### trade_settlement_records (required fields)
One row per atomic direct trade settlement:
```text
settlement_id                          -- UUID, primary key
settled_at                             -- UTC timestamp of settlement (indexed; see below)
initiator_character_id                 -- FK -> characters
initiator_account_id                   -- FK -> accounts
counterpart_character_id               -- FK -> characters
counterpart_account_id                 -- FK -> accounts
common_sent_by_initiator               -- common transferred from initiator to counterpart
common_sent_by_counterpart             -- common transferred from counterpart to initiator
trade_id                               -- FK -> trade root (audit linkage)
```

### Indexing Requirements
Raw settled records must support bounded rolling-window lookups without full table scans. Required indexes:

```text
auction_proceeds:
  INDEX (seller_account_id, settled_at)     -- 7-day net outflow per account
  INDEX (buyer_account_id, settled_at)      -- 7-day net inflow per account

trade_settlement_records:
  INDEX (initiator_account_id, settled_at)  -- 7-day net outflow per account
  INDEX (counterpart_account_id, settled_at)
  INDEX (initiator_character_id, settled_at) -- 30-day partner concentration per character
  INDEX (counterpart_character_id, settled_at)
```

For the rolling-window aggregation queries at production scale, a **periodic economy aggregate rollup** is maintained alongside the raw records. The rollup table accumulates per-(account_id, UTC-day) and per-(character_id, UTC-day) net-flow totals updated at each settlement commit; this allows the 7-day and 30-day window signals to sum at most ~7 or ~30 daily rows per account/character rather than scanning raw settlement rows. The rollup is an optimisation surface; the raw records with the indexes above remain the authoritative source and must be kept.

### economy_account_daily_rollups
```text
account_id             -- FK -> accounts
utc_day                -- date (UTC calendar day; no time component)
common_outflow         -- total common leaving this account on this day
common_inflow          -- total common entering this account on this day
updated_at             -- timestamptz of last upsert within this day

PRIMARY KEY (account_id, utc_day)
```

### economy_character_daily_rollups
```text
character_id           -- FK -> characters
utc_day                -- date (UTC calendar day; no time component)
common_outflow         -- total common leaving this character on this day
common_inflow          -- total common entering this character on this day
trade_partner_volumes  -- jsonb map of counterpart character_id UUID -> total common sent to that partner on this day;
                       --   e.g. {"<counterpart_uuid>": 500000};
                       --   bounded (at most one direct trade per character pair per session);
                       --   schema-versioned; not the authoritative ownership source
item_partner_counts    -- jsonb map of source character_id -> item units received on this day via trade, Auction purchase,
                       --   or Guild Storage withdrawal of another character's deposit
updated_at             -- timestamptz of last upsert within this day

PRIMARY KEY (character_id, utc_day)
```

**Upsert semantics**: At every settlement commit, the applicable rollup rows are upserted in the same transaction as the settlement record using `INSERT ... ON CONFLICT (primary key) DO UPDATE SET common_outflow = common_outflow + EXCLUDED.common_outflow, common_inflow = common_inflow + EXCLUDED.common_inflow, updated_at = EXCLUDED.updated_at` (plus jsonb key-value merge/addition of `trade_partner_volumes` for character rollup). The operation-level deduplication in the `operations` table prevents a replayed settlement from re-executing its rollup delta; rollup upserts therefore ride settlement-level idempotency and do not require an independent idempotency key.

The 7-day net-outflow signal (`anti_cheat.md`) sums `common_outflow - common_inflow` over at most 7 rows of `economy_account_daily_rollups`. The 30-day trade-partner concentration signal (`anti_cheat.md`) aggregates `trade_partner_volumes` over at most 30 rows of `economy_character_daily_rollups`. Both queries are bounded to a fixed small row count regardless of trade volume.

## Account Erasure & Anonymization Lifecycle (Law 91/2025/QH15 & Decree 13/2023)
When an account erasure request is executed under `../07_security/data_protection.md`:
1. The requesting account's status is set to `'TOMBSTONE_ERASED'`.
2. All credentials are deleted: external OAuth identities in `account_identities` and the `account_password_credentials` row (username, email, password hash), releasing `username_key` and `email_key`.
3. Active authentication session tokens and refresh credentials are deleted.
4. All characters owned by the requesting account undergo link severance:
   - `characters.account_id` is updated to `TOMBSTONE_ACCOUNT_ID` (`00000000-0000-0000-0000-000000000001`).
   - `characters.name` is replaced with deterministic non-personal placeholder: `Anonymized_` + substring(character_id, 1, 8).
   - `characters.name_key` is updated accordingly, freeing the original display name for release to other players.
5. Active Auction House listings and in-flight direct trades are cancelled and escrow drained to avoid orphaned market exposure.
6. Progression, completed quests, and historical economy logs remain intact to preserve relational integrity and prevent world economy corruption.
# Guild Stone & Boss Relic
Guild Stone weekly display is derived from guild_war rating and progression tables (updated Monday 00:00 UTC), not a separate persisted currency.

## Boss Aftermath Relic — Durable Aggregate
Boss aftermath relics (Di Tich) are **persisted** in authoritative server storage and survive server restart. They are not runtime-only.

Two durable aggregate families cover the world consequence:

These two tables together constitute the **`WorldConsequence` aggregate** (`world_consequence` / `WorldConsequence`) named by `../04_architecture/backend.md` and `../04_architecture/service_boundaries.md`. The decomposition into two tables is the authoritative persistence model; the architecture docs describe the same aggregate from a service-ownership perspective.

### world_consequence_relics
One row per active relic instance, keyed on stable content + channel identity (`map_id + channel_id + relic_id`, ADR-0053 amendment of ADR-0040). A durable aggregate must never be keyed on a runtime instance identity (`map_instance_id` is a runtime identity that changes on each process restart and cannot be used to match durable rows back to their running partition after restart — see ADR-0040):
```
map_id                -- content identity of the map (stable across restarts)
channel_id            -- channel within the map (stable logical identity)
relic_id              -- content relic identity: relic.boss.<boss_id> (launch Di Tích) or relic.season.<i>.<key> (seasonal)
source_id             -- content identity that spawned it: boss_id, dungeon_id or monster_id
relic_active          -- boolean; false if expired or not yet started
buff_effect_id        -- active buff identity (buff.di_tich.<boss_id> or buff.di_tich.season)
expires_at            -- UTC timestamp when the relic naturally despawns (spawn_time + 60 min)

PRIMARY KEY (map_id, channel_id, relic_id)
```
`region_di_tich_markers` is written only for launch boss relics (`relic.boss.*`); seasonal relics have no region marker.
Lifecycle: launch boss relics are written on `DEFEATED -> COOLDOWN`; seasonal relics are written when their source completes (dungeon completion → the channel of the map in `../02_world/bosses.md` that the party entered the dungeon from; monster kill → the kill's channel), and an already-active row with the same key is not refreshed; `relic_active` is set false on expiry or explicit despawn; row may be cleaned up after expiry. On server restart, all rows with `relic_active = true` and `expires_at > now()` are reloaded and the relic restored with remaining duration clamped to at least 1 second.

### region_di_tich_markers
One row per `(region_id, boss_id)`, written and updated on every `DEFEATED` transition. This is the permanent social-proof marker displayed at the safe anchor to all characters in the region regardless of channel:
```
region_id                      -- owning region
boss_id                        -- content identity
last_defeated_utc              -- UTC timestamp of the most recent DEFEATED transition
last_defeated_participant_count -- participant count at the most recent DEFEATED transition
relic_active_in_region         -- true if any channel of this region currently has relic_active = true

PRIMARY KEY (region_id, boss_id)
```
The marker is written transactionally with the `world_consequence_relics` upsert on each `DEFEATED` transition. `relic_active_in_region` is updated to false when the last active relic for this `(region_id, boss_id)` expires, subject to the following concurrent-update protocol:

**Concurrent expiry locking**: On every relic expiry, the handler must acquire a `SELECT ... FOR UPDATE` lock on the `region_di_tich_markers` row for `(region_id, boss_id)` before evaluating whether any active relics remain. Within the same transaction, after setting `world_consequence_relics.relic_active = false` for the expiring row, the handler applies the conditional false-update only when a `NOT EXISTS` subquery over `world_consequence_relics`—joining on region membership via the content map-to-region mapping and filtered to `relic_active = true`—confirms no remaining active relics for this `(region_id, boss_id)`. The `SELECT FOR UPDATE` serializes concurrent expiry handlers: the second handler waits for the first to commit, then reads the correct committed state before evaluating the guard. Without this lock, two handlers expiring the last two relics in a region simultaneously can each observe the other's row as still `relic_active = true` under `READ COMMITTED` isolation (uncommitted updates are not visible), both conclude active relics remain, and both skip the false-update — leaving `relic_active_in_region` permanently stale. This write is `CHECKPOINT_DURABLE` per `save_rules.md`.

**Restart recovery**: On startup the server queries all `world_consequence_relics` rows with `relic_active = true` and `expires_at > now()`, restores each relic with `remaining_duration = expires_at - now()` (floor 1 second), resolves the running partition by `map_id + channel_id`, and reapplies the channel-wide buff to all connected characters in that partition. Players are not accepted into the partition until this recovery read completes.

# Guild
Logical roots include guilds, guild_memberships, invites, applications, progression, weekly ritual state, storage and storage requests.

### guild_stone_category_completions
```text
guild_id        UUID NOT NULL
category_id     TEXT NOT NULL   -- guild_stone.season.<season_region_index>.<region>
season_number   INT  NOT NULL   -- repeat cycles add another row
completed_at    TIMESTAMPTZ NOT NULL
PRIMARY KEY (guild_id, category_id, season_number)
```
Permanent seasonal Guild Stone inscriptions (`../03_systems/guild.md` § Guild Stone). Kept after guild disband for display history.

### boss_chest_eligibility
```text
character_id                     UUID NOT NULL REFERENCES characters(character_id)
public_boss_spawn_generation_id  UUID NOT NULL
boss_id                          TEXT NOT NULL
eligible_until                   TIMESTAMPTZ NOT NULL   -- chest despawn time
claim_operation_id               UUID NULL              -- set when the chest or the fallback Reward Claim settles
PRIMARY KEY (character_id, public_boss_spawn_generation_id)
```
Written when a character first meets the contribution threshold; character-scoped (ADR-0029). At despawn, rows without `claim_operation_id` settle into Reward Claims (`../02_world/bosses.md`).

Constraints include one current guild membership per character, exactly one leader for an ACTIVE guild through transactional role rules, revision-checked permission/capacity, and one canonical location for shared items.

# Social / Party
Persist friends, blocks, chivalry_points lifetime and utc-day counter, and durable sanction/abandon data where owning PvP rules require it.

Party runtime membership may remain in memory unless an owning reconnect/dungeon rule explicitly requires a recovery record. Ordinary world parties are ephemeral-global runtime (`../04_architecture/service_boundaries.md`) and do not persist across full process restart.


# PvP / Guild War
Persist only durable rating/season/result/reward/sanction state. Do not persist every simulation tick/combat event into primary gameplay tables.

# Session / Routing
PostgreSQL may persist session/revocation metadata and transfer/handoff records needed for ambiguous ownership recovery.

Fast current connection routing remains runtime state; stale DB/cache state cannot resurrect old authority.

# Idempotency / Audit
operations stores durable dedupe/outcome data for retriable value mutations.

### operators
Admin/GM identities (`../07_security/auth.md` § Operator). Not player accounts.
```text
operator_id            UUID PRIMARY KEY
login_key              VARCHAR(32) NOT NULL UNIQUE
password_hash          TEXT NOT NULL
totp_secret_encrypted  BYTEA NOT NULL
role                   VARCHAR(16) NOT NULL  -- SUPPORT | MODERATOR | ECONOMY | ADMIN
status                 VARCHAR(16) NOT NULL  -- ACTIVE | DISABLED
created_at, last_login_at TIMESTAMPTZ
```

### chat_messages
```text
message_id           UUID PRIMARY KEY
sender_account_id    UUID NOT NULL
sender_character_id  UUID NOT NULL
channel              VARCHAR(16) NOT NULL   -- WORLD | PARTY | GUILD | WHISPER | LOCAL
scope_id             TEXT NULL              -- party/guild/whisper target/map-channel key
content              TEXT NOT NULL
created_at           TIMESTAMPTZ NOT NULL
INDEX (sender_account_id, created_at)
INDEX (created_at)
```
Moderation log only (`../03_systems/social.md`); 90-day rolling retention; erasure deletes the account's rows within 15 days (`../07_security/data_protection.md`). Written asynchronously in bounded batches; chat delivery never waits for it.

audit_events is append-only from gameplay perspective and records reconstructable high-value/security/admin context. Audit is evidence, not the authoritative balance/item table.

# Content Revision Metadata
Persist active schema/content revision metadata and activation history. Owned instances keep only generated state/provenance needed to survive later content revisions.

# Deletion
Character deletion is disabled; character records are permanent. For other entity aggregates (e.g., cancelled listings, expired invitations), prefer lifecycle tombstone/terminal state where audit/history references an entity.

Physical deletion of non-character rows is allowed only after no escrow, pending reward/proceeds, membership/ownership orphan or required history remains.
# Invariants
~~~
account -> <=3 characters
character is permanent (no deletion)
character name_key is globally UNIQUE
durable roots use canonical UUIDs
one item instance -> one current location
currency balance has one canonical row per owner/currency scope
one-time rewards are unique/idempotent
escrow/value settlement is transactional
static definitions use immutable content IDs
audit/history does not replace primary truth
runtime combat state is not per-tick DB rows
boss aftermath relics (Di Tich) are persisted; world_consequence_relics keyed on map_id + channel_id + relic_id (stable content/channel identity, not map_instance_id)
restart recovery resolves running partition by map_id + channel_id; restores active relics with remaining duration clamped to >= 1 second
region_di_tich_markers is permanent; updated on every DEFEATED transition
world_consequence_relics + region_di_tich_markers together constitute the WorldConsequence aggregate named by architecture docs
LIFESTEAL/REFLECT/ABSORB/HEAL_REDUCTION/HEALING_RECEIVED are derived-only; no new persisted row
new stat derivation uses existing equipment/Soul/Meridian/Formation persistence inputs
account_iap_entitlements carries entitlement_type CHECK (entitlement_type IN ('ONE_SHOT', 'ACCOUNT_SCOPED_ACCESS', 'DIRECT_ACCOUNT_COSMETIC')) and UNIQUE (entitlement_id, account_id, entitlement_type)
account_cosmetic_entitlements PK (account_id, cosmetic_id, entitlement_id); ownership = any row exists; direct store cosmetics equippable by all characters on the account without inventory items
accounts.status CHECK (status IN ('ACTIVE', 'SUSPENDED_PAYMENT_RECONCILIATION', 'BANNED', 'PENDING_DELETION', 'TOMBSTONE_ERASED')); iap_refund_consumed_score >= 2 triggers automatic SUSPENDED_PAYMENT_RECONCILIATION
ACCOUNT_SCOPED_ACCESS entitlement is never consumed by a claim; one-shot claim path must not process it
account_entitlement_claims enforces entitlement_type = ACCOUNT_SCOPED_ACCESS via denormalized column + composite FK (account_entitlement_id, account_id, entitlement_type); same-account ownership via (character_id, account_id) FK; both fire at commit time; application validation does not replace them
account_entitlement_claims UNIQUE (account_entitlement_id, character_id, reward_tier_id) enforces per-character claim idempotency
characters carries UNIQUE (character_id, account_id) declared explicitly in migration to support composite FK from account_entitlement_claims
world_consequence_relics PRIMARY KEY (map_id, channel_id, relic_id); region_di_tich_markers PRIMARY KEY (region_id, boss_id)
relic_active_in_region false-update requires SELECT FOR UPDATE on region_di_tich_markers row + NOT EXISTS guard evaluated under lock within the same transaction; prevents concurrent expiry stale-read
economy_account_daily_rollups PK (account_id, utc_day); economy_character_daily_rollups PK (character_id, utc_day); upsert rides settlement-level idempotency
current_exp = absolute cumulative total EXP; type = integer (32-bit); lifetime cap 702,100,000 fits int32; formula change requires migration per config.md
auction_proceeds and trade_settlement_records carry settled_at, both character_id and account_id for each side, and transferred amount
rolling-window aggregation queries (anti_cheat.md signals) are bounded by indexes on (account_id, settled_at) and (character_id, settled_at); daily rollup tables maintain ≤30-row window sums
~~~
