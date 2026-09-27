namespace ThinhThan.Core.Assets
{
    // Non-catalog key kind segment set of client_assets.md § Stable Asset Keys.
    // Kind segment occupies the position right after the "asset" prefix in
    // asset.<kind>.<name>.<facet> keys.
    public enum AssetKeyKind
    {
        Ui,
        Sfx,
        Bgm,
        Font,
        Prop,
        Tile,
        Parallax,
        Vfx,
    }
}
