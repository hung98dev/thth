using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Pixel input for the Cutout Quality Gate (presentation_asset_manifest.md
    // section 3.2). Pixels are RGBA32, row-major, y = 0 at the bottom edge
    // (Unity texture convention). Sizes are final 2x texture pixels.
    public sealed class CutoutGateInput
    {
        public Color32[] Pixels = null!;
        public int Width;
        public int Height;
        public PresentationAssetClass AssetClass;

        // Expected texture dimensions (2x cell); 0 disables the texture-size
        // check. Silhouette caps are the 2x limits of the size table; 0 caps
        // at the cell dimensions.
        public int CellWidth;
        public int CellHeight;
        public int SilhouetteMaxWidth;
        public int SilhouetteMaxHeight;

        // CHARACTER profiles only: the body height (176..192 texture px)
        // measured over the central band of the silhouette.
        public bool CheckBodyHeight;

        // Declared translucent mask (<texture>.translucent.png), white =
        // translucent; null when the asset declares none. Only ACTOR
        // ghost/smoke/water assets may declare a mask.
        public bool[]? TranslucentMask;

        // Declared import flags: detached_parts (legit separate pieces) and
        // pixel_art (binary cutout allowed — no launch assets qualify).
        public bool DeclaredDetachedParts;
        public bool DeclaredPixelArt;
    }
}
