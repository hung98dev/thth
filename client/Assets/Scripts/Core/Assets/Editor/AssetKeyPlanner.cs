using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor
{
    // Builds the deterministic key plan: every catalog requirement -> one
    // key per facet -> canonical group -> placeholder backing file, plus
    // the non-catalog keys (SFX cues, zone/shared BGM) and the one-hop
    // PresentationAlias BGM indirection for every playable scene
    // (client_assets.md § Grouping; presentation_asset_manifest.md §6).
    public static class AssetKeyPlanner
    {
        public const string PlaceholdersRoot = "Assets/AddressableAssetsData/Placeholders/";

        // Core SFX cues of presentation_asset_manifest.md §6 (14 cues).
        private static readonly string[] _sfxCues =
        {
            "ui_confirm",
            "ui_cancel",
            "ui_error",
            "jump",
            "land",
            "basic_attack",
            "hit",
            "guard",
            "just_guard_success",
            "skill_cast",
            "boss_telegraph",
            "item_pickup",
            "quest_complete",
            "map_transfer",
        };

        public static List<PlannedAssetEntry> Build(string docsRoot)
        {
            var entries = new List<PlannedAssetEntry>();
            var seen = new HashSet<string>();
            foreach (var req in ContentCatalogScanner.Scan(docsRoot))
            {
                foreach (var facet in req.Facets)
                {
                    var key = AssetKey.FromCatalogId(req.CatalogId, facet);
                    AddCatalogEntry(entries, seen, key, req, facet);
                }
                // Every playable scene space also owns a one-hop BGM
                // alias so `asset.<space_id>.bgm` resolves.
                if (req.Facets.Contains(AssetFacet.Scene))
                {
                    AddBgmAlias(entries, seen, req.CatalogId);
                }
            }
            AddNonCatalog(entries, seen);
            entries.Sort((a, b) => string.CompareOrdinal(a.Key, b.Key));
            return entries;
        }

        private static void AddCatalogEntry(List<PlannedAssetEntry> entries, HashSet<string> seen, string key, CatalogAssetRequirement req, AssetFacet facet)
        {
            var group = KeyGroupRule.Assign(key, req.Kind == CatalogAssetKind.Boss ? req.SpaceId : null);
            var entry = new PlannedAssetEntry();
            entry.Key = key;
            entry.CatalogId = req.CatalogId;
            entry.Group = group ?? "";
            entry.Facet = facet;
            entry.AssetClass = req.AssetClass;
            entry.SizeProfile = req.SizeProfile;
            entry.PlaceholderPath = PlaceholderPath(entry.Group, key, facet == AssetFacet.Icon ? "png" : "asset");
            if (seen.Add(key))
            {
                entries.Add(entry);
            }
        }

        private static void AddBgmAlias(List<PlannedAssetEntry> entries, HashSet<string> seen, string spaceId)
        {
            var key = spaceId + ".bgm";
            var aliasKey = AssetKey.Prefix + spaceId + "." + AssetFacet.Bgm.ToKeySegment();
            var target = BgmTargetFor(spaceId);
            var group = KeyGroupRule.Assign(aliasKey, null);
            var entry = new PlannedAssetEntry();
            entry.Key = aliasKey;
            entry.CatalogId = spaceId;
            entry.Group = group ?? "";
            entry.Facet = AssetFacet.Bgm;
            entry.AliasTarget = target;
            entry.PlaceholderPath = PlaceholdersRoot + entry.Group + "/alias." + key + ".asset";
            if (seen.Add(aliasKey))
            {
                entries.Add(entry);
            }
        }

        // Scene -> shared BGM key: map.<zone>.* -> zone BGM; pvp and
        // guild_war spaces -> shared BGM; dungeon.<d> -> the act zone's
        // BGM; the finale instance -> nui_thieng.
        private static string BgmTargetFor(string spaceId)
        {
            var zone = SpaceZoneOf(spaceId);
            if (zone != null)
            {
                return AssetKey.FromKind(AssetKeyKind.Bgm, "zone." + zone, AssetFacet.Bgm);
            }
            if (spaceId.StartsWith("map.pvp.", StringComparison.Ordinal))
            {
                return AssetKey.FromKind(AssetKeyKind.Bgm, "shared.pvp", AssetFacet.Bgm);
            }
            if (spaceId.StartsWith("map.guild_war.", StringComparison.Ordinal))
            {
                return AssetKey.FromKind(AssetKeyKind.Bgm, "shared.guild_war", AssetFacet.Bgm);
            }
            var dungeonZone = AddressableGroups.DungeonZone(spaceId);
            if (dungeonZone != null)
            {
                return AssetKey.FromKind(AssetKeyKind.Bgm, "zone." + dungeonZone, AssetFacet.Bgm);
            }
            return AssetKey.FromKind(AssetKeyKind.Bgm, "shared.pvp", AssetFacet.Bgm);
        }

        private static string? SpaceZoneOf(string spaceId)
        {
            var parts = spaceId.Split('.');
            if (parts.Length < 3 || parts[0] != "map")
            {
                return null;
            }
            var zone = parts[1];
            return AddressableGroups.IsZoneKey(zone) ? zone : null;
        }

        private static void AddNonCatalog(List<PlannedAssetEntry> entries, HashSet<string> seen)
        {
            foreach (var cue in _sfxCues)
            {
                var key = AssetKey.FromKind(AssetKeyKind.Sfx, cue, AssetFacet.Clip);
                var entry = new PlannedAssetEntry();
                entry.Key = key;
                entry.Group = KeyGroupRule.Assign(key, null) ?? "";
                entry.Facet = AssetFacet.Clip;
                entry.PlaceholderPath = PlaceholderPath(entry.Group, key, "asset");
                if (seen.Add(key))
                {
                    entries.Add(entry);
                }
            }
            foreach (var zone in AddressableGroups.ZoneKeys)
            {
                var key = AssetKey.FromKind(AssetKeyKind.Bgm, "zone." + zone, AssetFacet.Bgm);
                var entry = new PlannedAssetEntry();
                entry.Key = key;
                entry.Group = KeyGroupRule.Assign(key, null) ?? "";
                entry.Facet = AssetFacet.Bgm;
                entry.AssetClass = "AUDIO";
                entry.PlaceholderPath = PlaceholderPath(entry.Group, key, "asset");
                if (seen.Add(key))
                {
                    entries.Add(entry);
                }
            }
            foreach (var shared in new[] { "pvp", "guild_war" })
            {
                var key = AssetKey.FromKind(AssetKeyKind.Bgm, "shared." + shared, AssetFacet.Bgm);
                var entry = new PlannedAssetEntry();
                entry.Key = key;
                entry.Group = KeyGroupRule.Assign(key, null) ?? "";
                entry.Facet = AssetFacet.Bgm;
                entry.AssetClass = "AUDIO";
                entry.PlaceholderPath = PlaceholderPath(entry.Group, key, "asset");
                if (seen.Add(key))
                {
                    entries.Add(entry);
                }
            }
        }

        private static string PlaceholderPath(string group, string key, string extension)
        {
            return PlaceholdersRoot + group + "/" + key + "." + extension;
        }
    }
}
