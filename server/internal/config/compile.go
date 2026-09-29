package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CompileReport is the compiler's machine-readable output (contract_outputs:
// compile diagnostics + compile report).
type CompileReport struct {
	ContentRevision string   `json:"content_revision"`
	Catalogs        int      `json:"catalogs"`
	Rows            int      `json:"pk_rows"`
	Equipment       int      `json:"equipment_items"`
	Recipes         int      `json:"recipes"`
	Portals         int      `json:"portals"`
	SpawnGroups     int      `json:"spawn_groups"`
	Spaces          int      `json:"spaces"`
	Monsters        int      `json:"monsters"`
	Bosses          int      `json:"bosses"`
	SkillGeometries int      `json:"skill_geometries"`
	Secondary       int      `json:"secondary_effects"`
	Diagnostics     []string `json:"diagnostics"`
}

// parseSmallRoster reads the catalog's own fenced SMALL_ROSTER block (the
// block immediately following "`SMALL_ROSTER` is exactly:").
func parseSmallRoster(f *DocFile, ds *Diagnostics) map[string]bool {
	out := map[string]bool{}
	if f == nil {
		return out
	}
	marker := -1
	for i, l := range f.Lines {
		if strings.Contains(l, "SMALL_ROSTER` is exactly") {
			marker = i
			break
		}
	}
	if marker < 0 {
		ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "SMALL_ROSTER declaration block not found")
		return out
	}
	inFence := false
	for i := marker + 1; i < len(f.Lines); i++ {
		t := strings.TrimSpace(f.Lines[i])
		if strings.HasPrefix(t, "```") {
			if inFence {
				break
			}
			inFence = true
			continue
		}
		if inFence && IsID(t) {
			out[t] = true
		}
	}
	if len(out) != 10 {
		ds.Add(f.RelPath, marker+1, 0, CodeValueOutOfBounds,
			fmt.Sprintf("SMALL_ROSTER must declare exactly 10 ids, found %d", len(out)))
	}
	return out
}

// cellOr returns the first non-empty cell among the given column names.
func cellOr(cells map[string]string, names ...string) string {
	for _, n := range names {
		if v := cells[n]; v != "" {
			return v
		}
	}
	return ""
}

// Compile runs the full content compile: load -> PK/dedupe -> expansion ->
// geometry resolution -> reference validation -> canonical hash.
// Aux files (systems specs outside docs/07_content) contribute space rows
// (pvp duel/arena, guild war) that the space index must resolve.
func Compile(repoRoot string) (*CandidateSnapshot, *CompileReport, Diagnostics) {
	in, rep, ds := CompileAll(repoRoot)
	var snap *CandidateSnapshot
	if in != nil {
		snap = in.Snap
	}
	return snap, rep, ds
}

// GateInput bundles the compiled candidate with its parsed source documents
// so the activation gate (IMP-004) can validate table data and prose/fenced
// contract values alike.
type GateInput struct {
	Snap  *CandidateSnapshot
	Files map[string]*DocFile
	Aux   map[string]*DocFile
}

// CompileAll is Compile plus the parsed source documents, for callers
// (activation gates) that must also inspect prose and fenced blocks.
func CompileAll(repoRoot string) (*GateInput, *CompileReport, Diagnostics) {
	var ds Diagnostics
	files := map[string]*DocFile{}
	contentDir := filepath.Join(repoRoot, "docs", "07_content")
	for _, name := range LaunchCatalogFiles {
		if f := LoadDocFile(contentDir, name, &ds); f != nil {
			f.RelPath = "docs/07_content/" + name
			files[name] = f
		}
	}
	aux := map[string]*DocFile{}
	// Aux = normative spec docs the compile itself reads: pvp.md and
	// guild_war.md contribute playable-space rows to the space index; the
	// remaining specs are parsed so callers (activation gate) get one
	// consistent document set.
	for _, rel := range []string{
		"docs/01_gameplay/combat.md",
		"docs/01_gameplay/progression.md",
		"docs/01_gameplay/skills.md",
		"docs/01_gameplay/stats.md",
		"docs/02_world/bosses.md",
		"docs/02_world/dungeons.md",
		"docs/02_world/maps_zones.md",
		"docs/02_world/quests.md",
		"docs/02_world/world_rules.md",
		"docs/03_systems/atlas.md",
		"docs/03_systems/cosmetics.md",
		"docs/03_systems/crafting.md",
		"docs/03_systems/equipment.md",
		"docs/03_systems/formations.md",
		"docs/03_systems/guild_war.md",
		"docs/03_systems/items.md",
		"docs/03_systems/monetization.md",
		"docs/03_systems/pvp.md",
		"docs/03_systems/reward_claims.md",
		"docs/04_architecture/physics_geometry_contract.md",
		"docs/03_systems/seasons.md",
		"docs/03_systems/soul_contracts.md",
		"docs/03_systems/spirit_beasts.md",
		"docs/03_systems/spirit_meridian.md",
		"docs/03_systems/trading_auction.md",
		"docs/06_data/config.md",
	} {
		if f := LoadDocFile(repoRoot, rel, &ds); f != nil {
			aux[rel] = f
		}
	}

	snap := &CandidateSnapshot{
		SchemaVersion:  1,
		Catalogs:       map[string]*CatalogSet{},
		EntityProfiles: map[string]string{},
		IDIndex:        map[string]string{},
	}
	// Collect PK rows + declared ids per catalog.
	for _, name := range LaunchCatalogFiles {
		f := files[name]
		if f == nil {
			continue
		}
		cs := &CatalogSet{File: f.RelPath, PKCol: pkColumn[name]}
		cs.Rows = collectRows(f, pkColumn[name], &ds)
		cs.IDs = collectIDs(f, pkColumn[name], declaresByHeading[name])
		snap.Catalogs[name] = cs
		for id := range cs.IDs {
			snap.IDIndex[id] = f.RelPath
		}
	}
	// Monster/boss row compile (rank/element enums + size profiles + attacks).
	compileMonsters(files["monster_catalog.md"], snap, &ds)
	compileBosses(files["boss_catalog.md"], snap, &ds)
	// Finite expansions.
	mapSpawn := mapEntrySpawns(files["world_route_catalog.md"])
	if eq := files["equipment_catalog.md"]; eq != nil {
		expandEquipment(eq, snap, &ds)
		expandEquipmentCosts(files["crafting_catalog.md"], snap, &ds)
	}
	if wr := files["world_route_catalog.md"]; wr != nil {
		snap.Portals = expandPortals(wr, mapSpawn, &ds)
		for _, p := range snap.Portals {
			snap.IDIndex[p.PortalID] = wr.RelPath
		}
	}
	if ms := files["map_spawn_catalog.md"]; ms != nil {
		snap.Spawns = expandSpawns(ms, &ds)
		for _, g := range snap.Spawns {
			snap.IDIndex[g.GroupID] = ms.RelPath
		}
	}
	// Canonical currency ids (economy spec's fixed three-currency set).
	for _, c := range []string{"currency.common", "currency.bound", "currency.special"} {
		snap.IDIndex[c] = "economy_catalog.md"
	}
	// Space index first — spawn anchors and portal endpoints resolve against it.
	snap.Spaces = buildSpaceIndex(files, aux, &ds)
	for _, s := range snap.Spaces {
		snap.IDIndex[s.SpaceID] = s.SourceFile
	}
	canonicalSpawns := genSpawnAnchors(snap)
	// Endpoint validation: every portal source/destination/entry/return spawn
	// must resolve against the space index and the canonical spawn anchors
	// (world_route's declared reject conditions). Portal-derived values are
	// never self-registered, so authored endpoints still get validated.
	checkPortalRefs(snap, canonicalSpawns, &ds)
	// Space/portal/spawn cross-checks against the resolved index.
	checkSpawnSpaceRefs(files["map_spawn_catalog.md"], snap, &ds)
	// Skill geometry + secondary spatial effects + ADR-0047 checks.
	if sk := files["class_skill_catalog.md"]; sk != nil {
		compileSkills(sk, snap, &ds)
	}
	// Launch-wide count invariants (contract §4 acceptance invariants).
	checkCounts(snap, &ds)
	// Content hash over the 24 canonicalized catalogs (contract §5).
	snap.ContentRevision = ContentRevisionHash(files)
	// Reference validation last: needs the full ID index.
	validateReferences(files, aux, snap.IDIndex, &ds)
	ds.Sort()

	rep := &CompileReport{ContentRevision: snap.ContentRevision}
	rep.Catalogs = len(snap.Catalogs)
	for _, c := range snap.Catalogs {
		rep.Rows += len(c.Rows)
	}
	rep.Equipment = len(snap.Equipment)
	rep.Recipes = len(snap.Recipes)
	rep.Portals = len(snap.Portals)
	rep.SpawnGroups = len(snap.Spawns)
	rep.Spaces = len(snap.Spaces)
	rep.Monsters = len(snap.Monsters)
	rep.Bosses = len(snap.Bosses)
	rep.SkillGeometries = len(snap.Skills)
	rep.Secondary = len(snap.Secondary)
	for _, d := range ds {
		rep.Diagnostics = append(rep.Diagnostics, d.String())
	}
	return &GateInput{Snap: snap, Files: files, Aux: aux}, rep, ds
}

var elements = map[string]bool{"KIM": true, "MOC": true, "THUY": true, "HOA": true, "THO": true, "NONE": true}

// compileMonsters resolves rank/level/element/movement/combat + the ADR-0046
// size profile for every roster row; validates the 58+6 launch invariant with
// the authored 46 NORMAL / 12 ELITE split.
func compileMonsters(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	if f == nil {
		return
	}
	smallRoster := parseSmallRoster(f, ds)
	launch, season, launchNormal, launchElite := 0, 0, 0, 0
	for _, t := range f.TablesWithHeader("monster_id") {
		isSeason := strings.Contains(strings.ToLower(t.Section), "season")
		for _, r := range t.Rows {
			m := MonsterDef{
				MonsterID:   r.Cells["monster_id"],
				Rank:        r.Cells["rank"],
				Element:     r.Cells["element"],
				Movement:    r.Cells["movement"],
				Combat:      r.Cells["combat"],
				DropTableID: r.Cells["drop_table_id"],
			}
			lv, e := ParseInt(cellOr(r.Cells, "Lv", "level"))
			if e != nil || lv < 1 {
				ds.Add(f.RelPath, r.Line, 3, CodeValueOutOfBounds, "monster level not positive int")
			}
			m.Level = lv
			xp, e := ParseInt(r.Cells["base_exp"])
			if e != nil || xp <= 0 {
				ds.Add(f.RelPath, r.Line, 8, CodeValueOutOfBounds, "base_exp not positive")
			}
			m.BaseEXP = xp
			CheckEnumRequired(ds, f.RelPath, r.Line, 3, "rank", m.Rank, map[string]bool{"NORMAL": true, "ELITE": true})
			CheckEnumRequired(ds, f.RelPath, r.Line, 4, "element", m.Element, elements)
			if m.Movement == "" || m.Combat == "" || m.DropTableID == "" {
				ds.Add(f.RelPath, r.Line, 1, CodeValueOutOfBounds,
					"monster "+m.MonsterID+" missing movement/combat/drop_table_id")
			}
			// Entity Size Resolution (contract §4.4): exactly one profile,
			// derived only from rank + SMALL_ROSTER membership.
			switch {
			case m.Rank == "ELITE":
				m.SizeProfile = "MONSTER_ELITE"
			case smallRoster[m.MonsterID]:
				m.SizeProfile = "MONSTER_SMALL"
			case m.Rank == "NORMAL":
				m.SizeProfile = "MONSTER_MEDIUM"
			}
			if m.SizeProfile == "" {
				ds.Add(f.RelPath, r.Line, 3, CodeValueOutOfBounds,
					"monster "+m.MonsterID+" resolved no size_profile")
			}
			// Stable attack ids derived from the combat profile.
			m.AttackIDs = []string{"attack." + m.MonsterID + ".primary"}
			snap.Monsters = append(snap.Monsters, m)
			snap.EntityProfiles[m.MonsterID] = m.SizeProfile
			snap.IDIndex[m.MonsterID] = f.RelPath
			for _, a := range m.AttackIDs {
				snap.IDIndex[a] = f.RelPath
			}
			if isSeason {
				season++
			} else {
				launch++
				switch m.Rank {
				case "NORMAL":
					launchNormal++
				case "ELITE":
					launchElite++
				}
			}
		}
	}
	if launch != 58 || launchNormal != 46 || launchElite != 12 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("launch monster roster must be 58 (46 NORMAL + 12 ELITE), found %d (%d NORMAL + %d ELITE)",
				launch, launchNormal, launchElite))
	}
	if season != 6 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("season-0 monster roster must be 6, found %d", season))
	}
	// SMALL_ROSTER set-equality: every declared small id must exist in the
	// roster as NORMAL, and no extra NORMAL rows may be small.
	for id := range smallRoster {
		if p, ok := snap.EntityProfiles[id]; !ok || p != "MONSTER_SMALL" {
			ds.Add(f.RelPath, 0, 0, CodeUnresolvedReference, "SMALL_ROSTER id "+id+" not a NORMAL roster member")
		}
	}
}

// compileBosses resolves each boss row: mode/scaling enums + declared
// size_profile + space reference.
func compileBosses(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	if f == nil {
		return
	}
	for _, t := range f.TablesWithHeader("boss_id") {
		for _, r := range t.Rows {
			b := BossDef{
				BossID:      r.Cells["boss_id"],
				Mode:        r.Cells["mode"],
				SizeProfile: r.Cells["size_profile"],
				Element:     r.Cells["element"],
				SpaceID:     r.Cells["space_id"],
				DropTableID: r.Cells["drop_table_id"],
			}
			lv, e := ParseInt(cellOr(r.Cells, "Lv", "level"))
			if e == nil {
				b.Level = lv
			}
			xp, e := ParseInt(r.Cells["base_exp"])
			if e == nil {
				b.BaseEXP = xp
			}
			CheckEnumRequired(ds, f.RelPath, r.Line, 1, "mode", b.Mode,
				map[string]bool{"INSTANCED": true, "PUBLIC": true})
			CheckEnumRequired(ds, f.RelPath, r.Line, 1, "size_profile", b.SizeProfile,
				map[string]bool{"BOSS_LARGE": true, "WORLD_BOSS": true})
			if b.SizeProfile == "" {
				ds.Add(f.RelPath, r.Line, 1, CodeValueOutOfBounds, "boss missing size_profile")
			}
			if b.SpaceID == "" || b.Element == "" || b.DropTableID == "" {
				ds.Add(f.RelPath, r.Line, 1, CodeValueOutOfBounds,
					"boss "+b.BossID+" missing space_id/element/drop_table_id")
			}
			snap.Bosses = append(snap.Bosses, b)
			snap.EntityProfiles[b.BossID] = b.SizeProfile
			snap.IDIndex[b.BossID] = f.RelPath
		}
	}
}

// mapEntrySpawns collects each map's declared entry spawn id.
func mapEntrySpawns(f *DocFile) map[string]string {
	out := map[string]string{}
	if f == nil {
		return out
	}
	for _, t := range f.Tables {
		mi, ei := -1, -1
		for i, h := range t.Header {
			if h == "map_id" {
				mi = i
			}
			if h == "entry_spawn" || h == "entry spawn" {
				ei = i
			}
		}
		if mi < 0 || ei < 0 {
			continue
		}
		for _, r := range t.Rows {
			out[r.Order[mi]] = r.Order[ei]
		}
	}
	return out
}

// expandEquipmentCosts fills material/common recipe fields from the crafting
// catalog's tier mapping table.
func expandEquipmentCosts(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	if f == nil {
		return
	}
	type tierCost struct {
		matID  string
		matQty int64
		common int64
	}
	tiers := map[string]tierCost{}
	for _, t := range f.TablesWithHeader("Tier", "material_id", "material base", "common-currency base") {
		for _, r := range t.Rows {
			mq, e1 := ParseInt(r.Cells["material base"])
			cb, e2 := ParseInt(r.Cells["common-currency base"])
			if e1 != nil || e2 != nil {
				ds.Add(f.RelPath, r.Line, 3, CodeTableSyntaxError, "tier material row not numeric")
				continue
			}
			tiers[r.Cells["Tier"]] = tierCost{r.Cells["material_id"], mq, cb}
		}
	}
	slotW := map[string]int64{}
	for _, t := range f.TablesWithHeader("slot", "weight") {
		for _, r := range t.Rows {
			if w, e := ParseInt(r.Cells["weight"]); e == nil {
				slotW[r.Cells["slot"]] = w
			}
		}
	}
	for i := range snap.Equipment {
		it := &snap.Equipment[i]
		tc, ok := tiers[it.Tier]
		if !ok {
			ds.Add(f.RelPath, 0, 0, CodeUnresolvedReference, "no tier material row for "+it.Tier)
			continue
		}
		w, ok := slotW[it.Slot]
		if !ok {
			ds.Add(f.RelPath, 0, 0, CodeUnresolvedReference, "no slot weight for "+it.Slot)
			continue
		}
		it.MaterialID = tc.matID
		it.MaterialQty = tc.matQty * w
		it.CommonCost = tc.common * w
		for j := range snap.Recipes {
			if snap.Recipes[j].RecipeID == it.RecipeID {
				snap.Recipes[j].Inputs = []string{fmt.Sprintf("%d %s", it.MaterialQty, it.MaterialID)}
				snap.Recipes[j].CommonCost = it.CommonCost
			}
		}
	}
	// Utility recipes: recipe.utility.<charm>.<tier>. Tier->level mapping and
	// cost factors are authored in the catalog prose/fenced Input blocks.
	for _, name := range []string{"bua_may", "bua_giu_bac"} {
		levels := charmTierLevels(f, name, ds)
		matF, costF := charmFactors(f, name, ds)
		for i, lv := range levels {
			tier := fmt.Sprintf("t%d", i+1)
			tc := tiers["T"+strconv.Itoa(i+1)]
			snap.Recipes = append(snap.Recipes, RecipeRow{
				RecipeID:   "recipe.utility." + name + "." + tier,
				OutputID:   "item.consumable." + name + "." + lv,
				Inputs:     []string{fmt.Sprintf("%d %s", tc.matQty*matF, tc.matID)},
				CommonCost: tc.common * costF,
			})
			snap.IDIndex["recipe.utility."+name+"."+tier] = f.RelPath
		}
	}
}

// charmTierLevels parses "tA..tB produces `item.consumable.<name>.<level>`"
// lines into a per-tier level list (6 tiers).
var charmTierRe = regexp.MustCompile("t([0-9]+)(?:\\.\\.t([0-9]+))?`?\\s+produces\\s+`?item\\.consumable\\.(bua_may|bua_giu_bac)\\.([a-z_]+)`?")

func charmTierLevels(f *DocFile, name string, ds *Diagnostics) []string {
	levels := make([]string, 6)
	for _, l := range f.Lines {
		m := charmTierRe.FindStringSubmatch(l)
		if m == nil || m[3] != name {
			continue
		}
		lo, _ := strconv.Atoi(m[1])
		hi := lo
		if m[2] != "" {
			hi, _ = strconv.Atoi(m[2])
		}
		for i := lo; i <= hi && i <= 6; i++ {
			levels[i-1] = m[4]
		}
	}
	for i, v := range levels {
		if v == "" {
			ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError,
				"charm "+name+" tier t"+strconv.Itoa(i+1)+" level mapping not declared")
		}
	}
	return levels
}

// charmFactors parses the fenced Input block under `recipe.utility.<name>`:
// "<N> * mapped regional material" and "tier_common_currency_base * <M>".
var charmMatRe = regexp.MustCompile(`([0-9]+)\s*\*\s*mapped regional material`)
var charmCostRe = regexp.MustCompile(`tier_common_currency_base\s*\*\s*([0-9]+)`)

func charmFactors(f *DocFile, name string, ds *Diagnostics) (mat, cost int64) {
	anchor := -1
	for i, l := range f.Lines {
		if strings.Contains(l, "recipe.utility."+name+".") {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "recipe.utility."+name+" block not found")
		return 0, 0
	}
	// The Input block follows the Recipe ID block; scan the following lines
	// regardless of fence state for the two authored factors.
	for i := anchor + 1; i < len(f.Lines) && i < anchor+20; i++ {
		t := strings.TrimSpace(f.Lines[i])
		if strings.HasPrefix(t, "recipe.") || strings.HasPrefix(t, "# ") {
			break // next recipe/section
		}
		if m := charmMatRe.FindStringSubmatch(t); m != nil {
			mat, _ = strconv.ParseInt(m[1], 10, 64)
		}
		if m := charmCostRe.FindStringSubmatch(t); m != nil {
			cost, _ = strconv.ParseInt(m[1], 10, 64)
		}
	}
	if mat <= 0 || cost <= 0 {
		ds.Add(f.RelPath, anchor+1, 0, CodeTableSyntaxError,
			"recipe.utility."+name+" input factors missing")
	}
	return mat, cost
}

// genSpawnAnchors derives the canonical spawn-anchor namespace from the
// space index (world_route conventions): `spawn.entry.<space tail>` for
// every space. Anchors are registered so the global sweep resolves them;
// portal DestSpawn validates membership in this set, ReturnSpawn is
// exact-matched in checkPortalRefs.
func genSpawnAnchors(snap *CandidateSnapshot) map[string]bool {
	canonical := map[string]bool{}
	regions := map[string]bool{}
	keys := map[string]bool{}
	for _, s := range snap.Spaces {
		parts := strings.Split(s.SpaceID, ".")
		if len(parts) < 2 {
			continue
		}
		entry := "spawn.entry." + strings.Join(parts[1:], ".")
		canonical[entry] = true
		snap.IDIndex[entry] = "world_route_catalog.md"
		if parts[0] == "map" {
			regions[parts[1]] = true
		}
		if parts[0] == "dungeon" || parts[0] == "instance" {
			keys[parts[len(parts)-1]] = true
		}
	}
	// spawn.return.<region>.<key> anchors are the dungeon-entry/exit
	// convention — DestSpawn for dungeon portals lands on them.
	for r := range regions {
		for k := range keys {
			ret := "spawn.return." + r + "." + k
			canonical[ret] = true
			snap.IDIndex[ret] = "world_route_catalog.md"
		}
	}
	return canonical
}

// expectedReturnSpawn is the exact authored convention per portal:
// `spawn.return.<source region>.<destination key>` (world_route §"Dungeon
// completion" — returns land in the source field's region).
func expectedReturnSpawn(p Portal) string {
	src := strings.Split(p.SourceMap, ".")
	dst := strings.Split(p.DestSpace, ".")
	if len(src) < 2 || len(dst) < 2 {
		return ""
	}
	return "spawn.return." + src[1] + "." + dst[len(dst)-1]
}

// checkPortalRefs enforces world_route's declared reject conditions: portal
// source/destination spaces and destination/return spawns must resolve.
func checkPortalRefs(snap *CandidateSnapshot, canonicalSpawns map[string]bool, ds *Diagnostics) {
	const rel = "docs/07_content/world_route_catalog.md"
	spaces := map[string]bool{}
	for _, s := range snap.Spaces {
		spaces[s.SpaceID] = true
	}
	for _, p := range snap.Portals {
		if !spaces[p.SourceMap] {
			ds.Add(rel, 0, 0, CodeUnresolvedReference,
				"portal "+p.PortalID+" source space "+p.SourceMap+" unknown")
		}
		if !spaces[p.DestSpace] {
			ds.Add(rel, 0, 0, CodeUnresolvedReference,
				"portal "+p.PortalID+" destination space "+p.DestSpace+" unknown")
		}
		if p.DestSpawn == "" || !canonicalSpawns[p.DestSpawn] {
			ds.Add(rel, 0, 0, CodeUnresolvedReference,
				"portal "+p.PortalID+" destination entry spawn "+p.DestSpawn+" unknown")
		}
		if p.ReturnSpawn != "" {
			expected := expectedReturnSpawn(p)
			if p.ReturnSpawn != expected {
				ds.Add(rel, 0, 0, CodeUnresolvedReference,
					"portal "+p.PortalID+" return spawn "+p.ReturnSpawn+
						" != canonical "+expected)
			}
		}
	}
}

// checkCounts enforces the launch acceptance invariants inside Compile so a
// mutated catalog fails the build, not only the test suite.
func checkCounts(snap *CandidateSnapshot, ds *Diagnostics) {
	const rel = "docs/06_data/content_authoring_contract.md"
	require := func(name string, got, want int) {
		if got != want {
			ds.Add(rel, 0, 0, CodeValueOutOfBounds,
				fmt.Sprintf("%s must be exactly %d, found %d", name, want, got))
		}
	}
	require("equipment items", len(snap.Equipment), 168)
	require("portals", len(snap.Portals), 52)
	require("spawn groups", len(snap.Spawns), 60)
	require("spaces", len(snap.Spaces), 33)
	require("monsters", len(snap.Monsters), 64)
	require("bosses", len(snap.Bosses), 8)
	require("skill geometries", len(snap.Skills), 45)
	require("secondary effects", len(snap.Secondary), 10)
	// Per-kind space counts + world-map geometry invariants (world_route:
	// width 2.0..5.0 screens; all 24 span pairs and layout_profiles distinct).
	kinds := map[string]int{}
	spans := map[string]bool{}
	layouts := map[string]bool{}
	for _, s := range snap.Spaces {
		kinds[s.SpaceKind]++
		if s.SpaceKind == "WORLD" {
			if s.WidthScreens < 2.0 || s.WidthScreens > 5.0 {
				ds.Add(s.SourceFile, 0, 0, CodeValueOutOfBounds,
					fmt.Sprintf("world map %s width %.2f outside 2.0..5.0 screens", s.SpaceID, s.WidthScreens))
			}
			pair := fmt.Sprintf("%.2fx%.2f", s.WidthScreens, s.HeightScreens)
			if spans[pair] {
				ds.Add(s.SourceFile, 0, 0, CodeValueOutOfBounds,
					"duplicate world span pair "+pair)
			}
			spans[pair] = true
			if layouts[s.LayoutProfile] {
				ds.Add(s.SourceFile, 0, 0, CodeValueOutOfBounds,
					"duplicate world layout_profile "+s.LayoutProfile)
			}
			layouts[s.LayoutProfile] = true
		}
	}
	for k, want := range map[string]int{"WORLD": 24, "DUNGEON": 5, "FINALE": 1, "PVP": 2, "GUILD_WAR": 1} {
		require("spaces kind "+k, kinds[k], want)
	}
}

// checkSpawnSpaceRefs validates maps referenced by spawn groups exist in the
// world map catalog and are FIELD maps, and that each group's respawn /
// max_alive sits inside the authored per-kind bands.
func checkSpawnSpaceRefs(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	rel := "docs/07_content/map_spawn_catalog.md"
	fields := map[string]bool{}
	if c := snap.Catalogs["world_route_catalog.md"]; c != nil {
		for pk, r := range c.Rows {
			if r.Cells["type"] == "FIELD" {
				fields[pk] = true
			}
		}
	}
	bands, alive := parseSpawnRules(f, ds)
	count := map[string]int{}
	for _, g := range snap.Spawns {
		if !fields[g.MapID] {
			ds.Add(rel, 0, 0, CodeUnresolvedReference,
				"spawn group "+g.GroupID+" on non-field map "+g.MapID)
		}
		if g.Kind != "rare_night" {
			count[g.MapID]++
		}
		if g.RespawnSeconds <= 0 || g.MaxAlive <= 0 {
			ds.Add(rel, 0, 0, CodeValueOutOfBounds,
				"spawn group "+g.GroupID+" missing respawn/max_alive")
		}
		if b, ok := bands[g.Kind]; ok && g.RespawnSeconds > 0 {
			if g.RespawnSeconds < b[0] || g.RespawnSeconds > b[1] {
				ds.Add(rel, 0, 0, CodeValueOutOfBounds,
					fmt.Sprintf("spawn group %s respawn %ds outside %s band %d..%ds",
						g.GroupID, g.RespawnSeconds, g.Kind, b[0], b[1]))
			}
		}
		if a, ok := alive[g.Kind]; ok && a > 0 && g.MaxAlive != a {
			ds.Add(rel, 0, 0, CodeValueOutOfBounds,
				fmt.Sprintf("spawn group %s max_alive %d != authored %s value %d",
					g.GroupID, g.MaxAlive, g.Kind, a))
		}
		for _, m := range g.MonsterPool {
			if _, ok := snap.IDIndex[m]; !ok {
				ds.Add(rel, 0, 0, CodeUnresolvedReference,
					"spawn group "+g.GroupID+" pool monster "+m+" unresolved")
			}
		}
	}
	for m, n := range count {
		if n != 3 {
			ds.Add(rel, 0, 0, CodeValueOutOfBounds,
				fmt.Sprintf("field map %s must have exactly 3 spawn groups, has %d", m, n))
		}
	}
}

// parseSpawnRules reads the authored respawn bands (fenced block:
// "NORMAL = 10..14s") and max_alive rules ("Normal groups use
// `max_alive = 20`") from map_spawn_catalog.md.
var respawnBandRe = regexp.MustCompile(`^([A-Z_]+)\s*=\s*([0-9]+)\.\.([0-9]+)s`)
var maxAliveRe = regexp.MustCompile("(?i)(normal|elite|night rare) groups use `?max_alive = ([0-9]+)`?")

func parseSpawnRules(f *DocFile, ds *Diagnostics) (map[string][2]int64, map[string]int64) {
	bands := map[string][2]int64{}
	alive := map[string]int64{}
	if f == nil {
		return bands, alive
	}
	for _, l := range f.Lines {
		t := strings.TrimSpace(l)
		if m := respawnBandRe.FindStringSubmatch(t); m != nil {
			lo, _ := strconv.ParseInt(m[2], 10, 64)
			hi, _ := strconv.ParseInt(m[3], 10, 64)
			bands[strings.ToLower(m[1])] = [2]int64{lo, hi}
		}
		for _, m := range maxAliveRe.FindAllStringSubmatch(t, -1) {
			v, _ := strconv.ParseInt(m[2], 10, 64)
			key := strings.ToLower(m[1])
			if key == "night rare" {
				key = "rare_night"
			}
			alive[key] = v
		}
	}
	// Kind names in the catalog are NORMAL/ELITE/NIGHT_RARE; groups carry
	// normal/elite/rare_night.
	if v, ok := bands["night_rare"]; ok {
		bands["rare_night"] = v
		delete(bands, "night_rare")
	}
	for _, k := range []string{"normal", "elite", "rare_night"} {
		if _, ok := bands[k]; !ok {
			ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "respawn band for "+k+" not declared")
		}
		if _, ok := alive[k]; !ok {
			ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "max_alive rule for "+k+" not declared")
		}
	}
	return bands, alive
}
