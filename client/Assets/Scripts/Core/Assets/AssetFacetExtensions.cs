namespace ThinhThan.Core.Assets
{
    // Facet <-> key segment mapping. Segments are the lowercase spellings
    // pinned by client_assets.md § Stable Asset Keys.
    public static class AssetFacetExtensions
    {
        private static readonly string[] _segments =
        {
            "prefab",
            "sprite",
            "anim",
            "scene",
            "vfx",
            "icon",
            "portrait",
            "bgm",
            "clip",
            "font",
        };

        public static string ToKeySegment(this AssetFacet facet)
        {
            return _segments[(int)facet];
        }

        public static bool TryParseKeySegment(string segment, out AssetFacet facet)
        {
            for (var i = 0; i < _segments.Length; i++)
            {
                if (_segments[i] == segment)
                {
                    facet = (AssetFacet)i;
                    return true;
                }
            }
            facet = default;
            return false;
        }
    }
}
