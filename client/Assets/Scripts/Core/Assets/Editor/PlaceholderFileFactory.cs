using System.IO;
using UnityEditor;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor
{
    // Creates the placeholder backing files for addressable entries.
    // Icon facets get a real sprite PNG (128x128, transparent margin)
    // with the canonical import settings applied; other facets get a
    // serialized AddressablePlaceholder; alias keys get a
    // PresentationAlias ScriptableObject.
    public static class PlaceholderFileFactory
    {
        public const int IconTextureSize = 128;
        private const int _iconMarginPx = 4;

        public static void EnsureIcon(string path)
        {
            EnsureFolder(Path.GetDirectoryName(path));
            if (File.Exists(path))
            {
                return;
            }
            var size = IconTextureSize;
            var tex = new Texture2D(size, size, TextureFormat.RGBA32, false);
            var pixels = new Color32[size * size];
            for (var i = 0; i < pixels.Length; i++)
            {
                var x = i % size;
                var y = i / size;
                var inside = x >= _iconMarginPx && x < size - _iconMarginPx
                    && y >= _iconMarginPx && y < size - _iconMarginPx;
                pixels[i] = inside ? new Color32(72, 88, 120, 255) : new Color32(0, 0, 0, 0);
            }
            tex.SetPixels32(pixels);
            tex.Apply(false, false);
            File.WriteAllBytes(path, tex.EncodeToPNG());
            Object.DestroyImmediate(tex);
            AssetDatabase.ImportAsset(path, ImportAssetOptions.ForceUpdate);
            ApplyIconImportSettings(path);
        }

        public static void EnsurePlaceholderAsset(string path, string catalogId, AssetFacet facet, string assetClass, string sizeProfile)
        {
            EnsureFolder(Path.GetDirectoryName(path));
            var existing = AssetDatabase.LoadAssetAtPath<AddressablePlaceholder>(path);
            if (existing != null)
            {
                return;
            }
            var so = ScriptableObject.CreateInstance<AddressablePlaceholder>();
            so.Initialize(catalogId, facet.ToKeySegment(), assetClass, sizeProfile);
            AssetDatabase.CreateAsset(so, path);
        }

        public static void EnsureAliasAsset(string path, string targetKey)
        {
            EnsureFolder(Path.GetDirectoryName(path));
            var existing = AssetDatabase.LoadAssetAtPath<PresentationAlias>(path);
            if (existing != null && existing.TargetKey == targetKey)
            {
                return;
            }
            if (existing == null)
            {
                var so = ScriptableObject.CreateInstance<PresentationAlias>();
                so.TargetKey = targetKey;
                AssetDatabase.CreateAsset(so, path);
                return;
            }
            existing.TargetKey = targetKey;
            EditorUtility.SetDirty(existing);
        }

        private static void ApplyIconImportSettings(string path)
        {
            var importer = AssetImporter.GetAtPath(path) as TextureImporter;
            if (importer == null)
            {
                return;
            }
            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.spritePixelsPerUnit = SpriteImportRules.UiPpu;
            importer.spritePivot = new Vector2(0.5f, 0.5f);
            importer.filterMode = FilterMode.Bilinear;
            importer.mipmapEnabled = false;
            importer.maxTextureSize = 2048;
            importer.npotScale = TextureImporterNPOTScale.None;
            var android = importer.GetPlatformTextureSettings("Android");
            android.overridden = true;
            android.format = TextureImporterFormat.ASTC_4x4;
            importer.SetPlatformTextureSettings(android);
            var standalone = importer.GetPlatformTextureSettings("Standalone");
            standalone.overridden = true;
            standalone.format = TextureImporterFormat.BC7;
            importer.SetPlatformTextureSettings(standalone);
            importer.SaveAndReimport();
        }

        private static void EnsureFolder(string? projectRelativeDir)
        {
            if (string.IsNullOrEmpty(projectRelativeDir) || Directory.Exists(projectRelativeDir))
            {
                return;
            }
            EnsureFolder(Path.GetDirectoryName(projectRelativeDir));
            var parent = Path.GetDirectoryName(projectRelativeDir);
            if (parent != null && !AssetDatabase.IsValidFolder(projectRelativeDir))
            {
                AssetDatabase.CreateFolder(parent.Replace('\\', '/'), Path.GetFileName(projectRelativeDir));
            }
        }
    }
}
