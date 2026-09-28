using ThinhThan.Core.Assets;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Parser for the per-file import metadata declared in the importer's
    // userData (section 3.1a: "Moi file khai bao asset_class trong metadata
    // import"). Format is a semicolon-separated list of key=value pairs and
    // bare flags:
    //   asset_class=PROP                 (required)
    //   size_profile=CHARACTER           (actor classes)
    //   cell_ref=64x64                   (PROP / VFX_SOFT / PARALLAX_*)
    //   translucent=ghost|smoke|water    (declares <file>.translucent.png)
    //   detached_parts                   (legit separate components)
    //   pixel_art                        (binary cutout allowed)
    //   soft_edges                       (VFX_SOFT declaration)
    public sealed class ImportMetadata
    {
        public PresentationAssetClass? AssetClass;
        public string? SizeProfile;
        public int CellRefWidth;
        public int CellRefHeight;
        public string? TranslucentSubject;
        public bool DeclaredDetachedParts;
        public bool DeclaredPixelArt;
        public bool DeclaredSoftEdges;

        public static ImportMetadata Parse(string? userData)
        {
            var meta = new ImportMetadata();
            if (string.IsNullOrEmpty(userData))
            {
                return meta;
            }
            foreach (var raw in userData!.Split(';'))
            {
                var token = raw.Trim();
                if (token.Length == 0)
                {
                    continue;
                }
                var eq = token.IndexOf('=');
                var key = eq >= 0 ? token.Substring(0, eq).Trim() : token;
                var value = eq >= 0 ? token.Substring(eq + 1).Trim() : "";
                switch (key)
                {
                    case "asset_class":
                        meta.AssetClass = ParseClass(value);
                        break;
                    case "size_profile":
                        meta.SizeProfile = value;
                        break;
                    case "cell_ref":
                        var x = value.IndexOf('x');
                        if (x > 0
                            && int.TryParse(value.Substring(0, x), out var cw)
                            && int.TryParse(value.Substring(x + 1), out var ch))
                        {
                            meta.CellRefWidth = cw;
                            meta.CellRefHeight = ch;
                        }
                        break;
                    case "translucent":
                        meta.TranslucentSubject = value;
                        break;
                    case "detached_parts":
                        meta.DeclaredDetachedParts = true;
                        break;
                    case "pixel_art":
                        meta.DeclaredPixelArt = true;
                        break;
                    case "soft_edges":
                        meta.DeclaredSoftEdges = true;
                        break;
                }
            }
            return meta;
        }

        private static PresentationAssetClass? ParseClass(string name)
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
                default: return null;
            }
        }

        public static string ClassName(PresentationAssetClass cls)
        {
            switch (cls)
            {
                case PresentationAssetClass.Actor: return "ACTOR";
                case PresentationAssetClass.CosmeticAppearance: return "COSMETIC_APPEARANCE";
                case PresentationAssetClass.Prop: return "PROP";
                case PresentationAssetClass.ItemIcon: return "ITEM_ICON";
                case PresentationAssetClass.EquipmentIcon: return "EQUIPMENT_ICON";
                case PresentationAssetClass.UiArt: return "UI_ART";
                case PresentationAssetClass.FontAtlas: return "FONT_ATLAS";
                case PresentationAssetClass.Tile: return "TILE";
                case PresentationAssetClass.ParallaxNear: return "PARALLAX_NEAR";
                case PresentationAssetClass.ParallaxFar: return "PARALLAX_FAR";
                case PresentationAssetClass.VfxSoft: return "VFX_SOFT";
                default: return cls.ToString();
            }
        }
    }
}
