using System.Collections.Generic;
using System.IO;
using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;

namespace ThinhThan.Art.Editor
{
    // IMP-074: binds the produced cosmetic art into the IMP-070 review
    // scenes when they are opened for materialization/render (the
    // committed scene assets are never modified). ReviewUI gets a grid
    // of every produced cosmetic texture so the captures exercise real
    // presentation art; ReviewActor gets the eight appearance prefabs
    // spaced around ActorAnchor so the shared-skeleton binding renders.
    [InitializeOnLoad]
    public static class ReviewCosmeticInjector
    {
        private const string UiSceneSuffix = "Scenes/Review/ReviewUI.unity";
        private const string ActorSceneSuffix =
            "Scenes/Review/ReviewActor.unity";
        private const string UiAnchorName = "UiAnchor";
        private const string ActorAnchorName = "ActorAnchor";
        private const string CosmeticsRoot = "Assets/Art/Cosmetics/";
        private const string AppearancesRoot =
            "Assets/Art/Cosmetics/appearances/";
        private const float IconPpu = 200f;
        private const int IconColumns = 15;

        static ReviewCosmeticInjector()
        {
            EditorSceneManager.sceneOpened += OnSceneOpened;
        }

        private static void OnSceneOpened(UnityEngine.SceneManagement.Scene scene,
                                          OpenSceneMode mode)
        {
            if (scene.path.EndsWith(UiSceneSuffix, System.StringComparison.Ordinal))
            {
                MountUiGrid();
            }
            else if (scene.path.EndsWith(
                ActorSceneSuffix, System.StringComparison.Ordinal))
            {
                MountAppearances();
            }
        }

        private static void MountUiGrid()
        {
            var anchor = GameObject.Find(UiAnchorName);
            if (anchor == null)
            {
                return;
            }
            var sprites = new List<Sprite>();
            CollectSprites(sprites, CosmeticsRoot);
            if (sprites.Count == 0)
            {
                return;
            }
            var spacing = 128f / IconPpu * 1.1f;
            var rows = (sprites.Count + IconColumns - 1) / IconColumns;
            var gridW = IconColumns * spacing;
            var gridH = Mathf.Max(1, rows) * spacing;
            var cam = GameObject.Find("ReviewCamera");
            var camComponent = cam != null ? cam.GetComponent<Camera>() : null;
            var ortho = camComponent != null ? camComponent.orthographicSize : 3.6f;
            var aspect = camComponent != null ? camComponent.aspect : 16f / 9f;
            var scale = Mathf.Min(
                ortho * 1.86f * aspect / gridW,
                ortho * 1.86f / gridH);
            for (var i = 0; i < sprites.Count; i++)
            {
                var col = i % IconColumns;
                var row = i / IconColumns;
                var cell = new GameObject("CosmeticIcon_" + i);
                cell.transform.SetParent(anchor.transform, false);
                cell.transform.localPosition = new Vector3(
                    (col - (IconColumns - 1) * 0.5f) * spacing,
                    ((rows - 1) * 0.5f - row) * spacing,
                    -0.1f);
                var sr = cell.AddComponent<SpriteRenderer>();
                sr.sprite = sprites[i];
                sr.sortingOrder = 1;
            }
            anchor.transform.localScale = new Vector3(scale, scale, 1f);
            var anchorRenderer = anchor.GetComponent<SpriteRenderer>();
            if (anchorRenderer != null)
            {
                anchorRenderer.enabled = false;
            }
        }

        private static void CollectSprites(List<Sprite> into, string rootAssetPath)
        {
            var abs = Path.Combine(
                Path.GetDirectoryName(Application.dataPath) ?? "",
                rootAssetPath.TrimEnd('/'));
            if (!Directory.Exists(abs))
            {
                return;
            }
            foreach (var dir in Directory.GetDirectories(abs))
            {
                var rel = rootAssetPath + Path.GetFileName(dir) + "/";
                CollectSprites(into, rel);
            }
            foreach (var file in Directory.GetFiles(abs, "*.png"))
            {
                var rel = rootAssetPath + Path.GetFileName(file);
                var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(rel);
                if (sprite != null)
                {
                    into.Add(sprite);
                }
            }
        }

        private static void MountAppearances()
        {
            var anchor = GameObject.Find(ActorAnchorName);
            if (anchor == null)
            {
                return;
            }
            var ids = CosmeticArtMaterialization.AppearanceIds();
            var spacing = 1.35f;
            var placed = 0;
            foreach (var id in ids)
            {
                var prefab = AssetDatabase.LoadAssetAtPath<GameObject>(
                    CosmeticArtMaterialization.PrefabAssetPathFor(id));
                if (prefab == null)
                {
                    continue;
                }
                var go = (GameObject)PrefabUtility.InstantiatePrefab(prefab);
                go.name = "CosmeticAnchor_" + id;
                go.transform.position = anchor.transform.position
                    + new Vector3(
                        (placed - (ids.Count - 1) * 0.5f) * spacing,
                        0f, -0.1f);
                placed++;
            }
        }
    }
}
