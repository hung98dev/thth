using System.IO;
using UnityEditor;
using UnityEngine;

namespace ThinhThan.Art.Editor
{
    // IMP-074 cosmetic art import rules: stamps the canonical import
    // profile (presentation_asset_manifest.md section 3/3.1a) onto every
    // PNG under Assets/Art/Cosmetics/ and Assets/Art/StyleRef/cosmetics/.
    // The asset_class declaration lives in the committed .meta userData
    // (Unity seeds the importer with it before preprocessing and
    // round-trips it back out), so CI materialization produces identical
    // .meta files byte-for-byte. UI_ART imports at 200 PPU with a
    // centred pivot; COSMETIC_APPEARANCE cells import at 100 PPU with
    // the actor pivot (Bottom Center). Style Pack anchors import with
    // the same rules but never ship inside Addressables.
    public sealed class CosmeticArtImporter : AssetPostprocessor
    {
        private const string CosmeticsRoot = "Assets/Art/Cosmetics/";
        private const string StyleRefRoot = "Assets/Art/StyleRef/cosmetics/";

        private void OnPreprocessTexture()
        {
            var owned = assetPath.StartsWith(CosmeticsRoot, System.StringComparison.Ordinal)
                || assetPath.StartsWith(StyleRefRoot, System.StringComparison.Ordinal);
            if (!owned
                || !assetPath.EndsWith(".png", System.StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
            var importer = (TextureImporter)assetImporter;
            var userData = (importer.userData ?? string.Empty).Trim();
            if (userData.Length == 0)
            {
                return;
            }
            var assetClass = Declared(userData, "asset_class");
            var isAppearance = assetClass == "COSMETIC_APPEARANCE";

            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.spritePixelsPerUnit = isAppearance ? 100f : 200f;
            importer.mipmapEnabled = false;
            importer.sRGBTexture = true;
            importer.alphaIsTransparency = true;
            importer.isReadable = true;
            importer.filterMode = FilterMode.Bilinear;
            importer.npotScale = TextureImporterNPOTScale.None;
            importer.maxTextureSize = 2048;

            var settings = new TextureImporterSettings();
            importer.ReadTextureSettings(settings);
            settings.spriteAlignment = (int)SpriteAlignment.Custom;
            settings.spritePivot = isAppearance
                ? new Vector2(0.5f, 0f)
                : new Vector2(0.5f, 0.5f);
            settings.spriteGenerateFallbackPhysicsShape = false;
            var w = 0;
            var h = 0;
            GetSourceSize(assetPath, ref w, ref h);
            settings.spriteMeshType = isAppearance || Mathf.Max(w, h) >= 256
                ? SpriteMeshType.Tight
                : SpriteMeshType.FullRect;
            importer.SetTextureSettings(settings);

            importer.userData = userData;
            var standalone = new TextureImporterPlatformSettings
            {
                name = "Standalone",
                overridden = true,
                format = TextureImporterFormat.BC7,
                textureCompression = TextureImporterCompression.Compressed,
                maxTextureSize = 2048,
            };
            importer.SetPlatformTextureSettings(standalone);
            var android = new TextureImporterPlatformSettings
            {
                name = "Android",
                overridden = true,
                format = TextureImporterFormat.ASTC_4x4,
                textureCompression = TextureImporterCompression.Compressed,
                maxTextureSize = 2048,
            };
            importer.SetPlatformTextureSettings(android);
        }

        private static void GetSourceSize(string path, ref int w, ref int h)
        {
            var abs = Path.Combine(
                Path.GetDirectoryName(Application.dataPath) ?? "", path);
            if (!File.Exists(abs))
            {
                return;
            }
            using (var fs = File.OpenRead(abs))
            {
                var head = new byte[26];
                if (fs.Read(head, 0, head.Length) == head.Length
                    && head[0] == 0x89 && head[1] == 0x50)
                {
                    w = (head[16] << 24) | (head[17] << 16)
                        | (head[18] << 8) | head[19];
                    h = (head[20] << 24) | (head[21] << 16)
                        | (head[22] << 8) | head[23];
                }
            }
        }

        private static string Declared(string userData, string key)
        {
            foreach (var raw in userData.Split(';'))
            {
                var token = raw.Trim();
                var eq = token.IndexOf('=');
                if (eq > 0 && token.Substring(0, eq).Trim() == key)
                {
                    return token.Substring(eq + 1).Trim();
                }
            }
            return "";
        }
    }
}
