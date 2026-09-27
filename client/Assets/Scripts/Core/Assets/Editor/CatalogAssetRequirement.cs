using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor
{
    // One asset requirement derived from a catalog row: the catalog ID,
    // the facets it must ship, the declared size profile / asset class
    // used by the deterministic budget model, and the space_id used to
    // route space-bound keys (bosses) to their owning group.
    public sealed class CatalogAssetRequirement
    {
        public string CatalogId = "";
        public CatalogAssetKind Kind;
        public string AssetClass = "";
        public string SizeProfile = "";
        public string SpaceId = "";
        public readonly List<AssetFacet> Facets = new List<AssetFacet>();
        public readonly List<string> ExtraKeys = new List<string>();
    }
}
