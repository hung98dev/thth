using UnityEditor;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.CreatureArtCoverage
{
    // IMP-104 editor automation for the monster/boss/Spirit Beast art
    // packet.
    //
    // OnPreprocessTexture applies the canonical sprite import settings to
    // every PNG authored under the packet's owned art directories. Declared
    // import metadata comes from the per-file `.importmeta` sidecar (the
    // IMP-073 convention); StyleRef anchor/turnaround PNGs have no sidecar
    // and import as plain ACTOR references. Everything here is scoped by
    // path to client/Assets/Art/Actors/Creatures/ and
    // client/Assets/Art/StyleRef/actors_creatures/.
    public sealed class CreatureArtImportPolicy : AssetPostprocessor
    {
        public const string CreaturesRoot = "Assets/Art/Actors/Creatures";
        public const string StyleRefRoot = "Assets/Art/StyleRef/actors_creatures";
        public const string AnchorUserData =
            "asset_class=ACTOR;detached_parts";

        private void OnPreprocessTexture()
        {
            if (!IsOwnedPng(assetPath))
            {
                return;
            }
            var importer = (TextureImporter)assetImporter;
            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.spritePixelsPerUnit = 100f;
            importer.mipmapEnabled = false;
            importer.sRGBTexture = true;
            importer.alphaIsTransparency = true;
            importer.filterMode = FilterMode.Bilinear;
            importer.npotScale = TextureImporterNPOTScale.None;
            importer.maxTextureSize = 2048;
            importer.spritePivot = new Vector2(0.5f, 0f);
            importer.spriteBorder = Vector4.zero;
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
            importer.userData = UserDataFor(assetPath);
        }

        private static string UserDataFor(string path)
        {
            var sidecar = path + ".importmeta";
            var abs = System.IO.Path.Combine(
                UnityEngine.Application.dataPath, "..", sidecar);
            if (System.IO.File.Exists(abs))
            {
                var content = System.IO.File.ReadAllText(abs).Trim();
                if (content.Length > 0)
                {
                    return content;
                }
            }
            return AnchorUserData;
        }

        private static bool IsOwnedPng(string path)
        {
            return path.EndsWith(".png", System.StringComparison.Ordinal)
                && (path.StartsWith(CreaturesRoot + "/", System.StringComparison.Ordinal)
                    || path.StartsWith(StyleRefRoot + "/", System.StringComparison.Ordinal));
        }
    }
}
