namespace ThinhThan.Core.Assets
{
    // Texture block formats used by the deterministic RAM model
    // (presentation_asset_manifest.md §1). ASTC 4x4 and BC7 are 8 bpp
    // (16 bytes per 4x4 block); ASTC 6x6 is 16 bytes per 6x6 block.
    public enum TextureBlockFormat
    {
        Astc4x4,
        Astc6x6,
        Bc7,
        Rgba32,
    }
}
