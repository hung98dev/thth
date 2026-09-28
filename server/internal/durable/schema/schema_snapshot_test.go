package schema_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"thinhthan/internal/durable/db"
	"thinhthan/internal/durable/schema"
	"thinhthan/internal/testing/pgtest"
)

// col is one catalog column expectation: name, udt (information_schema
// udt_name), NOT NULL, character_maximum_length (0 when unbounded).
type col struct {
	name   string
	udt    string
	nn     bool
	maxlen int64
}

// fk is one foreign-key expectation: column list, referenced table and
// whether the constraint is DEFERRABLE.
type fk struct {
	cols       string
	ref        string
	deferrable bool
}

// wantTable encodes the physical_schema_contract.md expectations for one
// baseline table. checks/indexes map catalog names to the full normalized
// pg_get_constraintdef / pg_indexes.indexdef text.
type wantTable struct {
	cols    []col
	pk      string
	uniques []string
	fks     []fk
	checks  map[string]string
	indexes map[string]string
	noCols  []string
}

// migratedDB applies the baseline on a scratch database and returns a pool.
func migratedDB(t *testing.T, prefix string) *pgxpool.Pool {
	t.Helper()
	base := pgtest.DSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	dsn, drop, err := pgtest.ScratchDSN(ctx, base, prefix)
	if err != nil {
		t.Fatalf("scratch: %v", err)
	}
	t.Cleanup(drop)
	m, err := schema.NewMigrator(dsn, schema.MigrationsDir())
	if err != nil {
		t.Fatalf("migrator: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("migrator close: %v", err)
	}
	pool, err := db.Open(context.Background(), dsn, 4)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

var wsRe = regexp.MustCompile(`\s+`)

func norm(s string) string { return strings.TrimSpace(wsRe.ReplaceAllString(s, " ")) }

func tableColumns(t *testing.T, pool *pgxpool.Pool, table string) map[string]col {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT column_name, udt_name, is_nullable, COALESCE(character_maximum_length, 0)
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		t.Fatalf("columns %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]col{}
	for rows.Next() {
		var c col
		var nn string
		if err := rows.Scan(&c.name, &c.udt, &nn, &c.maxlen); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		c.nn = nn == "NO"
		out[c.name] = c
	}
	return out
}

func tableConstraints(t *testing.T, pool *pgxpool.Pool, table string, contype string) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT conname, pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE connamespace = 'public'::regnamespace
		  AND conrelid = $1::regclass AND contype = $2`, table, contype)
	if err != nil {
		t.Fatalf("constraints %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			t.Fatalf("scan constraint: %v", err)
		}
		out[name] = norm(def)
	}
	return out
}

func tableIndexes(t *testing.T, pool *pgxpool.Pool, table string) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT indexname, indexdef FROM pg_indexes
		WHERE schemaname = 'public' AND tablename = $1`, table)
	if err != nil {
		t.Fatalf("indexes %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			t.Fatalf("scan index: %v", err)
		}
		out[name] = norm(def)
	}
	return out
}

func deferrableFKs(t *testing.T, pool *pgxpool.Pool, table string) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT conname, pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE connamespace = 'public'::regnamespace
		  AND conrelid = $1::regclass AND contype = 'f' AND condeferrable`, table)
	if err != nil {
		t.Fatalf("deferrable fks %s: %v", table, err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			t.Fatalf("scan fk: %v", err)
		}
		out[name] = norm(def)
	}
	return out
}

// TestBaselineApplyDownApply applies 000001, rolls it back fully, re-applies
// it on real PostgreSQL 18.6 (acceptance: apply/rollback/re-apply).
func TestBaselineApplyDownApply(t *testing.T) {
	base := pgtest.DSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	dsn, drop, err := pgtest.ScratchDSN(ctx, base, "baseline_cycle")
	if err != nil {
		t.Fatalf("scratch: %v", err)
	}
	defer drop()

	newMigrator := func() *schema.Migrator {
		m, err := schema.NewMigrator(dsn, schema.MigrationsDir())
		if err != nil {
			t.Fatalf("migrator: %v", err)
		}
		return m
	}
	m := newMigrator()
	if err := m.Up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	if v, dirty, err := m.Version(); err != nil || v != 1 || dirty {
		t.Fatalf("after up: version=%d dirty=%v err=%v, want version=1 clean", v, dirty, err)
	}
	if err := m.Down(); err != nil {
		t.Fatalf("down: %v", err)
	}
	if v, dirty, err := m.Version(); err != nil || v != 0 || dirty {
		t.Fatalf("after down: version=%d dirty=%v err=%v, want version=0 clean", v, dirty, err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	m = newMigrator()
	defer func() { _ = m.Close() }()
	if err := m.Up(); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if v, dirty, err := m.Version(); err != nil || v != 1 || dirty {
		t.Fatalf("after re-apply: version=%d dirty=%v err=%v", v, dirty, err)
	}
}

// TestMigrationsImmutable enforces committed-migration immutability via the
// sha256 manifest: every NNNNNN_*.{up,down}.sql on disk is listed and hashes
// to its committed digest; the manifest names no absent file.
func TestMigrationsImmutable(t *testing.T) {
	dir := schema.MigrationsDir()
	if err := schema.CheckManifest(dir); err != nil {
		t.Fatalf("manifest check: %v", err)
	}
	// Negative: a mutated migration file must fail the check.
	tmp := t.TempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if strings.HasSuffix(e.Name(), ".down.sql") {
			data = append(data, []byte("\n-- tampered\n")...)
		}
		if err := os.WriteFile(filepath.Join(tmp, e.Name()), data, 0o644); err != nil {
			t.Fatalf("write tmp %s: %v", e.Name(), err)
		}
	}
	if err := schema.CheckManifest(tmp); err == nil {
		t.Fatal("manifest check passed on tampered migration files")
	}
}

// TestPerConstraintSnapshot asserts the migrated catalog matches the
// physical_schema_contract.md baseline — one subtest per table, column,
// key, check, foreign key and index.
func TestPerConstraintSnapshot(t *testing.T) {
	pool := migratedDB(t, "snap")

	var wantNames []string
	for name := range catalog {
		wantNames = append(wantNames, name)
	}
	sort.Strings(wantNames)

	var gotNames []string
	rows, err := pool.Query(context.Background(), `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'schema_migrations'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan table: %v", err)
		}
		gotNames = append(gotNames, n)
	}
	rows.Close()
	sort.Strings(gotNames)
	t.Run("tables", func(t *testing.T) {
		if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
			t.Fatalf("table set mismatch\n got: %v\nwant: %v", gotNames, wantNames)
		}
	})

	for _, table := range wantNames {
		want := catalog[table]
		t.Run(table, func(t *testing.T) {
			gotCols := tableColumns(t, pool, table)
			t.Run("columns", func(t *testing.T) {
				if len(gotCols) != len(want.cols) {
					t.Fatalf("column count %d, want %d", len(gotCols), len(want.cols))
				}
				for _, wc := range want.cols {
					gc, ok := gotCols[wc.name]
					if !ok {
						t.Errorf("column %s missing", wc.name)
						continue
					}
					if gc.udt != wc.udt || gc.nn != wc.nn || gc.maxlen != wc.maxlen {
						t.Errorf("column %s = (%s,nn=%v,len=%d), want (%s,nn=%v,len=%d)",
							wc.name, gc.udt, gc.nn, gc.maxlen, wc.udt, wc.nn, wc.maxlen)
					}
				}
			})
			t.Run("primary_key", func(t *testing.T) {
				pks := tableConstraints(t, pool, table, "p")
				if len(pks) != 1 {
					t.Fatalf("pk count %d, want 1", len(pks))
				}
				for _, def := range pks {
					if !strings.HasSuffix(def, "("+want.pk+")") {
						t.Fatalf("pk def %q, want columns (%s)", def, want.pk)
					}
				}
			})
			t.Run("checks", func(t *testing.T) {
				got := tableConstraints(t, pool, table, "c")
				if len(got) != len(want.checks) {
					var extra []string
					for n := range got {
						if _, ok := want.checks[n]; !ok {
							extra = append(extra, n)
						}
					}
					t.Fatalf("check count %d, want %d (extra: %v)", len(got), len(want.checks), extra)
				}
				for name, def := range want.checks {
					gd, ok := got[name]
					if !ok {
						t.Errorf("check %s missing", name)
						continue
					}
					if gd != def {
						t.Errorf("check %s def\n got: %s\nwant: %s", name, gd, def)
					}
				}
			})
			t.Run("uniques", func(t *testing.T) {
				got := map[string]bool{}
				for _, def := range tableConstraints(t, pool, table, "u") {
					got[def] = true
				}
				for _, cols := range want.uniques {
					wantDef := "UNIQUE (" + cols + ")"
					if !got[wantDef] {
						t.Errorf("unique %s missing (have: %v)", wantDef, got)
					}
				}
				if len(got) != len(want.uniques) {
					t.Fatalf("unique count %d, want %d", len(got), len(want.uniques))
				}
			})
			t.Run("foreign_keys", func(t *testing.T) {
				gotList := tableConstraints(t, pool, table, "f")
				got := map[string]bool{}
				for _, def := range gotList {
					got[def] = true
				}
				deferrable := deferrableFKs(t, pool, table)
				for _, w := range want.fks {
					wantPrefix := "FOREIGN KEY (" + w.cols + ") REFERENCES " + w.ref + "("
					matched := false
					for def := range got {
						if strings.HasPrefix(def, wantPrefix) {
							matched = true
						}
					}
					if !matched {
						t.Errorf("fk (%s)->%s missing", w.cols, w.ref)
						continue
					}
					// deferrable flag: the constraint whose def matches must
					// be in/not-in the deferrable set.
					isDef := false
					for _, ddef := range deferrable {
						if strings.HasPrefix(ddef, wantPrefix) {
							isDef = true
						}
					}
					if isDef != w.deferrable {
						t.Errorf("fk (%s)->%s deferrable=%v, want %v", w.cols, w.ref, isDef, w.deferrable)
					}
				}
				if len(gotList) != len(want.fks) {
					t.Fatalf("fk count %d, want %d", len(gotList), len(want.fks))
				}
			})
			t.Run("indexes", func(t *testing.T) {
				got := tableIndexes(t, pool, table)
				// pkey + UNIQUE-constraint backing indexes are asserted via
				// pk/uniques — not part of the declared index catalog. Filter
				// both sides so want.indexes may list them harmlessly.
				ucon := map[string]bool{}
				for name := range tableConstraints(t, pool, table, "u") {
					ucon[name] = true
				}
				for name := range got {
					if strings.HasSuffix(name, "_pkey") || ucon[name] {
						delete(got, name)
					}
				}
				wantIdx := map[string]string{}
				for name, def := range want.indexes {
					if !ucon[name] {
						wantIdx[name] = def
					}
				}
				if len(got) != len(wantIdx) {
					t.Fatalf("index count %d, want %d (got %v)", len(got), len(wantIdx), got)
				}
				for name, def := range wantIdx {
					gd, ok := got[name]
					if !ok {
						t.Errorf("index %s missing", name)
						continue
					}
					if gd != def {
						t.Errorf("index %s def\n got: %s\nwant: %s", name, gd, def)
					}
				}
			})
			for _, nc := range want.noCols {
				t.Run("no_column_"+nc, func(t *testing.T) {
					if _, ok := gotCols[nc]; ok {
						t.Errorf("forbidden column %s present on %s", nc, table)
					}
				})
			}
		})
	}
}

// TestBaselineAdr0065Tables asserts the ADR-0065 inventory: auth tables,
// account_login_history, IAP dedup/cursors, reward-claim tables,
// auction_listings, guild tables, audit_events, operations keyed
// (operation_family, owner_id, operation_id), name_key VARCHAR(256).
func TestBaselineAdr0065Tables(t *testing.T) {
	pool := migratedDB(t, "adr65")
	want := []string{
		// auth / account (ADR-0065)
		"accounts", "account_password_credentials", "account_identities",
		"account_login_history", "auth_session_families",
		"auth_refresh_credentials", "auth_revocations", "operators",
		// IAP dedup/cursors + entitlements
		"account_iap_entitlements", "account_refund_consumed_events",
		"account_cosmetic_entitlements", "account_entitlement_claims",
		"iap_notification_dedup", "iap_provider_cursors",
		// reward claims
		"reward_claims", "reward_claim_lines", "reward_claim_contributions",
		// auction
		"auction_listings", "auction_proceeds", "trade_settlement_records",
		// guild
		"guilds", "guild_memberships", "guild_invites", "guild_applications",
		"guild_progression", "guild_ritual_cycles", "guild_blessing_votes",
		"guild_stone_category_completions", "guild_member_contributions",
		"guild_storage_claims", "guild_storage_audit",
		// audit + operations
		"audit_events", "operations", "chat_messages",
	}
	for _, table := range want {
		t.Run(table, func(t *testing.T) {
			var n int
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*) FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1`, table).Scan(&n); err != nil || n != 1 {
				t.Fatalf("table %s present=%d err=%v", table, n, err)
			}
		})
	}
	t.Run("operations_key", func(t *testing.T) {
		pks := tableConstraints(t, pool, "operations", "p")
		found := false
		for _, def := range pks {
			if def == "PRIMARY KEY (operation_family, owner_id, operation_id)" {
				found = true
			}
		}
		if !found {
			t.Fatalf("operations pk = %v", pks)
		}
	})
	t.Run("name_key_varchar_256", func(t *testing.T) {
		cols := tableColumns(t, pool, "characters")
		c, ok := cols["name_key"]
		if !ok || c.udt != "varchar" || c.maxlen != 256 {
			t.Fatalf("characters.name_key = %+v, want varchar(256)", c)
		}
	})
	t.Run("owner_kind_check", func(t *testing.T) {
		checks := tableConstraints(t, pool, "operations", "c")
		def := checks["operations_owner_kind_check"]
		for _, kind := range []string{"ACCOUNT", "CHARACTER", "GUILD", "WORLD"} {
			if !strings.Contains(def, "'"+kind+"'") {
				t.Errorf("operations owner_kind check missing %s: %s", kind, def)
			}
		}
	})
}

// TestTombstoneAccountSeeded asserts the baseline seeds TOMBSTONE_ACCOUNT_ID
// (00000000-0000-0000-0000-000000000001) as a TOMBSTONE_ERASED account row.
func TestTombstoneAccountSeeded(t *testing.T) {
	pool := migratedDB(t, "tomb")
	var status string
	err := pool.QueryRow(context.Background(),
		`SELECT status FROM accounts WHERE account_id = '00000000-0000-0000-0000-000000000001'`).
		Scan(&status)
	if err != nil {
		t.Fatalf("tombstone row: %v", err)
	}
	if status != "TOMBSTONE_ERASED" {
		t.Fatalf("tombstone status = %q, want TOMBSTONE_ERASED", status)
	}
}

// TestSeasonTrackIndexExcludesTombstone: the season-track partial unique
// index on account_iap_entitlements excludes the tombstone account —
// duplicate PENDING rows for the tombstone under one season must not
// conflict, while duplicates for a real account must.
func TestSeasonTrackIndexExcludesTombstone(t *testing.T) {
	pool := migratedDB(t, "season")
	ctx := context.Background()
	const tomb = "00000000-0000-0000-0000-000000000001"
	var acct string
	if err := pool.QueryRow(ctx, `
		INSERT INTO accounts (account_id, status) VALUES (gen_random_uuid(), 'ACTIVE')
		RETURNING account_id`).Scan(&acct); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	insert := `INSERT INTO account_iap_entitlements
		(entitlement_id, account_id, product_id, entitlement_type, grant_state,
		 platform, platform_receipt, season_number, created_at)
		VALUES (gen_random_uuid(), $1, 'season.pass', 'ACCOUNT_SCOPED_ACCESS', 'PENDING',
		        'STEAM', $2, 1, NOW())`
	for i, receipt := range []string{"t1", "t2"} {
		if _, err := pool.Exec(ctx, insert, tomb, receipt); err != nil {
			t.Fatalf("tombstone season row %d rejected: %v", i, err)
		}
	}
	if _, err := pool.Exec(ctx, insert, acct, "r1"); err != nil {
		t.Fatalf("real account season row 1 rejected: %v", err)
	}
	if _, err := pool.Exec(ctx, insert, acct, "r2"); err == nil {
		t.Fatal("duplicate PENDING season row for real account allowed; index must reject it")
	}
}

// TestEntitlementClaimFksDeferrable: account_entitlement_claims carries
// exactly two DEFERRABLE composite FKs — (account_entitlement_id,
// account_id, entitlement_type) -> account_iap_entitlements and
// (character_id, account_id) -> characters.
func TestEntitlementClaimFksDeferrable(t *testing.T) {
	pool := migratedDB(t, "deferred")
	got := deferrableFKs(t, pool, "account_entitlement_claims")
	if len(got) != 2 {
		t.Fatalf("deferrable fks on account_entitlement_claims = %v, want 2", got)
	}
	var iap, char bool
	for _, def := range got {
		if strings.HasPrefix(def, "FOREIGN KEY (account_entitlement_id, account_id, entitlement_type) REFERENCES account_iap_entitlements(entitlement_id, account_id, entitlement_type)") {
			iap = true
		}
		if strings.HasPrefix(def, "FOREIGN KEY (character_id, account_id) REFERENCES characters(character_id, account_id)") {
			char = true
		}
	}
	if !iap || !char {
		t.Fatalf("deferrable fks = %v", got)
	}
	// The composite UNIQUE constraint on characters(character_id, account_id)
	// that the second FK references must exist.
	uniq := tableConstraints(t, pool, "characters", "u")
	found := false
	for _, def := range uniq {
		if def == "UNIQUE (character_id, account_id)" {
			found = true
		}
	}
	if !found {
		t.Fatalf("characters lacks UNIQUE (character_id, account_id): %v", uniq)
	}
}

// TestBaselineAdr0060Tables: ADR-0060 tables/columns/CHECKs exist —
// character_cosmetic_entitlements, character_cosmetic_equips,
// character_souls, character_beast_food_daily;
// account_iap_entitlements.grant_state incl. REJECTED + reject_reason +
// platform; account_cosmetic_entitlements.first_equipped_at.
func TestBaselineAdr0060Tables(t *testing.T) {
	pool := migratedDB(t, "adr60")
	for _, table := range []string{
		"character_cosmetic_entitlements", "character_cosmetic_equips",
		"character_souls", "character_beast_food_daily",
	} {
		t.Run(table, func(t *testing.T) {
			var n int
			if err := pool.QueryRow(context.Background(), `
				SELECT count(*) FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1`, table).Scan(&n); err != nil || n != 1 {
				t.Fatalf("table %s present=%d err=%v", table, n, err)
			}
		})
	}
	t.Run("grant_state_rejected", func(t *testing.T) {
		checks := tableConstraints(t, pool, "account_iap_entitlements", "c")
		def := checks["account_iap_entitlements_grant_state_check"]
		if !strings.Contains(def, "'REJECTED'") {
			t.Fatalf("grant_state check lacks REJECTED: %s", def)
		}
	})
	for _, c := range []string{"reject_reason", "platform"} {
		t.Run("iap_column_"+c, func(t *testing.T) {
			if _, ok := tableColumns(t, pool, "account_iap_entitlements")[c]; !ok {
				t.Fatalf("account_iap_entitlements.%s missing", c)
			}
		})
	}
	t.Run("first_equipped_at", func(t *testing.T) {
		c, ok := tableColumns(t, pool, "account_cosmetic_entitlements")["first_equipped_at"]
		if !ok || c.udt != "timestamptz" || c.nn {
			t.Fatalf("first_equipped_at = %+v, want nullable timestamptz", c)
		}
	})
	t.Run("no_trade_escrow_location", func(t *testing.T) {
		checks := tableConstraints(t, pool, "item_locations", "c")
		for name, def := range checks {
			if strings.Contains(def, "TRADE_ESCROW") {
				t.Errorf("item_locations check %s mentions forbidden TRADE_ESCROW", name)
			}
		}
	})
}

// TestNoStoredRefundScore: accounts must not carry
// iap_refund_consumed_score — the consumed score is derived from
// account_refund_consumed_events (ADR-0060, data_model.md).
func TestNoStoredRefundScore(t *testing.T) {
	pool := migratedDB(t, "norefund")
	for table, col := range map[string]string{
		"accounts": "iap_refund_consumed_score",
	} {
		if _, ok := tableColumns(t, pool, table)[col]; ok {
			t.Fatalf("forbidden column %s.%s present", table, col)
		}
	}
	var n int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'account_refund_consumed_events'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("account_refund_consumed_events present=%d err=%v", n, err)
	}
}

// TestBaselinePublicBossSchedules (ADR-0061): public_boss_schedules exists
// with the data_model.md CHECKs — state SCHEDULED/OPEN, OPEN requires
// opened_at + spawn generation + next_spawn_at, and revision >= 0.
func TestBaselinePublicBossSchedules(t *testing.T) {
	pool := migratedDB(t, "boss")
	ctx := context.Background()
	var n int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'public_boss_schedules'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("public_boss_schedules present=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO public_boss_schedules (boss_id, state, revision)
		VALUES ('b1', 'OPEN', 0)`); err == nil {
		t.Fatal("OPEN without opened_at/generation/next_spawn_at accepted")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO public_boss_schedules (boss_id, state, revision)
		VALUES ('b2', 'RUNNING', 0)`); err == nil {
		t.Fatal("invalid state accepted")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO public_boss_schedules (boss_id, state, opened_at, next_spawn_at, public_boss_spawn_generation_id, revision)
		VALUES ('b3', 'OPEN', NOW(), NULL, gen_random_uuid(), 0)`); err != nil {
		t.Fatalf("valid OPEN row rejected: %v", err)
	}
}
