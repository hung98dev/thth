package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// CatalogSet is the compiled row set of one launch catalog file.
type CatalogSet struct {
	File  string `json:"file"`
	PKCol string `json:"pk"`
	// Rows are the primary-keyed rows (from PK tables and ID headings).
	Rows map[string]Row `json:"-"`
	// IDs are all declared identifiers owned by this catalog (PK values,
	// ID-bearing section headings, and generated expansion IDs).
	IDs map[string]int `json:"-"` // id -> source line
}

// EquipmentItem is one expanded equipment row: item.eq.<tier>.<set>.<slot>.
type EquipmentItem struct {
	ItemID      string             `json:"item_id"`
	RecipeID    string             `json:"recipe_id"`
	SetKey      string             `json:"set_key"`
	Tier        string             `json:"tier"`
	Slot        string             `json:"slot"`
	Element     string             `json:"element"`
	Rarity      string             `json:"rarity"`
	Stats       map[string]float64 `json:"stats"`
	Binding     string             `json:"binding"`
	MaterialID  string             `json:"material_id"`
	MaterialQty int64              `json:"material_qty"`
	CommonCost  int64              `json:"common_cost"`
}

// Portal is one directional portal edge (52 total per world_route_catalog).
type Portal struct {
	PortalID    string `json:"portal_id"`
	SourceMap   string `json:"source_map"`
	DestSpace   string `json:"dest_space"`
	DestSpawn   string `json:"dest_spawn"`
	Requirement string `json:"requirement,omitempty"`
	ReturnSpawn string `json:"return_spawn,omitempty"`
}

// SpawnGroup is one persistent spawn group (54) or night-rare group (6).
type SpawnGroup struct {
	GroupID        string   `json:"spawn_group_id"`
	MapID          string   `json:"map_id"`
	AnchorID       string   `json:"anchor_id"`
	MonsterPool    []string `json:"monster_pool"`
	MaxAlive       int64    `json:"max_alive"`
	RespawnSeconds int64    `json:"respawn_seconds"`
	Activation     string   `json:"activation"`
	Kind           string   `json:"kind"` // normal | elite | rare_night
}

// SpaceDef is the compiled playable-space geometry index entry.
type SpaceDef struct {
	SpaceID       string  `json:"space_id"`
	SpaceKind     string  `json:"space_kind"` // WORLD|DUNGEON|FINALE|PVP|GUILD_WAR
	WidthScreens  float64 `json:"width_screens"`
	HeightScreens float64 `json:"height_screens"`
	BoundsMaxX    float64 `json:"bounds_max_x"`
	BoundsMaxY    float64 `json:"bounds_max_y"`
	LayoutProfile string  `json:"layout_profile"`
	Topology      string  `json:"topology"`
	SourceFile    string  `json:"source_file"`
}

// GeomVariant is one compiled typed skill geometry (union per skills.md).
type GeomVariant struct {
	Kind   string             `json:"kind"`
	Params map[string]float64 `json:"params"`
}

// SkillGeom is one compiled active/basic skill geometry row.
type SkillGeom struct {
	SkillID     string       `json:"skill_id"`
	Execution   string       `json:"execution_type"`
	Targeting   string       `json:"targeting_mode"`
	Tags        []string     `json:"tags"`
	SpeedStat   string       `json:"speed_stat"`
	StartupMS   int64        `json:"startup_ms"`
	ActiveMS    int64        `json:"active_ms"`
	RecoveryMS  int64        `json:"recovery_ms"`
	BaseCD      float64      `json:"base_cd_s"`
	CostMP      int64        `json:"cost_mp"`
	Geometry    GeomVariant  `json:"geometry"`
	AirGeometry *GeomVariant `json:"air_geometry,omitempty"`
}

// SecondaryEffect is one compiled spatial secondary-effect row.
type SecondaryEffect struct {
	EffectID   string   `json:"spatial_effect_id"`
	Sources    []string `json:"sources"`
	Origin     string   `json:"origin_shape"`
	Resolution string   `json:"resolution"`
	TargetCap  string   `json:"target_cap"`
	// Radius/Distance extracted for the ADR-0047 band check (0 = n/a).
	RadiusM float64 `json:"radius_m"`
	// Displaces is true when resolution forces target movement (push/pull).
	Displaces bool `json:"displaces"`
}

// MonsterDef is one compiled monster row with derived fields.
type MonsterDef struct {
	MonsterID   string   `json:"monster_id"`
	Rank        string   `json:"rank"`
	Level       int64    `json:"level"`
	Element     string   `json:"element"`
	Movement    string   `json:"movement"`
	Combat      string   `json:"combat"`
	SizeProfile string   `json:"size_profile"`
	BaseEXP     int64    `json:"base_exp"`
	DropTableID string   `json:"drop_table_id"`
	AttackIDs   []string `json:"attack_ids"`
}

// BossDef is one compiled boss row.
type BossDef struct {
	BossID      string `json:"boss_id"`
	Level       int64  `json:"level"`
	Mode        string `json:"mode"`
	SpaceID     string `json:"space_id"`
	SizeProfile string `json:"size_profile"`
	Element     string `json:"element"`
	BaseEXP     int64  `json:"base_exp"`
	DropTableID string `json:"drop_table_id"`
}

// CandidateSnapshot is the fully-compiled in-memory content revision. It is
// immutable once produced; activation (IMP-004) swaps it atomically.
type CandidateSnapshot struct {
	SchemaVersion   int    `json:"schema_version"`
	ContentRevision string `json:"content_revision"`

	Catalogs  map[string]*CatalogSet `json:"-"`
	Equipment []EquipmentItem        `json:"equipment"`
	Recipes   []RecipeRow            `json:"-"`
	Portals   []Portal               `json:"portals"`
	Spawns    []SpawnGroup           `json:"spawn_groups"`
	Spaces    []SpaceDef             `json:"spaces"`
	Monsters  []MonsterDef           `json:"monsters"`
	Bosses    []BossDef              `json:"bosses"`
	Skills    []SkillGeom            `json:"skill_geometries"`
	Secondary []SecondaryEffect      `json:"secondary_effects"`

	// EntityProfiles maps monster/boss id -> resolved size_profile.
	EntityProfiles map[string]string `json:"entity_profiles"`
	// IDIndex maps every declared/derived identifier to its owner file.
	IDIndex map[string]string `json:"-"`
}

// RecipeRow is one compiled crafting recipe (equipment expansion + authored).
type RecipeRow struct {
	RecipeID   string   `json:"recipe_id"`
	OutputID   string   `json:"output_id"`
	Inputs     []string `json:"inputs"`
	CommonCost int64    `json:"common_cost"`
}

// Get returns the active row of a catalog by primary key.
func (s *CandidateSnapshot) Get(catalog, pk string) (Row, bool) {
	c, ok := s.Catalogs[catalog]
	if !ok {
		return Row{}, false
	}
	r, ok := c.Rows[pk]
	return r, ok
}

// ContentRevisionHash computes the canonical SHA-256 over the 24 launch
// catalogs per contract §5: rows sorted by primary key, cells joined by
// '\t', lines terminated by '\n'. Fenced-block lines contribute too — they
// carry authored data (spawn pools, portal edges, rosters).
func ContentRevisionHash(files map[string]*DocFile) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		f := files[name]
		b.WriteString(fmt.Sprintf("file\t%s\n", name))
		// Heading-carried identifiers contribute to identity (catalogs such as
		// soul_catalog declare content only via `## <id>` headings).
		hids := append([]HeadingID(nil), f.HeadingIDs...)
		sort.Slice(hids, func(i, j int) bool { return hids[i].ID < hids[j].ID })
		for _, h := range hids {
			b.WriteString(fmt.Sprintf("H\t%s\n", h.ID))
		}
		for _, t := range f.Tables {
			b.WriteString("T\t" + strings.Join(t.Header, "\t") + "\n")
			rows := append([]Row(nil), t.Rows...)
			sort.Slice(rows, func(i, j int) bool {
				// Primary-key ordering; fall back to first cell.
				pk := sortKeyFor(t.Header)
				ri, rj := rows[i].Cells[pk], rows[j].Cells[pk]
				if ri == rj {
					return rows[i].Line < rows[j].Line
				}
				return ri < rj
			})
			for _, r := range rows {
				b.WriteString("R\t" + strings.Join(r.Order, "\t") + "\n")
			}
		}
		// Fenced blocks carry authored data (spawn pools, portal edges,
		// SMALL_ROSTER, element layouts, portal blocks): normalize each line
		// (trim, drop blanks) and include them in file order so two
		// behaviorally different candidates never share a revision.
		inFence := false
		for _, l := range f.Lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "```") {
				inFence = !inFence
				continue
			}
			if !inFence || t == "" {
				continue
			}
			b.WriteString("F\t" + t + "\n")
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// sortKeyFor picks the sort column: the catalog-declared PK column if present
// in this table's header, else the first column.
func sortKeyFor(header []string) string {
	for _, h := range header {
		if strings.HasSuffix(h, "_id") || h == "level" || h == "Level" {
			return h
		}
	}
	if len(header) == 0 {
		return ""
	}
	return header[0]
}
