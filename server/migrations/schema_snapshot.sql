--
-- PostgreSQL database dump
--


-- Dumped from database version 18.6 (Debian 18.6-1.pgdg13+2)
-- Dumped by pg_dump version 18.6 (Debian 18.6-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_cosmetic_entitlements; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_cosmetic_entitlements (
    account_id uuid NOT NULL,
    cosmetic_id text NOT NULL,
    entitlement_id uuid NOT NULL,
    granted_at timestamp with time zone NOT NULL,
    first_equipped_at timestamp with time zone
);


ALTER TABLE public.account_cosmetic_entitlements OWNER TO postgres;

--
-- Name: account_entitlement_claims; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_entitlement_claims (
    account_entitlement_id uuid NOT NULL,
    account_id uuid NOT NULL,
    entitlement_type character varying(24) NOT NULL,
    character_id uuid NOT NULL,
    reward_tier_id character varying(64) NOT NULL,
    claimed_at timestamp with time zone NOT NULL,
    claim_operation_id uuid NOT NULL,
    CONSTRAINT account_entitlement_claims_entitlement_type_check CHECK (((entitlement_type)::text = 'ACCOUNT_SCOPED_ACCESS'::text))
);


ALTER TABLE public.account_entitlement_claims OWNER TO postgres;

--
-- Name: account_iap_entitlements; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_iap_entitlements (
    entitlement_id uuid NOT NULL,
    account_id uuid NOT NULL,
    product_id character varying(64) NOT NULL,
    entitlement_type character varying(24) NOT NULL,
    grant_state character varying(20) NOT NULL,
    reject_reason character varying(24),
    platform character varying(16) NOT NULL,
    platform_receipt character varying(512) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    granted_at timestamp with time zone,
    ended_at timestamp with time zone,
    season_number integer,
    claim_deadline_at timestamp with time zone,
    CONSTRAINT account_iap_entitlements_check CHECK ((((grant_state)::text = 'REJECTED'::text) = (reject_reason IS NOT NULL))),
    CONSTRAINT account_iap_entitlements_entitlement_type_check CHECK (((entitlement_type)::text = ANY ((ARRAY['ONE_SHOT'::character varying, 'ACCOUNT_SCOPED_ACCESS'::character varying, 'DIRECT_ACCOUNT_COSMETIC'::character varying])::text[]))),
    CONSTRAINT account_iap_entitlements_grant_state_check CHECK (((grant_state)::text = ANY ((ARRAY['PENDING'::character varying, 'GRANTED'::character varying, 'REJECTED'::character varying, 'REFUNDED'::character varying, 'REFUNDED_CONSUMED'::character varying])::text[]))),
    CONSTRAINT account_iap_entitlements_platform_check CHECK (((platform)::text = ANY ((ARRAY['GOOGLE_PLAY'::character varying, 'APP_STORE'::character varying, 'STEAM'::character varying])::text[]))),
    CONSTRAINT account_iap_entitlements_reject_reason_check CHECK (((reject_reason)::text = ANY ((ARRAY['RECEIPT_INVALID'::character varying, 'PRODUCT_MISMATCH'::character varying, 'SEASON_TRACK_DUPLICATE'::character varying])::text[])))
);


ALTER TABLE public.account_iap_entitlements OWNER TO postgres;

--
-- Name: account_identities; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_identities (
    provider_id character varying(16) NOT NULL,
    provider_subject character varying(255) NOT NULL,
    account_id uuid NOT NULL,
    linked_at timestamp with time zone NOT NULL,
    CONSTRAINT account_identities_provider_id_check CHECK (((provider_id)::text = ANY ((ARRAY['apple'::character varying, 'google'::character varying, 'steam'::character varying])::text[])))
);


ALTER TABLE public.account_identities OWNER TO postgres;

--
-- Name: account_login_history; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_login_history (
    account_id uuid NOT NULL,
    observed_at timestamp with time zone NOT NULL,
    device_id_hash bytea NOT NULL,
    ip_prefix16_hash bytea NOT NULL,
    is_new_origin boolean NOT NULL
);


ALTER TABLE public.account_login_history OWNER TO postgres;

--
-- Name: account_password_credentials; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_password_credentials (
    account_id uuid NOT NULL,
    username_key character varying(20) NOT NULL,
    email character varying(254) NOT NULL,
    email_key character varying(254) NOT NULL,
    password_hash text NOT NULL,
    params_version smallint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.account_password_credentials OWNER TO postgres;

--
-- Name: account_refund_consumed_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.account_refund_consumed_events (
    event_id uuid NOT NULL,
    account_id uuid NOT NULL,
    entitlement_id uuid NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.account_refund_consumed_events OWNER TO postgres;

--
-- Name: accounts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.accounts (
    account_id uuid NOT NULL,
    status character varying(32) DEFAULT 'ACTIVE'::character varying NOT NULL,
    deletion_requested_at timestamp with time zone,
    erased_at timestamp with time zone,
    credential_guard_until timestamp with time zone,
    economy_review_flagged_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT accounts_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'SUSPENDED_PAYMENT_RECONCILIATION'::character varying, 'BANNED'::character varying, 'PENDING_DELETION'::character varying, 'TOMBSTONE_ERASED'::character varying])::text[])))
);


ALTER TABLE public.accounts OWNER TO postgres;

--
-- Name: atlas_milestones; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.atlas_milestones (
    character_id uuid NOT NULL,
    milestone_id character varying(64) NOT NULL,
    completed_at timestamp with time zone NOT NULL
);


ALTER TABLE public.atlas_milestones OWNER TO postgres;

--
-- Name: auction_listings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auction_listings (
    listing_id uuid NOT NULL,
    seller_character_id uuid NOT NULL,
    seller_account_id uuid NOT NULL,
    item_instance_id uuid NOT NULL,
    item_id character varying(64) NOT NULL,
    quantity integer NOT NULL,
    price_common bigint NOT NULL,
    listing_fee_common bigint NOT NULL,
    state character varying(16) NOT NULL,
    listed_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    ended_at timestamp with time zone,
    buyer_character_id uuid,
    revision bigint NOT NULL,
    CONSTRAINT auction_listings_check CHECK ((expires_at = (listed_at + '24:00:00'::interval))),
    CONSTRAINT auction_listings_check1 CHECK ((((state)::text = 'ACTIVE'::text) = (ended_at IS NULL))),
    CONSTRAINT auction_listings_listing_fee_common_check CHECK ((listing_fee_common >= 10)),
    CONSTRAINT auction_listings_price_common_check CHECK (((price_common >= 100) AND (price_common <= 2000000000))),
    CONSTRAINT auction_listings_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT auction_listings_state_check CHECK (((state)::text = ANY ((ARRAY['ACTIVE'::character varying, 'SOLD'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying, 'RECLAIMED'::character varying, 'MOVED_TO_CLAIM'::character varying])::text[])))
);


ALTER TABLE public.auction_listings OWNER TO postgres;

--
-- Name: auction_proceeds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auction_proceeds (
    proceeds_id uuid NOT NULL,
    settled_at timestamp with time zone NOT NULL,
    seller_character_id uuid NOT NULL,
    seller_account_id uuid NOT NULL,
    buyer_character_id uuid NOT NULL,
    buyer_account_id uuid NOT NULL,
    proceeds_amount bigint NOT NULL,
    listing_id uuid NOT NULL,
    state character varying(16) NOT NULL,
    claimed_at timestamp with time zone,
    claim_operation_id uuid,
    CONSTRAINT auction_proceeds_proceeds_amount_check CHECK ((proceeds_amount > 0)),
    CONSTRAINT auction_proceeds_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'CLAIMED'::character varying])::text[])))
);


ALTER TABLE public.auction_proceeds OWNER TO postgres;

--
-- Name: audit_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.audit_events (
    audit_event_id uuid NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    actor_kind character varying(16) NOT NULL,
    actor_id uuid,
    subject_account_id uuid,
    subject_character_id uuid,
    action character varying(48) NOT NULL,
    reason character varying(512),
    ticket_id character varying(64),
    operation_id uuid,
    payload jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT audit_events_actor_kind_check CHECK (((actor_kind)::text = ANY ((ARRAY['PLAYER'::character varying, 'OPERATOR'::character varying, 'SYSTEM'::character varying])::text[])))
);


ALTER TABLE public.audit_events OWNER TO postgres;

--
-- Name: auth_failure_backoff; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auth_failure_backoff (
    key_hash bytea NOT NULL,
    consecutive_failures integer NOT NULL,
    locked_until timestamp with time zone,
    last_failure_at timestamp with time zone NOT NULL,
    CONSTRAINT auth_failure_backoff_consecutive_failures_check CHECK ((consecutive_failures >= 0)),
    CONSTRAINT auth_failure_backoff_key_hash_check CHECK ((octet_length(key_hash) = 32))
);


ALTER TABLE public.auth_failure_backoff OWNER TO postgres;

--
-- Name: auth_refresh_credentials; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auth_refresh_credentials (
    credential_hash bytea NOT NULL,
    session_family_id uuid NOT NULL,
    generation integer NOT NULL,
    issued_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    first_presented_at timestamp with time zone,
    rotated_at timestamp with time zone
);


ALTER TABLE public.auth_refresh_credentials OWNER TO postgres;

--
-- Name: auth_revocations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auth_revocations (
    revocation_id uuid NOT NULL,
    scope character varying(16) NOT NULL,
    account_id uuid,
    session_family_id uuid,
    provider_id character varying(16),
    not_before timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT auth_revocations_scope_check CHECK (((scope)::text = ANY ((ARRAY['SESSION_FAMILY'::character varying, 'ACCOUNT'::character varying, 'PROVIDER_LINK'::character varying])::text[])))
);


ALTER TABLE public.auth_revocations OWNER TO postgres;

--
-- Name: auth_session_families; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.auth_session_families (
    session_family_id uuid NOT NULL,
    account_id uuid NOT NULL,
    provider_id character varying(16) NOT NULL,
    client_platform character varying(16) NOT NULL,
    app_version character varying(32) NOT NULL,
    device_model_class character varying(32),
    device_id_hash bytea NOT NULL,
    absolute_expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL,
    last_refreshed_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revoke_reason character varying(32),
    CONSTRAINT auth_session_families_client_platform_check CHECK (((client_platform)::text = ANY ((ARRAY['WINDOWS'::character varying, 'ANDROID'::character varying])::text[]))),
    CONSTRAINT auth_session_families_provider_id_check CHECK (((provider_id)::text = ANY ((ARRAY['password'::character varying, 'apple'::character varying, 'google'::character varying, 'steam'::character varying])::text[]))),
    CONSTRAINT auth_session_families_revoke_reason_check CHECK (((revoke_reason)::text = ANY ((ARRAY['LOGOUT'::character varying, 'REUSE_DETECTED'::character varying, 'ACCOUNT_REVOKE'::character varying, 'PASSWORD_CHANGE'::character varying, 'TAKEOVER_RULE'::character varying, 'ERASURE_REQUEST'::character varying, 'ADMIN'::character varying])::text[])))
);


ALTER TABLE public.auth_session_families OWNER TO postgres;

--
-- Name: beast_equipment_locations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.beast_equipment_locations (
    character_id uuid NOT NULL,
    beast_id character varying(64) NOT NULL,
    slot_id character varying(16) NOT NULL,
    item_instance_id uuid NOT NULL,
    CONSTRAINT beast_equipment_locations_slot_id_check CHECK (((slot_id)::text = ANY ((ARRAY['vong_co'::character varying, 'ao_giap'::character varying, 'linh_chau'::character varying])::text[])))
);


ALTER TABLE public.beast_equipment_locations OWNER TO postgres;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks (
    blocker_character_id uuid NOT NULL,
    blocked_character_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    operation_id uuid NOT NULL,
    CONSTRAINT blocks_check CHECK ((blocker_character_id <> blocked_character_id))
);


ALTER TABLE public.blocks OWNER TO postgres;

--
-- Name: boss_chest_eligibility; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.boss_chest_eligibility (
    character_id uuid NOT NULL,
    public_boss_spawn_generation_id uuid NOT NULL,
    boss_id text NOT NULL,
    copy_map_id character varying(64) NOT NULL,
    copy_channel_id smallint NOT NULL,
    copy_defeated boolean DEFAULT false NOT NULL,
    eligible_until timestamp with time zone,
    claim_operation_id uuid,
    CONSTRAINT boss_chest_eligibility_check CHECK ((copy_defeated = (eligible_until IS NOT NULL)))
);


ALTER TABLE public.boss_chest_eligibility OWNER TO postgres;

--
-- Name: character_atlas; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_atlas (
    character_id uuid NOT NULL,
    atlas_page_id character varying(64) NOT NULL,
    tier smallint NOT NULL,
    seen_count integer DEFAULT 0 NOT NULL,
    completed_at timestamp with time zone,
    reward_operation_id uuid,
    acknowledged_at timestamp with time zone,
    CONSTRAINT character_atlas_seen_count_check CHECK ((seen_count >= 0)),
    CONSTRAINT character_atlas_tier_check CHECK (((tier >= 1) AND (tier <= 3)))
);


ALTER TABLE public.character_atlas OWNER TO postgres;

--
-- Name: character_beast_food_daily; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_beast_food_daily (
    character_id uuid NOT NULL,
    utc_date date NOT NULL,
    food_points_gained smallint DEFAULT 0 NOT NULL,
    CONSTRAINT character_beast_food_daily_food_points_gained_check CHECK (((food_points_gained >= 0) AND (food_points_gained <= 20)))
);


ALTER TABLE public.character_beast_food_daily OWNER TO postgres;

--
-- Name: character_beasts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_beasts (
    character_id uuid NOT NULL,
    beast_id character varying(64) NOT NULL,
    level smallint DEFAULT 1 NOT NULL,
    bond_points integer DEFAULT 0 NOT NULL,
    is_active boolean DEFAULT false NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT character_beasts_bond_points_check CHECK ((bond_points >= 0)),
    CONSTRAINT character_beasts_level_check CHECK (((level >= 1) AND (level <= 60)))
);


ALTER TABLE public.character_beasts OWNER TO postgres;

--
-- Name: character_chivalry; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_chivalry (
    character_id uuid NOT NULL,
    lifetime_points bigint DEFAULT 0 NOT NULL,
    day_utc date,
    day_points smallint DEFAULT 0 NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT character_chivalry_day_points_check CHECK (((day_points >= 0) AND (day_points <= 100))),
    CONSTRAINT character_chivalry_lifetime_points_check CHECK ((lifetime_points >= 0))
);


ALTER TABLE public.character_chivalry OWNER TO postgres;

--
-- Name: character_cosmetic_entitlements; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_cosmetic_entitlements (
    character_id uuid NOT NULL,
    cosmetic_id character varying(64) NOT NULL,
    source_kind character varying(16) NOT NULL,
    source_entitlement_id uuid,
    source_ref character varying(64) NOT NULL,
    grant_operation_id uuid NOT NULL,
    granted_at timestamp with time zone NOT NULL,
    CONSTRAINT character_cosmetic_entitlements_check CHECK ((((source_kind)::text = 'SEASON_TRACK'::text) = (source_entitlement_id IS NOT NULL))),
    CONSTRAINT character_cosmetic_entitlements_source_kind_check CHECK (((source_kind)::text = ANY ((ARRAY['PLAY'::character varying, 'REDEMPTION'::character varying, 'SEASON_TRACK'::character varying])::text[])))
);


ALTER TABLE public.character_cosmetic_entitlements OWNER TO postgres;

--
-- Name: character_cosmetic_equips; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_cosmetic_equips (
    character_id uuid NOT NULL,
    slot character varying(32) NOT NULL,
    cosmetic_id character varying(64) NOT NULL,
    CONSTRAINT character_cosmetic_equips_slot_check CHECK (((slot)::text = ANY ((ARRAY['title'::character varying, 'title_glow'::character varying, 'frame'::character varying, 'nameplate'::character varying, 'appearance'::character varying, 'weapon_trail'::character varying, 'aura'::character varying, 'character_shrine'::character varying, 'guild_stone_inscription'::character varying])::text[])))
);


ALTER TABLE public.character_cosmetic_equips OWNER TO postgres;

--
-- Name: character_currencies; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_currencies (
    character_id uuid NOT NULL,
    currency_id character varying(32) NOT NULL,
    balance bigint DEFAULT 0 NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT character_currencies_balance_check CHECK ((balance >= 0)),
    CONSTRAINT character_currencies_currency_id_check CHECK (((currency_id)::text = ANY ((ARRAY['currency.common'::character varying, 'currency.bound'::character varying, 'currency.special'::character varying])::text[])))
);


ALTER TABLE public.character_currencies OWNER TO postgres;

--
-- Name: character_feat_milestones; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_feat_milestones (
    character_id uuid NOT NULL,
    feat_id character varying(64) NOT NULL,
    milestone_threshold integer NOT NULL,
    completed_at timestamp with time zone NOT NULL,
    reward_operation_id uuid
);


ALTER TABLE public.character_feat_milestones OWNER TO postgres;

--
-- Name: character_feats; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_feats (
    character_id uuid NOT NULL,
    feat_id character varying(64) NOT NULL,
    counter_value integer DEFAULT 0 NOT NULL,
    CONSTRAINT character_feats_counter_value_check CHECK ((counter_value >= 0))
);


ALTER TABLE public.character_feats OWNER TO postgres;

--
-- Name: character_inventories; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_inventories (
    character_id uuid NOT NULL,
    capacity integer NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT character_inventories_capacity_check CHECK (((capacity >= 60) AND (capacity <= 120)))
);


ALTER TABLE public.character_inventories OWNER TO postgres;

--
-- Name: character_loadouts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_loadouts (
    character_id uuid NOT NULL,
    loadout_index smallint NOT NULL,
    role character varying(8) NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT character_loadouts_loadout_index_check CHECK (((loadout_index >= 1) AND (loadout_index <= 3))),
    CONSTRAINT character_loadouts_role_check CHECK (((role)::text = ANY ((ARRAY['ACTIVE'::character varying, 'SUPPORT'::character varying])::text[])))
);


ALTER TABLE public.character_loadouts OWNER TO postgres;

--
-- Name: character_soul_resonance; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_soul_resonance (
    character_id uuid NOT NULL,
    soul_id character varying(64) NOT NULL,
    memory_resonance_count integer DEFAULT 0 NOT NULL,
    sheen_unlocked_at timestamp with time zone,
    CONSTRAINT character_soul_resonance_memory_resonance_count_check CHECK ((memory_resonance_count >= 0))
);


ALTER TABLE public.character_soul_resonance OWNER TO postgres;

--
-- Name: character_souls; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.character_souls (
    soul_instance_id uuid NOT NULL,
    character_id uuid NOT NULL,
    soul_id character varying(64) NOT NULL,
    level smallint NOT NULL,
    current_soul_exp integer NOT NULL,
    contracted_item_instance_id uuid,
    CONSTRAINT character_souls_current_soul_exp_check CHECK ((current_soul_exp >= 0)),
    CONSTRAINT character_souls_level_check CHECK (((level >= 1) AND (level <= 5)))
);


ALTER TABLE public.character_souls OWNER TO postgres;

--
-- Name: characters; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.characters (
    character_id uuid NOT NULL,
    account_id uuid NOT NULL,
    name character varying(64) NOT NULL,
    name_key character varying(256) NOT NULL,
    class_id character varying(64) NOT NULL,
    lifecycle character varying(16) DEFAULT 'ACTIVE'::character varying NOT NULL,
    level integer DEFAULT 1 NOT NULL,
    current_exp integer DEFAULT 0 NOT NULL,
    appearance jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT characters_current_exp_check CHECK (((current_exp >= 0) AND (current_exp <= 702100000))),
    CONSTRAINT characters_level_check CHECK (((level >= 1) AND (level <= 60))),
    CONSTRAINT characters_lifecycle_check CHECK (((lifecycle)::text = ANY ((ARRAY['ACTIVE'::character varying, 'OFFLINE'::character varying])::text[])))
);


ALTER TABLE public.characters OWNER TO postgres;

--
-- Name: chat_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chat_messages (
    message_id uuid NOT NULL,
    sender_account_id uuid NOT NULL,
    sender_character_id uuid NOT NULL,
    channel character varying(16) NOT NULL,
    scope_id text,
    content text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT chat_messages_channel_check CHECK (((channel)::text = ANY ((ARRAY['WORLD'::character varying, 'PARTY'::character varying, 'GUILD'::character varying, 'WHISPER'::character varying, 'LOCAL'::character varying])::text[])))
);


ALTER TABLE public.chat_messages OWNER TO postgres;

--
-- Name: economy_account_daily_rollups; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.economy_account_daily_rollups (
    account_id uuid NOT NULL,
    utc_day date NOT NULL,
    common_outflow bigint DEFAULT 0 NOT NULL,
    common_inflow bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT economy_account_daily_rollups_common_inflow_check CHECK ((common_inflow >= 0)),
    CONSTRAINT economy_account_daily_rollups_common_outflow_check CHECK ((common_outflow >= 0))
);


ALTER TABLE public.economy_account_daily_rollups OWNER TO postgres;

--
-- Name: economy_character_daily_rollups; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.economy_character_daily_rollups (
    character_id uuid NOT NULL,
    utc_day date NOT NULL,
    common_outflow bigint DEFAULT 0 NOT NULL,
    common_inflow bigint DEFAULT 0 NOT NULL,
    trade_partner_volumes jsonb DEFAULT '{}'::jsonb NOT NULL,
    item_partner_counts jsonb DEFAULT '{}'::jsonb NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT economy_character_daily_rollups_common_inflow_check CHECK ((common_inflow >= 0)),
    CONSTRAINT economy_character_daily_rollups_common_outflow_check CHECK ((common_outflow >= 0))
);


ALTER TABLE public.economy_character_daily_rollups OWNER TO postgres;

--
-- Name: enhancement_pity; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.enhancement_pity (
    item_instance_id uuid NOT NULL,
    target_level smallint NOT NULL,
    pity_fail_count smallint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT enhancement_pity_pity_fail_count_check CHECK (((pity_fail_count >= 0) AND (pity_fail_count <= 9))),
    CONSTRAINT enhancement_pity_target_level_check CHECK (((target_level >= 13) AND (target_level <= 16)))
);


ALTER TABLE public.enhancement_pity OWNER TO postgres;

--
-- Name: friend_requests; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.friend_requests (
    friend_request_id uuid NOT NULL,
    requester_character_id uuid NOT NULL,
    target_character_id uuid NOT NULL,
    state character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    create_operation_id uuid NOT NULL,
    resolve_operation_id uuid,
    CONSTRAINT friend_requests_check CHECK ((requester_character_id <> target_character_id)),
    CONSTRAINT friend_requests_check1 CHECK ((expires_at = (created_at + '7 days'::interval))),
    CONSTRAINT friend_requests_check2 CHECK ((((state)::text = 'PENDING'::text) = (resolved_at IS NULL))),
    CONSTRAINT friend_requests_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'ACCEPTED'::character varying, 'DECLINED'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying])::text[])))
);


ALTER TABLE public.friend_requests OWNER TO postgres;

--
-- Name: friends; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.friends (
    character_low_id uuid NOT NULL,
    character_high_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    created_operation_id uuid NOT NULL,
    CONSTRAINT friends_check CHECK ((character_low_id < character_high_id))
);


ALTER TABLE public.friends OWNER TO postgres;

--
-- Name: guild_applications; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_applications (
    application_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    applicant_character_id uuid NOT NULL,
    state character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    resolver_character_id uuid,
    CONSTRAINT guild_applications_check CHECK ((expires_at = (created_at + '7 days'::interval))),
    CONSTRAINT guild_applications_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'ACCEPTED'::character varying, 'REJECTED'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying])::text[])))
);


ALTER TABLE public.guild_applications OWNER TO postgres;

--
-- Name: guild_blessing_votes; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_blessing_votes (
    guild_id uuid NOT NULL,
    cycle_id character varying(16) NOT NULL,
    account_id uuid NOT NULL,
    character_id uuid NOT NULL,
    blessing_id character varying(64) NOT NULL,
    voted_at timestamp with time zone NOT NULL
);


ALTER TABLE public.guild_blessing_votes OWNER TO postgres;

--
-- Name: guild_invites; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_invites (
    invite_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    inviter_character_id uuid NOT NULL,
    target_character_id uuid NOT NULL,
    state character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    CONSTRAINT guild_invites_check CHECK ((expires_at = (created_at + '00:10:00'::interval))),
    CONSTRAINT guild_invites_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'ACCEPTED'::character varying, 'DECLINED'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying])::text[])))
);


ALTER TABLE public.guild_invites OWNER TO postgres;

--
-- Name: guild_member_contributions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_member_contributions (
    guild_id uuid NOT NULL,
    character_id uuid NOT NULL,
    lifetime_contribution bigint DEFAULT 0 NOT NULL,
    cycle_id character varying(16) NOT NULL,
    cycle_contribution bigint DEFAULT 0 NOT NULL,
    CONSTRAINT guild_member_contributions_cycle_contribution_check CHECK ((cycle_contribution >= 0)),
    CONSTRAINT guild_member_contributions_lifetime_contribution_check CHECK ((lifetime_contribution >= 0))
);


ALTER TABLE public.guild_member_contributions OWNER TO postgres;

--
-- Name: guild_memberships; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_memberships (
    character_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    role character varying(16) NOT NULL,
    joined_at timestamp with time zone NOT NULL,
    CONSTRAINT guild_memberships_role_check CHECK (((role)::text = ANY ((ARRAY['LEADER'::character varying, 'VICE_LEADER'::character varying, 'OFFICER'::character varying, 'MEMBER'::character varying])::text[])))
);


ALTER TABLE public.guild_memberships OWNER TO postgres;

--
-- Name: guild_progression; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_progression (
    guild_id uuid NOT NULL,
    guild_exp bigint DEFAULT 0 NOT NULL,
    guild_level smallint NOT NULL,
    ritual_streak integer DEFAULT 0 NOT NULL,
    active_blessing_id character varying(64),
    blessing_expires_at timestamp with time zone,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT guild_progression_check CHECK (((active_blessing_id IS NULL) = (blessing_expires_at IS NULL))),
    CONSTRAINT guild_progression_guild_exp_check CHECK ((guild_exp >= 0)),
    CONSTRAINT guild_progression_guild_level_check CHECK ((guild_level >= 1)),
    CONSTRAINT guild_progression_ritual_streak_check CHECK ((ritual_streak >= 0))
);


ALTER TABLE public.guild_progression OWNER TO postgres;

--
-- Name: guild_ritual_cycles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_ritual_cycles (
    guild_id uuid NOT NULL,
    cycle_id character varying(16) NOT NULL,
    m_effective smallint NOT NULL,
    required_points_per_element integer NOT NULL,
    points_kim integer DEFAULT 0 NOT NULL,
    points_moc integer DEFAULT 0 NOT NULL,
    points_thuy integer DEFAULT 0 NOT NULL,
    points_hoa integer DEFAULT 0 NOT NULL,
    points_tho integer DEFAULT 0 NOT NULL,
    rotation_pointer character varying(4) NOT NULL,
    completed_at timestamp with time zone,
    candidate_blessing_ids character varying(64)[],
    vote_closes_at timestamp with time zone,
    finalized_blessing_id character varying(64),
    CONSTRAINT guild_ritual_cycles_candidate_blessing_ids_check CHECK (((candidate_blessing_ids IS NULL) OR (array_length(candidate_blessing_ids, 1) = 3))),
    CONSTRAINT guild_ritual_cycles_points_hoa_check CHECK ((points_hoa >= 0)),
    CONSTRAINT guild_ritual_cycles_points_kim_check CHECK ((points_kim >= 0)),
    CONSTRAINT guild_ritual_cycles_points_moc_check CHECK ((points_moc >= 0)),
    CONSTRAINT guild_ritual_cycles_points_tho_check CHECK ((points_tho >= 0)),
    CONSTRAINT guild_ritual_cycles_points_thuy_check CHECK ((points_thuy >= 0)),
    CONSTRAINT guild_ritual_cycles_rotation_pointer_check CHECK (((rotation_pointer)::text = ANY ((ARRAY['KIM'::character varying, 'MOC'::character varying, 'THUY'::character varying, 'HOA'::character varying, 'THO'::character varying])::text[])))
);


ALTER TABLE public.guild_ritual_cycles OWNER TO postgres;

--
-- Name: guild_stone_category_completions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_stone_category_completions (
    guild_id uuid NOT NULL,
    category_id text NOT NULL,
    season_number integer NOT NULL,
    completed_at timestamp with time zone NOT NULL
);


ALTER TABLE public.guild_stone_category_completions OWNER TO postgres;

--
-- Name: guild_storage_audit; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_storage_audit (
    audit_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    operation_id uuid NOT NULL,
    actor_character_id uuid NOT NULL,
    action character varying(24) NOT NULL,
    section character varying(8) NOT NULL,
    item_id character varying(64) NOT NULL,
    quantity integer NOT NULL,
    receiver_character_id uuid,
    before_quantity integer NOT NULL,
    after_quantity integer NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    CONSTRAINT guild_storage_audit_action_check CHECK (((action)::text = ANY ((ARRAY['DEPOSIT'::character varying, 'WITHDRAW'::character varying, 'MOVE'::character varying, 'CLAIM_REQUEST'::character varying, 'CLAIM_APPROVE'::character varying, 'CLAIM_REJECT'::character varying, 'CLAIM_DELIVER'::character varying, 'CLAIM_CANCEL'::character varying, 'CLAIM_EXPIRE'::character varying])::text[]))),
    CONSTRAINT guild_storage_audit_after_quantity_check CHECK ((after_quantity >= 0)),
    CONSTRAINT guild_storage_audit_before_quantity_check CHECK ((before_quantity >= 0)),
    CONSTRAINT guild_storage_audit_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT guild_storage_audit_section_check CHECK (((section)::text = ANY ((ARRAY['COMMON'::character varying, 'RESERVE'::character varying])::text[])))
);


ALTER TABLE public.guild_storage_audit OWNER TO postgres;

--
-- Name: guild_storage_claims; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_storage_claims (
    claim_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    requester_character_id uuid NOT NULL,
    item_instance_id uuid NOT NULL,
    quantity integer NOT NULL,
    state character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    approved_at timestamp with time zone,
    approver_character_id uuid,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    CONSTRAINT guild_storage_claims_quantity_check CHECK ((quantity > 0)),
    CONSTRAINT guild_storage_claims_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying, 'COMPLETED'::character varying])::text[])))
);


ALTER TABLE public.guild_storage_claims OWNER TO postgres;

--
-- Name: guild_war_ratings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_war_ratings (
    guild_id uuid NOT NULL,
    season_id integer NOT NULL,
    guild_war_mmr integer NOT NULL,
    guild_war_games_played integer DEFAULT 0 NOT NULL,
    guild_war_wins integer DEFAULT 0 NOT NULL,
    lifetime_rated_wars integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT guild_war_ratings_check CHECK ((guild_war_wins <= guild_war_games_played)),
    CONSTRAINT guild_war_ratings_guild_war_games_played_check CHECK ((guild_war_games_played >= 0)),
    CONSTRAINT guild_war_ratings_guild_war_mmr_check CHECK ((guild_war_mmr >= 0)),
    CONSTRAINT guild_war_ratings_guild_war_wins_check CHECK ((guild_war_wins >= 0)),
    CONSTRAINT guild_war_ratings_lifetime_rated_wars_check CHECK ((lifetime_rated_wars >= 0)),
    CONSTRAINT guild_war_ratings_season_id_check CHECK ((season_id >= 0))
);


ALTER TABLE public.guild_war_ratings OWNER TO postgres;

--
-- Name: guild_war_settlements; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guild_war_settlements (
    guild_war_match_id uuid NOT NULL,
    settlement_type character varying(24) NOT NULL,
    recipient_id uuid NOT NULL,
    guild_id uuid NOT NULL,
    season_id integer NOT NULL,
    match_state character varying(12) NOT NULL,
    result character varying(8),
    mmr_before integer,
    mmr_after integer,
    participation character varying(12),
    reward_eligible boolean,
    bound_week_monday date,
    bound_slot smallint,
    settlement_operation_id uuid NOT NULL,
    settled_at timestamp with time zone NOT NULL,
    CONSTRAINT guild_war_settlements_bound_slot_check CHECK (((bound_slot >= 1) AND (bound_slot <= 3))),
    CONSTRAINT guild_war_settlements_check CHECK ((((match_state)::text = 'VOID'::text) = (result IS NULL))),
    CONSTRAINT guild_war_settlements_check1 CHECK ((((settlement_type)::text = 'GUILD_RATING'::text) = ((mmr_before IS NOT NULL) AND (mmr_after IS NOT NULL)))),
    CONSTRAINT guild_war_settlements_check2 CHECK ((((settlement_type)::text <> ALL ((ARRAY['GUILD_RATING'::character varying, 'GUILD_PROGRESSION'::character varying])::text[])) OR (recipient_id = guild_id))),
    CONSTRAINT guild_war_settlements_check3 CHECK ((((settlement_type)::text = ANY ((ARRAY['PERSONAL_REWARD'::character varying, 'SEASON_PARTICIPATION'::character varying])::text[])) = (participation IS NOT NULL))),
    CONSTRAINT guild_war_settlements_check4 CHECK ((((settlement_type)::text = 'PERSONAL_REWARD'::text) = (reward_eligible IS NOT NULL))),
    CONSTRAINT guild_war_settlements_check5 CHECK (((bound_slot IS NULL) = (bound_week_monday IS NULL))),
    CONSTRAINT guild_war_settlements_check6 CHECK (((bound_slot IS NULL) OR (((settlement_type)::text = 'PERSONAL_REWARD'::text) AND reward_eligible))),
    CONSTRAINT guild_war_settlements_check7 CHECK ((((match_state)::text = 'COMPLETED'::text) OR ((NOT (mmr_after IS DISTINCT FROM mmr_before)) AND (bound_slot IS NULL)))),
    CONSTRAINT guild_war_settlements_match_state_check CHECK (((match_state)::text = ANY ((ARRAY['COMPLETED'::character varying, 'VOID'::character varying])::text[]))),
    CONSTRAINT guild_war_settlements_participation_check CHECK (((participation)::text = ANY ((ARRAY['NORMAL'::character varying, 'AFK'::character varying, 'ABANDONED'::character varying])::text[]))),
    CONSTRAINT guild_war_settlements_result_check CHECK (((result)::text = ANY ((ARRAY['WIN'::character varying, 'LOSS'::character varying])::text[]))),
    CONSTRAINT guild_war_settlements_settlement_type_check CHECK (((settlement_type)::text = ANY ((ARRAY['GUILD_RATING'::character varying, 'GUILD_PROGRESSION'::character varying, 'PERSONAL_REWARD'::character varying, 'SEASON_PARTICIPATION'::character varying])::text[])))
);


ALTER TABLE public.guild_war_settlements OWNER TO postgres;

--
-- Name: guilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guilds (
    guild_id uuid NOT NULL,
    name character varying(96) NOT NULL,
    name_key character varying(256) NOT NULL,
    state character varying(16) NOT NULL,
    recruitment_mode character varying(16) NOT NULL,
    leader_character_id uuid,
    motd character varying(1024) DEFAULT ''::character varying NOT NULL,
    guild_revision bigint NOT NULL,
    guild_storage_revision bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    disbanded_at timestamp with time zone,
    CONSTRAINT guilds_check CHECK ((((state)::text <> 'ACTIVE'::text) OR (leader_character_id IS NOT NULL))),
    CONSTRAINT guilds_check1 CHECK ((((state)::text = 'DISBANDED'::text) = (disbanded_at IS NOT NULL))),
    CONSTRAINT guilds_recruitment_mode_check CHECK (((recruitment_mode)::text = ANY ((ARRAY['CLOSED'::character varying, 'APPLICATIONS'::character varying])::text[]))),
    CONSTRAINT guilds_state_check CHECK (((state)::text = ANY ((ARRAY['ACTIVE'::character varying, 'DISBANDING'::character varying, 'DISBANDED'::character varying])::text[])))
);


ALTER TABLE public.guilds OWNER TO postgres;

--
-- Name: iap_notification_dedup; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.iap_notification_dedup (
    provider character varying(16) NOT NULL,
    notification_key character varying(160) NOT NULL,
    received_at timestamp with time zone NOT NULL,
    processed_at timestamp with time zone,
    CONSTRAINT iap_notification_dedup_provider_check CHECK (((provider)::text = ANY ((ARRAY['APP_STORE'::character varying, 'GOOGLE_PLAY'::character varying, 'STEAM'::character varying])::text[])))
);


ALTER TABLE public.iap_notification_dedup OWNER TO postgres;

--
-- Name: iap_provider_cursors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.iap_provider_cursors (
    provider character varying(16) NOT NULL,
    cursor_value character varying(64) NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.iap_provider_cursors OWNER TO postgres;

--
-- Name: item_instances; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.item_instances (
    item_instance_id uuid NOT NULL,
    item_id character varying(64) NOT NULL,
    quantity integer NOT NULL,
    effective_binding character varying(24) NOT NULL,
    enhancement_level integer DEFAULT 0 NOT NULL,
    roll_state jsonb DEFAULT '{}'::jsonb NOT NULL,
    content_revision bigint NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT item_instances_effective_binding_check CHECK (((effective_binding)::text = ANY ((ARRAY['UNBOUND'::character varying, 'ACCOUNT_BOUND'::character varying, 'CHARACTER_BOUND'::character varying])::text[]))),
    CONSTRAINT item_instances_enhancement_level_check CHECK (((enhancement_level >= 0) AND (enhancement_level <= 16))),
    CONSTRAINT item_instances_quantity_check CHECK (((quantity > 0) AND (quantity <= 9999)))
);


ALTER TABLE public.item_instances OWNER TO postgres;

--
-- Name: item_locations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.item_locations (
    item_instance_id uuid NOT NULL,
    location_kind character varying(24) NOT NULL,
    character_id uuid,
    guild_id uuid,
    listing_id uuid,
    slot_index integer,
    equip_slot character varying(16),
    loadout_index smallint,
    section character varying(8),
    depositor_character_id uuid,
    depositor_account_id uuid,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT item_locations_check CHECK ((((location_kind)::text <> 'CHARACTER_INVENTORY'::text) OR (slot_index <= 119))),
    CONSTRAINT item_locations_check1 CHECK ((((location_kind)::text <> 'GUILD_STORAGE'::text) OR (slot_index <= 179))),
    CONSTRAINT item_locations_check2 CHECK (((((location_kind)::text = 'CHARACTER_INVENTORY'::text) AND (character_id IS NOT NULL) AND (slot_index IS NOT NULL) AND (guild_id IS NULL) AND (listing_id IS NULL) AND (equip_slot IS NULL) AND (loadout_index IS NULL) AND (section IS NULL) AND (depositor_character_id IS NULL) AND (depositor_account_id IS NULL)) OR (((location_kind)::text = 'EQUIPPED'::text) AND (character_id IS NOT NULL) AND (equip_slot IS NOT NULL) AND (loadout_index IS NOT NULL) AND (slot_index IS NULL) AND (guild_id IS NULL) AND (listing_id IS NULL) AND (section IS NULL) AND (depositor_character_id IS NULL) AND (depositor_account_id IS NULL)) OR (((location_kind)::text = 'BEAST_EQUIPMENT_SLOT'::text) AND (character_id IS NOT NULL) AND (slot_index IS NULL) AND (equip_slot IS NULL) AND (loadout_index IS NULL) AND (guild_id IS NULL) AND (listing_id IS NULL) AND (section IS NULL) AND (depositor_character_id IS NULL) AND (depositor_account_id IS NULL)) OR (((location_kind)::text = 'GUILD_STORAGE'::text) AND (guild_id IS NOT NULL) AND (section IS NOT NULL) AND (slot_index IS NOT NULL) AND (depositor_character_id IS NOT NULL) AND (depositor_account_id IS NOT NULL) AND (character_id IS NULL) AND (listing_id IS NULL) AND (equip_slot IS NULL) AND (loadout_index IS NULL)) OR (((location_kind)::text = 'AUCTION_ESCROW'::text) AND (listing_id IS NOT NULL) AND (character_id IS NULL) AND (guild_id IS NULL) AND (slot_index IS NULL) AND (equip_slot IS NULL) AND (loadout_index IS NULL) AND (section IS NULL) AND (depositor_character_id IS NULL) AND (depositor_account_id IS NULL)))),
    CONSTRAINT item_locations_equip_slot_check CHECK (((equip_slot)::text = ANY ((ARRAY['weapon'::character varying, 'head'::character varying, 'body'::character varying, 'hands'::character varying, 'legs'::character varying, 'feet'::character varying, 'necklace'::character varying, 'ring'::character varying, 'costume'::character varying, 'talisman'::character varying, 'jade'::character varying, 'seal'::character varying, 'relic'::character varying, 'charm'::character varying])::text[]))),
    CONSTRAINT item_locations_loadout_index_check CHECK (((loadout_index >= 1) AND (loadout_index <= 3))),
    CONSTRAINT item_locations_location_kind_check CHECK (((location_kind)::text = ANY ((ARRAY['CHARACTER_INVENTORY'::character varying, 'EQUIPPED'::character varying, 'BEAST_EQUIPMENT_SLOT'::character varying, 'GUILD_STORAGE'::character varying, 'AUCTION_ESCROW'::character varying])::text[]))),
    CONSTRAINT item_locations_section_check CHECK (((section)::text = ANY ((ARRAY['COMMON'::character varying, 'RESERVE'::character varying])::text[]))),
    CONSTRAINT item_locations_slot_index_check CHECK (((slot_index IS NULL) OR (slot_index >= 0)))
);


ALTER TABLE public.item_locations OWNER TO postgres;

--
-- Name: operations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.operations (
    operation_family character varying(48) NOT NULL,
    owner_kind character varying(16) NOT NULL,
    owner_id uuid NOT NULL,
    operation_id uuid NOT NULL,
    request_fingerprint bytea NOT NULL,
    outcome jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone NOT NULL,
    CONSTRAINT operations_owner_kind_check CHECK (((owner_kind)::text = ANY ((ARRAY['ACCOUNT'::character varying, 'CHARACTER'::character varying, 'GUILD'::character varying, 'WORLD'::character varying])::text[]))),
    CONSTRAINT operations_request_fingerprint_check CHECK ((octet_length(request_fingerprint) = 32))
);


ALTER TABLE public.operations OWNER TO postgres;

--
-- Name: operators; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.operators (
    operator_id uuid NOT NULL,
    login_key character varying(32) NOT NULL,
    password_hash text NOT NULL,
    totp_secret_encrypted bytea NOT NULL,
    role character varying(16) NOT NULL,
    status character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    last_login_at timestamp with time zone,
    CONSTRAINT operators_role_check CHECK (((role)::text = ANY ((ARRAY['SUPPORT'::character varying, 'MODERATOR'::character varying, 'ECONOMY'::character varying, 'ADMIN'::character varying])::text[]))),
    CONSTRAINT operators_status_check CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'DISABLED'::character varying])::text[])))
);


ALTER TABLE public.operators OWNER TO postgres;

--
-- Name: pending_erasure_ledger; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pending_erasure_ledger (
    operation_id uuid NOT NULL,
    account_id_hash bytea NOT NULL,
    executed_at timestamp with time zone NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    last_error character varying(256)
);


ALTER TABLE public.pending_erasure_ledger OWNER TO postgres;

--
-- Name: public_boss_schedules; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.public_boss_schedules (
    boss_id text NOT NULL,
    state text NOT NULL,
    public_boss_spawn_generation_id uuid,
    opened_at timestamp with time zone,
    next_spawn_at timestamp with time zone,
    revision bigint NOT NULL,
    CONSTRAINT public_boss_schedules_check CHECK ((((state = 'OPEN'::text) AND (public_boss_spawn_generation_id IS NOT NULL) AND (opened_at IS NOT NULL) AND (next_spawn_at IS NULL)) OR ((state = 'SCHEDULED'::text) AND (public_boss_spawn_generation_id IS NULL) AND (opened_at IS NULL) AND (next_spawn_at IS NOT NULL)))),
    CONSTRAINT public_boss_schedules_state_check CHECK ((state = ANY (ARRAY['SCHEDULED'::text, 'OPEN'::text])))
);


ALTER TABLE public.public_boss_schedules OWNER TO postgres;

--
-- Name: pvp_match_settlements; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pvp_match_settlements (
    pvp_match_id uuid NOT NULL,
    character_id uuid NOT NULL,
    pvp_mode_id character varying(48) NOT NULL,
    season_id integer NOT NULL,
    match_state character varying(12) NOT NULL,
    result character varying(8),
    participation character varying(12) NOT NULL,
    mmr_before integer NOT NULL,
    mmr_after integer NOT NULL,
    season_rating_before integer NOT NULL,
    season_rating_after integer NOT NULL,
    abandon_penalty smallint DEFAULT 0 NOT NULL,
    reward_eligible boolean NOT NULL,
    ranked_bound_utc_date date,
    ranked_bound_slot smallint,
    settlement_operation_id uuid NOT NULL,
    settled_at timestamp with time zone NOT NULL,
    CONSTRAINT pvp_match_settlements_abandon_penalty_check CHECK ((abandon_penalty = ANY (ARRAY[0, 20]))),
    CONSTRAINT pvp_match_settlements_check CHECK ((((match_state)::text = 'VOID'::text) = (result IS NULL))),
    CONSTRAINT pvp_match_settlements_check1 CHECK ((((match_state)::text = 'COMPLETED'::text) OR ((mmr_after = mmr_before) AND (season_rating_after = season_rating_before) AND (NOT reward_eligible) AND (abandon_penalty = 0)))),
    CONSTRAINT pvp_match_settlements_check2 CHECK ((((participation)::text = 'NORMAL'::text) OR (NOT reward_eligible))),
    CONSTRAINT pvp_match_settlements_check3 CHECK (((ranked_bound_slot IS NULL) = (ranked_bound_utc_date IS NULL))),
    CONSTRAINT pvp_match_settlements_check4 CHECK (((ranked_bound_slot IS NULL) OR reward_eligible)),
    CONSTRAINT pvp_match_settlements_match_state_check CHECK (((match_state)::text = ANY ((ARRAY['COMPLETED'::character varying, 'VOID'::character varying])::text[]))),
    CONSTRAINT pvp_match_settlements_participation_check CHECK (((participation)::text = ANY ((ARRAY['NORMAL'::character varying, 'AFK'::character varying, 'ABANDONED'::character varying])::text[]))),
    CONSTRAINT pvp_match_settlements_pvp_mode_id_check CHECK (((pvp_mode_id)::text = ANY ((ARRAY['pvp.mode.ranked_duel'::character varying, 'pvp.mode.five_element_arena'::character varying])::text[]))),
    CONSTRAINT pvp_match_settlements_ranked_bound_slot_check CHECK (((ranked_bound_slot >= 1) AND (ranked_bound_slot <= 5))),
    CONSTRAINT pvp_match_settlements_result_check CHECK (((result)::text = ANY ((ARRAY['WIN'::character varying, 'LOSS'::character varying, 'DRAW'::character varying])::text[])))
);


ALTER TABLE public.pvp_match_settlements OWNER TO postgres;

--
-- Name: pvp_ratings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pvp_ratings (
    character_id uuid NOT NULL,
    pvp_mode_id character varying(48) NOT NULL,
    season_id integer NOT NULL,
    pvp_mmr integer NOT NULL,
    season_rating integer NOT NULL,
    ranked_games_played integer DEFAULT 0 NOT NULL,
    lifetime_mode_games integer DEFAULT 0 NOT NULL,
    wins integer DEFAULT 0 NOT NULL,
    losses integer DEFAULT 0 NOT NULL,
    draws integer DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT pvp_ratings_check CHECK ((((wins + losses) + draws) = ranked_games_played)),
    CONSTRAINT pvp_ratings_draws_check CHECK ((draws >= 0)),
    CONSTRAINT pvp_ratings_lifetime_mode_games_check CHECK ((lifetime_mode_games >= 0)),
    CONSTRAINT pvp_ratings_losses_check CHECK ((losses >= 0)),
    CONSTRAINT pvp_ratings_pvp_mode_id_check CHECK (((pvp_mode_id)::text = ANY ((ARRAY['pvp.mode.ranked_duel'::character varying, 'pvp.mode.five_element_arena'::character varying])::text[]))),
    CONSTRAINT pvp_ratings_ranked_games_played_check CHECK ((ranked_games_played >= 0)),
    CONSTRAINT pvp_ratings_season_id_check CHECK ((season_id >= 0)),
    CONSTRAINT pvp_ratings_season_rating_check CHECK ((season_rating >= 0)),
    CONSTRAINT pvp_ratings_wins_check CHECK ((wins >= 0))
);


ALTER TABLE public.pvp_ratings OWNER TO postgres;

--
-- Name: pvp_sanctions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.pvp_sanctions (
    sanction_id uuid NOT NULL,
    character_id uuid NOT NULL,
    sanction_kind character varying(24) NOT NULL,
    reason character varying(12) NOT NULL,
    source_match_kind character varying(12) NOT NULL,
    source_match_id uuid NOT NULL,
    ladder_step smallint NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone NOT NULL,
    operation_id uuid NOT NULL,
    CONSTRAINT pvp_sanctions_check CHECK ((ends_at = (starts_at +
CASE ladder_step
    WHEN 1 THEN '00:15:00'::interval
    WHEN 2 THEN '00:30:00'::interval
    ELSE '02:00:00'::interval
END))),
    CONSTRAINT pvp_sanctions_ladder_step_check CHECK ((ladder_step = ANY (ARRAY[1, 2, 3]))),
    CONSTRAINT pvp_sanctions_reason_check CHECK (((reason)::text = ANY ((ARRAY['ABANDON'::character varying, 'AFK'::character varying])::text[]))),
    CONSTRAINT pvp_sanctions_sanction_kind_check CHECK (((sanction_kind)::text = 'QUEUE_RESTRICTION'::text)),
    CONSTRAINT pvp_sanctions_source_match_kind_check CHECK (((source_match_kind)::text = ANY ((ARRAY['PVP'::character varying, 'GUILD_WAR'::character varying])::text[])))
);


ALTER TABLE public.pvp_sanctions OWNER TO postgres;

--
-- Name: rate_limit_counters; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.rate_limit_counters (
    key_hash bytea NOT NULL,
    window_seconds integer NOT NULL,
    window_start timestamp with time zone NOT NULL,
    current_count integer NOT NULL,
    previous_count integer DEFAULT 0 NOT NULL,
    CONSTRAINT rate_limit_counters_current_count_check CHECK ((current_count >= 0)),
    CONSTRAINT rate_limit_counters_key_hash_check CHECK ((octet_length(key_hash) = 32)),
    CONSTRAINT rate_limit_counters_previous_count_check CHECK ((previous_count >= 0)),
    CONSTRAINT rate_limit_counters_window_seconds_check CHECK ((window_seconds > 0))
);


ALTER TABLE public.rate_limit_counters OWNER TO postgres;

--
-- Name: region_di_tich_markers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.region_di_tich_markers (
    region_id character varying(64) NOT NULL,
    boss_id character varying(64) NOT NULL,
    last_defeated_utc timestamp with time zone NOT NULL,
    last_defeated_participant_count integer NOT NULL,
    relic_active_in_region boolean NOT NULL,
    CONSTRAINT region_di_tich_markers_last_defeated_participant_count_check CHECK ((last_defeated_participant_count >= 1))
);


ALTER TABLE public.region_di_tich_markers OWNER TO postgres;

--
-- Name: reward_claim_contributions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.reward_claim_contributions (
    source_reward_operation_id uuid NOT NULL,
    owner_character_id uuid NOT NULL,
    reward_slot character varying(64) NOT NULL,
    reward_claim_id uuid NOT NULL,
    quantity_or_amount bigint NOT NULL,
    created_at timestamp with time zone NOT NULL
);


ALTER TABLE public.reward_claim_contributions OWNER TO postgres;

--
-- Name: reward_claim_lines; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.reward_claim_lines (
    reward_claim_id uuid NOT NULL,
    line_no smallint NOT NULL,
    line_kind character varying(16) NOT NULL,
    item_id character varying(64),
    quantity integer,
    effective_binding character varying(24),
    item_state jsonb DEFAULT '{}'::jsonb NOT NULL,
    content_revision bigint,
    currency_id character varying(32),
    amount bigint,
    CONSTRAINT reward_claim_lines_check CHECK (((((line_kind)::text = 'ITEM'::text) AND (item_id IS NOT NULL) AND (quantity > 0) AND (effective_binding IS NOT NULL) AND (content_revision IS NOT NULL) AND (currency_id IS NULL) AND (amount IS NULL)) OR (((line_kind)::text = 'CURRENCY'::text) AND (currency_id IS NOT NULL) AND (amount > 0) AND (item_id IS NULL) AND (quantity IS NULL) AND (effective_binding IS NULL) AND (item_state = '{}'::jsonb)))),
    CONSTRAINT reward_claim_lines_line_kind_check CHECK (((line_kind)::text = ANY ((ARRAY['ITEM'::character varying, 'CURRENCY'::character varying])::text[])))
);


ALTER TABLE public.reward_claim_lines OWNER TO postgres;

--
-- Name: reward_claims; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.reward_claims (
    reward_claim_id uuid NOT NULL,
    owner_character_id uuid NOT NULL,
    source_type character varying(32) NOT NULL,
    source_reference character varying(160) NOT NULL,
    reward_slot character varying(64) NOT NULL,
    claim_kind character varying(16) NOT NULL,
    consolidation_key character varying(128),
    state character varying(16) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    claimed_at timestamp with time zone,
    claim_operation_id uuid,
    revision bigint NOT NULL,
    CONSTRAINT reward_claims_check CHECK ((((claim_kind)::text = 'SINGLE'::text) = (consolidation_key IS NULL))),
    CONSTRAINT reward_claims_claim_kind_check CHECK (((claim_kind)::text = ANY ((ARRAY['SINGLE'::character varying, 'ITEM_CONSOLIDATED'::character varying, 'CURRENCY_AGGREGATE'::character varying])::text[]))),
    CONSTRAINT reward_claims_source_type_check CHECK (((source_type)::text = ANY ((ARRAY['MONSTER'::character varying, 'BOSS'::character varying, 'BOSS_CHEST'::character varying, 'DUNGEON'::character varying, 'QUEST'::character varying, 'WORLD_EVENT'::character varying, 'ATLAS'::character varying, 'FEAT'::character varying, 'LEVEL_MILESTONE'::character varying, 'PVP'::character varying, 'GUILD_WAR'::character varying, 'GUILD'::character varying, 'AUCTION_ESCROW_EXPIRY'::character varying, 'ADMIN_COMPENSATION'::character varying])::text[]))),
    CONSTRAINT reward_claims_state_check CHECK (((state)::text = ANY ((ARRAY['PENDING'::character varying, 'CLAIMING'::character varying, 'CLAIMED'::character varying, 'EXPIRED'::character varying])::text[])))
);


ALTER TABLE public.reward_claims OWNER TO postgres;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: trade_settlement_records; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.trade_settlement_records (
    settlement_id uuid NOT NULL,
    settled_at timestamp with time zone NOT NULL,
    initiator_character_id uuid NOT NULL,
    initiator_account_id uuid NOT NULL,
    counterpart_character_id uuid NOT NULL,
    counterpart_account_id uuid NOT NULL,
    common_sent_by_initiator bigint NOT NULL,
    common_sent_by_counterpart bigint NOT NULL,
    trade_id uuid NOT NULL,
    settlement_operation_id uuid NOT NULL,
    CONSTRAINT trade_settlement_records_common_sent_by_counterpart_check CHECK ((common_sent_by_counterpart >= 0)),
    CONSTRAINT trade_settlement_records_common_sent_by_initiator_check CHECK ((common_sent_by_initiator >= 0))
);


ALTER TABLE public.trade_settlement_records OWNER TO postgres;

--
-- Name: world_consequence_relics; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.world_consequence_relics (
    map_id character varying(64) NOT NULL,
    channel_id smallint NOT NULL,
    relic_id character varying(64) NOT NULL,
    source_id character varying(64) NOT NULL,
    relic_active boolean NOT NULL,
    buff_effect_id character varying(64) NOT NULL,
    spawned_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT world_consequence_relics_channel_id_check CHECK (((channel_id >= 1) AND (channel_id <= 30))),
    CONSTRAINT world_consequence_relics_check CHECK ((expires_at = (spawned_at + '01:00:00'::interval)))
);


ALTER TABLE public.world_consequence_relics OWNER TO postgres;

--
-- Name: account_cosmetic_entitlements account_cosmetic_entitlements_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_cosmetic_entitlements
    ADD CONSTRAINT account_cosmetic_entitlements_pkey PRIMARY KEY (account_id, cosmetic_id, entitlement_id);


--
-- Name: account_entitlement_claims account_entitlement_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_entitlement_claims
    ADD CONSTRAINT account_entitlement_claims_pkey PRIMARY KEY (account_entitlement_id, character_id, reward_tier_id);


--
-- Name: account_iap_entitlements account_iap_entitlements_composite_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_iap_entitlements
    ADD CONSTRAINT account_iap_entitlements_composite_unique UNIQUE (entitlement_id, account_id, entitlement_type);


--
-- Name: account_iap_entitlements account_iap_entitlements_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_iap_entitlements
    ADD CONSTRAINT account_iap_entitlements_pkey PRIMARY KEY (entitlement_id);


--
-- Name: account_iap_entitlements account_iap_entitlements_platform_receipt_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_iap_entitlements
    ADD CONSTRAINT account_iap_entitlements_platform_receipt_key UNIQUE (platform_receipt);


--
-- Name: account_identities account_identities_account_id_provider_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_identities
    ADD CONSTRAINT account_identities_account_id_provider_id_key UNIQUE (account_id, provider_id);


--
-- Name: account_identities account_identities_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_identities
    ADD CONSTRAINT account_identities_pkey PRIMARY KEY (provider_id, provider_subject);


--
-- Name: account_login_history account_login_history_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_login_history
    ADD CONSTRAINT account_login_history_pkey PRIMARY KEY (account_id, observed_at);


--
-- Name: account_password_credentials account_password_credentials_email_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_password_credentials
    ADD CONSTRAINT account_password_credentials_email_key_key UNIQUE (email_key);


--
-- Name: account_password_credentials account_password_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_password_credentials
    ADD CONSTRAINT account_password_credentials_pkey PRIMARY KEY (account_id);


--
-- Name: account_password_credentials account_password_credentials_username_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_password_credentials
    ADD CONSTRAINT account_password_credentials_username_key_key UNIQUE (username_key);


--
-- Name: account_refund_consumed_events account_refund_consumed_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_refund_consumed_events
    ADD CONSTRAINT account_refund_consumed_events_pkey PRIMARY KEY (event_id);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (account_id);


--
-- Name: atlas_milestones atlas_milestones_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.atlas_milestones
    ADD CONSTRAINT atlas_milestones_pkey PRIMARY KEY (character_id, milestone_id);


--
-- Name: auction_listings auction_listings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_listings
    ADD CONSTRAINT auction_listings_pkey PRIMARY KEY (listing_id);


--
-- Name: auction_proceeds auction_proceeds_listing_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_listing_id_key UNIQUE (listing_id);


--
-- Name: auction_proceeds auction_proceeds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_pkey PRIMARY KEY (proceeds_id);


--
-- Name: audit_events audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (audit_event_id);


--
-- Name: auth_failure_backoff auth_failure_backoff_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_failure_backoff
    ADD CONSTRAINT auth_failure_backoff_pkey PRIMARY KEY (key_hash);


--
-- Name: auth_refresh_credentials auth_refresh_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_refresh_credentials
    ADD CONSTRAINT auth_refresh_credentials_pkey PRIMARY KEY (credential_hash);


--
-- Name: auth_refresh_credentials auth_refresh_credentials_session_family_id_generation_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_refresh_credentials
    ADD CONSTRAINT auth_refresh_credentials_session_family_id_generation_key UNIQUE (session_family_id, generation);


--
-- Name: auth_revocations auth_revocations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_revocations
    ADD CONSTRAINT auth_revocations_pkey PRIMARY KEY (revocation_id);


--
-- Name: auth_session_families auth_session_families_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_session_families
    ADD CONSTRAINT auth_session_families_pkey PRIMARY KEY (session_family_id);


--
-- Name: beast_equipment_locations beast_equipment_locations_item_instance_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.beast_equipment_locations
    ADD CONSTRAINT beast_equipment_locations_item_instance_id_key UNIQUE (item_instance_id);


--
-- Name: beast_equipment_locations beast_equipment_locations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.beast_equipment_locations
    ADD CONSTRAINT beast_equipment_locations_pkey PRIMARY KEY (character_id, beast_id, slot_id);


--
-- Name: blocks blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (blocker_character_id, blocked_character_id);


--
-- Name: boss_chest_eligibility boss_chest_eligibility_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.boss_chest_eligibility
    ADD CONSTRAINT boss_chest_eligibility_pkey PRIMARY KEY (character_id, public_boss_spawn_generation_id);


--
-- Name: character_atlas character_atlas_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_atlas
    ADD CONSTRAINT character_atlas_pkey PRIMARY KEY (character_id, atlas_page_id, tier);


--
-- Name: character_atlas character_atlas_reward_operation_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_atlas
    ADD CONSTRAINT character_atlas_reward_operation_id_key UNIQUE (reward_operation_id);


--
-- Name: character_beast_food_daily character_beast_food_daily_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_beast_food_daily
    ADD CONSTRAINT character_beast_food_daily_pkey PRIMARY KEY (character_id, utc_date);


--
-- Name: character_beasts character_beasts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_beasts
    ADD CONSTRAINT character_beasts_pkey PRIMARY KEY (character_id, beast_id);


--
-- Name: character_chivalry character_chivalry_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_chivalry
    ADD CONSTRAINT character_chivalry_pkey PRIMARY KEY (character_id);


--
-- Name: character_cosmetic_entitlements character_cosmetic_entitlements_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_cosmetic_entitlements
    ADD CONSTRAINT character_cosmetic_entitlements_pkey PRIMARY KEY (character_id, cosmetic_id, source_ref);


--
-- Name: character_cosmetic_equips character_cosmetic_equips_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_cosmetic_equips
    ADD CONSTRAINT character_cosmetic_equips_pkey PRIMARY KEY (character_id, slot);


--
-- Name: character_currencies character_currencies_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_currencies
    ADD CONSTRAINT character_currencies_pkey PRIMARY KEY (character_id, currency_id);


--
-- Name: character_feat_milestones character_feat_milestones_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_feat_milestones
    ADD CONSTRAINT character_feat_milestones_pkey PRIMARY KEY (character_id, feat_id, milestone_threshold);


--
-- Name: character_feat_milestones character_feat_milestones_reward_operation_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_feat_milestones
    ADD CONSTRAINT character_feat_milestones_reward_operation_id_key UNIQUE (reward_operation_id);


--
-- Name: character_feats character_feats_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_feats
    ADD CONSTRAINT character_feats_pkey PRIMARY KEY (character_id, feat_id);


--
-- Name: character_inventories character_inventories_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_inventories
    ADD CONSTRAINT character_inventories_pkey PRIMARY KEY (character_id);


--
-- Name: character_loadouts character_loadouts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_loadouts
    ADD CONSTRAINT character_loadouts_pkey PRIMARY KEY (character_id, loadout_index);


--
-- Name: character_soul_resonance character_soul_resonance_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_soul_resonance
    ADD CONSTRAINT character_soul_resonance_pkey PRIMARY KEY (character_id, soul_id);


--
-- Name: character_souls character_souls_contracted_item_instance_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_souls
    ADD CONSTRAINT character_souls_contracted_item_instance_id_key UNIQUE (contracted_item_instance_id);


--
-- Name: character_souls character_souls_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_souls
    ADD CONSTRAINT character_souls_pkey PRIMARY KEY (soul_instance_id);


--
-- Name: characters characters_id_account_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.characters
    ADD CONSTRAINT characters_id_account_unique UNIQUE (character_id, account_id);


--
-- Name: characters characters_name_key_unique; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.characters
    ADD CONSTRAINT characters_name_key_unique UNIQUE (name_key);


--
-- Name: characters characters_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.characters
    ADD CONSTRAINT characters_pkey PRIMARY KEY (character_id);


--
-- Name: chat_messages chat_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_pkey PRIMARY KEY (message_id);


--
-- Name: economy_account_daily_rollups economy_account_daily_rollups_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.economy_account_daily_rollups
    ADD CONSTRAINT economy_account_daily_rollups_pkey PRIMARY KEY (account_id, utc_day);


--
-- Name: economy_character_daily_rollups economy_character_daily_rollups_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.economy_character_daily_rollups
    ADD CONSTRAINT economy_character_daily_rollups_pkey PRIMARY KEY (character_id, utc_day);


--
-- Name: enhancement_pity enhancement_pity_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.enhancement_pity
    ADD CONSTRAINT enhancement_pity_pkey PRIMARY KEY (item_instance_id, target_level);


--
-- Name: friend_requests friend_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friend_requests
    ADD CONSTRAINT friend_requests_pkey PRIMARY KEY (friend_request_id);


--
-- Name: friends friends_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_pkey PRIMARY KEY (character_low_id, character_high_id);


--
-- Name: guild_applications guild_applications_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_applications
    ADD CONSTRAINT guild_applications_pkey PRIMARY KEY (application_id);


--
-- Name: guild_blessing_votes guild_blessing_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_blessing_votes
    ADD CONSTRAINT guild_blessing_votes_pkey PRIMARY KEY (guild_id, cycle_id, account_id);


--
-- Name: guild_invites guild_invites_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_invites
    ADD CONSTRAINT guild_invites_pkey PRIMARY KEY (invite_id);


--
-- Name: guild_member_contributions guild_member_contributions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_member_contributions
    ADD CONSTRAINT guild_member_contributions_pkey PRIMARY KEY (guild_id, character_id);


--
-- Name: guild_memberships guild_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_memberships
    ADD CONSTRAINT guild_memberships_pkey PRIMARY KEY (character_id);


--
-- Name: guild_progression guild_progression_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_progression
    ADD CONSTRAINT guild_progression_pkey PRIMARY KEY (guild_id);


--
-- Name: guild_ritual_cycles guild_ritual_cycles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_ritual_cycles
    ADD CONSTRAINT guild_ritual_cycles_pkey PRIMARY KEY (guild_id, cycle_id);


--
-- Name: guild_stone_category_completions guild_stone_category_completions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_stone_category_completions
    ADD CONSTRAINT guild_stone_category_completions_pkey PRIMARY KEY (guild_id, category_id, season_number);


--
-- Name: guild_storage_audit guild_storage_audit_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_audit
    ADD CONSTRAINT guild_storage_audit_pkey PRIMARY KEY (audit_id);


--
-- Name: guild_storage_claims guild_storage_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_claims
    ADD CONSTRAINT guild_storage_claims_pkey PRIMARY KEY (claim_id);


--
-- Name: guild_war_ratings guild_war_ratings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_war_ratings
    ADD CONSTRAINT guild_war_ratings_pkey PRIMARY KEY (guild_id, season_id);


--
-- Name: guild_war_settlements guild_war_settlements_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_war_settlements
    ADD CONSTRAINT guild_war_settlements_pkey PRIMARY KEY (guild_war_match_id, settlement_type, recipient_id);


--
-- Name: guilds guilds_name_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_name_key_key UNIQUE (name_key);


--
-- Name: guilds guilds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_pkey PRIMARY KEY (guild_id);


--
-- Name: iap_notification_dedup iap_notification_dedup_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.iap_notification_dedup
    ADD CONSTRAINT iap_notification_dedup_pkey PRIMARY KEY (provider, notification_key);


--
-- Name: iap_provider_cursors iap_provider_cursors_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.iap_provider_cursors
    ADD CONSTRAINT iap_provider_cursors_pkey PRIMARY KEY (provider);


--
-- Name: item_instances item_instances_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_instances
    ADD CONSTRAINT item_instances_pkey PRIMARY KEY (item_instance_id);


--
-- Name: item_locations item_locations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_pkey PRIMARY KEY (item_instance_id);


--
-- Name: operations operations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.operations
    ADD CONSTRAINT operations_pkey PRIMARY KEY (operation_family, owner_id, operation_id);


--
-- Name: operators operators_login_key_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.operators
    ADD CONSTRAINT operators_login_key_key UNIQUE (login_key);


--
-- Name: operators operators_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.operators
    ADD CONSTRAINT operators_pkey PRIMARY KEY (operator_id);


--
-- Name: pending_erasure_ledger pending_erasure_ledger_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pending_erasure_ledger
    ADD CONSTRAINT pending_erasure_ledger_pkey PRIMARY KEY (operation_id);


--
-- Name: public_boss_schedules public_boss_schedules_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.public_boss_schedules
    ADD CONSTRAINT public_boss_schedules_pkey PRIMARY KEY (boss_id);


--
-- Name: pvp_match_settlements pvp_match_settlements_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_match_settlements
    ADD CONSTRAINT pvp_match_settlements_pkey PRIMARY KEY (pvp_match_id, character_id);


--
-- Name: pvp_ratings pvp_ratings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_ratings
    ADD CONSTRAINT pvp_ratings_pkey PRIMARY KEY (character_id, pvp_mode_id, season_id);


--
-- Name: pvp_sanctions pvp_sanctions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_sanctions
    ADD CONSTRAINT pvp_sanctions_pkey PRIMARY KEY (sanction_id);


--
-- Name: rate_limit_counters rate_limit_counters_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rate_limit_counters
    ADD CONSTRAINT rate_limit_counters_pkey PRIMARY KEY (key_hash);


--
-- Name: region_di_tich_markers region_di_tich_markers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.region_di_tich_markers
    ADD CONSTRAINT region_di_tich_markers_pkey PRIMARY KEY (region_id, boss_id);


--
-- Name: reward_claim_contributions reward_claim_contributions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claim_contributions
    ADD CONSTRAINT reward_claim_contributions_pkey PRIMARY KEY (source_reward_operation_id, owner_character_id, reward_slot);


--
-- Name: reward_claim_lines reward_claim_lines_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claim_lines
    ADD CONSTRAINT reward_claim_lines_pkey PRIMARY KEY (reward_claim_id, line_no);


--
-- Name: reward_claims reward_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claims
    ADD CONSTRAINT reward_claims_pkey PRIMARY KEY (reward_claim_id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: trade_settlement_records trade_settlement_records_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_pkey PRIMARY KEY (settlement_id);


--
-- Name: trade_settlement_records trade_settlement_records_trade_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_trade_id_key UNIQUE (trade_id);


--
-- Name: world_consequence_relics world_consequence_relics_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.world_consequence_relics
    ADD CONSTRAINT world_consequence_relics_pkey PRIMARY KEY (map_id, channel_id, relic_id);


--
-- Name: account_iap_entitlements_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX account_iap_entitlements_account_idx ON public.account_iap_entitlements USING btree (account_id);


--
-- Name: account_iap_entitlements_pending_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX account_iap_entitlements_pending_idx ON public.account_iap_entitlements USING btree (grant_state, created_at) WHERE ((grant_state)::text = 'PENDING'::text);


--
-- Name: account_iap_entitlements_season_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX account_iap_entitlements_season_unique ON public.account_iap_entitlements USING btree (account_id, season_number) WHERE ((season_number IS NOT NULL) AND ((grant_state)::text = ANY ((ARRAY['PENDING'::character varying, 'GRANTED'::character varying])::text[])) AND (account_id <> '00000000-0000-0000-0000-000000000001'::uuid));


--
-- Name: account_identities_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX account_identities_account_idx ON public.account_identities USING btree (account_id);


--
-- Name: account_login_history_observed_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX account_login_history_observed_idx ON public.account_login_history USING btree (observed_at);


--
-- Name: account_refund_consumed_events_window_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX account_refund_consumed_events_window_idx ON public.account_refund_consumed_events USING btree (account_id, occurred_at);


--
-- Name: auction_listings_active_expiry_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_listings_active_expiry_idx ON public.auction_listings USING btree (expires_at) WHERE ((state)::text = 'ACTIVE'::text);


--
-- Name: auction_listings_active_search_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_listings_active_search_idx ON public.auction_listings USING btree (item_id, price_common, listing_id) WHERE ((state)::text = 'ACTIVE'::text);


--
-- Name: auction_listings_item_escrow_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX auction_listings_item_escrow_unique ON public.auction_listings USING btree (item_instance_id) WHERE ((state)::text = ANY ((ARRAY['ACTIVE'::character varying, 'CANCELLED'::character varying, 'EXPIRED'::character varying])::text[]));


--
-- Name: auction_listings_seller_state_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_listings_seller_state_idx ON public.auction_listings USING btree (seller_character_id, state);


--
-- Name: auction_proceeds_buyer_window_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_proceeds_buyer_window_idx ON public.auction_proceeds USING btree (buyer_account_id, settled_at);


--
-- Name: auction_proceeds_pending_seller_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_proceeds_pending_seller_idx ON public.auction_proceeds USING btree (seller_character_id) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: auction_proceeds_seller_window_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auction_proceeds_seller_window_idx ON public.auction_proceeds USING btree (seller_account_id, settled_at);


--
-- Name: audit_events_occurred_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX audit_events_occurred_idx ON public.audit_events USING btree (occurred_at);


--
-- Name: audit_events_subject_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX audit_events_subject_account_idx ON public.audit_events USING btree (subject_account_id, occurred_at);


--
-- Name: audit_events_subject_character_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX audit_events_subject_character_idx ON public.audit_events USING btree (subject_character_id, occurred_at);


--
-- Name: auth_revocations_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auth_revocations_account_idx ON public.auth_revocations USING btree (account_id);


--
-- Name: auth_session_families_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auth_session_families_account_idx ON public.auth_session_families USING btree (account_id);


--
-- Name: auth_session_families_expires_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX auth_session_families_expires_idx ON public.auth_session_families USING btree (expires_at);


--
-- Name: blocks_blocked_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX blocks_blocked_idx ON public.blocks USING btree (blocked_character_id);


--
-- Name: boss_chest_eligibility_copy_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX boss_chest_eligibility_copy_idx ON public.boss_chest_eligibility USING btree (public_boss_spawn_generation_id, copy_map_id, copy_channel_id);


--
-- Name: character_beasts_active_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX character_beasts_active_unique ON public.character_beasts USING btree (character_id) WHERE is_active;


--
-- Name: character_cosmetic_entitlements_source_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX character_cosmetic_entitlements_source_idx ON public.character_cosmetic_entitlements USING btree (source_entitlement_id) WHERE (source_entitlement_id IS NOT NULL);


--
-- Name: character_loadouts_active_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX character_loadouts_active_unique ON public.character_loadouts USING btree (character_id) WHERE ((role)::text = 'ACTIVE'::text);


--
-- Name: characters_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX characters_account_idx ON public.characters USING btree (account_id);


--
-- Name: chat_messages_created_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chat_messages_created_idx ON public.chat_messages USING btree (created_at);


--
-- Name: chat_messages_sender_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chat_messages_sender_idx ON public.chat_messages USING btree (sender_account_id, created_at);


--
-- Name: friend_requests_pending_expiry_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX friend_requests_pending_expiry_idx ON public.friend_requests USING btree (expires_at) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: friend_requests_pending_pair_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX friend_requests_pending_pair_unique ON public.friend_requests USING btree (LEAST(requester_character_id, target_character_id), GREATEST(requester_character_id, target_character_id)) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: friend_requests_requester_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX friend_requests_requester_idx ON public.friend_requests USING btree (requester_character_id, state);


--
-- Name: friend_requests_target_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX friend_requests_target_idx ON public.friend_requests USING btree (target_character_id, state);


--
-- Name: friends_high_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX friends_high_idx ON public.friends USING btree (character_high_id);


--
-- Name: guild_applications_applicant_pending_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_applications_applicant_pending_idx ON public.guild_applications USING btree (applicant_character_id) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: guild_applications_pending_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX guild_applications_pending_unique ON public.guild_applications USING btree (guild_id, applicant_character_id) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: guild_invites_pending_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX guild_invites_pending_unique ON public.guild_invites USING btree (guild_id, target_character_id) WHERE ((state)::text = 'PENDING'::text);


--
-- Name: guild_memberships_guild_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_memberships_guild_idx ON public.guild_memberships USING btree (guild_id);


--
-- Name: guild_memberships_leader_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX guild_memberships_leader_unique ON public.guild_memberships USING btree (guild_id) WHERE ((role)::text = 'LEADER'::text);


--
-- Name: guild_storage_audit_guild_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_storage_audit_guild_idx ON public.guild_storage_audit USING btree (guild_id, occurred_at);


--
-- Name: guild_storage_claims_guild_state_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_storage_claims_guild_state_idx ON public.guild_storage_claims USING btree (guild_id, state);


--
-- Name: guild_war_ratings_leaderboard_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_war_ratings_leaderboard_idx ON public.guild_war_ratings USING btree (season_id, guild_war_mmr DESC, guild_war_wins DESC, guild_war_games_played);


--
-- Name: guild_war_settlements_bound_slot_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX guild_war_settlements_bound_slot_unique ON public.guild_war_settlements USING btree (recipient_id, bound_week_monday, bound_slot) WHERE (bound_slot IS NOT NULL);


--
-- Name: guild_war_settlements_participation_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX guild_war_settlements_participation_idx ON public.guild_war_settlements USING btree (recipient_id, season_id, guild_id) WHERE (((settlement_type)::text = 'SEASON_PARTICIPATION'::text) AND ((match_state)::text = 'COMPLETED'::text));


--
-- Name: iap_notification_dedup_received_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX iap_notification_dedup_received_idx ON public.iap_notification_dedup USING btree (received_at);


--
-- Name: item_locations_character_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX item_locations_character_idx ON public.item_locations USING btree (character_id) WHERE (character_id IS NOT NULL);


--
-- Name: item_locations_equip_slot_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX item_locations_equip_slot_unique ON public.item_locations USING btree (character_id, loadout_index, equip_slot) WHERE ((location_kind)::text = 'EQUIPPED'::text);


--
-- Name: item_locations_escrow_listing_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX item_locations_escrow_listing_unique ON public.item_locations USING btree (listing_id) WHERE ((location_kind)::text = 'AUCTION_ESCROW'::text);


--
-- Name: item_locations_guild_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX item_locations_guild_idx ON public.item_locations USING btree (guild_id) WHERE (guild_id IS NOT NULL);


--
-- Name: item_locations_guild_slot_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX item_locations_guild_slot_unique ON public.item_locations USING btree (guild_id, section, slot_index) WHERE ((location_kind)::text = 'GUILD_STORAGE'::text);


--
-- Name: item_locations_inventory_slot_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX item_locations_inventory_slot_unique ON public.item_locations USING btree (character_id, slot_index) WHERE ((location_kind)::text = 'CHARACTER_INVENTORY'::text);


--
-- Name: operations_completed_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX operations_completed_idx ON public.operations USING btree (completed_at);


--
-- Name: pvp_match_settlements_bound_slot_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX pvp_match_settlements_bound_slot_unique ON public.pvp_match_settlements USING btree (character_id, ranked_bound_utc_date, ranked_bound_slot) WHERE (ranked_bound_slot IS NOT NULL);


--
-- Name: pvp_match_settlements_character_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pvp_match_settlements_character_idx ON public.pvp_match_settlements USING btree (character_id, settled_at);


--
-- Name: pvp_ratings_leaderboard_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pvp_ratings_leaderboard_idx ON public.pvp_ratings USING btree (pvp_mode_id, season_id, season_rating DESC);


--
-- Name: pvp_sanctions_character_ends_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pvp_sanctions_character_ends_idx ON public.pvp_sanctions USING btree (character_id, ends_at);


--
-- Name: pvp_sanctions_character_starts_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX pvp_sanctions_character_starts_idx ON public.pvp_sanctions USING btree (character_id, starts_at);


--
-- Name: pvp_sanctions_source_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX pvp_sanctions_source_unique ON public.pvp_sanctions USING btree (character_id, source_match_kind, source_match_id);


--
-- Name: reward_claims_owner_state_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX reward_claims_owner_state_idx ON public.reward_claims USING btree (owner_character_id, state);


--
-- Name: reward_claims_pending_consolidation_unique; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX reward_claims_pending_consolidation_unique ON public.reward_claims USING btree (owner_character_id, claim_kind, consolidation_key) WHERE (((state)::text = 'PENDING'::text) AND (consolidation_key IS NOT NULL));


--
-- Name: trade_settlement_records_counterpart_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX trade_settlement_records_counterpart_account_idx ON public.trade_settlement_records USING btree (counterpart_account_id, settled_at);


--
-- Name: trade_settlement_records_counterpart_character_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX trade_settlement_records_counterpart_character_idx ON public.trade_settlement_records USING btree (counterpart_character_id, settled_at);


--
-- Name: trade_settlement_records_initiator_account_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX trade_settlement_records_initiator_account_idx ON public.trade_settlement_records USING btree (initiator_account_id, settled_at);


--
-- Name: trade_settlement_records_initiator_character_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX trade_settlement_records_initiator_character_idx ON public.trade_settlement_records USING btree (initiator_character_id, settled_at);


--
-- Name: world_consequence_relics_active_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX world_consequence_relics_active_idx ON public.world_consequence_relics USING btree (relic_id, expires_at) WHERE relic_active;


--
-- Name: world_consequence_relics_sweep_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX world_consequence_relics_sweep_idx ON public.world_consequence_relics USING btree (expires_at) WHERE relic_active;


--
-- Name: account_cosmetic_entitlements account_cosmetic_entitlements_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_cosmetic_entitlements
    ADD CONSTRAINT account_cosmetic_entitlements_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_cosmetic_entitlements account_cosmetic_entitlements_entitlement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_cosmetic_entitlements
    ADD CONSTRAINT account_cosmetic_entitlements_entitlement_id_fkey FOREIGN KEY (entitlement_id) REFERENCES public.account_iap_entitlements(entitlement_id) ON DELETE RESTRICT;


--
-- Name: account_entitlement_claims account_entitlement_claims_account_entitlement_id_account__fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_entitlement_claims
    ADD CONSTRAINT account_entitlement_claims_account_entitlement_id_account__fkey FOREIGN KEY (account_entitlement_id, account_id, entitlement_type) REFERENCES public.account_iap_entitlements(entitlement_id, account_id, entitlement_type) DEFERRABLE;


--
-- Name: account_entitlement_claims account_entitlement_claims_character_id_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_entitlement_claims
    ADD CONSTRAINT account_entitlement_claims_character_id_account_id_fkey FOREIGN KEY (character_id, account_id) REFERENCES public.characters(character_id, account_id) DEFERRABLE;


--
-- Name: account_iap_entitlements account_iap_entitlements_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_iap_entitlements
    ADD CONSTRAINT account_iap_entitlements_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_identities account_identities_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_identities
    ADD CONSTRAINT account_identities_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_login_history account_login_history_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_login_history
    ADD CONSTRAINT account_login_history_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_password_credentials account_password_credentials_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_password_credentials
    ADD CONSTRAINT account_password_credentials_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_refund_consumed_events account_refund_consumed_events_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_refund_consumed_events
    ADD CONSTRAINT account_refund_consumed_events_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: account_refund_consumed_events account_refund_consumed_events_entitlement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.account_refund_consumed_events
    ADD CONSTRAINT account_refund_consumed_events_entitlement_id_fkey FOREIGN KEY (entitlement_id) REFERENCES public.account_iap_entitlements(entitlement_id) ON DELETE RESTRICT;


--
-- Name: atlas_milestones atlas_milestones_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.atlas_milestones
    ADD CONSTRAINT atlas_milestones_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: auction_listings auction_listings_buyer_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_listings
    ADD CONSTRAINT auction_listings_buyer_character_id_fkey FOREIGN KEY (buyer_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: auction_listings auction_listings_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_listings
    ADD CONSTRAINT auction_listings_item_instance_id_fkey FOREIGN KEY (item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: auction_listings auction_listings_seller_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_listings
    ADD CONSTRAINT auction_listings_seller_account_id_fkey FOREIGN KEY (seller_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: auction_listings auction_listings_seller_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_listings
    ADD CONSTRAINT auction_listings_seller_character_id_fkey FOREIGN KEY (seller_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: auction_proceeds auction_proceeds_buyer_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_buyer_account_id_fkey FOREIGN KEY (buyer_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: auction_proceeds auction_proceeds_buyer_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_buyer_character_id_fkey FOREIGN KEY (buyer_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: auction_proceeds auction_proceeds_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_listing_id_fkey FOREIGN KEY (listing_id) REFERENCES public.auction_listings(listing_id) ON DELETE RESTRICT;


--
-- Name: auction_proceeds auction_proceeds_seller_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_seller_account_id_fkey FOREIGN KEY (seller_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: auction_proceeds auction_proceeds_seller_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auction_proceeds
    ADD CONSTRAINT auction_proceeds_seller_character_id_fkey FOREIGN KEY (seller_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: auth_refresh_credentials auth_refresh_credentials_session_family_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_refresh_credentials
    ADD CONSTRAINT auth_refresh_credentials_session_family_id_fkey FOREIGN KEY (session_family_id) REFERENCES public.auth_session_families(session_family_id) ON DELETE CASCADE;


--
-- Name: auth_session_families auth_session_families_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.auth_session_families
    ADD CONSTRAINT auth_session_families_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: beast_equipment_locations beast_equipment_locations_character_id_beast_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.beast_equipment_locations
    ADD CONSTRAINT beast_equipment_locations_character_id_beast_id_fkey FOREIGN KEY (character_id, beast_id) REFERENCES public.character_beasts(character_id, beast_id) ON DELETE RESTRICT;


--
-- Name: beast_equipment_locations beast_equipment_locations_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.beast_equipment_locations
    ADD CONSTRAINT beast_equipment_locations_item_instance_id_fkey FOREIGN KEY (item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: blocks blocks_blocked_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_blocked_character_id_fkey FOREIGN KEY (blocked_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: blocks blocks_blocker_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_blocker_character_id_fkey FOREIGN KEY (blocker_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: boss_chest_eligibility boss_chest_eligibility_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.boss_chest_eligibility
    ADD CONSTRAINT boss_chest_eligibility_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_atlas character_atlas_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_atlas
    ADD CONSTRAINT character_atlas_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_beast_food_daily character_beast_food_daily_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_beast_food_daily
    ADD CONSTRAINT character_beast_food_daily_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_beasts character_beasts_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_beasts
    ADD CONSTRAINT character_beasts_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_chivalry character_chivalry_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_chivalry
    ADD CONSTRAINT character_chivalry_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_cosmetic_entitlements character_cosmetic_entitlements_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_cosmetic_entitlements
    ADD CONSTRAINT character_cosmetic_entitlements_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_cosmetic_entitlements character_cosmetic_entitlements_source_entitlement_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_cosmetic_entitlements
    ADD CONSTRAINT character_cosmetic_entitlements_source_entitlement_id_fkey FOREIGN KEY (source_entitlement_id) REFERENCES public.account_iap_entitlements(entitlement_id) ON DELETE RESTRICT;


--
-- Name: character_cosmetic_equips character_cosmetic_equips_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_cosmetic_equips
    ADD CONSTRAINT character_cosmetic_equips_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_currencies character_currencies_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_currencies
    ADD CONSTRAINT character_currencies_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_feat_milestones character_feat_milestones_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_feat_milestones
    ADD CONSTRAINT character_feat_milestones_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_feats character_feats_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_feats
    ADD CONSTRAINT character_feats_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_inventories character_inventories_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_inventories
    ADD CONSTRAINT character_inventories_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_loadouts character_loadouts_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_loadouts
    ADD CONSTRAINT character_loadouts_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_soul_resonance character_soul_resonance_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_soul_resonance
    ADD CONSTRAINT character_soul_resonance_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_souls character_souls_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_souls
    ADD CONSTRAINT character_souls_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: character_souls character_souls_contracted_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.character_souls
    ADD CONSTRAINT character_souls_contracted_item_instance_id_fkey FOREIGN KEY (contracted_item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: characters characters_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.characters
    ADD CONSTRAINT characters_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: economy_account_daily_rollups economy_account_daily_rollups_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.economy_account_daily_rollups
    ADD CONSTRAINT economy_account_daily_rollups_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: economy_character_daily_rollups economy_character_daily_rollups_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.economy_character_daily_rollups
    ADD CONSTRAINT economy_character_daily_rollups_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: enhancement_pity enhancement_pity_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.enhancement_pity
    ADD CONSTRAINT enhancement_pity_item_instance_id_fkey FOREIGN KEY (item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: friend_requests friend_requests_requester_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friend_requests
    ADD CONSTRAINT friend_requests_requester_character_id_fkey FOREIGN KEY (requester_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: friend_requests friend_requests_target_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friend_requests
    ADD CONSTRAINT friend_requests_target_character_id_fkey FOREIGN KEY (target_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: friends friends_character_high_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_character_high_id_fkey FOREIGN KEY (character_high_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: friends friends_character_low_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_character_low_id_fkey FOREIGN KEY (character_low_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_applications guild_applications_applicant_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_applications
    ADD CONSTRAINT guild_applications_applicant_character_id_fkey FOREIGN KEY (applicant_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_applications guild_applications_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_applications
    ADD CONSTRAINT guild_applications_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_applications guild_applications_resolver_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_applications
    ADD CONSTRAINT guild_applications_resolver_character_id_fkey FOREIGN KEY (resolver_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_blessing_votes guild_blessing_votes_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_blessing_votes
    ADD CONSTRAINT guild_blessing_votes_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: guild_blessing_votes guild_blessing_votes_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_blessing_votes
    ADD CONSTRAINT guild_blessing_votes_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_blessing_votes guild_blessing_votes_guild_id_cycle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_blessing_votes
    ADD CONSTRAINT guild_blessing_votes_guild_id_cycle_id_fkey FOREIGN KEY (guild_id, cycle_id) REFERENCES public.guild_ritual_cycles(guild_id, cycle_id) ON DELETE RESTRICT;


--
-- Name: guild_blessing_votes guild_blessing_votes_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_blessing_votes
    ADD CONSTRAINT guild_blessing_votes_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_invites guild_invites_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_invites
    ADD CONSTRAINT guild_invites_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_invites guild_invites_inviter_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_invites
    ADD CONSTRAINT guild_invites_inviter_character_id_fkey FOREIGN KEY (inviter_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_invites guild_invites_target_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_invites
    ADD CONSTRAINT guild_invites_target_character_id_fkey FOREIGN KEY (target_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_member_contributions guild_member_contributions_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_member_contributions
    ADD CONSTRAINT guild_member_contributions_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_member_contributions guild_member_contributions_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_member_contributions
    ADD CONSTRAINT guild_member_contributions_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_memberships guild_memberships_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_memberships
    ADD CONSTRAINT guild_memberships_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_memberships guild_memberships_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_memberships
    ADD CONSTRAINT guild_memberships_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_progression guild_progression_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_progression
    ADD CONSTRAINT guild_progression_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_ritual_cycles guild_ritual_cycles_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_ritual_cycles
    ADD CONSTRAINT guild_ritual_cycles_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_stone_category_completions guild_stone_category_completions_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_stone_category_completions
    ADD CONSTRAINT guild_stone_category_completions_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_audit guild_storage_audit_actor_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_audit
    ADD CONSTRAINT guild_storage_audit_actor_character_id_fkey FOREIGN KEY (actor_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_audit guild_storage_audit_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_audit
    ADD CONSTRAINT guild_storage_audit_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_audit guild_storage_audit_receiver_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_audit
    ADD CONSTRAINT guild_storage_audit_receiver_character_id_fkey FOREIGN KEY (receiver_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_claims guild_storage_claims_approver_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_claims
    ADD CONSTRAINT guild_storage_claims_approver_character_id_fkey FOREIGN KEY (approver_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_claims guild_storage_claims_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_claims
    ADD CONSTRAINT guild_storage_claims_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_claims guild_storage_claims_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_claims
    ADD CONSTRAINT guild_storage_claims_item_instance_id_fkey FOREIGN KEY (item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: guild_storage_claims guild_storage_claims_requester_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_storage_claims
    ADD CONSTRAINT guild_storage_claims_requester_character_id_fkey FOREIGN KEY (requester_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: guild_war_ratings guild_war_ratings_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_war_ratings
    ADD CONSTRAINT guild_war_ratings_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guild_war_settlements guild_war_settlements_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guild_war_settlements
    ADD CONSTRAINT guild_war_settlements_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: guilds guilds_leader_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_leader_character_id_fkey FOREIGN KEY (leader_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_depositor_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_depositor_account_id_fkey FOREIGN KEY (depositor_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_depositor_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_depositor_character_id_fkey FOREIGN KEY (depositor_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_guild_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_guild_id_fkey FOREIGN KEY (guild_id) REFERENCES public.guilds(guild_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_item_instance_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_item_instance_id_fkey FOREIGN KEY (item_instance_id) REFERENCES public.item_instances(item_instance_id) ON DELETE RESTRICT;


--
-- Name: item_locations item_locations_listing_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.item_locations
    ADD CONSTRAINT item_locations_listing_id_fkey FOREIGN KEY (listing_id) REFERENCES public.auction_listings(listing_id) ON DELETE RESTRICT;


--
-- Name: pvp_match_settlements pvp_match_settlements_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_match_settlements
    ADD CONSTRAINT pvp_match_settlements_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: pvp_ratings pvp_ratings_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_ratings
    ADD CONSTRAINT pvp_ratings_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: pvp_sanctions pvp_sanctions_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.pvp_sanctions
    ADD CONSTRAINT pvp_sanctions_character_id_fkey FOREIGN KEY (character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: reward_claim_contributions reward_claim_contributions_reward_claim_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claim_contributions
    ADD CONSTRAINT reward_claim_contributions_reward_claim_id_fkey FOREIGN KEY (reward_claim_id) REFERENCES public.reward_claims(reward_claim_id) ON DELETE RESTRICT;


--
-- Name: reward_claim_lines reward_claim_lines_reward_claim_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claim_lines
    ADD CONSTRAINT reward_claim_lines_reward_claim_id_fkey FOREIGN KEY (reward_claim_id) REFERENCES public.reward_claims(reward_claim_id) ON DELETE RESTRICT;


--
-- Name: reward_claims reward_claims_owner_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reward_claims
    ADD CONSTRAINT reward_claims_owner_character_id_fkey FOREIGN KEY (owner_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: trade_settlement_records trade_settlement_records_counterpart_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_counterpart_account_id_fkey FOREIGN KEY (counterpart_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: trade_settlement_records trade_settlement_records_counterpart_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_counterpart_character_id_fkey FOREIGN KEY (counterpart_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- Name: trade_settlement_records trade_settlement_records_initiator_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_initiator_account_id_fkey FOREIGN KEY (initiator_account_id) REFERENCES public.accounts(account_id) ON DELETE RESTRICT;


--
-- Name: trade_settlement_records trade_settlement_records_initiator_character_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.trade_settlement_records
    ADD CONSTRAINT trade_settlement_records_initiator_character_id_fkey FOREIGN KEY (initiator_character_id) REFERENCES public.characters(character_id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--


