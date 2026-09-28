using ThinhThan.Core.Assets;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Resolves the expected texture and silhouette sizes (final 2x texture
    // px) for a declared asset_class per presentation_asset_manifest.md
    // section 3/3.1a. Actor classes resolve through size_profile; PROP,
    // VFX_SOFT and parallax layers through the declared cell_ref; icons and
    // tiles have fixed cells; UI_ART/FONT_ATLAS carry no size rule.
    public static class PresentationSizing
    {
        // 2x silhouette limits indexed by SpriteSizeProfile
        // (silhouette max ref px x 2).
        private static readonly int[] _silhouetteWidth2x = { 128, 100, 150, 250, 400, 500, 96 };
        private static readonly int[] _silhouetteHeight2x = { 192, 100, 200, 300, 440, 560, 96 };

        public static int SilhouetteMaxWidth2x(SpriteSizeProfile profile)
        {
            return _silhouetteWidth2x[(int)profile];
        }

        public static int SilhouetteMaxHeight2x(SpriteSizeProfile profile)
        {
            return _silhouetteHeight2x[(int)profile];
        }

        // Builds a cutout input for one file given its declared metadata.
        // CellWidth/Height are 0 when the class carries no texture-size rule.
        public static CutoutGateInput ToCutoutInput(
            Color32[] pixels,
            int width,
            int height,
            PresentationAssetClass assetClass,
            ImportMetadata meta,
            bool[]? translucentMask)
        {
            var input = new CutoutGateInput
            {
                Pixels = pixels,
                Width = width,
                Height = height,
                AssetClass = assetClass,
                TranslucentMask = translucentMask,
                DeclaredDetachedParts = meta.DeclaredDetachedParts,
                DeclaredPixelArt = meta.DeclaredPixelArt,
            };
            var profileName = meta.SizeProfile;
            if (profileName != null
                && SpriteSizeProfileExtensions.TryParseProfileName(profileName, out var profile))
            {
                input.CellWidth = profile.CellTextureWidth();
                input.CellHeight = profile.CellTextureHeight();
                input.SilhouetteMaxWidth = SilhouetteMaxWidth2x(profile);
                input.SilhouetteMaxHeight = SilhouetteMaxHeight2x(profile);
                input.CheckBodyHeight = profile == SpriteSizeProfile.Character;
            }
            else if (meta.CellRefWidth > 0 && meta.CellRefHeight > 0)
            {
                input.CellWidth = 2 * meta.CellRefWidth;
                input.CellHeight = 2 * meta.CellRefHeight;
                input.SilhouetteMaxWidth = input.CellWidth;
                input.SilhouetteMaxHeight = input.CellHeight;
            }
            else if (assetClass == PresentationAssetClass.ItemIcon
                || assetClass == PresentationAssetClass.EquipmentIcon)
            {
                input.CellWidth = SpriteImportRules.IconTextureSizePx();
                input.CellHeight = SpriteImportRules.IconTextureSizePx();
                input.SilhouetteMaxWidth = input.CellWidth;
                input.SilhouetteMaxHeight = input.CellHeight;
            }
            else if (assetClass == PresentationAssetClass.Tile)
            {
                input.CellWidth = SpriteImportRules.TileTextureSizePx();
                input.CellHeight = SpriteImportRules.TileTextureSizePx();
                input.SilhouetteMaxWidth = input.CellWidth;
                input.SilhouetteMaxHeight = input.CellHeight;
            }
            return input;
        }

        public static VolumeDepthInput ToVolumeInput(
            Color32[] pixels,
            int width,
            int height,
            PresentationAssetClass assetClass,
            bool[]? translucentMask)
        {
            return new VolumeDepthInput
            {
                Pixels = pixels,
                Width = width,
                Height = height,
                AssetClass = assetClass,
                TranslucentMask = translucentMask,
            };
        }
    }
}
