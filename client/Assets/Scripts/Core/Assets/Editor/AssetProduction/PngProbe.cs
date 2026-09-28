using System.IO;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Header probe for the section 3.2 format rule: PNG with real alpha —
    // signature valid, IHDR bit depth 8, colour type 6 (truecolour + alpha),
    // and sRGB colour. sRGB is satisfied by an sRGB chunk, by a gAMA chunk
    // of 45455, or by carrying no colour chunks at all (PNG default is sRGB).
    // A conflicting iCCP profile or a non-sRGB gAMA fails.
    public static class PngProbe
    {
        public static string? Check(byte[] png)
        {
            if (png.Length < 33)
            {
                return "file smaller than a minimal PNG header";
            }
            var sig = new byte[] { 137, 80, 78, 71, 13, 10, 26, 10 };
            for (var i = 0; i < 8; i++)
            {
                if (png[i] != sig[i])
                {
                    return "not a PNG file (bad signature)";
                }
            }
            long pos = 8;
            var sawIhdr = false;
            var sawSrgb = false;
            var sawIccp = false;
            long gama = -1;
            while (pos + 8 <= png.Length)
            {
                var length = ReadBe32(png, (int)pos);
                var type = System.Text.Encoding.ASCII.GetString(png, (int)(pos + 4), 4);
                var data = (int)(pos + 8);
                if (data + length + 4 > png.Length)
                {
                    return "truncated PNG chunk";
                }
                if (type == "IHDR")
                {
                    sawIhdr = true;
                    var bitDepth = png[data + 8];
                    var colorType = png[data + 9];
                    if (bitDepth != 8)
                    {
                        return "PNG bit depth " + bitDepth + " (expected 8 bits per channel)";
                    }
                    if (colorType != 6)
                    {
                        return "PNG colour type " + colorType + " (expected 6 = RGBA with real alpha)";
                    }
                }
                else if (type == "sRGB")
                {
                    sawSrgb = true;
                }
                else if (type == "iCCP")
                {
                    sawIccp = true;
                }
                else if (type == "gAMA")
                {
                    gama = ReadBe32(png, data);
                }
                else if (type == "IEND")
                {
                    break;
                }
                pos = data + length + 4;
            }
            if (!sawIhdr)
            {
                return "PNG missing IHDR chunk";
            }
            if (sawSrgb)
            {
                return null;
            }
            if (sawIccp)
            {
                return gama == 45455
                    ? null
                    : "PNG carries an iCCP profile without an sRGB/gAMA marker (sRGB required)";
            }
            if (gama >= 0 && gama != 45455)
            {
                return "PNG gAMA " + gama + " (expected 45455 = sRGB transfer)";
            }
            return null;
        }

        private static long ReadBe32(byte[] b, int pos)
        {
            return ((long)b[pos] << 24) | ((long)b[pos + 1] << 16)
                | ((long)b[pos + 2] << 8) | b[pos + 3];
        }

        public static string? CheckFile(string path)
        {
            if (!File.Exists(path))
            {
                return "file does not exist";
            }
            return Check(File.ReadAllBytes(path));
        }
    }
}
