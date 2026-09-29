// Package compiler_test holds the declared IMP-003 acceptance tests (packet
// ## Tests). They compile the real docs/07_content launch catalogs and assert
// the exact expansion counts, resolution rules, and ADR-0047 invariants.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"thinhthan/internal/config"
)

// repoRoot locates the repository root from server/cmd/compiler.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("repo root %q missing AGENTS.md: %v", root, err)
	}
	return root
}

// compileOnce compiles the real tree once for the whole test binary.
func compileReal(t *testing.T) (*config.CandidateSnapshot, *config.CompileReport) {
	t.Helper()
	snap, rep, _ := config.Compile(repoRoot(t))
	return snap, rep
}

func TestCompileAllCatalogs(t *testing.T) {
	snap, rep, ds := config.Compile(repoRoot(t))
	if ds.HasErrors() {
		for _, d := range ds {
			t.Errorf("[%s:%d:%d] [%s] %s", d.File, d.Line, d.Col, d.Code, d.Detail)
		}
		t.Fatalf("compile produced %d diagnostics", len(ds))
	}
	if rep.Catalogs != 24 {
		t.Fatalf("catalogs = %d, want 24", rep.Catalogs)
	}
	if len(rep.ContentRevision) != 64 {
		t.Fatalf("content_revision %q is not a SHA-256 hex", rep.ContentRevision)
	}
	if snap.ContentRevision != rep.ContentRevision {
		t.Fatalf("snapshot revision %q != report revision %q",
			snap.ContentRevision, rep.ContentRevision)
	}
	// Every one of the 24 files contributed a catalog entry.
	if len(snap.Catalogs) != 24 {
		t.Fatalf("snapshot catalogs = %d, want 24", len(snap.Catalogs))
	}
}

func TestEquipmentExpansion168(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Equipment) != 168 {
		t.Fatalf("equipment rows = %d, want 168 (12 sets x 14 slots)", len(snap.Equipment))
	}
	seen := map[string]bool{}
	sets := map[string]map[string]bool{}
	for _, it := range snap.Equipment {
		if seen[it.ItemID] {
			t.Fatalf("duplicate expanded item id %s", it.ItemID)
		}
		seen[it.ItemID] = true
		if !strings.HasPrefix(it.ItemID, "item.eq.") {
			t.Fatalf("item id %s lacks item.eq. namespace", it.ItemID)
		}
		if sets[it.SetKey] == nil {
			sets[it.SetKey] = map[string]bool{}
		}
		sets[it.SetKey][it.Slot] = true
		if len(it.Stats) == 0 {
			t.Fatalf("item %s has no resolved stats", it.ItemID)
		}
		if it.MaterialID == "" || it.MaterialQty <= 0 {
			t.Fatalf("item %s missing recipe material inputs", it.ItemID)
		}
	}
	if len(sets) != 12 {
		t.Fatalf("set keys = %d, want 12", len(sets))
	}
	for k, slots := range sets {
		if len(slots) != 14 {
			t.Fatalf("set %s covers %d slots, want 14", k, len(slots))
		}
	}
	if len(snap.Recipes) < 168 {
		t.Fatalf("recipes = %d, want >= 168 equipment recipes", len(snap.Recipes))
	}
}

func TestPortalExpansion52(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Portals) != 52 {
		t.Fatalf("portals = %d, want 52 directional edges", len(snap.Portals))
	}
	ids := map[string]bool{}
	for _, p := range snap.Portals {
		if ids[p.PortalID] {
			t.Fatalf("duplicate portal id %s", p.PortalID)
		}
		ids[p.PortalID] = true
		if !strings.HasPrefix(p.PortalID, "portal.") {
			t.Fatalf("portal id %s lacks portal. namespace", p.PortalID)
		}
		if p.SourceMap == "" || p.DestSpace == "" || p.DestSpawn == "" {
			t.Fatalf("portal %s missing endpoints: %+v", p.PortalID, p)
		}
		if _, ok := snap.IDIndex[p.DestSpace]; !ok {
			t.Fatalf("portal %s destination space %s not in id index", p.PortalID, p.DestSpace)
		}
	}
	// Undirected world edges expanded to both directions: at least one pair
	// must have both directions present.
	sawBidi := false
	for id := range ids {
		parts := strings.Split(strings.TrimPrefix(id, "portal."), ".to.")
		if len(parts) != 2 {
			continue
		}
		if ids["portal."+parts[1]+".to."+parts[0]] {
			sawBidi = true
			break
		}
	}
	if !sawBidi {
		t.Fatal("no bidirectional portal pair found in expansion")
	}
}

func TestEntitySizeProfileResolution(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Monsters) != 64 {
		t.Fatalf("monsters = %d, want 64 (58 launch + 6 season-0)", len(snap.Monsters))
	}
	if len(snap.Bosses) != 8 {
		t.Fatalf("bosses = %d, want 8", len(snap.Bosses))
	}
	for _, m := range snap.Monsters {
		p := snap.EntityProfiles[m.MonsterID]
		if p == "" {
			t.Fatalf("monster %s did not resolve a size_profile", m.MonsterID)
		}
		switch m.Rank {
		case "ELITE":
			if p != "MONSTER_ELITE" {
				t.Fatalf("elite monster %s resolved %s, want MONSTER_ELITE", m.MonsterID, p)
			}
		default:
			if p != "MONSTER_SMALL" && p != "MONSTER_MEDIUM" {
				t.Fatalf("monster %s resolved %s, want MONSTER_SMALL or MONSTER_MEDIUM", m.MonsterID, p)
			}
		}
	}
	for _, b := range snap.Bosses {
		p := snap.EntityProfiles[b.BossID]
		if p != "BOSS_LARGE" && p != "WORLD_BOSS" {
			t.Fatalf("boss %s resolved %s, want BOSS_LARGE or WORLD_BOSS", b.BossID, p)
		}
	}
}

func TestPlayableSpaceGeometryIndex(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Spaces) != 33 {
		t.Fatalf("spaces = %d, want 33 (24 world + 5 dungeon + finale + 2 pvp + guild_war)", len(snap.Spaces))
	}
	kinds := map[string]int{}
	for _, s := range snap.Spaces {
		kinds[s.SpaceKind]++
		if s.BoundsMaxX <= 0 || s.BoundsMaxY <= 0 {
			t.Fatalf("space %s has non-positive bounds %+v", s.SpaceID, s)
		}
		if s.LayoutProfile == "" {
			t.Fatalf("space %s missing layout_profile", s.SpaceID)
		}
		// bounds = span screens x tile-extent contract (25.6 x 14.4 m/screen).
		const eps = 0.001
		if got := s.WidthScreens * 25.6; got-s.BoundsMaxX > eps || s.BoundsMaxX-got > eps {
			t.Fatalf("space %s max_x %.2f != width %.2f x 25.6", s.SpaceID, s.BoundsMaxX, s.WidthScreens)
		}
		if got := s.HeightScreens * 14.4; got-s.BoundsMaxY > eps || s.BoundsMaxY-got > eps {
			t.Fatalf("space %s max_y %.2f != height %.2f x 14.4", s.SpaceID, s.BoundsMaxY, s.HeightScreens)
		}
	}
	for k, n := range map[string]int{"WORLD": 24, "DUNGEON": 5, "FINALE": 1, "PVP": 2, "GUILD_WAR": 1} {
		if kinds[k] != n {
			t.Fatalf("space kind %s = %d, want %d", k, kinds[k], n)
		}
	}
}

func TestSkillGeometryRows45(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Skills) != 45 {
		t.Fatalf("skill geometry rows = %d, want 45 (20 basic + 25 active)", len(snap.Skills))
	}
	basic, active := 0, 0
	for _, s := range snap.Skills {
		if strings.Contains(s.SkillID, ".basic.") {
			basic++
		} else if strings.Contains(s.SkillID, ".active.") {
			active++
		} else {
			t.Fatalf("geometry row %s is neither basic nor active", s.SkillID)
		}
		if s.Geometry.Kind == "" {
			t.Fatalf("skill %s has no compiled geometry kind", s.SkillID)
		}
		for p, v := range s.Geometry.Params {
			if v <= 0 {
				t.Fatalf("skill %s param %s=%v not > 0", s.SkillID, p, v)
			}
		}
	}
	if basic != 20 || active != 25 {
		t.Fatalf("basic=%d active=%d, want 20/25", basic, active)
	}
}

func TestSkillSecondaryGeometryCompile(t *testing.T) {
	snap, _ := compileReal(t)
	if len(snap.Secondary) != 10 {
		t.Fatalf("secondary spatial effects = %d, want 10", len(snap.Secondary))
	}
	for _, se := range snap.Secondary {
		if se.EffectID == "" || !strings.HasPrefix(se.EffectID, "spatial.") {
			t.Fatalf("secondary effect id %q missing spatial. namespace", se.EffectID)
		}
		if se.Resolution == "" {
			t.Fatalf("secondary effect %s missing exact resolution", se.EffectID)
		}
		if se.RadiusM > 0 && (se.RadiusM < 1.2 || se.RadiusM > 4.0) {
			t.Fatalf("secondary effect %s radius %.2f outside 1.2..4.0 band", se.EffectID, se.RadiusM)
		}
	}
}

func TestSkillDisplacementTagConsistency(t *testing.T) {
	snap, _ := compileReal(t)
	displaces := map[string]bool{}
	for _, se := range snap.Secondary {
		if !se.Displaces {
			continue
		}
		for _, src := range se.Sources {
			displaces[src] = true
		}
	}
	for _, s := range snap.Skills {
		tagged := false
		for _, tag := range s.Tags {
			if tag == "DISPLACEMENT" {
				tagged = true
			}
		}
		if tagged && !displaces[s.SkillID] {
			t.Fatalf("%s tagged DISPLACEMENT but has no push/pull/airborne secondary", s.SkillID)
		}
		if !tagged && displaces[s.SkillID] {
			t.Fatalf("%s produces displacement but lacks the DISPLACEMENT tag", s.SkillID)
		}
	}
}

func TestInvalidReferenceRejection(t *testing.T) {
	// Copy docs/ into a temp root and corrupt one reference; the compiler must
	// reject the candidate with UNRESOLVED_REFERENCE.
	src := filepath.Join(repoRoot(t), "docs")
	dst := filepath.Join(t.TempDir(), "docs")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copy docs: %v", err)
	}
	target := filepath.Join(dst, "07_content", "map_spawn_catalog.md")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read spawn catalog: %v", err)
	}
	// Inject an unresolvable monster id into the pool of the first spawn block.
	bad := strings.Replace(string(b), "pool = ", "pool = monster.nosuch.bogus, ", 1)
	if bad == string(b) {
		t.Fatal("spawn catalog contains no 'pool = ' line to corrupt")
	}
	if err := os.WriteFile(target, []byte(bad), 0o644); err != nil {
		t.Fatalf("write corrupted catalog: %v", err)
	}
	root := filepath.Dir(dst)
	_, _, ds := config.Compile(root)
	if !ds.HasErrors() {
		t.Fatal("compile of corrupted tree produced no diagnostics")
	}
	found := false
	for _, d := range ds {
		if d.Code == config.CodeUnresolvedReference || strings.Contains(d.Detail, "monster.nosuch.bogus") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no diagnostic mentions the injected bad ref; got %v", ds)
	}
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, b, 0o644)
	})
}
