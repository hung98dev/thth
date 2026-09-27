using System.Collections.Generic;

namespace ThinhThan.Core.Assets
{
    // Canonical group set, delivery mode and budgets.
    // Names: client_assets.md § Grouping (ADR-0071). Budgets:
    // presentation_asset_manifest.md §1 (compressed MB / RAM MB).
    public static class AddressableGroups
    {
        public const string BootstrapLocal = "bootstrap.local";
        public const string SharedLocal = "shared.local";
        public const string IconsShared = "icons.shared";
        public const string BeastShared = "beast.shared";
        public const string CosmeticShared = "cosmetic.shared";
        public const string PvpShared = "pvp.shared";
        public const string DungeonFinale = "dungeon.finale";
        public const string AudioBgmShared = "audio.bgm.shared";

        public const string RegionPrefix = "region.";
        public const string DungeonPrefix = "dungeon.";
        public const string AudioBgmPrefix = "audio.bgm.";

        // Zone keys: encounter_catalog.md progression route (zone.*).
        public static readonly string[] ZoneKeys =
        {
            "lang_da",
            "rung_u_minh",
            "ben_nuoc_den",
            "deo_may",
            "thanh_co",
            "nui_thieng",
        };

        // Dungeon keys: dungeon_catalog.md roster; the finale instance
        // instance.finale.than_trung uses dungeon.finale.
        public static readonly string[] DungeonKeys =
        {
            "dinh_lang_bo_hoang",
            "mieu_ba_trong_rung",
            "xom_chim",
            "hang_ma_tranh",
            "den_tran",
        };

        // Dungeon key -> owning zone key (encounter_catalog.md act table;
        // the finale instance belongs to zone.nui_thieng).
        private static readonly string[] _dungeonZones =
        {
            "lang_da",
            "rung_u_minh",
            "ben_nuoc_den",
            "deo_may",
            "thanh_co",
            "nui_thieng",
        };

        public static string RegionGroup(string zoneKey)
        {
            return RegionPrefix + zoneKey;
        }

        public static string DungeonGroup(string dungeonKey)
        {
            return dungeonKey == "finale" ? DungeonFinale : DungeonPrefix + dungeonKey;
        }

        public static string AudioBgmGroup(string zoneKey)
        {
            return AudioBgmPrefix + zoneKey;
        }

        public static bool IsZoneKey(string key)
        {
            for (var i = 0; i < ZoneKeys.Length; i++)
            {
                if (ZoneKeys[i] == key)
                {
                    return true;
                }
            }
            return false;
        }

        public static bool IsDungeonKey(string key)
        {
            for (var i = 0; i < DungeonKeys.Length; i++)
            {
                if (DungeonKeys[i] == key)
                {
                    return true;
                }
            }
            return false;
        }

        // Zone owning a dungeon or the finale instance; null when the
        // space id is not a known dungeon space.
        public static string? DungeonZone(string spaceId)
        {
            if (spaceId == "instance.finale.than_trung")
            {
                return "nui_thieng";
            }
            const string prefix = "dungeon.";
            if (!spaceId.StartsWith(prefix, System.StringComparison.Ordinal))
            {
                return null;
            }
            var key = spaceId.Substring(prefix.Length);
            for (var i = 0; i < DungeonKeys.Length; i++)
            {
                if (DungeonKeys[i] == key)
                {
                    return _dungeonZones[i];
                }
            }
            return null;
        }

        // Full canonical group set (25), in deterministic order.
        public static string[] CanonicalNames()
        {
            var names = new List<string>
            {
                BootstrapLocal,
                SharedLocal,
                IconsShared,
                BeastShared,
                CosmeticShared,
                PvpShared,
                DungeonFinale,
                AudioBgmShared,
            };
            foreach (var zone in ZoneKeys)
            {
                names.Add(RegionGroup(zone));
            }
            foreach (var dungeon in DungeonKeys)
            {
                names.Add(DungeonGroup(dungeon));
            }
            foreach (var zone in ZoneKeys)
            {
                names.Add(AudioBgmGroup(zone));
            }
            return names.ToArray();
        }

        public static bool IsCanonical(string group)
        {
            foreach (var name in CanonicalNames())
            {
                if (name == group)
                {
                    return true;
                }
            }
            return false;
        }

        // Groups that ship inside the player build (base install).
        public static bool IsLocal(string group)
        {
            return group == BootstrapLocal || group == SharedLocal;
        }

        // Only cosmetic.shared packs each asset separately (§ Grouping).
        public static bool IsPackSeparately(string group)
        {
            return group == CosmeticShared;
        }

        // §1 budgets: compressed download MB / runtime RAM MB.
        // audio.bgm RAM is the streaming buffer allowance per group.
        public static int CompressedBudgetMb(string group)
        {
            if (group == BootstrapLocal)
            {
                return 12;
            }
            if (group == SharedLocal)
            {
                return 70;
            }
            if (group == IconsShared || group == BeastShared)
            {
                return 20;
            }
            if (group == CosmeticShared)
            {
                return 60;
            }
            if (group == PvpShared || group == DungeonFinale)
            {
                return 25;
            }
            if (group.StartsWith(RegionPrefix, System.StringComparison.Ordinal))
            {
                return 60;
            }
            if (group.StartsWith(DungeonPrefix, System.StringComparison.Ordinal))
            {
                return 25;
            }
            if (group.StartsWith(AudioBgmPrefix, System.StringComparison.Ordinal))
            {
                return 15;
            }
            return -1;
        }

        public static int RamBudgetMb(string group)
        {
            if (group == BootstrapLocal)
            {
                return 24;
            }
            if (group == SharedLocal)
            {
                return 110;
            }
            if (group == IconsShared)
            {
                return 32;
            }
            if (group == BeastShared)
            {
                return 36;
            }
            if (group == CosmeticShared)
            {
                return 48;
            }
            if (group == PvpShared || group == DungeonFinale)
            {
                return 60;
            }
            if (group.StartsWith(RegionPrefix, System.StringComparison.Ordinal))
            {
                return 120;
            }
            if (group.StartsWith(DungeonPrefix, System.StringComparison.Ordinal))
            {
                return 60;
            }
            if (group.StartsWith(AudioBgmPrefix, System.StringComparison.Ordinal))
            {
                return 4;
            }
            return -1;
        }

        // Residency classes of §1: always-loaded groups (bootstrap,
        // shared, icons, beast) and cosmetic.shared (loaded while any
        // cosmetic shows), versus transient region/dungeon/pvp/audio
        // groups loaded on entry and released on exit.
        public static bool IsResident(string group)
        {
            return group == BootstrapLocal
                || group == SharedLocal
                || group == IconsShared
                || group == BeastShared
                || group == CosmeticShared;
        }
    }
}
