using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // sRGB to CIELAB (D65) conversion plus the single lightness metric of
    // section 3.6 (L* / Delta L*) and CIEDE2000 Delta E00 for the flat-region
    // test. All functions are pure and deterministic; a Lab triple is carried
    // in a Vector3 as (x = L*, y = a*, z = b*) so no new value type is needed.
    public static class CieLab
    {
        public static Vector3 ToLab(Color32 c)
        {
            var r = SrgbToLinear(c.r / 255f);
            var g = SrgbToLinear(c.g / 255f);
            var b = SrgbToLinear(c.b / 255f);
            var x = r * 0.4124564f + g * 0.3575761f + b * 0.1804375f;
            var y = r * 0.2126729f + g * 0.7151522f + b * 0.0721750f;
            var z = r * 0.0193339f + g * 0.1191920f + b * 0.9503041f;
            var fx = LabF(x / 0.95047f);
            var fy = LabF(y);
            var fz = LabF(z / 1.08883f);
            return new Vector3(116f * fy - 16f, 500f * (fx - fy), 200f * (fy - fz));
        }

        public static float LStar(Color32 c)
        {
            return ToLab(c).x;
        }

        public static float CStar(Vector3 lab)
        {
            return Mathf.Sqrt(lab.y * lab.y + lab.z * lab.z);
        }

        // CIEDE2000 colour difference, kL = kC = kH = 1.
        public static float DeltaE00(Vector3 a, Vector3 b)
        {
            var l1 = a.x;
            var a1 = a.y;
            var b1 = a.z;
            var l2 = b.x;
            var a2 = b.y;
            var b2 = b.z;
            var c1 = Mathf.Sqrt(a1 * a1 + b1 * b1);
            var c2 = Mathf.Sqrt(a2 * a2 + b2 * b2);
            var cMean = (c1 + c2) * 0.5f;
            var cMean7 = Mathf.Pow(cMean, 7f);
            var g = 0.5f * (1f - Mathf.Sqrt(cMean7 / (cMean7 + 6103515625f)));
            var a1p = a1 * (1f + g);
            var a2p = a2 * (1f + g);
            var c1p = Mathf.Sqrt(a1p * a1p + b1 * b1);
            var c2p = Mathf.Sqrt(a2p * a2p + b2 * b2);
            var h1p = HueAngle(b1, a1p);
            var h2p = HueAngle(b2, a2p);
            var dLp = l2 - l1;
            var dCp = c2p - c1p;
            var dHp = DeltaHp(c1p, c2p, h1p, h2p);
            var lMean = (l1 + l2) * 0.5f;
            var cpMean = (c1p + c2p) * 0.5f;
            var hpMean = HpMean(c1p, c2p, h1p, h2p);
            var t = 1f - 0.17f * Mathf.Cos(Mathf.Deg2Rad * (hpMean - 30f))
                + 0.24f * Mathf.Cos(Mathf.Deg2Rad * (2f * hpMean))
                + 0.32f * Mathf.Cos(Mathf.Deg2Rad * (3f * hpMean + 6f))
                - 0.20f * Mathf.Cos(Mathf.Deg2Rad * (4f * hpMean - 63f));
            var dTheta = 30f * Mathf.Exp(-((hpMean - 275f) / 25f) * ((hpMean - 275f) / 25f));
            var cpMean7 = Mathf.Pow(cpMean, 7f);
            var rc = 2f * Mathf.Sqrt(cpMean7 / (cpMean7 + 6103515625f));
            var sl = 1f + 0.015f * (lMean - 50f) * (lMean - 50f)
                / Mathf.Sqrt(20f + (lMean - 50f) * (lMean - 50f));
            var sc = 1f + 0.045f * cpMean;
            var sh = 1f + 0.015f * cpMean * t;
            var rt = -Mathf.Sin(Mathf.Deg2Rad * (2f * dTheta)) * rc;
            var termL = dLp / sl;
            var termC = dCp / sc;
            var termH = dHp / sh;
            var sum = termL * termL + termC * termC + termH * termH + rt * termC * termH;
            return Mathf.Sqrt(Mathf.Max(0f, sum));
        }

        public static float DeltaL(Color32 a, Color32 b)
        {
            return Mathf.Abs(LStar(a) - LStar(b));
        }

        // HSV of a pixel for the magenta-fringe hue/saturation test.
        public static void ToHsv(Color32 c, out float hue, out float saturation, out float value)
        {
            var r = c.r / 255f;
            var g = c.g / 255f;
            var b = c.b / 255f;
            var max = Mathf.Max(r, Mathf.Max(g, b));
            var min = Mathf.Min(r, Mathf.Min(g, b));
            var delta = max - min;
            value = max;
            saturation = max <= 0f ? 0f : delta / max;
            if (delta <= 0f)
            {
                hue = 0f;
                return;
            }
            float h;
            if (max == r)
            {
                h = 60f * (((g - b) / delta) % 6f);
            }
            else if (max == g)
            {
                h = 60f * ((b - r) / delta + 2f);
            }
            else
            {
                h = 60f * ((r - g) / delta + 4f);
            }
            if (h < 0f)
            {
                h += 360f;
            }
            hue = h;
        }

        private static float SrgbToLinear(float c)
        {
            return c <= 0.04045f ? c / 12.92f : Mathf.Pow((c + 0.055f) / 1.055f, 2.4f);
        }

        private static float LabF(float t)
        {
            var delta = 6f / 29f;
            return t > delta * delta * delta ? Mathf.Pow(t, 1f / 3f) : t / (3f * delta * delta) + 4f / 29f;
        }

        private static float HueAngle(float b, float a)
        {
            var h = Mathf.Atan2(b, a) * Mathf.Rad2Deg;
            return h < 0f ? h + 360f : h;
        }

        private static float DeltaHp(float c1p, float c2p, float h1p, float h2p)
        {
            if (c1p * c2p == 0f)
            {
                return 0f;
            }
            var diff = h2p - h1p;
            if (Mathf.Abs(diff) <= 180f)
            {
                return diff;
            }
            return diff > 180f ? diff - 360f : diff + 360f;
        }

        private static float HpMean(float c1p, float c2p, float h1p, float h2p)
        {
            if (c1p * c2p == 0f)
            {
                return h1p + h2p;
            }
            var sum = h1p + h2p;
            if (Mathf.Abs(h1p - h2p) <= 180f)
            {
                return sum * 0.5f;
            }
            return (sum + 360f) * 0.5f < 360f ? (sum + 360f) * 0.5f : (sum - 360f) * 0.5f;
        }
    }
}
