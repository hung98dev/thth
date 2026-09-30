using UnityEditor;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.CreatureArtCoverage
{
    // IMP-104 editor automation for the monster/boss/Spirit Beast art
    // packet.
    //
    // OnPreprocessTexture applies the canonical sprite import settings to
    // every PNG authored under the packet's owned art directories. Declared
    // import metadata (`asset_class=ACTOR;size_profile=<map profile>`) is
    // resolved from the presentation map by entity id — the file name
    // `asset.<entity_id>.<rest>.png` carries the full dotted id, and
    // style-pack anchors/turnarounds are matched by longest entity-id
    // prefix. Everything here is scoped by path to
    // client/Assets/Art/Actors/Creatures/ and
    // client/Assets/Art/StyleRef/actors_creatures/.
    public sealed class CreatureArtImportPolicy : AssetPostprocessor
    {
        public const string CreaturesRoot = "Assets/Art/Actors/Creatures";
        public const string StyleRefRoot = "Assets/Art/StyleRef/actors_creatures";

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
            var stem = System.IO.Path.GetFileNameWithoutExtension(path);
            var profile = ProfileFor(stem);
            if (profile != null)
            {
                return "asset_class=ACTOR;size_profile=" + profile
                    + ";detached_parts";
            }
            return "asset_class=ACTOR;detached_parts";
        }

        // Resolves the size profile by longest entity-id prefix match:
        // `asset.<entity_id>...` under Creatures, `<entity_id>[_<view>]`
        // under StyleRef. Returns null when nothing matches (foreign file).
        private static string ProfileFor(string stem)
        {
            var map = CreatureArtMaterialization.LoadMap();
            if (map == null || map.entries == null)
            {
                return null;
            }
            var fileKey = stem;
            const string prefix = "asset.";
            if (fileKey.StartsWith(prefix, System.StringComparison.Ordinal))
            {
                fileKey = fileKey.Substring(prefix.Length);
            }
            string? best = null;
            string? profile = null;
            foreach (var e in map.entries)
            {
                if (e == null || e.entity_id == null)
                {
                    continue;
                }
                if (!fileKey.StartsWith(e.entity_id, System.StringComparison.Ordinal)
                    || (e.entity_id.Length >= fileKey.Length
                        && e.entity_id != fileKey))
                {
                    continue;
                }
                if (best == null || e.entity_id.Length > best.Length)
                {
                    best = e.entity_id;
                    profile = e.size_profile;
                }
            }
            return profile;
        }

        private static bool IsOwnedPng(string path)
        {
            return path.EndsWith(".png", System.StringComparison.Ordinal)
                && (path.StartsWith(CreaturesRoot + "/", System.StringComparison.Ordinal)
                    || path.StartsWith(StyleRefRoot + "/", System.StringComparison.Ordinal));
        }
    }
}
