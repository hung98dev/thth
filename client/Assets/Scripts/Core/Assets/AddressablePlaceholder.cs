using UnityEngine;

namespace ThinhThan.Core.Assets
{
    // Placeholder payload standing in for a release asset at a fixed
    // addressable key (presentation_asset_manifest.md §4: internal
    // milestones may ship placeholders carrying the correct key from
    // the start; art packets replace the file, not the key). The
    // declared class/profile drives the deterministic budget and
    // import-validation checks until the real asset lands.
    public class AddressablePlaceholder : ScriptableObject
    {
        [SerializeField] private string _catalogId = "";
        [SerializeField] private string _facet = "";
        [SerializeField] private string _assetClass = "";
        [SerializeField] private string _sizeProfile = "";

        public string CatalogId
        {
            get { return _catalogId; }
        }

        public string Facet
        {
            get { return _facet; }
        }

        public string AssetClass
        {
            get { return _assetClass; }
        }

        public string SizeProfile
        {
            get { return _sizeProfile; }
        }

        public void Initialize(string catalogId, string facet, string assetClass, string sizeProfile)
        {
            _catalogId = catalogId;
            _facet = facet;
            _assetClass = assetClass;
            _sizeProfile = sizeProfile;
        }
    }
}
