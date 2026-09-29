using UnityEditor;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.PlayerArtCoverage
{
    // IMP-071 editor automation for the player-class art packet.
    //
    // OnPreprocessTexture applies the canonical sprite import settings and
    // the §3.1a import-metadata userData to every PNG authored under the
    // packet's owned art directories. Everything here is scoped by path to
    // client/Assets/Art/Actors/Players/ and
    // client/Assets/Art/StyleRef/actors_players/ — other art packets and
    // the IMP-063 placeholder registry are untouched.
    public sealed class PlayerArtImportPolicy : AssetPostprocessor
    {
        public const string ActorsRoot = "Assets/Art/Actors/Players";
        public const string StyleRefRoot = "Assets/Art/StyleRef/actors_players";
        public const string PlayerUserData =
            "asset_class=ACTOR;size_profile=CHARACTER;detached_parts";
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
            importer.userData = IsCharacterCell(assetPath)
                ? PlayerUserData : AnchorUserData;
        }

        private static bool IsCharacterCell(string path)
        {
            if (path.StartsWith(ActorsRoot + "/", System.StringComparison.Ordinal))
            {
                return true;
            }
            return path.StartsWith(
                StyleRefRoot + "/turnarounds/",
                System.StringComparison.Ordinal);
        }

        private static bool IsOwnedPng(string path)
        {
            return path.EndsWith(".png", System.StringComparison.Ordinal)
                && (path.StartsWith(ActorsRoot + "/", System.StringComparison.Ordinal)
                    || path.StartsWith(StyleRefRoot + "/", System.StringComparison.Ordinal));
        }
    }
}
