namespace ThinhThan.Core.Assets
{
    // Catalog kind <-> ID prefix mapping. A catalog ID is
    // <prefix>.<rest>; the prefix segment is what client_assets.md calls
    // the "kind segment" embedded verbatim in the key.
    public static class CatalogAssetKindExtensions
    {
        private static readonly string[] _prefixes =
        {
            "class",
            "monster",
            "boss",
            "npc",
            "beast",
            "skill",
            "item",
            "cosmetic",
            "map",
            "dungeon",
            "instance",
            "effect",
            "zone",
        };

        public static string ToIdPrefix(this CatalogAssetKind kind)
        {
            return _prefixes[(int)kind];
        }

        public static bool TryParseIdPrefix(string prefix, out CatalogAssetKind kind)
        {
            for (var i = 0; i < _prefixes.Length; i++)
            {
                if (_prefixes[i] == prefix)
                {
                    kind = (CatalogAssetKind)i;
                    return true;
                }
            }
            kind = default;
            return false;
        }
    }
}
