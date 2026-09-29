package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Skill catalog compilation: roster rows carry execution/targeting/tags; the
// per-class "Action Specifications" tables carry the typed geometry the
// compiler must materialize (45 primary rows: 20 basics + 25 actives).

var executionTypes = map[string]bool{
	"INSTANT": true, "CAST": true, "CHANNEL": true, "PROJECTILE": true,
	"AREA": true, "DASH_ATTACK": true, "MOVEMENT": true, "SUMMON": true, "NONE": true,
}

var targetingModes = map[string]bool{
	"SELF": true, "DIRECTION": true, "SINGLE_TARGET": true,
	"AREA_POSITION": true, "AREA_SELF": true, "PROJECTILE": true, "NONE": true,
}

// canonicalTags is the skill-tag enum from skills.md. PENETRATE is a
// damage-component tag, not a roster tag, so it is not accepted here.
var canonicalTags = map[string]bool{
	"BASIC_ATTACK": true, "DAMAGING": true, "AREA": true, "PROJECTILE": true,
	"MOVEMENT": true, "HEAL": true, "SHIELD": true, "STATUS_APPLY": true,
	"DISPLACEMENT": true, "DEFENSIVE": true, "SIGNATURE": true,
}

var speedStats = map[string]bool{"ATTACK_SPEED": true, "CAST_SPEED": true, "NONE": true}

// compileSkills reads the 5 roster tables (tags/execution/targeting) and the
// 5 Action Specifications tables (geometry/timing) and produces the 45
// compiled geometry rows with ADR-0047 band + tag/effect parity checks.
func compileSkills(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	// Pass 1: roster metadata per skill_id.
	type meta struct {
		exec, targ, tags, air string
		line                  int
	}
	metas := map[string]meta{}
	for _, t := range f.TablesWithHeader("skill_id", "execution_type", "targeting_mode", "tags") {
		for _, r := range t.Rows {
			id := r.Cells["skill_id"]
			if id == "" {
				continue
			}
			m := meta{
				exec: r.Cells["execution_type"], targ: r.Cells["targeting_mode"],
				tags: r.Cells["tags"], air: r.Cells["air"], line: r.Line,
			}
			metas[id] = m
			CheckEnum(ds, f.RelPath, r.Line, 6, "execution_type", m.exec, executionTypes)
			CheckEnum(ds, f.RelPath, r.Line, 6, "targeting_mode", m.targ, targetingModes)
			for _, tg := range SplitList(m.tags) {
				if !canonicalTags[tg] {
					ds.Add(f.RelPath, r.Line, 6, CodeValueOutOfBounds, "tag "+tg+" not canonical")
				}
			}
		}
	}
	// Pass 2: Action Specifications tables compile geometry rows.
	var secCap int
	for _, t := range f.TablesWithHeader("skill_id", "speed_stat", "startup/active/recovery", "geometry") {
		for _, r := range t.Rows {
			id := r.Cells["skill_id"]
			if !strings.Contains(id, ".basic.") && !strings.Contains(id, ".active.") {
				continue
			}
			basic := strings.Contains(id, ".basic.")
			m := metas[id]
			sg := SkillGeom{SkillID: id, Execution: m.exec, Targeting: m.targ, Tags: SplitList(m.tags), SpeedStat: r.Cells["speed_stat"]}
			CheckEnum(ds, f.RelPath, r.Line, 2, "speed_stat", sg.SpeedStat, speedStats)
			// startup/active/recovery = "520/120/480" ms.
			p := strings.Split(strings.TrimSpace(r.Cells["startup/active/recovery"]), "/")
			if len(p) != 3 {
				ds.Add(f.RelPath, r.Line, 3, CodeTableSyntaxError, "timing not S/A/R")
			} else {
				for i, v := range p {
					n, e := ParseInt(v)
					if e != nil || n < 0 {
						ds.Add(f.RelPath, r.Line, 3, CodeValueOutOfBounds, "timing "+v+" not non-negative int")
						continue
					}
					switch i {
					case 0:
						sg.StartupMS = n
					case 1:
						sg.ActiveMS = n
					case 2:
						sg.RecoveryMS = n
					}
				}
			}
			cd, err := ParseMagnitude(strings.TrimSuffix(r.Cells["base_cd"], "s"))
			if err != nil {
				ds.Add(f.RelPath, r.Line, 5, CodeTableSyntaxError, "base_cd not a magnitude")
			}
			sg.BaseCD = cd
			mp, err := ParseInt(strings.TrimSuffix(r.Cells["cost"], " MP"))
			if err != nil || mp < 0 {
				ds.Add(f.RelPath, r.Line, 6, CodeTableSyntaxError, "cost not non-negative MP int")
			}
			sg.CostMP = mp
			g, air := parseGeomCell(f.RelPath, r.Line, r.Cells["geometry"], ds)
			sg.Geometry = g
			if air != nil {
				sg.AirGeometry = air
			}
			heal := false
			for _, tg := range sg.Tags {
				if tg == "HEAL" {
					heal = true
				}
			}
			checkSkillBands(f.RelPath, r.Line, g, basic, heal, sg.SkillID, ds)
			if air != nil {
				checkSkillBands(f.RelPath, r.Line, *air, basic, heal, sg.SkillID, ds)
			}
			// Tag/geometry consistency: BASIC_ATTACK only on .basic.
			hasTag := func(w string) bool {
				for _, tg := range sg.Tags {
					if tg == w {
						return true
					}
				}
				return false
			}
			if hasTag("BASIC_ATTACK") && !basic {
				ds.Add(f.RelPath, r.Line, 6, CodeValueOutOfBounds, "BASIC_ATTACK on non-basic "+id)
			}
			if m.air == "none" && sg.AirGeometry != nil && !basic {
				ds.Add(f.RelPath, r.Line, 6, CodeValueOutOfBounds, "air geometry on non-basic "+id)
			}
			snap.Skills = append(snap.Skills, sg)
			secCap++
		}
	}
	if secCap != 45 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("skill geometry rows must be 45 (20 basic + 25 active), found %d", secCap))
	}
	compileSecondaryEffects(f, snap, ds)
}

var secRadiusRe = regexp.MustCompile("(?:circle|radius|by|of|maximum|outward|into|to)\\s*`?([0-9]+\\.?[0-9]*)m")

// compileSecondaryEffects parses the 10-row spatial secondary-effect table
// and enforces ADR-0047 tag/effect parity.
func compileSecondaryEffects(f *DocFile, snap *CandidateSnapshot, ds *Diagnostics) {
	tables := f.TablesWithHeader("spatial_effect_id", "source", "exact resolution")
	displaceBySource := map[string]bool{}
	for _, t := range tables {
		for _, r := range t.Rows {
			se := SecondaryEffect{
				EffectID:   r.Cells["spatial_effect_id"],
				Origin:     r.Cells["origin / shape"],
				Resolution: r.Cells["exact resolution"],
				TargetCap:  r.Cells["target-cap interaction"],
			}
			// Cells are already backtick-stripped by the table parser; a source
			// cell may list several skill ids separated by commas.
			se.Sources = append(se.Sources, ExtractIDs(r.Cells["source"])...)
			// Radius used by the secondary splash/aura band check.
			rad := 0.0
			for _, c := range []string{se.Origin, se.Resolution} {
				if m := secRadiusRe.FindStringSubmatch(c); m != nil {
					if v, e := ParseMagnitude(m[1]); e == nil && v > rad {
						rad = v
					}
				}
			}
			se.RadiusM = rad
			lr := strings.ToLower(se.Resolution + " " + se.Origin)
			se.Displaces = strings.Contains(lr, "push") || strings.Contains(lr, "pull") ||
				strings.Contains(lr, "knockback") || strings.Contains(lr, "displace") ||
				strings.Contains(lr, "airborne")
			for _, s := range se.Sources {
				if se.Displaces {
					displaceBySource[s] = true
				}
			}
			// Secondary splash/aura radius band 1.2..4.0 (ADR-0047 §6).
			if se.RadiusM > 0 && (se.RadiusM < 1.2 || se.RadiusM > 4.0) {
				ds.Add(f.RelPath, r.Line, 3, CodeValueOutOfBounds,
					fmt.Sprintf("secondary radius %.2f outside 1.2..4.0 band", se.RadiusM))
			}
			snap.Secondary = append(snap.Secondary, se)
			snap.IDIndex[se.EffectID] = f.RelPath
		}
	}
	if len(snap.Secondary) != 10 {
		ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
			fmt.Sprintf("secondary spatial effects must be exactly 10, found %d", len(snap.Secondary)))
	}
	// DISPLACEMENT parity: every secondary push/pull source must carry the tag,
	// and every tagged skill must have a displacement-capable effect.
	displaceSources := map[string]bool{}
	for _, se := range snap.Secondary {
		for _, s := range se.Sources {
			if se.Displaces {
				displaceSources[s] = true
			}
		}
	}
	// Ensure resolution text of push/pull also counts when the id name says so.
	for _, se := range snap.Secondary {
		if strings.Contains(se.EffectID, ".pull") || strings.Contains(se.EffectID, ".knockback") {
			for _, s := range se.Sources {
				displaceSources[s] = true
			}
		}
	}
	for _, sg := range snap.Skills {
		tagged := false
		for _, tg := range sg.Tags {
			if tg == "DISPLACEMENT" {
				tagged = true
			}
		}
		eff := displaceSources[sg.SkillID]
		if tagged && !eff {
			ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
				"DISPLACEMENT tag on "+sg.SkillID+" has no push/pull secondary effect")
		}
		if !tagged && eff {
			ds.Add(f.RelPath, 0, 0, CodeValueOutOfBounds,
				sg.SkillID+" produces displacement without DISPLACEMENT tag")
		}
	}
}
