using System.Collections.Generic;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Deterministic 2-D k-means over the CIELAB chroma plane (a*, b*) used
    // by the top-light and frame-consistency rules of section 3.6/3.7
    // (ART-003/004). Seeds are the (a*, b*) values of the silhouette pixels
    // sitting at the L* p12.5/p37.5/p62.5/p87.5 percentile ranks, Lloyd
    // iterations run to stability or MaxIterations, and ties go to the
    // lowest cluster index — no RNG anywhere.
    public static class KMeans2D
    {
        public const int MaxIterations = 100;
        public const int HueClusterCount = 4;

        // Seeds required by section 3.6: sorted L* percentile ranks
        // 12.5/37.5/62.5/87.5 mapped to the (a*, b*) of the pixel at that
        // rank. Caller supplies pixels in scan order; the L* ordering is
        // built here deterministically (stable sort by index).
        public static Vector2[] HueClusterSeeds(Vector3[] lab, bool[] silhouette)
        {
            var idx = new List<int>();
            for (var i = 0; i < lab.Length; i++)
            {
                if (silhouette[i])
                {
                    idx.Add(i);
                }
            }
            idx.Sort((a, b) => lab[a].x.CompareTo(lab[b].x));
            var seeds = new Vector2[HueClusterCount];
            var fractions = new[] { 0.125, 0.375, 0.625, 0.875 };
            for (var c = 0; c < HueClusterCount; c++)
            {
                var i = idx.Count == 0
                    ? 0
                    : idx[(int)System.Math.Floor(fractions[c] * (idx.Count - 1))];
                seeds[c] = new Vector2(lab[i].y, lab[i].z);
            }
            return seeds;
        }

        // Lloyd over Vector2 values with the given seed centres. Returns
        // the cluster index per value; centres and counts carry the final
        // cluster centres and member counts.
        public static int[] Cluster(
            Vector2[] values,
            Vector2[] seeds,
            out Vector2[] centers,
            out int[] counts)
        {
            centers = new Vector2[seeds.Length];
            for (var c = 0; c < seeds.Length; c++)
            {
                centers[c] = seeds[c];
            }
            var k = seeds.Length;
            var assign = new int[values.Length];
            for (var i = 0; i < assign.Length; i++)
            {
                assign[i] = -1;
            }
            counts = new int[k];
            for (var iter = 0; iter < MaxIterations; iter++)
            {
                var changed = false;
                var sums = new Vector2[k];
                var newCounts = new int[k];
                for (var i = 0; i < values.Length; i++)
                {
                    var best = NearestCenter(values[i], centers);
                    if (assign[i] != best)
                    {
                        assign[i] = best;
                        changed = true;
                    }
                    sums[best] += values[i];
                    newCounts[best]++;
                }
                if (!changed)
                {
                    counts = newCounts;
                    return assign;
                }
                for (var c = 0; c < k; c++)
                {
                    if (newCounts[c] > 0)
                    {
                        centers[c] = sums[c] / newCounts[c];
                    }
                }
            }
            var finalCounts = new int[k];
            for (var i = 0; i < assign.Length; i++)
            {
                if (assign[i] < 0)
                {
                    assign[i] = NearestCenter(values[i], centers);
                }
                finalCounts[assign[i]]++;
            }
            counts = finalCounts;
            return assign;
        }

        // Area-weighted median over (value, weight) pairs: the smallest v
        // where cumulative weight reaches half the total. Deterministic
        // lower median for even totals.
        public static float WeightedMedian(float[] values, int[] weights)
        {
            var order = new List<int>(values.Length);
            var total = 0;
            for (var i = 0; i < values.Length; i++)
            {
                if (weights[i] > 0)
                {
                    order.Add(i);
                    total += weights[i];
                }
            }
            if (order.Count == 0 || total <= 0)
            {
                return 0f;
            }
            order.Sort((a, b) => values[a].CompareTo(values[b]));
            var acc = 0;
            var half = total / 2f;
            foreach (var i in order)
            {
                acc += weights[i];
                if (acc >= half)
                {
                    return values[i];
                }
            }
            return values[order[order.Count - 1]];
        }

        private static int NearestCenter(Vector2 v, Vector2[] centers)
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
