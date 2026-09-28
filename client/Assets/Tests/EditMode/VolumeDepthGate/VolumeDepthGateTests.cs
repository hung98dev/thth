using System.Collections.Generic;
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
                    // Each interior tier is split into left/right halves whose
                    // tones differ by more than DeltaE00 2 so no single flat
                    // region exceeds 20% of the silhouette.
                    var right = x >= 32;
                    if (y <= 22)
                    {
                        px[i] = right
                            ? new Color32(75, 72, 68, 255)
                            : new Color32(60, 60, 65, 255);
                    }
                    else if (y <= 33)
                    {
                        px[i] = right
                            ? new Color32(110, 108, 103, 255)
                            : new Color32(95, 95, 100, 255);
                    }
                    else if (y <= 43)
                    {
                        px[i] = right
                            ? new Color32(150, 144, 136, 255)
                            : new Color32(135, 128, 120, 255);
                    }
                    else
                    {
                        px[i] = right
                            ? new Color32(190, 184, 176, 255)
                            : new Color32(175, 168, 160, 255);
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
            // A 20x20 block gives Chebyshev distances 1..10 at the centre.
            var mask = new bool[32 * 32];
            for (var y = 6; y <= 25; y++)
            {
                for (var x = 6; x <= 25; x++)
                {
                    mask[y * 32 + x] = true;
                }
            }
            VolumeDepthGateImpl.ComputeBandAndCore(mask, 32, 32, out var band, out var core);
            const int row = 15 * 32;
            Assert.IsTrue(band[row + 6], "dist 1 -> in B");
            Assert.IsTrue(band[row + 7], "dist 2 -> in B");
            Assert.IsTrue(band[row + 8], "dist 3 -> in B");
            Assert.IsTrue(band[row + 9] == false, "dist 4 -> not in B");
            Assert.IsTrue(core[row + 10], "dist 5 -> in K");
            Assert.IsTrue(core[row + 13], "dist 8 -> in K");
            Assert.IsTrue(core[row + 14] == false, "dist 9 -> not in K");
            Assert.IsTrue(band[row + 15] == false && core[row + 15] == false,
                "dist 10 -> in neither");
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

        // ---- ADR-0076 fixtures and gates (ART-002..008, ART-006/011) ----

        // Loads a committed PNG fixture through Unity's decoder, matching
        // the runtime path the gates see.
        private static Color32[] LoadFixture(string name, out int width, out int height)
        {
            var path = System.IO.Path.Combine(
                Application.dataPath,
                "Tests/EditMode/VolumeDepthGate/Fixtures/" + name);
            Assert.IsTrue(System.IO.File.Exists(path), "fixture missing: " + path);
            var tex = new Texture2D(2, 2, TextureFormat.RGBA32, false);
            Assert.IsTrue(
                ImageConversion.LoadImage(tex, System.IO.File.ReadAllBytes(path)),
                "fixture failed to decode: " + name);
            width = tex.width;
            height = tex.height;
            var px = tex.GetPixels32();
            Object.DestroyImmediate(tex);
            return px;
        }

        [Test]
        public void TestFlatRegionLabBinsGradientPasses()
        {
            // ART-002: a smooth gradient shares no identical Lab bin over
            // more than 20% of S — the Lab-bin flat rule must not fire.
            var px = LoadFixture("gradient_smooth_pass.png", out var w, out var h);
            var input = PresentationSizing.ToVolumeInput(
                px, w, h, PresentationAssetClass.Actor, null);
            Assert.IsFalse(Rules(input).Contains("flat_region"),
                "smooth gradient must not trip the Lab-bin flat rule");
        }

        [Test]
        public void TestTopLightPerHueClusterDarkHairPasses()
        {
            // ART-003: dark hair + skin + torso, each lit from above, must
            // satisfy the per-hue-cluster weighted median >= 4 even though
            // the hair's L* stays low — the old global rule misread this
            // as bottom-lit.
            var px = LoadFixture("dark_hair_toplit_pass.png", out var w, out var h);
            var input = PresentationSizing.ToVolumeInput(
                px, w, h, PresentationAssetClass.Actor, null);
            Assert.IsFalse(Rules(input).Contains("top_lit"),
                "top-lit dark hair must not trip the per-cluster top light rule");
        }

        [Test]
        public void TestFixtureFlatFillFails()
        {
            var px = LoadFixture("flat_fill_fail.png", out var w, out var h);
            var input = PresentationSizing.ToVolumeInput(
                px, w, h, PresentationAssetClass.Actor, null);
            Assert.IsTrue(Rules(input).Contains("flat_region"),
                "one identical colour over >20% of S must trip the Lab-bin rule");
        }

        [Test]
        public void TestFixtureBottomLitFails()
        {
            var px = LoadFixture("bottom_lit_fail.png", out var w, out var h);
            var input = PresentationSizing.ToVolumeInput(
                px, w, h, PresentationAssetClass.Actor, null);
            Assert.IsTrue(Rules(input).Contains("top_lit"),
                "bottom-lit silhouette must fail top light");
        }

        [Test]
        public void TestFixtureTileSolidEdgePasses()
        {
            var px = LoadFixture("tile_solid_edge_pass.png", out var w, out var h);
            Assert.AreEqual(
                0,
                TileVfxGate.CheckTileSeam(px, w, h).Count,
                "wrapping tile edges must pass the seam rule");
        }

        [Test]
        public void TestPaletteGateAgainstStylePack()
        {
            // ART-005: >= 85% of S within DeltaE00 <= 8 of the nearest
            // palette colour of the declared style_pack_id.
            var palette = new List<Vector3>
            {
                CieLab.ToLab(new Color32(60, 60, 65, 255)),
                CieLab.ToLab(new Color32(150, 150, 160, 255)),
                CieLab.ToLab(new Color32(215, 220, 235, 255)),
            };
            var px = CleanActor();
            Assert.AreEqual(
                0,
                StylePackGate.CheckPalette(px, W, H, null, palette).Count,
                "actor colours inside the pack palette must pass");
            var alien = new List<Vector3>
            {
                CieLab.ToLab(new Color32(0, 255, 0, 255)),
            };
            var fail = StylePackGate.CheckPalette(px, W, H, null, alien);
            Assert.IsTrue(fail.Exists(v => v.Rule == "palette_gate"),
                "colours outside the pack palette must fail");
        }

        [Test]
        public void TestFrameConsistencyAndPivot()
        {
            // ART-004: frame-by-frame frames keep hue-cluster mean Lab
            // within DeltaE00 3 and bbox width within 8 tex px of idle_0.
            var idle0 = CleanActor();
            var frame = (Color32[])idle0.Clone();
            var ok = AnimationContract.CheckFrameConsistency(
                idle0, frame, W, H, "idle");
            Assert.AreEqual(0, ok.Count, "identical frame must pass");

            // A widened bbox beyond 8 px fails (attack clips allow 32).
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 56; x <= 60; x++)
                {
                    frame[y * W + x] = new Color32(150, 150, 160, 255);
                }
            }
            var wide = AnimationContract.CheckFrameConsistency(
                idle0, frame, W, H, "idle");
            Assert.IsTrue(wide.Exists(v => v.Rule == "animation_frame_consistency"),
                "bbox widening > 8 px must fail");
            var allowed = AnimationContract.CheckFrameConsistency(
                idle0, frame, W, H, "attack_1");
            Assert.AreEqual(0, allowed.Count,
                "attack clips allow a 32 px bbox width diff");

            // A hue drift beyond DeltaE00 3 fails.
            for (var i = 0; i < frame.Length; i++)
            {
                if (frame[i].a >= 128)
                {
                    frame[i] = new Color32(255, 0, 200, frame[i].a);
                }
            }
            var drift = AnimationContract.CheckFrameConsistency(
                idle0, frame, W, H, "idle");
            Assert.IsTrue(drift.Exists(v => v.Rule == "animation_frame_consistency"),
                "hue drift > DeltaE00 3 must fail");

            // Pivot Bottom Center.
            Assert.AreEqual(
                0,
                AnimationContract.CheckPivot(new Vector2(0.5f, 0f)).Count);
            Assert.AreEqual(
                1,
                AnimationContract.CheckPivot(new Vector2(0.5f, 0.5f)).Count);
        }

        [Test]
        public void TestTileSeam()
        {
            // ART-007: |mean DeltaE00| of wrapping edges <= 2.
            var px = Blank(W, H);
            for (var i = 0; i < px.Length; i++)
            {
                var x = i % W;
                var y = i / W;
                px[i] = new Color32((byte)(100 + x / 4), 80, 60, 255);
            }
            Assert.AreEqual(0, TileVfxGate.CheckTileSeam(px, W, H).Count);
            var seam = new Color32(255, 255, 255, 255);
            for (var y = 0; y < H; y++)
            {
                px[y * W + W - 1] = seam;
            }
            var fail = TileVfxGate.CheckTileSeam(px, W, H);
            Assert.IsTrue(fail.Exists(v => v.Rule == "tile_seam"),
                "a visible wrapped edge must fail");
        }

        [Test]
        public void TestNineSliceBorder()
        {
            // ART-007: 9-slice needs a declared border; a non-uniform
            // centre band must draw Tiled, not Stretched.
            var px = Blank(W, H);
            for (var i = 0; i < px.Length; i++)
            {
                px[i] = new Color32(120, 120, 130, 255);
            }
            var undeclared = TileVfxGate.CheckNineSlice(
                px, W, H, 0, 0, 0, 0, "Stretched");
            Assert.IsTrue(undeclared.Exists(v => v.Rule == "nine_slice"),
                "missing border must fail");
            Assert.AreEqual(
                0,
                TileVfxGate.CheckNineSlice(px, W, H, 8, 8, 8, 8, "Stretched").Count,
                "uniform centre may stretch");
            for (var y = 8; y < H - 8; y++)
            {
                for (var x = 8; x < W - 8; x++)
                {
                    px[y * W + x] = new Color32((byte)(60 + x), 120, 130, 255);
                }
            }
            var stretched = TileVfxGate.CheckNineSlice(
                px, W, H, 8, 8, 8, 8, "Stretched");
            Assert.IsTrue(stretched.Exists(v => v.Rule == "nine_slice"),
                "gradient centre must not stretch");
            Assert.AreEqual(
                0,
                TileVfxGate.CheckNineSlice(px, W, H, 8, 8, 8, 8, "Tiled").Count,
                "gradient centre must draw Tiled");
        }

        [Test]
        public void TestVfxFlipbookLimits()
        {
            // ART-007: <= 16 frames, sheet <= 1024x1024, 12|24 fps,
            // ADDITIVE|ALPHA blend, declared max_instances.
            Assert.AreEqual(
                0,
                TileVfxGate.CheckVfxFlipbook(8, 512, 512, 12, "ADDITIVE", 4).Count);
            Assert.IsTrue(TileVfxGate.CheckVfxFlipbook(
                17, 512, 512, 12, "ADDITIVE", 4).Exists(v => v.Rule == "vfx_flipbook"));
            Assert.IsTrue(TileVfxGate.CheckVfxFlipbook(
                8, 2048, 512, 12, "ADDITIVE", 4).Exists(v => v.Rule == "vfx_flipbook"));
            Assert.IsTrue(TileVfxGate.CheckVfxFlipbook(
                8, 512, 512, 30, "ADDITIVE", 4).Exists(v => v.Rule == "vfx_flipbook"));
            Assert.IsTrue(TileVfxGate.CheckVfxFlipbook(
                8, 512, 512, 12, "SCREEN", 4).Exists(v => v.Rule == "vfx_flipbook"));
            Assert.IsTrue(TileVfxGate.CheckVfxFlipbook(
                8, 512, 512, 12, "ALPHA", null).Exists(v => v.Rule == "vfx_flipbook"));
        }

        [Test]
        public void TestHitboxSilhouetteAlignment()
        {
            // ART-008: idle_0 collider centre within +-4 ref px of the
            // silhouette centre; width ratio in 0.5..0.9.
            var px = Blank(W, H);
            for (var y = 8; y <= 55; y++)
            {
                for (var x = 8; x <= 55; x++)
                {
                    px[y * W + x] = new Color32(180, 170, 160, 255);
                }
            }
            // silhouette x 8..55 -> centre 31.5 tex = 15.75 ref; width 48/2 = 24 ref.
            Assert.AreEqual(
                0,
                HitboxGate.CheckHitboxSilhouette(px, W, H, 16f, 18f).Count,
                "centred collider inside the ratio band must pass");
            Assert.IsTrue(HitboxGate.CheckHitboxSilhouette(
                px, W, H, 24f, 18f).Exists(v => v.Rule == "hitbox"),
                "centre off by > 4 ref px must fail");
            Assert.IsTrue(HitboxGate.CheckHitboxSilhouette(
                px, W, H, 16f, 8f).Exists(v => v.Rule == "hitbox"),
                "ratio < 0.5 must fail");
            Assert.IsTrue(HitboxGate.CheckHitboxSilhouette(
                px, W, H, 16f, 22f).Exists(v => v.Rule == "hitbox"),
                "ratio > 0.9 must fail");
        }

        [Test]
        public void TestReviewLowProfileMotionAndRubric()
        {
            // ART-006: the LOW profile is pinned at 960x540 with a 2 s
            // horizontal motion clip over the actor object.
            Assert.AreEqual(960, VisualReviewMatrix.LowResolution.Width);
            Assert.AreEqual(540, VisualReviewMatrix.LowResolution.Height);
            Assert.AreEqual(2f, VisualReviewMatrix.LowMotionSeconds);

            // ART-011: the run emits the 0/1/2 rubric template and the
            // contact sheet beside the Style Pack anchors.
            var dir = System.IO.Path.Combine(
                System.IO.Path.GetTempPath(),
                "imp070_review_" + System.Guid.NewGuid().ToString("N"));
            try
            {
                var written = new List<string>();
                VisualReviewRenderer.WriteReviewAids(dir, written);
                var rubricPath = System.IO.Path.Combine(dir, "_review", "rubric.json");
                var sheetPath = System.IO.Path.Combine(dir, "_review", "contact_sheet.md");
                Assert.IsTrue(System.IO.File.Exists(rubricPath), "rubric.json missing");
                Assert.IsTrue(System.IO.File.Exists(sheetPath), "contact_sheet.md missing");
                var rubric = System.IO.File.ReadAllText(rubricPath);
                StringAssert.Contains("shimmer", rubric);
                StringAssert.Contains("0|1|2", rubric);
                Assert.IsTrue(written.Contains("_review/rubric.json"));
                Assert.IsTrue(written.Contains("_review/contact_sheet.md"));
            }
            finally
            {
                if (System.IO.Directory.Exists(dir))
                {
                    System.IO.Directory.Delete(dir, true);
                }
            }
        }
    }
}
