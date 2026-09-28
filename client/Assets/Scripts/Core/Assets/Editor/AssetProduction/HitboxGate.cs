using System.Collections.Generic;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Hitbox–silhouette alignment of section 3.10 (ART-008): at idle_0
    // the canonical collider's horizontal centre must sit within +-4 ref
    // px of the silhouette centre, and collider width / silhouette width
    // must lie in 0.5..0.9. Measurements are in reference px (texture px
    // / 2 — textures are authored at 2x).
    public static class HitboxGate
    {
        public const string RuleHitbox = "hitbox";

        public const float CentreToleranceRefPx = 4f;
        public const float WidthRatioMin = 0.5f;
        public const float WidthRatioMax = 0.9f;
        public const float TextureScale = 2f;

        // colliderCentreXRefPx/colliderWidthRefPx come from the collider
        // data at idle_0; the silhouette is derived from the frame's
        // alpha >= 128 mask.
        public static List<GateViolation> CheckHitboxSilhouette(
            Color32[] pixels,
            int width,
            int height,
            float colliderCentreXRefPx,
            float colliderWidthRefPx)
        {
            var violations = new List<GateViolation>();
            var mask = AlphaTopology.AlphaMask(pixels, 128, 255);
            if (!AlphaTopology.Bounds(mask, width, height,
                out var minX, out _, out var maxX, out _))
            {
                violations.Add(new GateViolation(
                    RuleHitbox, "empty silhouette"));
                return violations;
            }
            var silCentreRef = (minX + maxX) * 0.5f / TextureScale;
            var silWidthRef = (maxX - minX + 1) / TextureScale;
            var centreErr = Mathf.Abs(colliderCentreXRefPx - silCentreRef);
            if (centreErr > CentreToleranceRefPx)
            {
                violations.Add(new GateViolation(
                    RuleHitbox,
                    "collider centre " + colliderCentreXRefPx + " ref px is "
                        + centreErr + " off silhouette centre " + silCentreRef
                        + " (max 4 ref px)"));
            }
            if (silWidthRef <= 0f)
            {
                violations.Add(new GateViolation(
                    RuleHitbox, "degenerate silhouette width"));
                return violations;
            }
            var ratio = colliderWidthRefPx / silWidthRef;
            if (ratio < WidthRatioMin || ratio > WidthRatioMax)
            {
                violations.Add(new GateViolation(
                    RuleHitbox,
                    "collider/silhouette width ratio " + ratio
                        + " outside " + WidthRatioMin + ".." + WidthRatioMax));
            }
            return violations;
        }
    }
}
