using System.Collections.Generic;
using System.IO;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Cutout Quality Gate of presentation_asset_manifest.md section 3.2 with
    // the deterministic definitions pinned by ADR-0071. Measures the final
    // 2x texture; every rule emits GateViolation entries and one violation
    // fails the file. The per-class scope comes from GateScope.
    public static class CutoutQualityGate
    {
        public const string RuleFormat = "format";
        public const string RuleTranslucentBand = "translucent_band";
        public const string RuleFringeHue = "fringe_hue";
        public const string RuleFringeDeltaL = "fringe_delta_l";
        public const string RuleDilation = "dilation";
        public const string RuleSpeck = "speck";
        public const string RuleJaggies = "jaggies";
        public const string RuleInteriorHole = "interior_hole";
        public const string RuleCellPadding = "cell_padding";
        public const string RuleSize = "size";
        public const string RuleBodyHeight = "body_height";
        public const string RuleTranslucentScope = "translucent_scope";

        private const int SilhouetteAlpha = 128;
        private const int OpaqueAlpha = 255;
        private const int MinComponentAlpha = 16;
        private const int MinSpeckArea = 64;
        private const int CellMargin = 4;
        private const int DilationRadius = 4;
        private const float KeyHue = 300f;
        private const float KeyHueTolerance = 15f;
        private const float KeySaturationMin = 0.30f;
        private const float FringeDeltaLMax = 35f;
        private const float JaggiesMinSoftEdge = 0.5f;
        private const float TranslucentMaxFraction = 0.60f;
        private const int BodyHeightMin = 176;
        private const int BodyHeightMax = 192;

        public static List<GateViolation> ValidatePixels(CutoutGateInput input)
        {
            var violations = new List<GateViolation>();
            var pixels = input.Pixels;
            var width = input.Width;
            var height = input.Height;
            var rules = GateScope.CutoutRules(input.AssetClass);
            var translucent = input.TranslucentMask;

            if (translucent != null)
            {
                var silhouetteForMask = AlphaTopology.AlphaMask(pixels, SilhouetteAlpha, 255);
                var silhouetteCount = AlphaTopology.Count(silhouetteForMask);
                var maskCount = AlphaTopology.Count(translucent);
                if (input.AssetClass != PresentationAssetClass.Actor)
                {
                    violations.Add(new GateViolation(
                        RuleTranslucentScope,
                        "translucent mask declared on non-ACTOR asset class "
                            + input.AssetClass));
                }
                else if (silhouetteCount > 0 && maskCount > TranslucentMaxFraction * silhouetteCount)
                {
                    violations.Add(new GateViolation(
                        RuleTranslucentScope,
                        "translucent mask covers " + maskCount + " px > 60% of silhouette ("
                            + silhouetteCount + " px)"));
                }
            }

            // ART-001: the corner probe is scoped to the cell-based classes;
            // GateScope hands Corners only to those.
            if (rules.HasFlag(CutoutRule.Corners))
            {
                CheckCorners(pixels, width, height, violations);
            }
            if (rules.HasFlag(CutoutRule.TranslucentBand))
            {
                CheckTranslucentBand(pixels, width, height, translucent, violations);
            }
            if (rules.HasFlag(CutoutRule.Fringe))
            {
                CheckFringe(pixels, width, height, violations);
            }
            if (rules.HasFlag(CutoutRule.Dilation))
            {
                CheckDilation(pixels, width, height, violations);
            }
            if (rules.HasFlag(CutoutRule.Specks))
            {
                CheckSpecks(pixels, width, height, input.DeclaredDetachedParts, violations);
            }
            if (rules.HasFlag(CutoutRule.Jaggies))
            {
                CheckJaggies(pixels, width, height, input.DeclaredPixelArt, violations);
            }
            if (rules.HasFlag(CutoutRule.InteriorHoles))
            {
                CheckInteriorHoles(pixels, width, height, translucent, violations);
            }
            if (rules.HasFlag(CutoutRule.CellPadding))
            {
                CheckCellPadding(pixels, width, height, violations);
            }
            if (rules.HasFlag(CutoutRule.Size))
            {
                CheckSize(input, pixels, width, height, violations);
            }
            return violations;
        }

        // File-level entry point: PNG header probe plus the pixel gate. The
        // decoded pixels are returned through input.Pixels so the caller can
        // reuse them for the volume gate.
        public static List<GateViolation> ValidateFile(string absPath, CutoutGateInput input)
        {
            var violations = new List<GateViolation>();
            if (GateScope.CutoutRules(input.AssetClass).HasFlag(CutoutRule.Format))
            {
                var probeError = PngProbe.CheckFile(absPath);
                if (probeError != null)
                {
                    violations.Add(new GateViolation(RuleFormat, probeError));
                    return violations;
                }
            }
            var tex = new Texture2D(2, 2, TextureFormat.RGBA32, false);
            if (!ImageConversion.LoadImage(tex, File.ReadAllBytes(absPath)))
            {
                violations.Add(new GateViolation(RuleFormat, "PNG failed to decode"));
                return violations;
            }
            input.Pixels = tex.GetPixels32();
            input.Width = tex.width;
            input.Height = tex.height;
            Object.DestroyImmediate(tex);
            violations.AddRange(ValidatePixels(input));
            return violations;
        }

        // ART-009 (section 3.11): every SpriteAtlas must pack with Padding
        // >= 4 tex px so the mipmap and texture-compression fades cannot
        // bleed neighbouring sprites together.
        public const int AtlasMinPaddingPx = 4;

        public static List<GateViolation> CheckAtlasPadding(int paddingPx)
        {
            var violations = new List<GateViolation>();
            if (paddingPx < AtlasMinPaddingPx)
            {
                violations.Add(new GateViolation(
                    "atlas_padding",
                    "SpriteAtlas padding " + paddingPx + " < " + AtlasMinPaddingPx + " tex px"));
            }
            return violations;
        }

        // ART-009 (section 3.11): after import, the compressed texture is
        // decompressed (ASTC 4x4/6x6, BC7) and the fringe rule of
        // section 3.2 runs again on the decompressed pixels — compression
        // can re-introduce key-colour and lightness fringes that the source
        // pass removed.
        public static List<GateViolation> CheckPostCompressionFringe(
            Color32[] pixels,
            int width,
            int height)
        {
            var violations = new List<GateViolation>();
            CheckFringe(pixels, width, height, violations);
            for (var i = 0; i < violations.Count; i++)
            {
                var v = violations[i];
                violations[i] = new GateViolation(
                    "post_compression_fringe",
                    v.X, v.Y,
                    v.Detail + " (decompressed texture)");
            }
            return violations;
        }

        // The four 4x4 corners of the cell must be fully transparent; this
        // catches baked checkerboard fake-alpha and solid backgrounds.
        private static void CheckCorners(
            Color32[] pixels,
            int width,
            int height,
            List<GateViolation> violations)
        {
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var inCorner = (x < CellMargin || x >= width - CellMargin)
                        && (y < CellMargin || y >= height - CellMargin);
                    if (inCorner && pixels[y * width + x].a != 0)
                    {
                        violations.Add(new GateViolation(
                            RuleFormat, x, y,
                            "cell corner pixel alpha = " + pixels[y * width + x].a
                                + " (must be 0)"));
                        return;
                    }
                }
            }
        }

        // Every semi-transparent pixel must sit within Chebyshev 2 px of an
        // a = 255 pixel unless it lies inside the declared translucent mask.
        private static void CheckTranslucentBand(
            Color32[] pixels,
            int width,
            int height,
            bool[]? translucent,
            List<GateViolation> violations)
        {
            var opaque = AlphaTopology.AlphaMask(pixels, OpaqueAlpha, 255);
            var dist = AlphaTopology.DistanceToMask(opaque, width, height);
            for (var i = 0; i < pixels.Length; i++)
            {
                var a = pixels[i].a;
                if (a < 1 || a > 254)
                {
                    continue;
                }
                if (translucent != null && translucent[i])
                {
                    continue;
                }
                if (dist[i] > 2)
                {
                    violations.Add(new GateViolation(
                        RuleTranslucentBand, i % width, i / width,
                        "semi-transparent pixel " + dist[i]
                            + " px from the nearest opaque pixel (limit 2)"));
                    if (violations.Count >= 16)
                    {
                        return;
                    }
                }
            }
        }

        // Fringe pixels are 1 <= a <= 254 pixels 8-adjacent to a = 255. They
        // must not carry the key colour (hue +- 15 deg, HSV sat > 0.30) and
        // must not jump |Delta L*| > 35 against the nearest opaque pixel.
        private static void CheckFringe(
            Color32[] pixels,
            int width,
            int height,
            List<GateViolation> violations)
        {
            var opaque = AlphaTopology.AlphaMask(pixels, OpaqueAlpha, 255);
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var i = y * width + x;
                    var a = pixels[i].a;
                    if (a < 1 || a > 254 || !Touches(opaque, width, height, x, y))
                    {
                        continue;
                    }
                    CieLab.ToHsv(pixels[i], out var hue, out var sat, out _);
                    var hueDiff = Mathf.Abs(hue - KeyHue);
                    if (hueDiff > 180f)
                    {
                        hueDiff = 360f - hueDiff;
                    }
                    if (hueDiff <= KeyHueTolerance && sat > KeySaturationMin)
                    {
                        violations.Add(new GateViolation(
                            RuleFringeHue, x, y,
                            "fringe pixel near key hue " + hue + " deg sat " + sat));
                    }
                    var nearest = AlphaTopology.NearestMasked(opaque, width, height, x, y, 8);
                    if (nearest >= 0)
                    {
                        var dl = CieLab.DeltaL(pixels[i], pixels[nearest]);
                        if (dl > FringeDeltaLMax)
                        {
                            violations.Add(new GateViolation(
                                RuleFringeDeltaL, x, y,
                                "fringe |dL*| " + dl + " vs nearest opaque px"));
                        }
                    }
                    if (violations.Count >= 16)
                    {
                        return;
                    }
                }
            }
        }

        // Fully transparent pixels within Chebyshev 4 px of a visible pixel
        // must already carry the RGB of the nearest opaque pixel so bilinear
        // sampling and atlas packing cannot pull in a dark/light halo.
        private static void CheckDilation(
            Color32[] pixels,
            int width,
            int height,
            List<GateViolation> violations)
        {
            var any = AlphaTopology.AlphaMask(pixels, 1, 255);
            var opaque = AlphaTopology.AlphaMask(pixels, OpaqueAlpha, 255);
            var dist = AlphaTopology.DistanceToMask(any, width, height);
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var i = y * width + x;
                    if (pixels[i].a != 0 || dist[i] > DilationRadius)
                    {
                        continue;
                    }
                    var nearest = AlphaTopology.NearestMasked(opaque, width, height, x, y, 16);
                    if (nearest < 0)
                    {
                        continue;
                    }
                    var src = pixels[nearest];
                    var p = pixels[i];
                    if (p.r != src.r || p.g != src.g || p.b != src.b)
                    {
                        violations.Add(new GateViolation(
                            RuleDilation, x, y,
                            "undilated transparent RGB (" + p.r + "," + p.g + "," + p.b
                                + ") != nearest opaque (" + src.r + "," + src.g + ","
                                + src.b + ")"));
                        if (violations.Count >= 16)
                        {
                            return;
                        }
                    }
                }
            }
        }

        // Every 8-connected component (a >= 16) detached from the main body
        // must be at least 64 px, unless the asset declares detached_parts.
        private static void CheckSpecks(
            Color32[] pixels,
            int width,
            int height,
            bool declaredDetachedParts,
            List<GateViolation> violations)
        {
            if (declaredDetachedParts)
            {
                return;
            }
            var mask = AlphaTopology.AlphaMask(pixels, MinComponentAlpha, 255);
            var components = AlphaTopology.Components8(mask, width, height);
            if (components.Count <= 1)
            {
                return;
            }
            var largest = 0;
            for (var i = 1; i < components.Count; i++)
            {
                if (components[i].Length > components[largest].Length)
                {
                    largest = i;
                }
            }
            for (var i = 0; i < components.Count; i++)
            {
                if (i != largest && components[i].Length < MinSpeckArea)
                {
                    violations.Add(new GateViolation(
                        RuleSpeck,
                        components[i][0] % width,
                        components[i][0] / width,
                        "detached component of " + components[i].Length
                            + " px (minimum 64; declare detached_parts for legit pieces)"));
                }
            }
        }

        // At least half of the edge pixels (a > 0 next to a = 0) must be
        // semi-transparent; a binary 0/255 cutout is rejected unless the
        // asset declares pixel_art.
        private static void CheckJaggies(
            Color32[] pixels,
            int width,
            int height,
            bool declaredPixelArt,
            List<GateViolation> violations)
        {
            if (declaredPixelArt)
            {
                return;
            }
            var empty = AlphaTopology.AlphaMask(pixels, 0, 0);
            var edge = 0;
            var soft = 0;
            for (var y = 0; y < height; y++)
            {
                for (var x = 0; x < width; x++)
                {
                    var i = y * width + x;
                    if (pixels[i].a == 0 || !Touches(empty, width, height, x, y))
                    {
                        continue;
                    }
                    edge++;
                    if (pixels[i].a <= 254)
                    {
                        soft++;
                    }
                }
            }
            if (edge == 0)
            {
                return;
            }
            if (soft < edge * JaggiesMinSoftEdge)
            {
                violations.Add(new GateViolation(
                    RuleJaggies,
                    "only " + soft + "/" + edge
                        + " edge pixels are anti-aliased (need >= 50%)"));
            }
        }

        // No pixel with a < 250 may be fully enclosed inside the silhouette
        // (interior holes), except inside declared translucent regions.
        private static void CheckInteriorHoles(
            Color32[] pixels,
            int width,
            int height,
            bool[]? translucent,
            List<GateViolation> violations)
        {
            var candidate = new bool[pixels.Length];
            for (var i = 0; i < pixels.Length; i++)
            {
                candidate[i] = pixels[i].a < 250
                    && (translucent == null || !translucent[i]);
            }
            var components = AlphaTopology.Components8(candidate, width, height);
            foreach (var comp in components)
            {
                var touchesBorder = false;
                foreach (var i in comp)
                {
                    var x = i % width;
                    var y = i / width;
                    if (x == 0 || y == 0 || x == width - 1 || y == height - 1)
                    {
                        touchesBorder = true;
                        break;
                    }
                }
                if (!touchesBorder)
                {
                    violations.Add(new GateViolation(
                        RuleInteriorHole,
                        comp[0] % width,
                        comp[0] / width,
                        comp.Length + " interior pixels with a < 250 enclosed in silhouette"));
                }
            }
        }

        // The silhouette keeps >= 4 texture px from the left/right/top cell
        // edges and 0..4 px from the bottom edge (feet touch the ground).
        private static void CheckCellPadding(
            Color32[] pixels,
            int width,
            int height,
            List<GateViolation> violations)
        {
            var silhouette = AlphaTopology.AlphaMask(pixels, SilhouetteAlpha, 255);
            if (!AlphaTopology.Bounds(silhouette, width, height,
                out var minX, out var minY, out var maxX, out var maxY))
            {
                return;
            }
            if (minX < CellMargin)
            {
                violations.Add(new GateViolation(
                    RuleCellPadding, minX, minY,
                    "silhouette " + minX + " px from left edge (need >= 4)"));
            }
            var rightGap = width - 1 - maxX;
            if (rightGap < CellMargin)
            {
                violations.Add(new GateViolation(
                    RuleCellPadding, maxX, maxY,
                    "silhouette " + rightGap + " px from right edge (need >= 4)"));
            }
            var topGap = height - 1 - maxY;
            if (topGap < CellMargin)
            {
                violations.Add(new GateViolation(
                    RuleCellPadding, maxX, maxY,
                    "silhouette " + topGap + " px from top edge (need >= 4)"));
            }
            if (minY > CellMargin)
            {
                violations.Add(new GateViolation(
                    RuleCellPadding, minX, minY,
                    "silhouette " + minY + " px from bottom edge (need <= 4)"));
            }
        }

        // Texture is exactly 2x the reference cell; the silhouette bounding
        // box stays under the 2x silhouette limit; CHARACTER bodies measure
        // 176..192 texture px over the central band.
        private static void CheckSize(
            CutoutGateInput input,
            Color32[] pixels,
            int width,
            int height,
            List<GateViolation> violations)
        {
            if (input.CellWidth > 0 && (width != input.CellWidth || height != input.CellHeight))
            {
                violations.Add(new GateViolation(
                    RuleSize,
                    "texture " + width + "x" + height + " != 2x cell "
                        + input.CellWidth + "x" + input.CellHeight));
            }
            var silhouette = AlphaTopology.AlphaMask(pixels, SilhouetteAlpha, 255);
            if (!AlphaTopology.Bounds(silhouette, width, height,
                out var minX, out var minY, out var maxX, out var maxY))
            {
                return;
            }
            var bboxW = maxX - minX + 1;
            var bboxH = maxY - minY + 1;
            var capW = input.SilhouetteMaxWidth > 0 ? input.SilhouetteMaxWidth : input.CellWidth;
            var capH = input.SilhouetteMaxHeight > 0 ? input.SilhouetteMaxHeight : input.CellHeight;
            if (capW > 0 && (bboxW > capW || bboxH > capH))
            {
                violations.Add(new GateViolation(
                    RuleSize,
                    "silhouette bbox " + bboxW + "x" + bboxH + " exceeds 2x limit "
                        + capW + "x" + capH));
            }
            if (!input.CheckBodyHeight)
            {
                return;
            }
            var bandMin = minX + bboxW / 4;
            var bandMax = maxX - bboxW / 4;
            var bodyH = 0;
            for (var x = bandMin; x <= bandMax; x++)
            {
                var top = -1;
                var bottom = -1;
                for (var y = minY; y <= maxY; y++)
                {
                    if (!silhouette[y * width + x])
                    {
                        continue;
                    }
                    if (bottom < 0)
                    {
                        bottom = y;
                    }
                    top = y;
                }
                if (top >= 0 && top - bottom + 1 > bodyH)
                {
                    bodyH = top - bottom + 1;
                }
            }
            if (bodyH < BodyHeightMin || bodyH > BodyHeightMax)
            {
                violations.Add(new GateViolation(
                    RuleBodyHeight,
                    "character body height " + bodyH + " px (need 176..192)"));
            }
        }

        private static bool Touches(bool[] mask, int width, int height, int x, int y)
        {
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
                    if (mask[ny * width + nx])
                    {
                        return true;
                    }
                }
            }
            return false;
        }
    }
}
