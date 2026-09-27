namespace ThinhThan.Core.Assets
{
    // Canonical sprite import rules (presentation_asset_manifest.md §3,
    // §3.1; ADR-0055, ADR-0059, ADR-0071):
    //   texture = 2 x cell reference pixels,
    //   100 PPU for gameplay sprites, 200 PPU for UI/icon classes,
    //   PARALLAX_FAR may author at 1x and imports at 50 PPU,
    //   Bilinear filtering, Bottom Center pivot on gameplay sprites,
    //   no physics shape, no mipmaps on gameplay sprites,
    //   Mesh Type Tight iff the long side is >= 256 texture px AND the
    //   texture has transparent margins, Full Rect otherwise,
    //   mobile compression ASTC 4x4 for actor/UI/icon/font classes and
    //   ASTC 6x6 for backgrounds/parallax, desktop BC7 throughout.
    public static class SpriteImportRules
    {
        public const int GameplayPpu = 100;
        public const int UiPpu = 200;
        public const int ParallaxFarPpu = 50;
        public const int MeshTightLongSidePx = 256;

        public static bool IsUiClass(PresentationAssetClass cls)
        {
            return cls == PresentationAssetClass.UiArt
                || cls == PresentationAssetClass.ItemIcon
                || cls == PresentationAssetClass.EquipmentIcon
                || cls == PresentationAssetClass.FontAtlas;
        }

        public static bool IsGameplayClass(PresentationAssetClass cls)
        {
            return !IsUiClass(cls)
                && cls != PresentationAssetClass.ParallaxFar
                && cls != PresentationAssetClass.ParallaxNear;
        }

        public static int PpuFor(PresentationAssetClass cls)
        {
            if (cls == PresentationAssetClass.ParallaxFar)
            {
                return ParallaxFarPpu;
            }
            return IsUiClass(cls) ? UiPpu : GameplayPpu;
        }

        // Mesh type rule: Tight needs a long side of >= 256 texture px
        // AND transparent margins; everything else is Full Rect.
        public static bool MeshTightExpected(int textureWidth, int textureHeight, bool hasTransparentMargins)
        {
            var longSide = textureWidth >= textureHeight ? textureWidth : textureHeight;
            return longSide >= MeshTightLongSidePx && hasTransparentMargins;
        }

        // Mobile compression block size: 4 for actor/UI/icon/font, 6 for
        // environment layers (backgrounds and parallax).
        public static int MobileAstcBlock(PresentationAssetClass cls)
        {
            return cls == PresentationAssetClass.ParallaxFar
                || cls == PresentationAssetClass.ParallaxNear
                ? 6
                : 4;
        }

        // Bottom Center pivot applies to gameplay sprites; UI/icon
        // sprites keep the centre pivot of UI-space imports.
        public static bool PivotBottomCenter(PresentationAssetClass cls)
        {
            return IsGameplayClass(cls) || cls == PresentationAssetClass.ParallaxNear;
        }

        public static SpriteImportExpectation Expect(PresentationAssetClass cls, int textureWidth, int textureHeight, bool hasTransparentMargins)
        {
            var e = new SpriteImportExpectation();
            e.Ppu = PpuFor(cls);
            e.PivotBottomCenter = PivotBottomCenter(cls);
            e.Mipmaps = false;
            e.MeshTight = MeshTightExpected(textureWidth, textureHeight, hasTransparentMargins);
            e.MobileAstcBlock = MobileAstcBlock(cls);
            e.TextureWidth = textureWidth;
            e.TextureHeight = textureHeight;
            return e;
        }

        // The 2x authoring rule: a sprite cell texture is exactly 2 x
        // the reference cell. Icon cells are 64 x 64 reference px
        // (128 x 128 texture px), tiles 50 x 50 ref (100 x 100).
        public static int IconTextureSizePx()
        {
            return 128;
        }

        public static int TileTextureSizePx()
        {
            return 100;
        }
    }
}
