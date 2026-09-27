namespace ThinhThan.Core.Assets.Editor
{
    // One planned addressable entry: the key, the owning group, the
    // backing placeholder file (repo-relative under
    // Assets/AddressableAssetsData/Placeholders/), the declared
    // facet/class/profile, and the alias target when the entry is a
    // PresentationAlias indirection.
    public sealed class PlannedAssetEntry
    {
        public string Key = "";
        public string CatalogId = "";
        public string Group = "";
        public AssetFacet Facet;
        public string AssetClass = "";
        public string SizeProfile = "";
        // Project-relative placeholder path (forward slashes), e.g.
        // Assets/AddressableAssetsData/Placeholders/icons.shared/x.png.
        public string PlaceholderPath = "";
        // Empty = real placeholder asset; non-empty = PresentationAlias
        // whose target is this shared key.
        public string AliasTarget = "";
    }
}
