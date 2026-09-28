using System;
using System.Collections.Generic;
using System.IO;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Style Pack presence + palette gate of section 3.8 (ART-005). A pack
    // lives at client/Assets/Art/StyleRef/<fragment>/<pack_id>/ and holds
    // 6-10 APPROVED anchor images, palette.json (Lab colours), style.md
    // and the 4-angle turnarounds. The palette gate requires >= 85% of S
    // to sit within Delta E00 <= 8 of the nearest declared palette colour.
    public static class StylePackGate
    {
        public const string RuleStylePack = "style_pack";
        public const string RulePalette = "palette_gate";

        public const int AnchorMin = 6;
        public const int AnchorMax = 10;
        public const float PaletteMaxDeltaE = 8f;
        public const float PaletteMinFraction = 0.85f;

        private static readonly HashSet<string> AnchorExtensions = new HashSet<string>
        {
            ".png", ".jpg", ".jpeg", ".webp",
        };

        // palette.json shape: {"colors": [[L*, a*, b*], ...]}.
        [Serializable]
        private sealed class PaletteJson
        {
            public float[][] colors = new float[0][];
        }

        public static List<Vector3>? ParsePaletteJson(string json)
        {
            PaletteJson? parsed = null;
            try
            {
                parsed = JsonUtility.FromJson<PaletteJson>(json);
            }
            catch (ArgumentException)
            {
                return null;
            }
            if (parsed == null || parsed.colors == null || parsed.colors.Length == 0)
            {
                return null;
            }
            var result = new List<Vector3>(parsed.colors.Length);
            foreach (var c in parsed.colors)
            {
                if (c == null || c.Length < 3)
                {
                    return null;
                }
                result.Add(new Vector3(c[0], c[1], c[2]));
            }
            return result.Count == 0 ? null : result;
        }

        // Pack-directory checks (file-level; runs in ValidateFiles paths).
        public static List<GateViolation> CheckPackDir(string packDirAbs)
        {
            var violations = new List<GateViolation>();
            if (!Directory.Exists(packDirAbs))
            {
                violations.Add(new GateViolation(
                    RuleStylePack, "style pack directory missing: " + packDirAbs));
                return violations;
            }
            var palettePath = Path.Combine(packDirAbs, "palette.json");
            if (!File.Exists(palettePath))
            {
                violations.Add(new GateViolation(
                    RuleStylePack, "palette.json missing in " + packDirAbs));
            }
            if (!File.Exists(Path.Combine(packDirAbs, "style.md")))
            {
                violations.Add(new GateViolation(
                    RuleStylePack, "style.md missing in " + packDirAbs));
            }
            var anchors = 0;
            foreach (var file in Directory.GetFiles(packDirAbs))
            {
                var ext = Path.GetExtension(file).ToLowerInvariant();
                if (AnchorExtensions.Contains(ext))
                {
                    anchors++;
                }
            }
            if (anchors < AnchorMin || anchors > AnchorMax)
            {
                violations.Add(new GateViolation(
                    RuleStylePack,
                    "anchor count " + anchors + " outside " + AnchorMin + ".." + AnchorMax));
            }
            return violations;
        }

        // Palette gate: >= 85% of S (alpha >= 128 minus translucent mask)
        // must sit within Delta E00 <= 8 of the nearest palette colour.
        public static List<GateViolation> CheckPalette(
            Color32[] pixels,
            int width,
            int height,
            bool[]? translucent,
            List<Vector3> paletteLab)
        {
            var violations = new List<GateViolation>();
            if (paletteLab == null || paletteLab.Count == 0)
            {
                violations.Add(new GateViolation(
                    RulePalette, "empty palette"));
                return violations;
            }
            var inside = 0;
            var total = 0;
            for (var i = 0; i < pixels.Length; i++)
            {
                if (pixels[i].a < 128)
                {
                    continue;
                }
                if (translucent != null && i < translucent.Length && translucent[i])
                {
                    continue;
                }
                total++;
                var lab = CieLab.ToLab(pixels[i]);
                var best = float.MaxValue;
                for (var c = 0; c < paletteLab.Count; c++)
                {
                    var d = CieLab.DeltaE00(lab, paletteLab[c]);
                    if (d < best)
                    {
                        best = d;
                    }
                }
                if (best <= PaletteMaxDeltaE)
                {
                    inside++;
                }
            }
            if (total == 0)
            {
                violations.Add(new GateViolation(RulePalette, "empty silhouette"));
                return violations;
            }
            if (inside < PaletteMinFraction * total)
            {
                violations.Add(new GateViolation(
                    RulePalette,
                    inside + "/" + total + " S pixels within DeltaE00 <= "
                        + PaletteMaxDeltaE + " of the style palette (need >= 85%)"));
            }
            return violations;
        }
    }
}
