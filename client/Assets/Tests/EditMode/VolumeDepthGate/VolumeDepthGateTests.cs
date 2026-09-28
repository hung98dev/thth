using System.Linq;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEngine;
using CutoutGateImpl = ThinhThan.Core.Assets.Editor.AssetProduction.CutoutQualityGate;
using VolumeDepthGateImpl = ThinhThan.Core.Assets.Editor.AssetProduction.VolumeDepthGate;

namespace ThinhThan.Tests.EditMode.VolumeDepthGate
{
    // Volume & Depth Gate tests (IMP-070): one passing sprite and one
    // failing fixture per section 3.6 rule, plus the pinned-definition
    // tests named by the packet (k-means init, edge band, translucent
    // scope, spirit-beast/cell-ref sizing, asset_class scoping).
    public class VolumeDepthGateTests
    {
        private const int W = 64;
        private const int H = 64;

        private static Color32[] Blank(int w, int h)
        {
            var px = new Color32[w * h];
            for (var i = 0; i < px.Length; i++)
            {
                px[i] = new Color32(0, 0, 0, 0);
            }
            return px;
        }

        // Clean actor sprite on 64x64: 48x48 silhouette (x,y in 8..55)
        // with a 3 px edge rim split into a light top half and a darker
        // bottom half, and four interior horizontal bands of increasing
        // L* from bottom to top. Satisfies value range, tiers, top-lit,
        // edge separation and the flat-region cap.
        private static Color32[] CleanActor()
        {
            var px = Blank(W, H);
            var silhouette = new bool[W * H];
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    silhouette[y * W + x] = true;
                }
            }
            VolumeDepthGateImpl.ComputeBandAndCore(
                silhouette, W, H, out var band, out _);
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    var i = y * W + x;
                    if (band[i])
                    {
                        px[i] = y >= 32
                            ? new Color32(215, 220, 235, 255)
                            : new Color32(150, 150, 160, 255);
                        continue;
                    }
                    if (y <= 22)
                    {
                        px[i] = new Color32(60, 60, 65, 255);
                    }
                    else if (y <= 33)
                    {
                        px[i] = new Color32(95, 95, 100, 255);
                    }
                    else if (y <= 43)
                    {
                        px[i] = new Color32(135, 128, 120, 255);
                    }
                    else
                    {
                        px[i] = new Color32(175, 168, 160, 255);
                    }
                }
            }
            return px;
        }

        private static VolumeDepthInput Input(Color32[] px, PresentationAssetClass cls)
        {
            return PresentationSizing.ToVolumeInput(px, W, H, cls, null);
        }

        private static string[] Rules(VolumeDepthInput input)
        {
            return VolumeDepthGateImpl.ValidatePixels(input).Select(v => v.Rule).ToArray();
        }

        private static void AssertHasRule(Color32[] px, string rule)
        {
            AssertHasRule(Input(px, PresentationAssetClass.Actor), rule);
        }

        private static void AssertHasRule(VolumeDepthInput input, string rule)
        {
            var rules = Rules(input);
            Assert.IsTrue(rules.Contains(rule),
                "expected rule " + rule + " but got: " + string.Join(",", rules));
        }

        [Test]
        public void TestCleanSpritePasses()
        {
            var rules = Rules(Input(CleanActor(), PresentationAssetClass.Actor));
            Assert.AreEqual(0, rules.Length,
                "clean actor must pass all rules: " + string.Join(",", rules));
        }

        [Test]
        public void TestFlatFillFails()
        {
            var px = Blank(W, H);
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    px[y * W + x] = new Color32(120, 120, 120, 255);
                }
            }
            AssertHasRule(px, "flat_region");
            AssertHasRule(px, "value_range");
        }

        [Test]
        public void TestNarrowValueRangeFails()
        {
            var px = Blank(W, H);
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    var g = (byte)(110 + (y - 8) / 8);
                    px[y * W + x] = new Color32(g, g, g, 255);
                }
            }
            AssertHasRule(px, "value_range");
        }

        [Test]
        public void TestTwoTiersFail()
        {
            var px = Blank(W, H);
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    px[y * W + x] = y < 32
                        ? new Color32(40, 40, 40, 255)
                        : new Color32(200, 200, 200, 255);
                }
            }
            AssertHasRule(px, "value_tiers");
        }

        [Test]
        public void TestBottomLitFails()
        {
            var px = CleanActor();
            // Flip vertically: bright bands move to the bottom third.
            var flipped = Blank(W, H);
            for (var y = 0; y < H; y++)
            {
                for (var x = 0; x < W; x++)
                {
                    flipped[y * W + x] = px[(H - 1 - y) * W + x];
                }
            }
            AssertHasRule(flipped, "top_lit");
        }

        [Test]
        public void TestNoEdgeSeparationFails()
        {
            var px = CleanActor();
            var silhouette = AlphaTopology.AlphaMask(px, 128, 255);
            VolumeDepthGateImpl.ComputeBandAndCore(
                silhouette, W, H, out var band, out _);
            // Recolour the rim to match the adjacent interior band so the
            // edge band no longer separates from the core.
            for (var i = 0; i < px.Length; i++)
            {
                if (band[i])
                {
                    var y = i / W;
                    px[i] = y >= 32
                        ? new Color32(175, 168, 160, 255)
                        : new Color32(60, 60, 65, 255);
                }
            }
            AssertHasRule(px, "edge_separation");
        }

        [Test]
        public void TestFlatRegionFails()
        {
            var px = CleanActor();
            // Overwrite the whole interior with a uniform colour: one flat
            // region covering ~56% of S.
            var silhouette = AlphaTopology.AlphaMask(px, 128, 255);
            VolumeDepthGateImpl.ComputeBandAndCore(
                silhouette, W, H, out var band, out _);
            for (var i = 0; i < px.Length; i++)
            {
                if (silhouette[i] && !band[i])
                {
                    px[i] = new Color32(100, 100, 100, 255);
                }
            }
            AssertHasRule(px, "flat_region");
        }

        [Test]
        public void TestEnvironmentLayers()
        {
            // Passing set: per-layer opaque blocks with decreasing contrast
            // and saturation L1 -> L4.
            var layers = new Color32[4][];
            var contrasts = new[] { 60, 45, 30, 20 };
            var sats = new[] { 200, 160, 120, 80 };
            for (var l = 0; l < 4; l++)
            {
                layers[l] = Blank(64, 64);
                var mid = 128;
                var spread = contrasts[l] / 2;
                var lo = (byte)(mid - spread);
                var hi = (byte)(mid + spread);
                for (var y = 16; y < 48; y++)
                {
                    for (var x = 16; x < 48; x++)
                    {
                        var v = (x + y) % 2 == 0 ? hi : lo;
                        var sat = sats[l];
                        layers[l][y * 64 + x] = new Color32(
                            (byte)System.Math.Min(255, v + sat / 4), v, (byte)System.Math.Max(0, v - sat / 4), 255);
                    }
                }
            }
            var ok = VolumeDepthGateImpl.CheckEnvironmentLayers(layers, 64, 64);
            Assert.AreEqual(0, ok.Count,
                "ordered layers must pass: " + string.Join(",", ok.Select(v => v.Detail)));

            // Inverted contrast: L3 stronger than L1 -> violation.
            var bad = (Color32[][])layers.Clone();
            bad[2] = layers[0];
            var errors = VolumeDepthGateImpl.CheckEnvironmentLayers(bad, 64, 64);
            Assert.IsTrue(errors.Any(v => v.Rule == "environment"),
                "contrast inversion must fail the environment rule");
        }

        [Test]
        public void TestActorOnBackground()
        {
            var render = Blank(64, 64);
            for (var i = 0; i < render.Length; i++)
            {
                render[i] = new Color32(40, 45, 60, 255);
            }
            var mask = new bool[64 * 64];
            for (var y = 20; y < 44; y++)
            {
                for (var x = 20; x < 44; x++)
                {
                    mask[y * 64 + x] = true;
                }
            }
            VolumeDepthGateImpl.ComputeBandAndCore(mask, 64, 64, out var band, out _);
            for (var i = 0; i < render.Length; i++)
            {
                if (mask[i])
                {
                    render[i] = band[i]
                        ? new Color32(220, 200, 160, 255)
                        : new Color32(180, 160, 120, 255);
                }
            }
            var pass = VolumeDepthGateImpl.CheckActorOnBackground(render, mask, 64, 64);
            Assert.AreEqual(0, pass.Count, "bright actor on dark bg must pass");
            var flat = (Color32[])render.Clone();
            for (var i = 0; i < flat.Length; i++)
            {
                if (mask[i])
                {
                    flat[i] = new Color32(48, 52, 66, 255);
                }
            }
            var fail = VolumeDepthGateImpl.CheckActorOnBackground(flat, mask, 64, 64);
            Assert.IsTrue(fail.Any(v => v.Rule == "actor_on_background"),
                "invisible actor must fail the background rule");
        }

        [Test]
        public void TestKMeansDeterministicInit()
        {
            var px = CleanActor();
            var l = new System.Collections.Generic.List<float>();
            for (var i = 0; i < px.Length; i++)
            {
                if (px[i].a >= 128)
                {
                    l.Add(CieLab.LStar(px[i]));
                }
            }
            l.Sort();
            var arr = l.ToArray();
            var seeds = new[]
            {
                KMeans1D.Percentile(arr, 10.0),
                KMeans1D.Percentile(arr, 30.0),
                KMeans1D.Percentile(arr, 50.0),
                KMeans1D.Percentile(arr, 70.0),
                KMeans1D.Percentile(arr, 90.0),
            };
            var a = KMeans1D.Cluster(arr, seeds, out var centersA, out var countsA);
            var b = KMeans1D.Cluster(arr, seeds, out var centersB, out var countsB);
            Assert.AreEqual(centersA, centersB, "k-means must be deterministic");
            Assert.AreEqual(countsA, countsB);
            Assert.AreEqual(a, b);
            // Seeds are exactly the spec percentiles, not RNG picks.
            Assert.AreEqual(arr[(int)System.Math.Floor(0.10 * (arr.Length - 1))], seeds[0]);
            Assert.AreEqual(arr[(int)System.Math.Floor(0.90 * (arr.Length - 1))], seeds[4]);
        }

        [Test]
        public void TestEdgeBandDefinition()
        {
            var mask = new bool[32 * 32];
            for (var y = 10; y <= 21; y++)
            {
                for (var x = 10; x <= 21; x++)
                {
                    mask[y * 32 + x] = true;
                }
            }
            VolumeDepthGateImpl.ComputeBandAndCore(mask, 32, 32, out var band, out var core);
            // Distances are Chebyshev to outside-S measured on a 12x12 block.
            Assert.IsTrue(band[13 * 32 + 10], "dist 3 -> in B");
            Assert.IsTrue(band[13 * 32 + 12], "dist 2 -> in B");
            Assert.IsTrue(band[13 * 32 + 13], "dist 1 -> in B");
            Assert.IsTrue(band[13 * 32 + 14] == false, "dist 4 -> not in B");
            Assert.IsTrue(core[13 * 32 + 15], "dist 5 -> in K");
            Assert.IsTrue(core[13 * 32 + 18], "dist 8 -> in K");
            Assert.IsTrue(core[13 * 32 + 19] == false, "dist 9 -> not in K");
        }

        [Test]
        public void TestTranslucentMaskScope()
        {
            var px = CleanActor();
            var sMask = AlphaTopology.AlphaMask(px, 128, 255);
            var half = new bool[px.Length];
            var count = 0;
            var cap = (int)(0.6 * AlphaTopology.Count(sMask));
            for (var i = 0; i < px.Length && count < cap - 1; i++)
            {
                if (sMask[i])
                {
                    half[i] = true;
                    count++;
                }
            }
            // Mask > 60% of S fails on ACTOR.
            var over = (bool[])sMask.Clone();
            var overInput = Input(px, PresentationAssetClass.Actor);
            overInput.TranslucentMask = over;
            var rules = Rules(overInput);
            Assert.IsTrue(rules.Contains("translucent_scope"),
                "mask covering >60% of S must fail");
            // Any mask on a non-actor class fails outright.
            var badClass = Input(px, PresentationAssetClass.Prop);
            badClass.TranslucentMask = half;
            rules = Rules(badClass);
            Assert.IsTrue(rules.Contains("translucent_scope"));
            // <= 60% mask on ACTOR is accepted.
            var okInput = Input(px, PresentationAssetClass.Actor);
            okInput.TranslucentMask = half;
            rules = Rules(okInput);
            Assert.IsFalse(rules.Contains("translucent_scope"));
        }

        [Test]
        public void TestSpiritBeastAndCellRefSizes()
        {
            var px = CleanActor();
            // SPIRIT_BEAST: texture must be 128x128 (2x of 64x64 cell).
            var sbMeta = new ImportMetadata { SizeProfile = "SPIRIT_BEAST" };
            var sbInput = PresentationSizing.ToCutoutInput(
                px, W, H, PresentationAssetClass.Actor, sbMeta, null);
            Assert.AreEqual(128, sbInput.CellWidth);
            Assert.AreEqual(128, sbInput.CellHeight);
            Assert.AreEqual(96, sbInput.SilhouetteMaxWidth);
            Assert.AreEqual(96, sbInput.SilhouetteMaxHeight);
            var sbRules = CutoutGateImpl.ValidatePixels(sbInput)
                .Select(v => v.Rule).ToArray();
            Assert.IsTrue(sbRules.Contains("size"),
                "64x64 texture under SPIRIT_BEAST must fail size (128x128 required)");
            // PROP cell_ref 32x32 -> texture must be 64x64: clean sprite OK.
            var propMeta = new ImportMetadata { CellRefWidth = 32, CellRefHeight = 32 };
            var propInput = PresentationSizing.ToCutoutInput(
                px, W, H, PresentationAssetClass.Prop, propMeta, null);
            Assert.AreEqual(64, propInput.CellWidth);
            Assert.AreEqual(64, propInput.CellHeight);
            // And a 1x texture (32x32) under the same cell_ref fails.
            var small = PresentationSizing.ToCutoutInput(
                px, 32, 32, PresentationAssetClass.Prop, propMeta, null);
            var smallRules = CutoutGateImpl.ValidatePixels(small)
                .Select(v => v.Rule).ToArray();
            Assert.IsTrue(smallRules.Contains("size"));
        }

        [Test]
        public void TestAssetClassScoping()
        {
            var px = CleanActor();
            // UI_ART is exempt from all volume rules; TILE gets only
            // value range + top-lit; VFX_SOFT and FONT_ATLAS are exempt.
            Assert.AreEqual(0, Rules(Input(px, PresentationAssetClass.UiArt)).Length);
            var tile = Rules(Input(px, PresentationAssetClass.Tile));
            Assert.IsTrue(tile.Contains("value_range") || tile.Length == 0);
            Assert.IsFalse(tile.Contains("edge_separation"),
                "TILE must not run edge separation");
            Assert.AreEqual(0, Rules(Input(px, PresentationAssetClass.VfxSoft)).Length);
            Assert.AreEqual(0, Rules(Input(px, PresentationAssetClass.FontAtlas)).Length);
            // Prop runs all pixel rules but not actor-on-background.
            var scope = GateScope.VolumeRules(PresentationAssetClass.Prop);
            Assert.IsFalse(scope.HasFlag(VolumeRule.ActorOnBackground));
            Assert.IsTrue(scope.HasFlag(VolumeRule.ValueRange | VolumeRule.EdgeSeparation));
        }
    }
}
