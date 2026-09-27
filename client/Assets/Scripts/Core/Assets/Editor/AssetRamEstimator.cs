using UnityEditor;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor
{
    // Deterministic per-entry RAM estimate used by the budget gates
    // (presentation_asset_manifest.md §1): texture entries model the
    // declared profile at mobile ASTC block size (the binding platform
    // for the RAM budgets), BGM entries charge the streaming buffer
    // allowance, and every other placeholder carries the fixed
    // structural cost of AssetRamModel.StructuralBytesPerEntry.
    public static class AssetRamEstimator
    {
        public static long EntryBytes(string projectRelativePath)
        {
            var placeholder = AssetDatabase.LoadAssetAtPath<AddressablePlaceholder>(projectRelativePath);
            if (placeholder != null)
            {
                return PlaceholderBytes(placeholder);
            }
            var texture = AssetDatabase.LoadAssetAtPath<Texture2D>(projectRelativePath);
            if (texture != null)
            {
                return AssetRamModel.TextureBytes(texture.width, texture.height, TextureBlockFormat.Astc4x4, false);
            }
            return AssetRamModel.StructuralBytesPerEntry;
        }

        private static long PlaceholderBytes(AddressablePlaceholder placeholder)
        {
            if (placeholder.Facet == AssetFacet.Bgm.ToKeySegment())
            {
                return AssetRamModel.BgmStreamingBufferBytes;
            }
            if (placeholder.Facet == AssetFacet.Clip.ToKeySegment())
            {
                // SFX cues are short; a 4s stereo cue at 44.1 kHz is a
                // fixed decompressed-PCM allowance.
                return AssetRamModel.AudioClipBytes(4.0, 44100, 2);
            }
            if (SpriteSizeProfileExtensions.TryParseProfileName(placeholder.SizeProfile, out var profile))
            {
                var cls = ParseClass(placeholder.AssetClass);
                var block = SpriteImportRules.MobileAstcBlock(cls) == 4 ? TextureBlockFormat.Astc4x4 : TextureBlockFormat.Astc6x6;
                return AssetRamModel.TextureBytes(profile.CellTextureWidth(), profile.CellTextureHeight(), block, false)
                    + AssetRamModel.StructuralBytesPerEntry;
            }
            return AssetRamModel.StructuralBytesPerEntry;
        }

        private static PresentationAssetClass ParseClass(string name)
        {
            switch (name)
            {
                case "ACTOR": return PresentationAssetClass.Actor;
                case "COSMETIC_APPEARANCE": return PresentationAssetClass.CosmeticAppearance;
                case "PROP": return PresentationAssetClass.Prop;
                case "ITEM_ICON": return PresentationAssetClass.ItemIcon;
                case "EQUIPMENT_ICON": return PresentationAssetClass.EquipmentIcon;
                case "UI_ART": return PresentationAssetClass.UiArt;
                case "FONT_ATLAS": return PresentationAssetClass.FontAtlas;
                case "TILE": return PresentationAssetClass.Tile;
                case "PARALLAX_NEAR": return PresentationAssetClass.ParallaxNear;
                case "PARALLAX_FAR": return PresentationAssetClass.ParallaxFar;
                case "VFX_SOFT": return PresentationAssetClass.VfxSoft;
                default: return PresentationAssetClass.UiArt;
            }
        }
    }
}
