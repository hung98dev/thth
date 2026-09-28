using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Folklore card of presentation_asset_manifest.md section 5 (ART-010):
    // cultural-origin entities (monsters, bosses, spirit beasts, NPCs, maps,
    // cosmetics) carry one inside their register row; the reviewer checks it
    // against the forbidden-motif list before APPROVED.
    [Serializable]
    public sealed class FolkloreCard
    {
        public List<string>? source_tales;
        public string? regional_variants;
        public List<string>? motifs_checked;
    }
}
