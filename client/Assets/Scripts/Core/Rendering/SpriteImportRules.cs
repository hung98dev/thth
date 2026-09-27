namespace ThinhThan.Core.Rendering
{
    // Sprite import conventions (presentation_asset_manifest.md §3.1):
    // gameplay sprites import at 100 PPU, UI sprites at 200 PPU against a
    // 100 PPU canvas reference, PARALLAX_FAR 1x textures at 50 PPU.
    public static class SpriteImportRules
    {
        public const int GameplayPixelsPerUnit = 100;
        public const int UiPixelsPerUnit = 200;
        public const int ParallaxFarPixelsPerUnit = 50;
        public const int CanvasReferencePixelsPerUnit = 100;
    }
}
