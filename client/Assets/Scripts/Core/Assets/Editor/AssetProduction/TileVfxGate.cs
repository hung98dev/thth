using System.Collections.Generic;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Tile seam, 9-slice border and VFX flipbook limits of section 3.9
    // (ART-007): TILE edge wraps within mean Delta E00 <= 2; a UI_ART
    // 9-slice must declare its border and its stretch-axis centre band
    // must hold stddev L* <= 2 or the sprite draws Tiled; gameplay VFX is
    // a <= 16-frame flipbook sheet <= 1024x1024 at 12 or 24 fps with a
    // declared blend and max_instances.
    public static class TileVfxGate
    {
        public const string RuleTileSeam = "tile_seam";
        public const string RuleNineSlice = "nine_slice";
        public const string RuleVfxFlipbook = "vfx_flipbook";

        public const float TileSeamMaxDeltaE = 2f;
        public const float NineSliceCentreStddevMax = 2f;
        public const int FlipbookMaxFrames = 16;
        public const int FlipbookMaxSheetSize = 1024;

        // TILE: the wrapped join must be invisible — |mean Delta E00| of
        // the left/right column pair and of the top/bottom row pair <= 2.
        public static List<GateViolation> CheckTileSeam(
            Color32[] pixels,
            int width,
            int height)
        {
            var violations = new List<GateViolation>();
            var sumX = 0f;
            var sumY = 0f;
            for (var y = 0; y < height; y++)
            {
                sumX += CieLab.DeltaE00(
                    CieLab.ToLab(pixels[y * width]),
                    CieLab.ToLab(pixels[y * width + width - 1]));
            }
            for (var x = 0; x < width; x++)
            {
                sumY += CieLab.DeltaE00(
                    CieLab.ToLab(pixels[x]),
                    CieLab.ToLab(pixels[(height - 1) * width + x]));
            }
            var meanX = height > 0 ? sumX / height : 0f;
            var meanY = width > 0 ? sumY / width : 0f;
            if (meanX > TileSeamMaxDeltaE)
            {
                violations.Add(new GateViolation(
                    RuleTileSeam,
                    "left/right column mean DeltaE00 " + meanX + " > 2 (seam visible)"));
            }
            if (meanY > TileSeamMaxDeltaE)
            {
                violations.Add(new GateViolation(
                    RuleTileSeam,
                    "top/bottom row mean DeltaE00 " + meanY + " > 2 (seam visible)"));
            }
            return violations;
        }

        // UI_ART 9-slice: border must be declared; along the stretch axis
        // the centre band must hold stddev L* <= 2, otherwise the sprite
        // must draw Tiled (a stretched gradient smears).
        public static List<GateViolation> CheckNineSlice(
            Color32[] pixels,
            int width,
            int height,
            int borderLeft,
            int borderRight,
            int borderTop,
            int borderBottom,
            string drawMode)
        {
            var violations = new List<GateViolation>();
            if (borderLeft <= 0 || borderRight <= 0 || borderTop <= 0 || borderBottom <= 0)
            {
                violations.Add(new GateViolation(
                    RuleNineSlice, "9-slice border not declared (left/right/top/bottom > 0)"));
                return violations;
            }
            var centreX0 = borderLeft;
            var centreX1 = width - borderRight;
            var centreY0 = borderBottom;
            var centreY1 = height - borderTop;
            var stretchOkH = CentreBandStddevL(
                pixels, width, height,
                centreX0, centreX1, centreY0, centreY1, horizontal: true);
            var stretchOkV = CentreBandStddevL(
                pixels, width, height,
                centreX0, centreX1, centreY0, centreY1, horizontal: false);
            if ((stretchOkH > NineSliceCentreStddevMax || stretchOkV > NineSliceCentreStddevMax)
                && drawMode != "Tiled")
            {
                violations.Add(new GateViolation(
                    RuleNineSlice,
                    "centre band stddev L* h=" + stretchOkH + " v=" + stretchOkV
                        + " > 2 requires drawMode Tiled, found '" + drawMode + "'"));
            }
            return violations;
        }

        // stddev of L* across the centre band, sampled along the stretch
        // axis (columns for horizontal, rows for vertical).
        private static float CentreBandStddevL(
            Color32[] pixels,
            int width,
            int height,
            int x0,
            int x1,
            int y0,
            int y1,
            bool horizontal)
        {
            var values = new List<float>();
            if (horizontal)
            {
                for (var x = x0; x < x1; x++)
                {
                    for (var y = y0; y < y1; y++)
                    {
                        values.Add(CieLab.LStar(pixels[y * width + x]));
                    }
                }
            }
            else
            {
                for (var y = y0; y < y1; y++)
                {
                    for (var x = x0; x < x1; x++)
                    {
                        values.Add(CieLab.LStar(pixels[y * width + x]));
                    }
                }
            }
            if (values.Count == 0)
            {
                return 0f;
            }
            var mean = 0f;
            foreach (var v in values)
            {
                mean += v;
            }
            mean /= values.Count;
            var sq = 0f;
            foreach (var v in values)
            {
                sq += (v - mean) * (v - mean);
            }
            return Mathf.Sqrt(sq / values.Count);
        }

        // VFX gameplay flipbook: <= 16 frames on a sheet <= 1024x1024 at
        // 12 or 24 fps with declared blend (ADDITIVE|ALPHA) and a declared
        // max_instances cap.
        public static List<GateViolation> CheckVfxFlipbook(
            int frameCount,
            int sheetWidth,
            int sheetHeight,
            int fps,
            string blend,
            int? maxInstances)
        {
            var violations = new List<GateViolation>();
            if (frameCount < 1 || frameCount > FlipbookMaxFrames)
            {
                violations.Add(new GateViolation(
                    RuleVfxFlipbook,
                    "flipbook frame count " + frameCount + " outside 1.." + FlipbookMaxFrames));
            }
            if (sheetWidth > FlipbookMaxSheetSize || sheetHeight > FlipbookMaxSheetSize)
            {
                violations.Add(new GateViolation(
                    RuleVfxFlipbook,
                    "flipbook sheet " + sheetWidth + "x" + sheetHeight
                        + " exceeds " + FlipbookMaxSheetSize + "x" + FlipbookMaxSheetSize));
            }
            if (fps != 12 && fps != 24)
            {
                violations.Add(new GateViolation(
                    RuleVfxFlipbook, "flipbook fps " + fps + " not in {12, 24}"));
            }
            if (blend != "ADDITIVE" && blend != "ALPHA")
            {
                violations.Add(new GateViolation(
                    RuleVfxFlipbook,
                    "flipbook blend '" + blend + "' not in ADDITIVE|ALPHA"));
            }
            if (!maxInstances.HasValue || maxInstances.Value <= 0)
            {
                violations.Add(new GateViolation(
                    RuleVfxFlipbook, "max_instances not declared"));
            }
            return violations;
        }
    }
}
