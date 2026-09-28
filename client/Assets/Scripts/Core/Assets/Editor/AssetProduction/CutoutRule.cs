using System;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Bit flags naming each Cutout Quality Gate rule of
    // presentation_asset_manifest.md section 3.2. GateScope maps every
    // asset_class to its rule subset per the scope table of section 3.1a.
    // Corners (ART-001) is the 4-corner alpha = 0 probe, scoped to the
    // cell-based classes only.
    [Flags]
    public enum CutoutRule
    {
        None = 0,
        Format = 1,
        TranslucentBand = 2,
        Fringe = 4,
        Dilation = 8,
        Specks = 16,
        Jaggies = 32,
        InteriorHoles = 64,
        CellPadding = 128,
        Size = 256,
        Corners = 512,
        All = Format | TranslucentBand | Fringe | Dilation | Specks
            | Jaggies | InteriorHoles | CellPadding | Size | Corners,
    }
}
