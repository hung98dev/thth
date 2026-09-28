package config

import (
	"strings"
)

// LaunchCatalogFiles is the canonical 24-catalog list from
// content_authoring_contract.md §3, in declaration order.
var LaunchCatalogFiles = []string{
	"monster_catalog.md",
	"boss_catalog.md",
	"class_skill_catalog.md",
	"equipment_catalog.md",
	"item_catalog.md",
	"drop_tables.md",
	"crafting_catalog.md",
	"npc_shop_catalog.md",
	"quest_catalog.md",
	"dungeon_catalog.md",
	"world_route_catalog.md",
	"map_spawn_catalog.md",
	"atlas_catalog.md",
	"cosmetic_catalog.md",
	"soul_catalog.md",
	"build_catalog.md",
	"spirit_beast_catalog.md",
	"economy_catalog.md",
	"world_event_catalog.md",
	"encounter_catalog.md",
	"progression_route.md",
	"balance_validation.md",
	"integration_validation.md",
	"README.md",
}

// pkColumn maps a catalog file to the primary-key column declared in
// content_authoring_contract.md §3. Multiple PK-bearing first columns may
// exist per file; the declared PK is used for catalog row identity.
var pkColumn = map[string]string{
	"monster_catalog.md":        "monster_id",
	"boss_catalog.md":           "boss_id",
	"class_skill_catalog.md":    "skill_id",
	"equipment_catalog.md":      "equipment_id",
	"item_catalog.md":           "item_id",
	"drop_tables.md":            "drop_table_id",
	"crafting_catalog.md":       "recipe_id",
	"npc_shop_catalog.md":       "shop_id",
	"quest_catalog.md":          "quest_id",
	"dungeon_catalog.md":        "dungeon_id",
	"world_route_catalog.md":    "map_id",
	"map_spawn_catalog.md":      "spawn_group_id",
	"atlas_catalog.md":          "atlas_page_id",
	"cosmetic_catalog.md":       "cosmetic_id",
	"soul_catalog.md":           "soul_id",
	"build_catalog.md":          "entry_id",
	"spirit_beast_catalog.md":   "beast_id",
	"economy_catalog.md":        "table_id",
	"world_event_catalog.md":    "event_id",
	"encounter_catalog.md":      "encounter_id",
	"progression_route.md":      "level",
	"balance_validation.md":     "gate_id",
	"integration_validation.md": "rule_id",
}

// declaresByHeading maps catalogs whose concrete rows live in `## \`id\“
// section headings (not table PK columns) to the owned id prefixes.
var declaresByHeading = map[string][]string{
	"boss_catalog.md":           {"boss."},
	"class_skill_catalog.md":    {"skill.", "spatial.", "effect."},
	"equipment_catalog.md":      {"set."},
	"item_catalog.md":           {"item.", "fishing."},
	"npc_shop_catalog.md":       {"shop."},
	"quest_catalog.md":          {"quest.", "daily.", "bounty.", "quest_object.", "quest_hazard.", "platform."},
	"dungeon_catalog.md":        {"dungeon.", "endgame.", "stage.", "secret."},
	"cosmetic_catalog.md":       {"cosmetic.", "feat.", "product."},
	"soul_catalog.md":           {"soul.", "effect.soul."},
	"build_catalog.md":          {"meridian.", "node.", "formation.", "resonance."},
	"spirit_beast_catalog.md":   {"beast."},
	"economy_catalog.md":        {"economy."},
	"world_event_catalog.md":    {"event."},
	"encounter_catalog.md":      {"encounter.", "zone."},
	"progression_route.md":      {"progression."},
	"balance_validation.md":     {"gate."},
	"integration_validation.md": {"rule."},
}

// refColumns maps a column name to the id namespaces its cells must resolve
// into. Applied to every table column of that name in any catalog.
var refColumns = map[string][]string{
	"drop_table_id":               {"drop."},
	"soul_id":                     {"soul."},
	"source_id":                   {"monster.", "boss."},
	"map_id":                      {"map."},
	"dungeon_id":                  {"dungeon.", "endgame."},
	"featured_dungeon_id":         {"dungeon.", "endgame."},
	"reward_table_id":             {"drop.", "reward_table."},
	"item_id":                     {"item.", "fishing."},
	"material_id":                 {"item.material."},
	"Material Required":           {"item.material."},
	"monster":                     {"monster."},
	"Source monster":              {"monster."},
	"Source soul":                 {"soul."},
	"Source boss":                 {"boss."},
	"Source":                      {"monster.", "soul.", "boss.", "item.", "fishing.", "relic."},
	"output":                      {"item."},
	"extra_output":                {"item."},
	"currency":                    {"currency."},
	"space_id":                    {"map.", "dungeon.", "instance."},
	"Title reward (T3)":           {"cosmetic."},
	"Title (T3)":                  {"cosmetic."},
	"Quest":                       {"quest."},
	"quest_giver":                 {"npc.", "quest."},
	"set":                         {"set.", "item.", "fishing."},
	"material_id | minimum level": {"item.material."},
}

// refCellSkips are sentinel cell values that never resolve as references.
var refCellSkips = map[string]bool{
	"": true, "none": true, "—": true, "-": true, "N/A": true,
}

// declaredTokensInFile: every backticked id matching one of these prefixes in
// the named file is a declaration site (content authored in prose/list form
// rather than in a PK table).
var declaredTokensInFile = map[string][]string{
	"spirit_beast_catalog.md":   {"beast.skill.", "beast."},
	"npc_shop_catalog.md":       {"npc.", "shop.", "offer."},
	"drop_tables.md":            {"drop.", "reward_table."},
	"atlas_catalog.md":          {"cosmetic.", "relic.", "buff."},
	"world_route_catalog.md":    {"chest.", "fishing_spot.", "bonfire.", "map."},
	"quest_catalog.md":          {"quest.", "template.", "platform.", "quest_object.", "quest_hazard.", "daily.", "bounty."},
	"dungeon_catalog.md":        {"dungeon.", "stage.", "endgame.", "secret."},
	"cosmetic_catalog.md":       {"cosmetic.", "feat.", "product."},
	"build_catalog.md":          {"meridian.", "node.", "formation.", "resonance."},
	"world_event_catalog.md":    {"event."},
	"encounter_catalog.md":      {"encounter.", "zone.", "reward_table."},
	"progression_route.md":      {"progression."},
	"class_skill_catalog.md":    {"skill.", "effect.", "spatial."},
	"equipment_catalog.md":      {"set."},
	"soul_catalog.md":           {"soul.", "effect.soul."},
	"economy_catalog.md":        {"economy.", "table."},
	"item_catalog.md":           {"item.", "fishing.", "buff."},
	"monster_catalog.md":        {"monster."},
	"boss_catalog.md":           {"boss."},
	"map_spawn_catalog.md":      {"spawn."},
	"balance_validation.md":     {"gate."},
	"integration_validation.md": {"rule."},
}

// openNamespaces are dotted-id prefixes that are not owned by a catalog (they
// refer to runtime state, other specs, or anchors that arrive with geometry
// exports). They are never UNRESOLVED_REFERENCE candidates.
var openNamespaces = map[string]bool{
	"progression": true, "anchor": true, "attack": true, "asset": true,
	"class": true, "life_skill": true, "pvp": true, "reward": true,
	"bonfire": true, "first_session": true, "fishing_spot": true,
	"status_effects": true, "skills": true, "stats": true, "classes": true,
	"seasons": true, "reward_claims": true, "world_rules": true,
	"spawning": true, "soul_contracts": true, "spirit_beasts": true,
	"movement": true, "physics_geometry_contract": true, "crafting": true,
	"economy": true, "guild_war": true, "bosses": true,
	"crafting_catalog": true, "integration_validation": true,
	"balance_validation": true, "atlas_catalog": true, "boss_catalog": true,
	"class_skill_catalog": true, "cosmetic_catalog": true,
	"dungeon_catalog": true, "encounter_catalog": true,
	"equipment_catalog": true, "item_catalog": true,
	"map_spawn_catalog": true, "monster_catalog": true,
	"npc_shop_catalog": true, "quest_catalog": true, "soul_catalog": true,
	"world_event_catalog": true, "world_route_catalog": true,
	"presentation_asset_manifest": true, "build_catalog": true,
	"spirit_beast_catalog": true, "drop_tables": true,
	"progression_route": true, "economy_catalog": true, "skin": true,
}

// namespaceOf returns the first dotted segment of an id.
func namespaceOf(id string) string {
	if i := strings.IndexByte(id, '.'); i > 0 {
		return id[:i]
	}
	return ""
}

// collectIDs gathers declared ids for a file: PK-column values, heading ids
// matching declared prefixes, and first-column `_id` values.
func collectIDs(f *DocFile, pk string, headings []string) map[string]int {
	ids := map[string]int{}
	// First-column `_id` declarations (any table whose leftmost column is an
	// *_id column declares its rows' ids).
	for _, t := range f.Tables {
		if len(t.Header) == 0 {
			continue
		}
		first := t.Header[0]
		if !strings.HasSuffix(first, "_id") && !strings.Contains(first, "_id /") {
			continue
		}
		for _, r := range t.Rows {
			v := strings.TrimSpace(r.Order[0])
			if IsID(v) {
				ids[v] = r.Line
			}
		}
	}
	// Heading declarations.
	for _, h := range f.HeadingIDs {
		for _, p := range headings {
			if strings.HasPrefix(h.ID, p) {
				ids[h.ID] = h.Line
			}
		}
	}
	// Prose/list declarations: every backticked id in this file matching one of
	// its declared prefixes is authored content (not a reference).
	if prefixes, ok := declaredTokensInFile[base(f.RelPath)]; ok {
		for li, line := range f.Lines {
			for _, bt := range backtickRe.FindAllStringSubmatch(line, -1) {
				for _, p := range prefixes {
					if strings.HasPrefix(bt[1], p) && IsID(bt[1]) {
						ids[bt[1]] = li + 1
					}
				}
			}
		}
	}
	return ids
}

// base returns the last path element.
func base(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

// collectRows gathers PK rows for the catalog. Catalogs legitimately repeat a
// PK across different aspect tables (roster vs bounds vs spawn tables), so
// duplicate detection applies within tables sharing the same header signature:
// two rows with the same PK in the same aspect = DUPLICATE_PRIMARY_KEY.
func collectRows(f *DocFile, pk string, ds *Diagnostics) map[string]Row {
	rows := map[string]Row{}
	if pk == "" {
		return rows
	}
	seenBySignature := map[string]map[string]int{}
	for _, t := range f.TablesWithHeader(pk) {
		sig := strings.Join(t.Header, "|")
		seen := seenBySignature[sig]
		if seen == nil {
			seen = map[string]int{}
			seenBySignature[sig] = seen
		}
		for _, r := range t.Rows {
			v := strings.TrimSpace(r.Cells[pk])
			if v == "" {
				continue
			}
			if prev, dup := seen[v]; dup {
				ds.Add(f.RelPath, r.Line, 1, CodeDuplicatePrimaryKey,
					pk+"="+v+" also defined at line "+itoa(prev))
				continue
			}
			seen[v] = r.Line
			if _, have := rows[v]; !have {
				rows[v] = r
			}
		}
	}
	return rows
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

// validateReferences checks declared reference columns and a global backtick
// sweep: any backticked id whose namespace is closed (declared somewhere)
// must resolve to a declared id.
func validateReferences(files, aux map[string]*DocFile, idIndex map[string]string, ds *Diagnostics) {
	// 1. Declared column rules.
	for name, f := range files {
		for _, t := range f.Tables {
			for _, col := range t.Header {
				nss, ok := refColumns[col]
				if !ok {
					continue
				}
				for _, r := range t.Rows {
					checkRefCell(name, r, col, nss, idIndex, ds)
				}
			}
		}
	}
	// 2. Global sweep: backticked tokens in closed namespaces must resolve.
	// Only space-contributing aux files (pvp.md, guild_war.md) are swept: the
	// other aux docs are normative specs that legitimately cite example ids.
	sweep := map[string]*DocFile{}
	for n, f := range files {
		sweep[n] = f
	}
	for _, rel := range []string{"docs/03_systems/pvp.md", "docs/03_systems/guild_war.md"} {
		if f := aux[rel]; f != nil {
			sweep[rel] = f
		}
	}
	for name, f := range sweep {
		for li, line := range f.Lines {
			for _, tok := range backtickRe.FindAllStringSubmatch(line, -1) {
				id := tok[1]
				if strings.HasSuffix(id, ".md") {
					continue
				}
				ns := namespaceOf(id)
				if ns == "" || openNamespaces[ns] {
					continue
				}
				if _, declared := idIndex[id]; declared {
					continue
				}
				// Wildcard family references (`ns.x.*`) are patterns, not
				// concrete references — and several catalogs document families
				// that intentionally do not exist ("has no drop.quest.*").
				// Templates containing <> are patterns too.
				if strings.HasSuffix(id, ".*") || strings.Contains(line, "`"+id+".*`") ||
					strings.Contains(line, "`"+id+".*") || strings.Contains(id, "<") {
					continue
				}
				// A token that is not even in id grammar is prose, not a ref.
				if !IsID(id) {
					continue
				}
				// Closed namespaces: flag only tokens whose namespace owns ids.
				if nsClosed(ns, idIndex) {
					ds.Add(name, li+1, 1, CodeUnresolvedReference,
						"unresolved id `"+id+"`")
				}
			}
		}
	}
}

// nsClosed reports whether any declared id exists under the namespace.
func nsClosed(ns string, idIndex map[string]string) bool {
	p := ns + "."
	for id := range idIndex {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

// checkRefCell validates one reference-column cell.
func checkRefCell(file string, r Row, col string, nss []string, idIndex map[string]string, ds *Diagnostics) {
	raw := strings.TrimSpace(r.Cells[col])
	if refCellSkips[raw] {
		return
	}
	// Cells may carry several backticked or bare ids.
	var tokens []string
	for _, bt := range backtickRe.FindAllStringSubmatch(raw, -1) {
		tokens = append(tokens, bt[1])
	}
	if len(tokens) == 0 {
		tokens = ExtractIDs(raw)
	}
	// Last resort: whole-cell bare id.
	if len(tokens) == 0 && IsID(raw) {
		tokens = []string{raw}
	}
	for _, tok := range tokens {
		allowed := false
		for _, p := range nss {
			if strings.HasPrefix(tok, p) {
				allowed = true
				break
			}
		}
		if !allowed {
			continue // token of another namespace inside the cell — swept separately
		}
		if strings.Contains(raw, tok+".*") || strings.Contains(raw, tok+"*") {
			// Family reference like `item.material.linh_dan.*` — resolves when
			// any declared id exists under the prefix.
			if !nsClosed(tok, idIndex) {
				ds.Add(file, r.Line, 1, CodeUnresolvedReference,
					col+" references unknown family `"+tok+".*`")
			}
			continue
		}
		if _, ok := idIndex[tok]; !ok {
			ds.Add(file, r.Line, 1, CodeUnresolvedReference,
				col+" references unknown `"+tok+"`")
		}
	}
}
