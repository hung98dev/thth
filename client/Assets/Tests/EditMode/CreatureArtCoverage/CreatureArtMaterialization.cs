using System.Collections.Generic;
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

namespace ThinhThan.Tests.EditMode.CreatureArtCoverage
{
    // Materializes the rig assets the spec cannot express as hand-edited
    // YAML: one shared skeleton per skeletal size_profile, per-entity
    // AnimatorControllers holding the §3.7 clip set, one prefab per roster
    // entity (imported .psb rig or SpriteRenderer frame rig + Animator;
    // shared variants as tinted prefab variants) and one Sprite Library
    // whose categories are the fixed PSB layer names. Idempotent — files
    // that already exist are left alone, so the §4b materialization
    // artifact is stable once committed.
    public static class CreatureArtMaterialization
    {
        public const string CreaturesRoot = "Assets/Art/Actors/Creatures";
        private const string AnimRoot = CreaturesRoot + "/anim";
        private const string SpriteLibPath = AnimRoot + "/creatures.spriteLib.asset";
        private const string MapPath =
            CreaturesRoot + "/creature_presentation_map.json";

        private static readonly string[] Layers = AnimationContract.PsbLayers;

        [InitializeOnLoadMethod]
        private static void Register()
        {
            EditorApplication.delayCall += EnsureGenerated;
            EditorSceneManager.sceneOpened += OnSceneOpened;
        }

        // ---- map model (JsonUtility) ----

        [System.Serializable]
        public class MapFolklore
        {
            public string[] source_tales = new string[0];
            public string regional_variants = "";
            public string[] motifs_checked = new string[0];
        }

        [System.Serializable]
        public class MapEntry
        {
            public string entity_id = "";
            public string kind = "";
            public string rank = "";
            public string element = "";
            public string zone = "";
            public string size_profile = "";
            public string combat = "";
            public int attack_clips;
            public int phases;
            public long seed;
            public string dir = "";
            public string presentation = "";
            public string target = "";
            public string tint = "";
            public string species = "";
            public MapFolklore folklore = new MapFolklore();
        }

        [System.Serializable]
        public class MapSharedTarget
        {
            public string name = "";
            public string base_entity = "";
            public string dir = "";
            public string size_profile = "";
            public string technique = "";
        }

        [System.Serializable]
        public class MapRoot
        {
            public int schema_version;
            public string fragment = "";
            public string style_pack_id = "";
            public MapSharedTarget[] shared_targets = new MapSharedTarget[0];
            public MapEntry[] entries = new MapEntry[0];
        }

        private static MapRoot? _map;

        public static MapRoot LoadMap()
        {
            if (_map != null)
            {
                return _map;
            }
            var abs = Path.Combine(Application.dataPath, "..", MapPath);
            _map = JsonUtility.FromJson<MapRoot>(File.ReadAllText(abs));
            return _map!;
        }

        public static string PngName(MapEntry e)
        {
            return "asset." + e.entity_id + ".png";
        }

        public static string PsbName(MapEntry e)
        {
            return "asset." + e.entity_id + ".psb";
        }

        public static string PrefabRel(MapEntry e)
        {
            return CreaturesRoot + "/" + e.dir + "/asset." + e.entity_id + ".prefab";
        }

        public static string AnimDirFor(string dir)
        {
            return AnimRoot + "/" + dir;
        }

        public static string SharedTargetDir(MapEntry e)
        {
            foreach (var st in LoadMap().shared_targets)
            {
                if (e.target == "shared:" + st.name)
                {
                    return st.dir;
                }
            }
            return "";
        }

        public static string SharedPngPath(MapSharedTarget st)
        {
            return CreaturesRoot + "/" + st.dir + "/asset." + st.base_entity
                + ".png";
        }

        public static string SharedPsbPath(MapSharedTarget st)
        {
            return CreaturesRoot + "/" + st.dir + "/asset." + st.base_entity
                + ".psb";
        }

        public static string SharedPrefabPath(MapSharedTarget st)
        {
            return CreaturesRoot + "/" + st.dir + "/asset." + st.base_entity
                + ".prefab";
        }

        public static bool IsSkeletal(MapEntry e)
        {
            return e.size_profile == "MONSTER_MEDIUM"
                || e.size_profile == "MONSTER_ELITE"
                || e.size_profile == "BOSS_LARGE"
                || e.size_profile == "WORLD_BOSS";
        }

        // Canonical bone layout as fractions of the 96x128 reference cell,
        // scaled per size_profile (pivot Bottom Center, feet at y = 0).
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

        private static float ProfileScale(string sizeProfile)
        {
            // Bone layout is authored for the 128 ref px tall MEDIUM cell;
            // taller cells scale it up.
            switch (sizeProfile)
            {
                case "MONSTER_ELITE": return 192f / 128f;
                case "BOSS_LARGE": return 256f / 128f;
                case "WORLD_BOSS": return 320f / 128f;
                default: return 1f;
            }
        }

        private static string SkeletonPath(string sizeProfile)
        {
            return AnimRoot + "/skeleton." + sizeProfile.ToLowerInvariant()
                + ".asset";
        }

        private static void EnsureGenerated()
        {
            var root = Path.Combine(Application.dataPath, "Art", "Actors", "Creatures");
            if (!Directory.Exists(root))
            {
                return;
            }
            var map = LoadMap();
            var changed = false;
            changed |= EnsureClips(map);
            EnsureSkeletons(map);
            var controllers = EnsureControllers(map);
            changed |= EnsurePrefabs(map, controllers);
            changed |= EnsureSpriteLibrary(map);
            if (changed)
            {
                AssetDatabase.SaveAssets();
            }
        }

        // ---- clips ----

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
            MoveCurves =
        {
            ("torso/leg_front", "localEulerAngles.z", new[] { K(0f, 32f), K(0.3f, -32f), K(0.6f, 32f) }),
            ("torso/leg_back", "localEulerAngles.z", new[] { K(0f, -32f), K(0.3f, 32f), K(0.6f, -32f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, -24f), K(0.3f, 24f), K(0.6f, -24f) }),
            ("torso/arm_back", "localEulerAngles.z", new[] { K(0f, 24f), K(0.3f, -24f), K(0.6f, 24f) }),
            ("torso", "localPosition.y", new[] { K(0f, 0.55f), K(0.15f, 0.6f), K(0.3f, 0.55f), K(0.45f, 0.6f), K(0.6f, 0.55f) }),
        };

        private static (string Path, string Prop, (float t, float v)[] Keys)[]
            AttackCurves(float strength)
        {
            return new[]
            {
                ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.18f, -95f * strength), K(0.36f, 45f * strength), K(0.6f, 0f) }),
                ("torso/arm_front/weapon", "localEulerAngles.z", new[] { K(0f, 0f), K(0.18f, -25f * strength), K(0.36f, 15f * strength), K(0.6f, 0f) }),
                ("torso", "localEulerAngles.y", new[] { K(0f, 0f), K(0.18f, -10f * strength), K(0.36f, 8f * strength), K(0.6f, 0f) }),
                ("torso", "localPosition.x", new[] { K(0f, 0f), K(0.36f, 0.06f * strength), K(0.6f, 0f) }),
            };
        }

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            HitCurves =
        {
            ("torso", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 9f), K(0.3f, 0f) }),
            ("", "localPosition.x", new[] { K(0f, 0f), K(0.1f, -0.12f), K(0.3f, 0f) }),
            ("torso/head", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 6f), K(0.3f, 0f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.1f, 20f), K(0.3f, 0f) }),
        };

        private static readonly (string Path, string Prop, (float t, float v)[] Keys)[]
            DefeatCurves =
        {
            ("", "localEulerAngles.z", new[] { K(0f, 0f), K(0.4f, 30f), K(0.8f, 80f), K(1f, 88f) }),
            ("", "localPosition.y", new[] { K(0f, 0f), K(0.8f, -0.05f), K(1f, -0.05f) }),
            ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.8f, 30f), K(1f, 35f) }),
            ("torso/head", "localEulerAngles.z", new[] { K(0f, 0f), K(0.8f, -15f), K(1f, -18f) }),
        };

        private static (string Path, string Prop, (float t, float v)[] Keys)[]
            PhaseCurves(int phase)
        {
            var sign = phase % 2 == 0 ? -1f : 1f;
            return new[]
            {
                ("", "localScale.x", new[] { K(0f, 1f), K(0.25f, 1.12f), K(0.55f, 1.04f), K(0.9f, 1f) }),
                ("", "localScale.y", new[] { K(0f, 1f), K(0.25f, 0.9f), K(0.55f, 0.97f), K(0.9f, 1f) }),
                ("torso", "localEulerAngles.z", new[] { K(0f, 0f), K(0.3f, 6f * sign), K(0.7f, -3f * sign), K(1f, 0f) }),
                ("torso/arm_front", "localEulerAngles.z", new[] { K(0f, 0f), K(0.3f, -40f * sign), K(0.7f, 20f * sign), K(1f, 0f) }),
            };
        }

        private static (string Path, string Prop, (float t, float v)[] Keys)[]
            CurvesFor(string clipName, int attackIndex, int phaseIndex)
        {
            switch (clipName)
            {
                case "idle": return IdleCurves;
                case "move": return MoveCurves;
                case "hit": return HitCurves;
                case "defeat": return DefeatCurves;
            }
            if (clipName.StartsWith("attack_", System.StringComparison.Ordinal))
            {
                return AttackCurves(0.85f + 0.15f * attackIndex);
            }
            if (clipName.StartsWith("phase_transition_",
                    System.StringComparison.Ordinal))
            {
                return PhaseCurves(phaseIndex);
            }
            return IdleCurves;
        }

        private static bool ClipLoops(string clipName)
        {
            return clipName == "idle" || clipName == "move";
        }

        private static bool EnsureClips(MapRoot map)
        {
            var changed = false;
            foreach (var e in map.entries)
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                if (IsSkeletal(e))
                {
                    changed |= EnsureSkeletalClips(e);
                }
                else
                {
                    changed |= EnsureFrameClips(e);
                }
            }
            return changed;
        }

        public static string[] RequiredClipNames(MapEntry e)
        {
            var profile = ProfileOf(e.size_profile);
            return AnimationContract.RequiredClips(
                profile, e.attack_clips, e.phases);
        }

        public static SpriteSizeProfile ProfileOf(string name)
        {
            switch (name)
            {
                case "MONSTER_SMALL": return SpriteSizeProfile.MonsterSmall;
                case "MONSTER_MEDIUM": return SpriteSizeProfile.MonsterMedium;
                case "MONSTER_ELITE": return SpriteSizeProfile.MonsterElite;
                case "BOSS_LARGE": return SpriteSizeProfile.BossLarge;
                case "WORLD_BOSS": return SpriteSizeProfile.WorldBoss;
                case "SPIRIT_BEAST": return SpriteSizeProfile.SpiritBeast;
                default: return SpriteSizeProfile.Character;
            }
        }

        private static bool EnsureSkeletalClips(MapEntry e)
        {
            var changed = false;
            var dir = AnimDirFor(e.dir);
            foreach (var clipName in RequiredClipNames(e))
            {
                var path = dir + "/" + clipName + ".anim";
                if (AssetDatabase.LoadAssetAtPath<AnimationClip>(path) != null)
                {
                    continue;
                }
                var attackIndex = AttackIndex(clipName);
                var phaseIndex = PhaseIndex(clipName);
                var clip = new AnimationClip
                {
                    name = clipName,
                    frameRate = AnimationContract.SkeletalFps,
                };
                var settings = AnimationUtility.GetAnimationClipSettings(clip);
                settings.loopTime = ClipLoops(clipName);
                AnimationUtility.SetAnimationClipSettings(clip, settings);
                foreach (var (path2, prop, keys) in CurvesFor(clipName, attackIndex, phaseIndex))
                {
                    var curve = new AnimationCurve();
                    foreach (var (t, v) in keys)
                    {
                        curve.AddKey(new Keyframe(t, v));
                    }
                    clip.SetCurve(path2, typeof(Transform), prop, curve);
                }
                Directory.CreateDirectory(Path.Combine(
                    Application.dataPath, "..", dir));
                AssetDatabase.CreateAsset(clip, path);
                changed = true;
            }
            return changed;
        }

        private static bool EnsureFrameClips(MapEntry e)
        {
            var changed = false;
            var dir = AnimDirFor(e.dir);
            foreach (var clipName in RequiredClipNames(e))
            {
                var path = dir + "/" + clipName + ".anim";
                if (AssetDatabase.LoadAssetAtPath<AnimationClip>(path) != null)
                {
                    continue;
                }
                var frames = LoadFrameSprites(e, clipName);
                if (frames.Length < AnimationContract.FrameByFrameMinFrames)
                {
                    continue;
                }
                var clip = new AnimationClip
                {
                    name = clipName,
                    frameRate = AnimationContract.FrameByFrameFps,
                };
                var settings = AnimationUtility.GetAnimationClipSettings(clip);
                settings.loopTime = ClipLoops(clipName);
                AnimationUtility.SetAnimationClipSettings(clip, settings);
                var keys = new ObjectReferenceKeyframe[frames.Length];
                var dt = 1f / AnimationContract.FrameByFrameFps;
                for (var i = 0; i < frames.Length; i++)
                {
                    keys[i] = new ObjectReferenceKeyframe
                    {
                        time = i * dt,
                        value = frames[i],
                    };
                }
                var binding = EditorCurveBinding.PPtrCurve("", typeof(SpriteRenderer), "m_Sprite");
                AnimationUtility.SetObjectReferenceCurve(clip, binding, keys);
                Directory.CreateDirectory(Path.Combine(
                    Application.dataPath, "..", dir));
                AssetDatabase.CreateAsset(clip, path);
                changed = true;
            }
            return changed;
        }

        private static Sprite[] LoadFrameSprites(MapEntry e, string clipName)
        {
            var sprites = new List<Sprite>();
            var dir = CreaturesRoot + "/" + e.dir;
            var prefix = "asset." + e.entity_id + ".anim." + clipName + ".";
            var guids = AssetDatabase.FindAssets("t:Sprite " + prefix,
                new[] { dir });
            var paths = new List<string>();
            foreach (var g in guids)
            {
                var p = AssetDatabase.GUIDToAssetPath(g);
                if (Path.GetFileNameWithoutExtension(p).StartsWith(prefix,
                        System.StringComparison.Ordinal))
                {
                    paths.Add(p);
                }
            }
            paths.Sort(System.StringComparer.Ordinal);
            foreach (var p in paths)
            {
                var s = AssetDatabase.LoadAssetAtPath<Sprite>(p);
                if (s != null)
                {
                    sprites.Add(s);
                }
            }
            return sprites.ToArray();
        }

        private static int AttackIndex(string clipName)
        {
            if (!clipName.StartsWith("attack_", System.StringComparison.Ordinal))
            {
                return 1;
            }
            return int.TryParse(clipName.Substring(7), out var n) ? n : 1;
        }

        private static int PhaseIndex(string clipName)
        {
            if (!clipName.StartsWith("phase_transition_",
                    System.StringComparison.Ordinal))
            {
                return 1;
            }
            return int.TryParse(clipName.Substring(17), out var n) ? n : 1;
        }

        // ---- skeletons ----

        private static SpriteBone[] BuildBones(string sizeProfile)
        {
            var scale = ProfileScale(sizeProfile);
            var bones = new SpriteBone[SkeletonSpec.Length];
            for (var i = 0; i < SkeletonSpec.Length; i++)
            {
                bones[i] = new SpriteBone
                {
                    name = SkeletonSpec[i].Name,
                    position = SkeletonSpec[i].Pos * scale,
                    rotation = Quaternion.identity,
                    length = 0.4f * scale,
                    parentId = SkeletonSpec[i].Parent,
                    color = Color.white,
                    // The importer re-generates an empty bone guid on every
                    // reimport (drift); a deterministic one is preserved.
                    guid = BoneGuid(sizeProfile, i, SkeletonSpec[i].Name),
                };
            }
            return bones;
        }

        private static string BoneGuid(string sizeProfile, int index, string name)
        {
            using (var md5 = System.Security.Cryptography.MD5.Create())
            {
                return new System.Guid(md5.ComputeHash(
                    System.Text.Encoding.UTF8.GetBytes(
                        "imp104.creatures.bone." + sizeProfile + "." + index
                        + "." + name))).ToString("N");
            }
        }

        private static void EnsureSkeletons(MapRoot map)
        {
            var profiles = new HashSet<string>();
            foreach (var e in map.entries)
            {
                if (e.presentation == "asset" && IsSkeletal(e))
                {
                    profiles.Add(e.size_profile);
                }
            }
            foreach (var st in map.shared_targets)
            {
                if (st.technique == "skeletal")
                {
                    profiles.Add(st.size_profile);
                }
            }
            foreach (var profile in profiles)
            {
                var path = SkeletonPath(profile);
                var existing =
                    AssetDatabase.LoadAssetAtPath<SkeletonAsset>(path);
                var spec = BuildBones(profile);
                if (existing != null)
                {
                    var cur = existing.GetSpriteBones();
                    if (cur == null || cur.Length != spec.Length
                        || string.IsNullOrEmpty(cur[0].guid))
                    {
                        existing.SetSpriteBones(spec);
                        EditorUtility.SetDirty(existing);
                        AssetDatabase.SaveAssetIfDirty(existing);
                    }
                    continue;
                }
                Directory.CreateDirectory(Path.Combine(
                    Application.dataPath, "..", AnimRoot));
                var skel = ScriptableObject.CreateInstance<SkeletonAsset>();
                skel.SetSpriteBones(spec);
                AssetDatabase.CreateAsset(skel, path);
            }
            // Bind every produced PSB importer to its profile skeleton.
            foreach (var e in map.entries)
            {
                if (e.presentation != "asset" || !IsSkeletal(e))
                {
                    continue;
                }
                BindSkeleton(CreaturesRoot + "/" + e.dir + "/"
                    + PsbName(e), e.size_profile);
            }
        }

        private static void BindSkeleton(string psbPath, string sizeProfile)
        {
            var skeletonGuid = AssetDatabase.AssetPathToGUID(
                SkeletonPath(sizeProfile));
            if (skeletonGuid.Length == 0)
            {
                return;
            }
            var importer = AssetImporter.GetAtPath(psbPath);
            if (importer == null)
            {
                return;
            }
            var so = new SerializedObject(importer);
            var prop = so.FindProperty("m_SkeletonAssetReferenceID");
            if (prop == null || prop.stringValue == skeletonGuid)
            {
                return;
            }
            prop.stringValue = skeletonGuid;
            so.ApplyModifiedPropertiesWithoutUndo();
            importer.SaveAndReimport();
        }

        // ---- controllers ----

        private static Dictionary<string, AnimatorController> EnsureControllers(
            MapRoot map)
        {
            var controllers = new Dictionary<string, AnimatorController>();
            foreach (var e in map.entries)
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var path = AnimDirFor(e.dir) + "/" + e.entity_id + ".controller";
                var controller =
                    AssetDatabase.LoadAssetAtPath<AnimatorController>(path);
                if (controller == null)
                {
                    Directory.CreateDirectory(Path.Combine(
                        Application.dataPath, "..", AnimDirFor(e.dir)));
                    controller = AnimatorController.CreateAnimatorControllerAtPath(path);
                    var sm = controller.layers[0].stateMachine;
                    AnimatorState? defaultState = null;
                    foreach (var clipName in RequiredClipNames(e))
                    {
                        var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(
                            AnimDirFor(e.dir) + "/" + clipName + ".anim");
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
                }
                controllers[e.entity_id] = controller;
            }
            return controllers;
        }

        // ---- prefabs ----

        private static bool EnsurePrefabs(
            MapRoot map,
            Dictionary<string, AnimatorController> controllers)
        {
            var changed = false;
            // Asset entities first — shared variants are prefab variants of
            // their family base's prefab and need it to exist already.
            foreach (var e in map.entries)
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var prefabPath = PrefabRel(e);
                if (AssetDatabase.LoadAssetAtPath<GameObject>(prefabPath)
                    != null)
                {
                    continue;
                }
                if (IsSkeletal(e))
                {
                    changed |= EnsureRigPrefab(e, controllers, prefabPath);
                }
                else
                {
                    changed |= EnsureFramePrefab(e, controllers, prefabPath);
                }
            }
            foreach (var e in map.entries)
            {
                if (e.presentation != "shared")
                {
                    continue;
                }
                var prefabPath = PrefabRel(e);
                if (AssetDatabase.LoadAssetAtPath<GameObject>(prefabPath)
                    != null)
                {
                    continue;
                }
                changed |= EnsureSharedPrefab(map, e, prefabPath);
            }
            return changed;
        }

        private static GameObject? InstantiateBase(
            GameObject src, AnimatorController? controller)
        {
            var instance = PrefabUtility.InstantiatePrefab(src) as GameObject;
            if (instance == null)
            {
                instance = Object.Instantiate(src);
            }
            if (instance == null)
            {
                return null;
            }
            var animator = instance.GetComponent<Animator>();
            if (animator == null)
            {
                animator = instance.AddComponent<Animator>();
            }
            if (controller != null)
            {
                animator.runtimeAnimatorController = controller;
            }
            return instance;
        }

        private static bool EnsureRigPrefab(
            MapEntry e,
            Dictionary<string, AnimatorController> controllers,
            string prefabPath)
        {
            var psb = AssetDatabase.LoadAssetAtPath<GameObject>(
                CreaturesRoot + "/" + e.dir + "/" + PsbName(e));
            if (psb == null)
            {
                return false;
            }
            controllers.TryGetValue(e.entity_id, out var controller);
            var instance = InstantiateBase(psb, controller);
            if (instance == null)
            {
                return false;
            }
            try
            {
                Directory.CreateDirectory(Path.GetDirectoryName(
                    Path.Combine(Application.dataPath, "..", prefabPath))!);
                PrefabUtility.SaveAsPrefabAsset(instance, prefabPath);
                return true;
            }
            finally
            {
                Object.DestroyImmediate(instance);
            }
        }

        private static bool EnsureFramePrefab(
            MapEntry e,
            Dictionary<string, AnimatorController> controllers,
            string prefabPath)
        {
            var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(
                CreaturesRoot + "/" + e.dir + "/" + PngName(e));
            if (sprite == null)
            {
                return false;
            }
            var go = new GameObject(e.entity_id);
            try
            {
                var sr = go.AddComponent<SpriteRenderer>();
                sr.sprite = sprite;
                var animator = go.AddComponent<Animator>();
                if (controllers.TryGetValue(e.entity_id, out var controller)
                    && controller != null)
                {
                    animator.runtimeAnimatorController = controller;
                }
                Directory.CreateDirectory(Path.GetDirectoryName(
                    Path.Combine(Application.dataPath, "..", prefabPath))!);
                PrefabUtility.SaveAsPrefabAsset(go, prefabPath);
                return true;
            }
            finally
            {
                Object.DestroyImmediate(go);
            }
        }

        private static bool EnsureSharedPrefab(
            MapRoot map, MapEntry e, string prefabPath)
        {
            var targetDir = SharedTargetDir(e);
            if (targetDir.Length == 0)
            {
                return false;
            }
            MapSharedTarget? target = null;
            foreach (var st in map.shared_targets)
            {
                if (e.target == "shared:" + st.name)
                {
                    target = st;
                    break;
                }
            }
            if (target == null)
            {
                return false;
            }
            var basePath = SharedPrefabPath(target);
            var basePrefab =
                AssetDatabase.LoadAssetAtPath<GameObject>(basePath);
            if (basePrefab == null)
            {
                return false;
            }
            var instance = PrefabUtility.InstantiatePrefab(basePrefab)
                as GameObject;
            if (instance == null)
            {
                instance = Object.Instantiate(basePrefab);
            }
            if (instance == null)
            {
                return false;
            }
            try
            {
                var tint = ParseTint(e.tint);
                foreach (var sr in instance.GetComponentsInChildren<SpriteRenderer>(true))
                {
                    sr.color = tint;
                }
                Directory.CreateDirectory(Path.GetDirectoryName(
                    Path.Combine(Application.dataPath, "..", prefabPath))!);
                PrefabUtility.SaveAsPrefabAsset(instance, prefabPath);
                return true;
            }
            finally
            {
                Object.DestroyImmediate(instance);
            }
        }

        private static Color ParseTint(string hex)
        {
            if (ColorUtility.TryParseHtmlString(
                    hex != null && hex.Length > 0 ? hex : "#FFFFFF",
                    out var c))
            {
                return c;
            }
            return Color.white;
        }

        // ---- sprite library ----

        private static bool EnsureSpriteLibrary(MapRoot map)
        {
            if (AssetDatabase.LoadAssetAtPath<SpriteLibraryAsset>(SpriteLibPath)
                != null)
            {
                return false;
            }
            var lib = ScriptableObject.CreateInstance<SpriteLibraryAsset>();
            foreach (var category in Layers)
            {
                foreach (var e in map.entries)
                {
                    if (e.presentation != "asset" || !IsSkeletal(e))
                    {
                        continue;
                    }
                    var sprite = FindLayerSprite(
                        CreaturesRoot + "/" + e.dir + "/" + PsbName(e),
                        category);
                    if (sprite != null)
                    {
                        lib.AddCategoryLabel(sprite, category, e.entity_id);
                    }
                }
            }
            Directory.CreateDirectory(Path.Combine(
                Application.dataPath, "..", AnimRoot));
            AssetDatabase.CreateAsset(lib, SpriteLibPath);
            return true;
        }

        private static Sprite? FindLayerSprite(string psbPath, string layerName)
        {
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
        // §3.3 visual-review artifact, so a representative set of produced
        // creature sprites is mounted under ActorAnchor in memory whenever
        // the scene opens for render. The scene file on disk is never
        // modified.
        private static void OnSceneOpened(UnityEngine.SceneManagement.Scene scene,
            OpenSceneMode mode)
        {
            if (!scene.path.EndsWith("ReviewActor.unity",
                    System.StringComparison.Ordinal))
            {
                return;
            }
            GameObject? anchor = null;
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
            var shown = 0;
            const int columns = 7;
            const float spacing = 1.5f;
            const float rowStride = 2.0f;
            foreach (var e in LoadMap().entries)
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(
                    CreaturesRoot + "/" + e.dir + "/" + PngName(e));
                if (sprite == null)
                {
                    continue;
                }
                if (shown == 0 && renderer != null)
                {
                    renderer.sprite = sprite;
                    renderer.color = Color.white;
                    shown++;
                    continue;
                }
                var row = shown / columns;
                var col = shown % columns;
                var child = new GameObject("Creature_" + e.entity_id);
                child.transform.SetParent(anchor.transform, false);
                child.transform.localPosition = new Vector3(
                    (col - (columns - 1) * 0.5f) * spacing,
                    -row * rowStride,
                    -0.1f);
                var sr = child.AddComponent<SpriteRenderer>();
                sr.sprite = sprite;
                sr.sortingOrder = 1;
                shown++;
            }
            if (renderer != null)
            {
                renderer.enabled = true;
            }
        }
    }
}
