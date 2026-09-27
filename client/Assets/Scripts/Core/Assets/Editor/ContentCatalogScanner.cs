using System;
using System.Collections.Generic;
using System.IO;
using System.Text.RegularExpressions;

namespace ThinhThan.Core.Assets.Editor
{
    // Scans docs/ content catalogs and produces the deterministic asset
    // requirement list the addressables plan is built from (the
    // "manifest extraction" half of presentation_asset_manifest.md §6
    // provenance). Only backticked lowercase dotted IDs are collected;
    // each catalog file is filtered to its own ID prefix.
    public static class ContentCatalogScanner
    {
        private static readonly Regex IdPattern = new Regex("`([a-z_][a-z0-9_]*(?:\\.[a-z0-9_]+)+)`");

        public static string DefaultDocsRoot
        {
            get
            {
                // Tests run with the client project root as CWD.
                return Path.GetFullPath(Path.Combine(Directory.GetCurrentDirectory(), "..", "docs"));
            }
        }

        public static List<CatalogAssetRequirement> Scan(string docsRoot)
        {
            var reqs = new SortedDictionary<string, CatalogAssetRequirement>(StringComparer.Ordinal);
            ScanClassSkill(Path.Combine(docsRoot, "07_content", "class_skill_catalog.md"), reqs);
            ScanMonster(Path.Combine(docsRoot, "07_content", "monster_catalog.md"), reqs);
            ScanBoss(Path.Combine(docsRoot, "07_content", "boss_catalog.md"), reqs);
            ScanNpc(Path.Combine(docsRoot, "07_content", "npc_shop_catalog.md"), reqs);
            ScanBeast(Path.Combine(docsRoot, "07_content", "spirit_beast_catalog.md"), reqs);
            ScanEquipment(Path.Combine(docsRoot, "07_content", "equipment_catalog.md"), reqs);
            ScanItems(Path.Combine(docsRoot, "07_content", "item_catalog.md"), reqs);
            ScanCosmetics(Path.Combine(docsRoot, "07_content", "cosmetic_catalog.md"), reqs);
            ScanMaps(Path.Combine(docsRoot, "07_content", "world_route_catalog.md"), reqs);
            ScanDungeons(Path.Combine(docsRoot, "07_content", "dungeon_catalog.md"), reqs);
            ScanSpaceIds(docsRoot, reqs);
            ScanEffects(docsRoot, reqs);
            var list = new List<CatalogAssetRequirement>(reqs.Values.Count);
            list.AddRange(reqs.Values);
            return list;
        }

        private static string[] TableCells(string line)
        {
            var trimmed = line.Trim();
            if (!trimmed.StartsWith("|", StringComparison.Ordinal))
            {
                return Array.Empty<string>();
            }
            var parts = trimmed.Split('|');
            var cells = new List<string>(parts.Length);
            foreach (var part in parts)
            {
                var cell = part.Trim().Trim('`').Trim();
                if (cell.Length > 0)
                {
                    cells.Add(cell);
                }
            }
            return cells.ToArray();
        }

        private static readonly Regex ClassHeadingPattern = new Regex("^## (KIM|MOC|THUY|HOA|THO)\\s");

        private static void ScanClassSkill(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            if (!File.Exists(file))
            {
                return;
            }
            foreach (var line in File.ReadAllLines(file))
            {
                var cells = TableCells(line);
                if (cells.Length > 0 && cells[0].StartsWith("skill.", StringComparison.Ordinal))
                {
                    var req = NewOrExisting(reqs, cells[0]);
                    req.Kind = CatalogAssetKind.Skill;
                    req.AssetClass = "VFX_SOFT";
                    if (!req.Facets.Contains(AssetFacet.Vfx))
                    {
                        req.Facets.Add(AssetFacet.Vfx);
                        req.Facets.Add(AssetFacet.Icon);
                    }
                    continue;
                }
                // Classes are declared as "## KIM — Kiếm Khách" headings;
                // the catalog ID is class.<element> lowercase.
                var m = ClassHeadingPattern.Match(line);
                if (m.Success)
                {
                    var id = "class." + m.Groups[1].Value.ToLowerInvariant();
                    var req = NewOrExisting(reqs, id);
                    req.Kind = CatalogAssetKind.Class;
                    req.AssetClass = "ACTOR";
                    req.SizeProfile = "CHARACTER";
                    if (!req.Facets.Contains(AssetFacet.Prefab))
                    {
                        req.Facets.Add(AssetFacet.Prefab);
                    }
                }
            }
        }

        private static void ScanMonster(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            // monster_catalog.md: | monster_id | rank | ... |. SMALL_ROSTER
            // monsters rank NORMAL still import at MONSTER_SMALL; the
            // roster is parsed from the catalog's fenced declaration
            // block, never duplicated here.
            var small = ParseSmallRoster(file);
            foreach (var line in File.Exists(file) ? File.ReadAllLines(file) : Array.Empty<string>())
            {
                var cells = TableCells(line);
                if (cells.Length < 2 || !cells[0].StartsWith("monster.", StringComparison.Ordinal))
                {
                    continue;
                }
                var id = cells[0];
                var rank = cells[1];
                var profile = "MONSTER_MEDIUM";
                if (rank == "ELITE")
                {
                    profile = "MONSTER_ELITE";
                }
                else if (rank == "NORMAL" && small.Contains(id))
                {
                    profile = "MONSTER_SMALL";
                }
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Monster;
                req.AssetClass = "ACTOR";
                req.SizeProfile = profile;
                req.Facets.Add(AssetFacet.Prefab);
            }
        }

        private static HashSet<string> ParseSmallRoster(string file)
        {
            // The catalog declares SMALL_ROSTER inside a fenced block
            // following the "`SMALL_ROSTER` is exactly:" line.
            var roster = new HashSet<string>();
            if (!File.Exists(file))
            {
                return roster;
            }
            var inBlock = false;
            var afterLabel = false;
            foreach (var line in File.ReadAllLines(file))
            {
                if (line.Contains("SMALL_ROSTER` is exactly"))
                {
                    afterLabel = true;
                    continue;
                }
                if (afterLabel && line.TrimStart().StartsWith("```", StringComparison.Ordinal))
                {
                    inBlock = !inBlock;
                    if (!inBlock)
                    {
                        afterLabel = false;
                    }
                    continue;
                }
                if (inBlock)
                {
                    foreach (Match m in IdPattern.Matches(line))
                    {
                        var id = m.Groups[1].Value;
                        if (id.StartsWith("monster.", StringComparison.Ordinal))
                        {
                            roster.Add(id);
                        }
                    }
                }
            }
            return roster;
        }

        private static void ScanBoss(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            // boss_catalog.md: | boss_id | Lv | mode | space_id | size_profile | ... |
            foreach (var line in File.Exists(file) ? File.ReadAllLines(file) : Array.Empty<string>())
            {
                var cells = TableCells(line);
                if (cells.Length < 5 || !cells[0].StartsWith("boss.", StringComparison.Ordinal))
                {
                    continue;
                }
                var req = NewOrExisting(reqs, cells[0]);
                req.Kind = CatalogAssetKind.Boss;
                req.AssetClass = "ACTOR";
                req.SpaceId = cells[3];
                req.SizeProfile = cells[4] == "WORLD_BOSS" ? "WORLD_BOSS" : "BOSS_LARGE";
                req.Facets.Add(AssetFacet.Prefab);
            }
        }

        private static void ScanNpc(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "npc"))
            {
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Npc;
                req.AssetClass = "ACTOR";
                req.SizeProfile = "CHARACTER";
                req.Facets.Add(AssetFacet.Prefab);
            }
        }

        private static readonly HashSet<string> _elements = new HashSet<string>
        {
            "kim",
            "moc",
            "thuy",
            "hoa",
            "tho",
        };

        private static void ScanBeast(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "beast"))
            {
                // beast.<element>.<name> is a beast; beast.skill.* rows
                // belong to the beast's own ability set and get no key.
                var parts = id.Split('.');
                if (parts.Length != 3 || !_elements.Contains(parts[1]))
                {
                    continue;
                }
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Beast;
                req.AssetClass = "ACTOR";
                req.SizeProfile = "SPIRIT_BEAST";
                req.Facets.Add(AssetFacet.Prefab);
            }
        }

        // 14 equipment slots in canonical order (equipment_catalog.md § slot list).
        private static readonly string[] _equipmentSlots =
        {
            "weapon",
            "head",
            "body",
            "hands",
            "legs",
            "feet",
            "necklace",
            "ring",
            "costume",
            "talisman",
            "jade",
            "seal",
            "relic",
            "charm",
        };

        private static void ScanEquipment(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            // Sets are declared as `set.t<tier>.<key>`; each set expands to
            // item.eq.<tier>.<key>.<slot> for all 14 slots (icons only).
            foreach (var setId in IdsInFile(file, "set"))
            {
                var parts = setId.Split('.');
                if (parts.Length != 3 || parts[1].Length < 2 || parts[1][0] != 't')
                {
                    continue;
                }
                var tier = parts[1].Substring(1);
                var setKey = parts[2];
                foreach (var slot in _equipmentSlots)
                {
                    var itemId = "item.eq." + tier + "." + setKey + "." + slot;
                    if (reqs.ContainsKey(itemId))
                    {
                        continue;
                    }
                    var req = new CatalogAssetRequirement();
                    req.CatalogId = itemId;
                    req.Kind = CatalogAssetKind.Item;
                    req.AssetClass = "EQUIPMENT_ICON";
                    req.Facets.Add(AssetFacet.Icon);
                    reqs.Add(itemId, req);
                }
            }
        }

        private static void ScanItems(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "item"))
            {
                // item.eq.* rows are expansions of equipment sets.
                if (id.StartsWith("item.eq.", StringComparison.Ordinal))
                {
                    continue;
                }
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Item;
                req.AssetClass = "ITEM_ICON";
                req.Facets.Add(AssetFacet.Icon);
            }
        }

        private static void ScanCosmetics(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "cosmetic"))
            {
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Cosmetic;
                var isAppearance = id.StartsWith("cosmetic.appearance.", StringComparison.Ordinal)
                    || id.StartsWith("cosmetic.iap.appearance.", StringComparison.Ordinal);
                if (isAppearance)
                {
                    req.AssetClass = "COSMETIC_APPEARANCE";
                    req.Facets.Add(AssetFacet.Prefab);
                    req.Facets.Add(AssetFacet.Icon);
                }
                else
                {
                    req.AssetClass = "UI_ART";
                    req.Facets.Add(AssetFacet.Icon);
                }
            }
        }

        private static void ScanMaps(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "map"))
            {
                var zone = SpaceSegment(id, 1);
                if (!AddressableGroups.IsZoneKey(zone) && zone != "pvp" && zone != "guild_war")
                {
                    continue;
                }
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Map;
                req.Facets.Add(AssetFacet.Scene);
            }
        }

        private static void ScanDungeons(string file, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            foreach (var id in IdsInFile(file, "dungeon"))
            {
                var req = NewOrExisting(reqs, id);
                req.Kind = CatalogAssetKind.Dungeon;
                req.Facets.Add(AssetFacet.Scene);
            }
        }

        // The encounter_catalog act table: zone -> dungeon rows. Used to
        // cross-check AddressableGroups' baked dungeon->zone mapping.
        public static List<(string Zone, string Dungeon)> ScanActTable(string docsRoot)
        {
            var pairs = new List<(string, string)>();
            var file = Path.Combine(docsRoot, "07_content", "encounter_catalog.md");
            if (!File.Exists(file))
            {
                return pairs;
            }
            foreach (var line in File.ReadAllLines(file))
            {
                var cells = TableCells(line);
                string? zone = null;
                string? dungeon = null;
                foreach (var cell in cells)
                {
                    if (cell.StartsWith("zone.", StringComparison.Ordinal))
                    {
                        zone = cell;
                    }
                    else if (cell.StartsWith("dungeon.", StringComparison.Ordinal))
                    {
                        dungeon = cell;
                    }
                }
                if (zone != null && dungeon != null)
                {
                    pairs.Add((zone, dungeon));
                }
            }
            return pairs;
        }

        private static void ScanSpaceIds(string docsRoot, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            // Space ids outside the map catalogs: pvp.md, guild_war.md
            // (03_systems) and physics_geometry_contract.md §6.1.
            var files = new List<string>
            {
                Path.Combine(docsRoot, "03_systems", "pvp.md"),
                Path.Combine(docsRoot, "03_systems", "guild_war.md"),
                Path.Combine(docsRoot, "04_architecture", "physics_geometry_contract.md"),
                Path.Combine(docsRoot, "07_content", "world_route_catalog.md"),
            };
            var raw = new Regex("(?<![a-z0-9_.])(map\\.(?:pvp|guild_war)\\.[a-z0-9_]+|instance\\.[a-z0-9_.]+)");
            foreach (var file in files)
            {
                if (!File.Exists(file))
                {
                    continue;
                }
                // Space ids appear both backticked (tables) and bare
                // (space_id = map.guild_war.five_seal_conflict).
                foreach (Match m in raw.Matches(File.ReadAllText(file)))
                {
                    var id = m.Groups[1].Value.TrimEnd('.');
                    if (id.StartsWith("instance.", StringComparison.Ordinal))
                    {
                        AddSpaceScene(reqs, id, CatalogAssetKind.Instance);
                    }
                    else
                    {
                        AddSpaceScene(reqs, id, CatalogAssetKind.Map);
                    }
                }
            }
        }

        private static void ScanEffects(string docsRoot, SortedDictionary<string, CatalogAssetRequirement> reqs)
        {
            var dir = Path.Combine(docsRoot, "07_content");
            if (!Directory.Exists(dir))
            {
                return;
            }
            foreach (var file in Directory.GetFiles(dir, "*.md"))
            {
                foreach (var id in IdsInFile(file, "effect.basic"))
                {
                    AddEffectIcon(reqs, id);
                }
                foreach (var id in IdsInFile(file, "effect.skill"))
                {
                    AddEffectIcon(reqs, id);
                }
            }
        }

        private static void AddSpaceScene(SortedDictionary<string, CatalogAssetRequirement> reqs, string id, CatalogAssetKind kind)
        {
            var req = NewOrExisting(reqs, id);
            req.Kind = kind;
            req.Facets.Add(AssetFacet.Scene);
        }

        private static void AddEffectIcon(SortedDictionary<string, CatalogAssetRequirement> reqs, string id)
        {
            var req = NewOrExisting(reqs, id);
            req.Kind = CatalogAssetKind.Effect;
            req.AssetClass = "UI_ART";
            req.Facets.Add(AssetFacet.Icon);
        }

        private static List<string> IdsInFile(string file, string idPrefix)
        {
            var ids = new List<string>();
            if (!File.Exists(file))
            {
                return ids;
            }
            var prefix = idPrefix + ".";
            foreach (Match m in IdPattern.Matches(File.ReadAllText(file)))
            {
                var id = m.Groups[1].Value;
                if (id.StartsWith(prefix, StringComparison.Ordinal))
                {
                    ids.Add(id);
                }
            }
            return ids;
        }

        private static string SpaceSegment(string id, int index)
        {
            var parts = id.Split('.');
            return parts.Length > index ? parts[index] : "";
        }

        private static CatalogAssetRequirement NewOrExisting(SortedDictionary<string, CatalogAssetRequirement> reqs, string id)
        {
            if (!reqs.TryGetValue(id, out var req))
            {
                req = new CatalogAssetRequirement();
                req.CatalogId = id;
                reqs.Add(id, req);
            }
            return req;
        }
    }
}
