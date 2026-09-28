using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Pixel input for the Volume & Depth Gate (section 3.6). Pixels are
    // RGBA32, row-major, y = 0 at the bottom edge. The silhouette S is the
    // a >= 128 pixel set minus the declared translucent mask.
    public sealed class VolumeDepthInput
    {
        public Color32[] Pixels = null!;
        public int Width;
        public int Height;
        public PresentationAssetClass AssetClass;
        public bool[]? TranslucentMask;
    }
}
