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
account_refund_consumed_events
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
deletion_requested_at      TIMESTAMPTZ NULL
erased_at                  TIMESTAMPTZ NULL      -- set by the erasure transaction; row purged 1 year later
credential_guard_until     TIMESTAMPTZ NULL      -- account-takeover rule (../07_security/anti_cheat.md): while > now,
                                                 --   credential changes require the old password
economy_review_flagged_at  TIMESTAMPTZ NULL      -- ECONOMY_REVIEW queue entry (../07_security/anti_cheat.md); not a ban,
                                                 --   blocks nothing; set NULL when the signal clears
created_at                 TIMESTAMPTZ NOT NULL
```
`PENDING_DELETION` = player requested deletion; login is allowed only to cancel it; erasure runs after 7 days (`../07_security/data_protection.md`). `deletion_requested_at TIMESTAMPTZ NULL` records the request. After erasure the row keeps only `account_id`, `status = 'TOMBSTONE_ERASED'`, `created_at`, `deletion_requested_at` and `erased_at` (no personal column exists on `accounts`); nothing references it by FK (§ Account Erasure), so the 1-year purge is a plain `DELETE`.

`TOMBSTONE_ACCOUNT_ID = '00000000-0000-0000-0000-000000000001'` is the reserved non-personal owner of anonymized characters and of financial records whose account link was severed. The baseline migration inserts it (`status = 'TOMBSTONE_ERASED'`, `created_at` = migration time); it can never log in, own a provider/password credential, or be erased or purged. `WORLD_OWNER_ID = '00000000-0000-0000-0000-000000000002'` is the reserved `operations.owner_id` for world-scoped operations; it is not an account row.

### account_login_history (ADR-0065)
Security signal for the account-takeover rule (`../07_security/anti_cheat.md`); Category D, 90-day rolling retention.
```text
account_id        UUID NOT NULL REFERENCES accounts(account_id)
observed_at       TIMESTAMPTZ NOT NULL
device_id_hash    BYTEA NOT NULL      -- SHA-256(ACCOUNT_SIGNAL_SALT || client-reported install ID)
ip_prefix16_hash  BYTEA NOT NULL      -- SHA-256(ACCOUNT_SIGNAL_SALT || IPv4 /16 or IPv6 /48 prefix)
is_new_origin     BOOLEAN NOT NULL    -- device and prefix pair unseen on this account in the previous 90 days
PRIMARY KEY (account_id, observed_at)
INDEX (observed_at)                   -- retention purge
```
One row per successful login (password or federated; refresh does not write). A password change or federated unlink checks for an `is_new_origin` row of the account within the last hour (takeover rule); a match revokes the other session families and sets `accounts.credential_guard_until = now + 24 h`.

`iap_refund_consumed_score` is **derived only** (ADR-0060): the count of `account_refund_consumed_events` rows for the account with `occurred_at >= now - 180 days`; it is never stored or incremented. The transaction that inserts a refund-consumed event evaluates the score including the new row and, when it is `>= 2`, transitions `status` to `SUSPENDED_PAYMENT_RECONCILIATION` in the same transaction.

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

### account_identities (ADR-0065)
```text
provider_id        VARCHAR(16) NOT NULL   -- CHECK IN ('apple','google','steam')
provider_subject   VARCHAR(255) NOT NULL  -- provider-issued stable subject (Apple sub, Google sub, SteamID64)
account_id         UUID NOT NULL REFERENCES accounts(account_id)
linked_at          TIMESTAMPTZ NOT NULL
PRIMARY KEY (provider_id, provider_subject)
UNIQUE (account_id, provider_id)          -- one link per provider per account
```

### Auth sessions (ADR-0065)
Access credentials (15 min), gameplay tickets (60 s) and resume credentials (10 min) are held only in the memory of the single world process (ADR-0052); a process restart invalidates them and clients use their refresh credential. Only refresh families and revocations persist:
```text
auth_session_families
  session_family_id   UUID PRIMARY KEY
  account_id          UUID NOT NULL REFERENCES accounts(account_id)
  provider_id         VARCHAR(16) NOT NULL   -- CHECK IN ('password','apple','google','steam')
  client_platform     VARCHAR(16) NOT NULL   -- CHECK IN ('WINDOWS','ANDROID')
  app_version         VARCHAR(32) NOT NULL
  device_model_class  VARCHAR(32) NULL
  created_at          TIMESTAMPTZ NOT NULL
  last_refreshed_at   TIMESTAMPTZ NOT NULL
  expires_at          TIMESTAMPTZ NOT NULL   -- last_refreshed_at + 30 days
  revoked_at          TIMESTAMPTZ NULL
  revoke_reason       VARCHAR(32) NULL       -- LOGOUT | REUSE_DETECTED | ACCOUNT_REVOKE | PASSWORD_CHANGE | TAKEOVER_RULE | ERASURE_REQUEST | ADMIN
  INDEX (account_id)
  INDEX (expires_at)

auth_refresh_credentials
  credential_hash     BYTEA PRIMARY KEY      -- SHA-256 of the opaque refresh token
  session_family_id   UUID NOT NULL REFERENCES auth_session_families(session_family_id) ON DELETE CASCADE
  generation          INTEGER NOT NULL       -- 1, 2, ... per family
  issued_at           TIMESTAMPTZ NOT NULL
  expires_at          TIMESTAMPTZ NOT NULL
  rotated_at          TIMESTAMPTZ NULL       -- set when superseded; presenting a rotated credential = reuse (auth.md)
  UNIQUE (session_family_id, generation)

auth_revocations
  revocation_id       UUID PRIMARY KEY
  scope               VARCHAR(16) NOT NULL   -- CHECK IN ('SESSION_FAMILY','ACCOUNT','PROVIDER_LINK')
  account_id          UUID NULL              -- no FK: must outlive nothing; purged with the account at erasure
  session_family_id   UUID NULL
  provider_id         VARCHAR(16) NULL
  not_before          TIMESTAMPTZ NOT NULL   -- credentials issued before this instant are invalid
  created_at          TIMESTAMPTZ NOT NULL
  expires_at          TIMESTAMPTZ NOT NULL   -- created_at + 30 days (longest credential TTL); purged after
  INDEX (account_id)
```
Families are purged 30 days after `revoked_at` or `expires_at` (Category C, `../07_security/personal_data_register.md`).

### account_refund_consumed_events schema
Immutable event ledger supporting the rolling 180-day refund window and audit reconciliation:
```text
event_id            UUID PRIMARY KEY
account_id          UUID NOT NULL REFERENCES accounts(account_id)
entitlement_id      UUID NOT NULL REFERENCES account_iap_entitlements(entitlement_id)
occurred_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()

INDEX (account_id, occurred_at)
```
The derived `iap_refund_consumed_score` is the count of events where `account_id = $1 AND occurred_at >= $now - INTERVAL '180 days'` (`$now` = Go UTC); no score column exists.

There is no gameplay item vault or `account_storage` container (ADR-0029). `../03_systems/account_storage.md` is the IAP entitlement panel.

### account_iap_entitlements schema
```text
entitlement_id       UUID PRIMARY KEY      -- one per purchase event
account_id           UUID NOT NULL REFERENCES accounts(account_id)
product_id           VARCHAR(64) NOT NULL  -- store catalog in monetization.md
entitlement_type     VARCHAR(24) NOT NULL  -- CHECK IN ('ONE_SHOT', 'ACCOUNT_SCOPED_ACCESS', 'DIRECT_ACCOUNT_COSMETIC')
grant_state          VARCHAR(20) NOT NULL  -- CHECK IN ('PENDING','GRANTED','REJECTED','REFUNDED','REFUNDED_CONSUMED')
reject_reason        VARCHAR(24) NULL      -- CHECK IN ('RECEIPT_INVALID','PRODUCT_MISMATCH','SEASON_TRACK_DUPLICATE')
platform             VARCHAR(16) NOT NULL  -- CHECK IN ('GOOGLE_PLAY','APP_STORE','STEAM')
platform_receipt     VARCHAR(512) NOT NULL UNIQUE  -- opaque verification token (Google purchaseToken, Apple
                                           --   transactionId, Steam orderid)
created_at           TIMESTAMPTZ NOT NULL  -- PENDING row creation (Steam: /steam/init; others: first /verify)
granted_at           TIMESTAMPTZ NULL      -- GRANTED transition
ended_at             TIMESTAMPTZ NULL      -- REJECTED / REFUNDED / REFUNDED_CONSUMED transition
season_number        INTEGER NULL          -- NOT NULL exactly when product_id is a season track
claim_deadline_at    TIMESTAMPTZ NULL      -- season track only = season end + 14 days

UNIQUE (entitlement_id, account_id, entitlement_type)  -- composite FK target for account_entitlement_claims
UNIQUE (account_id, season_number) WHERE season_number IS NOT NULL AND grant_state IN ('PENDING','GRANTED')
       AND account_id <> '00000000-0000-0000-0000-000000000001'  -- one live season track per real account per season
CHECK ((grant_state = 'REJECTED') = (reject_reason IS NOT NULL))
INDEX (grant_state, created_at) WHERE grant_state = 'PENDING'  -- Steam pending timeout sweep
```

`grant_state` transitions: `PENDING -> GRANTED | REJECTED | REFUNDED`; `GRANTED -> REFUNDED | REFUNDED_CONSUMED`. `REJECTED`, `REFUNDED` and `REFUNDED_CONSUMED` are terminal. `PENDING -> REFUNDED` is a provider refund of a purchase that was never granted (e.g. Google auto-refund of an unacknowledged purchase); it grants and revokes nothing and inserts no refund-consumed event. A Steam `PENDING` row older than 15 minutes is resolved by `ISteamMicroTxn/QueryTxn`: provider status approved/succeeded → finalize and `GRANTED`; failed/denied/unknown order → `REJECTED` (`RECEIPT_INVALID`); `Init` (never approved by the user) stays `PENDING` and is re-queried every 15 minutes until it becomes `REJECTED` (`RECEIPT_INVALID`) 24 h after `InitTxn`; `QueryTxn` status is the only authority, so a failed `FinalizeTxn` call alone never rejects (`../07_security/validation.md`). A second `/steam/init` for a season track while a `PENDING` row exists for that account and season returns the existing order. A definitive negative provider answer sets `REJECTED` (`RECEIPT_INVALID` or `PRODUCT_MISMATCH`), so a forged or failed receipt never holds the per-season unique slot. A valid season-track receipt for a season in which the account already holds a `PENDING`/`GRANTED` row is inserted as `REJECTED` with `SEASON_TRACK_DUPLICATE` (payment handling: `../03_systems/monetization.md`). Verification flow: `../07_security/validation.md` § IAP Receipt Verification.

The `entitlement_type` discriminator is mandatory and enforced at the DB layer by a `CHECK` constraint on `account_iap_entitlements`:
- `ONE_SHOT` — consumed by the first character to claim it; the one-shot claim rule in `../03_systems/account_storage.md` applies; no per-character claim rows are created.
- `ACCOUNT_SCOPED_ACCESS` — never consumed by a claim; the access entitlement remains GRANTED for its full validity period. Per-character reward claims are tracked in `account_entitlement_claims` (see below). The one-shot claim rule does NOT apply to this type.
- `DIRECT_ACCOUNT_COSMETIC` — store cosmetic purchases (`product.cosmetic.*`) granting account-wide wardrobe unlocks in `account_cosmetic_entitlements`. Equippable directly by all characters on the account. On refund: if every row of that entitlement has `first_equipped_at IS NULL` -> `REFUNDED`, deletes its rows; otherwise -> `REFUNDED_CONSUMED`, deletes its rows, un-equips on session sync, logs `IAP_REFUND_CONSUMED` audit event and inserts one `account_refund_consumed_events` row (the derived score then includes it).

### iap_notification_dedup / iap_provider_cursors (ADR-0065)
```text
iap_notification_dedup
  provider           VARCHAR(16) NOT NULL   -- CHECK IN ('APP_STORE','GOOGLE_PLAY','STEAM')
  notification_key   VARCHAR(160) NOT NULL  -- Apple notificationUUID | Google messageId | Steam "<orderid>:<status>"
  received_at        TIMESTAMPTZ NOT NULL
  processed_at       TIMESTAMPTZ NULL       -- NULL = inserted, processing not yet committed
  PRIMARY KEY (provider, notification_key)
  INDEX (received_at)                       -- 180-day purge

iap_provider_cursors
  provider           VARCHAR(16) PRIMARY KEY  -- 'STEAM' (GetReport `time` cursor)
  cursor_value       VARCHAR(64) NOT NULL
  updated_at         TIMESTAMPTZ NOT NULL
```
The dedup row is inserted in the same transaction that applies the notification, so a crash before commit re-processes it and a duplicate after commit is a no-op.

### account_cosmetic_entitlements schema (DIRECT_ACCOUNT_COSMETIC)
Stores account-wide wardrobe unlocks for direct store cosmetics (`cosmetic.iap.*`).
```text
account_id           UUID NOT NULL REFERENCES accounts(account_id)
cosmetic_id          TEXT NOT NULL -- references cosmetic.iap.* in cosmetic_catalog.md
entitlement_id       UUID NOT NULL REFERENCES account_iap_entitlements(entitlement_id)
granted_at           TIMESTAMPTZ NOT NULL
first_equipped_at    TIMESTAMPTZ NULL      -- set once, in the transaction of the first successful equip of this
                                           --   cosmetic by any character of the account while this row exists

PRIMARY KEY (account_id, cosmetic_id, entitlement_id)
FOREIGN KEY (account_id) REFERENCES accounts(account_id)
FOREIGN KEY (entitlement_id) REFERENCES account_iap_entitlements(entitlement_id)
```

An account owns a store cosmetic while **any** row for `(account_id, cosmetic_id)` exists. A bundle and a standalone purchase of the same cosmetic each keep their own row; refunding one purchase deletes only its rows, so the other purchase keeps the unlock (ADR-0053).

### character_cosmetic_entitlements (ADR-0060)
Character-scoped cosmetic ownership (`../03_systems/cosmetics.md` § Identity / Ownership): story, feat, atlas, chivalry, bond, PvP, Guild War, world-event and Guild Stone grants, redemptions, and season-track tier claims.
```text
character_id           UUID NOT NULL REFERENCES characters(character_id)
cosmetic_id            VARCHAR(64) NOT NULL
source_kind            VARCHAR(16) NOT NULL  -- CHECK IN ('PLAY', 'REDEMPTION', 'SEASON_TRACK')
source_entitlement_id  UUID NULL REFERENCES account_iap_entitlements(entitlement_id)
                       -- CHECK ((source_kind = 'SEASON_TRACK') = (source_entitlement_id IS NOT NULL))
source_ref             VARCHAR(64) NOT NULL  -- source_entitlement_id as text for SEASON_TRACK, else source_kind
grant_operation_id     UUID NOT NULL
granted_at             TIMESTAMPTZ NOT NULL

PRIMARY KEY (character_id, cosmetic_id, source_ref)
INDEX (source_entitlement_id) WHERE source_entitlement_id IS NOT NULL
```
A character owns a cosmetic while **any** row for `(character_id, cosmetic_id)` exists, or while the account owns it through `account_cosmetic_entitlements`. A repeated grant of the same `(character, cosmetic, source_ref)` is idempotent. A season-track refund deletes exactly the rows whose `source_entitlement_id` is the refunded entitlement on every character of the account (`../03_systems/account_storage.md`), so the same cosmetic granted by another purchase or by play stays owned.

### character_cosmetic_equips
```text
character_id   UUID NOT NULL REFERENCES characters(character_id)
slot           VARCHAR(32) NOT NULL  -- character slots of cosmetics.md § Categories (incl. GUILD_STONE_INSCRIPTION)
cosmetic_id    VARCHAR(64) NOT NULL
PRIMARY KEY (character_id, slot)
```
Equip validates ownership at commit time; a lost entitlement removes the row (fallback to default) at the next state sync.

### character_feats / character_feat_milestones
Declared for the baseline migration; columns are canonical in `../03_systems/cosmetics.md` § Persistence Schema.
```text
character_feats            PK (character_id, feat_id); character_id FK characters
character_feat_milestones  PK (character_id, feat_id, milestone_threshold); reward_operation_id UNIQUE
```

An `ACCOUNT_SCOPED_ACCESS` entitlement MUST NOT be stored or processed using the one-shot claim path. Enforcement uses a **denormalized discriminator column on the claim row backed by a composite foreign key** (see `account_entitlement_claims` below). A PostgreSQL `CHECK` constraint cannot reference another table's columns, so application-layer guards alone are insufficient; the composite FK mechanism enforces this invariant at commit time. The composite FKs and the `UNIQUE (entitlement_id, account_id, entitlement_type)` constraint on `account_iap_entitlements` are migration-versioned alongside these tables.

### account_entitlement_claims (ACCOUNT_SCOPED_ACCESS only)
Tracks which reward tiers each character has already claimed under a given `ACCOUNT_SCOPED_ACCESS` entitlement. This is the persistence layer for the composite idempotency key defined by `../03_systems/monetization.md`.

```text
account_entitlement_id   UUID NOT NULL         -- composite FK component (see below)
account_id               UUID NOT NULL         -- denormalized from account_iap_entitlements; composite FK component
entitlement_type         VARCHAR(24) NOT NULL  -- CHECK (entitlement_type = 'ACCOUNT_SCOPED_ACCESS')
character_id             UUID NOT NULL         -- composite FK component (see below)
reward_tier_id           VARCHAR(64) NOT NULL  -- reward tier within the season track
claimed_at               TIMESTAMPTZ NOT NULL
claim_operation_id       UUID NOT NULL

PRIMARY KEY (account_entitlement_id, character_id, reward_tier_id)

FOREIGN KEY (account_entitlement_id, account_id, entitlement_type)
    REFERENCES account_iap_entitlements(entitlement_id, account_id, entitlement_type)
    DEFERRABLE INITIALLY IMMEDIATE
    -- enforces: (a) entitlement belongs to the correct account;
    --           (b) entitlement_type = ACCOUNT_SCOPED_ACCESS
    --           (the CHECK on this row forces the FK to match only ACCOUNT_SCOPED_ACCESS rows in parent)

FOREIGN KEY (character_id, account_id)
    REFERENCES characters(character_id, account_id)
    DEFERRABLE INITIALLY IMMEDIATE
    -- enforces: character belongs to the same account as the entitlement
    -- requires UNIQUE (character_id, account_id) on characters; trivially unique since character_id
    --   is the PK, but must be declared as an explicit composite unique constraint in the migration
```
Both composite FKs are `DEFERRABLE INITIALLY IMMEDIATE` with no `ON UPDATE` action; only the erasure transaction runs `SET CONSTRAINTS ALL DEFERRED` and re-points parents and claim rows explicitly (§ Account Erasure).

The three-column composite FK `(account_entitlement_id, account_id, entitlement_type)` targets the explicit `UNIQUE (entitlement_id, account_id, entitlement_type)` constraint on `account_iap_entitlements`. Because this row carries `CHECK (entitlement_type = 'ACCOUNT_SCOPED_ACCESS')`, the FK can only reference parent rows where `entitlement_type = 'ACCOUNT_SCOPED_ACCESS'`, making it structurally impossible at the DB layer to insert a claim row against a `ONE_SHOT` entitlement. The `(character_id, account_id)` FK enforces that the claiming character belongs to the same account that owns the entitlement. Both constraints fire at commit time; application validation does not replace them.

The PRIMARY KEY `(account_entitlement_id, character_id, reward_tier_id)` is the DB-level enforcement of the composite idempotency key. A character cannot claim the same `reward_tier_id` twice; another character on the same account can insert its own row for the same `account_entitlement_id + reward_tier_id` pair.

Secrets store only verifier/hash material required by security specs.

# Character
Root includes character_id, account_id, display/normalized name, class_id, lifecycle, appearance (fixed class default at creation), `created_at`, and `updated_at`. Both timestamps are server-owned `timestamptz`; `updated_at` starts at creation and advances on every committed update to the `characters` row. Changes only to child/projection rows do not advance it. When migration `000003` is implemented, existing rows receive its transaction timestamp because their earlier edit times cannot be reconstructed.

Owned projections include progression, potential allocation, skills, currencies, progression flags, discoveries/first-clears, checkpoint, cosmetic selection, fishing UTC-date catch count (`world_rules.md` daily cap 50), and chivalry lifetime plus utc-day counters.

### character_currencies
```text
character_id   UUID NOT NULL REFERENCES characters(character_id)
currency_id    VARCHAR(32) NOT NULL   -- currency.common | currency.bound | currency.special
balance        BIGINT NOT NULL CHECK (balance >= 0)   -- caps: ../03_systems/economy.md § Balance Caps
revision       BIGINT NOT NULL DEFAULT 0
PRIMARY KEY (character_id, currency_id)
```
The baseline also creates `rate_limit_counters` and `auth_failure_backoff` with the schema in `../07_security/external_integrations.md` § 3.

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

Canonical location kinds are CHARACTER_INVENTORY, EQUIPPED, BEAST_EQUIPMENT_SLOT, GUILD_STORAGE and AUCTION_ESCROW. There is no trade escrow kind (ADR-0060): items offered in a direct trade stay in CHARACTER_INVENTORY until settlement.

The location row carries only fields legal for its kind, such as character/guild/listing reference, inventory slot, or loadout/slot. `GUILD_STORAGE` rows also carry `depositor_character_id` and `depositor_account_id` (ADR-0049 same-account check and item-transfer signal). No account-storage slot exists.


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
Persist owned Soul instances, Soul EXP/level, contract placements, Meridian/loadout configuration, Formation selection, the skill loadout and the inputs needed to reconstruct the build snapshot.

### character_souls
```text
soul_instance_id             UUID PRIMARY KEY
character_id                 UUID NOT NULL REFERENCES characters(character_id)
soul_id                      VARCHAR(64) NOT NULL
level                        SMALLINT NOT NULL      -- CHECK (level BETWEEN 1 AND 5)
current_soul_exp             INTEGER NOT NULL       -- CHECK (current_soul_exp >= 0)
contracted_item_instance_id  UUID NULL UNIQUE REFERENCES item_instances(item_instance_id)
                             -- NULL = in Collection; one equipment <= 1 soul
```
Per-loadout limits (3 souls, 1 BOSS soul, same `soul_id` once, 9 contracts total) are validated transactionally under the character lock (`../03_systems/soul_contracts.md`).

```text
character_soul_resonance PK (character_id, soul_id) plus memory_resonance_count int default 0, sheen_unlocked_at NULL
```
Vanity-only duplicate counter (`../03_systems/soul_contracts.md` § Memory Resonance); written in the duplicate-acquisition transaction.

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
character_beast_food_daily PK (character_id, utc_date) plus food_points_gained (0..20; shared by all beasts, spirit_beasts.md)
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
character_atlas(character_id, atlas_page_id, tier, seen_count, completed_at, reward_operation_id, acknowledged_at NULL)
atlas_milestones(character_id, milestone_id, completed_at)
```
Atlas rewards are idempotent per page tier and auto-settle at tier promotion; `acknowledged_at` is set once by `C2S_ATLAS_CLAIM` (504), which never grants (`../03_systems/atlas.md`).

# Reward Claims
Root reward_claims stores claim ID, owner character, `source_type` (enum canonical in `../03_systems/reward_claims.md` § Claim Creation, incl. `LEVEL_MILESTONE`), `source_reference`, reward_slot, state and timestamps.

Claim value is represented by typed child lines for item/currency/other explicitly supported types. Random choices are finalized before rows commit.

A pending item/equipment claim stores an immutable finalized item-creation line and **does not** reference a materialized `item_instance_id`. On successful claim, the transaction creates/merges the exact item value into CHARACTER_INVENTORY and marks the claim CLAIMED. REWARD_CLAIM is not an `item_locations` kind. See ADR-0012.

Currency-overflow aggregation has an append-only contribution ledger with UNIQUE:
~~~
source_reward_operation_id + owner_character_id + reward_slot
~~~

### Reward claim tables (ADR-0065)
```text
reward_claims
  reward_claim_id      UUID PRIMARY KEY
  owner_character_id   UUID NOT NULL REFERENCES characters(character_id)
  source_type          VARCHAR(32) NOT NULL   -- CHECK IN the reward_claims.md § Claim Creation enum
  source_reference     VARCHAR(160) NOT NULL
  reward_slot          VARCHAR(64) NOT NULL
  claim_kind           VARCHAR(16) NOT NULL   -- CHECK IN ('SINGLE','ITEM_CONSOLIDATED','CURRENCY_AGGREGATE')
  consolidation_key    VARCHAR(128) NULL      -- '<item_id>|<effective_binding>' or '<currency_id>'; NULL for SINGLE
  state                VARCHAR(16) NOT NULL   -- CHECK IN ('PENDING','CLAIMING','CLAIMED','EXPIRED')
  created_at           TIMESTAMPTZ NOT NULL
  updated_at           TIMESTAMPTZ NOT NULL
  claimed_at           TIMESTAMPTZ NULL
  claim_operation_id   UUID NULL
  revision             BIGINT NOT NULL
  CHECK ((claim_kind = 'SINGLE') = (consolidation_key IS NULL))
  UNIQUE (owner_character_id, claim_kind, consolidation_key) WHERE state = 'PENDING' AND consolidation_key IS NOT NULL
  INDEX (owner_character_id, state)

reward_claim_lines                           -- typed value; no opaque value blob
  reward_claim_id      UUID NOT NULL REFERENCES reward_claims(reward_claim_id)
  line_no              SMALLINT NOT NULL
  line_kind            VARCHAR(16) NOT NULL   -- CHECK IN ('ITEM','CURRENCY')
  item_id              VARCHAR(64) NULL
  quantity             INTEGER NULL           -- CHECK (quantity > 0) for ITEM
  effective_binding    VARCHAR(24) NULL
  item_state           JSONB NOT NULL DEFAULT '{}'::jsonb  -- immutable finalized rolls/enhancement/provenance,
                                              --   schema-versioned (ADR-0012); '{}' for plain stackables
  content_revision     BIGINT NULL
  currency_id          VARCHAR(32) NULL
  amount               BIGINT NULL            -- CHECK (amount > 0) for CURRENCY
  PRIMARY KEY (reward_claim_id, line_no)

reward_claim_contributions                   -- idempotency ledger for every claim creation or merge
  source_reward_operation_id  UUID NOT NULL
  owner_character_id          UUID NOT NULL
  reward_slot                 VARCHAR(64) NOT NULL
  reward_claim_id             UUID NOT NULL REFERENCES reward_claims(reward_claim_id)
  quantity_or_amount          BIGINT NOT NULL
  created_at                  TIMESTAMPTZ NOT NULL
  PRIMARY KEY (source_reward_operation_id, owner_character_id, reward_slot)
```
Every claim creation and every consolidation/aggregate merge inserts one contribution row in the same transaction; a duplicate key means the source was already credited and nothing changes.

Do not use one mutable opaque JSON reward blob as the only authoritative value representation.

# Auction / Trade
Auction uses auction_listings, auction_proceeds and item location AUCTION_ESCROW.

### auction_listings (ADR-0065)
```text
listing_id            UUID PRIMARY KEY
seller_character_id   UUID NOT NULL REFERENCES characters(character_id)
seller_account_id     UUID NOT NULL REFERENCES accounts(account_id)
item_instance_id      UUID NOT NULL REFERENCES item_instances(item_instance_id)
item_id               VARCHAR(64) NOT NULL   -- denormalized for search
quantity              INTEGER NOT NULL       -- CHECK (quantity > 0)
price_common          BIGINT NOT NULL        -- CHECK (price_common BETWEEN 100 AND 2000000000)
listing_fee_common    BIGINT NOT NULL        -- CHECK (listing_fee_common >= 10)
state                 VARCHAR(16) NOT NULL   -- CHECK IN ('ACTIVE','SOLD','CANCELLED','EXPIRED','RECLAIMED','MOVED_TO_CLAIM')
listed_at             TIMESTAMPTZ NOT NULL
expires_at            TIMESTAMPTZ NOT NULL   -- listed_at + 24 h
ended_at              TIMESTAMPTZ NULL       -- SOLD / CANCELLED / EXPIRED transition
buyer_character_id    UUID NULL REFERENCES characters(character_id)
revision              BIGINT NOT NULL
UNIQUE (item_instance_id) WHERE state IN ('ACTIVE','CANCELLED','EXPIRED')  -- asset still in AUCTION_ESCROW
INDEX (expires_at) WHERE state = 'ACTIVE'
INDEX (seller_character_id, state)
INDEX (item_id, price_common, listing_id) WHERE state = 'ACTIVE'            -- keyset search
```
`SETTLING` (`../03_systems/trading_auction.md`) exists only inside the purchase transaction that holds the listing row lock; it is never committed. `MOVED_TO_CLAIM` = unreclaimed asset moved to a Reward Claim after 7 days.

Purchase atomically changes listing state, buyer balance, item location, tax sink and seller proceeds.

Direct trade sessions (offers, revision, confirmations, `OPEN -> LOCKED -> COMMITTING`) are runtime state of the map-instance simulation that owns both participants; they are not persisted. Offered items stay in `CHARACTER_INVENTORY` with a session lock and are revalidated (owner, location, quantity, binding, not otherwise locked) inside the settlement transaction. Settlement is one atomic transaction that moves items/common, applies the fee and inserts `trade_settlement_records` + rollups; its `operation_id` is recorded in `operations`. A process restart drops every unsettled session; nothing needs recovery because no value moved before settlement.

## Economy Aggregation Fields (Anti-Cheat Support)
The behavioural anomaly signals defined in `../07_security/anti_cheat.md` (net 7-day common outflow per account; 30-day trade-partner concentration per character) require rolling-window aggregation over settled economy records. Every settled record in the two families below MUST carry the following fields to make those queries bounded:

### auction_proceeds (required fields)
```text
proceeds_id               UUID PRIMARY KEY
settled_at                TIMESTAMPTZ NOT NULL   -- indexed; see below
seller_character_id       UUID NOT NULL REFERENCES characters
seller_account_id         UUID NOT NULL REFERENCES accounts
buyer_character_id        UUID NOT NULL REFERENCES characters
buyer_account_id          UUID NOT NULL REFERENCES accounts
proceeds_amount           BIGINT NOT NULL        -- common credited to seller after tax; CHECK (> 0)
listing_id                UUID NOT NULL UNIQUE REFERENCES auction_listings (audit linkage)
state                     VARCHAR(16) NOT NULL   -- CHECK IN ('PENDING','CLAIMED')
claimed_at                TIMESTAMPTZ NULL
claim_operation_id        UUID NULL
INDEX (seller_character_id) WHERE state = 'PENDING'
```

### trade_settlement_records (required fields)
One row per atomic direct trade settlement:
```text
settlement_id                          UUID PRIMARY KEY
settled_at                             TIMESTAMPTZ NOT NULL  -- indexed; see below
initiator_character_id                 UUID NOT NULL REFERENCES characters
initiator_account_id                   UUID NOT NULL REFERENCES accounts
counterpart_character_id               UUID NOT NULL REFERENCES characters
counterpart_account_id                 UUID NOT NULL REFERENCES accounts
common_sent_by_initiator               BIGINT NOT NULL       -- CHECK (>= 0)
common_sent_by_counterpart             BIGINT NOT NULL       -- CHECK (>= 0)
trade_id                               UUID NOT NULL UNIQUE  -- wire trade_id of the runtime session (no FK; not persisted)
settlement_operation_id                UUID NOT NULL         -- operations key of the settlement
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

For the rolling-window aggregation queries at production scale, a **periodic economy aggregate rollup** is maintained alongside the raw records. The rollup table accumulates per-(account_id, UTC-day) and per-(character_id, UTC-day) net-flow totals updated at each settlement commit; this allows the 7-day and 30-day window signals to sum at most ~7 or ~30 daily rows per account/character rather than scanning raw settlement rows. The rollup is an optimisation surface; the raw records with the indexes above remain the authoritative source for their whole retention period (180 days, Category G in `../07_security/personal_data_register.md`; a `PENDING` proceeds row is never purged). Rollup rows share the same 180-day retention.

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

## Account Erasure & Anonymization Lifecycle (Law 91/2025/QH15, Decree 356/2025/ND-CP; ADR-0065)
Canonical erasure transaction; retention per category is canonical in `../07_security/personal_data_register.md`. It runs once per account after the 7-day `PENDING_DELETION` window (`../07_security/data_protection.md`), as one PostgreSQL transaction under operation family `account.erasure` (owner = the account) that begins with `SET CONSTRAINTS ALL DEFERRED`:
1. Lock the account row; require `status = 'PENDING_DELETION'` (a retry after commit finds `TOMBSTONE_ERASED` and returns the recorded outcome).
2. Delete personal rows: `account_password_credentials`, `account_identities`, `auth_session_families` (cascades `auth_refresh_credentials`), `auth_revocations`, `account_login_history`, `chat_messages` where `sender_account_id` is the account, `economy_account_daily_rollups` of the account, `friends` / `friend_requests` / `blocks` rows naming any of its characters, and `guild_blessing_votes` of the account.
3. Re-point to `TOMBSTONE_ACCOUNT_ID` (financial and relational history; no row is copied): `account_iap_entitlements.account_id`, `account_entitlement_claims.account_id`, `account_cosmetic_entitlements.account_id`, `account_refund_consumed_events.account_id`, `auction_listings.seller_account_id`, `auction_proceeds.seller_account_id` / `buyer_account_id`, `trade_settlement_records.initiator_account_id` / `counterpart_account_id`, `item_locations.depositor_account_id`, `characters.account_id`. The tombstone is excluded from the season-track unique index, so re-pointed season rows never collide.
4. Anonymize every re-pointed character: `name = 'Anonymized_' || replace(character_id::text, '-', '')`, `name_key = 'anonymized_' || replace(character_id::text, '-', '')` (full 32-hex UUID: collision-free; the `anonymized_` key prefix is reserved and rejected for player names by `text.md`), releasing the original `name_key`.
5. Detach each character from its guild: a non-leader membership is removed under the normal leave rules (storage claims cancelled); a `LEADER` with other members first transfers leadership to the member with the highest role, then the earliest `joined_at`, then the lowest `character_id`; a sole-member guild moves every `GUILD_STORAGE` item into Reward Claims of that character (`source_type = GUILD`) and becomes `DISBANDED`.
6. Cancel the characters' `ACTIVE` Auction listings (`CANCELLED`; assets follow the 7-day move to Reward Claims). Runtime direct-trade sessions and party membership were already ended when the deletion request revoked all sessions.
7. Set `accounts.status = 'TOMBSTONE_ERASED'`, `erased_at = now`; insert the `operations` row. Deferred FKs are checked at commit.
8. After commit, append `{account_id_hash, executed_at, operation_id}` to the erasure ledger (`../08_scale_ops/backup_recovery.md`; hash = SHA-256(`ERASURE_LEDGER_SALT` || account_id)).

Progression, quests, Reward Claims, items and settled economy records stay attached to the anonymized characters or the tombstone. `audit_events.subject_account_id` keeps the erased UUID without an FK (Category H). One year after `erased_at` the residual `accounts` row is deleted; no FK references it by then.

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
INDEX (relic_id) WHERE relic_active = true   -- world-wide "already active" check before an INSTANCED/seasonal spawn (ADR-0061)
```
`region_di_tich_markers` is written only for launch boss relics (`relic.boss.*`); seasonal relics have no region marker.
Lifecycle: launch boss relics are written on `DEFEATED -> COOLDOWN` (PUBLIC: the defeated copy's map/channel; INSTANCED: the instance's source field map in its recorded entry channel, ADR-0061); seasonal relics are written when their source completes (dungeon completion or INSTANCED boss defeat → the listed map in `../02_world/bosses.md`, entry channel recorded by the instance; monster kill → the kill's channel), and an already-active row with the same key is not refreshed; `relic_active` is set false on expiry or explicit despawn; row may be cleaned up after expiry. On server restart, all rows with `relic_active = true` and `expires_at > now()` are reloaded and the relic restored with remaining duration clamped to at least 1 second.

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
The marker is written transactionally with the `world_consequence_relics` upsert on each `DEFEATED` transition. **Lock order (every relic/marker transaction):** the `region_di_tich_markers` row is locked (upsert or `SELECT ... FOR UPDATE`) before any `world_consequence_relics` row; a seasonal relic transaction (no marker) locks only its relic row (`database.md` priority 18). `relic_active_in_region` is updated to false when the last active relic for this `(region_id, boss_id)` expires, subject to the following concurrent-update protocol:

**Concurrent expiry locking**: On every relic expiry, the handler first acquires a `SELECT ... FOR UPDATE` lock on the `region_di_tich_markers` row for `(region_id, boss_id)`, before touching the relic row. Within the same transaction, it then sets `world_consequence_relics.relic_active = false` for the expiring row and applies the marker conditional false-update only when a `NOT EXISTS` subquery over `world_consequence_relics`—joining on region membership via the content map-to-region mapping and filtered to `relic_active = true`—confirms no remaining active relics for this `(region_id, boss_id)`. The `SELECT FOR UPDATE` serializes concurrent expiry handlers: the second handler waits for the first to commit, then reads the correct committed state before evaluating the guard. Without this lock, two handlers expiring the last two relics in a region simultaneously can each observe the other's row as still `relic_active = true` under `READ COMMITTED` isolation (uncommitted updates are not visible), both conclude active relics remain, and both skip the false-update — leaving `relic_active_in_region` permanently stale. This write is `CHECKPOINT_DURABLE` per `save_rules.md`.

**Restart recovery**: On startup the server queries all `world_consequence_relics` rows with `relic_active = true` and `expires_at > now()`, restores each relic with `remaining_duration = expires_at - now()` (floor 1 second), resolves the running partition by `map_id + channel_id`, and reapplies the channel-wide buff to all connected characters in that partition. Players are not accepted into the partition until this recovery read completes. Load validity (ADR-0065): zero rows is a valid state (fresh launch or all relics expired); the load marks relic rows with `expires_at <= now` inactive and recomputes `relic_active_in_region` of every marker from the relic rows (markers left stale by a stop are repaired, not fatal); it fails closed only when a table is unreadable or a row references an unknown map, channel, relic or boss content ID. The read must finish within `WORLD_CONSEQUENCE_LOAD_TIMEOUT = 5 s`; a timeout or failure keeps the partition closed (`TEMPORARY_DEPENDENCY_FAILURE` to entry attempts) and the load is retried with backoff 1..30 s.

# Guild
Logical roots include `guilds` (incl. `recruitment_mode CLOSED | APPLICATIONS`), `guild_memberships`, `guild_invites`, `guild_applications`, `guild_progression`, `guild_ritual_cycles`, `guild_blessing_votes`, storage and storage requests.

### Guild tables (ADR-0065)
Rules: `../03_systems/guild.md`, `guild_progression.md`, `guild_storage.md`. Guild storage items use `item_locations` kind `GUILD_STORAGE` (section column `COMMON | RESERVE`).
```text
guilds
  guild_id              UUID PRIMARY KEY
  name                  VARCHAR(96) NOT NULL
  name_key              VARCHAR(256) NOT NULL UNIQUE   -- kept after DISBANDED (guild names are never reused)
  state                 VARCHAR(16) NOT NULL   -- CHECK IN ('ACTIVE','DISBANDING','DISBANDED')
  recruitment_mode      VARCHAR(16) NOT NULL   -- CHECK IN ('CLOSED','APPLICATIONS')
  leader_character_id   UUID NULL REFERENCES characters(character_id)   -- NOT NULL while ACTIVE (CHECK)
  motd                  VARCHAR(1024) NOT NULL DEFAULT ''
  guild_revision        BIGINT NOT NULL
  guild_storage_revision BIGINT NOT NULL
  created_at            TIMESTAMPTZ NOT NULL
  disbanded_at          TIMESTAMPTZ NULL

guild_memberships
  character_id          UUID PRIMARY KEY REFERENCES characters(character_id)  -- one current guild per character
  guild_id              UUID NOT NULL REFERENCES guilds(guild_id)
  role                  VARCHAR(16) NOT NULL   -- CHECK IN ('LEADER','VICE_LEADER','OFFICER','MEMBER')
  joined_at             TIMESTAMPTZ NOT NULL
  UNIQUE (guild_id) WHERE role = 'LEADER'
  INDEX (guild_id)

guild_member_contributions                     -- survives leave; rejoin resumes total
  guild_id, character_id  PRIMARY KEY; lifetime_contribution BIGINT NOT NULL; cycle_id VARCHAR(16) NOT NULL; cycle_contribution BIGINT NOT NULL

guild_invites
  invite_id UUID PRIMARY KEY; guild_id UUID NOT NULL; inviter_character_id UUID NOT NULL; target_character_id UUID NOT NULL
  state VARCHAR(16) NOT NULL  -- CHECK IN ('PENDING','ACCEPTED','DECLINED','CANCELLED','EXPIRED')
  created_at, expires_at (created_at + 10 min), resolved_at NULL
  UNIQUE (guild_id, target_character_id) WHERE state = 'PENDING'

guild_applications
  application_id UUID PRIMARY KEY; guild_id UUID NOT NULL; applicant_character_id UUID NOT NULL
  state VARCHAR(16) NOT NULL  -- CHECK IN ('PENDING','ACCEPTED','REJECTED','CANCELLED','EXPIRED')
  created_at, expires_at (created_at + 7 days), resolved_at NULL, resolver_character_id NULL
  UNIQUE (guild_id, applicant_character_id) WHERE state = 'PENDING'
  INDEX (applicant_character_id) WHERE state = 'PENDING'   -- max 5 PENDING per character (transactional check)

guild_progression
  guild_id PRIMARY KEY; guild_exp BIGINT NOT NULL; guild_level SMALLINT NOT NULL; ritual_streak INTEGER NOT NULL
  active_blessing_id VARCHAR(64) NULL; blessing_expires_at TIMESTAMPTZ NULL; revision BIGINT NOT NULL

guild_ritual_cycles
  guild_id, cycle_id (UTC Monday date 'YYYY-MM-DD')  PRIMARY KEY
  m_effective SMALLINT NOT NULL; required_points_per_element INTEGER NOT NULL
  points_kim, points_moc, points_thuy, points_hoa, points_tho INTEGER NOT NULL
  rotation_pointer VARCHAR(4) NOT NULL; completed_at NULL
  candidate_blessing_ids VARCHAR(64)[] NULL (3, ordered); vote_closes_at NULL; finalized_blessing_id NULL

guild_blessing_votes
  guild_id, cycle_id, account_id  PRIMARY KEY          -- one vote per account per draft
  character_id UUID NOT NULL; blessing_id VARCHAR(64) NOT NULL; voted_at TIMESTAMPTZ NOT NULL

guild_storage_claims
  claim_id UUID PRIMARY KEY; guild_id UUID NOT NULL; requester_character_id UUID NOT NULL
  item_instance_id UUID NOT NULL; quantity INTEGER NOT NULL
  state VARCHAR(16) NOT NULL  -- CHECK IN ('PENDING','APPROVED','REJECTED','CANCELLED','EXPIRED','COMPLETED')
  created_at, approved_at NULL, approver_character_id NULL, expires_at, resolved_at NULL
  INDEX (guild_id, state)

guild_storage_audit                              -- player-facing log, 180-day retention
  audit_id UUID PRIMARY KEY; guild_id; operation_id; actor_character_id; action VARCHAR(24); section VARCHAR(8)
  item_id; quantity; receiver_character_id NULL; before_quantity; after_quantity; occurred_at
  INDEX (guild_id, occurred_at)
```
FKs: every `guild_id` column references `guilds`, every character column references `characters`.

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
copy_map_id                      VARCHAR(64) NOT NULL   -- the channel copy the character contributed to
copy_channel_id                  SMALLINT NOT NULL
copy_defeated                    BOOLEAN NOT NULL DEFAULT FALSE  -- set true in the copy's DEFEATED transaction
eligible_until                   TIMESTAMPTZ NULL       -- chest despawn time; set with copy_defeated
claim_operation_id               UUID NULL              -- set when the chest or the fallback Reward Claim settles
PRIMARY KEY (character_id, public_boss_spawn_generation_id)
INDEX (public_boss_spawn_generation_id, copy_map_id, copy_channel_id)
CHECK (copy_defeated = (eligible_until IS NOT NULL))
```
Written when a character first meets the contribution threshold; character-scoped (ADR-0029). Settlement (`../02_world/bosses.md`):
- chest despawn: rows of that copy with `copy_defeated` and no `claim_operation_id` settle into Reward Claims (`source_type = BOSS_CHEST`);
- timeout despawn of an undefeated copy: that copy's rows are deleted (no kill, no reward);
- process start (before partitions accept players): every row with `copy_defeated` and no `claim_operation_id` settles into Reward Claims; every row with `copy_defeated = false` is deleted (the copy did not survive the restart).

### public_boss_schedules
```text
boss_id                          TEXT PRIMARY KEY          -- standalone PUBLIC boss content identity
state                            TEXT NOT NULL CHECK (state IN ('SCHEDULED','OPEN'))
public_boss_spawn_generation_id  UUID NULL                 -- set while OPEN, NULL while SCHEDULED
opened_at                        TIMESTAMPTZ NULL          -- set while OPEN
next_spawn_at                    TIMESTAMPTZ NULL          -- set while SCHEDULED
revision                         BIGINT NOT NULL
CHECK ((state = 'OPEN' AND public_boss_spawn_generation_id IS NOT NULL AND opened_at IS NOT NULL AND next_spawn_at IS NULL)
    OR (state = 'SCHEDULED' AND public_boss_spawn_generation_id IS NULL AND opened_at IS NULL AND next_spawn_at IS NOT NULL))
```
World-scoped PUBLIC generation lifecycle (`../02_world/bosses.md` § PUBLIC Generation Lifecycle, ADR-0061). Written only by Ephemeral Global in single-row `CHECKPOINT_DURABLE` transactions (never combined with another aggregate); revision-checked.

Constraints include one current guild membership per character, exactly one leader for an ACTIVE guild through transactional role rules, revision-checked permission/capacity, and one canonical location for shared items.

# Social / Party
Persist `friends`, `friend_requests`, `blocks`, chivalry_points lifetime and utc-day counter, and durable sanction/abandon data where owning PvP rules require it.

Party runtime membership may remain in memory unless an owning reconnect/dungeon rule explicitly requires a recovery record. Ordinary world parties are ephemeral-global runtime (`../04_architecture/service_boundaries.md`) and do not persist across full process restart.


# PvP / Guild War
Persist only durable rating/season/result/reward/sanction state. Do not persist every simulation tick/combat event into primary gameplay tables. Logical families: `pvp_ratings` (character + mode + season), `pvp_match_settlements`, `pvp_sanctions`, `guild_war_ratings` (guild + season), `guild_war_settlements`.

# Session / Routing
PostgreSQL may persist session/revocation metadata and transfer/handoff records needed for ambiguous ownership recovery.

Fast current connection routing remains runtime state; stale DB/cache state cannot resurrect old authority.

# Idempotency / Audit
operations stores durable dedupe/outcome data for retriable value mutations.

### operations (ADR-0065)
```text
operation_family     VARCHAR(48) NOT NULL   -- stable dotted family, e.g. inventory.mutate, auction.buy, sim.kill_settlement
owner_kind           VARCHAR(16) NOT NULL   -- CHECK IN ('ACCOUNT','CHARACTER','GUILD','WORLD')
owner_id             UUID NOT NULL          -- account/character/guild ID; WORLD_OWNER_ID for world-scoped jobs
operation_id         UUID NOT NULL
request_fingerprint  BYTEA NOT NULL         -- SHA-256 of the canonical invariant-relevant request fields
outcome              JSONB NOT NULL DEFAULT '{}'::jsonb  -- bounded, schema-versioned result reference (IDs, amounts)
created_at           TIMESTAMPTZ NOT NULL
completed_at         TIMESTAMPTZ NOT NULL
PRIMARY KEY (operation_family, owner_id, operation_id)
INDEX (completed_at)                        -- 180-day purge
```
A row is inserted last in the committing transaction (`database.md`), so only committed outcomes exist. Same key + same fingerprint → return/reconstruct `outcome`; same key + different fingerprint → `OPERATION_CONFLICT`. Per-owner scoping lets one simulation `operation_id` (`save_rules.md`) settle for many characters without collision and makes a client-chosen ID unable to collide with another owner's operation. `owner_id` has no FK (rows outlive nothing and are purged by age).

### audit_events (ADR-0065)
```text
audit_event_id        UUID PRIMARY KEY
occurred_at           TIMESTAMPTZ NOT NULL
actor_kind            VARCHAR(16) NOT NULL   -- CHECK IN ('PLAYER','OPERATOR','SYSTEM')
actor_id              UUID NULL              -- character_id (PLAYER) or operator_id (OPERATOR); no FK
subject_account_id    UUID NULL              -- no FK (kept after erasure, Category H)
subject_character_id  UUID NULL              -- no FK
action                VARCHAR(48) NOT NULL   -- e.g. IAP_REFUND_CONSUMED, ADMIN_GRANT, ACCOUNT_BAN, ACCOUNT_SECURITY_REVIEW
reason                VARCHAR(512) NULL
ticket_id             VARCHAR(64) NULL
operation_id          UUID NULL
payload               JSONB NOT NULL DEFAULT '{}'::jsonb  -- bounded before/after snapshot, schema-versioned
INDEX (subject_account_id, occurred_at)
INDEX (subject_character_id, occurred_at)
INDEX (occurred_at)                          -- 3-year purge
```

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
no trade escrow location; trade sessions are runtime-only and settle in one transaction
iap_refund_consumed_score is derived from account_refund_consumed_events (180 days); never stored
account_iap_entitlements.grant_state includes terminal REJECTED (with reject_reason); REJECTED rows never hold the per-season unique slot
character_cosmetic_entitlements PK (character_id, cosmetic_id, source_ref); ownership = any character row or account row
character_souls.contracted_item_instance_id UNIQUE; memory resonance persisted per (character_id, soul_id) in character_soul_resonance
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
account_entitlement_claims PRIMARY KEY (account_entitlement_id, character_id, reward_tier_id) enforces per-character claim idempotency
characters carries UNIQUE (character_id, account_id) declared explicitly in migration to support composite FK from account_entitlement_claims
world_consequence_relics PRIMARY KEY (map_id, channel_id, relic_id); region_di_tich_markers PRIMARY KEY (region_id, boss_id)
relic_active_in_region false-update requires SELECT FOR UPDATE on region_di_tich_markers row + NOT EXISTS guard evaluated under lock within the same transaction; prevents concurrent expiry stale-read
economy_account_daily_rollups PK (account_id, utc_day); economy_character_daily_rollups PK (character_id, utc_day); upsert rides settlement-level idempotency
current_exp = absolute cumulative total EXP; type = integer (32-bit); lifetime cap 702,100,000 fits int32; formula change requires migration per config.md
auction_proceeds and trade_settlement_records carry settled_at, both character_id and account_id for each side, and transferred amount
rolling-window aggregation queries (anti_cheat.md signals) are bounded by indexes on (account_id, settled_at) and (character_id, settled_at); daily rollup tables maintain ≤30-row window sums
operations PRIMARY KEY (operation_family, owner_id, operation_id); rows exist only for committed operations
TOMBSTONE_ACCOUNT_ID row is seeded by the baseline migration and excluded from the season-track unique index
erasure = one transaction (§ Account Erasure); anonymized name = 'Anonymized_' + 32-hex character_id; residual accounts row purged 1 year after erased_at
access credentials, gameplay tickets and resume credentials are process memory only; refresh families and revocations persist
relic/marker transactions lock region_di_tich_markers before world_consequence_relics
boss_chest_eligibility of a defeated copy settles into Reward Claims at chest despawn or process start; undefeated-copy rows are deleted
~~~
