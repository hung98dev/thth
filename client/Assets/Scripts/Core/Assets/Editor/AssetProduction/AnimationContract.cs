using System.Collections.Generic;
using ThinhThan.Core.Assets;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Animation Contract of section 3.7 (ART-004): technique, required
    // clip list, minimum keys/fps, frame consistency and pivot rules per
    // size_profile. The pixel-free parts (clip lists, fps, keys, pivot)
    // validate declared metadata; frame consistency compares each
    // frame-by-frame frame against idle_0 in CIELAB.
    public static class AnimationContract
    {
        public const string RuleTechnique = "animation_technique";
        public const string RuleClip = "animation_clip";
        public const string RuleFrameFps = "animation_frame_fps";
        public const string RuleFrameConsistency = "animation_frame_consistency";
        public const string RulePivot = "animation_pivot";

        public const int SkeletalMinKeys = 4;
        public const int SkeletalFps = 30;
        public const int FrameByFrameMinFrames = 4;
        public const int FrameByFrameFps = 12;
        public const float FrameDeltaE00Max = 3f;
        public const int FrameBboxWidthDiffMax = 8;
        public const int FrameBboxWidthDiffMaxAction = 32;

        // Skeletal: CHARACTER plus every medium+ monster and boss.
        public static bool IsSkeletal(SpriteSizeProfile profile)
        {
            switch (profile)
            {
                case SpriteSizeProfile.Character:
                case SpriteSizeProfile.MonsterMedium:
                case SpriteSizeProfile.MonsterElite:
                case SpriteSizeProfile.BossLarge:
                case SpriteSizeProfile.WorldBoss:
                    return true;
                default:
                    return false;
            }
        }

        // Frame-by-frame: MONSTER_SMALL and SPIRIT_BEAST only.
        public static bool IsFrameByFrame(SpriteSizeProfile profile)
        {
            return profile == SpriteSizeProfile.MonsterSmall
                || profile == SpriteSizeProfile.SpiritBeast;
        }

        // Minimum required clip list (section 3.7). monsterSkills and
        // bossPhases come from the catalog row: attack_<n> per declared
        // skill, phase_transition per boss phase.
        public static string[] RequiredClips(
            SpriteSizeProfile profile,
            int monsterSkills,
            int bossPhases)
        {
            switch (profile)
            {
                case SpriteSizeProfile.Character:
                    return new[]
                    {
                        "idle", "run", "jump_up", "fall", "land",
                        "attack_basic", "cast", "hit", "guard", "defeat",
                    };
                case SpriteSizeProfile.MonsterSmall:
                    return new[] { "idle", "move", "attack_basic", "hit", "defeat" };
                case SpriteSizeProfile.SpiritBeast:
                    return new[] { "idle", "move", "cast" };
                case SpriteSizeProfile.MonsterMedium:
                case SpriteSizeProfile.MonsterElite:
                case SpriteSizeProfile.BossLarge:
                case SpriteSizeProfile.WorldBoss:
                    var clips = new List<string> { "idle", "move" };
                    for (var i = 1; i <= monsterSkills; i++)
                    {
                        clips.Add("attack_" + i);
                    }
                    clips.Add("hit");
                    clips.Add("defeat");
                    if (profile == SpriteSizeProfile.BossLarge
                        || profile == SpriteSizeProfile.WorldBoss)
                    {
                        for (var i = 1; i <= bossPhases; i++)
                        {
                            clips.Add("phase_transition_" + i);
                        }
                    }
                    return clips.ToArray();
                default:
                    return new string[0];
            }
        }

        // Declared-metadata validation: technique, clip coverage, minimum
        // keys/frames and fps for one actor. clips maps clip name to the
        // produced key/frame count; fps is the clip sample rate;
        // skeletal rigs must also pass PsbLayerCheck.
        public static List<GateViolation> ValidateDeclared(
            SpriteSizeProfile profile,
            string technique,
            string[] clips,
            int[] keysPerClip,
            int fps,
            int monsterSkills,
            int bossPhases)
        {
            var violations = new List<GateViolation>();
            var expected = RequiredClips(profile, monsterSkills, bossPhases);
            var skeletal = IsSkeletal(profile);
            var frameByFrame = IsFrameByFrame(profile);
            if (skeletal && technique != "skeletal")
            {
                violations.Add(new GateViolation(
                    RuleTechnique,
                    "size_profile " + profile + " requires skeletal animation, found '"
                        + technique + "'"));
            }
            else if (frameByFrame && technique != "frame_by_frame")
            {
                violations.Add(new GateViolation(
                    RuleTechnique,
                    "size_profile " + profile + " requires frame-by-frame, found '"
                        + technique + "'"));
            }
            var clipSet = new HashSet<string>(clips ?? new string[0]);
            foreach (var c in expected)
            {
                if (!clipSet.Contains(c))
                {
                    violations.Add(new GateViolation(
                        RuleClip,
                        "required clip '" + c + "' missing for " + profile));
                }
            }
            if (skeletal && fps != SkeletalFps)
            {
                violations.Add(new GateViolation(
                    RuleFrameFps,
                    "skeletal clips sample at " + fps + " fps (need " + SkeletalFps + ")"));
            }
            if (frameByFrame && fps != FrameByFrameFps)
            {
                violations.Add(new GateViolation(
                    RuleFrameFps,
                    "frame-by-frame clips at " + fps + " fps (need " + FrameByFrameFps + ")"));
            }
            if (clips != null && keysPerClip != null)
            {
                var minKeys = skeletal ? SkeletalMinKeys : FrameByFrameMinFrames;
                for (var i = 0; i < clips.Length && i < keysPerClip.Length; i++)
                {
                    if (keysPerClip[i] < minKeys)
                    {
                        violations.Add(new GateViolation(
                            RuleFrameFps,
                            "clip '" + clips[i] + "' has " + keysPerClip[i]
                                + (skeletal ? " keys" : " frames")
                                + " (need >= " + minKeys + ")"));
                    }
                }
            }
            return violations;
        }

        // Fixed PSB layer names for every skeletal rig (Sprite Library
        // maps cosmetics onto them). accessory_* names are optional.
        public static readonly string[] PsbLayers =
        {
            "head", "hair", "torso", "arm_front", "arm_back",
            "leg_front", "leg_back", "weapon",
        };

        public static List<GateViolation> CheckPsbLayers(string[] layers)
        {
            var violations = new List<GateViolation>();
            var have = new HashSet<string>(layers ?? new string[0]);
            foreach (var l in PsbLayers)
            {
                if (!have.Contains(l))
                {
                    violations.Add(new GateViolation(
                        RuleTechnique, "PSB layer '" + l + "' missing"));
                }
            }
            return violations;
        }

        // Pivot Bottom Center is constant over every frame and clip; feet
        // touch y = 0 in the idle/run/land clips. Pivot is the declared
        // import pivot (0.5, 0).
        public static List<GateViolation> CheckPivot(Vector2 pivot)
        {
            var violations = new List<GateViolation>();
            if (Mathf.Abs(pivot.x - 0.5f) > 0.0001f || Mathf.Abs(pivot.y) > 0.0001f)
            {
                violations.Add(new GateViolation(
                    RulePivot,
                    "pivot (" + pivot.x + ", " + pivot.y + ") != Bottom Center (0.5, 0)"));
            }
            return violations;
        }

        // Frame consistency (section 3.7): every frame-by-frame frame f
        // versus idle_0 — each hue cluster (KMeans2D on idle_0, nearest
        // centre for f) must keep |mean Lab(f) - mean Lab(idle_0)|
        // Delta E00 <= 3 and the silhouette bbox width must stay within 8
        // tex px of idle_0 (32 for attack/hit/defeat clips). clipName is
        // the frame's owning clip.
        public static List<GateViolation> CheckFrameConsistency(
            Color32[] idle0,
            Color32[] frame,
            int width,
            int height,
            string clipName)
        {
            var violations = new List<GateViolation>();
            var mask0 = AlphaTopology.AlphaMask(idle0, 128, 255);
            var maskF = AlphaTopology.AlphaMask(frame, 128, 255);
            if (AlphaTopology.Count(mask0) == 0)
            {
                return violations;
            }
            var lab0 = new Vector3[idle0.Length];
            var labF = new Vector3[frame.Length];
            var values0 = new List<Vector2>();
            var index0 = new List<int>();
            for (var i = 0; i < idle0.Length; i++)
            {
                lab0[i] = CieLab.ToLab(idle0[i]);
                labF[i] = CieLab.ToLab(frame[i]);
                if (mask0[i])
                {
                    values0.Add(new Vector2(lab0[i].y, lab0[i].z));
                    index0.Add(i);
                }
            }
            var seeds = KMeans2D.HueClusterSeeds(lab0, mask0);
            var centers = new Vector2[seeds.Length];
            for (var c = 0; c < seeds.Length; c++)
            {
                centers[c] = seeds[c];
            }
            // Fix the assignment on idle_0 once, then reuse its centres.
            var assign0 = KMeans2D.Cluster(
                values0.ToArray(), seeds, out var fitted, out _);
            centers = fitted;
            var k = seeds.Length;
            var sum0 = new Vector3[k];
            var sumF = new Vector3[k];
            var n0 = new int[k];
            var nF = new int[k];
            for (var v = 0; v < index0.Count; v++)
            {
                var i = index0[v];
                var c = assign0[v];
                sum0[c] += lab0[i];
                n0[c]++;
                if (maskF[i])
                {
                    sumF[c] += labF[i];
                    nF[c]++;
                }
            }
            // f's silhouette pixels outside idle_0's shape join the nearest
            // idle_0 hue cluster.
            for (var i = 0; i < frame.Length; i++)
            {
                if (!maskF[i] || mask0[i])
                {
                    continue;
                }
                var c = Nearest(new Vector2(labF[i].y, labF[i].z), centers);
                sumF[c] += labF[i];
                nF[c]++;
            }
            for (var c = 0; c < k; c++)
            {
                if (n0[c] == 0 || nF[c] == 0)
                {
                    continue;
                }
                var de = CieLab.DeltaE00(sum0[c] / n0[c], sumF[c] / nF[c]);
                if (de > FrameDeltaE00Max)
                {
                    violations.Add(new GateViolation(
                        RuleFrameConsistency,
                        "clip '" + clipName + "' hue cluster " + c
                            + " mean Lab drift DeltaE00 " + de + " > 3 vs idle_0"));
                }
            }
            AlphaTopology.Bounds(mask0, width, height,
                out var minX0, out _, out var maxX0, out _);
            var width0 = maxX0 - minX0 + 1;
            var action = clipName.StartsWith("attack")
                || clipName.StartsWith("hit")
                || clipName.StartsWith("defeat");
            var limit = action ? FrameBboxWidthDiffMaxAction : FrameBboxWidthDiffMax;
            if (AlphaTopology.Bounds(maskF, width, height,
                out var minXF, out _, out var maxXF, out _))
            {
                var widthF = maxXF - minXF + 1;
                var diff = System.Math.Abs(widthF - width0);
                if (diff > limit)
                {
                    violations.Add(new GateViolation(
                        RuleFrameConsistency,
                        "clip '" + clipName + "' bbox width diff " + diff
                            + " tex px > " + limit + " vs idle_0"));
                }
            }
            return violations;
        }

        private static int Nearest(Vector2 v, Vector2[] centers)
        {
            var best = 0;
            var bestD = float.MaxValue;
            for (var c = 0; c < centers.Length; c++)
            {
                var d = (v - centers[c]).sqrMagnitude;
                if (d < bestD)
                {
                    bestD = d;
                    best = c;
                }
            }
            return best;
        }
    }
}
