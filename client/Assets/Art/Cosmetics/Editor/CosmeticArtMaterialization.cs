using System.Collections.Generic;
using System.IO;
using UnityEditor;
using UnityEngine;
using UnityEngine.U2D.Animation;

namespace ThinhThan.Art.Editor
{
    // IMP-074 materialization for cosmetic appearances (ADR-0076): each
    // appearance .psb carries a subset of the fixed PSB layer names and
    // binds the shared class skeleton; one shared Sprite Library maps
    // every appearance cosmetic id onto the layers it replaces, and one
    // .prefab per appearance instances the PSB rig with that library.
    // Idempotent — existing assets are left alone so the section-4b
    // materialization artifact is stable once committed.
    public static class CosmeticArtMaterialization
    {
        private const string GeneratedDir = "Assets/Art/Cosmetics/appearances";
        private const string SpriteLibPath =
            "Assets/Art/Cosmetics/asset.cosmetic.appearance.spriteLib.asset";
        private const string SharedSkeletonPath =
            "Assets/Art/Actors/Players/asset.class.players.skeleton.asset";
        private const string PresentationMapPath =
            "Assets/Art/Cosmetics/cosmetic_presentation_map.json";

        private static readonly string[] Layers =
        {
            "head", "hair", "torso", "arm_front", "arm_back",
            "leg_front", "leg_back", "weapon",
        };

        [InitializeOnLoadMethod]
        private static void Register()
        {
            EditorApplication.delayCall += EnsureGenerated;
        }

        private static void EnsureGenerated()
        {
            if (!Directory.Exists(Path.Combine(
                    Application.dataPath, "Art", "Cosmetics", "appearances")))
            {
                return;
            }
            var ids = AppearanceIds();
            if (ids.Count == 0)
            {
                return;
            }
            var changed = false;
            changed |= BindSharedSkeleton(ids);
            changed |= EnsureSpriteLibrary(ids);
            changed |= EnsurePrefabs(ids);
            if (changed)
            {
                AssetDatabase.SaveAssets();
            }
        }

        // Appearance cosmetic ids and their replaced layers, read from the
        // presentation map so this script and the coverage test share one
        // source of truth.
        public static List<string> AppearanceIds()
        {
            var ids = new List<string>();
            var mapAbs = Path.Combine(
                Path.GetDirectoryName(Application.dataPath) ?? "",
                PresentationMapPath);
            if (!File.Exists(mapAbs))
            {
                return ids;
            }
            var doc = JsonUtility.FromJson<PresentationMapDoc>(
                File.ReadAllText(mapAbs));
            if (doc?.entries == null)
            {
                return ids;
            }
            foreach (var e in doc.entries)
            {
                if (e != null && e.kind == "appearance")
                {
                    ids.Add(e.cosmetic_id);
                }
            }
            return ids;
        }

        // Binds every appearance PSB onto the shared class skeleton
        // (m_SkeletonAssetReferenceID, same mechanism as IMP-071) so a
        // cosmetic layer inherits the class rig exactly.
        private static bool BindSharedSkeleton(List<string> ids)
        {
            var skeletonGuid = AssetDatabase.AssetPathToGUID(
                SharedSkeletonPath);
            if (string.IsNullOrEmpty(skeletonGuid))
            {
                return false;
            }
            var changed = false;
            foreach (var id in ids)
            {
                var psbPath = PsbAssetPathFor(id);
                var importer = AssetImporter.GetAtPath(psbPath);
                if (importer == null)
                {
                    continue;
                }
                var so = new SerializedObject(importer);
                var prop = so.FindProperty("m_SkeletonAssetReferenceID");
                if (prop == null || prop.stringValue == skeletonGuid)
                {
                    continue;
                }
                prop.stringValue = skeletonGuid;
                so.ApplyModifiedPropertiesWithoutUndo();
                importer.SaveAndReimport();
                changed = true;
            }
            return changed;
        }

        private static bool EnsureSpriteLibrary(List<string> ids)
        {
            var existing =
                AssetDatabase.LoadAssetAtPath<SpriteLibraryAsset>(SpriteLibPath);
            if (existing != null)
            {
                return false;
            }
            var lib = ScriptableObject.CreateInstance<SpriteLibraryAsset>();
            foreach (var id in ids)
            {
                foreach (var category in Layers)
                {
                    var sprite = FindLayerSprite(id, category);
                    if (sprite != null)
                    {
                        lib.AddCategoryLabel(sprite, category, id);
                    }
                }
            }
            AssetDatabase.CreateAsset(lib, SpriteLibPath);
            return true;
        }

        private static bool EnsurePrefabs(List<string> ids)
        {
            var lib =
                AssetDatabase.LoadAssetAtPath<SpriteLibraryAsset>(SpriteLibPath);
            var changed = false;
            foreach (var id in ids)
            {
                var prefabPath = PrefabAssetPathFor(id);
                if (AssetDatabase.LoadAssetAtPath<GameObject>(prefabPath)
                    != null)
                {
                    continue;
                }
                var psb = AssetDatabase.LoadAssetAtPath<GameObject>(PsbAssetPathFor(id));
                if (psb == null)
                {
                    continue;
                }
                var instance = PrefabUtility.InstantiatePrefab(psb)
                    as GameObject;
                if (instance == null)
                {
                    continue;
                }
                try
                {
                    if (lib != null)
                    {
                        var libHolder = instance.GetComponent<SpriteLibrary>();
                        if (libHolder == null)
                        {
                            libHolder = instance.AddComponent<SpriteLibrary>();
                        }
                        libHolder.spriteLibraryAsset = lib;
                    }
                    PrefabUtility.SaveAsPrefabAsset(instance, prefabPath);
                    changed = true;
                }
                finally
                {
                    Object.DestroyImmediate(instance);
                }
            }
            return changed;
        }

        private static Sprite FindLayerSprite(string cosmeticId, string layerName)
        {
            foreach (var asset in AssetDatabase.LoadAllAssetsAtPath(
                PsbAssetPathFor(cosmeticId)))
            {
                if (asset is Sprite sprite && sprite.name == layerName)
                {
                    return sprite;
                }
            }
            return null;
        }

        public static string PsbAssetPathFor(string cosmeticId)
        {
            return GeneratedDir + "/" + ShortId(cosmeticId)
                + "/asset." + cosmeticId + ".psb";
        }

        public static string PrefabAssetPathFor(string cosmeticId)
        {
            return GeneratedDir + "/" + ShortId(cosmeticId)
                + "/asset." + cosmeticId + ".prefab";
        }

        public static string IconAssetPathFor(string cosmeticId)
        {
            return GeneratedDir + "/" + ShortId(cosmeticId)
                + "/asset." + cosmeticId + ".icon.png";
        }

        private static string ShortId(string cosmeticId)
        {
            const string prefix = "cosmetic.";
            var tail = cosmeticId.StartsWith(prefix, System.StringComparison.Ordinal)
                ? cosmeticId.Substring(prefix.Length)
                : cosmeticId;
            const string iapPrefix = "iap.appearance.";
            return tail.StartsWith(iapPrefix, System.StringComparison.Ordinal)
                ? tail.Substring(iapPrefix.Length)
                : tail.Substring("appearance.".Length);
        }

        [System.Serializable]
        private sealed class PresentationMapDoc
        {
            public List<Entry>? entries;

            [System.Serializable]
            public sealed class Entry
            {
                public string cosmetic_id = "";
                public string kind = "";
            }
        }
    }
}
