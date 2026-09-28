using System.Linq;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEngine;
using CutoutGateImpl = ThinhThan.Core.Assets.Editor.AssetProduction.CutoutQualityGate;

namespace ThinhThan.Tests.EditMode.CutoutQualityGate
{
    // Cutout Quality Gate tests (IMP-070): one passing clean sprite and one
    // failing fixture per section 3.2 rule. Pixels are fabricated in code —
    // the gate is a pure pixel function so the fixtures run under
    // -nographics EditMode.
    public class CutoutQualityGateTests
    {
        private const int W = 64;
        private const int H = 64;

        private static Color32[] Blank(int w, int h, Color32 fill)
        {
            var px = new Color32[w * h];
            for (var i = 0; i < px.Length; i++)
            {
                px[i] = fill;
            }
            return px;
        }

        // Clean PROP sprite: 64x64 texture over cell_ref 32x32, opaque body
        // inset with a one-pixel a=160 anti-aliased ring of the same colour,
        // transparent surround pre-dilated with the body colour. Passes
        // every section 3.2 rule in scope for PROP (all of them).
        private static CutoutGateInput CleanSprite()
        {
            var edge = new Color32(200, 150, 80, 255);
            var px = Blank(W, H, new Color32(200, 150, 80, 0));
            for (var y = 0; y <= 38; y++)
            {
                for (var x = 12; x <= 51; x++)
                {
                    px[y * W + x] = edge;
                }
            }
            for (var y = 0; y <= 38; y++)
            {
                for (var x = 12; x <= 51; x++)
                {
                    var i = y * W + x;
                    var border = x == 12 || x == 51 || y == 0 || y == 38;
                    var touchEmpty = border
                        && (x == 12 || x == 51 || y == 38);
                    if (touchEmpty)
                    {
                        px[i] = new Color32(200, 150, 80, 160);
                    }
                }
            }
            var meta = new ImportMetadata
            {
                CellRefWidth = 32,
                CellRefHeight = 32,
            };
            return PresentationSizing.ToCutoutInput(
                px, W, H, PresentationAssetClass.Prop, meta, null);
        }

        private static string[] Rules(CutoutGateInput input)
        {
            return global::ThinhThan.Core.Assets.Editor.AssetProduction
                .CutoutQualityGate.ValidatePixels(input)
                .Select(v => v.Rule).ToArray();
        }

        private static void AssertHasRule(CutoutGateInput input, string rule)
        {
            var rules = Rules(input);
            Assert.IsTrue(rules.Contains(rule),
                "expected rule " + rule + " but got: " + string.Join(",", rules));
        }

        [Test]
        public void TestCleanSpritePasses()
        {
            var rules = Rules(CleanSprite());
            Assert.AreEqual(0, rules.Length,
                "clean sprite must pass all rules: " + string.Join(",", rules));
        }

        [Test]
        public void TestFormatCornerOpaqueFails()
        {
            var input = CleanSprite();
            input.Pixels[0] = new Color32(10, 10, 10, 255);
            AssertHasRule(input, "format");
        }

        [Test]
        public void TestFormatPngProbeRejectsNonRgba()
        {
            // IHDR with colour type 2 (RGB, no alpha).
            var png = new byte[33];
            var sig = new byte[] { 137, 80, 78, 71, 13, 10, 26, 10 };
            sig.CopyTo(png, 0);
            png[11] = 13;
            png[12] = (byte)'I';
            png[13] = (byte)'H';
            png[14] = (byte)'D';
            png[15] = (byte)'R';
            png[24] = 8;
            png[25] = 2;
            var err = PngProbe.Check(png);
            Assert.NotNull(err, "RGB PNG must fail the format rule");
        }

        [Test]
        public void TestFormatPngProbeAcceptsRgba8()
        {
            var png = new byte[33];
            var sig = new byte[] { 137, 80, 78, 71, 13, 10, 26, 10 };
            sig.CopyTo(png, 0);
            png[11] = 13;
            png[12] = (byte)'I';
            png[13] = (byte)'H';
            png[14] = (byte)'D';
            png[15] = (byte)'R';
            png[24] = 8;
            png[25] = 6;
            var err = PngProbe.Check(png);
            Assert.IsNull(err);
        }

        [Test]
        public void TestTranslucentBandFails()
        {
            var input = CleanSprite();
            // A semi-transparent pixel 3+ px from the opaque body.
            input.Pixels[20 * W + 56] = new Color32(200, 150, 80, 100);
            var rules = Rules(input);
            Assert.IsTrue(rules.Contains("translucent_band"));
        }

        [Test]
        public void TestTranslucentBandMaskExemptsActor()
        {
            var meta = new ImportMetadata
            {
                SizeProfile = "SPIRIT_BEAST",
            };
            var input = PresentationSizing.ToCutoutInput(
                CleanSprite().Pixels, W, H, PresentationAssetClass.Actor, meta, null);
            input.Pixels[20 * W + 56] = new Color32(200, 150, 80, 100);
            var rules = Rules(input);
            Assert.IsTrue(rules.Contains("translucent_band"));
            var mask = new bool[W * H];
            mask[20 * W + 56] = true;
            // The mask must cover every semi-transparent px: add the rim.
            for (var i = 0; i < input.Pixels.Length; i++)
            {
                if (input.Pixels[i].a >= 1 && input.Pixels[i].a <= 254)
                {
                    mask[i] = true;
                }
            }
            input.TranslucentMask = mask;
            rules = Rules(input);
            Assert.IsFalse(rules.Contains("translucent_band"),
                "declared translucent mask must exempt the band rule");
        }

        [Test]
        public void TestFringeKeyHueFails()
        {
            var input = CleanSprite();
            // Magenta fringe (key colour) on the rim, adjacent to a=255.
            input.Pixels[38 * W + 30] = new Color32(255, 0, 255, 160);
            AssertHasRule(input, "fringe_hue");
        }

        [Test]
        public void TestFringeDeltaLFails()
        {
            var input = CleanSprite();
            // Near-white fringe on a dark body -> |dL*| > 35.
            for (var i = 0; i < input.Pixels.Length; i++)
            {
                if (input.Pixels[i].a == 255)
                {
                    input.Pixels[i] = new Color32(30, 30, 30, 255);
                }
            }
            input.Pixels[38 * W + 30] = new Color32(245, 245, 245, 160);
            AssertHasRule(input, "fringe_delta_l");
        }

        [Test]
        public void TestDilationFails()
        {
            var input = CleanSprite();
            // Transparent pixel 2 px right of the body carries wrong RGB.
            input.Pixels[20 * W + 53] = new Color32(0, 0, 0, 0);
            AssertHasRule(input, "dilation");
        }

        [Test]
        public void TestSpeckFails()
        {
            var input = CleanSprite();
            input.Pixels[58 * W + 58] = new Color32(200, 150, 80, 255);
            AssertHasRule(input, "speck");
        }

        [Test]
        public void TestSpeckDetachedPartsDeclared()
        {
            var input = CleanSprite();
            input.Pixels[58 * W + 58] = new Color32(200, 150, 80, 255);
            input.DeclaredDetachedParts = true;
            var rules = Rules(input);
            Assert.IsFalse(rules.Contains("speck"),
                "declared detached_parts exempts small components");
        }

        [Test]
        public void TestJaggiesBinaryEdgeFails()
        {
            var px = Blank(W, H, new Color32(200, 150, 80, 0));
            for (var y = 0; y <= 38; y++)
            {
                for (var x = 12; x <= 51; x++)
                {
                    px[y * W + x] = new Color32(200, 150, 80, 255);
                }
            }
            var meta = new ImportMetadata
            {
                CellRefWidth = 32,
                CellRefHeight = 32,
            };
            var input = PresentationSizing.ToCutoutInput(
                px, W, H, PresentationAssetClass.Prop, meta, null);
            AssertHasRule(input, "jaggies");
        }

        [Test]
        public void TestJaggiesPixelArtExempt()
        {
            var px = Blank(W, H, new Color32(200, 150, 80, 0));
            for (var y = 0; y <= 38; y++)
            {
                for (var x = 12; x <= 51; x++)
                {
                    px[y * W + x] = new Color32(200, 150, 80, 255);
                }
            }
            var meta = new ImportMetadata
            {
                CellRefWidth = 32,
                CellRefHeight = 32,
            };
            var input = PresentationSizing.ToCutoutInput(
                px, W, H, PresentationAssetClass.Prop, meta, null);
            input.DeclaredPixelArt = true;
            var rules = Rules(input);
            Assert.IsFalse(rules.Contains("jaggies"));
        }

        [Test]
        public void TestInteriorHoleFails()
        {
            var input = CleanSprite();
            input.Pixels[20 * W + 30] = new Color32(200, 150, 80, 0);
            input.Pixels[21 * W + 30] = new Color32(200, 150, 80, 0);
            AssertHasRule(input, "interior_hole");
        }

        [Test]
        public void TestCellPaddingFails()
        {
            var input = CleanSprite();
            // Push the body to the left edge (gap < 4 px).
            for (var y = 0; y <= 38; y++)
            {
                input.Pixels[y * W + 2] = new Color32(200, 150, 80, 255);
                input.Pixels[y * W + 3] = new Color32(200, 150, 80, 160);
            }
            AssertHasRule(input, "cell_padding");
        }

        [Test]
        public void TestSizeWrongTextureFails()
        {
            var input = CleanSprite();
            input.Width = 60;
            var px = new Color32[60 * 64];
            for (var i = 0; i < px.Length; i++)
            {
                px[i] = new Color32(200, 150, 80, 0);
            }
            input.Pixels = px;
            AssertHasRule(input, "size");
        }

        [Test]
        public void TestBodyHeightFails()
        {
            // CHARACTER profile: body height must be 176..192 px.
            var w = 192;
            var h = 256;
            var px = Blank(w, h, new Color32(120, 90, 60, 0));
            for (var y = 0; y <= 100; y++)
            {
                for (var x = 40; x <= 150; x++)
                {
                    px[y * w + x] = new Color32(120, 90, 60, 255);
                }
            }
            var meta = new ImportMetadata
            {
                SizeProfile = "CHARACTER",
            };
            var input = PresentationSizing.ToCutoutInput(
                px, w, h, PresentationAssetClass.Actor, meta, null);
            AssertHasRule(input, "body_height");
        }

        [Test]
        public void TestTranslucentScopeFailsNonActor()
        {
            var input = CleanSprite();
            input.TranslucentMask = new bool[W * H];
            AssertHasRule(input, "translucent_scope");
        }

        [Test]
        public void TestTranslucentScopeFailsTooLarge()
        {
            var meta = new ImportMetadata
            {
                SizeProfile = "SPIRIT_BEAST",
            };
            var input = PresentationSizing.ToCutoutInput(
                CleanSprite().Pixels, W, H, PresentationAssetClass.Actor, meta, null);
            var silhouette = AlphaTopology.AlphaMask(input.Pixels, 128, 255);
            var mask = AlphaTopology.AlphaMask(input.Pixels, 1, 255);
            var sCount = AlphaTopology.Count(silhouette);
            var mCount = AlphaTopology.Count(mask);
            Assert.IsTrue(mCount > 0.6f * sCount,
                "fixture: visible mask must exceed 60% of silhouette");
            input.TranslucentMask = mask;
            AssertHasRule(input, "translucent_scope");
        }

        [Test]
        public void TestCornerRuleScopedByAssetClass()
        {
            // ART-001 (section 3.2): the 4-corner alpha = 0 probe applies
            // to the cell-based classes only — TILE, UI_ART, PARALLAX_FAR
            // and VFX_SOFT may legitimately fill the cell corners.
            var meta = new ImportMetadata { CellRefWidth = 32, CellRefHeight = 32 };
            var tile = PresentationSizing.ToCutoutInput(
                Blank(W, H, new Color32(120, 90, 60, 255)),
                W, H, PresentationAssetClass.Tile, meta, null);
            var tileRules = Rules(tile);
            Assert.IsFalse(tileRules.Contains("format"),
                "TILE with opaque corners must not trip the corner probe");
            var solidUi = PresentationSizing.ToCutoutInput(
                Blank(W, H, new Color32(120, 90, 60, 255)),
                W, H, PresentationAssetClass.UiArt, meta, null);
            Assert.IsFalse(Rules(solidUi).Contains("format"),
                "UI_ART with opaque corners must not trip the corner probe");

            var prop = PresentationSizing.ToCutoutInput(
                Blank(W, H, new Color32(120, 90, 60, 255)),
                W, H, PresentationAssetClass.Prop, meta, null);
            AssertHasRule(prop, "format");
        }

        [Test]
        public void TestAtlasPaddingAndPostCompressionFringe()
        {
            // ART-009 (section 3.11): atlas padding >= 4 tex px.
            Assert.AreEqual(0, CutoutGateImpl.CheckAtlasPadding(4).Count);
            Assert.AreEqual(0, CutoutGateImpl.CheckAtlasPadding(8).Count);
            var pad = CutoutGateImpl.CheckAtlasPadding(2);
            Assert.IsTrue(pad.Exists(v => v.Rule == "atlas_padding"),
                "padding < 4 must fail");

            // Post-compression fringe re-check: the fringe rule runs again
            // on the decompressed texture.
            var px = Blank(W, H, new Color32(200, 150, 80, 0));
            for (var y = 10; y <= 50; y++)
            {
                for (var x = 10; x <= 50; x++)
                {
                    px[y * W + x] = new Color32(200, 150, 80, 255);
                }
            }
            Assert.AreEqual(
                0,
                CutoutGateImpl.CheckPostCompressionFringe(px, W, H).Count,
                "clean decompressed pixels must pass");
            // Magenta key colour surviving in a decompressed semi-transparent
            // edge pixel (1 <= a <= 254 next to an opaque pixel is a fringe).
            px[10 * W + 25] = new Color32(255, 0, 255, 128);
            var fringe = CutoutGateImpl.CheckPostCompressionFringe(px, W, H);
            Assert.IsTrue(fringe.Exists(v => v.Rule == "post_compression_fringe"),
                "fringe on the decompressed texture must fail");
        }

        [Test]
        public void TestScopeUiArtSkipsPixelRules()
        {
            // Same specked sprite: PROP flags a speck, UI_ART does not
            // (scope: format, band, fringe, dilation only).
            var specked = CleanSprite();
            specked.Pixels[58 * W + 58] = new Color32(200, 150, 80, 255);
            AssertHasRule(specked, "speck");
            var ui = PresentationSizing.ToCutoutInput(
                specked.Pixels, W, H, PresentationAssetClass.UiArt,
                new ImportMetadata(), null);
            var rules = Rules(ui);
            Assert.IsFalse(rules.Contains("speck"));
            Assert.IsFalse(rules.Contains("jaggies"));
            Assert.IsFalse(rules.Contains("interior_hole"));
        }

        [Test]
        public void TestScopeVfxSoftFormatOnly()
        {
            var input = PresentationSizing.ToCutoutInput(
                CleanSprite().Pixels, W, H, PresentationAssetClass.VfxSoft,
                new ImportMetadata { CellRefWidth = 32, CellRefHeight = 32 }, null);
            var rules = Rules(input);
            Assert.IsFalse(rules.Contains("speck"));
            Assert.IsFalse(rules.Contains("jaggies"));
            Assert.IsFalse(rules.Contains("interior_hole"));
        }
    }
}
