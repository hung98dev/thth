using System;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Bit flags naming each Volume & Depth Gate rule of
    // presentation_asset_manifest.md section 3.6.
    [Flags]
    public enum VolumeRule
    {
        None = 0,
        ValueRange = 1,
        ValueTiers = 2,
        TopLit = 4,
        EdgeSeparation = 8,
        FlatRegions = 16,
        Environment = 32,
        ActorOnBackground = 64,
        All = ValueRange | ValueTiers | TopLit | EdgeSeparation
            | FlatRegions | Environment | ActorOnBackground,
    }
}
