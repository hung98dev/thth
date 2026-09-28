package schema_test

import (
	"context"
	"strings"
	"testing"
)

// ADR-0070: durable restart relic expiry + erasure ledger + entity budgets.

// TestPendingErasureLedgerTable: pending_erasure_ledger exists with the
// data_model.md columns — operation_id PK, account_id_hash BYTEA,
// executed_at, attempts DEFAULT 0, last_error VARCHAR(256).
func TestPendingErasureLedgerTable(t *testing.T) {
	pool := migratedDB(t, "adr70_erasure")
	cols := tableColumns(t, pool, "pending_erasure_ledger")
	want := []col{
		{"operation_id", "uuid", true, 0},
		{"account_id_hash", "bytea", true, 0},
		{"executed_at", "timestamptz", true, 0},
		{"attempts", "int4", true, 0},
		{"last_error", "varchar", false, 256},
	}
	for _, w := range want {
		g, ok := cols[w.name]
		if !ok {
			t.Errorf("column %s missing", w.name)
			continue
		}
		if g.udt != w.udt || g.nn != w.nn || g.maxlen != w.maxlen {
			t.Errorf("column %s = (%s,nn=%v,len=%d), want (%s,nn=%v,len=%d)",
				w.name, g.udt, g.nn, g.maxlen, w.udt, w.nn, w.maxlen)
		}
	}
	pks := tableConstraints(t, pool, "pending_erasure_ledger", "p")
	if len(pks) != 1 {
		t.Fatalf("pk count %d, want 1", len(pks))
	}
	for _, def := range pks {
		if def != "PRIMARY KEY (operation_id)" {
			t.Fatalf("pk def = %q", def)
		}
	}
}

// TestRelicTypedColumnsAndIndexes: world_consequence_relics typed columns
// (map_id/channel_id/relic_id PK, spawned_at, expires_at = spawned_at + 60
// min CHECK) and the two partial indexes (relic_id, expires_at) and
// (expires_at) WHERE relic_active; region_di_tich_markers typed columns.
func TestRelicTypedColumnsAndIndexes(t *testing.T) {
	pool := migratedDB(t, "adr70_relic")
	cols := tableColumns(t, pool, "world_consequence_relics")
	want := []col{
		{"map_id", "varchar", true, 64},
		{"channel_id", "int2", true, 0},
		{"relic_id", "varchar", true, 64},
		{"source_id", "varchar", true, 64},
		{"relic_active", "bool", true, 0},
		{"buff_effect_id", "varchar", true, 64},
		{"spawned_at", "timestamptz", true, 0},
		{"expires_at", "timestamptz", true, 0},
	}
	for _, w := range want {
		g, ok := cols[w.name]
		if !ok {
			t.Errorf("column %s missing", w.name)
			continue
		}
		if g.udt != w.udt || g.nn != w.nn || g.maxlen != w.maxlen {
			t.Errorf("column %s = (%s,nn=%v,len=%d), want (%s,nn=%v,len=%d)",
				w.name, g.udt, g.nn, g.maxlen, w.udt, w.nn, w.maxlen)
		}
	}
	checks := tableConstraints(t, pool, "world_consequence_relics", "c")
	var expiry, channel bool
	for _, def := range checks {
		if strings.Contains(def, "expires_at = (spawned_at + '01:00:00'::interval)") ||
			strings.Contains(def, "expires_at = (spawned_at + '60 minutes'::interval)") ||
			strings.Contains(def, "expires_at = (spawned_at + '3600 seconds'::interval)") {
			expiry = true
		}
		if strings.Contains(def, "channel_id >= 1") && strings.Contains(def, "channel_id <= 30") {
			channel = true
		}
	}
	if !expiry {
		t.Fatalf("expires_at=spawned_at+60min check missing: %v", checks)
	}
	if !channel {
		t.Fatalf("channel_id 1..30 check missing: %v", checks)
	}
	pks := tableConstraints(t, pool, "world_consequence_relics", "p")
	for _, def := range pks {
		if def != "PRIMARY KEY (map_id, channel_id, relic_id)" {
			t.Fatalf("relic pk = %q", def)
		}
	}
	idx := tableIndexes(t, pool, "world_consequence_relics")
	var activeIdx, sweepIdx bool
	for name, def := range idx {
		if strings.HasPrefix(def, "CREATE INDEX") &&
			strings.Contains(def, "(relic_id, expires_at)") &&
			strings.Contains(def, "WHERE relic_active") {
			activeIdx = true
			_ = name
		}
		if strings.HasPrefix(def, "CREATE INDEX") &&
			strings.Contains(def, "(expires_at)") &&
			strings.Contains(def, "WHERE relic_active") {
			sweepIdx = true
		}
	}
	if !activeIdx || !sweepIdx {
		t.Fatalf("partial indexes missing (active=%v sweep=%v): %v", activeIdx, sweepIdx, idx)
	}

	// region_di_tich_markers typed columns
	mcols := tableColumns(t, pool, "region_di_tich_markers")
	for _, w := range []col{
		{"region_id", "varchar", true, 64},
		{"boss_id", "varchar", true, 64},
		{"last_defeated_utc", "timestamptz", true, 0},
		{"last_defeated_participant_count", "int4", true, 0},
		{"relic_active_in_region", "bool", true, 0},
	} {
		g, ok := mcols[w.name]
		if !ok {
			t.Errorf("region_di_tich_markers.%s missing", w.name)
			continue
		}
		if g.udt != w.udt || g.nn != w.nn || g.maxlen != w.maxlen {
			t.Errorf("region_di_tich_markers.%s = (%s,nn=%v,len=%d)", w.name, g.udt, g.nn, g.maxlen)
		}
	}
	mpk := tableConstraints(t, pool, "region_di_tich_markers", "p")
	for _, def := range mpk {
		if def != "PRIMARY KEY (region_id, boss_id)" {
			t.Fatalf("marker pk = %q", def)
		}
	}
	mchecks := tableConstraints(t, pool, "region_di_tich_markers", "c")
	var part bool
	for _, def := range mchecks {
		if strings.Contains(def, "last_defeated_participant_count >= 1") {
			part = true
		}
	}
	if !part {
		t.Fatalf("participant_count >= 1 check missing: %v", mchecks)
	}
}

// TestRewardClaimLineKindCheck: reward_claim_lines enforces the ADR-0070
// line-kind XOR — ITEM lines carry item fields and no currency fields;
// CURRENCY lines the reverse.
func TestRewardClaimLineKindCheck(t *testing.T) {
	pool := migratedDB(t, "adr70_lines")
	ctx := context.Background()
	// fixture: account -> character -> reward_claim
	var charID string
	if err := pool.QueryRow(ctx, `
		WITH a AS (
			INSERT INTO accounts (account_id, status) VALUES (gen_random_uuid(), 'ACTIVE')
			RETURNING account_id)
		INSERT INTO characters (character_id, account_id, name, name_key, class_id, lifecycle, level, current_exp)
		SELECT gen_random_uuid(), account_id, 'c', 'c', 'class.blade', 'ACTIVE', 1, 0 FROM a
		RETURNING character_id`).Scan(&charID); err != nil {
		t.Fatalf("character fixture: %v", err)
	}
	var claimID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO reward_claims
			(reward_claim_id, owner_character_id, source_type, source_reference,
			 reward_slot, claim_kind, state, created_at, updated_at, revision)
		VALUES (gen_random_uuid(), $1, 'MONSTER', 'src', 'reward.slot.test',
		        'SINGLE', 'PENDING', NOW(), NOW(), 0)
		RETURNING reward_claim_id`, charID).Scan(&claimID); err != nil {
		t.Fatalf("claim fixture: %v", err)
	}
	line := `INSERT INTO reward_claim_lines
		(reward_claim_id, line_no, line_kind, item_id, quantity, effective_binding,
		 item_state, content_revision, currency_id, amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)`

	// ITEM line without item fields must be rejected.
	if _, err := pool.Exec(ctx, line, claimID, 1, "ITEM", nil, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("ITEM line without item fields accepted")
	}
	// CURRENCY line with item fields must be rejected.
	if _, err := pool.Exec(ctx, line, claimID, 2, "CURRENCY", "item.x", 1, "UNBOUND", "{}", 7, nil, nil); err == nil {
		t.Fatal("CURRENCY line carrying item fields accepted")
	}
	// Valid ITEM and CURRENCY lines accepted.
	if _, err := pool.Exec(ctx, line, claimID, 3, "ITEM", "item.sword", 1, "CHARACTER_BOUND", "{}", 12, nil, nil); err != nil {
		t.Fatalf("valid ITEM line rejected: %v", err)
	}
	if _, err := pool.Exec(ctx, line, claimID, 4, "CURRENCY", nil, nil, nil, "{}", nil, "currency.common", 500); err != nil {
		t.Fatalf("valid CURRENCY line rejected: %v", err)
	}
}

// TestAuctionEndedAtCheck: auction_listings enforces
// (state = 'ACTIVE') = (ended_at IS NULL) (ADR-0070).
func TestAuctionEndedAtCheck(t *testing.T) {
	pool := migratedDB(t, "adr70_auction")
	ctx := context.Background()
	var charID, acctID, itemID string
	if err := pool.QueryRow(ctx, `
		WITH a AS (
			INSERT INTO accounts (account_id, status) VALUES (gen_random_uuid(), 'ACTIVE')
			RETURNING account_id)
		INSERT INTO characters (character_id, account_id, name, name_key, class_id, lifecycle, level, current_exp)
		SELECT gen_random_uuid(), account_id, 'c', 'c', 'class.blade', 'ACTIVE', 1, 0 FROM a
		RETURNING character_id, (SELECT account_id FROM a)`).Scan(&charID, &acctID); err != nil {
		t.Fatalf("char fixture: %v", err)
	}
	mkItem := func() string {
		if err := pool.QueryRow(ctx, `
			INSERT INTO item_instances (item_instance_id, item_id, quantity, effective_binding, enhancement_level, roll_state, content_revision)
			VALUES (gen_random_uuid(), 'item.x', 1, 'UNBOUND', 0, '{}'::jsonb, 1)
			RETURNING item_instance_id`).Scan(&itemID); err != nil {
			t.Fatalf("item fixture: %v", err)
		}
		return itemID
	}
	list := `INSERT INTO auction_listings
		(listing_id, seller_character_id, seller_account_id, item_instance_id, item_id,
		 quantity, price_common, listing_fee_common, state, listed_at, expires_at, ended_at, revision)
		VALUES (gen_random_uuid(), $1, $2, $3, 'item.x', 1, 1000, 10, $4, NOW(), NOW() + interval '24 hours', $5, 0)`

	// ACTIVE with ended_at set must be rejected.
	if _, err := pool.Exec(ctx, list, charID, acctID, mkItem(), "ACTIVE", "2020-01-01"); err == nil {
		t.Fatal("ACTIVE listing with ended_at accepted")
	}
	// SOLD without ended_at must be rejected.
	if _, err := pool.Exec(ctx, list, charID, acctID, mkItem(), "SOLD", nil); err == nil {
		t.Fatal("SOLD listing without ended_at accepted")
	}
	// ACTIVE + NULL ended_at is valid.
	if _, err := pool.Exec(ctx, list, charID, acctID, mkItem(), "ACTIVE", nil); err != nil {
		t.Fatalf("valid ACTIVE listing rejected: %v", err)
	}
	// SOLD + ended_at is valid.
	if _, err := pool.Exec(ctx, list, charID, acctID, mkItem(), "SOLD", "2020-01-01"); err != nil {
		t.Fatalf("valid SOLD listing rejected: %v", err)
	}
}

// TestGuildStorageAuditChecks: guild_storage_audit enforces the action and
// section CHECKs (ADR-0070).
func TestGuildStorageAuditChecks(t *testing.T) {
	pool := migratedDB(t, "adr70_audit")
	ctx := context.Background()
	var guildID, charID string
	if err := pool.QueryRow(ctx, `
		WITH a AS (
			INSERT INTO accounts (account_id, status) VALUES (gen_random_uuid(), 'ACTIVE')
			RETURNING account_id),
		c AS (
			INSERT INTO characters (character_id, account_id, name, name_key, class_id, lifecycle, level, current_exp)
			SELECT gen_random_uuid(), account_id, 'c', 'c', 'class.blade', 'ACTIVE', 1, 0 FROM a
			RETURNING character_id)
		INSERT INTO guilds (guild_id, name, name_key, state, recruitment_mode, leader_character_id, motd, guild_revision, guild_storage_revision, created_at)
		SELECT gen_random_uuid(), 'g', 'g', 'ACTIVE', 'APPLICATIONS', character_id, '', 0, 0, NOW() FROM c
		RETURNING guild_id, (SELECT character_id FROM c)`).Scan(&guildID, &charID); err != nil {
		t.Fatalf("guild fixture: %v", err)
	}
	audit := `INSERT INTO guild_storage_audit
		(audit_id, guild_id, operation_id, actor_character_id, action, section,
		 item_id, quantity, receiver_character_id, before_quantity, after_quantity, occurred_at)
		VALUES (gen_random_uuid(), $1, gen_random_uuid(), $2, $3, $4, 'item.x', 1, NULL, 0, 1, NOW())`

	if _, err := pool.Exec(ctx, audit, guildID, charID, "STEAL", "COMMON"); err == nil {
		t.Fatal("invalid action accepted")
	}
	if _, err := pool.Exec(ctx, audit, guildID, charID, "DEPOSIT", "VAULT"); err == nil {
		t.Fatal("invalid section accepted")
	}
	if _, err := pool.Exec(ctx, audit, guildID, charID, "DEPOSIT", "COMMON"); err != nil {
		t.Fatalf("valid audit row rejected: %v", err)
	}
}
