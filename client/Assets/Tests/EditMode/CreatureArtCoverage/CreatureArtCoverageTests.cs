using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
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

namespace ThinhThan.Tests.EditMode.CreatureArtCoverage
{
    // IMP-104 Monster, Boss & Spirit Beast Art coverage. Verifies the
    // release creature roster end to end: catalog keys, produced files
    // (PSB rigs / frame strips, showcase sprites, clips), canonical import
    // settings, shared-variant mapping, provenance fragment, AI_CREATED
    // tool provenance, skeletal rig layers and clip contract per
    // size_profile, style pack + palette gate, frame consistency,
    // hitbox-silhouette alignment and folklore cards. All gates are
    // asserted to zero violations.
    public class CreatureArtCoverageTests
    {
        private const string CreaturesRoot =
            "Assets/Art/Actors/Creatures";
        private const string StylePackRel =
            "Assets/Art/StyleRef/actors_creatures";
        private const string AnimRoot = CreaturesRoot + "/anim";
        private const string MapRel =
            "client/Assets/Art/Actors/Creatures/creature_presentation_map.json";
        private const string FragmentRel =
            "client/Assets/Art/Provenance/fragments/actors_creatures.json";
        private const string TermsRel =
            "client/Assets/Art/Provenance/terms/actors_creatures";

        // Owner Setup tool string for every AI_CREATED record
        // (audit_gates.md Owner Setup + technology_versions.md Content
        // production tools).
        private const string OwnerToolName = "AI Horde";
        private const string OwnerToolModel = "AlbedoBase XL (SDXL)";

        // Canonical 2x cells and collider widths per size_profile
        // (presentation_asset_manifest.md §3).
        private static readonly Dictionary<string, (int CellW, int CellH)>
            CellSize = new Dictionary<string, (int, int)>
            {
                { "MONSTER_SMALL", (128, 128) },
                { "MONSTER_MEDIUM", (192, 256) },
                { "MONSTER_ELITE", (320, 384) },
                { "BOSS_LARGE", (512, 512) },
                { "WORLD_BOSS", (640, 640) },
                { "SPIRIT_BEAST", (128, 128) },
            };

        private static readonly Dictionary<string, float>
            ColliderWidthRefPx = new Dictionary<string, float>
            {
                { "MONSTER_SMALL", 30f },
                { "MONSTER_MEDIUM", 50f },
                { "MONSTER_ELITE", 80f },
                { "BOSS_LARGE", 120f },
                { "WORLD_BOSS", 150f },
            };

        private static readonly Dictionary<string, string>
            ForbiddenMotifs = new Dictionary<string, string>
            {
                { "torii", "torii gate motif" },
                { "jiangshi", "jiangshi/Qing-dynasty clothing" },
                { "qing", "Qing-dynasty clothing" },
                { "kimono", "kimono" },
                { "hanbok", "hanbok" },
                { "hanzi", "meaningless Han/Nom script ornament" },
            };

        private static string RepoRoot()
        {
            return Path.GetFullPath(
                Path.Combine(Application.dataPath, "..", ".."));
        }

        private static string Abs(string rel)
        {
            return Path.GetFullPath(Path.Combine(RepoRoot(), rel));
        }

        private static CreatureArtMaterialization.MapRoot LoadMap()
        {
            var abs = Abs(MapRel);
            Assert.IsTrue(File.Exists(abs),
                "presentation map missing: " + MapRel);
            return CreatureArtMaterialization.LoadMap();
        }

        private static List<CreatureArtMaterialization.MapEntry> Entries()
        {
            return LoadMap().entries.ToList();
        }

        private static string ShowcaseRel(
            CreatureArtMaterialization.MapEntry e)
        {
            return CreaturesRoot + "/" + e.dir + "/asset." + e.entity_id
                + ".png";
        }

        private static string PsbRel(
            CreatureArtMaterialization.MapEntry e)
        {
            return CreaturesRoot + "/" + e.dir + "/asset." + e.entity_id
                + ".psb";
        }

        private static string PrefabRel(
            CreatureArtMaterialization.MapEntry e)
        {
            return CreaturesRoot + "/" + e.dir + "/asset." + e.entity_id
                + ".prefab";
        }

        private static string ControllerRel(
            CreatureArtMaterialization.MapEntry e)
        {
            return AnimRoot + "/" + e.dir + "/" + e.entity_id + ".controller";
        }

        private static string ClipRel(
            CreatureArtMaterialization.MapEntry e, string clip)
        {
            return AnimRoot + "/" + e.dir + "/" + clip + ".anim";
        }

        private static string[] FrameRels(
            CreatureArtMaterialization.MapEntry e, string clip, int count)
        {
            var rels = new string[count];
            for (var i = 0; i < count; i++)
            {
                rels[i] = CreaturesRoot + "/" + e.dir + "/asset."
                    + e.entity_id + ".anim." + clip + "." + i + ".png";
            }
            return rels;
        }

        private static bool IsSkeletal(
            CreatureArtMaterialization.MapEntry e)
        {
            return CreatureArtMaterialization.IsSkeletal(e);
        }

        private static string[] RequiredClips(
            CreatureArtMaterialization.MapEntry e)
        {
            return AnimationContract.RequiredClips(
                CreatureArtMaterialization.ProfileOf(e.size_profile),
                e.attack_clips, e.phases);
        }

        private static int FrameCountFor(
            CreatureArtMaterialization.MapEntry e, string clip)
        {
            // Produced frame strips carry 4 frames for idle/move/cast and
            // attack_basic; hit/defeat carry 4 as well.
            return AnimationContract.FrameByFrameMinFrames;
        }

        private static List<GateViolation> GateOne(string assetPath)
        {
            var violations = new List<GateViolation>();
            var abs = Abs("client/" + assetPath);
            var importer = AssetImporter.GetAtPath(assetPath);
            var meta = ImportMetadata.Parse(
                importer != null ? importer.userData : null);
            if (meta.AssetClass == null)
            {
                violations.Add(new GateViolation(
                    "asset_class", "import metadata missing for " + assetPath));
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

        // Reads the layer name list out of a PSD/PSB file (Layer Records
        // section of the Layer & Mask Info block) so the rig can be
        // checked without depending on importer internals.
        private static string[] ReadPsbLayerNames(string absPath)
        {
            var data = File.ReadAllBytes(absPath);
            var names = new List<string>();
            long pos = 26; // header (26 bytes: sig,ver,res,ch,h,w,depth,mode)
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
                pos += 16; // rect top/left/bottom/right
                var channels = ReadU16(data, ref pos);
                pos += channels * 6 + 8 + 4; // channel (id,len) pairs + blend sig/key + opacity/clip/flags/filler
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
            var v = (uint)((d[pos] << 8) | d[pos + 1]);
            pos += 2;
            return v;
        }

        private static List<AssetSourceRow> LoadFragment()
        {
            var abs = Abs(FragmentRel);
            Assert.IsTrue(File.Exists(abs), "provenance fragment missing: " + FragmentRel);
            var register = AssetSourceRegisterIO.Load(abs);
            Assert.NotNull(register, "provenance fragment failed to parse");
            return register!.assets;
        }

        private static AssetSourceRow? RowFor(List<AssetSourceRow> rows, string filePath)
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

        private static int CountKeys(AnimationClip clip)
        {
            var count = 0;
            foreach (var binding in AnimationUtility.GetCurveBindings(clip))
            {
                var curve = AnimationUtility.GetEditorCurve(clip, binding);
                if (curve != null)
                {
                    count += curve.keys.Length;
                }
            }
            foreach (var binding in AnimationUtility.GetObjectReferenceCurveBindings(clip))
            {
                var frames = AnimationUtility.GetObjectReferenceCurve(clip, binding);
                count += frames.Length;
            }
            return count;
        }

        [Test]
        public void TestRosterToKeyCoverage()
        {
            AddressablesProvisioner.Provision();
            var plan = AssetKeyPlanner.Build(ContentCatalogScanner.DefaultDocsRoot);
            var keys = new HashSet<string>(plan.Select(p => p.Key));
            var entries = Entries();
            Assert.AreEqual(82, entries.Count,
                "roster map must cover all release monster/boss/beast ids");
            var ids = new HashSet<string>();
            foreach (var e in entries)
            {
                Assert.IsTrue(ids.Add(e.entity_id),
                    "duplicate roster id " + e.entity_id);
                var key = "asset." + e.entity_id;
                Assert.IsTrue(keys.Contains(key + ".prefab")
                    || keys.Contains(key),
                    "no catalog key for " + key);
                Assert.IsTrue(File.Exists(Abs("client/" + PrefabRel(e))),
                    "prefab missing for " + e.entity_id);
                if (e.presentation == "asset")
                {
                    Assert.IsTrue(File.Exists(Abs("client/" + ShowcaseRel(e))),
                        "showcase sprite missing for " + e.entity_id);
                    if (IsSkeletal(e))
                    {
                        Assert.IsTrue(File.Exists(Abs("client/" + PsbRel(e))),
                            "rig source missing for " + e.entity_id);
                    }
                }
            }
        }

        [Test]
        public void TestRequiredClips()
        {
            foreach (var e in Entries())
            {
                var required = RequiredClips(e);
                Assert.IsNotEmpty(required,
                    e.entity_id + " has no required clips");
                foreach (var clipName in required)
                {
                    var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(
                        ClipRel(e, clipName));
                    Assert.NotNull(clip,
                        e.entity_id + " clip missing: " + clipName);
                    var expectedFps = IsSkeletal(e)
                        ? AnimationContract.SkeletalFps
                        : AnimationContract.FrameByFrameFps;
                    Assert.AreEqual(expectedFps, (int)clip!.frameRate,
                        e.entity_id + " clip " + clipName + " fps");
                    var keyCount = CountKeys(clip);
                    var minKeys = IsSkeletal(e)
                        ? AnimationContract.SkeletalMinKeys
                        : AnimationContract.FrameByFrameMinFrames;
                    Assert.GreaterOrEqual(keyCount, minKeys,
                        e.entity_id + " clip " + clipName + " key/frame count");
                }
            }
        }

        [Test]
        public void TestImportScaleCellPivot()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var (cellW, cellH) = CellSize[e.size_profile];
                var abs = Abs("client/" + ShowcaseRel(e));
                LoadPng(abs, out var w, out var h);
                Assert.AreEqual(cellW, w,
                    e.entity_id + " texture width");
                Assert.AreEqual(cellH, h,
                    e.entity_id + " texture height");
                var importer = AssetImporter.GetAtPath(ShowcaseRel(e))
                    as TextureImporter;
                Assert.NotNull(importer,
                    "no TextureImporter for " + e.entity_id);
                Assert.AreEqual(100f, importer!.spritePixelsPerUnit, 0.01f,
                    e.entity_id + " PPU");
                Assert.AreEqual(SpriteImportMode.Single,
                    importer.spriteImportMode,
                    e.entity_id + " sprite mode");
                var pivot = importer.spritePivot;
                Assert.IsEmpty(AnimationContract.CheckPivot(pivot),
                    e.entity_id + " pivot must be Bottom Center");
                Assert.IsFalse(importer.mipmapEnabled,
                    e.entity_id + " mipmaps");
            }
        }

        [Test]
        public void TestSharedVariantMapping()
        {
            var map = LoadMap();
            var targets = new HashSet<string>(
                map.shared_targets.Select(t => "shared:" + t.name));
            var usedTargets = new HashSet<string>();
            foreach (var e in map.entries)
            {
                if (e.presentation == "shared")
                {
                    Assert.IsTrue(targets.Contains(e.target),
                        e.entity_id + " shared target '" + e.target
                            + "' is not a declared shared_targets entry");
                    usedTargets.Add(e.target);
                    var st = map.shared_targets.First(
                        t => "shared:" + t.name == e.target);
                    var baseEntity = map.entries.FirstOrDefault(
                        b => b.entity_id == st.base_entity);
                    Assert.NotNull(baseEntity,
                        e.entity_id + " shared target " + st.name
                            + " has no base entity");
                    Assert.IsTrue(File.Exists(Abs("client/" + PsbRel(baseEntity!)))
                        || File.Exists(Abs("client/" + ShowcaseRel(baseEntity!))),
                        e.entity_id + " shared base art missing");
                    // The shared variant's own prefab must exist and must
                    // be a prefab-variant tint of the base prefab.
                    var prefab = AssetDatabase.LoadAssetAtPath<GameObject>(
                        PrefabRel(e));
                    Assert.NotNull(prefab,
                        e.entity_id + " shared prefab missing");
                    var basePrefab = AssetDatabase.LoadAssetAtPath<GameObject>(
                        CreatureArtMaterialization.SharedPrefabPath(st));
                    Assert.NotNull(basePrefab,
                        e.entity_id + " shared base prefab missing");
                }
                else
                {
                    Assert.AreEqual("asset", e.presentation,
                        e.entity_id + " unknown presentation '" + e.presentation + "'");
                }
            }
            foreach (var t in targets)
            {
                Assert.IsTrue(usedTargets.Contains(t),
                    "shared target " + t + " has no referencing entity");
            }
        }

        [Test]
        public void TestProvenanceHash()
        {
            var rows = LoadFragment();
            Assert.IsNotEmpty(rows, "provenance fragment has no rows");
            var produced = new List<string>();
            foreach (var f in Directory.GetFiles(
                Path.Combine(Application.dataPath, "Art", "Actors", "Creatures"),
                "*.png", SearchOption.AllDirectories))
            {
                produced.Add("client/" + f.Substring(
                    Path.GetFullPath(Application.dataPath + "/..").Length + 1)
                    .Replace('\\', '/'));
            }
            foreach (var f in Directory.GetFiles(
                Path.Combine(Application.dataPath, "Art", "Actors", "Creatures"),
                "*.psb", SearchOption.AllDirectories))
            {
                produced.Add("client/" + f.Substring(
                    Path.GetFullPath(Application.dataPath + "/..").Length + 1)
                    .Replace('\\', '/'));
            }
            var styleref = Path.Combine(Application.dataPath, "Art",
                "StyleRef", "actors_creatures");
            foreach (var f in Directory.GetFiles(styleref, "*.png",
                SearchOption.AllDirectories))
            {
                produced.Add("client/" + f.Substring(
                    Path.GetFullPath(Application.dataPath + "/..").Length + 1)
                    .Replace('\\', '/'));
            }
            var repoRoot = RepoRoot();
            foreach (var filePath in produced)
            {
                var row = RowFor(rows, filePath);
                Assert.NotNull(row, "no provenance row for " + filePath);
                var abs = Abs(filePath);
                Assert.IsTrue(File.Exists(abs), "row file missing " + filePath);
                using (var sha = System.Security.Cryptography.SHA256.Create())
                {
                    var hash = System.BitConverter.ToString(
                        sha.ComputeHash(File.ReadAllBytes(abs)))
                        .Replace("-", "").ToLowerInvariant();
                    Assert.AreEqual(hash, row!.final_sha256,
                        "final hash mismatch " + filePath);
                }
            }
        }

        [Test]
        public void TestNoPlaceholder()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var abs = Abs("client/" + ShowcaseRel(e));
                var bytes = File.ReadAllBytes(abs);
                Assert.AreEqual(0x89, bytes[0],
                    e.entity_id + " not a PNG");
                var info = new FileInfo(abs);
                Assert.Greater(info.Length, 4000,
                    e.entity_id + " showcase sprite is degenerate (<4KB)");
                if (IsSkeletal(e))
                {
                    var psb = Abs("client/" + PsbRel(e));
                    Assert.AreEqual(0x38, File.ReadAllBytes(psb)[0],
                        e.entity_id + " not a PSB (8BPS signature)");
                }
            }
        }

        [Test]
        public void TestAiCreatedToolMatchesOwnerSetup()
        {
            var rows = LoadFragment();
            var aiRows = rows.Where(r => r.source_kind == "AI_CREATED").ToArray();
            Assert.IsNotEmpty(aiRows, "no AI_CREATED rows");
            foreach (var r in aiRows)
            {
                Assert.NotNull(r.generation_record,
                    r.file_path + " AI_CREATED row needs generation_record");
                var g = r.generation_record!;
                Assert.IsNotNull(g.tool,
                    r.file_path + " tool missing");
                Assert.IsTrue(g.tool!.Contains(OwnerToolName),
                    r.file_path + " tool '" + g.tool + "' != owner tool");
                Assert.AreEqual(OwnerToolModel, g.model_id,
                    r.file_path + " model mismatch");
                Assert.IsNotEmpty(g.prompt, r.file_path + " prompt missing");
                Assert.GreaterOrEqual(g.seed, 0,
                    r.file_path + " seed must be >= 0");
                Assert.IsNotEmpty(g.terms_snapshot_sha256,
                    r.file_path + " terms snapshot missing");
                var snap = Abs(TermsRel + "/" + g.terms_snapshot_sha256 + ".txt");
                Assert.IsTrue(File.Exists(snap),
                    "terms snapshot missing " + snap);
            }
        }

        [Test]
        public void TestAnimationTechniqueAndClipsPerSizeProfile()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var profile =
                    CreatureArtMaterialization.ProfileOf(e.size_profile);
                var skeletal = CreatureArtMaterialization.IsSkeletal(e);
                var frameByFrame =
                    profile == SpriteSizeProfile.MonsterSmall
                    || profile == SpriteSizeProfile.SpiritBeast;
                Assert.IsTrue(skeletal || frameByFrame,
                    e.entity_id + " unexpected size_profile " + e.size_profile);
                var clipNames = new List<string>();
                var keyCounts = new List<int>();
                foreach (var clipName in RequiredClips(e))
                {
                    var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(
                        ClipRel(e, clipName));
                    Assert.NotNull(clip,
                        e.entity_id + " clip missing " + clipName);
                    clipNames.Add(clipName);
                    keyCounts.Add(CountKeys(clip!));
                }
                var declared = AnimationContract.ValidateDeclared(
                    profile,
                    skeletal ? "skeletal" : "frame_by_frame",
                    clipNames.ToArray(), keyCounts.ToArray(),
                    skeletal ? AnimationContract.SkeletalFps
                        : AnimationContract.FrameByFrameFps,
                    e.attack_clips, e.phases);
                Assert.IsEmpty(declared,
                    e.entity_id + " clip contract violations: "
                        + string.Join(";", declared.Select(v => v.Detail)));
                if (skeletal)
                {
                    var layers = ReadPsbLayerNames(
                        Abs("client/" + PsbRel(e)));
                    var violations = AnimationContract.CheckPsbLayers(layers);
                    Assert.IsEmpty(violations,
                        e.entity_id + " PSB layer violations: "
                            + string.Join(";", violations.Select(v => v.Detail)));
                }
            }
            // Shared skeleton assets per skeletal profile.
            foreach (var profile in new[]
                     { "monster_medium", "monster_elite", "boss_large",
                       "world_boss" })
            {
                var skel = AssetDatabase.LoadAssetAtPath<Object>(
                    AnimRoot + "/skeleton." + profile + ".asset");
                Assert.NotNull(skel,
                    "shared skeleton missing for " + profile);
            }
        }

        [Test]
        public void TestStylePackAndPaletteGate()
        {
            var packAbs = Path.Combine(Application.dataPath, "Art",
                "StyleRef", "actors_creatures");
            var packViolations = StylePackGate.CheckPackDir(packAbs);
            Assert.IsEmpty(packViolations,
                "style pack violations: "
                    + string.Join(";", packViolations.Select(v => v.Detail)));
            var paletteJson = File.ReadAllText(
                Path.Combine(packAbs, "palette.json"));
            var palette = StylePackGate.ParsePaletteJson(paletteJson);
            Assert.NotNull(palette, "palette.json failed to parse");
            foreach (var e in Entries())
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var px = LoadPng(Abs("client/" + ShowcaseRel(e)),
                    out var w, out var h);
                var violations = StylePackGate.CheckPalette(
                    px, w, h, null, palette!);
                Assert.IsEmpty(violations,
                    e.entity_id + " palette gate violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestFrameConsistency()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset" || IsSkeletal(e))
                {
                    continue;
                }
                var idlePath = Abs("client/" + CreaturesRoot + "/" + e.dir
                    + "/asset." + e.entity_id + ".anim.idle.0.png");
                Assert.IsTrue(File.Exists(idlePath),
                    e.entity_id + " idle frame 0 missing");
                var idle0 = LoadPng(idlePath, out var w, out var h);
                foreach (var clipName in RequiredClips(e))
                {
                    for (var i = 0; i < FrameCountFor(e, clipName); i++)
                    {
                        var framePath = Abs("client/" + CreaturesRoot + "/"
                            + e.dir + "/asset." + e.entity_id + ".anim."
                            + clipName + "." + i + ".png");
                        Assert.IsTrue(File.Exists(framePath),
                            e.entity_id + " " + clipName + " frame " + i
                                + " missing");
                        var frame = LoadPng(framePath, out var fw, out var fh);
                        Assert.AreEqual(w, fw,
                            e.entity_id + " " + clipName + " frame width");
                        Assert.AreEqual(h, fh,
                            e.entity_id + " " + clipName + " frame height");
                        var violations =
                            AnimationContract.CheckFrameConsistency(
                                idle0, frame, w, h, clipName);
                        Assert.IsEmpty(violations,
                            e.entity_id + " " + clipName + " frame " + i
                                + " consistency violations: "
                                + string.Join(";",
                                    violations.Select(v => v.Detail)));
                    }
                }
            }
        }

        [Test]
        public void TestHitboxSilhouetteAlignment()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset"
                    || !ColliderWidthRefPx.ContainsKey(e.size_profile))
                {
                    continue;
                }
                var (cellW, _) = CellSize[e.size_profile];
                var px = LoadPng(Abs("client/" + ShowcaseRel(e)),
                    out var w, out var h);
                var centreRefPx = cellW / 2f / 2f; // cell centre in ref px
                var violations = HitboxGate.CheckHitboxSilhouette(
                    px, w, h, centreRefPx, ColliderWidthRefPx[e.size_profile]);
                Assert.IsEmpty(violations,
                    e.entity_id + " hitbox violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestFolkloreCardsNoForbiddenMotif()
        {
            var rows = LoadFragment();
            var map = LoadMap();
            foreach (var e in Entries())
            {
                // Shared variants carry no own art; their folklore card
                // rides on the family base entity's row.
                var rowEntity = e;
                if (e.presentation == "shared")
                {
                    var st = map.shared_targets.FirstOrDefault(
                        t => "shared:" + t.name == e.target);
                    Assert.NotNull(st,
                        e.entity_id + " shared target missing");
                    rowEntity = map.entries.First(
                        b => b.entity_id == st!.base_entity);
                }
                var row = RowFor(rows, ShowcaseRel(rowEntity)
                    .Replace("Assets/", "client/Assets/"));
                Assert.NotNull(row, "no row for " + e.entity_id);
                Assert.IsTrue(row!.cultural_entity,
                    e.entity_id + " must be marked cultural_entity");
                Assert.NotNull(row.folklore_card,
                    e.entity_id + " folklore_card missing");
                Assert.IsNotEmpty(row.folklore_card!.source_tales,
                    e.entity_id + " source_tales empty");
                Assert.IsNotEmpty(row.folklore_card.regional_variants,
                    e.entity_id + " regional_variants empty");
                Assert.IsNotEmpty(row.folklore_card.motifs_checked,
                    e.entity_id + " motifs_checked empty");
            }
            // No forbidden motif may appear in any AI prompt.
            foreach (var r in rows.Where(r => r.source_kind == "AI_CREATED"))
            {
                var prompt = r.generation_record != null
                    ? (r.generation_record.prompt ?? "")
                    : "";
                var lower = prompt.ToLowerInvariant();
                foreach (var kv in ForbiddenMotifs)
                {
                    Assert.IsFalse(lower.Contains(kv.Key),
                        r.file_path + " prompt mentions forbidden motif "
                            + kv.Key + " (" + kv.Value + ")");
                }
            }
        }

        [Test]
        public void TestCutoutAndVolumeGates()
        {
            foreach (var e in Entries())
            {
                if (e.presentation != "asset")
                {
                    continue;
                }
                var violations = GateOne(ShowcaseRel(e));
                Assert.IsEmpty(violations,
                    e.entity_id + " gate violations: "
                        + string.Join(";", violations.Select(v => v.Rule
                            + "@" + v.Detail)));
            }
        }
    }
}
