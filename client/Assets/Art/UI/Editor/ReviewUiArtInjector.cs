using System.IO;
using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;

namespace ThinhThan.Art.Editor
{
    // IMP-073: binds the produced interface art into the IMP-070 review
    // scene when it is opened (materialization + Visual Review renders
    // run the scene in-memory; the committed scene asset is unchanged).
    // UiAnchor's SpriteRenderer gets the produced showcase sprite so the
    // rendered captures exercise a real UI surface instead of the shell
    // placeholder, and the anchor is scaled to the reference ortho view.
    [InitializeOnLoad]
    public static class ReviewUiArtInjector
    {
        private const string ReviewSceneSuffix = "Scenes/Review/ReviewUI.unity";
        private const string AnchorName = "UiAnchor";
        private const string ShowcasePath = "Assets/Art/UI/showcase/hud_showcase.png";

        static ReviewUiArtInjector()
        {
            EditorSceneManager.sceneOpened += OnSceneOpened;
        }

        private static void OnSceneOpened(UnityEngine.SceneManagement.Scene scene,
                                          OpenSceneMode mode)
        {
            if (!scene.path.EndsWith(ReviewSceneSuffix, System.StringComparison.Ordinal))
            {
                return;
            }
            var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(ShowcasePath);
            var anchor = GameObject.Find(AnchorName);
            if (sprite == null || anchor == null)
            {
                return;
            }
            var renderer = anchor.GetComponent<SpriteRenderer>();
            if (renderer == null)
            {
                return;
            }
            renderer.sprite = sprite;
            renderer.color = Color.white;
            var cam = GameObject.Find("ReviewCamera");
            var camComponent = cam != null ? cam.GetComponent<Camera>() : null;
            var ortho = camComponent != null ? camComponent.orthographicSize : 3.6f;
            var texW = sprite.texture.width;
            var texH = sprite.texture.height;
            var worldW = texW / sprite.pixelsPerUnit;
            var worldH = texH / sprite.pixelsPerUnit;
            var scale = Mathf.Min(ortho * 1.8f / worldW, ortho * 1.6f / worldH);
            anchor.transform.localScale = new Vector3(scale, scale, 1f);
        }
    }
}
