using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Text.RegularExpressions;
using NUnit.Framework;
using CutoutGateImpl =
    ThinhThan.Core.Assets.Editor.AssetProduction.CutoutQualityGate;
using VolumeDepthGateImpl =
    ThinhThan.Core.Assets.Editor.AssetProduction.VolumeDepthGate;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEditor;
using UnityEngine;
using UnityEngine.U2D.Animation;

namespace ThinhThan.Tests.EditMode.CosmeticArtCoverage
{
    // IMP-074 Cosmetic Presentation Art coverage. Verifies the release
    // cosmetic universe end to end: every release cosmetic ID enumerated
    // from cosmetic_catalog.md + atlas_catalog.md resolves to exactly one
    // presentation (text-only title, intentional shared art, or a
    // produced texture/PSB), every produced file clears the Cutout and
    // Volume & Depth gates, import settings are canonical, the
    // cosmetics.json provenance fragment covers every shipped file,
    // AI_CREATED rows match the owner-setup tool, appearances bind the
    // shared skeleton through fixed PSB layer names, the cosmetics Style
    // Pack + palette gate pass, cultural entities carry folklore cards,
    // and the cultural review file records evidence per named cosmetic.
    public class CosmeticArtCoverageTests
    {
        private const string CosmeticsRoot = "Assets/Art/Cosmetics";
        private const string MapAssetPath =
            CosmeticsRoot + "/cosmetic_presentation_map.json";
        private const string StylePackRel =
            "Assets/Art/StyleRef/cosmetics";
        private const string FragmentRel =
            "client/Assets/Art/Provenance/fragments/cosmetics.json";
        private const string TermsRel =
            "client/Assets/Art/Provenance/terms/cosmetics";
        private const string CulturalReviewRel =
            "client/Assets/Art/Provenance/cultural_review.md";
        private const string SpriteLibRel =
            CosmeticsRoot + "/asset.cosmetic.appearance.spriteLib.asset";
        private const string SharedSkeletonRel =
            "Assets/Art/Actors/Players/asset.class.players.skeleton.asset";

        private const int IconTex = 128;   // 64x64 ref at 2x
        private const int CellW = 192;     // 96x128 ref at 2x
        private const int CellH = 256;

        // Owner Setup tool string for every AI_CREATED record
        // (audit_gates.md Owner Setup + technology_versions.md Content
        // production tools).
        private const string OwnerToolName = "AI Horde";
        private const string OwnerToolModel = "AlbedoBase XL (SDXL)";

        private static readonly string[] PsbLayers = AnimationContract.PsbLayers;

        private static readonly string[] ValidSlots =
        {
            "title", "title_glow", "frame", "nameplate", "appearance",
            "weapon_trail", "aura", "emote", "character_shrine",
            "guild_stone_inscription", "guild_crest", "guild_banner",
            "guild_shrine",
        };

        private static readonly string[] PowerKeys =
        {
            "stat", "stats", "attack", "defense", "hp", "damage",
            "collider", "hitbox", "speed", "element", "formation",
            "equipment_slot", "power", "reward", "drop_rate", "buff",
        };

        // ---------- helpers -------------------------------------------------

        private static string RepoRoot()
        {
            return Path.GetFullPath(
                Path.Combine(Application.dataPath, "..", ".."));
        }

        private static string Abs(string rel)
        {
            return Path.GetFullPath(Path.Combine(RepoRoot(), rel));
        }

        [System.Serializable]
        private sealed class MapDoc
        {
            public int schema_version;
            public string fragment = "";
            public List<Entry>? entries;
        }

        [System.Serializable]
        private sealed class Entry
        {
            public string cosmetic_id = "";
            public string kind = "";
            public string slot = "";
            public string display = "";
            public int seed;
            public string presentation = "";
            public string asset_key = "";
            public string icon = "";
            public string target = "";
            public string tint = "";
            public string file = "";
            public string icon_file = "";
            public List<string>? layers;
        }

        // JsonUtility cannot deserialise Dictionary<string, T>; the map
        // stores shared_targets as an object, so it is parsed with a
        // minimal regex pass instead.
        private sealed class SharedTarget
        {
            public string File = "";
        }

        private static MapDoc LoadMap()
        {
            var abs = Abs("client/" + MapAssetPath);
            Assert.IsTrue(File.Exists(abs),
                "presentation map missing: " + MapAssetPath);
            var doc = JsonUtility.FromJson<MapDoc>(File.ReadAllText(abs));
            Assert.NotNull(doc, "presentation map failed to parse");
            Assert.NotNull(doc!.entries, "presentation map entries missing");
            return doc;
        }

        private static Dictionary<string, SharedTarget>? _shared;

        private static Dictionary<string, SharedTarget> SharedTargets()
        {
            if (_shared != null)
            {
                return _shared;
            }
            var abs = Abs("client/" + MapAssetPath);
            var text = File.ReadAllText(abs);
            var result = new Dictionary<string, SharedTarget>();
            var m = Regex.Match(
                text,
                "\"shared_targets\"\\s*:\\s*\\{",
                RegexOptions.None);
            Assert.IsTrue(m.Success, "shared_targets block missing");
            var section = text.Substring(m.Index);
            var em = Regex.Match(
                section,
                "\"([^\"]+)\"\\s*:\\s*\\{\\s*\"file\"\\s*:\\s*\"([^\"]+)\"");
            while (em.Success)
            {
                result[em.Groups[1].Value] =
                    new SharedTarget { File = em.Groups[2].Value };
                em = em.NextMatch();
            }
            return result;
        }

        // All release cosmetic IDs: cosmetic_catalog.md + atlas_catalog.md
        // backticked `cosmetic.*` tokens (the compiler-expanded seasonal
        // atlas titles are concrete IDs in atlas_catalog.md).
        private static List<string> CatalogIds()
        {
            var ids = new SortedSet<string>();
            foreach (var rel in new[]
                {
                    "docs/07_content/cosmetic_catalog.md",
                    "docs/07_content/atlas_catalog.md",
                })
            {
                var text = File.ReadAllText(Abs(rel));
                foreach (Match m in Regex.Matches(
                    text, "`(cosmetic\\.[a-z0-9_.]+)`"))
                {
                    ids.Add(m.Groups[1].Value);
                }
            }
            return ids.ToList();
        }

        private static string ProducedFileFor(Entry e)
        {
            return "client/" + CosmeticsRoot + "/" + e.file;
        }

        private static string IconFileFor(Entry e)
        {
            if (!string.IsNullOrEmpty(e.icon_file))
            {
                return "client/" + CosmeticsRoot + "/" + e.icon_file;
            }
            return IconPathFor(e);
        }

        // Resolves the texture that carries this cosmetic's .icon facet:
        // its own produced file for asset entries, the shared target file
        // for text/shared entries.
        private static string IconPathFor(Entry e)
        {
            var shared = SharedTargets();
            var refName = e.icon != null && e.icon.StartsWith("shared:")
                ? e.icon.Substring("shared:".Length)
                : (e.target != null && e.target.StartsWith("shared:")
                    ? e.target.Substring("shared:".Length)
                    : null);
            if (refName != null)
            {
                Assert.IsTrue(shared.ContainsKey(refName),
                    e.cosmetic_id + " unknown shared target " + refName);
                return "client/" + CosmeticsRoot + "/" + shared[refName].File;
            }
            return "client/" + CosmeticsRoot + "/" + e.file;
        }

        private static Color32[] LoadPng(string absPath, out int w, out int h)
        {
            var tex = new Texture2D(2, 2, TextureFormat.RGBA32, false);
            Assert.IsTrue(
                ImageConversion.LoadImage(tex, File.ReadAllBytes(absPath)),
                "failed to decode " + absPath);
            w = tex.width;
            h = tex.height;
            var px = tex.GetPixels32();
            Object.DestroyImmediate(tex);
            return px;
        }

        private static List<GateViolation> GateOne(string repoPath)
        {
            var violations = new List<GateViolation>();
            var assetPath = repoPath.Substring("client/".Length);
            var abs = Abs(repoPath);
            var importer = AssetImporter.GetAtPath(assetPath);
            var meta = ImportMetadata.Parse(
                importer != null ? importer.userData : null);
            if (meta.AssetClass == null)
            {
                violations.Add(new GateViolation(
                    "asset_class", "import metadata missing for " + repoPath));
                return violations;
            }
            var input = PresentationSizing.ToCutoutInput(
                null!, 0, 0, meta.AssetClass.Value, meta, null);
            violations.AddRange(CutoutGateImpl.ValidateFile(abs, input));
            if (input.Width > 0)
            {
                var vinput = PresentationSizing.ToVolumeInput(
                    input.Pixels, input.Width, input.Height,
                    meta.AssetClass.Value, null);
                violations.AddRange(VolumeDepthGateImpl.ValidatePixels(vinput));
            }
            return violations;
        }

        private static string[] ReadPsbLayerNames(string absPath)
        {
            var data = File.ReadAllBytes(absPath);
            var names = new List<string>();
            long pos = 26;
            var colorLen = ReadU32(data, ref pos);
            pos += colorLen;
            var resLen = ReadU32(data, ref pos);
            pos += resLen;
            var maskLen = ReadU32(data, ref pos);
            var maskEnd = pos + maskLen;
            if (pos >= maskEnd)
            {
                return names.ToArray();
            }
            var layerInfoLen = ReadU32(data, ref pos);
            var layerInfoEnd = pos + layerInfoLen;
            var count = (short)ReadU16(data, ref pos);
            if (count < 0)
            {
                count = (short)-count;
            }
            for (var i = 0; i < count && pos + 18 <= data.Length; i++)
            {
                pos += 16;
                var channels = ReadU16(data, ref pos);
                pos += channels * 6 + 8 + 4;
                var extraLen = ReadU32(data, ref pos);
                var extraEnd = pos + extraLen;
                var maskDataLen = ReadU32(data, ref pos);
                pos += maskDataLen;
                var blendLen = ReadU32(data, ref pos);
                pos += blendLen;
                var nameLen = data[pos];
                pos++;
                var name = Encoding.ASCII.GetString(
                    data, (int)pos, nameLen);
                names.Add(name);
                pos = extraEnd;
            }
            return names.ToArray();
        }

        private static uint ReadU32(byte[] d, ref long pos)
        {
            var v = ((uint)d[pos] << 24) | ((uint)d[pos + 1] << 16)
                | ((uint)d[pos + 2] << 8) | d[pos + 3];
            pos += 4;
            return v;
        }

        private static uint ReadU16(byte[] d, ref long pos)
        {
            var v = (uint)((d[pos] << 8) | (d[pos + 1]));
            pos += 2;
            return v;
        }

        private static List<AssetSourceRow> LoadFragment()
        {
            var abs = Abs(FragmentRel);
            Assert.IsTrue(File.Exists(abs),
                "provenance fragment missing: " + FragmentRel);
            var register = AssetSourceRegisterIO.Load(abs);
            Assert.NotNull(register, "provenance fragment failed to parse");
            return register!.assets;
        }

        private static AssetSourceRow? RowFor(
            List<AssetSourceRow> rows, string filePath)
        {
            foreach (var r in rows)
            {
                if (r.file_path == filePath)
                {
                    return r;
                }
            }
            return null;
        }

        // Every produced texture/PSB under Cosmetics/ + the StyleRef
        // pack files, for provenance coverage.
        private static List<string> ProducedRepoPaths(MapDoc doc)
        {
            var produced = new List<string>();
            var shared = SharedTargets();
            foreach (var t in shared.Values)
            {
                produced.Add("client/" + CosmeticsRoot + "/" + t.File);
            }
            foreach (var e in doc.entries!)
            {
                if (e.presentation == "asset")
                {
                    produced.Add(ProducedFileFor(e));
                    if (e.kind == "appearance")
                    {
                        produced.Add(IconFileFor(e));
                    }
                }
            }
            var styleref = Abs("client/" + StylePackRel);
            if (Directory.Exists(styleref))
            {
                foreach (var f in Directory.GetFiles(
                    styleref, "*.png", SearchOption.AllDirectories))
                {
                    produced.Add("client/" + f.Substring(
                        Path.GetFullPath(Application.dataPath + "/..").Length + 1)
                        .Replace('\\', '/'));
                }
            }
            return produced;
        }

        // ---------- tests ---------------------------------------------------

        // Every release cosmetic ID resolves to exactly one presentation
        // mapping: text-only titles declare their display string, shared
        // entries resolve to a real shared file, asset entries point at a
        // produced file under Cosmetics/.
        [Test]
        public void TestCosmeticIdPresentationCoverage()
        {
            var doc = LoadMap();
            var byId = new Dictionary<string, Entry>();
            foreach (var e in doc.entries!)
            {
                Assert.IsFalse(byId.ContainsKey(e.cosmetic_id),
                    "duplicate map entry " + e.cosmetic_id);
                byId[e.cosmetic_id] = e;
            }
            var ids = CatalogIds();
            Assert.GreaterOrEqual(ids.Count, 290,
                "catalog ID enumeration incomplete");
            Assert.AreEqual(ids.Count, byId.Count,
                "map must cover exactly the release cosmetic ID set");
            var shared = SharedTargets();
            foreach (var cid in ids)
            {
                Assert.IsTrue(byId.TryGetValue(cid, out var e),
                    "no presentation mapping for " + cid);
                Assert.IsNotEmpty(e!.display, cid + " display missing");
                Assert.IsTrue(ValidSlots.Contains(e.slot),
                    cid + " invalid slot " + e.slot);
                switch (e.presentation)
                {
                    case "text":
                        Assert.IsTrue(
                            e.icon != null && e.icon.StartsWith("shared:"),
                            cid + " text title needs shared icon mapping");
                        break;
                    case "shared":
                        Assert.IsTrue(
                            e.target != null && e.target.StartsWith("shared:"),
                            cid + " shared entry needs shared target");
                        break;
                    case "asset":
                        Assert.IsNotEmpty(e.file,
                            cid + " asset entry needs produced file");
                        break;
                    default:
                        Assert.Fail(cid + " unknown presentation "
                            + e.presentation);
                        break;
                }
                var icon = IconPathFor(e);
                Assert.IsTrue(File.Exists(Abs(icon)),
                    cid + " icon texture missing " + icon);
                if (e.presentation == "asset")
                {
                    var producedAbs = Abs(ProducedFileFor(e));
                    Assert.IsTrue(File.Exists(producedAbs),
                        cid + " produced file missing "
                            + ProducedFileFor(e));
                }
            }
            foreach (var t in shared)
            {
                var p = "client/" + CosmeticsRoot + "/" + t.Value.File;
                Assert.IsTrue(File.Exists(Abs(p)),
                    "shared target missing " + p);
            }
        }

        // Equip-slot preview: every cosmetic resolves a previewable
        // presentation for its slot and every slot named by the catalog
        // is exercised by at least one release ID.
        [Test]
        public void TestEquipSlotPreviewResolves()
        {
            var doc = LoadMap();
            var seen = new HashSet<string>();
            foreach (var e in doc.entries!)
            {
                seen.Add(e.slot);
                var icon = IconPathFor(e);
                var abs = Abs(icon);
                Assert.IsTrue(File.Exists(abs),
                    e.cosmetic_id + " preview texture missing " + icon);
                LoadPng(abs, out var w, out var h);
                Assert.AreEqual(IconTex, w,
                    e.cosmetic_id + " preview width");
                Assert.AreEqual(IconTex, h,
                    e.cosmetic_id + " preview height");
            }
            foreach (var slot in new[]
                {
                    "title", "title_glow", "frame", "nameplate",
                    "appearance", "weapon_trail", "aura", "emote",
                    "character_shrine", "guild_stone_inscription",
                    "guild_crest", "guild_banner", "guild_shrine",
                })
            {
                Assert.IsTrue(seen.Contains(slot),
                    "slot " + slot + " has no cosmetic");
            }
        }

        // Non-power invariant: the map carries no gameplay fields and no
        // produced cosmetic asset can alter stats/collision/equipment.
        [Test]
        public void TestNonPowerInvariant()
        {
            var text = File.ReadAllText(Abs("client/" + MapAssetPath));
            foreach (var key in PowerKeys)
            {
                Assert.IsFalse(
                    Regex.IsMatch(
                        text, "\"" + Regex.Escape(key) + "\"\\s*:"),
                    "map contains power field " + key);
            }
            var doc = LoadMap();
            foreach (var e in doc.entries!)
            {
                if (e.kind != "appearance")
                {
                    continue;
                }
                var prefab = AssetDatabase.LoadAssetAtPath<GameObject>(
                    IconFileFor(e).Replace(".icon.png", ".prefab")
                        .Substring("client/".Length));
                Assert.NotNull(prefab,
                    e.cosmetic_id + " appearance prefab missing");
                var colliders = prefab!.GetComponentsInChildren<Collider2D>(true);
                Assert.AreEqual(0, colliders.Length,
                    e.cosmetic_id + " prefab carries a collider");
                var colliders3d = prefab.GetComponentsInChildren<Collider>(true);
                Assert.AreEqual(0, colliders3d.Length,
                    e.cosmetic_id + " prefab carries a 3D collider");
                var layers = e.layers;
                Assert.NotNull(layers, e.cosmetic_id + " layers missing");
                Assert.IsFalse(layers!.Contains("weapon"),
                    e.cosmetic_id + " must not replace the weapon layer "
                        + "(equipment identity invariant)");
            }
        }

        // Import settings: cosmetic icons import as UI_ART sprites at
        // 200 PPU / centred pivot; appearance cells import as
        // COSMETIC_APPEARANCE at 100 PPU / Bottom Center pivot.
        [Test]
        public void TestImportScaleCellPivot()
        {
            var doc = LoadMap();
            foreach (var e in doc.entries!)
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var iconRel = IconFileFor(e).Substring("client/".Length);
                var importer = AssetImporter.GetAtPath(iconRel)
                    as TextureImporter;
                Assert.NotNull(importer, "no TextureImporter for " + iconRel);
                var meta = ImportMetadata.Parse(importer!.userData);
                if (e.kind == "appearance")
                {
                    Assert.AreEqual(PresentationAssetClass.CosmeticAppearance,
                        meta.AssetClass, e.cosmetic_id + " asset_class");
                    Assert.AreEqual(100f, importer.spritePixelsPerUnit, 0.01f,
                        e.cosmetic_id + " icon PPU");
                    Assert.IsEmpty(AnimationContract.CheckPivot(
                        importer.spritePivot),
                        e.cosmetic_id + " icon pivot must be Bottom Center");
                }
                else
                {
                    Assert.AreEqual(PresentationAssetClass.UiArt,
                        meta.AssetClass, e.cosmetic_id + " asset_class");
                    Assert.AreEqual(200f, importer.spritePixelsPerUnit, 0.01f,
                        e.cosmetic_id + " icon PPU");
                }
                Assert.AreEqual(SpriteImportMode.Single,
                    importer.spriteImportMode, e.cosmetic_id + " sprite mode");
                Assert.IsFalse(importer.mipmapEnabled,
                    e.cosmetic_id + " mipmaps");
            }
        }

        // Cutout Gate + Volume & Depth Gate over every produced texture:
        // UI_ART runs the UI subset, COSMETIC_APPEARANCE runs the full
        // pair. Zero violations.
        [Test]
        public void TestCutoutAndVolumeGates()
        {
            var doc = LoadMap();
            var produced = ProducedRepoPaths(doc)
                .Where(p => p.EndsWith(".png", System.StringComparison.Ordinal))
                .ToList();
            Assert.IsNotEmpty(produced, "no produced textures");
            foreach (var p in produced)
            {
                var violations = GateOne(p);
                Assert.IsEmpty(violations,
                    p + " gate violations: "
                        + string.Join(";", violations.Select(v => v.Rule
                            + "@" + v.Detail)));
            }
        }

        // Icons are authored at exactly 2x (128x128) and appearance cells
        // at 192x256.
        [Test]
        public void TestTexturesAreExactly2x()
        {
            var doc = LoadMap();
            foreach (var e in doc.entries!)
            {
                var icon = IconPathFor(e);
                LoadPng(Abs(icon), out var w, out var h);
                Assert.AreEqual(IconTex, w, e.cosmetic_id + " icon width");
                Assert.AreEqual(IconTex, h, e.cosmetic_id + " icon height");
                if (e.kind == "appearance")
                {
                    var cellAbs = Abs(ProducedFileFor(e)
                        .Replace(".psb", ".png"));
                    if (File.Exists(cellAbs))
                    {
                        LoadPng(cellAbs, out var cw, out var ch);
                        Assert.AreEqual(CellW, cw,
                            e.cosmetic_id + " cell width");
                        Assert.AreEqual(CellH, ch,
                            e.cosmetic_id + " cell height");
                    }
                }
            }
        }

        [Test]
        public void TestNoPlaceholder()
        {
            var doc = LoadMap();
            foreach (var e in doc.entries!)
            {
                var icon = IconPathFor(e);
                var bytes = File.ReadAllBytes(Abs(icon));
                Assert.AreEqual(0x89, bytes[0],
                    e.cosmetic_id + " icon not a PNG");
                Assert.Greater(bytes.Length, 2000,
                    e.cosmetic_id + " icon is degenerate (<2KB)");
                if (e.kind == "appearance")
                {
                    var psb = Abs(ProducedFileFor(e));
                    Assert.AreEqual(0x38, File.ReadAllBytes(psb)[0],
                        e.cosmetic_id + " not a PSB (8BPS signature)");
                    Assert.NotNull(
                        AssetDatabase.LoadAssetAtPath<GameObject>(
                            IconFileFor(e)
                                .Replace(".icon.png", ".prefab")
                                .Substring("client/".Length)),
                        e.cosmetic_id + " prefab missing");
                }
            }
        }

        // Provenance coverage: every shipped file has a fragment row and
        // the recorded final_sha256 matches the file on disk.
        [Test]
        public void TestProvenanceHash()
        {
            var rows = LoadFragment();
            Assert.IsNotEmpty(rows, "provenance fragment has no rows");
            var doc = LoadMap();
            var produced = ProducedRepoPaths(doc);
            foreach (var filePath in produced)
            {
                var row = RowFor(rows, filePath);
                Assert.NotNull(row, "no provenance row for " + filePath);
                var abs = Abs(filePath);
                Assert.IsTrue(File.Exists(abs),
                    "row file missing " + filePath);
                using (var sha = System.Security.Cryptography.SHA256.Create())
                {
                    var hash = System.BitConverter.ToString(
                        sha.ComputeHash(File.ReadAllBytes(abs)))
                        .Replace("-", "").ToLowerInvariant();
                    Assert.AreEqual(hash, row!.final_sha256,
                        "final hash mismatch " + filePath);
                }
                Assert.AreEqual("PENDING", row.review_state,
                    filePath + " review_state must start PENDING");
            }
        }

        [Test]
        public void TestAiCreatedToolMatchesOwnerSetup()
        {
            var rows = LoadFragment();
            var aiRows = rows.Where(r => r.source_kind == "AI_CREATED")
                .ToArray();
            Assert.IsNotEmpty(aiRows, "no AI_CREATED rows");
            foreach (var r in aiRows)
            {
                Assert.NotNull(r.generation_record,
                    r.file_path + " AI_CREATED row needs generation_record");
                var g = r.generation_record!;
                Assert.IsNotNull(g.tool, r.file_path + " tool missing");
                Assert.IsTrue(g.tool!.Contains(OwnerToolName),
                    r.file_path + " tool '" + g.tool + "' != owner tool");
                Assert.AreEqual(OwnerToolModel, g.model_id,
                    r.file_path + " model mismatch");
                Assert.IsNotEmpty(g.prompt, r.file_path + " prompt missing");
                Assert.IsNotEmpty(g.terms_snapshot_sha256,
                    r.file_path + " terms snapshot missing");
                var snap = Abs(TermsRel + "/"
                    + g.terms_snapshot_sha256 + ".txt");
                Assert.IsTrue(File.Exists(snap),
                    "terms snapshot missing " + snap);
            }
        }

        // ADR-0076: cosmetic appearance PSBs use only the fixed layer
        // names, bind the shared class skeleton, and every appearance
        // cosmetic id is a label under each layer it replaces inside the
        // shared cosmetics Sprite Library.
        [Test]
        public void TestCosmeticSpriteLibraryBinding()
        {
            var doc = LoadMap();
            var appearances = doc.entries!
                .Where(e => e.kind == "appearance").ToList();
            Assert.AreEqual(8, appearances.Count,
                "expected 8 appearance cosmetics");
            var skeleton = AssetDatabase.LoadAssetAtPath<Object>(
                SharedSkeletonRel);
            Assert.NotNull(skeleton, "shared skeleton asset missing");
            var lib = AssetDatabase.LoadAssetAtPath<SpriteLibraryAsset>(
                SpriteLibRel);
            Assert.NotNull(lib, "cosmetics sprite library missing");
            var fixedNames = new HashSet<string>(PsbLayers);
            foreach (var e in appearances)
            {
                var psbAbs = Abs(ProducedFileFor(e));
                var layers = ReadPsbLayerNames(psbAbs);
                Assert.IsNotEmpty(layers,
                    e.cosmetic_id + " PSB has no layers");
                var declared = new HashSet<string>(e.layers!);
                var have = new HashSet<string>(layers);
                Assert.IsTrue(have.SetEquals(declared),
                    e.cosmetic_id + " PSB layers "
                        + string.Join(",", layers)
                        + " != declared " + string.Join(",", declared));
                Assert.IsTrue(declared.IsSubsetOf(fixedNames),
                    e.cosmetic_id + " uses a non-fixed layer name");
                var psbImporter = AssetImporter.GetAtPath(
                    ProducedFileFor(e).Substring("client/".Length));
                Assert.NotNull(psbImporter,
                    e.cosmetic_id + " no importer");
                var so = new SerializedObject(psbImporter!);
                var prop = so.FindProperty("m_SkeletonAssetReferenceID");
                Assert.NotNull(prop,
                    e.cosmetic_id + " importer lacks skeleton binding");
                Assert.AreEqual(
                    AssetDatabase.AssetPathToGUID(SharedSkeletonRel),
                    prop!.stringValue,
                    e.cosmetic_id + " bound to wrong skeleton");
                foreach (var layer in e.layers!)
                {
                    var sprite = lib!.GetSprite(layer, e.cosmetic_id);
                    Assert.NotNull(sprite,
                        e.cosmetic_id + " no sprite under category "
                            + layer);
                }
            }
        }

        [Test]
        public void TestCosmeticStylePackAndPalette()
        {
            var packAbs = Path.Combine(
                Application.dataPath, "Art", "StyleRef", "cosmetics");
            var packViolations = StylePackGate.CheckPackDir(packAbs);
            Assert.IsEmpty(packViolations,
                "style pack violations: "
                    + string.Join(";", packViolations.Select(v => v.Detail)));
            var palette = StylePackGate.ParsePaletteJson(
                File.ReadAllText(Path.Combine(packAbs, "palette.json")));
            Assert.NotNull(palette, "palette.json failed to parse");
            var doc = LoadMap();
            var produced = ProducedRepoPaths(doc)
                .Where(p => p.EndsWith(".png", System.StringComparison.Ordinal))
                .ToList();
            foreach (var p in produced)
            {
                var px = LoadPng(Abs(p), out var w, out var h);
                var violations = StylePackGate.CheckPalette(
                    px, w, h, null, palette!);
                Assert.IsEmpty(violations,
                    p + " palette gate violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestCosmeticFolkloreCards()
        {
            var rows = LoadFragment();
            var doc = LoadMap();
            var produced = ProducedRepoPaths(doc);
            foreach (var filePath in produced)
            {
                var row = RowFor(rows, filePath);
                Assert.NotNull(row, "no row for " + filePath);
                if (!row!.cultural_entity)
                {
                    continue;
                }
                Assert.NotNull(row.folklore_card,
                    filePath + " folklore_card missing");
                Assert.IsNotEmpty(row.folklore_card!.source_tales,
                    filePath + " source_tales empty");
                Assert.IsNotEmpty(row.folklore_card.regional_variants,
                    filePath + " regional_variants empty");
                Assert.IsNotEmpty(row.folklore_card.motifs_checked,
                    filePath + " motifs_checked empty");
            }
        }

        // Cultural review evidence: every cosmetic ID named by
        // cosmetic_catalog.md has a per-ID line in cultural_review.md.
        [Test]
        public void TestCulturalReviewEvidence()
        {
            var abs = Abs(CulturalReviewRel);
            Assert.IsTrue(File.Exists(abs),
                "cultural review missing: " + CulturalReviewRel);
            var text = File.ReadAllText(abs);
            var catText = File.ReadAllText(
                Abs("docs/07_content/cosmetic_catalog.md"));
            var named = new SortedSet<string>();
            foreach (Match m in Regex.Matches(
                catText, "`(cosmetic\\.[a-z0-9_.]+)`"))
            {
                named.Add(m.Groups[1].Value);
            }
            foreach (var cid in named)
            {
                Assert.IsTrue(text.Contains(cid),
                    "cultural_review.md missing entry for " + cid);
            }
        }
    }
}
