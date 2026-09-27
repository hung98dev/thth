namespace ThinhThan.Core.Assets
{
    // Non-catalog kind <-> key segment mapping (client_assets.md § Stable
    // Asset Keys). Kind segments and catalog-ID prefixes are disjoint by
    // construction; AssetKey.TryParse relies on that.
    public static class AssetKeyKindExtensions
    {
        private static readonly string[] _segments =
        {
            "ui",
            "sfx",
            "bgm",
            "font",
            "prop",
            "tile",
            "parallax",
            "vfx",
        };

        public static string ToKeySegment(this AssetKeyKind kind)
        {
            return _segments[(int)kind];
        }

        public static bool TryParseKeySegment(string segment, out AssetKeyKind kind)
        {
            for (var i = 0; i < _segments.Length; i++)
            {
                if (_segments[i] == segment)
                {
                    kind = (AssetKeyKind)i;
                    return true;
                }
            }
            kind = default;
            return false;
        }
    }
}
