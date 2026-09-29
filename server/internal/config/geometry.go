package config

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Playable-space geometry index (contract §4.5): every playable PvE/PvP space
// resolves to exactly one SpaceDef carrying authored span screens, pixel
// bounds, layout profile and required topology.

var spaceKinds = map[string]bool{
	"WORLD": true, "DUNGEON": true, "FINALE": true, "PVP": true, "GUILD_WAR": true,
}

// buildSpaceIndex collects 24 world maps + 5 dungeons + finale + 2 PvP spaces
// + guild war = 33 playable spaces.
func buildSpaceIndex(files, aux map[string]*DocFile, ds *Diagnostics) []SpaceDef {
	var spaces []SpaceDef
	seen := map[string]bool{}
	add := func(s SpaceDef) {
		if s.SpaceID == "" {
			return
		}
		if seen[s.SpaceID] {
			ds.Add(s.SourceFile, 0, 0, CodeDuplicatePrimaryKey, "duplicate space "+s.SpaceID)
			return
		}
		seen[s.SpaceID] = true
		spaces = append(spaces, s)
	}

	// World maps: world_route's map table.
	if f := files["world_route_catalog.md"]; f != nil {
		for _, t := range f.TablesWithHeader("map_id", "span (screens)", "bounds max (m)") {
			for _, r := range t.Rows {
				s := SpaceDef{SpaceID: r.Cells["map_id"], SpaceKind: "WORLD", SourceFile: f.RelPath}
				parseSpanBoundsRow(r, &s, f.RelPath, ds)
				s.LayoutProfile = r.Cells["layout_profile"]
				s.Topology = r.Cells["required traversable topology"]
				add(s)
			}
		}
	}
	// Dungeons: dungeon_catalog's per-dungeon space tables.
	if f := files["dungeon_catalog.md"]; f != nil {
		for _, t := range f.Tables {
			// Tables keyed by "dungeon_id / space_id" or "space_id" with span/bounds.
			idCol := ""
			for _, h := range t.Header {
				if h == "dungeon_id / space_id" || h == "space_id" || h == "dungeon_id" {
					idCol = h
					break
				}
			}
			if idCol == "" {
				continue
			}
			hasSpan, hasBounds := false, false
			for _, h := range t.Header {
				if strings.Contains(h, "span") {
					hasSpan = true
				}
				if strings.Contains(h, "bounds") {
					hasBounds = true
				}
			}
			if !hasSpan || !hasBounds {
				continue
			}
			for _, r := range t.Rows {
				id := firstID(r.Cells[idCol])
				if !strings.HasPrefix(id, "dungeon.") {
					continue
				}
				s := SpaceDef{SpaceID: id, SpaceKind: "DUNGEON", SourceFile: f.RelPath}
				parseSpanBoundsRow(r, &s, f.RelPath, ds)
				s.LayoutProfile = r.Cells["layout_profile"]
				s.Topology = r.Cells["required traversable topology"]
				add(s)
			}
		}
	}
	// Kind-tagged space tables (finale arena etc.): any catalog table carrying
	// both a space-id column and a `kind` column whose value is a space kind.
	for _, f := range files {
		for _, t := range f.Tables {
			hasID, hasKind := "", false
			for _, h := range t.Header {
				if h == "space_id" || strings.HasSuffix(h, "/ space_id") {
					hasID = h
				}
				if h == "kind" {
					hasKind = true
				}
			}
			if hasID == "" || !hasKind {
				continue
			}
			for _, r := range t.Rows {
				id := firstID(r.Cells[hasID])
				kind := strings.TrimSpace(r.Cells["kind"])
				if !spaceKinds[kind] {
					continue
				}
				s := SpaceDef{SpaceID: id, SpaceKind: kind, SourceFile: f.RelPath}
				parseSpanBoundsRow(r, &s, f.RelPath, ds)
				s.LayoutProfile = r.Cells["layout_profile"]
				s.Topology = r.Cells["required topology"]
				if s.Topology == "" {
					s.Topology = r.Cells["required traversable topology"]
				}
				add(s)
			}
		}
	}
	// PvP spaces: pvp.md modes table (duel court + five-element arena).
	if f := aux["docs/03_systems/pvp.md"]; f != nil {
		for _, t := range f.Tables {
			idCol := ""
			for _, h := range t.Header {
				if h == "space_id" {
					idCol = h
				}
			}
			if idCol == "" {
				continue
			}
			hasBounds := false
			for _, h := range t.Header {
				if strings.Contains(h, "bounds") {
					hasBounds = true
				}
			}
			if !hasBounds {
				continue
			}
			for _, r := range t.Rows {
				id := firstID(r.Cells[idCol])
				if !strings.HasPrefix(id, "map.pvp.") && !strings.HasPrefix(id, "instance.pvp.") {
					continue
				}
				s := SpaceDef{SpaceID: id, SpaceKind: "PVP", SourceFile: f.RelPath}
				parseSpanBoundsRow(r, &s, f.RelPath, ds)
				s.LayoutProfile = r.Cells["layout_profile"]
				s.Topology = r.Cells["required topology"]
				if s.Topology == "" {
					s.Topology = r.Cells["required traversable topology"]
				}
				add(s)
			}
		}
	}
	// Guild war space: guild_war.md field lines ("space_id = ..." kv pairs).
	if f := aux["docs/03_systems/guild_war.md"]; f != nil {
		fields := map[string]string{}
		inBlock := false
		for _, l := range f.Lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "```") {
				inBlock = !inBlock
				continue
			}
			if !inBlock {
				continue
			}
			if i := strings.Index(t, "="); i > 0 {
				k := strings.TrimSpace(t[:i])
				v := strings.TrimSpace(t[i+1:])
				fields[k] = v
			}
		}
		if id := fields["space_id"]; id != "" {
			s := SpaceDef{SpaceID: id, SpaceKind: "GUILD_WAR", SourceFile: f.RelPath}
			s.LayoutProfile = fields["layout_profile"]
			s.Topology = fields["required_topology"]
			if sp := fields["span"]; sp != "" || fields["span (screens)"] != "" {
				if sp == "" {
					sp = fields["span (screens)"]
				}
				if w, h, ok := parseSpan(sp); ok {
					s.WidthScreens, s.HeightScreens = w, h
				}
			}
			if bm := firstNonEmpty(fields["bounds"], fields["bounds_max"], fields["bounds max"]); bm != "" {
				if x, y, ok := parseBoundsMax(bm); ok {
					s.BoundsMaxX, s.BoundsMaxY = x, y
				} else if x, y, ok := parseRectMax(bm); ok {
					s.BoundsMaxX, s.BoundsMaxY = x, y
				}
			}
			add(s)
		}
	}
	return spaces
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstID extracts the first backticked/id token from a combined cell like
// "`dungeon.x` / `dungeon.x.wave_1`" or a plain "dungeon.x".
func firstID(cell string) string {
	cell = strings.ReplaceAll(cell, "`", "")
	for _, tok := range idTokenRe.FindAllString(cell, -1) {
		return tok
	}
	return strings.TrimSpace(cell)
}

var spanRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*[x×]\s*([0-9]+(?:\.[0-9]+)?)`)
var boundsRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*[x×]\s*([0-9]+(?:\.[0-9]+)?)`)

// parseSpanBoundsRow fills width/height screens + bounds from a row's cells
// and cross-checks bounds against span (bounds = span * screen px: 25.6x14.4).
func parseSpanBoundsRow(r Row, s *SpaceDef, rel string, ds *Diagnostics) {
	for k, v := range r.Cells {
		lk := strings.ToLower(k)
		switch {
		case strings.Contains(lk, "span"):
			if w, h, ok := parseSpan(v); ok {
				s.WidthScreens, s.HeightScreens = w, h
			} else if strings.TrimSpace(v) != "" {
				ds.Add(rel, r.Line, 0, CodeTableSyntaxError, "span not 'WxH': "+v)
			}
		case strings.Contains(lk, "bounds"):
			if x, y, ok := parseBoundsMax(v); ok {
				s.BoundsMaxX, s.BoundsMaxY = x, y
			} else if strings.TrimSpace(v) != "" {
				ds.Add(rel, r.Line, 0, CodeTableSyntaxError, "bounds not 'XxY': "+v)
			}
		}
	}
	// Cross-check: bounds_max = screens * (25.6, 14.4) per physics contract.
	if s.WidthScreens > 0 && s.BoundsMaxX > 0 {
		wantX, wantY := s.WidthScreens*25.6, s.HeightScreens*14.4
		if math.Abs(s.BoundsMaxX-wantX) > 0.05 || math.Abs(s.BoundsMaxY-wantY) > 0.05 {
			ds.Add(rel, r.Line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("space %s bounds %.1fx%.1f inconsistent with span %.2fx%.2f screens",
					s.SpaceID, s.BoundsMaxX, s.BoundsMaxY, s.WidthScreens, s.HeightScreens))
		}
	}
}

func parseSpan(v string) (w, h float64, ok bool) {
	m := spanRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, false
	}
	w, _ = strconv.ParseFloat(m[1], 64)
	h, _ = strconv.ParseFloat(m[2], 64)
	return w, h, w > 0 && h > 0
}

func parseBoundsMax(v string) (x, y float64, ok bool) {
	m := boundsRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, false
	}
	x, _ = strconv.ParseFloat(m[1], 64)
	y, _ = strconv.ParseFloat(m[2], 64)
	return x, y, x > 0 && y > 0
}

// rectMaxRe matches the guild_war bounds form "(0,0)..(128.0m,25.2m)": the
// max corner is the bounds extent.
var rectMaxRe = regexp.MustCompile(`\(\s*(-?[0-9]+(?:\.[0-9]+)?)\s*m?\s*,\s*(-?[0-9]+(?:\.[0-9]+)?)\s*m?\s*\)`)

func parseRectMax(v string) (x, y float64, ok bool) {
	var corners [][2]float64
	for _, m := range rectMaxRe.FindAllStringSubmatch(v, -1) {
		cx, _ := strconv.ParseFloat(m[1], 64)
		cy, _ := strconv.ParseFloat(m[2], 64)
		corners = append(corners, [2]float64{cx, cy})
	}
	if len(corners) < 2 {
		return 0, 0, false
	}
	x, y = corners[1][0], corners[1][1]
	return x, y, x > 0 && y > 0
}

// ---- Skill geometry typing (skills.md spatial geometry union + ADR-0047) ----

// geomCallRe matches a geometry constructor cell: KIND(name = v, ...) or
// positional KIND(v, v).
var geomCallRe = regexp.MustCompile(`^([A-Z_]+)\((.*)\)\s*$`)

// geomParams declares the named parameters of each geometry variant.
var geomParams = map[string][]string{
	"SELF":                {},
	"MELEE_BOX":           {"reach_m", "half_height_m"},
	"DIRECTION_BOX":       {"length_m", "half_height_m"},
	"PROJECTILE":          {"max_range_m", "speed_mps", "hit_radius_m"},
	"AREA_SELF":           {"radius_m"},
	"AREA_POSITION":       {"cast_range_m", "radius_m"},
	"SINGLE_TARGET_RANGE": {"range_m"},
	"DASH_LINE":           {"distance_m", "duration_ms", "hit_half_height_m"},
	"MOVE_LINE":           {"distance_m", "duration_ms"},
	"MOVE_CONTACT_LINE":   {"distance_m", "duration_ms", "hit_half_height_m"},
	"BARRIER_POSITION":    {"cast_range_m", "thickness_m", "height_m", "duration_ms"},
}

// paramAliases normalizes authored parameter spellings to the canonical names.
var paramAliases = map[string]map[string]string{
	"MELEE_BOX":           {"reach": "reach_m", "half_height": "half_height_m"},
	"DIRECTION_BOX":       {"length": "length_m", "half_height": "half_height_m"},
	"PROJECTILE":          {"range": "max_range_m", "range_m": "max_range_m", "radius_m": "hit_radius_m", "radius": "hit_radius_m", "speed": "speed_mps"},
	"AREA_SELF":           {"radius": "radius_m"},
	"AREA_POSITION":       {"cast_range": "cast_range_m", "range_m": "cast_range_m", "cast": "cast_range_m", "radius": "radius_m"},
	"SINGLE_TARGET_RANGE": {"range": "range_m"},
	"DASH_LINE":           {"distance": "distance_m", "duration": "duration_ms", "hit_half_height": "hit_half_height_m"},
	"MOVE_LINE":           {"distance": "distance_m", "duration": "duration_ms"},
	"MOVE_CONTACT_LINE":   {"distance": "distance_m", "duration": "duration_ms", "hit_half_height": "hit_half_height_m"},
	"BARRIER_POSITION":    {"cast_range": "cast_range_m", "cast": "cast_range_m", "thickness": "thickness_m", "height": "height_m", "duration": "duration_ms"},
}

// parseGeomCell parses a `geometry` cell like
// "DIRECTION_BOX(length_m = 3.2, half_height_m = 1.2); air PROJECTILE(...)".
func parseGeomCell(rel string, line int, cell string, ds *Diagnostics) (GeomVariant, *GeomVariant) {
	var air *GeomVariant
	parts := strings.Split(cell, ";")
	main := GeomVariant{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		isAir := false
		if strings.HasPrefix(p, "air ") {
			isAir = true
			p = strings.TrimSpace(strings.TrimPrefix(p, "air "))
		}
		g := parseGeomPart(p, line, rel, ds)
		if isAir {
			g2 := g
			air = &g2
		} else {
			main = g
		}
	}
	return main, air
}

// parseGeomPart parses one "KIND(args)" geometry expression.
func parseGeomPart(expr string, line int, rel string, ds *Diagnostics) GeomVariant {
	m := geomCallRe.FindStringSubmatch(expr)
	if m == nil {
		// Bare zero-param kinds are authored without parentheses ("SELF").
		if params, ok := geomParams[expr]; ok && len(params) == 0 {
			return GeomVariant{Kind: expr, Params: map[string]float64{}}
		}
		ds.Add(rel, line, 0, CodeTableSyntaxError, "geometry not KIND(...): "+expr)
		return GeomVariant{Kind: "INVALID"}
	}
	kind := m[1]
	params, ok := geomParams[kind]
	if !ok {
		ds.Add(rel, line, 0, CodeValueOutOfBounds, "geometry kind "+kind+" not in spatial union")
		return GeomVariant{Kind: kind}
	}
	g := GeomVariant{Kind: kind, Params: map[string]float64{}}
	if len(params) == 0 {
		return g
	}
	args := m[2]
	// Named "k = v" segments.
	named := regexp.MustCompile(`([a-z_]+)\s*=\s*(-?[0-9]+(?:\.[0-9]+)?)`)
	pos := map[string]bool{}
	rest := named.ReplaceAllStringFunc(args, func(s string) string {
		sub := named.FindStringSubmatch(s)
		k := sub[1]
		if al, ok := paramAliases[kind][k]; ok {
			k = al
		}
		v, _ := strconv.ParseFloat(sub[2], 64)
		g.Params[k] = v
		pos[k] = true
		return ""
	})
	// Positional leftovers (bare numbers in declared param order).
	var bare []float64
	for _, tok := range strings.Split(rest, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if v, e := ParseMagnitude(tok); e == nil {
			bare = append(bare, v)
		}
	}
	bi := 0
	for _, p := range params {
		if pos[p] {
			continue
		}
		if bi < len(bare) {
			g.Params[p] = bare[bi]
			bi++
		}
	}
	// All declared params must resolve to a positive value.
	for _, p := range params {
		v, ok := g.Params[p]
		if !ok {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				"geometry "+kind+" missing param "+p+" in "+expr)
			continue
		}
		if v <= 0 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("geometry %s param %s must be > 0 (%.3f)", kind, p, v))
		}
	}
	return g
}

// roleBand returns the ADR-0047 numeric band for a geometry kind under the
// skill's role (basic vs active).
func roleBand(kind string, basic, heal bool) (param string, lo, hi float64, ok bool) {
	switch kind {
	case "MELEE_BOX":
		return "reach_m", 1.8, 2.8, true
	case "DIRECTION_BOX":
		if basic {
			return "length_m", 2.8, 3.5, true
		}
		return "length_m", 5.0, 5.5, true
	case "DASH_LINE", "MOVE_LINE", "MOVE_CONTACT_LINE":
		if basic {
			return "distance_m", 2.4, 2.6, true
		}
		return "distance_m", 4.0, 4.5, true
	case "PROJECTILE":
		return "max_range_m", 7.5, 8.5, true
	case "AREA_POSITION":
		return "cast_range_m", 6.5, 7.5, true
	case "AREA_SELF":
		return "radius_m", 2.8, 3.8, true
	case "SINGLE_TARGET_RANGE":
		if heal {
			return "range_m", 7.0, 8.0, true
		}
		return "range_m", 1.8, 2.8, true
	case "BARRIER_POSITION":
		return "cast_range_m", 6.0, 7.0, true
	}
	return "", 0, 0, false
}

// checkSkillBands enforces the ADR-0047 geometry bands on one compiled
// geometry for the named skill row.
func checkSkillBands(rel string, line int, g GeomVariant, basic, heal bool, skillID string, ds *Diagnostics) {
	if p, lo, hi, ok := roleBand(g.Kind, basic, heal); ok {
		v := g.Params[p]
		if v < lo-1e-9 || v > hi+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s %s %s=%.2f outside ADR-0047 band %.2f..%.2f",
					skillID, g.Kind, p, v, lo, hi))
		}
	}
	// Projectile envelope: range + hit_radius must fit the 8.8m budget.
	if g.Kind == "PROJECTILE" {
		env := g.Params["max_range_m"] + g.Params["hit_radius_m"]
		if env > 8.8+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s projectile envelope %.2fm exceeds 8.8m", skillID, env))
		}
	}
	// Area position: outer radius budget.
	if g.Kind == "AREA_POSITION" {
		outer := g.Params["cast_range_m"] + g.Params["radius_m"]
		if outer > 11.0+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s area outer %.2fm exceeds 11.0m", skillID, outer))
		}
		r := g.Params["radius_m"]
		if r < 2.5-1e-9 || r > 3.5+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s area radius %.2fm outside 2.5..3.5m", skillID, r))
		}
	}
	if g.Kind == "BARRIER_POSITION" {
		if th := g.Params["thickness_m"]; th < 0.5-1e-9 || th > 1.0+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s barrier thickness %.2fm outside 0.5..1.0m", skillID, th))
		}
		if h := g.Params["height_m"]; h < 3.0-1e-9 || h > 4.5+1e-9 {
			ds.Add(rel, line, 0, CodeValueOutOfBounds,
				fmt.Sprintf("skill %s barrier height %.2fm outside 3.0..4.5m", skillID, h))
		}
	}
}
