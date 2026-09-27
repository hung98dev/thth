namespace ThinhThan.Core.Assets
{
    // Deterministic key -> canonical group assignment. Catalog routing
    // follows client_assets.md § Grouping: zoned content lives in
    // region.<zone>, instanced content in dungeon.<dungeon>/finale, icons
    // in icons.shared, audio in the audio.bgm.* set, boot/session content
    // in the .local groups, and anything cross-region in a *.shared
    // group. Boss keys resolve through their catalog space_id (supplied
    // by the caller, which reads boss_catalog.md).
    public static class KeyGroupRule
    {
        // Returns the owning group name, or null when the key cannot be
        // assigned (non-catalog prop/tile/parallax/vfx names carry no
        // zone and must be assigned explicitly by their owning packet).
        public static string? Assign(string key, string? bossSpaceId)
        {
            if (!TrySplitUnchecked(key, out var identity, out var facet, out var catalogBacked))
            {
                return null;
            }
            var head = HeadSegment(identity);
            if (catalogBacked)
            {
                return AssignCatalog(identity, head, facet, bossSpaceId);
            }
            return AssignNonCatalog(identity, head, facet);
        }

        private static string? AssignCatalog(string identity, string head, AssetFacet facet, string? bossSpaceId)
        {
            CatalogAssetKind kind;
            if (!CatalogAssetKindExtensions.TryParseIdPrefix(head, out kind))
            {
                return null;
            }
            switch (kind)
            {
                case CatalogAssetKind.Map:
                {
                    // space ids: map.<zone>.<map>, map.pvp.*,
                    // map.guild_war.* (physics_geometry_contract.md §6.1).
                    var spaceZone = SpaceZone(identity);
                    return facet == AssetFacet.Bgm ? BgmGroupForZone(spaceZone) : MapGroupFor(spaceZone);
                }
                case CatalogAssetKind.Dungeon:
                {
                    var dungeonKey = identity.Substring(head.Length + 1);
                    if (facet == AssetFacet.Bgm)
                    {
                        var zone = AddressableGroups.DungeonZone(identity);
                        return BgmGroupForZone(zone);
                    }
                    return AddressableGroups.DungeonGroup(dungeonKey);
                }
                case CatalogAssetKind.Instance:
                {
                    if (facet == AssetFacet.Bgm)
                    {
                        var zone = AddressableGroups.DungeonZone(identity);
                        return BgmGroupForZone(zone);
                    }
                    return AddressableGroups.DungeonFinale;
                }
                case CatalogAssetKind.Monster:
                case CatalogAssetKind.Npc:
                {
                    // monster.<zone>.<name> / npc.<zone>.<name>.
                    var zone = SpaceZone(identity);
                    if (facet == AssetFacet.Bgm)
                    {
                        return BgmGroupForZone(zone);
                    }
                    return zone == null ? null : AddressableGroups.RegionGroup(zone);
                }
                case CatalogAssetKind.Boss:
                {
                    // boss.<name> resolves through its space_id:
                    // PUBLIC bosses join the host map's region group;
                    // INSTANCED bosses join the dungeon/finale group.
                    if (bossSpaceId == null)
                    {
                        return null;
                    }
                    return AssignCatalog(bossSpaceId, HeadSegment(bossSpaceId), facet, null);
                }
                case CatalogAssetKind.Class:
                    return facet == AssetFacet.Icon ? AddressableGroups.IconsShared : AddressableGroups.SharedLocal;
                case CatalogAssetKind.Skill:
                    return facet == AssetFacet.Icon ? AddressableGroups.IconsShared : AddressableGroups.SharedLocal;
                case CatalogAssetKind.Item:
                    return AddressableGroups.IconsShared;
                case CatalogAssetKind.Effect:
                    return AddressableGroups.IconsShared;
                case CatalogAssetKind.Beast:
                    return AddressableGroups.BeastShared;
                case CatalogAssetKind.Cosmetic:
                    if (facet == AssetFacet.Icon)
                    {
                        return AddressableGroups.IconsShared;
                    }
                    return AddressableGroups.CosmeticShared;
                default:
                    return null;
            }
        }

        private static string? AssignNonCatalog(string identity, string head, AssetFacet facet)
        {
            AssetKeyKind kind;
            if (!AssetKeyKindExtensions.TryParseKeySegment(head, out kind))
            {
                return null;
            }
            switch (kind)
            {
                case AssetKeyKind.Bgm:
                {
                    // asset.bgm.zone.<zone>.bgm -> audio.bgm.<zone>;
                    // any other bgm name -> audio.bgm.shared.
                    var rest = identity.Substring(head.Length + 1);
                    const string zonePrefix = "zone.";
                    if (rest.StartsWith(zonePrefix, System.StringComparison.Ordinal))
                    {
                        var zone = rest.Substring(zonePrefix.Length);
                        return AddressableGroups.AudioBgmGroup(zone);
                    }
                    return AddressableGroups.AudioBgmShared;
                }
                case AssetKeyKind.Sfx:
                    // core SFX cues live in shared.local.
                    return AddressableGroups.SharedLocal;
                case AssetKeyKind.Ui:
                case AssetKeyKind.Font:
                    // boot/login/error UI and fonts live in the player
                    // build bootstrap group.
                    return AddressableGroups.BootstrapLocal;
                default:
                    return null;
            }
        }

        private static string? MapGroupFor(string? zoneSegment)
        {
            if (zoneSegment == null)
            {
                return null;
            }
            if (zoneSegment == "pvp" || zoneSegment == "guild_war")
            {
                return AddressableGroups.PvpShared;
            }
            return AddressableGroups.RegionGroup(zoneSegment);
        }

        private static string BgmGroupForZone(string? zoneKey)
        {
            return zoneKey == null ? AddressableGroups.AudioBgmShared : AddressableGroups.AudioBgmGroup(zoneKey);
        }

        private static string? SpaceZone(string catalogId)
        {
            var firstDot = catalogId.IndexOf('.');
            if (firstDot <= 0)
            {
                return null;
            }
            var rest = catalogId.Substring(firstDot + 1);
            var secondDot = rest.IndexOf('.');
            return secondDot <= 0 ? rest : rest.Substring(0, secondDot);
        }

        private static string HeadSegment(string identity)
        {
            var dot = identity.IndexOf('.');
            return dot <= 0 ? identity : identity.Substring(0, dot);
        }

        private static bool TrySplitUnchecked(string key, out string identity, out AssetFacet facet, out bool catalogBacked)
        {
            if (!AssetKey.TrySplit(key, out identity, out facet, out catalogBacked))
            {
                return false;
            }
            return true;
        }
    }
}
