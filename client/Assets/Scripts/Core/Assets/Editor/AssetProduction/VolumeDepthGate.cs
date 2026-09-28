using System.Collections.Generic;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Volume & Depth Gate of presentation_asset_manifest.md section 3.6 with
    // the deterministic definitions pinned by ADR-0071:
    //   S   = pixels with a >= 128 minus the declared translucent mask
    //   B   = pixels of S within Chebyshev 3 px of outside-S
    //   K   = pixels of S at Chebyshev 5..8 px from outside-S
    //   K(p) = nearest K pixel to p (Euclidean, row-scan tie)
    //   flat region = 8-connected components over S with adjacent edges
    //                 where Delta E00 < 2
    // The lightness metric everywhere is L* / Delta L* of CIELAB (D65).
    public static class VolumeDepthGate
    {
        public const string RuleValueRange = "value_range";
        public const string RuleValueTiers = "value_tiers";
        public const string RuleTopLit = "top_lit";
        public const string RuleEdgeSeparation = "edge_separation";
        public const string RuleFlatRegion = "flat_region";
        public const string RuleEnvironment = "environment";
        public const string RuleActorOnBackground = "actor_on_background";
        public const string RuleTranslucentScope = "translucent_scope";

        private const int SilhouetteAlpha = 128;
        private const float ValueRangeMin = 40f;
        private const int ValueTierClusters = 5;
        private const float ValueTierMinFraction = 0.05f;
        private const int ValueTierMinCount = 3;
        private const float TopLitMinDelta = 6f;
        private const float EdgeSeparationMinDeltaL = 12f;
        private const float EdgeSeparationMinFraction = 0.60f;
        private const float FlatRegionMaxFraction = 0.20f;
        private const float FlatRegionMaxDeltaE = 2f;
        private const float EnvironmentL4Fraction = 0.5f;
        private const float ActorBackgroundMinDeltaL = 20f;
        private const float TranslucentMaxFraction = 0.60f;

        public static List<GateViolation> ValidatePixels(VolumeDepthInput input)
        {
            var violations = new List<GateViolation>();
            var pixels = input.Pixels;
            var width = input.Width;
            var height = input.Height;
            var rules = GateScope.VolumeRules(input.AssetClass);

            var silhouette = AlphaTopology.AlphaMask(pixels, SilhouetteAlpha, 255);
            if (input.TranslucentMask != null)
            {
                if (input.AssetClass != PresentationAssetClass.Actor)
                {
                    violations.Add(new GateViolation(
                        RuleTranslucentScope,
                        "translucent mask declared on non-ACTOR asset class "
                            + input.AssetClass));
                }
                else
                {
                    var sCount = AlphaTopology.Count(silhouette);
                    var mCount = AlphaTopology.Count(
                        Intersect(input.TranslucentMask, silhouette));
                    if (sCount > 0 && mCount > TranslucentMaxFraction * sCount)
                    {
                        violations.Add(new GateViolation(
                            RuleTranslucentScope,
                            "translucent mask covers " + mCount + " px > 60% of S ("
                                + sCount + " px)"));
                    }
                }
                silhouette = AlphaTopology.Subtract(silhouette, input.TranslucentMask);
            }
            var sCountFinal = AlphaTopology.Count(silhouette);
            if (sCountFinal == 0)
            {
                violations.Add(new GateViolation(RuleValueRange, "empty silhouette"));
                return violations;
            }

            var lab = new Vector3[pixels.Length];
            var sortedL = new List<float>(sCountFinal);
            for (var i = 0; i < pixels.Length; i++)
            {
                if (!silhouette[i])
                {
                    continue;
                }
                lab[i] = CieLab.ToLab(pixels[i]);
                sortedL.Add(lab[i].x);
            }
            sortedL.Sort();
            var lArr = sortedL.ToArray();

            if (rules.HasFlag(VolumeRule.ValueRange))
            {
                var range = KMeans1D.Percentile(lArr, 95.0) - KMeans1D.Percentile(lArr, 5.0);
                if (range < ValueRangeMin)
                {
                    violations.Add(new GateViolation(
                        RuleValueRange,
                        "L* range " + range + " < 40 (flat value spread)"));
                }
            }
            if (rules.HasFlag(VolumeRule.ValueTiers))
            {
                CheckValueTiers(lArr, violations);
            }
            if (rules.HasFlag(VolumeRule.TopLit))
            {
                CheckTopLit(silhouette, lab, width, height, violations);
            }
            if (rules.HasFlag(VolumeRule.EdgeSeparation))
            {
                CheckEdgeSeparation(silhouette, lab, width, height, violations);
            }
            if (rules.HasFlag(VolumeRule.FlatRegions))
            {
                CheckFlatRegions(silhouette, lab, width, height, sCountFinal, violations);
            }
            return violations;
        }

        // >= 5-seeded 1-D k-means over L* must leave >= 3 clusters each
        // holding >= 5% of S.
        private static void CheckValueTiers(float[] lArr, List<GateViolation> violations)
        {
            var seeds = new[]
            {
                KMeans1D.Percentile(lArr, 10.0),
                KMeans1D.Percentile(lArr, 30.0),
                KMeans1D.Percentile(lArr, 50.0),
                KMeans1D.Percentile(lArr, 70.0),
                KMeans1D.Percentile(lArr, 90.0),
            };
            KMeans1D.Cluster(lArr, seeds, out _, out var counts);
            var largeEnough = 0;
            var minCount = (int)System.Math.Ceiling(ValueTierMinFraction * lArr.Length);
            foreach (var c in counts)
            {
                if (c >= minCount)
                {
                    largeEnough++;
                }
            }
            if (largeEnough < ValueTierMinCount)
            {
                violations.Add(new GateViolation(
                    RuleValueTiers,
                    "only " + largeEnough + " value tiers hold >= 5% of S (need >= 3)"));
            }
        }

        // Top third of bbox(S) must be brighter than the bottom third by
        // >= 6 L* — the top-front key light of section 3.5.
        private static void CheckTopLit(
            bool[] silhouette,
            Vector3[] lab,
            int width,
            int height,
            List<GateViolation> violations)
        {
            if (!AlphaTopology.Bounds(silhouette, width, height,
                out _, out var minY, out _, out var maxY))
            {
                return;
            }
            var thirdH = (maxY - minY + 1) / 3;
            if (thirdH < 1)
            {
                return;
            }
            var topSum = 0f;
            var bottomSum = 0f;
            var topN = 0;
            var bottomN = 0;
            var topStart = maxY - thirdH + 1;
            var bottomEnd = minY + thirdH - 1;
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var i = y * width + x;
                    if (!silhouette[i])
                    {
                        continue;
                    }
                    if (y >= topStart)
                    {
                        topSum += lab[i].x;
                        topN++;
                    }
                    else if (y <= bottomEnd)
                    {
                        bottomSum += lab[i].x;
                        bottomN++;
                    }
                }
            }
            if (topN == 0 || bottomN == 0)
            {
                return;
            }
            var delta = topSum / topN - bottomSum / bottomN;
            if (delta < TopLitMinDelta)
            {
                violations.Add(new GateViolation(
                    RuleTopLit,
                    "top-third minus bottom-third mean L* = " + delta + " (need >= 6; "
                        + "bottom-lit or flat)"));
            }
        }

        // The pinned edge-band/core-ring definition of section 3.6: B = S
        // pixels within Chebyshev 3 px of outside-S; K = S pixels at
        // Chebyshev 5..8 px from outside-S.
        public static void ComputeBandAndCore(
            bool[] silhouette,
            int width,
            int height,
            out bool[] band,
            out bool[] core)
        {
            var distOut = AlphaTopology.DistanceToOutside(silhouette, width, height);
            band = new bool[silhouette.Length];
            core = new bool[silhouette.Length];
            for (var i = 0; i < silhouette.Length; i++)
            {
                if (!silhouette[i])
                {
                    continue;
                }
                if (distOut[i] <= 3)
                {
                    band[i] = true;
                }
                else if (distOut[i] >= 5 && distOut[i] <= 8)
                {
                    core[i] = true;
                }
            }
        }

        // >= 60% of edge-band pixels must differ in L* from their nearest
        // core-ring pixel by >= 12 (rim light or outline).
        private static void CheckEdgeSeparation(
            bool[] silhouette,
            Vector3[] lab,
            int width,
            int height,
            List<GateViolation> violations)
        {
            ComputeBandAndCore(silhouette, width, height, out var band, out var core);
            var bandCount = AlphaTopology.Count(band);
            if (bandCount == 0)
            {
                return;
            }
            var separated = 0;
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var i = y * width + x;
                    if (!band[i])
                    {
                        continue;
                    }
                    var k = AlphaTopology.NearestMasked(core, width, height, x, y, 12);
                    if (k < 0)
                    {
                        continue;
                    }
                    if (Mathf.Abs(lab[i].x - lab[k].x) >= EdgeSeparationMinDeltaL)
                    {
                        separated++;
                    }
                }
            }
            if (separated < bandCount * EdgeSeparationMinFraction)
            {
                violations.Add(new GateViolation(
                    RuleEdgeSeparation,
                    separated + "/" + bandCount
                        + " edge pixels differ from the core ring by >= 12 L* (need >= 60%)"));
            }
        }

        // No 8-connected flat region (adjacent pixels linked when
        // Delta E00 < 2) may cover more than 20% of S.
        private static void CheckFlatRegions(
            bool[] silhouette,
            Vector3[] lab,
            int width,
            int height,
            int sCount,
            List<GateViolation> violations)
        {
            var visited = new bool[silhouette.Length];
            var stack = new Stack<int>();
            var members = new List<int>();
            var limit = FlatRegionMaxFraction * sCount;
            for (var i = 0; i < silhouette.Length; i++)
            {
                if (!silhouette[i] || visited[i])
                {
                    continue;
                }
                members.Clear();
                stack.Push(i);
                visited[i] = true;
                while (stack.Count > 0)
                {
                    var cur = stack.Pop();
                    members.Add(cur);
                    var x = cur % width;
                    var y = cur / width;
                    for (var dy = -1; dy <= 1; dy++)
                    {
                        for (var dx = -1; dx <= 1; dx++)
                        {
                            if (dx == 0 && dy == 0)
                            {
                                continue;
                            }
                            var nx = x + dx;
                            var ny = y + dy;
                            if (nx < 0 || ny < 0 || nx >= width || ny >= height)
                            {
                                continue;
                            }
                            var ni = ny * width + nx;
                            if (!silhouette[ni] || visited[ni])
                            {
                                continue;
                            }
                            if (CieLab.DeltaE00(lab[cur], lab[ni]) >= FlatRegionMaxDeltaE)
                            {
                                continue;
                            }
                            visited[ni] = true;
                            stack.Push(ni);
                        }
                    }
                }
                if (members.Count > limit)
                {
                    violations.Add(new GateViolation(
                        RuleFlatRegion,
                        members[0] % width,
                        members[0] / width,
                        "flat region of " + members.Count + " px > 20% of S ("
                            + sCount + " px)"));
                }
            }
        }

        // Environment rule (section 3.6): measured on isolated per-layer
        // 1280x720 day review renders. layerPixels[0..3] are L1..L4
        // (gameplay -> far). Contrast (L* p95 - p5 over a >= 128) must satisfy
        // L1 >= L2 >= L3 >= L4 and L4 <= 0.5 x L1; mean C*ab must be
        // non-increasing L1 -> L4.
        public static List<GateViolation> CheckEnvironmentLayers(
            Color32[][] layerPixels,
            int width,
            int height)
        {
            var violations = new List<GateViolation>();
            if (layerPixels.Length < 4)
            {
                violations.Add(new GateViolation(
                    RuleEnvironment, "need 4 layer renders (L1..L4)"));
                return violations;
            }
            var contrast = new float[4];
            var chroma = new float[4];
            for (var i = 0; i < 4; i++)
            {
                var px = layerPixels[i];
                var lValues = new List<float>();
                var cSum = 0f;
                var n = 0;
                for (var p = 0; p < px.Length; p++)
                {
                    if (px[p].a < SilhouetteAlpha)
                    {
                        continue;
                    }
                    var lab = CieLab.ToLab(px[p]);
                    lValues.Add(lab.x);
                    cSum += CieLab.CStar(lab);
                    n++;
                }
                if (n == 0)
                {
                    violations.Add(new GateViolation(
                        RuleEnvironment, "layer L" + (i + 1) + " has no opaque pixels"));
                    continue;
                }
                lValues.Sort();
                var arr = lValues.ToArray();
                contrast[i] = KMeans1D.Percentile(arr, 95.0) - KMeans1D.Percentile(arr, 5.0);
                chroma[i] = cSum / n;
            }
            if (violations.Count > 0)
            {
                return violations;
            }
            for (var i = 1; i < 4; i++)
            {
                if (contrast[i] > contrast[i - 1] + 0.001f)
                {
                    violations.Add(new GateViolation(
                        RuleEnvironment,
                        "layer L" + (i + 1) + " contrast " + contrast[i]
                            + " exceeds L" + i + " (" + contrast[i - 1] + ")"));
                }
                if (chroma[i] > chroma[i - 1] + 0.001f)
                {
                    violations.Add(new GateViolation(
                        RuleEnvironment,
                        "layer L" + (i + 1) + " mean C*ab " + chroma[i]
                            + " exceeds L" + i + " (" + chroma[i - 1] + ")"));
                }
            }
            if (contrast[3] > EnvironmentL4Fraction * contrast[0])
            {
                violations.Add(new GateViolation(
                    RuleEnvironment,
                    "L4 contrast " + contrast[3] + " > 0.5x L1 (" + contrast[0] + ")"));
            }
            return violations;
        }

        // Actor-on-background rule: on the review render, |mean L*(B) minus
        // mean L*(annulus 4..12 px outside S)| >= 20. actorMask is the
        // actor's silhouette (a >= 128) positioned in the render.
        public static List<GateViolation> CheckActorOnBackground(
            Color32[] renderPixels,
            bool[] actorMask,
            int width,
            int height)
        {
            var violations = new List<GateViolation>();
            var distOut = AlphaTopology.DistanceToOutside(actorMask, width, height);
            var distToActor = AlphaTopology.DistanceToMask(actorMask, width, height);
            var bandL = new List<float>();
            var ringL = new List<float>();
            for (var i = 0; i < actorMask.Length; i++)
            {
                var lab = CieLab.ToLab(renderPixels[i]).x;
                if (actorMask[i] && distOut[i] <= 3)
                {
                    bandL.Add(lab);
                }
                else if (!actorMask[i] && distToActor[i] >= 4 && distToActor[i] <= 12)
                {
                    ringL.Add(lab);
                }
            }
            if (bandL.Count == 0 || ringL.Count == 0)
            {
                violations.Add(new GateViolation(
                    RuleActorOnBackground, "empty edge band or background ring"));
                return violations;
            }
            var bandMean = Mean(bandL);
            var ringMean = Mean(ringL);
            var delta = Mathf.Abs(bandMean - ringMean);
            if (delta < ActorBackgroundMinDeltaL)
            {
                violations.Add(new GateViolation(
                    RuleActorOnBackground,
                    "|mean L* band - ring| = " + delta + " (need >= 20)"));
            }
            return violations;
        }

        private static float Mean(List<float> values)
        {
            var sum = 0f;
            foreach (var v in values)
            {
                sum += v;
            }
            return values.Count > 0 ? sum / values.Count : 0f;
        }

        private static bool[] Intersect(bool[] a, bool[] b)
        {
            var r = new bool[a.Length];
            for (var i = 0; i < a.Length; i++)
            {
                r[i] = a[i] && b[i];
            }
            return r;
        }
    }
}
