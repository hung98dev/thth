-- 000001_baseline_schema.up.sql
-- Launch baseline schema per docs/06_data/physical_schema_contract.md §3 and
-- data_model.md (ADR-0048/0053/0060/0061/0065/0070). Creates every table
-- declared in data_model.md plus rate_limit_counters / auth_failure_backoff
-- (../07_security/external_integrations.md § 3).
--
-- Migration declaration (migrations.md § Lock / Runtime Safety):
--   affected tables : all listed below are NEW (empty) — zero existing rows.
--   expected lock   : table creation only; no data rewrites, no hot-table locks.
--   runtime         : sub-second on an empty database.
--   index strategy  : PKs/UNIQUE/secondary indexes created inline (empty tables).
--   backfill        : none beyond the fixed TOMBSTONE_ACCOUNT_ID seed row.
--   rollback        : 000001_baseline_schema.down.sql drops all objects
--                     (local/staging only; production uses forward-fix).

-- ====================================================================
-- 1. accounts — durable account root (data_model.md § accounts schema)
-- ====================================================================
CREATE TABLE accounts (
    account_id                 UUID PRIMARY KEY,
    status                     VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'
                               CHECK (status IN ('ACTIVE',
                                                 'SUSPENDED_PAYMENT_RECONCILIATION',
                                                 'BANNED',
                                                 'PENDING_DELETION',
                                                 'TOMBSTONE_ERASED')),
    deletion_requested_at      TIMESTAMPTZ NULL,
    erased_at                  TIMESTAMPTZ NULL,
    credential_guard_until     TIMESTAMPTZ NULL,
    economy_review_flagged_at  TIMESTAMPTZ NULL,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- iap_refund_consumed_score is derived only (ADR-0060); no score column exists.

-- Reserved non-personal tombstone owner for anonymized characters and
-- severed financial rows (data_model.md § Account Erasure).
INSERT INTO accounts (account_id, status)
VALUES ('00000000-0000-0000-0000-000000000001', 'TOMBSTONE_ERASED');

-- ====================================================================
-- 2. Auth tables (data_model.md § Account; ADR-0051, ADR-0065)
-- ====================================================================
CREATE TABLE account_password_credentials (
    account_id      UUID PRIMARY KEY REFERENCES accounts(account_id)
                    ON DELETE RESTRICT,
    username_key    VARCHAR(20) NOT NULL UNIQUE,
    email           VARCHAR(254) NOT NULL,
    email_key       VARCHAR(254) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    params_version  SMALLINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE account_identities (
    provider_id       VARCHAR(16) NOT NULL
                      CHECK (provider_id IN ('apple', 'google', 'steam')),
    provider_subject  VARCHAR(255) NOT NULL,
    account_id        UUID NOT NULL REFERENCES accounts(account_id)
                      ON DELETE RESTRICT,
    linked_at         TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (provider_id, provider_subject),
    UNIQUE (account_id, provider_id)
);
CREATE INDEX account_identities_account_idx ON account_identities (account_id);

CREATE TABLE account_login_history (
    account_id       UUID NOT NULL REFERENCES accounts(account_id)
                     ON DELETE RESTRICT,
    observed_at      TIMESTAMPTZ NOT NULL,
    device_id_hash   BYTEA NOT NULL,
    ip_prefix16_hash BYTEA NOT NULL,
    is_new_origin    BOOLEAN NOT NULL,
    PRIMARY KEY (account_id, observed_at)
);
CREATE INDEX account_login_history_observed_idx
    ON account_login_history (observed_at);

-- ====================================================================
-- 3. Auth sessions (data_model.md § Auth sessions; ADR-0065)
-- ====================================================================
CREATE TABLE auth_session_families (
    session_family_id   UUID PRIMARY KEY,
    account_id          UUID NOT NULL REFERENCES accounts(account_id)
                        ON DELETE RESTRICT,
    provider_id         VARCHAR(16) NOT NULL
                        CHECK (provider_id IN ('password', 'apple', 'google', 'steam')),
    client_platform     VARCHAR(16) NOT NULL
                        CHECK (client_platform IN ('WINDOWS', 'ANDROID')),
    app_version         VARCHAR(32) NOT NULL,
    device_model_class  VARCHAR(32) NULL,
    device_id_hash      BYTEA NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL,
    last_refreshed_at   TIMESTAMPTZ NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ NULL,
    revoke_reason       VARCHAR(32) NULL
                        CHECK (revoke_reason IN ('LOGOUT', 'REUSE_DETECTED',
                                                 'ACCOUNT_REVOKE', 'PASSWORD_CHANGE',
                                                 'TAKEOVER_RULE', 'ERASURE_REQUEST',
                                                 'ADMIN'))
);
CREATE INDEX auth_session_families_account_idx
    ON auth_session_families (account_id);
CREATE INDEX auth_session_families_expires_idx
    ON auth_session_families (expires_at);

CREATE TABLE auth_refresh_credentials (
    credential_hash    BYTEA PRIMARY KEY,
    session_family_id  UUID NOT NULL REFERENCES auth_session_families(session_family_id)
                       ON DELETE CASCADE,
    generation         INTEGER NOT NULL,
    issued_at          TIMESTAMPTZ NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    first_presented_at TIMESTAMPTZ NULL,
    rotated_at         TIMESTAMPTZ NULL,
    UNIQUE (session_family_id, generation)
);

CREATE TABLE auth_revocations (
    revocation_id     UUID PRIMARY KEY,
    scope             VARCHAR(16) NOT NULL
                      CHECK (scope IN ('SESSION_FAMILY', 'ACCOUNT', 'PROVIDER_LINK')),
    account_id        UUID NULL,   -- no FK (contract §4.1)
    session_family_id UUID NULL,
    provider_id       VARCHAR(16) NULL,
    not_before        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL
);
CREATE INDEX auth_revocations_account_idx ON auth_revocations (account_id);

-- ====================================================================
-- 4. characters — player characters (data_model.md § Character)
-- ====================================================================
CREATE TABLE characters (
    character_id    UUID PRIMARY KEY,
    account_id      UUID NOT NULL REFERENCES accounts(account_id)
                    ON DELETE RESTRICT,
    name            VARCHAR(64) NOT NULL,
    name_key        VARCHAR(256) NOT NULL,
    class_id        VARCHAR(64) NOT NULL,
    lifecycle       VARCHAR(16) NOT NULL DEFAULT 'ACTIVE'
                    CHECK (lifecycle IN ('ACTIVE', 'OFFLINE')),
    level           INTEGER NOT NULL DEFAULT 1
                    CHECK (level >= 1 AND level <= 60),
    current_exp     INTEGER NOT NULL DEFAULT 0
                    CHECK (current_exp >= 0 AND current_exp <= 702100000),
    appearance      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT characters_name_key_unique UNIQUE (name_key),
    -- Explicit composite unique required as FK target by
    -- account_entitlement_claims (data_model.md § Invariants).
    CONSTRAINT characters_id_account_unique UNIQUE (character_id, account_id)
);
CREATE INDEX characters_account_idx ON characters (account_id);

-- ====================================================================
-- 5. Character owned projections with declared tables
-- ====================================================================
CREATE TABLE character_currencies (
    character_id UUID NOT NULL REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    currency_id  VARCHAR(32) NOT NULL
                 CHECK (currency_id IN ('currency.common', 'currency.bound',
                                        'currency.special')),
    balance      BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    revision     BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, currency_id)
);

CREATE TABLE character_inventories (
    character_id UUID PRIMARY KEY REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    capacity     INTEGER NOT NULL CHECK (capacity >= 60 AND capacity <= 120),
    revision     BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE character_loadouts (
    character_id   UUID NOT NULL REFERENCES characters(character_id)
                   ON DELETE RESTRICT,
    loadout_index  SMALLINT NOT NULL CHECK (loadout_index BETWEEN 1 AND 3),
    role           VARCHAR(8) NOT NULL CHECK (role IN ('ACTIVE', 'SUPPORT')),
    revision       BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, loadout_index)
);
CREATE UNIQUE INDEX character_loadouts_active_unique
    ON character_loadouts (character_id)
    WHERE role = 'ACTIVE';

CREATE TABLE character_chivalry (
    character_id    UUID PRIMARY KEY REFERENCES characters(character_id)
                    ON DELETE RESTRICT,
    lifetime_points BIGINT NOT NULL DEFAULT 0 CHECK (lifetime_points >= 0),
    day_utc         DATE NULL,
    day_points      SMALLINT NOT NULL DEFAULT 0
                    CHECK (day_points BETWEEN 0 AND 100),
    revision        BIGINT NOT NULL DEFAULT 0
);

-- ====================================================================
-- 6. item_instances / enhancement_pity (data_model.md § Inventory)
-- ====================================================================
CREATE TABLE item_instances (
    item_instance_id   UUID PRIMARY KEY,
    item_id            VARCHAR(64) NOT NULL,
    quantity           INTEGER NOT NULL
                       CHECK (quantity > 0 AND quantity <= 9999),
    effective_binding  VARCHAR(24) NOT NULL
                       CHECK (effective_binding IN ('UNBOUND', 'ACCOUNT_BOUND',
                                                    'CHARACTER_BOUND')),
    enhancement_level  INTEGER NOT NULL DEFAULT 0
                       CHECK (enhancement_level >= 0 AND enhancement_level <= 16),
    roll_state         JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_revision   BIGINT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per equipment instance targeting +13..+16 (data_model.md § Enhancement
-- Pity): one record per item_instance_id + target_level.
CREATE TABLE enhancement_pity (
    item_instance_id UUID NOT NULL REFERENCES item_instances(item_instance_id)
                     ON DELETE RESTRICT,
    target_level     SMALLINT NOT NULL CHECK (target_level BETWEEN 13 AND 16),
    pity_fail_count  SMALLINT NOT NULL DEFAULT 0
                     CHECK (pity_fail_count BETWEEN 0 AND 9),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_instance_id, target_level)
);

-- ====================================================================
-- 7. Spirit Beasts (data_model.md § Spirit Beasts; ADR-0019, ADR-0043)
-- ====================================================================
CREATE TABLE character_beasts (
    character_id UUID NOT NULL REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    beast_id     VARCHAR(64) NOT NULL,
    level        SMALLINT NOT NULL DEFAULT 1
                 CHECK (level >= 1 AND level <= 60),
    bond_points  INTEGER NOT NULL DEFAULT 0 CHECK (bond_points >= 0),
    is_active    BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (character_id, beast_id)
);
-- At most one active beast per character.
CREATE UNIQUE INDEX character_beasts_active_unique
    ON character_beasts (character_id)
    WHERE is_active;

CREATE TABLE character_beast_food_daily (
    character_id       UUID NOT NULL REFERENCES characters(character_id)
                       ON DELETE RESTRICT,
    utc_date           DATE NOT NULL,
    food_points_gained SMALLINT NOT NULL DEFAULT 0
                       CHECK (food_points_gained BETWEEN 0 AND 20),
    PRIMARY KEY (character_id, utc_date)
);

CREATE TABLE beast_equipment_locations (
    character_id     UUID NOT NULL,
    beast_id         VARCHAR(64) NOT NULL,
    slot_id          VARCHAR(16) NOT NULL
                     CHECK (slot_id IN ('vong_co', 'ao_giap', 'linh_chau')),
    item_instance_id UUID NOT NULL UNIQUE
                     REFERENCES item_instances(item_instance_id)
                     ON DELETE RESTRICT,
    PRIMARY KEY (character_id, beast_id, slot_id),
    FOREIGN KEY (character_id, beast_id)
        REFERENCES character_beasts(character_id, beast_id)
        ON DELETE RESTRICT
);

-- ====================================================================
-- 8. Souls (data_model.md § Souls / Builds; ADR-0060)
-- ====================================================================
CREATE TABLE character_souls (
    soul_instance_id             UUID PRIMARY KEY,
    character_id                 UUID NOT NULL REFERENCES characters(character_id)
                                 ON DELETE RESTRICT,
    soul_id                      VARCHAR(64) NOT NULL,
    level                        SMALLINT NOT NULL
                                 CHECK (level BETWEEN 1 AND 5),
    current_soul_exp             INTEGER NOT NULL
                                 CHECK (current_soul_exp >= 0),
    contracted_item_instance_id  UUID NULL UNIQUE
                                 REFERENCES item_instances(item_instance_id)
                                 ON DELETE RESTRICT
);

CREATE TABLE character_soul_resonance (
    character_id           UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    soul_id                VARCHAR(64) NOT NULL,
    memory_resonance_count INTEGER NOT NULL DEFAULT 0
                           CHECK (memory_resonance_count >= 0),
    sheen_unlocked_at      TIMESTAMPTZ NULL,
    PRIMARY KEY (character_id, soul_id)
);

-- ====================================================================
-- 9. IAP (data_model.md § account_iap_entitlements; ADR-0060, ADR-0065)
-- ====================================================================
CREATE TABLE account_iap_entitlements (
    entitlement_id     UUID PRIMARY KEY,
    account_id         UUID NOT NULL REFERENCES accounts(account_id)
                       ON DELETE RESTRICT,
    product_id         VARCHAR(64) NOT NULL,
    entitlement_type   VARCHAR(24) NOT NULL
                       CHECK (entitlement_type IN ('ONE_SHOT',
                                                   'ACCOUNT_SCOPED_ACCESS',
                                                   'DIRECT_ACCOUNT_COSMETIC')),
    grant_state        VARCHAR(20) NOT NULL
                       CHECK (grant_state IN ('PENDING', 'GRANTED', 'REJECTED',
                                              'REFUNDED', 'REFUNDED_CONSUMED')),
    reject_reason      VARCHAR(24) NULL
                       CHECK (reject_reason IN ('RECEIPT_INVALID',
                                                'PRODUCT_MISMATCH',
                                                'SEASON_TRACK_DUPLICATE')),
    platform           VARCHAR(16) NOT NULL
                       CHECK (platform IN ('GOOGLE_PLAY', 'APP_STORE', 'STEAM')),
    platform_receipt   VARCHAR(512) NOT NULL UNIQUE,
    created_at         TIMESTAMPTZ NOT NULL,
    granted_at         TIMESTAMPTZ NULL,
    ended_at           TIMESTAMPTZ NULL,
    season_number      INTEGER NULL,
    claim_deadline_at  TIMESTAMPTZ NULL,
    CONSTRAINT account_iap_entitlements_composite_unique
        UNIQUE (entitlement_id, account_id, entitlement_type),
    CHECK ((grant_state = 'REJECTED') = (reject_reason IS NOT NULL))
);
-- One live season track per real account per season; the tombstone account
-- accumulates re-pointed rows and is exempt.
CREATE UNIQUE INDEX account_iap_entitlements_season_unique
    ON account_iap_entitlements (account_id, season_number)
    WHERE season_number IS NOT NULL
      AND grant_state IN ('PENDING', 'GRANTED')
      AND account_id <> '00000000-0000-0000-0000-000000000001';
CREATE INDEX account_iap_entitlements_pending_idx
    ON account_iap_entitlements (grant_state, created_at)
    WHERE grant_state = 'PENDING';
CREATE INDEX account_iap_entitlements_account_idx
    ON account_iap_entitlements (account_id);

CREATE TABLE account_refund_consumed_events (
    event_id       UUID PRIMARY KEY,
    account_id     UUID NOT NULL REFERENCES accounts(account_id)
                   ON DELETE RESTRICT,
    entitlement_id UUID NOT NULL
                   REFERENCES account_iap_entitlements(entitlement_id)
                   ON DELETE RESTRICT,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX account_refund_consumed_events_window_idx
    ON account_refund_consumed_events (account_id, occurred_at);

CREATE TABLE iap_notification_dedup (
    provider         VARCHAR(16) NOT NULL
                     CHECK (provider IN ('APP_STORE', 'GOOGLE_PLAY', 'STEAM')),
    notification_key VARCHAR(160) NOT NULL,
    received_at      TIMESTAMPTZ NOT NULL,
    processed_at     TIMESTAMPTZ NULL,
    PRIMARY KEY (provider, notification_key)
);
CREATE INDEX iap_notification_dedup_received_idx
    ON iap_notification_dedup (received_at);

CREATE TABLE iap_provider_cursors (
    provider     VARCHAR(16) PRIMARY KEY,
    cursor_value VARCHAR(64) NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

-- ====================================================================
-- 10. Cosmetics (data_model.md; ADR-0053, ADR-0060)
-- ====================================================================
CREATE TABLE account_cosmetic_entitlements (
    account_id        UUID NOT NULL REFERENCES accounts(account_id)
                      ON DELETE RESTRICT,
    cosmetic_id       TEXT NOT NULL,
    entitlement_id    UUID NOT NULL
                      REFERENCES account_iap_entitlements(entitlement_id)
                      ON DELETE RESTRICT,
    granted_at        TIMESTAMPTZ NOT NULL,
    first_equipped_at TIMESTAMPTZ NULL,
    PRIMARY KEY (account_id, cosmetic_id, entitlement_id)
);

CREATE TABLE account_entitlement_claims (
    account_entitlement_id UUID NOT NULL,
    account_id             UUID NOT NULL,
    entitlement_type       VARCHAR(24) NOT NULL
                           CHECK (entitlement_type = 'ACCOUNT_SCOPED_ACCESS'),
    character_id           UUID NOT NULL,
    reward_tier_id         VARCHAR(64) NOT NULL,
    claimed_at             TIMESTAMPTZ NOT NULL,
    claim_operation_id     UUID NOT NULL,
    PRIMARY KEY (account_entitlement_id, character_id, reward_tier_id),
    FOREIGN KEY (account_entitlement_id, account_id, entitlement_type)
        REFERENCES account_iap_entitlements(entitlement_id, account_id,
                                            entitlement_type)
        DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (character_id, account_id)
        REFERENCES characters(character_id, account_id)
        DEFERRABLE INITIALLY IMMEDIATE
);

CREATE TABLE character_cosmetic_entitlements (
    character_id           UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    cosmetic_id            VARCHAR(64) NOT NULL,
    source_kind            VARCHAR(16) NOT NULL
                           CHECK (source_kind IN ('PLAY', 'REDEMPTION',
                                                  'SEASON_TRACK')),
    source_entitlement_id  UUID NULL
                           REFERENCES account_iap_entitlements(entitlement_id)
                           ON DELETE RESTRICT,
    source_ref             VARCHAR(64) NOT NULL,
    grant_operation_id     UUID NOT NULL,
    granted_at             TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (character_id, cosmetic_id, source_ref),
    CHECK ((source_kind = 'SEASON_TRACK') = (source_entitlement_id IS NOT NULL))
);
CREATE INDEX character_cosmetic_entitlements_source_idx
    ON character_cosmetic_entitlements (source_entitlement_id)
    WHERE source_entitlement_id IS NOT NULL;

CREATE TABLE character_cosmetic_equips (
    character_id UUID NOT NULL REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    slot         VARCHAR(32) NOT NULL
                 CHECK (slot IN ('title', 'title_glow', 'frame', 'nameplate',
                                 'appearance', 'weapon_trail', 'aura',
                                 'character_shrine', 'guild_stone_inscription')),
    cosmetic_id  VARCHAR(64) NOT NULL,
    PRIMARY KEY (character_id, slot)
);

-- ====================================================================
-- 11. Feats (cosmetics.md § Persistence Schema)
-- ====================================================================
CREATE TABLE character_feats (
    character_id   UUID NOT NULL REFERENCES characters(character_id)
                   ON DELETE RESTRICT,
    feat_id        VARCHAR(64) NOT NULL,
    counter_value  INTEGER NOT NULL DEFAULT 0 CHECK (counter_value >= 0),
    PRIMARY KEY (character_id, feat_id)
);

CREATE TABLE character_feat_milestones (
    character_id         UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    feat_id              VARCHAR(64) NOT NULL,
    milestone_threshold  INTEGER NOT NULL,
    completed_at         TIMESTAMPTZ NOT NULL,
    reward_operation_id  UUID NULL UNIQUE,
    PRIMARY KEY (character_id, feat_id, milestone_threshold)
);

-- ====================================================================
-- 12. Atlas (data_model.md § Atlas)
-- ====================================================================
CREATE TABLE character_atlas (
    character_id        UUID NOT NULL REFERENCES characters(character_id)
                        ON DELETE RESTRICT,
    atlas_page_id       VARCHAR(64) NOT NULL,
    tier                SMALLINT NOT NULL CHECK (tier BETWEEN 1 AND 3),
    seen_count          INTEGER NOT NULL DEFAULT 0 CHECK (seen_count >= 0),
    completed_at        TIMESTAMPTZ NULL,
    reward_operation_id UUID NULL UNIQUE,
    acknowledged_at     TIMESTAMPTZ NULL,
    PRIMARY KEY (character_id, atlas_page_id, tier)
);

CREATE TABLE atlas_milestones (
    character_id UUID NOT NULL REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    milestone_id VARCHAR(64) NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (character_id, milestone_id)
);

-- ====================================================================
-- 13. Reward Claims (data_model.md § Reward claim tables; ADR-0012,
--     ADR-0065, ADR-0070)
-- ====================================================================
CREATE TABLE reward_claims (
    reward_claim_id     UUID PRIMARY KEY,
    owner_character_id  UUID NOT NULL REFERENCES characters(character_id)
                        ON DELETE RESTRICT,
    source_type         VARCHAR(32) NOT NULL
                        CHECK (source_type IN ('MONSTER', 'BOSS', 'BOSS_CHEST',
                                               'DUNGEON', 'QUEST', 'WORLD_EVENT',
                                               'ATLAS', 'FEAT', 'LEVEL_MILESTONE',
                                               'PVP', 'GUILD_WAR', 'GUILD',
                                               'AUCTION_ESCROW_EXPIRY',
                                               'ADMIN_COMPENSATION')),
    source_reference    VARCHAR(160) NOT NULL,
    reward_slot         VARCHAR(64) NOT NULL,
    claim_kind          VARCHAR(16) NOT NULL
                        CHECK (claim_kind IN ('SINGLE', 'ITEM_CONSOLIDATED',
                                              'CURRENCY_AGGREGATE')),
    consolidation_key   VARCHAR(128) NULL,
    state               VARCHAR(16) NOT NULL
                        CHECK (state IN ('PENDING', 'CLAIMING', 'CLAIMED',
                                         'EXPIRED')),
    created_at          TIMESTAMPTZ NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL,
    claimed_at          TIMESTAMPTZ NULL,
    claim_operation_id  UUID NULL,
    revision            BIGINT NOT NULL,
    CHECK ((claim_kind = 'SINGLE') = (consolidation_key IS NULL))
);
CREATE UNIQUE INDEX reward_claims_pending_consolidation_unique
    ON reward_claims (owner_character_id, claim_kind, consolidation_key)
    WHERE state = 'PENDING' AND consolidation_key IS NOT NULL;
CREATE INDEX reward_claims_owner_state_idx
    ON reward_claims (owner_character_id, state);

CREATE TABLE reward_claim_lines (
    reward_claim_id   UUID NOT NULL REFERENCES reward_claims(reward_claim_id)
                      ON DELETE RESTRICT,
    line_no           SMALLINT NOT NULL,
    line_kind         VARCHAR(16) NOT NULL CHECK (line_kind IN ('ITEM', 'CURRENCY')),
    item_id           VARCHAR(64) NULL,
    quantity          INTEGER NULL,
    effective_binding VARCHAR(24) NULL,
    item_state        JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_revision  BIGINT NULL,
    currency_id       VARCHAR(32) NULL,
    amount            BIGINT NULL,
    PRIMARY KEY (reward_claim_id, line_no),
    CHECK ((line_kind = 'ITEM'
            AND item_id IS NOT NULL AND quantity > 0
            AND effective_binding IS NOT NULL
            AND content_revision IS NOT NULL
            AND currency_id IS NULL AND amount IS NULL)
        OR (line_kind = 'CURRENCY'
            AND currency_id IS NOT NULL AND amount > 0
            AND item_id IS NULL AND quantity IS NULL
            AND effective_binding IS NULL
            AND item_state = '{}'::jsonb))
);

CREATE TABLE reward_claim_contributions (
    source_reward_operation_id UUID NOT NULL,
    owner_character_id         UUID NOT NULL,
    reward_slot                VARCHAR(64) NOT NULL,
    reward_claim_id            UUID NOT NULL
                               REFERENCES reward_claims(reward_claim_id)
                               ON DELETE RESTRICT,
    quantity_or_amount         BIGINT NOT NULL,
    created_at                 TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (source_reward_operation_id, owner_character_id, reward_slot)
);

-- ====================================================================
-- 14. Guild (data_model.md § Guild tables; ADR-0065)
-- ====================================================================
CREATE TABLE guilds (
    guild_id               UUID PRIMARY KEY,
    name                   VARCHAR(96) NOT NULL,
    name_key               VARCHAR(256) NOT NULL UNIQUE,
    state                  VARCHAR(16) NOT NULL
                           CHECK (state IN ('ACTIVE', 'DISBANDING', 'DISBANDED')),
    recruitment_mode       VARCHAR(16) NOT NULL
                           CHECK (recruitment_mode IN ('CLOSED', 'APPLICATIONS')),
    leader_character_id    UUID NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    motd                   VARCHAR(1024) NOT NULL DEFAULT '',
    guild_revision         BIGINT NOT NULL,
    guild_storage_revision BIGINT NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL,
    disbanded_at           TIMESTAMPTZ NULL,
    CHECK (state <> 'ACTIVE' OR leader_character_id IS NOT NULL),
    CHECK ((state = 'DISBANDED') = (disbanded_at IS NOT NULL))
);

CREATE TABLE guild_memberships (
    character_id UUID PRIMARY KEY REFERENCES characters(character_id)
                 ON DELETE RESTRICT,
    guild_id     UUID NOT NULL REFERENCES guilds(guild_id)
                 ON DELETE RESTRICT,
    role         VARCHAR(16) NOT NULL
                 CHECK (role IN ('LEADER', 'VICE_LEADER', 'OFFICER', 'MEMBER')),
    joined_at    TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX guild_memberships_leader_unique
    ON guild_memberships (guild_id)
    WHERE role = 'LEADER';
CREATE INDEX guild_memberships_guild_idx ON guild_memberships (guild_id);

CREATE TABLE guild_member_contributions (
    guild_id              UUID NOT NULL REFERENCES guilds(guild_id)
                          ON DELETE RESTRICT,
    character_id          UUID NOT NULL REFERENCES characters(character_id)
                          ON DELETE RESTRICT,
    lifetime_contribution BIGINT NOT NULL DEFAULT 0
                          CHECK (lifetime_contribution >= 0),
    cycle_id              VARCHAR(16) NOT NULL,
    cycle_contribution    BIGINT NOT NULL DEFAULT 0
                          CHECK (cycle_contribution >= 0),
    PRIMARY KEY (guild_id, character_id)
);

CREATE TABLE guild_invites (
    invite_id            UUID PRIMARY KEY,
    guild_id             UUID NOT NULL REFERENCES guilds(guild_id)
                         ON DELETE RESTRICT,
    inviter_character_id UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    target_character_id  UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    state                VARCHAR(16) NOT NULL
                         CHECK (state IN ('PENDING', 'ACCEPTED', 'DECLINED',
                                          'CANCELLED', 'EXPIRED')),
    created_at           TIMESTAMPTZ NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    resolved_at          TIMESTAMPTZ NULL,
    CHECK (expires_at = created_at + INTERVAL '10 minutes')
);
CREATE UNIQUE INDEX guild_invites_pending_unique
    ON guild_invites (guild_id, target_character_id)
    WHERE state = 'PENDING';

CREATE TABLE guild_applications (
    application_id          UUID PRIMARY KEY,
    guild_id                UUID NOT NULL REFERENCES guilds(guild_id)
                            ON DELETE RESTRICT,
    applicant_character_id  UUID NOT NULL REFERENCES characters(character_id)
                            ON DELETE RESTRICT,
    state                   VARCHAR(16) NOT NULL
                            CHECK (state IN ('PENDING', 'ACCEPTED', 'REJECTED',
                                             'CANCELLED', 'EXPIRED')),
    created_at              TIMESTAMPTZ NOT NULL,
    expires_at              TIMESTAMPTZ NOT NULL,
    resolved_at             TIMESTAMPTZ NULL,
    resolver_character_id   UUID NULL REFERENCES characters(character_id)
                            ON DELETE RESTRICT,
    CHECK (expires_at = created_at + INTERVAL '7 days')
);
CREATE UNIQUE INDEX guild_applications_pending_unique
    ON guild_applications (guild_id, applicant_character_id)
    WHERE state = 'PENDING';
CREATE INDEX guild_applications_applicant_pending_idx
    ON guild_applications (applicant_character_id)
    WHERE state = 'PENDING';

CREATE TABLE guild_progression (
    guild_id             UUID PRIMARY KEY REFERENCES guilds(guild_id)
                         ON DELETE RESTRICT,
    guild_exp            BIGINT NOT NULL DEFAULT 0 CHECK (guild_exp >= 0),
    guild_level          SMALLINT NOT NULL CHECK (guild_level >= 1),
    ritual_streak        INTEGER NOT NULL DEFAULT 0 CHECK (ritual_streak >= 0),
    active_blessing_id   VARCHAR(64) NULL,
    blessing_expires_at  TIMESTAMPTZ NULL,
    revision             BIGINT NOT NULL DEFAULT 0,
    CHECK ((active_blessing_id IS NULL) = (blessing_expires_at IS NULL))
);

CREATE TABLE guild_ritual_cycles (
    guild_id                    UUID NOT NULL REFERENCES guilds(guild_id)
                                ON DELETE RESTRICT,
    cycle_id                    VARCHAR(16) NOT NULL,
    m_effective                 SMALLINT NOT NULL,
    required_points_per_element INTEGER NOT NULL,
    points_kim                  INTEGER NOT NULL DEFAULT 0
                                CHECK (points_kim >= 0),
    points_moc                  INTEGER NOT NULL DEFAULT 0
                                CHECK (points_moc >= 0),
    points_thuy                 INTEGER NOT NULL DEFAULT 0
                                CHECK (points_thuy >= 0),
    points_hoa                  INTEGER NOT NULL DEFAULT 0
                                CHECK (points_hoa >= 0),
    points_tho                  INTEGER NOT NULL DEFAULT 0
                                CHECK (points_tho >= 0),
    rotation_pointer            VARCHAR(4) NOT NULL
                                CHECK (rotation_pointer IN ('KIM', 'MOC',
                                                            'THUY', 'HOA',
                                                            'THO')),
    completed_at                TIMESTAMPTZ NULL,
    candidate_blessing_ids      VARCHAR(64)[] NULL
                                CHECK (candidate_blessing_ids IS NULL
                                       OR array_length(candidate_blessing_ids, 1) = 3),
    vote_closes_at              TIMESTAMPTZ NULL,
    finalized_blessing_id       VARCHAR(64) NULL,
    PRIMARY KEY (guild_id, cycle_id)
);

CREATE TABLE guild_blessing_votes (
    guild_id      UUID NOT NULL REFERENCES guilds(guild_id)
                  ON DELETE RESTRICT,
    cycle_id      VARCHAR(16) NOT NULL,
    account_id    UUID NOT NULL REFERENCES accounts(account_id)
                  ON DELETE RESTRICT,
    character_id  UUID NOT NULL REFERENCES characters(character_id)
                  ON DELETE RESTRICT,
    blessing_id   VARCHAR(64) NOT NULL,
    voted_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (guild_id, cycle_id, account_id),
    FOREIGN KEY (guild_id, cycle_id)
        REFERENCES guild_ritual_cycles(guild_id, cycle_id)
        ON DELETE RESTRICT
);

CREATE TABLE guild_storage_claims (
    claim_id                UUID PRIMARY KEY,
    guild_id                UUID NOT NULL REFERENCES guilds(guild_id)
                            ON DELETE RESTRICT,
    requester_character_id  UUID NOT NULL REFERENCES characters(character_id)
                            ON DELETE RESTRICT,
    item_instance_id        UUID NOT NULL
                            REFERENCES item_instances(item_instance_id)
                            ON DELETE RESTRICT,
    quantity                INTEGER NOT NULL CHECK (quantity > 0),
    state                   VARCHAR(16) NOT NULL
                            CHECK (state IN ('PENDING', 'APPROVED', 'REJECTED',
                                             'CANCELLED', 'EXPIRED',
                                             'COMPLETED')),
    created_at              TIMESTAMPTZ NOT NULL,
    approved_at             TIMESTAMPTZ NULL,
    approver_character_id   UUID NULL REFERENCES characters(character_id)
                            ON DELETE RESTRICT,
    expires_at              TIMESTAMPTZ NOT NULL,
    resolved_at             TIMESTAMPTZ NULL
);
CREATE INDEX guild_storage_claims_guild_state_idx
    ON guild_storage_claims (guild_id, state);

CREATE TABLE guild_storage_audit (
    audit_id               UUID PRIMARY KEY,
    guild_id               UUID NOT NULL REFERENCES guilds(guild_id)
                           ON DELETE RESTRICT,
    operation_id           UUID NOT NULL,
    actor_character_id     UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    action                 VARCHAR(24) NOT NULL
                           CHECK (action IN ('DEPOSIT', 'WITHDRAW', 'MOVE',
                                             'CLAIM_REQUEST', 'CLAIM_APPROVE',
                                             'CLAIM_REJECT', 'CLAIM_DELIVER',
                                             'CLAIM_CANCEL', 'CLAIM_EXPIRE')),
    section                VARCHAR(8) NOT NULL
                           CHECK (section IN ('COMMON', 'RESERVE')),
    item_id                VARCHAR(64) NOT NULL,
    quantity               INTEGER NOT NULL CHECK (quantity > 0),
    receiver_character_id  UUID NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    before_quantity        INTEGER NOT NULL CHECK (before_quantity >= 0),
    after_quantity         INTEGER NOT NULL CHECK (after_quantity >= 0),
    occurred_at            TIMESTAMPTZ NOT NULL
);
CREATE INDEX guild_storage_audit_guild_idx
    ON guild_storage_audit (guild_id, occurred_at);

CREATE TABLE guild_stone_category_completions (
    guild_id      UUID NOT NULL REFERENCES guilds(guild_id)
                  ON DELETE RESTRICT,
    category_id   TEXT NOT NULL,
    season_number INTEGER NOT NULL,
    completed_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (guild_id, category_id, season_number)
);

-- ====================================================================
-- 15. Auction / Trade (data_model.md § Auction; ADR-0060, ADR-0065)
-- ====================================================================
CREATE TABLE auction_listings (
    listing_id          UUID PRIMARY KEY,
    seller_character_id UUID NOT NULL REFERENCES characters(character_id)
                        ON DELETE RESTRICT,
    seller_account_id   UUID NOT NULL REFERENCES accounts(account_id)
                        ON DELETE RESTRICT,
    item_instance_id    UUID NOT NULL REFERENCES item_instances(item_instance_id)
                        ON DELETE RESTRICT,
    item_id             VARCHAR(64) NOT NULL,
    quantity            INTEGER NOT NULL CHECK (quantity > 0),
    price_common        BIGINT NOT NULL
                        CHECK (price_common BETWEEN 100 AND 2000000000),
    listing_fee_common  BIGINT NOT NULL CHECK (listing_fee_common >= 10),
    state               VARCHAR(16) NOT NULL
                        CHECK (state IN ('ACTIVE', 'SOLD', 'CANCELLED',
                                         'EXPIRED', 'RECLAIMED',
                                         'MOVED_TO_CLAIM')),
    listed_at           TIMESTAMPTZ NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    ended_at            TIMESTAMPTZ NULL,
    buyer_character_id  UUID NULL REFERENCES characters(character_id)
                        ON DELETE RESTRICT,
    revision            BIGINT NOT NULL,
    CHECK (expires_at = listed_at + INTERVAL '24 hours'),
    CHECK ((state = 'ACTIVE') = (ended_at IS NULL))
);
-- One escrow row per item while the asset is still in AUCTION_ESCROW.
CREATE UNIQUE INDEX auction_listings_item_escrow_unique
    ON auction_listings (item_instance_id)
    WHERE state IN ('ACTIVE', 'CANCELLED', 'EXPIRED');
CREATE INDEX auction_listings_active_expiry_idx
    ON auction_listings (expires_at)
    WHERE state = 'ACTIVE';
CREATE INDEX auction_listings_seller_state_idx
    ON auction_listings (seller_character_id, state);
CREATE INDEX auction_listings_active_search_idx
    ON auction_listings (item_id, price_common, listing_id)
    WHERE state = 'ACTIVE';

-- item_locations — one row per live owned item instance (data_model.md
-- § Inventory / Item Ownership). No TRADE_ESCROW kind exists (ADR-0060).
CREATE TABLE item_locations (
    item_instance_id       UUID PRIMARY KEY
                           REFERENCES item_instances(item_instance_id)
                           ON DELETE RESTRICT,
    location_kind          VARCHAR(24) NOT NULL
                           CHECK (location_kind IN ('CHARACTER_INVENTORY',
                                                    'EQUIPPED',
                                                    'BEAST_EQUIPMENT_SLOT',
                                                    'GUILD_STORAGE',
                                                    'AUCTION_ESCROW')),
    character_id           UUID NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    guild_id               UUID NULL REFERENCES guilds(guild_id)
                           ON DELETE RESTRICT,
    listing_id             UUID NULL REFERENCES auction_listings(listing_id)
                           ON DELETE RESTRICT,
    slot_index             INTEGER NULL,
    equip_slot             VARCHAR(16) NULL
                           CHECK (equip_slot IN ('weapon', 'head', 'body',
                                                 'hands', 'legs', 'feet',
                                                 'necklace', 'ring', 'costume',
                                                 'talisman', 'jade', 'seal',
                                                 'relic', 'charm')),
    loadout_index          SMALLINT NULL
                           CHECK (loadout_index BETWEEN 1 AND 3),
    section                VARCHAR(8) NULL
                           CHECK (section IN ('COMMON', 'RESERVE')),
    depositor_character_id UUID NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    depositor_account_id   UUID NULL REFERENCES accounts(account_id)
                           ON DELETE RESTRICT,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- The location row carries only fields legal for its kind.
    CHECK (slot_index IS NULL OR slot_index >= 0),
    CHECK (location_kind <> 'CHARACTER_INVENTORY' OR slot_index <= 119),
    CHECK (location_kind <> 'GUILD_STORAGE' OR slot_index <= 179),
    CHECK (
        (location_kind = 'CHARACTER_INVENTORY'
         AND character_id IS NOT NULL AND slot_index IS NOT NULL
         AND guild_id IS NULL AND listing_id IS NULL AND equip_slot IS NULL
         AND loadout_index IS NULL AND section IS NULL
         AND depositor_character_id IS NULL AND depositor_account_id IS NULL)
        OR (location_kind = 'EQUIPPED'
         AND character_id IS NOT NULL AND equip_slot IS NOT NULL
         AND loadout_index IS NOT NULL AND slot_index IS NULL
         AND guild_id IS NULL AND listing_id IS NULL AND section IS NULL
         AND depositor_character_id IS NULL AND depositor_account_id IS NULL)
        OR (location_kind = 'BEAST_EQUIPMENT_SLOT'
         AND character_id IS NOT NULL AND slot_index IS NULL
         AND equip_slot IS NULL AND loadout_index IS NULL
         AND guild_id IS NULL AND listing_id IS NULL AND section IS NULL
         AND depositor_character_id IS NULL AND depositor_account_id IS NULL)
        OR (location_kind = 'GUILD_STORAGE'
         AND guild_id IS NOT NULL AND section IS NOT NULL
         AND slot_index IS NOT NULL
         AND depositor_character_id IS NOT NULL
         AND depositor_account_id IS NOT NULL
         AND character_id IS NULL AND listing_id IS NULL
         AND equip_slot IS NULL AND loadout_index IS NULL)
        OR (location_kind = 'AUCTION_ESCROW'
         AND listing_id IS NOT NULL
         AND character_id IS NULL AND guild_id IS NULL
         AND slot_index IS NULL AND equip_slot IS NULL
         AND loadout_index IS NULL AND section IS NULL
         AND depositor_character_id IS NULL AND depositor_account_id IS NULL)
    )
);
-- Slot uniqueness inside each container.
CREATE UNIQUE INDEX item_locations_inventory_slot_unique
    ON item_locations (character_id, slot_index)
    WHERE location_kind = 'CHARACTER_INVENTORY';
CREATE UNIQUE INDEX item_locations_equip_slot_unique
    ON item_locations (character_id, loadout_index, equip_slot)
    WHERE location_kind = 'EQUIPPED';
CREATE UNIQUE INDEX item_locations_guild_slot_unique
    ON item_locations (guild_id, section, slot_index)
    WHERE location_kind = 'GUILD_STORAGE';
CREATE UNIQUE INDEX item_locations_escrow_listing_unique
    ON item_locations (listing_id)
    WHERE location_kind = 'AUCTION_ESCROW';
CREATE INDEX item_locations_character_idx
    ON item_locations (character_id)
    WHERE character_id IS NOT NULL;
CREATE INDEX item_locations_guild_idx
    ON item_locations (guild_id)
    WHERE guild_id IS NOT NULL;

CREATE TABLE auction_proceeds (
    proceeds_id          UUID PRIMARY KEY,
    settled_at           TIMESTAMPTZ NOT NULL,
    seller_character_id  UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    seller_account_id    UUID NOT NULL REFERENCES accounts(account_id)
                         ON DELETE RESTRICT,
    buyer_character_id   UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    buyer_account_id     UUID NOT NULL REFERENCES accounts(account_id)
                         ON DELETE RESTRICT,
    proceeds_amount      BIGINT NOT NULL CHECK (proceeds_amount > 0),
    listing_id           UUID NOT NULL UNIQUE
                         REFERENCES auction_listings(listing_id)
                         ON DELETE RESTRICT,
    state                VARCHAR(16) NOT NULL
                         CHECK (state IN ('PENDING', 'CLAIMED')),
    claimed_at           TIMESTAMPTZ NULL,
    claim_operation_id   UUID NULL
);
CREATE INDEX auction_proceeds_pending_seller_idx
    ON auction_proceeds (seller_character_id)
    WHERE state = 'PENDING';
CREATE INDEX auction_proceeds_seller_window_idx
    ON auction_proceeds (seller_account_id, settled_at);
CREATE INDEX auction_proceeds_buyer_window_idx
    ON auction_proceeds (buyer_account_id, settled_at);

CREATE TABLE trade_settlement_records (
    settlement_id                UUID PRIMARY KEY,
    settled_at                   TIMESTAMPTZ NOT NULL,
    initiator_character_id       UUID NOT NULL REFERENCES characters(character_id)
                                 ON DELETE RESTRICT,
    initiator_account_id         UUID NOT NULL REFERENCES accounts(account_id)
                                 ON DELETE RESTRICT,
    counterpart_character_id     UUID NOT NULL REFERENCES characters(character_id)
                                 ON DELETE RESTRICT,
    counterpart_account_id       UUID NOT NULL REFERENCES accounts(account_id)
                                 ON DELETE RESTRICT,
    common_sent_by_initiator     BIGINT NOT NULL
                                 CHECK (common_sent_by_initiator >= 0),
    common_sent_by_counterpart   BIGINT NOT NULL
                                 CHECK (common_sent_by_counterpart >= 0),
    trade_id                     UUID NOT NULL UNIQUE,
    settlement_operation_id      UUID NOT NULL
);
CREATE INDEX trade_settlement_records_initiator_account_idx
    ON trade_settlement_records (initiator_account_id, settled_at);
CREATE INDEX trade_settlement_records_counterpart_account_idx
    ON trade_settlement_records (counterpart_account_id, settled_at);
CREATE INDEX trade_settlement_records_initiator_character_idx
    ON trade_settlement_records (initiator_character_id, settled_at);
CREATE INDEX trade_settlement_records_counterpart_character_idx
    ON trade_settlement_records (counterpart_character_id, settled_at);

-- ====================================================================
-- 16. Social (data_model.md § Social / Party)
-- ====================================================================
CREATE TABLE friends (
    character_low_id     UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    character_high_id    UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    created_at           TIMESTAMPTZ NOT NULL,
    created_operation_id UUID NOT NULL,
    PRIMARY KEY (character_low_id, character_high_id),
    CHECK (character_low_id < character_high_id)
);
CREATE INDEX friends_high_idx ON friends (character_high_id);

CREATE TABLE friend_requests (
    friend_request_id      UUID PRIMARY KEY,
    requester_character_id UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    target_character_id    UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    state                  VARCHAR(16) NOT NULL
                           CHECK (state IN ('PENDING', 'ACCEPTED', 'DECLINED',
                                            'CANCELLED', 'EXPIRED')),
    created_at             TIMESTAMPTZ NOT NULL,
    expires_at             TIMESTAMPTZ NOT NULL,
    resolved_at            TIMESTAMPTZ NULL,
    create_operation_id    UUID NOT NULL,
    resolve_operation_id   UUID NULL,
    CHECK (requester_character_id <> target_character_id),
    CHECK (expires_at = created_at + INTERVAL '7 days'),
    CHECK ((state = 'PENDING') = (resolved_at IS NULL))
);
CREATE UNIQUE INDEX friend_requests_pending_pair_unique
    ON friend_requests (LEAST(requester_character_id, target_character_id),
                        GREATEST(requester_character_id, target_character_id))
    WHERE state = 'PENDING';
CREATE INDEX friend_requests_target_idx
    ON friend_requests (target_character_id, state);
CREATE INDEX friend_requests_requester_idx
    ON friend_requests (requester_character_id, state);
CREATE INDEX friend_requests_pending_expiry_idx
    ON friend_requests (expires_at)
    WHERE state = 'PENDING';

CREATE TABLE blocks (
    blocker_character_id UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    blocked_character_id UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    created_at           TIMESTAMPTZ NOT NULL,
    operation_id         UUID NOT NULL,
    PRIMARY KEY (blocker_character_id, blocked_character_id),
    CHECK (blocker_character_id <> blocked_character_id)
);
CREATE INDEX blocks_blocked_idx ON blocks (blocked_character_id);

-- ====================================================================
-- 17. PvP / Guild War (data_model.md § PvP / Guild War)
-- ====================================================================
CREATE TABLE pvp_ratings (
    character_id         UUID NOT NULL REFERENCES characters(character_id)
                         ON DELETE RESTRICT,
    pvp_mode_id          VARCHAR(48) NOT NULL
                         CHECK (pvp_mode_id IN ('pvp.mode.ranked_duel',
                                                'pvp.mode.five_element_arena')),
    season_id            INTEGER NOT NULL CHECK (season_id >= 0),
    pvp_mmr              INTEGER NOT NULL,
    season_rating        INTEGER NOT NULL CHECK (season_rating >= 0),
    ranked_games_played  INTEGER NOT NULL DEFAULT 0
                         CHECK (ranked_games_played >= 0),
    lifetime_mode_games  INTEGER NOT NULL DEFAULT 0
                         CHECK (lifetime_mode_games >= 0),
    wins                 INTEGER NOT NULL DEFAULT 0 CHECK (wins >= 0),
    losses               INTEGER NOT NULL DEFAULT 0 CHECK (losses >= 0),
    draws                INTEGER NOT NULL DEFAULT 0 CHECK (draws >= 0),
    updated_at           TIMESTAMPTZ NOT NULL,
    revision             BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, pvp_mode_id, season_id),
    CHECK (wins + losses + draws = ranked_games_played)
);
CREATE INDEX pvp_ratings_leaderboard_idx
    ON pvp_ratings (pvp_mode_id, season_id, season_rating DESC);

CREATE TABLE pvp_match_settlements (
    pvp_match_id            UUID NOT NULL,
    character_id            UUID NOT NULL REFERENCES characters(character_id)
                            ON DELETE RESTRICT,
    pvp_mode_id             VARCHAR(48) NOT NULL
                            CHECK (pvp_mode_id IN ('pvp.mode.ranked_duel',
                                                   'pvp.mode.five_element_arena')),
    season_id               INTEGER NOT NULL,
    match_state             VARCHAR(12) NOT NULL
                            CHECK (match_state IN ('COMPLETED', 'VOID')),
    result                  VARCHAR(8) NULL
                            CHECK (result IN ('WIN', 'LOSS', 'DRAW')),
    participation           VARCHAR(12) NOT NULL
                            CHECK (participation IN ('NORMAL', 'AFK', 'ABANDONED')),
    mmr_before              INTEGER NOT NULL,
    mmr_after               INTEGER NOT NULL,
    season_rating_before    INTEGER NOT NULL,
    season_rating_after     INTEGER NOT NULL,
    abandon_penalty         SMALLINT NOT NULL DEFAULT 0
                            CHECK (abandon_penalty IN (0, 20)),
    reward_eligible         BOOLEAN NOT NULL,
    ranked_bound_utc_date   DATE NULL,
    ranked_bound_slot       SMALLINT NULL
                            CHECK (ranked_bound_slot BETWEEN 1 AND 5),
    settlement_operation_id UUID NOT NULL,
    settled_at              TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (pvp_match_id, character_id),
    CHECK ((match_state = 'VOID') = (result IS NULL)),
    CHECK (match_state = 'COMPLETED'
           OR (mmr_after = mmr_before
               AND season_rating_after = season_rating_before
               AND NOT reward_eligible AND abandon_penalty = 0)),
    CHECK (participation = 'NORMAL' OR NOT reward_eligible),
    CHECK ((ranked_bound_slot IS NULL) = (ranked_bound_utc_date IS NULL)),
    CHECK (ranked_bound_slot IS NULL OR reward_eligible)
);
CREATE UNIQUE INDEX pvp_match_settlements_bound_slot_unique
    ON pvp_match_settlements (character_id, ranked_bound_utc_date,
                              ranked_bound_slot)
    WHERE ranked_bound_slot IS NOT NULL;
CREATE INDEX pvp_match_settlements_character_idx
    ON pvp_match_settlements (character_id, settled_at);

CREATE TABLE pvp_sanctions (
    sanction_id       UUID PRIMARY KEY,
    character_id      UUID NOT NULL REFERENCES characters(character_id)
                      ON DELETE RESTRICT,
    sanction_kind     VARCHAR(24) NOT NULL
                      CHECK (sanction_kind IN ('QUEUE_RESTRICTION')),
    reason            VARCHAR(12) NOT NULL
                      CHECK (reason IN ('ABANDON', 'AFK')),
    source_match_kind VARCHAR(12) NOT NULL
                      CHECK (source_match_kind IN ('PVP', 'GUILD_WAR')),
    source_match_id   UUID NOT NULL,
    ladder_step       SMALLINT NOT NULL CHECK (ladder_step IN (1, 2, 3)),
    starts_at         TIMESTAMPTZ NOT NULL,
    ends_at           TIMESTAMPTZ NOT NULL,
    operation_id      UUID NOT NULL,
    CHECK (ends_at = starts_at + CASE ladder_step
                                 WHEN 1 THEN INTERVAL '15 minutes'
                                 WHEN 2 THEN INTERVAL '30 minutes'
                                 ELSE INTERVAL '2 hours' END)
);
CREATE UNIQUE INDEX pvp_sanctions_source_unique
    ON pvp_sanctions (character_id, source_match_kind, source_match_id);
CREATE INDEX pvp_sanctions_character_starts_idx
    ON pvp_sanctions (character_id, starts_at);
CREATE INDEX pvp_sanctions_character_ends_idx
    ON pvp_sanctions (character_id, ends_at);

CREATE TABLE guild_war_ratings (
    guild_id                UUID NOT NULL REFERENCES guilds(guild_id)
                            ON DELETE RESTRICT,
    season_id               INTEGER NOT NULL CHECK (season_id >= 0),
    guild_war_mmr           INTEGER NOT NULL CHECK (guild_war_mmr >= 0),
    guild_war_games_played  INTEGER NOT NULL DEFAULT 0
                            CHECK (guild_war_games_played >= 0),
    guild_war_wins          INTEGER NOT NULL DEFAULT 0
                            CHECK (guild_war_wins >= 0),
    lifetime_rated_wars     INTEGER NOT NULL DEFAULT 0
                            CHECK (lifetime_rated_wars >= 0),
    updated_at              TIMESTAMPTZ NOT NULL,
    revision                BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, season_id),
    CHECK (guild_war_wins <= guild_war_games_played)
);
CREATE INDEX guild_war_ratings_leaderboard_idx
    ON guild_war_ratings (season_id, guild_war_mmr DESC,
                          guild_war_wins DESC, guild_war_games_played ASC);

CREATE TABLE guild_war_settlements (
    guild_war_match_id        UUID NOT NULL,
    settlement_type           VARCHAR(24) NOT NULL
                              CHECK (settlement_type IN ('GUILD_RATING',
                                                         'GUILD_PROGRESSION',
                                                         'PERSONAL_REWARD',
                                                         'SEASON_PARTICIPATION')),
    recipient_id              UUID NOT NULL,
    guild_id                  UUID NOT NULL REFERENCES guilds(guild_id)
                              ON DELETE RESTRICT,
    season_id                 INTEGER NOT NULL,
    match_state               VARCHAR(12) NOT NULL
                              CHECK (match_state IN ('COMPLETED', 'VOID')),
    result                    VARCHAR(8) NULL CHECK (result IN ('WIN', 'LOSS')),
    mmr_before                INTEGER NULL,
    mmr_after                 INTEGER NULL,
    participation             VARCHAR(12) NULL
                              CHECK (participation IN ('NORMAL', 'AFK',
                                                       'ABANDONED')),
    reward_eligible           BOOLEAN NULL,
    bound_week_monday         DATE NULL,
    bound_slot                SMALLINT NULL
                              CHECK (bound_slot BETWEEN 1 AND 3),
    settlement_operation_id   UUID NOT NULL,
    settled_at                TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (guild_war_match_id, settlement_type, recipient_id),
    CHECK ((match_state = 'VOID') = (result IS NULL)),
    CHECK ((settlement_type = 'GUILD_RATING')
           = (mmr_before IS NOT NULL AND mmr_after IS NOT NULL)),
    CHECK (settlement_type NOT IN ('GUILD_RATING', 'GUILD_PROGRESSION')
           OR recipient_id = guild_id),
    CHECK ((settlement_type IN ('PERSONAL_REWARD', 'SEASON_PARTICIPATION'))
           = (participation IS NOT NULL)),
    CHECK ((settlement_type = 'PERSONAL_REWARD')
           = (reward_eligible IS NOT NULL)),
    CHECK ((bound_slot IS NULL) = (bound_week_monday IS NULL)),
    CHECK (bound_slot IS NULL
           OR (settlement_type = 'PERSONAL_REWARD' AND reward_eligible)),
    CHECK (match_state = 'COMPLETED'
           OR (mmr_after IS NOT DISTINCT FROM mmr_before
               AND bound_slot IS NULL))
);
CREATE UNIQUE INDEX guild_war_settlements_bound_slot_unique
    ON guild_war_settlements (recipient_id, bound_week_monday, bound_slot)
    WHERE bound_slot IS NOT NULL;
CREATE INDEX guild_war_settlements_participation_idx
    ON guild_war_settlements (recipient_id, season_id, guild_id)
    WHERE settlement_type = 'SEASON_PARTICIPATION' AND match_state = 'COMPLETED';

-- ====================================================================
-- 18. Boss (data_model.md § boss_chest_eligibility / public_boss_schedules;
--      ADR-0053, ADR-0061)
-- ====================================================================
CREATE TABLE boss_chest_eligibility (
    character_id                    UUID NOT NULL
                                    REFERENCES characters(character_id)
                                    ON DELETE RESTRICT,
    public_boss_spawn_generation_id UUID NOT NULL,
    boss_id                         TEXT NOT NULL,
    copy_map_id                     VARCHAR(64) NOT NULL,
    copy_channel_id                 SMALLINT NOT NULL,
    copy_defeated                   BOOLEAN NOT NULL DEFAULT FALSE,
    eligible_until                  TIMESTAMPTZ NULL,
    claim_operation_id              UUID NULL,
    PRIMARY KEY (character_id, public_boss_spawn_generation_id),
    CHECK (copy_defeated = (eligible_until IS NOT NULL))
);
CREATE INDEX boss_chest_eligibility_copy_idx
    ON boss_chest_eligibility (public_boss_spawn_generation_id,
                               copy_map_id, copy_channel_id);

CREATE TABLE public_boss_schedules (
    boss_id                         TEXT PRIMARY KEY,
    state                           TEXT NOT NULL
                                    CHECK (state IN ('SCHEDULED', 'OPEN')),
    public_boss_spawn_generation_id UUID NULL,
    opened_at                       TIMESTAMPTZ NULL,
    next_spawn_at                   TIMESTAMPTZ NULL,
    revision                        BIGINT NOT NULL,
    CHECK ((state = 'OPEN'
            AND public_boss_spawn_generation_id IS NOT NULL
            AND opened_at IS NOT NULL AND next_spawn_at IS NULL)
        OR (state = 'SCHEDULED'
            AND public_boss_spawn_generation_id IS NULL
            AND opened_at IS NULL AND next_spawn_at IS NOT NULL))
);

-- ====================================================================
-- 19. WorldConsequence (data_model.md § Boss Aftermath Relic; ADR-0053,
--     ADR-0061, ADR-0070)
-- ====================================================================
CREATE TABLE world_consequence_relics (
    map_id         VARCHAR(64) NOT NULL,
    channel_id     SMALLINT NOT NULL CHECK (channel_id BETWEEN 1 AND 30),
    relic_id       VARCHAR(64) NOT NULL,
    source_id      VARCHAR(64) NOT NULL,
    relic_active   BOOLEAN NOT NULL,
    buff_effect_id VARCHAR(64) NOT NULL,
    spawned_at     TIMESTAMPTZ NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL
                   CHECK (expires_at = spawned_at + INTERVAL '60 minutes'),
    PRIMARY KEY (map_id, channel_id, relic_id)
);
CREATE INDEX world_consequence_relics_active_idx
    ON world_consequence_relics (relic_id, expires_at)
    WHERE relic_active;
CREATE INDEX world_consequence_relics_sweep_idx
    ON world_consequence_relics (expires_at)
    WHERE relic_active;

CREATE TABLE region_di_tich_markers (
    region_id                        VARCHAR(64) NOT NULL,
    boss_id                          VARCHAR(64) NOT NULL,
    last_defeated_utc                TIMESTAMPTZ NOT NULL,
    last_defeated_participant_count  INTEGER NOT NULL
                                     CHECK (last_defeated_participant_count >= 1),
    relic_active_in_region           BOOLEAN NOT NULL,
    PRIMARY KEY (region_id, boss_id)
);

-- ====================================================================
-- 20. Economy rollups (data_model.md § Economy Aggregation Fields)
-- ====================================================================
CREATE TABLE economy_account_daily_rollups (
    account_id     UUID NOT NULL REFERENCES accounts(account_id)
                   ON DELETE RESTRICT,
    utc_day        DATE NOT NULL,
    common_outflow BIGINT NOT NULL DEFAULT 0 CHECK (common_outflow >= 0),
    common_inflow  BIGINT NOT NULL DEFAULT 0 CHECK (common_inflow >= 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, utc_day)
);

CREATE TABLE economy_character_daily_rollups (
    character_id           UUID NOT NULL REFERENCES characters(character_id)
                           ON DELETE RESTRICT,
    utc_day                DATE NOT NULL,
    common_outflow         BIGINT NOT NULL DEFAULT 0 CHECK (common_outflow >= 0),
    common_inflow          BIGINT NOT NULL DEFAULT 0 CHECK (common_inflow >= 0),
    trade_partner_volumes  JSONB NOT NULL DEFAULT '{}'::jsonb,
    item_partner_counts    JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (character_id, utc_day)
);

-- ====================================================================
-- 21. Idempotency / Audit (data_model.md § Idempotency / Audit; ADR-0065)
-- ====================================================================
CREATE TABLE operations (
    operation_family    VARCHAR(48) NOT NULL,
    owner_kind          VARCHAR(16) NOT NULL
                        CHECK (owner_kind IN ('ACCOUNT', 'CHARACTER', 'GUILD',
                                              'WORLD')),
    owner_id            UUID NOT NULL,   -- no FK (contract §4.1)
    operation_id        UUID NOT NULL,
    request_fingerprint BYTEA NOT NULL
                        CHECK (octet_length(request_fingerprint) = 32),
    outcome             JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL,
    completed_at        TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (operation_family, owner_id, operation_id)
);
CREATE INDEX operations_completed_idx ON operations (completed_at);

CREATE TABLE audit_events (
    audit_event_id        UUID PRIMARY KEY,
    occurred_at           TIMESTAMPTZ NOT NULL,
    actor_kind            VARCHAR(16) NOT NULL
                          CHECK (actor_kind IN ('PLAYER', 'OPERATOR', 'SYSTEM')),
    actor_id              UUID NULL,   -- no FK
    subject_account_id    UUID NULL,   -- no FK (Category H; kept after erasure)
    subject_character_id  UUID NULL,   -- no FK
    action                VARCHAR(48) NOT NULL,
    reason                VARCHAR(512) NULL,
    ticket_id             VARCHAR(64) NULL,
    operation_id          UUID NULL,
    payload               JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX audit_events_subject_account_idx
    ON audit_events (subject_account_id, occurred_at);
CREATE INDEX audit_events_subject_character_idx
    ON audit_events (subject_character_id, occurred_at);
CREATE INDEX audit_events_occurred_idx ON audit_events (occurred_at);

CREATE TABLE operators (
    operator_id            UUID PRIMARY KEY,
    login_key              VARCHAR(32) NOT NULL UNIQUE,
    password_hash          TEXT NOT NULL,
    totp_secret_encrypted  BYTEA NOT NULL,
    role                   VARCHAR(16) NOT NULL
                           CHECK (role IN ('SUPPORT', 'MODERATOR', 'ECONOMY',
                                           'ADMIN')),
    status                 VARCHAR(16) NOT NULL
                           CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_at             TIMESTAMPTZ NOT NULL,
    last_login_at          TIMESTAMPTZ NULL
);

CREATE TABLE chat_messages (
    message_id           UUID PRIMARY KEY,
    sender_account_id    UUID NOT NULL,
    sender_character_id  UUID NOT NULL,
    channel              VARCHAR(16) NOT NULL
                         CHECK (channel IN ('WORLD', 'PARTY', 'GUILD',
                                            'WHISPER', 'LOCAL')),
    scope_id             TEXT NULL,
    content              TEXT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL
);
CREATE INDEX chat_messages_sender_idx
    ON chat_messages (sender_account_id, created_at);
CREATE INDEX chat_messages_created_idx ON chat_messages (created_at);

-- ====================================================================
-- 22. Security counters + erasure ledger (external_integrations.md § 3;
--     ADR-0064, ADR-0069, ADR-0070)
-- ====================================================================
CREATE TABLE rate_limit_counters (
    key_hash       BYTEA PRIMARY KEY
                   CHECK (octet_length(key_hash) = 32),
    window_seconds INTEGER NOT NULL CHECK (window_seconds > 0),
    window_start   TIMESTAMPTZ NOT NULL,
    current_count  INTEGER NOT NULL CHECK (current_count >= 0),
    previous_count INTEGER NOT NULL DEFAULT 0 CHECK (previous_count >= 0)
);

CREATE TABLE auth_failure_backoff (
    key_hash             BYTEA PRIMARY KEY
                         CHECK (octet_length(key_hash) = 32),
    consecutive_failures INTEGER NOT NULL CHECK (consecutive_failures >= 0),
    locked_until         TIMESTAMPTZ NULL,
    last_failure_at      TIMESTAMPTZ NOT NULL
);

CREATE TABLE pending_erasure_ledger (
    operation_id    UUID PRIMARY KEY,
    account_id_hash BYTEA NOT NULL,
    executed_at     TIMESTAMPTZ NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      VARCHAR(256) NULL
);
