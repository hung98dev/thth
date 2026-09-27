namespace ThinhThan.Core.Assets
{
    // Declared import/audit classes of presentation_asset_manifest.md
    // §3.1a. Every sprite or placeholder declares exactly one class; the
    // validator applies the matching cutout/volume scope and import
    // profile rules.
    public enum PresentationAssetClass
    {
        Actor,
        CosmeticAppearance,
        Prop,
        ItemIcon,
        EquipmentIcon,
        UiArt,
        FontAtlas,
        Tile,
        ParallaxNear,
        ParallaxFar,
        VfxSoft,
    }
}
