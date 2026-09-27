namespace ThinhThan.Core.Assets
{
    // The expected import settings for one sprite, computed from the
    // canonical rules of presentation_asset_manifest.md §3/§3.1.
    public struct SpriteImportExpectation
    {
        public int Ppu;
        public bool PivotBottomCenter;
        public bool Mipmaps;
        public bool MeshTight;
        public int MobileAstcBlock;
        public int TextureWidth;
        public int TextureHeight;
    }
}
