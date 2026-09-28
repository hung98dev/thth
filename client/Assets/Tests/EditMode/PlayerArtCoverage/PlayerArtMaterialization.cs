using System.IO;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEditor;
using UnityEditor.Animations;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEngine.U2D;
using UnityEngine.U2D.Animation;

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

    // Materializes the rig assets the spec cannot express as hand-edited
    // YAML: one shared skeleton for the five classes, one shared
    // AnimatorController holding the ten §3.7 clips, one prefab per class
    // (imported .psb + Animator) and one Sprite Library whose categories
    // are the fixed PSB layer names. Idempotent — files that already
    // exist are left alone, so the §4b materialization artifact is stable
    // once committed.
    public static class PlayerArtMaterialization
    {
        private const string GeneratedDir = "Assets/Art/Actors/Players";
        private const string SkeletonPath =
            GeneratedDir + "/asset.class.players.skeleton.asset";
        private const string ControllerPath =
            GeneratedDir + "/asset.class.players.controller";
        private const string SpriteLibPath =
            GeneratedDir + "/asset.class.players.spriteLib.asset";
        private const string AnimDir = GeneratedDir + "/anim";

        private static readonly string[] ClassIds =
        {
            "kim", "moc", "thuy", "hoa", "tho",
        };

        private static readonly string[] Layers = AnimationContract.PsbLayers;

        [InitializeOnLoadMethod]
        private static void Register()
        {
            EditorApplication.delayCall += EnsureGenerated;
            EditorSceneManager.sceneOpened += OnSceneOpened;
        }

        // Skeleton rig shared by all five classes (§3.7): a canonical
        // bone layout whose positions are fractions of the 96x128
        // reference cell (pivot Bottom Center, feet at y = 0).
        private static readonly (string Name, int Parent, Vector3 Pos)[]
            SkeletonSpec =
        {
            ("torso", -1, new Vector3(0f, 0.55f, 0f)),
            ("head", 0, new Vector3(0f, 1.15f, 0f)),
            ("hair", 1, new Vector3(0f, 1.45f, 0f)),
            ("arm_front", 0, new Vector3(0.28f, 1.0f, -0.01f)),
            ("arm_back", 0, new Vector3(-0.28f, 1.0f, 0.01f)),
            ("leg_front", 0, new Vector3(0.14f, 0.35f, -0.01f)),
            ("leg_back", 0, new Vector3(-0.14f, 0.35f, 0.01f)),
            ("weapon", 3, new Vector3(0.35f, 0.6f, -0.02f)),
        };

        private static void EnsureGenerated()
        {
            if (!Directory.Exists(Path.Combine(
                    Application.dataPath, "Art", "Actors", "Players")))
            {
                return;
            }
            var changed = false;
            changed |= EnsureClips();
            var skeleton = EnsureSkeleton();
            if (skeleton != null)
            {
                changed |= BindSkeletonToPsbs(
                    AssetDatabase.AssetPathToGUID(SkeletonPath));
            }
            var controller = EnsureController();
            changed |= EnsurePrefabs(controller);
            changed |= EnsureSpriteLibrary();
            if (changed)
            {
                AssetDatabase.SaveAssets();
            }
        }

        // One shared clip set for all five classes (§3.7): the transform
        // bone paths are the fixed PSB layer names, so every rig plays
        // the same clips. Keys per clip >= 4 at 30 fps; looped clips are
        // idle, run, fall and guard.
        private static bool EnsureClips()
        {
            var changed = false;
            foreach (var spec in ClipSpecs)
            {
                var path = AnimDir + "/" + spec.Name + ".anim";
                if (AssetDatabase.LoadAssetAtPath<AnimationClip>(path) != null)
                {
                    continue;
                }
                var clip = new AnimationClip
                {
                    name = spec.Name,
                    frameRate = AnimationContract.SkeletalFps,
                };
                var settings = AnimationUtility.GetAnimationClipSettings(clip);
                settings.loopTime = spec.Loop;
                AnimationUtility.SetAnimationClipSettings(clip, settings);
                foreach (var (path2, prop, keys) in spec.Curves)
                {
                    var curve = new AnimationCurve();
                    foreach (var (t, v) in keys)
                    {
                        curve.AddKey(new Keyframe(t, v));
                    }
                    clip.SetCurve(path2, typeof(Transform), prop, curve);
                }
                Directory.CreateDirectory(
                    Path.Combine(Application.dataPath, "Art",
                        "Actors", "Players", "anim"));
                AssetDatabase.CreateAsset(clip, path);
                changed = true;
            }
            return changed;
        }

        private static (float t, float v) K(float t, float v)
        {
            return (t, v);
        }

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            IdleCurves =
        {
            ("torso", "localEulerAngles.z", new[] { K(0f, 0f), K(0.5f, 1.6f), K(1f, 0f) }),
            ("torso", "localScale.y", new[] { K(0f, 1f), K(0.5f, 1.018f), K(1f, 1f) }),
            ("torso/head", "localEulerAngles.z", new[] { K(0f, 0f), K(0.5f, -2f), K(1f, 0f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.5f, 3f), K(1f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            RunCurves =
        {
            ("torso/leg_front", "localEulerAngles.z", new[] { K(0f, 32f), K(0.3f, -32f), K(0.6f, 32f) }),
            ("torso/leg_back", "localEulerAngles.z", new[] { K(0f, -32f), K(0.3f, 32f), K(0.6f, -32f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, -24f), K(0.3f, 24f), K(0.6f, -24f) }),
            ("torso/arm_back", "localEulerAngles.z", new[] { K(0f, 24f), K(0.3f, -24f), K(0.6f, 24f) }),
            ("torso", "localPosition.y", new[] { K(0f, 0.55f), K(0.15f, 0.6f), K(0.3f, 0.55f), K(0.45f, 0.6f), K(0.6f, 0.55f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            JumpCurves =
        {
            ("", "localPosition.y", new[] { K(0f, 0f), K(0.2f, 0.55f), K(0.4f, 0.9f), K(0.5f, 1.05f) }),
            ("torso/leg_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.25f, 45f), K(0.5f, 50f) }),
            ("torso/leg_back", "localEulerAngles.z", new[] { K(0f, 0f), K(0.25f, -30f), K(0.5f, -35f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.5f, -35f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            FallCurves =
        {
            ("", "localPosition.y", new[] { K(0f, 1.05f), K(0.3f, 0.6f), K(0.6f, 0.15f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, -35f), K(0.6f, -55f) }),
            ("torso/arm_back", "localEulerAngles.z", new[] { K(0f, 25f), K(0.6f, 45f) }),
            ("torso/leg_front", "localEulerAngles.z", new[] { K(0f, 25f), K(0.6f, 15f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            LandCurves =
        {
            ("", "localPosition.y", new[] { K(0f, 0.15f), K(0.15f, 0f), K(0.4f, 0f) }),
            ("torso", "localScale.y", new[] { K(0f, 1f), K(0.15f, 0.92f), K(0.4f, 1f) }),
            ("torso/leg_front", "localEulerAngles.z", new[] { K(0f, 15f), K(0.4f, 0f) }),
            ("torso/leg_back", "localEulerAngles.z", new[] { K(0f, -15f), K(0.4f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            AttackCurves =
        {
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.18f, -95f), K(0.36f, 45f), K(0.6f, 0f) }),
            ("torso/arm_front/weapon", "localEulerAngles.z", new[] { K(0f, 0f), K(0.18f, -25f), K(0.36f, 15f), K(0.6f, 0f) }),
            ("torso", "localEulerAngles.y", new[] { K(0f, 0f), K(0.18f, -10f), K(0.36f, 8f), K(0.6f, 0f) }),
            ("torso", "localPosition.x", new[] { K(0f, 0f), K(0.36f, 0.06f), K(0.6f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            CastCurves =
        {
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.3f, -140f), K(0.55f, -150f), K(0.8f, 0f) }),
            ("torso/arm_back", "localEulerAngles.z", new[] { K(0f, 0f), K(0.3f, 140f), K(0.55f, 150f), K(0.8f, 0f) }),
            ("torso", "localEulerAngles.x", new[] { K(0f, 0f), K(0.3f, -6f), K(0.8f, 0f) }),
            ("torso/head", "localEulerAngles.x", new[] { K(0f, 0f), K(0.3f, -8f), K(0.8f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            HitCurves =
        {
            ("torso", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 9f), K(0.3f, 0f) }),
            ("", "localPosition.x", new[] { K(0f, 0f), K(0.1f, -0.12f), K(0.3f, 0f) }),
            ("torso/head", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 6f), K(0.3f, 0f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 20f), K(0.3f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            GuardCurves =
        {
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.15f, -70f), K(0.65f, -70f), K(0.8f, 0f) }),
            ("torso/arm_front/weapon", "localEulerAngles.z", new[] { K(0f, 0f), K(0.15f, -60f), K(0.65f, -60f), K(0.8f, 0f) }),
            ("torso", "localScale.x", new[] { K(0f, 1f), K(0.15f, 0.96f), K(0.65f, 0.96f), K(0.8f, 1f) }),
            ("torso", "localEulerAngles.z", new[] { K(0f, 0f), K(0.15f, -4f), K(0.65f, -4f), K(0.8f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            DefeatCurves =
        {
            ("", "localEulerAngles.z", new[] { K(0f, 0f), K(0.4f, 30f), K(0.8f, 80f), K(1f, 88f) }),
            ("", "localPosition.y", new[] { K(0f, 0f), K(0.8f, -0.05f), K(1f, -0.05f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.8f, 30f), K(1f, 35f) }),
            ("torso/head", "localEulerAngles.z", new[] { K(0f, 0f), K(0.8f, -15f), K(1f, -18f) }),
        };

        private static readonly (string Name, bool Loop,
            (string Path, string Prop, (float t, float v)[] Keys)[] Curves)[]
            ClipSpecs =
        {
            ("idle", true, IdleCurves),
            ("run", true, RunCurves),
            ("jump_up", false, JumpCurves),
            ("fall", true, FallCurves),
            ("land", false, LandCurves),
            ("attack_basic", false, AttackCurves),
            ("cast", false, CastCurves),
            ("hit", false, HitCurves),
            ("guard", true, GuardCurves),
            ("defeat", false, DefeatCurves),
        };

        private static Object? EnsureSkeleton()
        {
            var existing = AssetDatabase.LoadAssetAtPath<Object>(SkeletonPath);
            if (existing != null)
            {
                return existing;
            }
            var skel = ScriptableObject.CreateInstance<SkeletonAsset>();
            var bones = new SpriteBone[SkeletonSpec.Length];
            for (var i = 0; i < SkeletonSpec.Length; i++)
            {
                bones[i] = new SpriteBone
                {
                    name = SkeletonSpec[i].Name,
                    position = SkeletonSpec[i].Pos,
                    rotation = Quaternion.identity,
                    length = 0.4f,
                    parentId = SkeletonSpec[i].Parent,
                    color = Color.white,
                };
            }
            skel.SetSpriteBones(bones);
            AssetDatabase.CreateAsset(skel, SkeletonPath);
            return skel;
        }

        private static bool BindSkeletonToPsbs(string skeletonGuid)
        {
            var changed = false;
            foreach (var id in ClassIds)
            {
                var psbPath = GeneratedDir + "/" + id + "/asset.class." + id + ".psb";
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

        private static AnimatorController? EnsureController()
        {
            var existing =
                AssetDatabase.LoadAssetAtPath<AnimatorController>(ControllerPath);
            if (existing != null)
            {
                return existing;
            }
            var required = AnimationContract.RequiredClips(
                SpriteSizeProfile.Character, 0, 0);
            var controller = AnimatorController.CreateAnimatorControllerAtPath(
                ControllerPath);
            var sm = controller.layers[0].stateMachine;
            AnimatorState? defaultState = null;
            foreach (var clipName in required)
            {
                var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(
                    AnimDir + "/" + clipName + ".anim");
                if (clip == null)
                {
                    continue;
                }
                var state = sm.AddState(clipName);
                state.motion = clip;
                if (clipName == "idle")
                {
                    defaultState = state;
                }
            }
            if (defaultState != null)
            {
                sm.defaultState = defaultState;
            }
            EditorUtility.SetDirty(controller);
            return controller;
        }

        private static bool EnsurePrefabs(AnimatorController? controller)
        {
            var changed = false;
            foreach (var id in ClassIds)
            {
                var prefabPath =
                    GeneratedDir + "/" + id + "/asset.class." + id + ".prefab";
                if (AssetDatabase.LoadAssetAtPath<GameObject>(prefabPath) != null)
                {
                    continue;
                }
                var psbPath = GeneratedDir + "/" + id + "/asset.class." + id + ".psb";
                var psb = AssetDatabase.LoadAssetAtPath<GameObject>(psbPath);
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
                    var animator = instance.GetComponent<Animator>();
                    if (animator == null)
                    {
                        animator = instance.AddComponent<Animator>();
                    }
                    if (controller != null)
                    {
                        animator.runtimeAnimatorController = controller;
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

        private static bool EnsureSpriteLibrary()
        {
            if (AssetDatabase.LoadAssetAtPath<SpriteLibraryAsset>(SpriteLibPath)
                != null)
            {
                return false;
            }
            var lib = ScriptableObject.CreateInstance<SpriteLibraryAsset>();
            foreach (var category in Layers)
            {
                foreach (var id in ClassIds)
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

        private static Sprite? FindLayerSprite(string classId, string layerName)
        {
            var psbPath =
                GeneratedDir + "/" + classId + "/asset.class." + classId + ".psb";
            foreach (var asset in AssetDatabase.LoadAllAssetsAtPath(psbPath))
            {
                if (asset is Sprite sprite && sprite.name == layerName)
                {
                    return sprite;
                }
            }
            return null;
        }

        // The ReviewActor scene is owned by IMP-070 and carries the built-in
        // placeholder sprite. The produced art still has to appear in the
        // §3.3 visual-review artifact, so the produced class sprites are
        // mounted under ActorAnchor in memory whenever the scene opens for
        // render. The scene file on disk is never modified.
        private static void OnSceneOpened(UnityEngine.SceneManagement.Scene scene,
            OpenSceneMode mode)
        {
            if (!scene.path.EndsWith("ReviewActor.unity",
                    System.StringComparison.Ordinal))
            {
                return;
            }
            GameObject anchor = null;
            foreach (var root in scene.GetRootGameObjects())
            {
                if (root.name == "ActorAnchor")
                {
                    anchor = root;
                    break;
                }
            }
            if (anchor == null)
            {
                return;
            }
            var renderer = anchor.GetComponent<SpriteRenderer>();
            var spacing = 1.35f;
            for (var i = 0; i < ClassIds.Length; i++)
            {
                var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(
                    GeneratedDir + "/" + ClassIds[i]
                        + "/asset.class." + ClassIds[i] + ".png");
                if (sprite == null)
                {
                    continue;
                }
                if (i == (ClassIds.Length - 1) / 2 && renderer != null)
                {
                    renderer.sprite = sprite;
                    renderer.color = Color.white;
                    continue;
                }
                var child = new GameObject("ActorAnchor_" + ClassIds[i]);
                child.transform.SetParent(anchor.transform, false);
                child.transform.localPosition =
                    new Vector3((i - (ClassIds.Length - 1) * 0.5f) * spacing, 0f, -0.1f);
                var sr = child.AddComponent<SpriteRenderer>();
                sr.sprite = sprite;
                sr.sortingOrder = 1;
            }
            if (renderer != null)
            {
                renderer.enabled = true;
            }
        }
    }
}
