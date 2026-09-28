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

namespace ThinhThan.Tests.EditMode.PlayerArtCoverage
{
    // IMP-071 Player Character & Class Art coverage. Verifies the five
    // class actors end to end: catalog keys, produced files (PSB rig,
    // showcase sprite, clips), canonical import settings, provenance
    // fragment, AI_CREATED tool provenance, skeletal rig layers, style
    // pack + palette gate, hitbox-silhouette alignment and folklore
    // cards. All gates are asserted to zero violations.
    public class PlayerArtCoverageTests
    {
        private static readonly string[] ClassIds =
        {
            "kim", "moc", "thuy", "hoa", "tho",
        };

        private const string ActorsRoot = "Assets/Art/Actors/Players";
        private const string StylePackRel =
            "Assets/Art/StyleRef/actors_players";
        private const string AnimRel = ActorsRoot + "/anim";
        private const string FragmentRel =
            "client/Assets/Art/Provenance/fragments/actors_players.json";
        private const string TermsRel =
            "client/Assets/Art/Provenance/terms/actors_players";

        // Canonical CHARACTER cell: 96x128 ref px at 2x = 192x256 tex px.
        private const int CellW = 192;
        private const int CellH = 256;

        // Canonical character collider 0.8 x 1.8 m at 50 ref px/m:
        // 40 ref px wide, centred on the 96 ref px cell (centre x = 48).
        private const float ColliderWidthRefPx = 40f;
        private const float ColliderCentreXRefPx = 48f;

        // Owner Setup tool string for every AI_CREATED record
        // (audit_gates.md Owner Setup + technology_versions.md Content
        // production tools).
        private const string OwnerToolName = "AI Horde";
        private const string OwnerToolModel = "AlbedoBase XL (SDXL)";

        private static string RepoRoot()
        {
            return Path.GetFullPath(
                Path.Combine(Application.dataPath, "..", ".."));
        }

        private static string Abs(string rel)
        {
            return Path.GetFullPath(Path.Combine(RepoRoot(), rel));
        }

        private static string ClassDir(string id)
        {
            return ActorsRoot + "/" + id;
        }

        private static string ShowcasePath(string id)
        {
            return ClassDir(id) + "/asset.class." + id + ".png";
        }

        private static string PsbPath(string id)
        {
            return ClassDir(id) + "/asset.class." + id + ".psb";
        }

        private static string PrefabPath(string id)
        {
            return ClassDir(id) + "/asset.class." + id + ".prefab";
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
                pos += channels * 4 + 4 + 4 + 4; // channel lens + blendsig + blendmode + opacity/clip/flags/filler
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

        [Test]
        public void TestClassToKeyCoverage()
        {
            AddressablesProvisioner.Provision();
            var plan = AssetKeyPlanner.Build(ContentCatalogScanner.DefaultDocsRoot);
            var keys = new HashSet<string>(plan.Select(p => p.Key));
            foreach (var id in ClassIds)
            {
                var key = "asset.class." + id;
                Assert.IsTrue(keys.Contains(key + ".prefab")
                    || keys.Contains(key),
                    "no catalog key for " + key);
                Assert.IsTrue(File.Exists(Abs("client/" + PsbPath(id))),
                    "rig source missing for " + id);
                Assert.IsTrue(File.Exists(Abs("client/" + ShowcasePath(id))),
                    "showcase sprite missing for " + id);
                Assert.IsTrue(File.Exists(Abs("client/" + PrefabPath(id))),
                    "prefab missing for " + id);
            }
        }

        [Test]
        public void TestRequiredClips()
        {
            var required = AnimationContract.RequiredClips(
                SpriteSizeProfile.Character, 0, 0);
            Assert.AreEqual(10, required.Length, "CHARACTER requires 10 clips");
            foreach (var clipName in required)
            {
                var path = AnimRel + "/" + clipName + ".anim";
                var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(path);
                Assert.NotNull(clip, "clip missing: " + path);
                Assert.AreEqual(AnimationContract.SkeletalFps,
                    (int)clip!.frameRate,
                    "clip " + clipName + " fps");
                var keyCount = CountKeys(clip);
                Assert.GreaterOrEqual(keyCount, AnimationContract.SkeletalMinKeys,
                    "clip " + clipName + " key count");
            }
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
        public void TestImportScaleCellPivot()
        {
            foreach (var id in ClassIds)
            {
                var abs = Abs("client/" + ShowcasePath(id));
                LoadPng(abs, out var w, out var h);
                Assert.AreEqual(CellW, w, id + " texture width");
                Assert.AreEqual(CellH, h, id + " texture height");
                var importer = AssetImporter.GetAtPath(ShowcasePath(id))
                    as TextureImporter;
                Assert.NotNull(importer, "no TextureImporter for " + id);
                Assert.AreEqual(100f, importer!.spritePixelsPerUnit, 0.01f,
                    id + " PPU");
                Assert.AreEqual(SpriteImportMode.Single, importer.spriteImportMode,
                    id + " sprite mode");
                var pivot = importer.spritePivot;
                Assert.IsEmpty(AnimationContract.CheckPivot(pivot),
                    id + " pivot must be Bottom Center");
                Assert.IsFalse(importer.mipmapEnabled, id + " mipmaps");
            }
        }

        [Test]
        public void TestProvenanceHash()
        {
            var rows = LoadFragment();
            Assert.IsNotEmpty(rows, "provenance fragment has no rows");
            var produced = new List<string>();
            foreach (var id in ClassIds)
            {
                produced.Add(PsbPath(id).Replace("Assets/", "client/Assets/"));
                produced.Add(ShowcasePath(id).Replace("Assets/", "client/Assets/"));
            }
            var styleref = Path.Combine(Application.dataPath, "Art",
                "StyleRef", "actors_players");
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
            foreach (var id in ClassIds)
            {
                var abs = Abs("client/" + ShowcasePath(id));
                var bytes = File.ReadAllBytes(abs);
                Assert.AreEqual(0x89, bytes[0], id + " not a PNG");
                var info = new FileInfo(abs);
                Assert.Greater(info.Length, 4000,
                    id + " showcase sprite is degenerate (<4KB)");
                var psb = Abs("client/" + PsbPath(id));
                Assert.AreEqual(0x38, File.ReadAllBytes(psb)[0],
                    id + " not a PSB (8BPS signature)");
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
                Assert.IsNotEmpty(g.terms_snapshot_sha256,
                    r.file_path + " terms snapshot missing");
                var snap = Abs(TermsRel + "/" + g.terms_snapshot_sha256 + ".txt");
                Assert.IsTrue(File.Exists(snap),
                    "terms snapshot missing " + snap);
            }
        }

        [Test]
        public void TestSkeletalRigLayersAndClips()
        {
            foreach (var id in ClassIds)
            {
                var layers = ReadPsbLayerNames(Abs("client/" + PsbPath(id)));
                var violations = AnimationContract.CheckPsbLayers(layers);
                Assert.IsEmpty(violations,
                    id + " PSB layer violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
            var required = AnimationContract.RequiredClips(
                SpriteSizeProfile.Character, 0, 0);
            var clipNames = new List<string>();
            var keyCounts = new List<int>();
            foreach (var clipName in required)
            {
                var clip = AssetDatabase.LoadAssetAtPath<AnimationClip>(
                    AnimRel + "/" + clipName + ".anim");
                Assert.NotNull(clip, "clip missing " + clipName);
                clipNames.Add(clipName);
                keyCounts.Add(CountKeys(clip!));
            }
            var declared = AnimationContract.ValidateDeclared(
                SpriteSizeProfile.Character, "skeletal",
                clipNames.ToArray(), keyCounts.ToArray(),
                AnimationContract.SkeletalFps, 0, 0);
            Assert.IsEmpty(declared,
                "clip contract violations: "
                    + string.Join(";", declared.Select(v => v.Detail)));
            var shared = AssetDatabase.LoadAssetAtPath<Object>(
                ActorsRoot + "/asset.class.players.skeleton.asset");
            Assert.NotNull(shared, "shared skeleton asset missing");
        }

        [Test]
        public void TestStylePackAndPaletteGate()
        {
            var packAbs = Path.Combine(Application.dataPath, "Art",
                "StyleRef", "actors_players");
            var packViolations = StylePackGate.CheckPackDir(packAbs);
            Assert.IsEmpty(packViolations,
                "style pack violations: "
                    + string.Join(";", packViolations.Select(v => v.Detail)));
            var paletteJson = File.ReadAllText(
                Path.Combine(packAbs, "palette.json"));
            var palette = StylePackGate.ParsePaletteJson(paletteJson);
            Assert.NotNull(palette, "palette.json failed to parse");
            foreach (var id in ClassIds)
            {
                var px = LoadPng(Abs("client/" + ShowcasePath(id)),
                    out var w, out var h);
                var violations = StylePackGate.CheckPalette(
                    px, w, h, null, palette!);
                Assert.IsEmpty(violations,
                    id + " palette gate violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestHitboxSilhouetteAlignment()
        {
            foreach (var id in ClassIds)
            {
                var px = LoadPng(Abs("client/" + ShowcasePath(id)),
                    out var w, out var h);
                var violations = HitboxGate.CheckHitboxSilhouette(
                    px, w, h, ColliderCentreXRefPx, ColliderWidthRefPx);
                Assert.IsEmpty(violations,
                    id + " hitbox violations: "
                        + string.Join(";", violations.Select(v => v.Detail)));
            }
        }

        [Test]
        public void TestFolkloreCards()
        {
            var rows = LoadFragment();
            foreach (var id in ClassIds)
            {
                var row = RowFor(rows, ShowcasePath(id)
                    .Replace("Assets/", "client/Assets/"));
                Assert.NotNull(row, "no row for " + id);
                Assert.IsTrue(row!.cultural_entity,
                    id + " must be marked cultural_entity");
                Assert.NotNull(row.folklore_card, id + " folklore_card missing");
                Assert.IsNotEmpty(row.folklore_card!.source_tales,
                    id + " source_tales empty");
                Assert.IsNotEmpty(row.folklore_card.regional_variants,
                    id + " regional_variants empty");
                Assert.IsNotEmpty(row.folklore_card.motifs_checked,
                    id + " motifs_checked empty");
            }
        }

        [Test]
        public void TestCutoutAndVolumeGates()
        {
            foreach (var id in ClassIds)
            {
                var violations = GateOne(ShowcasePath(id));
                Assert.IsEmpty(violations,
                    id + " gate violations: "
                        + string.Join(";", violations.Select(v => v.Rule
                            + "@" + v.Detail)));
            }
        }
    }
}
