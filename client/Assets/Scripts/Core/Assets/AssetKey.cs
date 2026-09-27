namespace ThinhThan.Core.Assets
{
    // Asset key grammar of client_assets.md § Stable Asset Keys:
    //   catalog-backed   asset.<catalog_id>.<facet>
    //   non-catalog      asset.<kind>.<name>.<facet>
    // Keys are ASCII lowercase dot-separated; the final segment is a
    // facet; catalog IDs keep their kind prefix verbatim; no variant
    // segment exists — a variant is its own catalog ID or facet.
    public static class AssetKey
    {
        public const string Prefix = "asset.";

        public static string FromCatalogId(string catalogId, AssetFacet facet)
        {
            return Prefix + catalogId + "." + facet.ToKeySegment();
        }

        public static string FromKind(AssetKeyKind kind, string name, AssetFacet facet)
        {
            return Prefix + kind.ToKeySegment() + "." + name + "." + facet.ToKeySegment();
        }

        // Grammar-only check: shape, charset and a valid final facet
        // segment. Returns true when the key is syntactically valid.
        public static bool IsValid(string key)
        {
            if (string.IsNullOrEmpty(key) || !key.StartsWith(Prefix, System.StringComparison.Ordinal))
            {
                return false;
            }
            var body = key.Substring(Prefix.Length);
            var segments = body.Split('.');
            if (segments.Length < 2)
            {
                return false;
            }
            foreach (var segment in segments)
            {
                if (!IsValidSegment(segment))
                {
                    return false;
                }
            }
            return AssetFacetExtensions.TryParseKeySegment(segments[segments.Length - 1], out _);
        }

        // Splits a valid key into its catalogue identity and facet.
        // For non-catalog keys <kind>.<name> replaces the catalog ID and
        // isCatalogBacked is false.
        public static bool TrySplit(string key, out string identity, out AssetFacet facet, out bool isCatalogBacked)
        {
            identity = "";
            facet = default;
            isCatalogBacked = false;
            if (!IsValid(key))
            {
                return false;
            }
            var body = key.Substring(Prefix.Length);
            var lastDot = body.LastIndexOf('.');
            AssetFacetExtensions.TryParseKeySegment(body.Substring(lastDot + 1), out facet);
            identity = body.Substring(0, lastDot);
            var firstDot = identity.IndexOf('.');
            if (firstDot <= 0 || firstDot == identity.Length - 1)
            {
                return false;
            }
            var head = identity.Substring(0, firstDot);
            isCatalogBacked = CatalogAssetKindExtensions.TryParseIdPrefix(head, out _);
            if (!isCatalogBacked && !AssetKeyKindExtensions.TryParseKeySegment(head, out _))
            {
                return false;
            }
            return true;
        }

        public static bool TryFacet(string key, out AssetFacet facet)
        {
            facet = default;
            if (!IsValid(key))
            {
                return false;
            }
            var body = key.Substring(Prefix.Length);
            return AssetFacetExtensions.TryParseKeySegment(body.Substring(body.LastIndexOf('.') + 1), out facet);
        }

        private static bool IsValidSegment(string segment)
        {
            if (segment.Length == 0)
            {
                return false;
            }
            foreach (var c in segment)
            {
                var lower = c >= 'a' && c <= 'z';
                var digit = c >= '0' && c <= '9';
                if (!lower && !digit && c != '_')
                {
                    return false;
                }
            }
            return true;
        }
    }
}
