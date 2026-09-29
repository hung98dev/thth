package config

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Finite expansions per content_authoring_contract.md §4. Every expansion is
// deterministic: derived solely from catalog tables and the normative edge
// lists embedded in the catalog code blocks.

var statTermRe = regexp.MustCompile(`([0-9]+\.?[0-9]*)([ADHM])\s+([A-Z_]+)`)

// tierStatRe matches authored "tier <STAT>" fixed-stat terms (ring/charm
// utility columns), resolved against the catalog's tier utility table.
var tierStatRe = regexp.MustCompile(`\btier\s+([A-Z_]+)`)

// expandEquipment expands equipment_catalog.md's 12 sets x 14 slots into 168
// concrete item.eq.* rows plus their 168 recipe.eq.* crafting recipes.
func expandEquipment(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	// Tier budget table.
	tierBase := map[string]struct {
		Rarity         string
		A, D, H, M     float64
		SecondaryRolls int64
	}{}
	for _, t := range f.TablesWithHeader("Tier", "A", "D", "H", "M") {
		for _, r := range t.Rows {
			tier := r.Cells["Tier"]
			a, ea := ParseFloat(r.Cells["A"])
			d, ed := ParseFloat(r.Cells["D"])
			h, eh := ParseFloat(r.Cells["H"])
			m, em := ParseFloat(r.Cells["M"])
			sr, es := ParseInt(r.Cells["secondary rolls"])
			if ea != nil || ed != nil || eh != nil || em != nil || es != nil {
				ds.Add(f.RelPath, r.Line, 1, CodeTableSyntaxError, "tier budget row not numeric")
				continue
			}
			tierBase[tier] = struct {
				Rarity         string
				A, D, H, M     float64
				SecondaryRolls int64
			}{Rarity: r.Cells["rarity"], A: a, D: d, H: h, M: m, SecondaryRolls: sr}
		}
	}
	// Tier fixed utility values: "Tier | <slot> <STAT> | ..." table maps a
	// slot's `tier STAT` fixed-stat term to its per-tier flat fraction.
	tierUtility := map[string]map[string]float64{} // "slot STAT" -> tier -> value
	for _, t := range f.Tables {
		utilityCols := map[string]string{} // header -> key
		for _, h := range t.Header {
			if h == "Tier" {
				continue
			}
			parts := strings.Fields(h)
			if len(parts) == 2 && IsID(parts[0]) {
				utilityCols[h] = parts[0] + " " + parts[1]
			}
		}
		if len(utilityCols) == 0 {
			continue
		}
		for _, r := range t.Rows {
			tier := r.Cells["Tier"]
			if !strings.HasPrefix(tier, "T") {
				continue
			}
			for h, key := range utilityCols {
				v, e := ParseFloat(r.Cells[h])
				if e != nil {
					continue
				}
				if tierUtility[key] == nil {
					tierUtility[key] = map[string]float64{}
				}
				tierUtility[key][tier] = v
			}
		}
	}
	// Slot fixed-stat table: its row PKs are the canonical 14-slot list.
	slotStats := map[string]string{}
	var slots []string
	for _, t := range f.TablesWithHeader("slot", "fixed base stats") {
		for _, r := range t.Rows {
			slotStats[r.Cells["slot"]] = r.Cells["fixed base stats"]
			slots = append(slots, r.Cells["slot"])
		}
	}
	if len(slots) != 14 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("equipment slot table must declare 14 slots, found %d", len(slots)))
	}
	// Set roster: heading `set.tN.<key>` + `key:`/`layout:` lines beneath it.
	type setDef struct {
		id, key, layout string
		line            int
	}
	var sets []setDef
	cur := -1
	keyRe := regexp.MustCompile(`^key:\s*` + "`?" + `([a-z0-9_]+)` + "`?")
	layoutRe := regexp.MustCompile(`^layout:\s*` + "`?" + `([AB])` + "`?")
	setHeadRe := regexp.MustCompile(`^#+\s*` + "`" + `(set\.t[0-9]+\.[a-z0-9_]+)` + "`")
	for i, l := range f.Lines {
		if m := setHeadRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			sets = append(sets, setDef{id: m[1], line: i + 1})
			cur = len(sets) - 1
			continue
		}
		if cur < 0 {
			continue
		}
		if m := keyRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			sets[cur].key = m[1]
		}
		if m := layoutRe.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			sets[cur].layout = m[1]
		}
	}
	if len(sets) != 12 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("equipment set roster must be exactly 12, found %d", len(sets)))
	}
	// Element layouts A/B parsed from the catalog code blocks.
	layouts := parseElementLayouts(f)
	if len(layouts) != 2 {
		ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "element layouts A/B missing or malformed")
	}
	for _, s := range sets {
		tier := ""
		if m := regexp.MustCompile(`^set\.t([1-6])\.`).FindStringSubmatch(s.id); m != nil {
			tier = "T" + m[1]
		}
		base, ok := tierBase[tier]
		if !ok {
			ds.Add(f.RelPath, s.line, 1, CodeValueOutOfBounds, "set "+s.id+" has no tier budget row "+tier)
			continue
		}
		if s.key == "" || s.layout == "" {
			ds.Add(f.RelPath, s.line, 1, CodeTableSyntaxError, "set "+s.id+" missing key:/layout:")
			continue
		}
		for _, slot := range slots {
			it := EquipmentItem{
				ItemID:   "item.eq." + strings.ToLower(tier) + "." + s.key + "." + slot,
				RecipeID: "recipe.eq." + strings.ToLower(tier) + "." + s.key + "." + slot,
				SetKey:   s.key,
				Tier:     tier,
				Slot:     slot,
				Element:  layoutElem(layouts, s.layout, slot),
				Rarity:   base.Rarity,
				Binding:  "UNBOUND",
				Stats:    map[string]float64{},
			}
			// Evaluate "<coef><A|D|H|M> STAT" fixed base stats; authored values
			// round down after multiplication (equipment_catalog §Fixed Base
			// Stats by Slot).
			spec := slotStats[slot]
			for _, tm := range statTermRe.FindAllStringSubmatch(spec, -1) {
				coef, _ := strconv.ParseFloat(tm[1], 64)
				var statBase float64
				switch tm[2] {
				case "A":
					statBase = base.A
				case "D":
					statBase = base.D
				case "H":
					statBase = base.H
				case "M":
					statBase = base.M
				}
				it.Stats[tm[3]] += math.Floor(coef * statBase)
			}
			// `tier <STAT>` terms resolve to the per-tier utility value.
			for _, tm := range tierStatRe.FindAllStringSubmatch(spec, -1) {
				key := slot + " " + tm[1]
				v, ok := tierUtility[key][tier]
				if !ok {
					ds.Add(f.RelPath, s.line, 1, CodeUnresolvedReference,
						"no tier utility value for "+key+" at "+tier)
					continue
				}
				it.Stats[tm[1]] += v
			}
			if len(it.Stats) == 0 {
				ds.Add(f.RelPath, s.line, 1, CodeTableSyntaxError,
					"slot "+slot+" has no evaluable fixed base stats")
			}
			snap.Equipment = append(snap.Equipment, it)
			snap.IDIndex[it.ItemID] = f.RelPath
			snap.IDIndex[it.RecipeID] = f.RelPath
			snap.Recipes = append(snap.Recipes, RecipeRow{
				RecipeID: it.RecipeID,
				OutputID: it.ItemID,
			})
		}
		// Set effect/support ids declared by the set's bonuses.
		for _, n := range []string{"2", "4", "6"} {
			snap.IDIndex["effect.set."+strings.ToLower(tier)+"."+s.key+"."+n] = f.RelPath
		}
		snap.IDIndex["support.set."+s.key] = f.RelPath
	}
}

// layoutElem resolves a slot's element under layout A or B.
func layoutElem(layouts map[string]map[string]string, layout, slot string) string {
	if m, ok := layouts[layout]; ok {
		return m[slot]
	}
	return ""
}

// parseElementLayouts reads the "## Layout A" / "## Layout B" code blocks:
// lines like "weapon KIM".
func parseElementLayouts(f *DocFile) map[string]map[string]string {
	out := map[string]map[string]string{}
	var cur string
	inBlock := false
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			inBlock = !inBlock
			continue
		}
		if m := regexp.MustCompile(`^##\s*Layout\s+([AB])`).FindStringSubmatch(t); m != nil {
			cur = m[1]
			continue
		}
		if !inBlock || cur == "" {
			continue
		}
		parts := strings.Fields(t)
		if len(parts) == 2 && elements[parts[1]] {
			if out[cur] == nil {
				out[cur] = map[string]string{}
			}
			out[cur][parts[0]] = parts[1]
		}
		if i > 400 && cur != "" && !inBlock {
			cur = ""
		}
	}
	return out
}

// portal line in a fenced block: "a <-> b" (intra/cross) or the expanded
// "portal.<src>.to.<dst>" blocks with indented fields.
var portalEdgeRe = regexp.MustCompile(`^(map\.[a-z0-9_.]+)\s*<->\s*(map\.[a-z0-9_.]+)`)

// expandPortals expands world_route_catalog.md's normative edge lists into
// the 52 directional portal definitions.
func expandPortals(f *DocFile, mapEntrySpawn map[string]string, ds *Diagnostics) []Portal {
	var portals []Portal
	seen := map[string]bool{}
	add := func(line int, src, dstSpace, dstSpawn, req, ret, dstOverride string) {
		srcSuffix := strings.TrimPrefix(src, "map.")
		var dstSuffix string
		switch {
		case strings.HasPrefix(dstSpace, "map."):
			dstSuffix = strings.TrimPrefix(dstSpace, "map.")
		case strings.HasPrefix(dstSpace, "dungeon."):
			dstSuffix = "dungeon_" + strings.TrimPrefix(dstSpace, "dungeon.")
		default:
			dstSuffix = strings.ReplaceAll(strings.TrimPrefix(dstSpace, "instance.finale."), ".", "_")
		}
		if dstOverride != "" {
			dstSuffix = dstOverride
		}
		id := "portal." + srcSuffix + ".to." + dstSuffix
		if seen[id] {
			ds.Add(f.RelPath, line, 1, CodeDuplicatePrimaryKey, "duplicate portal "+id)
			return
		}
		seen[id] = true
		portals = append(portals, Portal{
			PortalID:    id,
			SourceMap:   src,
			DestSpace:   dstSpace,
			DestSpawn:   dstSpawn,
			Requirement: req,
			ReturnSpawn: ret,
		})
	}

	// Intra-region undirected edges in fenced blocks: "map.X <-> map.Y".
	inCode := false
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if !inCode {
			continue
		}
		if m := portalEdgeRe.FindStringSubmatch(t); m != nil {
			add(i+1, m[1], m[2], mapEntrySpawn[m[2]], "", "", "")
			add(i+1, m[2], m[1], mapEntrySpawn[m[1]], "", "", "")
		}
	}
	// Cross-region edges from the normative table (forward requirement).
	for _, t := range f.TablesWithHeader("edge", "forward requirement") {
		for _, r := range t.Rows {
			m := portalEdgeRe.FindStringSubmatch(strings.ReplaceAll(r.Cells["edge"], "`", ""))
			if m == nil {
				continue
			}
			add(r.Line, m[1], m[2], mapEntrySpawn[m[2]], r.Cells["forward requirement"], "", "")
			add(r.Line, m[2], m[1], mapEntrySpawn[m[1]], "", "", "")
		}
	}
	// Dungeon-entry + finale-entry blocks: "portal.<src>.to.<dst>" followed by
	// indented "source =", "destination =", "requirement =", ...
	kv := regexp.MustCompile(`^([a-z_ ]+?)\s*=\s*(.+)$`)
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "portal.") || strings.Contains(t, "<") {
			// skip the ID-template line, which uses <placeholders>
			continue
		}
		src, dst, enc, req, ret := "", "", "", "", ""
		for j := i + 1; j < len(f.Lines) && j < i+10; j++ {
			b := strings.TrimSpace(f.Lines[j])
			if m := kv.FindStringSubmatch(b); m != nil {
				switch m[1] {
				case "source":
					src = m[2]
				case "destination":
					dst = m[2]
				case "destination space_id":
					dst = m[2]
				case "destination encounter":
					enc = m[2]
				case "requirement":
					req = m[2]
				case "return_spawn":
					ret = m[2]
				}
			}
			if b == "" || strings.HasPrefix(b, "portal.") || strings.HasPrefix(b, "```") {
				if src != "" {
					break
				}
			}
		}
		if src == "" || dst == "" {
			ds.Add(f.RelPath, i+1, 1, CodeTableSyntaxError, "portal block missing source/destination")
			continue
		}
		var dstSpace, dstSpawn, dstOverride string
		switch {
		case strings.HasPrefix(dst, "dungeon."):
			dstSpace = dst
			dstSpawn = "spawn.return." + regionOf(src) + "." + strings.TrimPrefix(dst, "dungeon.")
		case strings.HasPrefix(dst, "instance."):
			dstSpace = dst
			dstSpawn = "spawn.entry." + strings.TrimPrefix(dst, "instance.")
			// Authored id uses the encounter key: ...to.boss_than_trung.
			if enc != "" {
				dstOverride = strings.ReplaceAll(enc, ".", "_")
			}
		case strings.HasPrefix(dst, "boss."):
			dstSpace = "instance.finale.than_trung"
			dstSpawn = "spawn.entry." + strings.TrimPrefix(dstSpace, "instance.")
			dstOverride = strings.ReplaceAll(dst, ".", "_")
		default:
			dstSpace = dst
			dstSpawn = mapEntrySpawn[dst]
		}
		add(i+1, src, dstSpace, dstSpawn, req, ret, dstOverride)
	}
	return portals
}

// regionOf extracts the region token from a map id: map.<region>.<key>.
func regionOf(mapID string) string {
	p := strings.Split(mapID, ".")
	if len(p) >= 2 {
		return p[1]
	}
	return ""
}

// expandSpawns parses map_spawn_catalog.md fenced spawn blocks: 18 fields x
// 3 groups + 6 night-rare table rows.
func expandSpawns(f *DocFile, ds *Diagnostics) []SpawnGroup {
	var groups []SpawnGroup
	seen := map[string]bool{}
	curMap := ""
	inBlock := false
	var g *SpawnGroup
	flush := func() {
		if g == nil {
			return
		}
		if g.GroupID == "" || g.AnchorID == "" {
			ds.Add(f.RelPath, 0, 0, CodeTableSyntaxError, "spawn group block missing id/anchor")
		}
		kind := "normal"
		if strings.Contains(g.GroupID, ".elite_") {
			kind = "elite"
		}
		g.Kind = kind
		g.Activation = "ALWAYS"
		g.MapID = curMap
		// Pool suffixes resolve to monster.<map_region>.<suffix>.
		region := regionOf(curMap)
		for i, s := range g.MonsterPool {
			if !strings.HasPrefix(s, "monster.") {
				g.MonsterPool[i] = "monster." + region + "." + s
			}
		}
		if seen[g.GroupID] {
			ds.Add(f.RelPath, 0, 0, CodeDuplicatePrimaryKey, "duplicate "+g.GroupID)
		}
		seen[g.GroupID] = true
		groups = append(groups, *g)
		g = nil
	}
	anchorRe := regexp.MustCompile(`^(spawn\.[a-z0-9_.]+)\s*@\s*(anchor\.[a-z0-9_.]+)`)
	kvRe := regexp.MustCompile(`^(pool|max_alive|respawn|respawn_seconds|activation)\s*=\s*(.+)$`)
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "##") {
			flush()
			if m := regexp.MustCompile("`([^`]+)`").FindStringSubmatch(l); m != nil {
				curMap = m[1]
			}
			continue
		}
		if strings.HasPrefix(t, "```") {
			if inBlock {
				flush()
			}
			inBlock = !inBlock
			continue
		}
		if !inBlock {
			continue
		}
		if m := anchorRe.FindStringSubmatch(t); m != nil {
			flush()
			g = &SpawnGroup{GroupID: m[1], AnchorID: m[2]}
			continue
		}
		if g == nil {
			continue
		}
		if m := kvRe.FindStringSubmatch(t); m != nil {
			switch m[1] {
			case "pool":
				g.MonsterPool = SplitList(m[2])
			case "max_alive":
				v, e := ParseInt(m[2])
				if e != nil {
					ds.Add(f.RelPath, i+1, 1, CodeTableSyntaxError, "max_alive not integer")
				} else {
					g.MaxAlive = v
				}
			case "respawn", "respawn_seconds":
				v, e := ParseMagnitude(m[2])
				if e != nil {
					ds.Add(f.RelPath, i+1, 1, CodeTableSyntaxError, "respawn not numeric")
				} else {
					g.RespawnSeconds = int64(v)
				}
			case "activation":
				g.Activation = strings.TrimSpace(m[2])
			}
		}
	}
	flush()
	// Night-rare table rows.
	for _, t := range f.TablesWithHeader("spawn_group_id", "map_id", "monster") {
		for _, r := range t.Rows {
			id := strings.TrimSpace(r.Cells["spawn_group_id"])
			if id == "" {
				continue
			}
			if seen[id] {
				ds.Add(f.RelPath, r.Line, 1, CodeDuplicatePrimaryKey, "duplicate "+id)
				continue
			}
			seen[id] = true
			groups = append(groups, SpawnGroup{
				GroupID:        id,
				MapID:          strings.TrimSpace(r.Cells["map_id"]),
				MonsterPool:    []string{strings.TrimSpace(r.Cells["monster"])},
				MaxAlive:       1,
				RespawnSeconds: 300,
				Activation:     "NIGHT",
				Kind:           "rare_night",
			})
		}
	}
	return groups
}
