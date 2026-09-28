namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Deterministic 1-D k-means used by the value-tier rule of section 3.6:
    // centres are seeded at the L* p10/p30/p50/p70/p90 percentiles of S and
    // Lloyd iterations run until the assignment is stable or 100 iterations
    // pass. There is no RNG anywhere in the path; ties go to the lowest
    // cluster index.
    public static class KMeans1D
    {
        public const int MaxIterations = 100;

        // Nearest-rank percentile over a sorted array:
        // sorted[floor(p / 100 * (n - 1))].
        public static float Percentile(float[] sorted, double p)
        {
            if (sorted.Length == 0)
            {
                return 0f;
            }
            var idx = (int)System.Math.Floor(p / 100.0 * (sorted.Length - 1));
            if (idx < 0)
            {
                idx = 0;
            }
            if (idx >= sorted.Length)
            {
                idx = sorted.Length - 1;
            }
            return sorted[idx];
        }

        // Runs Lloyd's algorithm over values with the given seed centres.
        // Returns the cluster index per value; centres and counts carry the
        // final cluster centres and member counts.
        public static int[] Cluster(
            float[] values,
            float[] seeds,
            out float[] centers,
            out int[] counts)
        {
            centers = (float[])seeds.Clone();
            var k = centers.Length;
            var assign = new int[values.Length];
            for (var i = 0; i < assign.Length; i++)
            {
                assign[i] = -1;
            }
            counts = new int[k];
            for (var iter = 0; iter < MaxIterations; iter++)
            {
                var changed = false;
                var sums = new float[k];
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

        private static int NearestCenter(float v, float[] centers)
        {
            var best = 0;
            var bestD = float.MaxValue;
            for (var c = 0; c < centers.Length; c++)
            {
                var d = System.Math.Abs(v - centers[c]);
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
