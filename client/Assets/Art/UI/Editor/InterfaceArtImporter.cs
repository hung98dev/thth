using System;
using System.IO;
using UnityEditor;
using UnityEngine;

namespace ThinhThan.Art.Editor
{
    // IMP-073 interface art import rules: stamps the canonical import
    // profile (presentation_asset_manifest.md section 3/3.1a, SpriteImportRules)
    // onto every PNG under Assets/Art that carries a sidecar declaration
    // file (<asset>.importmeta). The sidecar holds the importer userData
    // verbatim (asset_class=..., cell_ref=..., soft_edges, nine_slice=...,
    // draw_mode=..., frames=..., fps=..., blend=..., max_instances=...)
    // so CI materialization produces identical .meta files byte-for-byte.
    // Runs in Assembly-CSharp-Editor during batch import; also exposed as
    // a menu entry for manual re-application.
    public sealed class InterfaceArtImporter : AssetPostprocessor
    {
        private const string ArtRoot = "Assets/Art/";
        private const string SidecarSuffix = ".importmeta";

        private void OnPreprocessTexture()
        {
            if (!assetPath.StartsWith(ArtRoot, StringComparison.Ordinal)
                || !assetPath.EndsWith(".png", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
            var sidecar = Path.Combine(
                Path.GetDirectoryName(Application.dataPath) ?? "",
                assetPath + SidecarSuffix);
            if (!File.Exists(sidecar))
            {
                return;
            }
            var importer = (TextureImporter)assetImporter;
            var userData = File.ReadAllText(sidecar).Trim();
            var assetClass = Declared(userData, "asset_class");
            var isUiClass = assetClass == "UI_ART"
                || assetClass == "ITEM_ICON"
                || assetClass == "EQUIPMENT_ICON"
                || assetClass == "FONT_ATLAS";

            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.spritePixelsPerUnit = isUiClass ? 200 : 100;
            importer.filterMode = FilterMode.Bilinear;
            importer.mipmapEnabled = false;
            importer.maxTextureSize = 2048;
            importer.npotScale = TextureImporterNPOTScale.None;
            importer.alphaIsTransparency = true;
            importer.isReadable = true;

            var settings = new TextureImporterSettings();
            importer.ReadTextureSettings(settings);
            settings.spriteAlignment = (int)SpriteAlignment.Custom;
            settings.spritePivot = isUiClass
                ? new Vector2(0.5f, 0.5f)
                : new Vector2(0.5f, 0f);
            settings.spriteGenerateFallbackPhysicsShape = false;
            var nineSlice = Declared(userData, "nine_slice");
            if (nineSlice != null)
            {
                var b = nineSlice.Split(',');
                if (b.Length == 4)
                {
                    settings.spriteBorder = new Vector4(
                        float.Parse(b[0]), float.Parse(b[3]),
                        float.Parse(b[1]), float.Parse(b[2]));
                }
            }
            settings.spriteMeshType = Declared(userData, "nine_slice") == null
                && LongSide(atPath: assetPath) >= 256
                ? SpriteMeshType.Tight
                : SpriteMeshType.FullRect;
            importer.SetTextureSettings(settings);

            importer.userData = userData;
            var android = importer.GetPlatformTextureSettings("Android");
            android.overridden = true;
            android.format = TextureImporterFormat.ASTC_4x4;
            importer.SetPlatformTextureSettings(android);
            var standalone = importer.GetPlatformTextureSettings("Standalone");
            standalone.overridden = true;
            standalone.format = TextureImporterFormat.BC7;
            importer.SetPlatformTextureSettings(standalone);
        }

        private static string Declared(string userData, string key)
        {
            foreach (var raw in userData.Split(';'))
            {
                var token = raw.Trim();
                var eq = token.IndexOf('=');
                var k = eq >= 0 ? token.Substring(0, eq).Trim() : token;
                if (k == key)
                {
                    return eq >= 0 ? token.Substring(eq + 1).Trim() : "";
                }
            }
            return null;
        }

        private static int LongSide(string atPath)
        {
            var abs = Path.Combine(
                Path.GetDirectoryName(Application.dataPath) ?? "", atPath);
            try
            {
                using (var fs = File.OpenRead(abs))
                {
                    var head = new byte[24];
                    if (fs.Read(head, 0, 24) < 24)
                    {
                        return 0;
                    }
                    var w = (head[16] << 24) | (head[17] << 16) | (head[18] << 8) | head[19];
                    var h = (head[20] << 24) | (head[21] << 16) | (head[22] << 8) | head[23];
                    return Math.Max(w, h);
                }
            }
            catch
            {
                return 0;
            }
        }
    }
}
