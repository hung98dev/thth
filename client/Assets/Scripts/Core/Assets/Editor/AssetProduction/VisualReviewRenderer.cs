using System.Collections.Generic;
using System.IO;
using System.Text;
using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEngine.Rendering;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Visual Review renderer of section 3.3: renders the review scenes under
    // the fixed resolution/lighting/zoom matrix and writes
    // artifacts/visual-review/ (gitignored) for the CI upload step.
    //
    // CI integration: verify.yml materializes the project in a plain
    // `-batchmode -quit` invocation that HAS a GL device (xvfb + Mesa
    // llvmpipe on Linux), while EditMode/PlayMode test runs use
    // `-nographics`. The InitializeOnLoadMethod + delayCall hook therefore
    // renders only when (batch mode) AND (graphics device present) AND
    // (Linux editor) AND (not a -runTests invocation). Editor interactive
    // use goes through the ThinhThan menu items instead.
    public static class VisualReviewRenderer
    {
        private const string OutputDirName = "artifacts/visual-review";
        private const string CameraName = "ReviewCamera";
        private const string LightName = "ReviewLight";

        private static readonly Color DaySky = new Color(0.45f, 0.62f, 0.82f);
        private static readonly Color NightSky = new Color(0.04f, 0.05f, 0.10f);
        private static readonly Color DayAmbient = new Color(0.55f, 0.62f, 0.75f);
        private static readonly Color NightAmbient = new Color(0.05f, 0.07f, 0.12f);
        private static readonly Color DayLight = new Color(1.00f, 0.96f, 0.88f);
        private static readonly Color NightLight = new Color(0.56f, 0.66f, 0.82f);

        private static readonly List<string> _diag = new List<string>();

        [InitializeOnLoadMethod]
        private static void Install()
        {
            EditorApplication.delayCall += BatchHook;
        }

        private static void BatchHook()
        {
            if (!Application.isBatchMode || Application.platform != RuntimePlatform.LinuxEditor)
            {
                return;
            }
            // The marker file makes the artifact upload non-empty even when a
            // gate rejects the run, so the captured gate values are visible in
            // the visual-review artifact rather than only the editor log.
            var outRoot = OutputRoot();
            var runTests = System.Environment.CommandLine.Contains("-runTests");
            Directory.CreateDirectory(outRoot);
            File.WriteAllText(
                Path.Combine(outRoot, "_hook.txt"),
                "batch=" + Application.isBatchMode
                    + " gfx=" + SystemInfo.graphicsDeviceType + "/" + SystemInfo.graphicsDeviceName
                    + " platform=" + Application.platform
                    + " runTests=" + runTests + "\n");
            if (SystemInfo.graphicsDeviceType == GraphicsDeviceType.Null || runTests)
            {
                return;
            }
            try
            {
                var written = RenderAll(outRoot);
                File.AppendAllText(
                    Path.Combine(outRoot, "_hook.txt"),
                    "rendered=" + written.Count + " diag=[" + string.Join("; ", _diag) + "]\n");
            }
            catch (System.Exception e)
            {
                Debug.LogException(e);
                EditorApplication.Exit(1);
            }
        }

        // Project root-relative output root: <repo>/artifacts/visual-review.
        public static string OutputRoot()
        {
            var repoRoot = Path.GetFullPath(Path.Combine(Application.dataPath, "..", ".."));
            return Path.Combine(repoRoot, OutputDirName);
        }

        // Renders every scene of the matrix; returns the capture paths.
        public static List<string> RenderAll(string outRoot)
        {
            var written = new List<string>();
            foreach (var scenePath in VisualReviewMatrix.ScenePaths)
            {
                // Scene paths are project-relative; the editor process cwd is
                // not the project root in CI (docker workdir is the repo
                // root), so existence must go through the AssetDatabase.
                var sceneAsset = AssetDatabase.LoadAssetAtPath<SceneAsset>(scenePath);
                _diag.Add(scenePath + " -> " + (sceneAsset == null ? "missing" : "found"));
                if (sceneAsset == null)
                {
                    Debug.LogWarning("visual-review: scene missing " + scenePath);
                    continue;
                }
                RenderScene(scenePath, outRoot, written);
            }
            WriteReviewAids(outRoot, written);
            return written;
        }

        public static void RenderScene(string scenePath, string outRoot, List<string> written)
        {
            var sceneName = Path.GetFileNameWithoutExtension(scenePath);
            var scene = EditorSceneManager.OpenScene(scenePath, OpenSceneMode.Single);
            var camera = FindCamera();
            if (camera == null)
            {
                _diag.Add(scenePath + " -> no " + CameraName);
                Debug.LogError("visual-review: no " + CameraName + " in " + scenePath);
                return;
            }
            var baseOrtho = camera.orthographicSize;
            var meta = new StringBuilder();
            meta.Append("{\"scene\":\"").Append(sceneName).Append("\",");
            meta.Append("\"renderer\":\"").Append(EscapeJson(SystemInfo.graphicsDeviceName)).Append("\",");
            meta.Append("\"graphics_api\":\"").Append(SystemInfo.graphicsDeviceType.ToString()).Append("\",");
            meta.Append("\"unity\":\"").Append(Application.unityVersion).Append("\",");
            meta.Append("\"captures\":[");
            var first = true;
            foreach (var combo in VisualReviewMatrix.Combos())
            {
                var rel = sceneName + "/" + combo.Width + "x" + combo.Height + "/"
                    + combo.Lighting + "/" + combo.ZoomPercent + "/full.png";
                RenderCombo(camera, baseOrtho, combo, Path.Combine(outRoot, rel));
                written.Add(rel);
                if (!first)
                {
                    meta.Append(',');
                }
                first = false;
                meta.Append("{\"file\":\"").Append(rel).Append("\",")
                    .Append("\"width\":").Append(combo.Width).Append(",")
                    .Append("\"height\":").Append(combo.Height).Append(",")
                    .Append("\"lighting\":\"").Append(combo.Lighting).Append("\",")
                    .Append("\"zoom\":").Append(combo.ZoomPercent).Append('}');
            }
            if (sceneName == "ReviewEnvironment")
            {
                RenderEnvironmentLayers(
                    camera, baseOrtho, outRoot, sceneName, meta, written);
            }
            if (sceneName == "ReviewActor")
            {
                RenderLowProfile(
                    camera, baseOrtho, outRoot, sceneName, meta, written);
            }
            meta.Append("]}");
            var metaPath = Path.Combine(outRoot, sceneName, "_run.json");
            Directory.CreateDirectory(Path.GetDirectoryName(metaPath)!);
            File.WriteAllText(metaPath, meta.ToString());
            written.Add(sceneName + "/_run.json");
        }

        // ART-006 (section 3.3): the LOW profile renders the actor scene at
        // 960x540 through a 2 s horizontal motion clip — start, midpoint
        // and end stills of the actor moved across the frame — so the
        // reviewer can score shimmer against the LOW preset. ART-011 adds
        // the contact sheet and the 0/1/2 rubric template next to it.
        private static void RenderLowProfile(
            Camera camera,
            float baseOrtho,
            string outRoot,
            string sceneName,
            StringBuilder meta,
            List<string> written)
        {
            var actor = GameObject.Find(VisualReviewMatrix.LowMotionObject);
            var startPos = actor != null ? actor.transform.position : Vector3.zero;
            var combo = new VisualReviewCombo
            {
                Width = VisualReviewMatrix.LowResolution.Width,
                Height = VisualReviewMatrix.LowResolution.Height,
                Lighting = "day",
                ZoomPercent = 100,
            };
            // 1 world unit of travel over the clip; sampled at t = 0/1/2 s.
            var offsets = new[] { 0f, 0.5f, 1f };
            foreach (var t in offsets)
            {
                if (actor != null)
                {
                    actor.transform.position = startPos + new Vector3(t, 0f, 0f);
                }
                var rel = sceneName + "/960x540/day/100/low_t" + (int)(t * 2) + ".png";
                RenderCombo(camera, baseOrtho, combo, Path.Combine(outRoot, rel));
                written.Add(rel);
                meta.Append(",{\"file\":\"").Append(rel)
                    .Append("\",\"profile\":\"low\",\"t\":").Append(t).Append('}');
            }
            if (actor != null)
            {
                actor.transform.position = startPos;
            }
        }

        // ART-011 (section 3.3): the rubric template the reviewer fills
        // per capture (0/1/2 per criterion; pass iff no 0 and total >= 80%
        // of max) plus the contact sheet placed beside the Style Pack
        // anchors. Written once per RenderAll run under _review/.
        public static void WriteReviewAids(string outRoot, List<string> written)
        {
            var reviewDir = Path.Combine(outRoot, "_review");
            Directory.CreateDirectory(reviewDir);
            var rubric = new StringBuilder();
            rubric.Append("{\n  \"criteria\": {\n");
            var criteria = new[]
            {
                "cutout_clean", "silhouette_readable", "volume_toplit",
                "value_tiers", "edge_separation", "palette_consistent",
                "cultural_motif", "shimmer_low",
            };
            for (var i = 0; i < criteria.Length; i++)
            {
                if (i > 0)
                {
                    rubric.Append(",\n");
                }
                rubric.Append("    \"").Append(criteria[i])
                    .Append("\": {\"score\": null, \"note\": \"\"}");
            }
            rubric.Append("\n  },\n  \"scores\": \"0|1|2 per criterion; PASS = no 0 and total >= 80% of max\",\n");
            rubric.Append("  \"shimmer\": \"none|visible\",\n  \"verdict\": null\n}\n");
            var rubricPath = Path.Combine(reviewDir, "rubric.json");
            File.WriteAllText(rubricPath, rubric.ToString());
            written.Add("_review/rubric.json");

            var repoRoot = Path.GetFullPath(Path.Combine(Application.dataPath, "..", ".."));
            var styleRefRoot = Path.Combine(repoRoot, "client", "Assets", "Art", "StyleRef");
            var sheet = new StringBuilder();
            sheet.Append("# Visual Review contact sheet\n\n");
            var packs = 0;
            if (Directory.Exists(styleRefRoot))
            {
                foreach (var packDir in Directory.GetDirectories(styleRefRoot, "*", SearchOption.AllDirectories))
                {
                    var palette = Path.Combine(packDir, "palette.json");
                    var styleMd = Path.Combine(packDir, "style.md");
                    if (!File.Exists(palette) && !File.Exists(styleMd))
                    {
                        continue;
                    }
                    var rel = packDir.Substring(styleRefRoot.Length).Replace('\\', '/').TrimStart('/');
                    var anchors = 0;
                    foreach (var f in Directory.GetFiles(packDir))
                    {
                        var ext = Path.GetExtension(f).ToLowerInvariant();
                        if (ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp")
                        {
                            anchors++;
                        }
                    }
                    sheet.Append("- `").Append(rel).Append("` — ").Append(anchors)
                        .Append(" anchors; place each reviewed asset beside these references.\n");
                    packs++;
                }
            }
            if (packs == 0)
            {
                sheet.Append("(no Style Packs under client/Assets/Art/StyleRef yet)\n");
            }
            var sheetPath = Path.Combine(reviewDir, "contact_sheet.md");
            File.WriteAllText(sheetPath, sheet.ToString());
            written.Add("_review/contact_sheet.md");
        }

        // Environment rule (section 3.6): isolated 1280x720 day captures of
        // each depth layer so contrast/chroma can be compared L1 -> L4.
        private static void RenderEnvironmentLayers(
            Camera camera,
            float baseOrtho,
            string outRoot,
            string sceneName,
            StringBuilder meta,
            List<string> written)
        {
            var roots = new List<GameObject>();
            foreach (var name in VisualReviewMatrix.EnvironmentLayers)
            {
                var go = GameObject.Find(name);
                if (go != null)
                {
                    roots.Add(go);
                }
            }
            if (roots.Count == 0)
            {
                return;
            }
            foreach (var root in roots)
            {
                root.SetActive(false);
            }
            var combo = new VisualReviewCombo
            {
                Width = 1280,
                Height = 720,
                Lighting = "day",
                ZoomPercent = 100,
            };
            foreach (var root in roots)
            {
                root.SetActive(true);
                var rel = sceneName + "/" + combo.Width + "x" + combo.Height + "/"
                    + combo.Lighting + "/" + combo.ZoomPercent + "/"
                    + root.name + ".png";
                RenderCombo(camera, baseOrtho, combo, Path.Combine(outRoot, rel));
                written.Add(rel);
                meta.Append(",{\"file\":\"").Append(rel).Append("\",\"layer\":\"")
                    .Append(root.name).Append("\"}");
                root.SetActive(false);
            }
            foreach (var root in roots)
            {
                root.SetActive(true);
            }
        }

        private static void RenderCombo(
            Camera camera,
            float baseOrtho,
            VisualReviewCombo combo,
            string outPath)
        {
            ApplyLighting(combo.Lighting == "day", camera);
            camera.orthographicSize = baseOrtho * 100f / combo.ZoomPercent;
            var rt = new RenderTexture(combo.Width, combo.Height, 24);
            var prevActive = RenderTexture.active;
            var prevTarget = camera.targetTexture;
            try
            {
                camera.targetTexture = rt;
                camera.Render();
                RenderTexture.active = rt;
                var tex = new Texture2D(combo.Width, combo.Height, TextureFormat.RGBA32, false);
                tex.ReadPixels(new Rect(0, 0, combo.Width, combo.Height), 0, 0);
                tex.Apply();
                RenderTexture.active = prevActive;
                camera.targetTexture = prevTarget;
                Directory.CreateDirectory(Path.GetDirectoryName(outPath)!);
                File.WriteAllBytes(outPath, ImageConversion.EncodeToPNG(tex));
                Object.DestroyImmediate(tex);
            }
            finally
            {
                RenderTexture.active = prevActive;
                camera.targetTexture = prevTarget;
                rt.Release();
            }
        }

        // Day/night for review purposes: sky/ambient colour plus a
        // directional key light. Sprites are authored unlit with painted
        // volume, so the captures differ in sky, ambient and any lit
        // content, matching the runtime lighting model.
        private static void ApplyLighting(bool day, Camera camera)
        {
            RenderSettings.ambientMode = AmbientMode.Flat;
            RenderSettings.ambientLight = day ? DayAmbient : NightAmbient;
            RenderSettings.ambientIntensity = day ? 1.0f : 0.35f;
            camera.clearFlags = CameraClearFlags.SolidColor;
            camera.backgroundColor = day ? DaySky : NightSky;
            var lightGo = GameObject.Find(LightName);
            Light? light = null;
            if (lightGo != null)
            {
                light = lightGo.GetComponent<Light>();
            }
            if (light == null)
            {
                lightGo = new GameObject(LightName);
                light = lightGo.AddComponent<Light>();
                light.type = LightType.Directional;
            }
            light.color = day ? DayLight : NightLight;
            light.intensity = day ? 1.0f : 0.35f;
        }

        private static Camera? FindCamera()
        {
            var go = GameObject.Find(CameraName);
            return go == null ? null : go.GetComponent<Camera>();
        }

        private static string EscapeJson(string s)
        {
            return s.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
