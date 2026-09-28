using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Root of the versioned source register (asset_source_register.json,
    // presentation_asset_manifest.md section 6). Deserialized with
    // JsonUtility; field names are the snake_case JSON keys verbatim. One
    // row per packaged media file; assets is sorted by file_path ascending.
    [Serializable]
    public sealed class AssetSourceRegister
    {
        public int schema_version;
        public List<AssetSourceRow> assets = new List<AssetSourceRow>();
    }
}
