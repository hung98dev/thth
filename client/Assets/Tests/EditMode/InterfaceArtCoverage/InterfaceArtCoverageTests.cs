using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEngine;
using CutoutGateImpl = ThinhThan.Core.Assets.Editor.AssetProduction.CutoutQualityGate;
using VolumeDepthGateImpl = ThinhThan.Core.Assets.Editor.AssetProduction.VolumeDepthGate;

namespace ThinhThan.Tests.EditMode.InterfaceArtCoverage
{
    // IMP-073 interface art coverage: catalog ID -> produced file mapping,
    // font glyph coverage, VFX/telegraph declarations, budgets, gates and
    // provenance. The produced-file layout conventions live here in one
    // place so a layout change is a single diff:
    //   skill vfx  : Assets/Art/VFX/skills/<el>/<el>_<name>.png  (VFX_SOFT flipbook)
    //   skill icon : Assets/Art/VFX/skills/<el>/icons/<el>_<name>.png (ITEM_ICON)
    //   item icon  : Assets/Art/Items/icons/<id segments flattened>.png
    //   eq icon    : Assets/Art/Items/eq/<tier>_<set>/<slot>.png
    //   effect icon: Assets/Art/UI/effects/<id segments flattened>.png
    //   telegraph  : Assets/Art/VFX/telegraphs/telegraph_<mode>.png
    //   ui sprite  : Assets/Art/UI/<suffix>.png from _uiInventory below
    //   font       : Assets/Art/UI/Fonts/BeVietnamPro-<Weight>.ttf
    public class InterfaceArtCoverageTests
    {
        private const string UiRoot = "Assets/Art/UI/";
        private const string ItemsRoot = "Assets/Art/Items/";
        private const string VfxRoot = "Assets/Art/VFX/";
        private const string StyleRefRoot = "Assets/Art/StyleRef/interface/";
        private const string FragmentRepoPath =
            "client/Assets/Art/Provenance/fragments/interface.json";
        private const string TermsDir = "Assets/Art/Provenance/terms/interface/";
        private const string FontRegular = "Assets/Art/UI/Fonts/BeVietnamPro-Regular.ttf";
        private const string FontSemiBold = "Assets/Art/UI/Fonts/BeVietnamPro-SemiBold.ttf";

        // Canonical Vietnamese glyph set (client_localization.md Fonts):
        // ASCII printable, Latin-1 supplement, Vietnamese extensions, đồng.
        private const string RequiredGlyphs =
            " !\"#$%&'()*+,-./0123456789:;<=>?@"
            + "ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
            + "\u00A0\u00A1\u00A2\u00A3\u00A4\u00A5\u00A6\u00A7\u00A8\u00A9\u00AA\u00AB\u00AC\u00AD\u00AE\u00AF"
            + "\u00B0\u00B1\u00B2\u00B3\u00B4\u00B5\u00B6\u00B7\u00B8\u00B9\u00BA\u00BB\u00BC\u00BD\u00BE\u00BF"
            + "\u00C0\u00C1\u00C2\u00C3\u00C4\u00C5\u00C6\u00C7\u00C8\u00C9\u00CA\u00CB\u00CC\u00CD\u00CE\u00CF"
            + "\u00D0\u00D1\u00D2\u00D3\u00D4\u00D5\u00D6\u00D7\u00D8\u00D9\u00DA\u00DB\u00DC\u00DD\u00DE\u00DF"
            + "\u00E0\u00E1\u00E2\u00E3\u00E4\u00E5\u00E6\u00E7\u00E8\u00E9\u00EA\u00EB\u00EC\u00ED\u00EE\u00EF"
            + "\u00F0\u00F1\u00F2\u00F3\u00F4\u00F5\u00F6\u00F7\u00F8\u00F9\u00FA\u00FB\u00FC\u00FD\u00FE\u00FF"
            + "\u0102\u0103\u0110\u0111\u0128\u0129\u0168\u0169\u01A0\u01A1\u01AF\u01B0"
            + "\u1EA0\u1EA1\u1EA2\u1EA3\u1EA4\u1EA5\u1EA6\u1EA7\u1EA8\u1EA9\u1EAA\u1EAB"
            + "\u1EAC\u1EAD\u1EAE\u1EAF\u1EB0\u1EB1\u1EB2\u1EB3\u1EB4\u1EB5\u1EB6\u1EB7"
            + "\u1EB8\u1EB9\u1EBA\u1EBB\u1EBC\u1EBD\u1EBE\u1EBF\u1EC0\u1EC1\u1EC2\u1EC3"
            + "\u1EC4\u1EC5\u1EC6\u1EC7\u1EC8\u1EC9\u1ECA\u1ECB\u1ECC\u1ECD\u1ECE\u1ECF"
            + "\u1ED0\u1ED1\u1ED2\u1ED3\u1ED4\u1ED5\u1ED6\u1ED7\u1ED8\u1ED9\u1EDA\u1EDB"
            + "\u1EDC\u1EDD\u1EDE\u1EDF\u1EE0\u1EE1\u1EE2\u1EE3\u1EE4\u1EE5\u1EE6\u1EE7"
            + "\u1EE8\u1EE9\u1EEA\u1EEB\u1EEC\u1EED\u1EEE\u1EEF\u1EF0\u1EF1\u1EF2\u1EF3"
            + "\u1EF4\u1EF5\u1EF6\u1EF7\u1EF8\u1EF9"
            + "\u20AB";
        private const string StackedDiacriticChars =
            "\u1EB2\u1EB4\u1ED4\u1ED6\u1EAB\u1EA4\u1EE0\u1EEE"; // Ẳ Ẵ Ổ Ỗ Ẫ Ấ Ỡ Ữ

        // Release UI inventory (client_experience_contract.md HUD + controls
        // + menus). Suffix -> expected nine-slice declaration (null = none).
        private static readonly (string Suffix, bool ExpectNineSlice)[] UiInventory =
        {
            ("hud/hud_panel", true),
            ("hud/avatar_frame", false),
            ("hud/bar_bg", true),
            ("hud/bar_fill_hp", true),
            ("hud/bar_fill_mp", true),
            ("hud/buff_frame", false),
            ("hud/buff_icon", false),
            ("hud/debuff_frame", false),
            ("hud/debuff_icon", false),
            ("hud/btn_attack", false),
            ("hud/btn_skill", false),
            ("hud/btn_jump", false),
            ("hud/btn_interact", false),
            ("hud/joystick_base", false),
            ("hud/joystick_knob", false),
            ("hud/chat_dock", true),
            ("hud/minimap_frame", true),
            ("hud/quest_panel", true),
            ("hud/ping_icon", false),
            ("hud/level_badge", false),
            ("hud/menu_btn", false),
            ("screens/title_bg", false),
            ("screens/panel_frame", true),
            ("screens/loading_bar_bg", true),
            ("screens/loading_bar_fill", true),
            ("controls/btn_primary", true),
            ("controls/btn_secondary", true),
            ("controls/input_field", true),
            ("controls/toggle_on", false),
            ("controls/toggle_off", false),
            ("controls/slider_track", true),
            ("controls/slider_handle", false),
            ("controls/checkbox", false),
            ("controls/tooltip", true),
            ("controls/scrollbar", true),
            ("icons/icon_close", false),
            ("icons/icon_settings", false),
            ("icons/icon_inventory", false),
            ("icons/icon_skillbook", false),
            ("icons/icon_quest", false),
            ("icons/icon_party", false),
            ("icons/icon_guild", false),
            ("icons/icon_map", false),
            ("icons/icon_mail", false),
            ("icons/icon_shop", false),
            ("icons/icon_channel", false),
            ("icons/icon_lock", false),
        };

        // Every non-NONE targeting_mode of class_skill_catalog resolves to
        // exactly one shared decal; SELF maps to the self ring and
        // DIRECTION_BOX/MELEE_BOX share the lane box.
        private static readonly string[] TelegraphModes =
        {
            "direction", "direction_box", "projectile", "area_position",
            "area_self", "single_target",
        };

        private static readonly Dictionary<string, string> TelegraphByMode =
            new Dictionary<string, string>(StringComparer.Ordinal)
        {
            ["DIRECTION"] = "direction",
            ["DIRECTION_BOX"] = "direction_box",
            ["MELEE_BOX"] = "direction_box",
            ["PROJECTILE"] = "projectile",
            ["AREA_POSITION"] = "area_position",
            ["AREA_SELF"] = "area_self",
            ["SELF"] = "area_self",
            ["SINGLE_TARGET"] = "single_target",
        };

        private static string ProjectRoot
        {
            get { return Path.GetDirectoryName(Application.dataPath); }
        }

        private static string Abs(string assetsRelPath)
        {
            return Path.Combine(ProjectRoot, assetsRelPath);
        }

        private static Color32[] LoadPx(string assetsRelPath, out int w, out int h)
        {
            var abs = Abs(assetsRelPath);
            Assert.IsTrue(File.Exists(abs), "missing file: " + assetsRelPath);
            var tex = new Texture2D(2, 2, TextureFormat.RGBA32, false);
            Assert.IsTrue(
                ImageConversion.LoadImage(tex, File.ReadAllBytes(abs)),
                "PNG decode failed: " + assetsRelPath);
            w = tex.width;
            h = tex.height;
            var px = tex.GetPixels32();
            UnityEngine.Object.DestroyImmediate(tex);
            return px;
        }

        // Reads the sidecar declaration next to a produced PNG. The editor
        // importer (InterfaceArtImporter) stamps this into the TextureImporter
        // userData during materialization; the test reads the sidecar so the
        // declarations are verified even before the .meta is materialized.
        private static Dictionary<string, string> ReadDecl(string assetsRelPath)
        {
            var sidecar = Abs(assetsRelPath + ".importmeta");
            Assert.IsTrue(File.Exists(sidecar),
                "missing .importmeta sidecar: " + assetsRelPath);
            var map = new Dictionary<string, string>();
            foreach (var raw in File.ReadAllText(sidecar).Trim().Split(';'))
            {
                var token = raw.Trim();
                if (token.Length == 0)
                {
                    continue;
                }
                var eq = token.IndexOf('=');
                var k = eq >= 0 ? token.Substring(0, eq).Trim() : token;
                map[k] = eq >= 0 ? token.Substring(eq + 1).Trim() : "";
            }
            return map;
        }

        private static string SkillVfxPath(string skillId)
        {
            var parts = skillId.Split('.');
            var el = parts[1];
            var name = parts[3];
            return VfxRoot + "skills/" + el + "/" + el + "_" + name + ".png";
        }

        private static string SkillIconPath(string skillId)
        {
            var parts = skillId.Split('.');
            var el = parts[1];
            var name = parts[3];
            return VfxRoot + "skills/" + el + "/icons/" + el + "_" + name + ".png";
        }

        private static string ItemIconPath(string itemId)
        {
            return ItemsRoot + "icons/" + itemId.Substring(5).Replace('.', '_') + ".png";
        }

        private static string EquipmentIconPath(string eqId)
        {
            var parts = eqId.Split('.');
            return ItemsRoot + "eq/" + parts[2] + "_" + parts[3] + "/" + parts[4] + ".png";
        }

        private static string EffectIconPath(string effectId)
        {
            return UiRoot + "effects/" + effectId.Substring(7).Replace('.', '_') + ".png";
        }

        private static string TelegraphPath(string mode)
        {
            return VfxRoot + "telegraphs/telegraph_" + mode + ".png";
        }

        private static void AssertGateClean(
            string rel, PresentationAssetClass cls, ImportMetadata meta)
        {
            var px = LoadPx(rel, out var w, out var h);
            var cutout = CutoutGateImpl.ValidatePixels(
                PresentationSizing.ToCutoutInput(px, w, h, cls, meta, null));
            Assert.AreEqual(0, cutout.Count,
                rel + " cutout violations: "
                + string.Join(", ", cutout.Select(v => v.Rule)));
            var volume = VolumeDepthGateImpl.ValidatePixels(
                PresentationSizing.ToVolumeInput(px, w, h, cls, null));
            Assert.AreEqual(0, volume.Count,
                rel + " volume violations: "
                + string.Join(", ", volume.Select(v => v.Rule)));
        }

        // ---- catalog coverage ------------------------------------------------

        [Test]
        public void TestCatalogSkillVfxAndIconCoverage()
        {
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var skills = reqs.Where(r => r.Kind == CatalogAssetKind.Skill).ToList();
            Assert.GreaterOrEqual(skills.Count, 60, "skill roster scanned");
            foreach (var req in skills)
            {
                var vfx = SkillVfxPath(req.CatalogId);
                Assert.IsTrue(File.Exists(Abs(vfx)),
                    "skill VFX missing: " + req.CatalogId);
                Assert.AreEqual("VFX_SOFT", ReadDecl(vfx)["asset_class"], vfx);
                var icon = SkillIconPath(req.CatalogId);
                Assert.IsTrue(File.Exists(Abs(icon)),
                    "skill icon missing: " + req.CatalogId);
                Assert.AreEqual("ITEM_ICON", ReadDecl(icon)["asset_class"], icon);
            }
        }

        [Test]
        public void TestCatalogItemIconCoverage()
        {
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var items = reqs.Where(
                r => r.Kind == CatalogAssetKind.Item
                    && !r.CatalogId.StartsWith("item.eq.", System.StringComparison.Ordinal));
            foreach (var req in items)
            {
                var icon = ItemIconPath(req.CatalogId);
                Assert.IsTrue(File.Exists(Abs(icon)),
                    "item icon missing: " + req.CatalogId);
                Assert.AreEqual("ITEM_ICON", ReadDecl(icon)["asset_class"], icon);
            }
        }

        [Test]
        public void TestCatalogEquipmentIconCoverage()
        {
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var eqs = reqs.Where(
                r => r.Kind == CatalogAssetKind.Item
                    && r.CatalogId.StartsWith("item.eq.", System.StringComparison.Ordinal));
            var seen = new HashSet<string>();
            foreach (var req in eqs)
            {
                if (!seen.Add(req.CatalogId))
                {
                    continue;
                }
                var icon = EquipmentIconPath(req.CatalogId);
                Assert.IsTrue(File.Exists(Abs(icon)),
                    "equipment icon missing: " + req.CatalogId);
                Assert.AreEqual("EQUIPMENT_ICON", ReadDecl(icon)["asset_class"], icon);
            }
        }

        [Test]
        public void TestCatalogEffectIconCoverage()
        {
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var effects = reqs.Where(r => r.Kind == CatalogAssetKind.Effect);
            foreach (var req in effects)
            {
                var icon = EffectIconPath(req.CatalogId);
                Assert.IsTrue(File.Exists(Abs(icon)),
                    "effect icon missing: " + req.CatalogId);
                Assert.AreEqual("UI_ART", ReadDecl(icon)["asset_class"], icon);
            }
        }

        [Test]
        public void TestUiSpriteInventoryResolves()
        {
            foreach (var (suffix, nineSlice) in UiInventory)
            {
                var rel = UiRoot + suffix + ".png";
                Assert.IsTrue(File.Exists(Abs(rel)), "UI sprite missing: " + rel);
                var decl = ReadDecl(rel);
                Assert.AreEqual("UI_ART", decl["asset_class"], rel);
                Assert.AreEqual(nineSlice, decl.ContainsKey("nine_slice"),
                    rel + " nine_slice declaration mismatch");
            }
        }

        [Test]
        public void TestTelegraphInventoryResolves()
        {
            foreach (var mode in TelegraphModes)
            {
                var rel = TelegraphPath(mode);
                Assert.IsTrue(File.Exists(Abs(rel)), "telegraph missing: " + rel);
                var decl = ReadDecl(rel);
                Assert.AreEqual("VFX_SOFT", decl["asset_class"], rel);
            }
            // Every non-NONE targeting_mode declared by the catalog binds to
            // exactly one shared decal on disk.
            var modes = new HashSet<string>(StringComparer.Ordinal);
            foreach (Match m in Regex.Matches(
                        File.ReadAllText(Path.Combine(
                            Path.GetFullPath(Path.Combine(ProjectRoot, "..")),
                            "docs/07_content/class_skill_catalog.md")),
                        @"\|\s*(DIRECTION_BOX|MELEE_BOX|DIRECTION|PROJECTILE|AREA_POSITION|AREA_SELF|SINGLE_TARGET|SELF|NONE)\s*\|"))
            {
                modes.Add(m.Groups[1].Value);
            }
            modes.Remove("NONE");
            foreach (var mode in modes)
            {
                Assert.IsTrue(TelegraphByMode.ContainsKey(mode),
                    "no telegraph decal bound for targeting mode " + mode);
            }
        }

        // ---- exact 2x finishing ----------------------------------------------

        [Test]
        public void TestIconsAreExactly2x()
        {
            // §3.2: texture = exactly 2x the declared cell ref. Icon cell_ref
            // is 64x64, so produced icons must be 128x128.
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var paths = new List<string>();
            foreach (var req in reqs)
            {
                if (req.Kind == CatalogAssetKind.Skill)
                {
                    paths.Add(SkillIconPath(req.CatalogId));
                }
                else if (req.Kind == CatalogAssetKind.Item
                        && !req.CatalogId.StartsWith("item.eq.", System.StringComparison.Ordinal))
                {
                    paths.Add(ItemIconPath(req.CatalogId));
                }
            }
            var eqs = reqs.Where(
                r => r.Kind == CatalogAssetKind.Item
                    && r.CatalogId.StartsWith("item.eq.", System.StringComparison.Ordinal));
            var seenEq = new HashSet<string>();
            foreach (var req in eqs)
            {
                if (seenEq.Add(req.CatalogId))
                {
                    paths.Add(EquipmentIconPath(req.CatalogId));
                }
            }
            foreach (var rel in paths)
            {
                if (!File.Exists(Abs(rel)))
                {
                    continue; // existence asserted by coverage tests
                }
                LoadPx(rel, out var w, out var h);
                Assert.AreEqual(128, w, rel);
                Assert.AreEqual(128, h, rel);
            }
        }

        // ---- gates ------------------------------------------------------------

        [Test]
        public void TestAssetClassGateScope()
        {
            // ADR-0076: each file is gated under its own declared asset_class
            // scope, not a uniform one. Icon classes get the full cutout +
            // volume rules; UI_ART the translucent/dilation subset; VFX_SOFT
            // format-only (soft_edges declared).
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var fail = new List<string>();
            foreach (var req in reqs)
            {
                var rel = req.Kind == CatalogAssetKind.Skill
                    ? SkillIconPath(req.CatalogId)
                    : req.Kind == CatalogAssetKind.Item
                        && req.CatalogId.StartsWith("item.eq.", System.StringComparison.Ordinal)
                        ? EquipmentIconPath(req.CatalogId)
                        : req.Kind == CatalogAssetKind.Item
                            ? ItemIconPath(req.CatalogId)
                            : req.Kind == CatalogAssetKind.Effect
                            ? EffectIconPath(req.CatalogId)
                            : null;
                if (rel == null || !File.Exists(Abs(rel)))
                {
                    continue;
                }
                var sidecar = File.ReadAllText(Abs(rel + ".importmeta"));
                var meta = ImportMetadata.Parse(sidecar);
                var cls = meta.AssetClass;
                Assert.IsTrue(cls.HasValue, rel + " has no asset_class");
                try
                {
                    AssertGateClean(rel, cls ?? PresentationAssetClass.UiArt, meta);
                }
                catch (AssertionException e)
                {
                    fail.Add(rel + ": " + e.Message);
                }
            }
            Assert.AreEqual(0, fail.Count,
                "gate failures:\n" + string.Join("\n", fail));
        }

        [Test]
        public void TestNineSliceBordersDeclared()
        {
            foreach (var (suffix, expect) in UiInventory)
            {
                var rel = UiRoot + suffix + ".png";
                if (!File.Exists(Abs(rel)))
                {
                    continue;
                }
                var decl = ReadDecl(rel);
                var has = decl.TryGetValue("nine_slice", out var border);
                Assert.AreEqual(expect, has, rel);
                if (!has)
                {
                    continue;
                }
                var b = border!.Split(',')
                    .Select(int.Parse).ToArray();
                Assert.AreEqual(4, b.Length, rel);
                foreach (var v in b)
                {
                    Assert.Greater(v, 0, rel + " border must be > 0");
                }
                Assert.AreEqual("Sliced", decl.GetValueOrDefault("draw_mode"), rel);
                var px = LoadPx(rel, out var w, out var h);
                var violations = TileVfxGate.CheckNineSlice(
                    px, w, h, b[0], b[1], b[2], b[3], "Sliced");
                Assert.AreEqual(0, violations.Count,
                    rel + " nine-slice violations: "
                    + string.Join(", ", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestVfxFlipbookLimits()
        {
            var reqs = ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot);
            var skills = reqs.Where(r => r.Kind == CatalogAssetKind.Skill);
            foreach (var req in skills)
            {
                var rel = SkillVfxPath(req.CatalogId);
                if (!File.Exists(Abs(rel)))
                {
                    continue;
                }
                var decl = ReadDecl(rel);
                Assert.IsTrue(decl.ContainsKey("soft_edges"), rel);
                var frames = int.Parse(decl["frames"]);
                var fps = int.Parse(decl["fps"]);
                var blend = decl["blend"];
                var maxInst = int.Parse(decl["max_instances"]);
                LoadPx(rel, out var w, out var h);
                var violations = TileVfxGate.CheckVfxFlipbook(
                    frames, w, h, fps, blend, maxInst);
                Assert.AreEqual(0, violations.Count,
                    rel + " flipbook violations: "
                    + string.Join(", ", violations.Select(v => v.Detail)));
                // The sheet is laid out frames x 1: each frame an exact cell.
                Assert.AreEqual(0, w % frames, rel + " frame cells must tile evenly");
            }
        }

        [Test]
        public void TestTelegraphColorIndependence()
        {
            // Mobile/small-screen readability: a telegraph must separate
            // silhouette bands by luminance, not only hue, so dangerous
            // areas stay readable to color-blind players.
            foreach (var mode in TelegraphModes)
            {
                var rel = TelegraphPath(mode);
                if (!File.Exists(Abs(rel)))
                {
                    continue;
                }
                var px = LoadPx(rel, out var w, out var h);
                var opaque = new List<float>();
                for (var i = 0; i < px.Length; i++)
                {
                    if (px[i].a >= 200)
                    {
                        opaque.Add(CieLab.ToLab(px[i]).x);
                    }
                }
                Assert.Greater(opaque.Count, 64, rel + " has no readable mark");
                opaque.Sort();
                // compare ring-dominated high band vs fill-dominated low band
                var q90 = opaque[(int)(opaque.Count * 0.90f)];
                var q10 = opaque[(int)(opaque.Count * 0.10f)];
                Assert.GreaterOrEqual(q90 - q10, 20f,
                    rel + " telegraph lacks luminance separation (needs >= 20 L*)");
            }
        }

        // ---- fonts ------------------------------------------------------------

        [Test]
        public void TestFontGlyphCoverage()
        {
            foreach (var font in new[] { FontRegular, FontSemiBold })
            {
                Assert.IsTrue(File.Exists(Abs(font)), "font missing: " + font);
                var covered = ParseCmap(Abs(font));
                var missing = RequiredGlyphs
                    .Where(c => !covered.ContainsKey((uint)c))
                    .Select(c => "U+" + ((int)c).ToString("X4"))
                    .ToList();
                Assert.AreEqual(0, missing.Count,
                    font + " missing glyphs: " + string.Join(" ", missing));
            }
        }

        [Test]
        public void TestVietnameseGlyphSetNfc()
        {
            // The required glyph string and the locale display-name corpus
            // must already be NFC: stacked-diacritic codepoints are
            // precomposed, never combining sequences.
            var vi = new[]
            {
                "Ẳ", "Ẵ", "Ổ", "Ỗ", "Ẫ", "Ấ", "Ỡ", "Ữ", "Đồng", "Kiếm Khách",
                "Mộc Sư", "Thủy Nữ", "Hỏa Linh", "Thổ Tướng", "Đình Làng",
                "Ư Minh", "Đốm Lửa Rừng", "Bến Nước", "Xóm Chìm", "Đầu Hổ",
                "Đèo Mây", "Thành Cổ", "Trống Trận", "Đấu Cừ", "Núi Thiêng",
            };
            foreach (var s in new[] { RequiredGlyphs }.Concat(vi))
            {
                Assert.AreEqual(
                    s,
                    s.Normalize(NormalizationForm.FormC),
                    "not NFC: " + s);
            }
        }

        [Test]
        public void TestStackedDiacriticsDoNotClip()
        {
            // Stacked-diacritic glyphs are the tallest in the Vietnamese set.
            // A font that fits them inside its declared clipping box keeps
            // ymax <= OS/2.usWinAscent: the box the renderer (TMP text rect,
            // Windows rasterizer) cannot paint beyond. Be Vietnam Pro was
            // designed that way (usWinAscent 1261 >= max glyph ymax 1199);
            // asserting hhea.ascent instead would misread the font, because
            // the typographic ascender is deliberately tighter.
            foreach (var font in new[] { FontRegular, FontSemiBold })
            {
                var metrics = ParseFontMetrics(Abs(font));
                var cmap = ParseCmap(Abs(font));
                foreach (var c in StackedDiacriticChars)
                {
                    Assert.IsTrue(cmap.TryGetValue((uint)c, out var glyphId),
                        font + " lacks stacked glyph U+" + ((int)c).ToString("X4"));
                    var yMax = ParseGlyphYMax(Abs(font), metrics, glyphId);
                    Assert.LessOrEqual(yMax, metrics.UsWinAscent,
                        font + " U+" + ((int)c).ToString("X4")
                        + " ymax " + yMax + " exceeds clipping box "
                        + metrics.UsWinAscent);
                }
            }
        }

        // ---- provenance -------------------------------------------------------

        [Test]
        public void TestProvenanceRowsComplete()
        {
            var abs = Path.Combine(ProjectRoot, "..", FragmentRepoPath);
            Assert.IsTrue(File.Exists(abs), "fragment missing: " + FragmentRepoPath);
            var register = AssetSourceRegisterIO.LoadOrEmpty(abs);
            Assert.Greater(register.assets.Count, 0);
            var repoRoot = Path.GetFullPath(Path.Combine(ProjectRoot, ".."));
            var errors = AssetProvenanceValidator.ValidateFiles(register, repoRoot);
            Assert.AreEqual(0, errors.Count,
                "provenance errors:\n" + string.Join(
                    "\n", errors.Select(e => e.Path + ": " + e.Message)));
            // every produced file under the owned roots has a row
            var rows = new HashSet<string>(
                register.assets.Select(a => a.file_path).OfType<string>());
            foreach (var rel in EnumerateProduced())
            {
                Assert.IsTrue(rows.Contains("client/" + rel),
                    "no provenance row for " + rel);
            }
        }

        [Test]
        public void TestAiCreatedToolMatchesOwnerSetup()
        {
            // ADR-0072: every AI_CREATED row names exactly the owner-provided
            // tool/version of technology_versions.md Content production tools:
            // AI Horde (stablehorde.net API v2), SDXL-family model (the exact
            // model name is recorded per generation — AlbedoBase XL (SDXL),
            // matching the IMP-071 packet for cross-art consistency).
            var abs = Path.Combine(ProjectRoot, "..", FragmentRepoPath);
            var register = AssetSourceRegisterIO.LoadOrEmpty(abs);
            foreach (var row in register.assets)
            {
                if (row.source_kind != "AI_CREATED")
                {
                    continue;
                }
                var gen = row.generation_record;
                Assert.IsNotNull(gen, row.file_path);
                Assert.AreEqual("AI Horde", gen!.tool, row.file_path);
                Assert.AreEqual("stablehorde.net API v2", gen.version, row.file_path);
                Assert.AreEqual("AlbedoBase XL (SDXL)", gen.model_id, row.file_path);
                Assert.IsTrue(AssetProvenanceValidator.IsSha256(gen.terms_snapshot_sha256),
                    row.file_path + " terms_snapshot_sha256");
                Assert.IsTrue(
                    File.Exists(Abs(TermsDir + gen.terms_snapshot_sha256 + ".txt")),
                    "terms snapshot missing for " + row.file_path);
                Assert.GreaterOrEqual(gen.seed, 0, row.file_path + " seed");
            }
        }

        [Test]
        public void TestStylePackPresent()
        {
            Assert.IsTrue(Directory.Exists(Abs(StyleRefRoot)), StyleRefRoot);
            Assert.IsTrue(File.Exists(Abs(StyleRefRoot + "palette.json")));
            Assert.IsTrue(File.Exists(Abs(StyleRefRoot + "style.md")));
            var anchors = Directory.GetFiles(Abs(StyleRefRoot), "anchor_*.png");
            Assert.GreaterOrEqual(anchors.Length, 6,
                "style pack needs >= 6 anchors");
        }

        [Test]
        public void TestPaletteGateAgainstStylePack()
        {
            // §3.8: >= 85% of each produced sprite's silhouette pixels must be
            // within DeltaE00 <= 8 of the nearest Style Pack palette colour.
            var palette = LoadPalette(Abs(StyleRefRoot + "palette.json"));
            Assert.Greater(palette.Count, 0);
            var fail = new List<string>();
            var cache = new Dictionary<int, float>();
            foreach (var rel in EnumerateProduced()
                    .Where(r => r.EndsWith(".png") && !r.Contains("telegraph")))
            {
                var px = LoadPx(rel, out var w, out var h);
                var inBand = 0;
                var total = 0;
                foreach (var c in px)
                {
                    if (c.a < 128)
                    {
                        continue;
                    }
                    total++;
                    var key = (c.r << 16) | (c.g << 8) | c.b;
                    if (!cache.TryGetValue(key, out var best))
                    {
                        var lab = CieLab.ToLab(c);
                        best = float.MaxValue;
                        foreach (var p in palette)
                        {
                            var d = CieLab.DeltaE00(lab, p);
                            if (d < best)
                            {
                                best = d;
                            }
                        }
                        cache[key] = best;
                    }
                    if (best <= 8f)
                    {
                        inBand++;
                    }
                }
                if (total > 0 && (float)inBand / total < 0.85f)
                {
                    fail.Add(rel + " " + (100f * inBand / total).ToString("F1") + "%");
                }
            }
            Assert.AreEqual(0, fail.Count,
                "palette gate failures:\n" + string.Join("\n", fail));
        }

        [Test]
        public void TestBundleBudgets()
        {
            // manifest §1: icon group <= 20 MB; total shared art <= 70 MB.
            long iconBytes = 0;
            long allBytes = 0;
            foreach (var rel in EnumerateProduced())
            {
                var size = new FileInfo(Abs(rel)).Length;
                allBytes += size;
                if (rel.Contains("icons/") || rel.Contains("/eq/"))
                {
                    iconBytes += size;
                }
            }
            Assert.Less(iconBytes, 20L * 1024 * 1024, "icon group exceeds 20 MB");
            Assert.Less(allBytes, 70L * 1024 * 1024, "shared art exceeds 70 MB");
        }

        private static List<string> EnumerateProduced()
        {
            var result = new List<string>();
            foreach (var root in new[] { UiRoot, ItemsRoot, VfxRoot, StyleRefRoot })
            {
                var abs = Abs(root);
                if (!Directory.Exists(abs))
                {
                    continue;
                }
                foreach (var f in Directory.GetFiles(abs, "*", SearchOption.AllDirectories))
                {
                    var ext = Path.GetExtension(f).ToLowerInvariant();
                    if (ext == ".meta" || ext == ".importmeta" || ext == ".json"
                        || ext == ".md" || ext == ".txt")
                    {
                        continue;
                    }
                    if (f.Contains("/Editor/"))
                    {
                        continue;
                    }
                    result.Add(Path.GetRelativePath(ProjectRoot, f)
                                .Replace('\\', '/'));
                }
            }
            return result;
        }

        private static List<Vector3> LoadPalette(string abs)
        {
            var json = File.ReadAllText(abs);
            var palette = new List<Vector3>();
            // palette.json: {"colors": [{"hex":"#RRGGBB"}, ...]}
            foreach (System.Text.RegularExpressions.Match m in
                    System.Text.RegularExpressions.Regex.Matches(
                        json, "#[0-9A-Fa-f]{6}"))
            {
                var s = m.Value.Substring(1);
                var c = new Color32(
                    byte.Parse(s.Substring(0, 2), System.Globalization.NumberStyles.HexNumber),
                    byte.Parse(s.Substring(2, 2), System.Globalization.NumberStyles.HexNumber),
                    byte.Parse(s.Substring(4, 2), System.Globalization.NumberStyles.HexNumber),
                    255);
                palette.Add(CieLab.ToLab(c));
            }
            return palette;
        }

        // ---- minimal TTF reader (cmap + hhea + glyf bbox) ----------------------

        private struct FontMetrics
        {
            public int UnitsPerEm;
            public int Ascent;
            public int Descent;
            public int UsWinAscent;
            public int UsWinDescent;
            public bool[] Loca32; // true => 'loca' entries are uint32
        }

        private static (string Tag, uint Offset, uint Length)[] Tables(string abs)
        {
            var b = File.ReadAllBytes(abs);
            var nTables = (b[4] << 8) | b[5];
            var list = new List<(string, uint, uint)>();
            for (var i = 0; i < nTables; i++)
            {
                var off = 12 + 16 * i;
                var tag = Encoding.ASCII.GetString(b, off, 4);
                var offset = (uint)((b[off + 8] << 24) | (b[off + 9] << 16) | (b[off + 10] << 8) | b[off + 11]);
                var length = (uint)((b[off + 12] << 24) | (b[off + 13] << 16) | (b[off + 14] << 8) | b[off + 15]);
                list.Add((tag, offset, length));
            }
            return list.ToArray();
        }

        private static byte[] TableBytes(byte[] file, (string Tag, uint Offset, uint Length)[] tables, string tag)
        {
            foreach (var t in tables)
            {
                if (t.Tag == tag)
                {
                    var sub = new byte[t.Length];
                    System.Buffer.BlockCopy(file, (int)t.Offset, sub, 0, (int)t.Length);
                    return sub;
                }
            }
            Assert.Fail("font missing table " + tag);
            return null!;
        }

        private static FontMetrics ParseFontMetrics(string abs)
        {
            var file = File.ReadAllBytes(abs);
            var tables = Tables(abs);
            var head = TableBytes(file, tables, "head");
            var hhea = TableBytes(file, tables, "hhea");
            var os2 = TableBytes(file, tables, "OS/2");
            var m = new FontMetrics
            {
                UnitsPerEm = (head[18] << 8) | head[19],
                Ascent = (short)((hhea[4] << 8) | hhea[5]),
                Descent = (short)((hhea[6] << 8) | hhea[7]),
                UsWinAscent = (os2[74] << 8) | os2[75],
                UsWinDescent = (os2[76] << 8) | os2[77],
                Loca32 = new bool[] { head[51] != 0 },
            };
            return m;
        }

        private static Dictionary<uint, uint> ParseCmap(string abs)
        {
            var file = File.ReadAllBytes(abs);
            var cmap = TableBytes(file, Tables(abs), "cmap");
            var nSub = (cmap[2] << 8) | cmap[3];
            // prefer format 12, else format 4 (platform 3 / encoding 10 or 1)
            uint subOff = 0;
            int fmt4Off = 0;
            for (var i = 0; i < nSub; i++)
            {
                var platform = (cmap[4 + 8 * i] << 8) | cmap[5 + 8 * i];
                var enc = (cmap[6 + 8 * i] << 8) | cmap[7 + 8 * i];
                var off = (uint)((cmap[8 + 8 * i] << 24) | (cmap[9 + 8 * i] << 16)
                                | (cmap[10 + 8 * i] << 8) | cmap[11 + 8 * i]);
                var fmt = (cmap[off] << 8) | cmap[off + 1];
                if (fmt == 12)
                {
                    subOff = off;
                    break;
                }
                if (fmt == 4 && platform == 3)
                {
                    fmt4Off = (int)off;
                }
            }
            var map = new Dictionary<uint, uint>();
            if (subOff != 0)
            {
                var o = (int)subOff;
                var nGroups = (int)((cmap[o + 12] << 24) | (cmap[o + 13] << 16)
                                    | (cmap[o + 14] << 8) | cmap[o + 15]);
                for (var g = 0; g < nGroups; g++)
                {
                    var go = o + 16 + 12 * g;
                    var start = (uint)((cmap[go] << 24) | (cmap[go + 1] << 16) | (cmap[go + 2] << 8) | cmap[go + 3]);
                    var end = (uint)((cmap[go + 4] << 24) | (cmap[go + 5] << 16) | (cmap[go + 6] << 8) | cmap[go + 7]);
                    var gid0 = (uint)((cmap[go + 8] << 24) | (cmap[go + 9] << 16) | (cmap[go + 10] << 8) | cmap[go + 11]);
                    for (var cp = start; cp <= end && cp - start < 4096; cp++)
                    {
                        map[cp] = gid0 + (cp - start);
                    }
                }
                return map;
            }
            Assert.IsTrue(fmt4Off != 0, "no cmap format 4 or 12");
            var f4 = fmt4Off;
            var segCount = ((cmap[f4 + 6] << 8) | cmap[f4 + 7]) / 2;
            var endBase = f4 + 14;
            var startBase = endBase + 2 * segCount + 2;
            var deltaBase = startBase + 2 * segCount;
            var rangeBase = deltaBase + 2 * segCount;
            for (var s = 0; s < segCount; s++)
            {
                var end = (uint)((cmap[endBase + 2 * s] << 8) | cmap[endBase + 2 * s + 1]);
                var start = (uint)((cmap[startBase + 2 * s] << 8) | cmap[startBase + 2 * s + 1]);
                var delta = (short)((cmap[deltaBase + 2 * s] << 8) | cmap[deltaBase + 2 * s + 1]);
                var ro = (cmap[rangeBase + 2 * s] << 8) | cmap[rangeBase + 2 * s + 1];
                for (var cp = start; cp <= end && cp != 0xFFFF; cp++)
                {
                    uint gid;
                    if (ro == 0)
                    {
                        gid = (uint)((cp + delta) & 0xFFFF);
                    }
                    else
                    {
                        var gi = rangeBase + 2 * s + ro + 2 * (int)(cp - start);
                        if (gi + 1 >= cmap.Length)
                        {
                            continue;
                        }
                        var g = (cmap[gi] << 8) | cmap[gi + 1];
                        gid = g == 0 ? 0 : (uint)((g + delta) & 0xFFFF);
                    }
                    if (gid != 0)
                    {
                        map[cp] = gid;
                    }
                }
            }
            return map;
        }

        private static int ParseGlyphYMax(string abs, FontMetrics m, uint glyphId)
        {
            var file = File.ReadAllBytes(abs);
            var tables = Tables(abs);
            var loca = TableBytes(file, tables, "loca");
            var glyf = TableBytes(file, tables, "glyf");
            int start;
            int end;
            if (m.Loca32[0])
            {
                var i = (int)glyphId * 4;
                start = (loca[i] << 24) | (loca[i + 1] << 16) | (loca[i + 2] << 8) | loca[i + 3];
                end = (loca[i + 4] << 24) | (loca[i + 5] << 16) | (loca[i + 6] << 8) | loca[i + 7];
            }
            else
            {
                var i = (int)glyphId * 2;
                start = ((loca[i] << 8) | loca[i + 1]) * 2;
                end = ((loca[i + 2] << 8) | loca[i + 3]) * 2;
            }
            if (end <= start || start + 10 > glyf.Length)
            {
                return 0; // empty glyph
            }
            return (short)((glyf[start + 8] << 8) | glyf[start + 9]);
        }
    }
}
