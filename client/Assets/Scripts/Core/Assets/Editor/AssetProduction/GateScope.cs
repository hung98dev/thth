using ThinhThan.Core.Assets;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // The section 3.1a scope table: which cutout and volume rules apply to
    // each declared asset_class. Declaring the wrong class to dodge a gate
    // is a review violation, so the mapping lives in one place and both
    // gates consume it instead of duplicating the table.
    public static class GateScope
    {
        public static CutoutRule CutoutRules(PresentationAssetClass assetClass)
        {
            switch (assetClass)
            {
                case PresentationAssetClass.Actor:
                case PresentationAssetClass.CosmeticAppearance:
                case PresentationAssetClass.Prop:
                case PresentationAssetClass.ItemIcon:
                case PresentationAssetClass.EquipmentIcon:
                case PresentationAssetClass.ParallaxNear:
                    return CutoutRule.All;
                case PresentationAssetClass.UiArt:
                    return CutoutRule.Format | CutoutRule.TranslucentBand
                        | CutoutRule.Fringe | CutoutRule.Dilation;
                case PresentationAssetClass.Tile:
                    return CutoutRule.Format | CutoutRule.Fringe | CutoutRule.Dilation;
                case PresentationAssetClass.ParallaxFar:
                    return CutoutRule.Format | CutoutRule.Fringe;
                case PresentationAssetClass.VfxSoft:
                    // VFX gameplay textures must still be exactly 2x the
                    // declared cell_ref; every other pixel rule is exempted
                    // because additive soft edges cannot satisfy them.
                    return CutoutRule.Format | CutoutRule.Size;
                case PresentationAssetClass.FontAtlas:
                default:
                    return CutoutRule.None;
            }
        }

        public static VolumeRule VolumeRules(PresentationAssetClass assetClass)
        {
            switch (assetClass)
            {
                case PresentationAssetClass.Actor:
                case PresentationAssetClass.CosmeticAppearance:
                    return VolumeRule.All;
                case PresentationAssetClass.Prop:
                case PresentationAssetClass.ItemIcon:
                case PresentationAssetClass.EquipmentIcon:
                    return VolumeRule.All & ~VolumeRule.ActorOnBackground;
                case PresentationAssetClass.Tile:
                    return VolumeRule.ValueRange | VolumeRule.TopLit;
                case PresentationAssetClass.ParallaxNear:
                case PresentationAssetClass.ParallaxFar:
                    return VolumeRule.Environment;
                case PresentationAssetClass.UiArt:
                case PresentationAssetClass.FontAtlas:
                case PresentationAssetClass.VfxSoft:
                default:
                    return VolumeRule.None;
            }
        }
    }
}
