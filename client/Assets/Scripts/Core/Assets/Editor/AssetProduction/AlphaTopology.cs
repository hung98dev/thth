using System.Collections.Generic;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Shared pixel-topology helpers for the section 3.2 / 3.6 gates. All
    // functions are deterministic: distances use multi-source BFS for the
    // Chebyshev metric and every "nearest" search uses Euclidean distance
    // with row-scan (smallest y * width + x index) tie-breaking, matching the
    // definitions pinned by ADR-0071.
    public static class AlphaTopology
    {
        // Indices of pixels whose alpha lies in [minA, maxA] inclusive.
        public static bool[] AlphaMask(Color32[] pixels, int minA, int maxA)
        {
            var mask = new bool[pixels.Length];
            for (var i = 0; i < pixels.Length; i++)
            {
                var a = pixels[i].a;
                mask[i] = a >= minA && a <= maxA;
            }
            return mask;
        }

        // Chebyshev distance (king-move) from every pixel to the nearest
        // pixel OUTSIDE mask. Pixels outside mask get distance 0.
        public static int[] DistanceToOutside(bool[] mask, int width, int height)
        {
            var dist = new int[mask.Length];
            var queue = new Queue<int>(mask.Length);
            for (var i = 0; i < mask.Length; i++)
            {
                if (mask[i])
                {
                    dist[i] = -1;
                }
                else
                {
                    dist[i] = 0;
                    queue.Enqueue(i);
                }
            }
            while (queue.Count > 0)
            {
                var i = queue.Dequeue();
                var x = i % width;
                var y = i / width;
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
                        if (dist[ni] != -1)
                        {
                            continue;
                        }
                        dist[ni] = dist[i] + 1;
                        queue.Enqueue(ni);
                    }
                }
            }
            return dist;
        }

        // Chebyshev distance from a non-mask pixel to the nearest mask
        // pixel; mask pixels get 0.
        public static int[] DistanceToMask(bool[] mask, int width, int height)
        {
            return DistanceToOutside(Negate(mask), width, height);
        }

        public static bool[] Negate(bool[] mask)
        {
            var r = new bool[mask.Length];
            for (var i = 0; i < mask.Length; i++)
            {
                r[i] = !mask[i];
            }
            return r;
        }

        // 8-connected components over mask. Returns component index lists.
        public static List<int[]> Components8(bool[] mask, int width, int height)
        {
            var visited = new bool[mask.Length];
            var components = new List<int[]>();
            var stack = new Stack<int>();
            var memberList = new List<int>();
            for (var i = 0; i < mask.Length; i++)
            {
                if (!mask[i] || visited[i])
                {
                    continue;
                }
                memberList.Clear();
                stack.Push(i);
                visited[i] = true;
                while (stack.Count > 0)
                {
                    var cur = stack.Pop();
                    memberList.Add(cur);
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
                            if (!mask[ni] || visited[ni])
                            {
                                continue;
                            }
                            visited[ni] = true;
                            stack.Push(ni);
                        }
                    }
                }
                components.Add(memberList.ToArray());
            }
            return components;
        }

        // Nearest pixel where mask is true to (x, y), Euclidean distance with
        // row-scan tie-break, searched inside a Chebyshev window of
        // maxRadius. Returns -1 when no mask pixel lies within the window.
        public static int NearestMasked(
            bool[] mask,
            int width,
            int height,
            int x,
            int y,
            int maxRadius)
        {
            var best = -1;
            var bestDist2 = long.MaxValue;
            var minX = Mathf.Max(0, x - maxRadius);
            var maxX = Mathf.Min(width - 1, x + maxRadius);
            var minY = Mathf.Max(0, y - maxRadius);
            var maxY = Mathf.Min(height - 1, y + maxRadius);
            for (var ny = minY; ny <= maxY; ny++)
            {
                for (var nx = minX; nx <= maxX; nx++)
                {
                    var ni = ny * width + nx;
                    if (!mask[ni])
                    {
                        continue;
                    }
                    var dx = (long)(nx - x);
                    var dy = (long)(ny - y);
                    var d2 = dx * dx + dy * dy;
                    if (d2 < bestDist2 || (d2 == bestDist2 && ni < best))
                    {
                        bestDist2 = d2;
                        best = ni;
                    }
                }
            }
            return best;
        }

        // Bounding box of mask pixels: minX, minY, maxX, maxY (inclusive).
        // Returns false when the mask is empty.
        public static bool Bounds(
            bool[] mask,
            int width,
            int height,
            out int minX,
            out int minY,
            out int maxX,
            out int maxY)
        {
            minX = width;
            minY = height;
            maxX = -1;
            maxY = -1;
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    if (!mask[y * width + x])
                    {
                        continue;
                    }
                    if (x < minX)
                    {
                        minX = x;
                    }
                    if (x > maxX)
                    {
                        maxX = x;
                    }
                    if (y < minY)
                    {
                        minY = y;
                    }
                    if (y > maxY)
                    {
                        maxY = y;
                    }
                }
            }
            return maxX >= 0;
        }

        public static int Count(bool[] mask)
        {
            var n = 0;
            for (var i = 0; i < mask.Length; i++)
            {
                if (mask[i])
                {
                    n++;
                }
            }
            return n;
        }

        public static bool[] Union(bool[] a, bool[] b)
        {
            var r = new bool[a.Length];
            for (var i = 0; i < a.Length; i++)
            {
                r[i] = a[i] || b[i];
            }
            return r;
        }

        public static bool[] Subtract(bool[] a, bool[] b)
        {
            var r = new bool[a.Length];
            for (var i = 0; i < a.Length; i++)
            {
                r[i] = a[i] && !b[i];
            }
            return r;
        }
    }
}
