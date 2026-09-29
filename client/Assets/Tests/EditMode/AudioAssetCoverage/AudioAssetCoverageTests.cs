using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEditor;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.AudioAssetCoverage
{
    // IMP-075 SFX & Folklore BGM Production coverage. Verifies the
    // release audio universe end to end: every SFX cue and every BGM
    // key from the canonical key plan resolves to exactly one produced
    // .ogg under Assets/Audio/, every playable scene's .bgm alias
    // resolves one hop to a produced track, import settings follow the
    // streaming/decompress contract of presentation_asset_manifest.md
    // §1, audio.bgm.* groups stay under the 15 MB compressed / 4 MB
    // streaming-buffer budget, BGM loops cleanly and SFX do not clip or
    // mask combat feedback, the audio.json provenance fragment covers
    // every shipped file with source hashes and CC0/CC-BY licenses, and
    // every row names the owner-provided audio tool.
    public class AudioAssetCoverageTests
    {
        private const string AudioRoot = "Assets/Audio";
        private const string MapAssetPath = AudioRoot + "/audio_cue_map.json";
        private const string AnalysisAssetPath = AudioRoot + "/audio_analysis.json";
        private const string FragmentRel =
            "client/Assets/Art/Provenance/fragments/audio.json";
        private const string TermsRel =
            "client/Assets/Art/Provenance/terms/audio";

        // Owner Setup tool string for audio (audit_gates.md Owner Setup
        // + technology_versions.md § Content production tools):
        // "Freesound.org API (https://freesound.org/apiv2)".
        private const string OwnerToolName = "Freesound.org API";
        private const string OwnerToolVersion = "apiv2";

        private const float MinBgmSeconds = 45f;
        private const float MaxBgmSeconds = 150f;
        private const float MaxSeamDeltaDb = 3f;
        private const float MaxPeakDbfs = -0.2f;
        private const float MinSfxPeakDbfs = -1.5f;
        private const float MaxSfxSeconds = 8f;
        private const float MaxCombatSfxSeconds = 2.5f;
        private const float MaxCombatHeadSilenceMs = 60f;
        private const long MaxBgmGroupBytes = 15L * 1024 * 1024;
        private const double SeamSentinel = -999.0;

        // Cues whose SFX must stay short and attack-fast so they never
        // mask combat feedback (presentation_asset_manifest.md §6).
        private static readonly string[] CombatCues =
        {
            "basic_attack", "hit", "guard", "just_guard_success",
            "skill_cast", "boss_telegraph",
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

        [Serializable]
        private sealed class MapDoc
        {
            public int schema_version;
            public string fragment = "";
            public List<SfxEntry>? sfx;
            public List<BgmEntry>? bgm;
            public List<SceneBgmEntry>? scene_bgm;
            public TermsSnapshots? terms_snapshots;
        }

        [Serializable]
        private sealed class SfxEntry
        {
            public string cue = "";
            public string asset_key = "";
            public string group = "";
            public string file = "";
        }

        [Serializable]
        private sealed class BgmEntry
        {
            public string id = "";
            public string asset_key = "";
            public string group = "";
            public string file = "";
        }

        [Serializable]
        private sealed class SceneBgmEntry
        {
            public string space_id = "";
            public string alias_key = "";
            public string target_key = "";
            public string group = "";
        }

        [Serializable]
        private sealed class TermsSnapshots
        {
            public string tool_terms = "";
            public List<LicenseSnapshot>? licenses;
        }

        [Serializable]
        private sealed class LicenseSnapshot
        {
            public string license_id = "";
            public string sha256 = "";
        }

        [Serializable]
        private sealed class AnalysisDoc
        {
            public int schema_version;
            public List<AnalysisEntry>? files;
        }

        [Serializable]
        private sealed class AnalysisEntry
        {
            public string file = "";
            public string sha256 = "";
            public long size_bytes;
            public double duration_s;
            public int sample_rate;
            public int channels;
            public double peak_dbfs;
            public double rms_dbfs;
            public double head_silence_ms;
            public double tail_silence_ms;
            public double loop_seam_delta_db = SeamSentinel;
        }

        private static MapDoc LoadMap()
        {
            var abs = Abs("client/" + MapAssetPath);
            Assert.IsTrue(File.Exists(abs),
                "audio cue map missing: " + MapAssetPath);
            var doc = JsonUtility.FromJson<MapDoc>(File.ReadAllText(abs));
            Assert.NotNull(doc, "audio cue map failed to parse");
            Assert.AreEqual(1, doc!.schema_version, "map schema_version");
            Assert.AreEqual("audio", doc.fragment, "map fragment");
            Assert.NotNull(doc.sfx, "map sfx missing");
            Assert.NotNull(doc.bgm, "map bgm missing");
            Assert.NotNull(doc.scene_bgm, "map scene_bgm missing");
            return doc;
        }

        private static AnalysisDoc LoadAnalysis()
        {
            var abs = Abs("client/" + AnalysisAssetPath);
            Assert.IsTrue(File.Exists(abs),
                "audio analysis missing: " + AnalysisAssetPath);
            var doc = JsonUtility.FromJson<AnalysisDoc>(File.ReadAllText(abs));
            Assert.NotNull(doc, "audio analysis failed to parse");
            Assert.NotNull(doc!.files, "audio analysis files missing");
            return doc;
        }

        private static List<AssetSourceRow> LoadFragment()
        {
            var abs = Abs(FragmentRel);
            Assert.IsTrue(File.Exists(abs),
                "provenance fragment missing: " + FragmentRel);
            var register = AssetSourceRegisterIO.Load(abs);
            Assert.NotNull(register, "provenance fragment failed to parse");
            return register!.assets ?? new List<AssetSourceRow>();
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

        private static string Sha256Of(string absPath)
        {
            using (var sha = System.Security.Cryptography.SHA256.Create())
            {
                return BitConverter.ToString(
                        sha.ComputeHash(File.ReadAllBytes(absPath)))
                    .Replace("-", "").ToLowerInvariant();
            }
        }

        // Every produced media file the packet ships, repo-relative.
        private static List<string> ProducedRepoPaths(MapDoc doc)
        {
            var produced = new List<string>();
            foreach (var e in doc.sfx!)
            {
                produced.Add("client/" + AudioRoot + "/" + e.file);
            }
            foreach (var e in doc.bgm!)
            {
                produced.Add("client/" + AudioRoot + "/" + e.file);
            }
            return produced;
        }

        private static List<(string rel, bool isBgm)> ProducedWithRole(
            MapDoc doc)
        {
            var list = new List<(string, bool)>();
            foreach (var e in doc.sfx!)
            {
                list.Add(("client/" + AudioRoot + "/" + e.file, false));
            }
            foreach (var e in doc.bgm!)
            {
                list.Add(("client/" + AudioRoot + "/" + e.file, true));
            }
            return list;
        }

        // The canonical audio requirement set from the key plan: SFX cue
        // keys (facet clip) and BGM keys/aliases (facet bgm).
        private static List<PlannedAssetEntry> AudioPlan()
        {
            return AssetKeyPlanner.Build(ContentCatalogScanner.DefaultDocsRoot)
                .Where(e => e.Facet == AssetFacet.Clip
                    || e.Facet == AssetFacet.Bgm)
                .ToList();
        }

        private static string Declared(string userData, string key)
        {
            foreach (var raw in userData.Split(';'))
            {
                var token = raw.Trim();
                var eq = token.IndexOf('=');
                if (eq > 0 && token.Substring(0, eq).Trim() == key)
                {
                    return token.Substring(eq + 1).Trim();
                }
            }
            return "";
        }

        private static bool IsOgg(string absPath)
        {
            var head = new byte[4];
            using (var fs = File.OpenRead(absPath))
            {
                if (fs.Read(head, 0, 4) != 4)
                {
                    return false;
                }
            }
            return head[0] == 'O' && head[1] == 'g'
                && head[2] == 'g' && head[3] == 'S';
        }

        // ---------- tests -------------------------------------------------

        // Cue/key coverage: every required SFX cue and BGM key from the
        // canonical key plan resolves to exactly one produced file under
        // Assets/Audio/, carrying a grammatically valid key in the
        // canonical group.
        [Test]
        public void TestCueKeyCoverage()
        {
            var doc = LoadMap();
            var plan = AudioPlan();
            var wantedClips = plan.Where(e => e.Facet == AssetFacet.Clip)
                .Select(e => e.Key).OrderBy(k => k, StringComparer.Ordinal)
                .ToList();
            var wantedBgm = plan.Where(e => e.Facet == AssetFacet.Bgm
                    && e.AliasTarget.Length == 0)
                .Select(e => e.Key).OrderBy(k => k, StringComparer.Ordinal)
                .ToList();
            Assert.AreEqual(14, wantedClips.Count,
                "canonical plan must carry 14 SFX cues");
            Assert.AreEqual(8, wantedBgm.Count,
                "canonical plan must carry 8 BGM keys (6 zone + 2 shared)");

            var sfxKeys = doc.sfx!.Select(e => e.asset_key)
                .OrderBy(k => k, StringComparer.Ordinal).ToList();
            var bgmKeys = doc.bgm!.Select(e => e.asset_key)
                .OrderBy(k => k, StringComparer.Ordinal).ToList();
            Assert.IsTrue(sfxKeys.SequenceEqual(wantedClips),
                "map sfx keys != plan clip keys");
            Assert.IsTrue(bgmKeys.SequenceEqual(wantedBgm),
                "map bgm keys != plan bgm keys");

            foreach (var e in doc.sfx!)
            {
                Assert.IsTrue(e.cue.Length > 0, "sfx entry without cue");
                Assert.AreEqual("asset.sfx." + e.cue + ".clip", e.asset_key,
                    e.cue + " key must follow asset.sfx.<cue>.clip");
                Assert.IsTrue(AssetKey.IsValid(e.asset_key),
                    "invalid key " + e.asset_key);
                Assert.AreEqual(KeyGroupRule.Assign(e.asset_key, null),
                    e.group, e.asset_key + " group");
                Assert.AreEqual("shared.local", e.group,
                    e.asset_key + " must live in shared.local");
                Assert.IsTrue(
                    File.Exists(Abs("client/" + AudioRoot + "/" + e.file)),
                    e.asset_key + " file missing " + e.file);
            }
            foreach (var e in doc.bgm!)
            {
                Assert.IsTrue(AssetKey.IsValid(e.asset_key),
                    "invalid key " + e.asset_key);
                Assert.AreEqual(KeyGroupRule.Assign(e.asset_key, null),
                    e.group, e.asset_key + " group");
                Assert.IsTrue(e.group.StartsWith("audio.bgm.",
                        StringComparison.Ordinal),
                    e.asset_key + " must live in an audio.bgm.* group");
                Assert.IsTrue(
                    File.Exists(Abs("client/" + AudioRoot + "/" + e.file)),
                    e.asset_key + " file missing " + e.file);
            }
        }

        // Scene BGM coverage: every playable scene owns a one-hop
        // PresentationAlias in its own group whose target resolves to a
        // produced BGM track (manifest §2 + §6 minimum audio rule).
        [Test]
        public void TestSceneBgmAliasResolution()
        {
            var doc = LoadMap();
            var plan = AudioPlan()
                .Where(e => e.Facet == AssetFacet.Bgm
                    && e.AliasTarget.Length > 0)
                .ToList();
            Assert.AreEqual(33, plan.Count,
                "33 release scenes each need a .bgm alias");
            var byKey = doc.scene_bgm!.ToDictionary(e => e.alias_key);
            var targetFiles = doc.bgm!.ToDictionary(
                e => e.asset_key, e => e.file);
            foreach (var p in plan)
            {
                Assert.IsTrue(byKey.TryGetValue(p.Key, out var e),
                    "scene_bgm missing alias " + p.Key);
                Assert.AreEqual(p.AliasTarget, e!.target_key,
                    p.Key + " alias target must match the plan's "
                        + p.AliasTarget);
                Assert.AreEqual(p.Group, e.group,
                    p.Key + " alias must live in its space group "
                        + p.Group);
                Assert.IsTrue(targetFiles.ContainsKey(e.target_key),
                    p.Key + " target " + e.target_key + " has no track");
                Assert.IsTrue(
                    File.Exists(Abs("client/" + p.PlaceholderPath)),
                    p.Key + " placeholder alias missing "
                        + p.PlaceholderPath);
                var alias = AssetDatabase.LoadAssetAtPath<PresentationAlias>(
                    p.PlaceholderPath);
                Assert.NotNull(alias,
                    p.Key + " placeholder is not a PresentationAlias");
                Assert.AreEqual(e.target_key, alias!.TargetKey,
                    p.Key + " alias target_key drifted");
            }
        }

        // Loop and clip import settings: BGM streams (never whole files
        // in RAM, manifest §1) with Vorbis + loadInBackground; SFX cues
        // decompress on load so one-shots decode without allocations.
        [Test]
        public void TestAudioImportSettings()
        {
            foreach (var (rel, isBgm) in ProducedWithRole(LoadMap()))
            {
                var assetPath = rel.Substring("client/".Length);
                var importer = AssetImporter.GetAtPath(assetPath)
                    as AudioImporter;
                Assert.NotNull(importer,
                    assetPath + " has no AudioImporter .meta");
                var userData = importer!.userData ?? "";
                Assert.AreEqual("AUDIO", Declared(userData, "asset_class"),
                    assetPath + " userData must declare asset_class=AUDIO");
                Assert.AreEqual(isBgm ? "bgm" : "sfx",
                    Declared(userData, "audio_role"),
                    assetPath + " audio_role drifted");
                Assert.IsFalse(importer.forceToMono,
                    assetPath + " must stay stereo");
                Assert.IsFalse(importer.ambisonic,
                    assetPath + " must not be ambisonic");
                Assert.AreEqual(isBgm, importer.loadInBackground,
                    assetPath + " loadInBackground must be " + isBgm);
                var s = importer.defaultSampleSettings;
                Assert.AreEqual(
                    isBgm ? AudioClipLoadType.Streaming
                        : AudioClipLoadType.DecompressOnLoad,
                    s.loadType,
                    assetPath + " loadType: BGM must stream, SFX must "
                        + "decode ahead");
                Assert.AreEqual(AudioCompressionFormat.Vorbis,
                    s.compressionFormat, assetPath + " must use Vorbis");
                Assert.AreEqual(AudioSampleRateSetting.PreserveSampleRate,
                    s.sampleRateSetting, assetPath + " sample rate");
                Assert.IsTrue(s.quality > 0f && s.quality <= 1f,
                    assetPath + " quality out of range");
                Assert.AreEqual(!isBgm, s.preloadAudioData,
                    assetPath + " preloadAudioData must be " + !isBgm);
            }
        }

        // Bundle/streaming budgets (manifest §1): each audio.bgm.* group
        // stays under 15 MB compressed, BGM streams (one 4 MB buffer per
        // group, never whole-file RAM), and the produced SFX share fits
        // inside shared.local's 70 MB budget alongside every other class.
        [Test]
        public void TestAudioGroupBudgets()
        {
            var doc = LoadMap();
            var perGroup = new Dictionary<string, long>(
                StringComparer.Ordinal);
            foreach (var e in doc.bgm!)
            {
                var size = new FileInfo(
                    Abs("client/" + AudioRoot + "/" + e.file)).Length;
                Assert.Greater(size, 0, e.asset_key + " is empty");
                Assert.Less(size, MaxBgmGroupBytes,
                    e.asset_key + " alone exceeds the group budget");
                perGroup.TryGetValue(e.group, out var sum);
                perGroup[e.group] = sum + size;
                // Streaming is the RAM-budget contract: a streaming clip
                // only ever holds a small decode buffer per group.
                var importer = AssetImporter.GetAtPath(
                    "Assets/Audio/" + e.file) as AudioImporter;
                Assert.NotNull(importer);
                Assert.AreEqual(AudioClipLoadType.Streaming,
                    importer!.defaultSampleSettings.loadType,
                    e.asset_key + " must stream to stay under the "
                        + "4 MB RAM budget");
            }
            foreach (var kv in perGroup)
            {
                Assert.LessOrEqual(kv.Value, MaxBgmGroupBytes,
                    kv.Key + " exceeds 15 MB compressed: " + kv.Value);
            }

            long sfxBytes = 0;
            foreach (var e in doc.sfx!)
            {
                sfxBytes += new FileInfo(
                    Abs("client/" + AudioRoot + "/" + e.file)).Length;
            }
            const long sharedLocalBudget = 70L * 1024 * 1024;
            Assert.Less(sfxBytes, sharedLocalBudget,
                "SFX cannot fit shared.local's 70 MB budget");
        }

        // Loop + level quality: every file matches its sha256-bound
        // analysis row; BGM loop seams stay under MaxSeamDeltaDb so the
        // loop is clean; SFX peak stays under clipping but above silence;
        // combat cues are short with fast attack so they never mask
        // combat feedback.
        [Test]
        public void TestLoopSeamAndClipLevels()
        {
            var doc = LoadMap();
            var analysis = LoadAnalysis();
            var produced = ProducedRepoPaths(doc);
            var byRel = analysis.files!.ToDictionary(
                e => "client/" + AudioRoot + "/" + e.file);
            Assert.AreEqual(produced.Count, analysis.files!.Count,
                "analysis entries != produced files");
            var combat = new HashSet<string>(CombatCues,
                StringComparer.Ordinal);
            var cueOf = doc.sfx!.ToDictionary(
                e => "client/" + AudioRoot + "/" + e.file, e => e.cue);
            var bgmRels = new HashSet<string>(
                doc.bgm!.Select(e => "client/" + AudioRoot + "/" + e.file),
                StringComparer.Ordinal);
            foreach (var rel in produced)
            {
                Assert.IsTrue(byRel.TryGetValue(rel, out var a),
                    rel + " missing analysis row");
                Assert.AreEqual(Sha256Of(Abs(rel)), a!.sha256,
                    rel + " analysis sha256 does not match the file");
                Assert.AreEqual(new FileInfo(Abs(rel)).Length,
                    a.size_bytes, rel + " size drifted");
                Assert.AreEqual(44100, a.sample_rate,
                    rel + " must be 44.1 kHz");
                Assert.IsTrue(a.channels == 1 || a.channels == 2,
                    rel + " must be mono or stereo");
                Assert.LessOrEqual(a.peak_dbfs, MaxPeakDbfs,
                    rel + " peak reaches clipping");
                if (bgmRels.Contains(rel))
                {
                    Assert.IsTrue(a.duration_s >= MinBgmSeconds
                            && a.duration_s <= MaxBgmSeconds,
                        rel + " BGM duration out of range: "
                            + a.duration_s);
                    Assert.IsTrue(a.loop_seam_delta_db >= 0
                            && a.loop_seam_delta_db <= MaxSeamDeltaDb,
                        rel + " loop seam jump too loud: "
                            + a.loop_seam_delta_db + " dB");
                    Assert.Greater(a.tail_silence_ms, 0,
                        rel + " must carry loop tail padding");
                }
                else
                {
                    Assert.IsTrue(a.duration_s >= 0.05
                            && a.duration_s <= MaxSfxSeconds,
                        rel + " SFX duration out of range: "
                            + a.duration_s);
                    Assert.IsTrue(a.peak_dbfs >= MinSfxPeakDbfs
                            && a.peak_dbfs <= MaxPeakDbfs,
                        rel + " SFX peak out of range: " + a.peak_dbfs);
                    if (combat.Contains(cueOf[rel]))
                    {
                        Assert.LessOrEqual(a.duration_s,
                            MaxCombatSfxSeconds,
                            rel + " combat cue too long: " + a.duration_s);
                        Assert.LessOrEqual(a.head_silence_ms,
                            MaxCombatHeadSilenceMs,
                            rel + " combat cue starts late: "
                                + a.head_silence_ms + " ms");
                    }
                }
            }
        }

        // No-placeholder: every SFX decodes to real audio (non-constant
        // window energy + varying zero-crossing rate rule out synthetic
        // beeps and silence), every file is a real Ogg container.
        [Test]
        public void TestSfxDecodeAndNoPlaceholder()
        {
            var doc = LoadMap();
            foreach (var e in doc.sfx!)
            {
                var abs = Abs("client/" + AudioRoot + "/" + e.file);
                Assert.IsTrue(IsOgg(abs), e.file + " is not an Ogg");
                Assert.Greater(new FileInfo(abs).Length, 8192,
                    e.file + " too small to be a real clip");
                var clip = AssetDatabase.LoadAssetAtPath<AudioClip>(
                    "Assets/Audio/" + e.file);
                Assert.NotNull(clip, e.file + " does not load");
                Assert.Greater(clip!.samples, 0, e.file + " empty");
                var data = new float[clip.samples * clip.channels];
                Assert.IsTrue(clip.GetData(data, 0),
                    e.file + " GetData failed — must decode in EditMode");
                float peak = 0f;
                var window = 4096;
                var energies = new List<double>();
                var zcrs = new List<double>();
                for (var i = 0; i + window <= data.Length; i += window)
                {
                    double sum = 0.0;
                    var zeroX = 0;
                    for (var j = i; j < i + window; j++)
                    {
                        var v = data[j];
                        sum += v * v;
                        if (Mathf.Abs(v) > peak)
                        {
                            peak = Mathf.Abs(v);
                        }
                        if (j > i && (data[j - 1] < 0f) != (v < 0f))
                        {
                            zeroX++;
                        }
                    }
                    energies.Add(sum / window);
                    zcrs.Add((double)zeroX / window);
                }
                Assert.Less(peak, 1f, e.file + " clips at full scale");
                Assert.Greater(peak, 0.05f, e.file + " is near-silent");
                double Std(List<double> xs)
                {
                    var m = xs.Average();
                    return Math.Sqrt(xs.Average(x => (x - m) * (x - m)));
                }
                Assert.Greater(Std(energies), 1e-6,
                    e.file + " constant energy: placeholder beep/silence");
                Assert.Greater(Std(zcrs), 1e-4,
                    e.file + " constant spectrum: placeholder beep");
            }
            foreach (var e in doc.bgm!)
            {
                var abs = Abs("client/" + AudioRoot + "/" + e.file);
                Assert.IsTrue(IsOgg(abs), e.file + " is not an Ogg");
                Assert.Greater(new FileInfo(abs).Length, 512 * 1024,
                    e.file + " too small to be a real BGM track");
            }
        }

        // Provenance coverage + hash audit: the audio fragment passes the
        // validator, covers every produced file, and each row's
        // final_sha256 matches the shipped bytes.
        [Test]
        public void TestProvenanceHash()
        {
            var doc = LoadMap();
            var rows = LoadFragment();
            var repoRoot = RepoRoot();
            var produced = ProducedRepoPaths(doc);
            foreach (var rel in produced)
            {
                var row = RowFor(rows, rel);
                Assert.NotNull(row, rel + " missing provenance row");
                Assert.AreEqual("PENDING", row!.review_state,
                    rel + " review_state must be PENDING");
                Assert.AreEqual(Sha256Of(Abs(rel)),
                    row.final_sha256 ?? "", rel + " final_sha256 drifted");
            }
            var fragmentFiles = new HashSet<string>(
                rows.Select(r => r.file_path).OfType<string>(),
                StringComparer.Ordinal);
            var orphan = fragmentFiles.Except(produced).ToList();
            Assert.IsEmpty(orphan,
                "provenance rows for files the map does not ship: "
                    + string.Join(",", orphan));

            var register = AssetSourceRegisterIO.Load(
                Abs(FragmentRel));
            Assert.NotNull(register);
            var errors = AssetProvenanceValidator
                .ValidateFiles(register!, repoRoot);
            Assert.IsEmpty(errors,
                "provenance validation failed: " + string.Join("; ",
                    errors.Select(e => e.ToString())));
        }

        // License + attribution audit: every row is CC0 or CC-BY-4.0
        // from a real source URI (no search pages, no unlicensed
        // recordings); CC-BY rows carry full attribution; inputs[] carry
        // the same license evidence chain.
        [Test]
        public void TestLicenseAttribution()
        {
            var rows = LoadFragment();
            Assert.GreaterOrEqual(rows.Count, 22,
                "audio fragment must cover every produced file");
            foreach (var row in rows)
            {
                Assert.AreEqual(AssetProvenanceValidator.SourceKindFree, row.source_kind,
                    row.file_path + " must be FREE_LICENSED (Freesound "
                        + "API sources licensed recordings)");
                Assert.IsTrue(
                    row.license_id == "CC0-1.0"
                        || row.license_id == "CC-BY-4.0",
                    row.file_path + " license " + row.license_id
                        + " is not CC0/CC-BY");
                Assert.NotNull(row.source_uri,
                    row.file_path + " missing source_uri");
                Assert.IsTrue(row.source_uri!.StartsWith("http",
                        StringComparison.Ordinal),
                    row.file_path + " source_uri must be http(s)");
                Assert.IsFalse(
                    AssetProvenanceValidator.IsSearchUri(row.source_uri),
                    row.file_path + " source_uri points at a search page");
                Assert.NotNull(row.license_uri,
                    row.file_path + " missing license_uri");
                Assert.IsTrue(row.license_uri!.StartsWith("http",
                        StringComparison.Ordinal),
                    row.file_path + " license_uri must be http(s)");
                var licenseToken = row.license_uri!.ToLowerInvariant();
                Assert.IsFalse(
                    licenseToken.Contains("noncommercial")
                        || licenseToken.Contains("by-nc")
                        || licenseToken.Contains("by-nd")
                        || licenseToken.Contains("by-sa"),
                    row.file_path + " license forbids use: "
                        + row.license_uri);
                if (row.license_id == "CC-BY-4.0")
                {
                    Assert.NotNull(row.attribution,
                        row.file_path + " CC-BY requires attribution");
                    Assert.IsTrue(row.attribution!.Contains("CC BY 4.0"),
                        row.file_path + " attribution must name "
                            + "CC BY 4.0");
                    var host = new Uri(row.source_uri!).Host;
                    Assert.IsTrue(row.attribution!.Contains(host),
                        row.file_path + " attribution must name the "
                            + "source site");
                }
                foreach (var input in row.inputs
                    ?? new List<InputProvenance>())
                {
                    Assert.NotNull(input.source_uri,
                        row.file_path + " input missing source_uri");
                    Assert.IsTrue(input.source_uri!.StartsWith("http",
                            StringComparison.Ordinal),
                        row.file_path + " input source_uri http(s)");
                    Assert.IsFalse(AssetProvenanceValidator.IsSearchUri(
                            input.source_uri),
                        row.file_path + " input source_uri is a "
                            + "search page");
                    Assert.IsTrue(
                        input.license_id == "CC0-1.0"
                            || input.license_id == "CC-BY-4.0",
                        row.file_path + " input license "
                            + input.license_id + " is not CC0/CC-BY");
                    Assert.NotNull(input.creator,
                        row.file_path + " input missing creator");
                }
            }
        }

        // Tool-string audit (manifest §6 + ADR-0076): every audio row is
        // produced with the owner-provided tool — the Freesound.org API
        // matrix row — recorded on the row itself.
        [Test]
        public void TestAiCreatedToolMatchesOwnerSetup()
        {
            var rows = LoadFragment();
            foreach (var row in rows)
            {
                Assert.NotNull(row.creator,
                    row.file_path + " missing creator");
                Assert.IsTrue(row.creator!.Contains(OwnerToolName),
                    row.file_path + " must name the audio tool "
                        + OwnerToolName);
                Assert.IsTrue(row.creator!.Contains(OwnerToolVersion),
                    row.file_path + " must name the API version "
                        + OwnerToolVersion);
                if (row.source_kind == AssetProvenanceValidator.SourceKindAi)
                {
                    var g = row.generation_record;
                    Assert.NotNull(g,
                        row.file_path + " AI_CREATED missing record");
                    Assert.IsTrue(g!.tool != null
                            && g.tool.Contains(OwnerToolName),
                        row.file_path + " tool must contain "
                            + OwnerToolName);
                    Assert.IsTrue(g.version != null
                            && g.version.Contains(OwnerToolVersion),
                        row.file_path + " version must contain "
                            + OwnerToolVersion);
                }
            }
        }

        // ADR-0076 contract: AI_CREATED rows carry the extended
        // generation_record including a terms_snapshot_sha256 whose file
        // exists under Provenance/terms/audio/ and hashes to the same
        // value; FREE_LICENSED rows carry no record. License + tool
        // terms snapshots bound in the cue map exist and hash-match.
        [Test]
        public void TestAudioGenerationRecordAndTermsSnapshot()
        {
            var doc = LoadMap();
            var rows = LoadFragment();
            foreach (var row in rows)
            {
                if (row.source_kind == AssetProvenanceValidator.SourceKindAi)
                {
                    var g = row.generation_record;
                    Assert.NotNull(g,
                        row.file_path + " AI_CREATED missing record");
                    Assert.IsTrue(g!.tool != null
                            && g.tool.Contains(OwnerToolName),
                        row.file_path + " tool must name "
                            + OwnerToolName);
                    Assert.IsFalse(string.IsNullOrEmpty(g.version),
                        row.file_path + " version empty");
                    Assert.IsFalse(string.IsNullOrEmpty(g.terms_uri),
                        row.file_path + " terms_uri empty");
                    Assert.IsTrue(g.terms_uri!.StartsWith("http",
                            StringComparison.Ordinal),
                        row.file_path + " terms_uri must be http(s)");
                    Assert.NotNull(g.terms_snapshot_sha256,
                        row.file_path + " terms_snapshot_sha256 null");
                    Assert.AreEqual(64,
                        g.terms_snapshot_sha256!.Length,
                        row.file_path + " snapshot hash malformed");
                    var snapshot = Abs(TermsRel + "/"
                        + g.terms_snapshot_sha256 + ".txt");
                    Assert.IsTrue(File.Exists(snapshot),
                        row.file_path + " terms snapshot missing: "
                            + g.terms_snapshot_sha256);
                    Assert.AreEqual(Sha256Of(snapshot),
                        g.terms_snapshot_sha256,
                        row.file_path + " snapshot content hash drifted");
                    Assert.IsFalse(string.IsNullOrEmpty(g.prompt),
                        row.file_path + " prompt empty");
                    Assert.GreaterOrEqual(g.seed, 0,
                        row.file_path + " seed must be >= 0 "
                            + "(long, -1 = none)");
                }
                else
                {
                    Assert.IsTrue(row.generation_record == null
                            || row.generation_record.IsEmpty(),
                        row.file_path + " FREE_LICENSED row carries a "
                            + "generation_record");
                }
            }

            // Terms snapshots for the tool and every license actually
            // used must exist under terms/audio/ and hash-match the
            // cue map binding (audit trail for license reviews).
            Assert.NotNull(doc.terms_snapshots,
                "map missing terms_snapshots");
            var snaps = doc.terms_snapshots!;
            Assert.IsTrue(snaps.tool_terms.Length == 64,
                "tool_terms snapshot hash malformed");
            var toolFile = Abs(TermsRel + "/" + snaps.tool_terms + ".txt");
            Assert.IsTrue(File.Exists(toolFile),
                "tool terms snapshot missing " + snaps.tool_terms);
            Assert.AreEqual(Sha256Of(toolFile), snaps.tool_terms,
                "tool terms snapshot content hash drifted");
            var snapByLicense = (snaps.licenses
                    ?? new List<LicenseSnapshot>())
                .ToDictionary(l => l.license_id, l => l.sha256);
            foreach (var license in rows.Select(r => r.license_id)
                .Concat(rows.SelectMany(r => (r.inputs
                        ?? new List<InputProvenance>())
                    .Select(i => i.license_id)))
                .Where(l => l != null).Distinct())
            {
                Assert.IsTrue(snapByLicense.TryGetValue(license!, out var sha),
                    "no terms snapshot for used license " + license);
                var file = Abs(TermsRel + "/" + sha + ".txt");
                Assert.IsTrue(File.Exists(file),
                    "license snapshot missing for " + license);
                Assert.AreEqual(Sha256Of(file), sha,
                    license + " snapshot content hash drifted");
            }
        }

        // Folklore cards: every BGM track is a cultural-origin entity and
        // carries a complete folklore card (manifest §5).
        [Test]
        public void TestAudioFolkloreCards()
        {
            var doc = LoadMap();
            var rows = LoadFragment();
            foreach (var e in doc.bgm!)
            {
                var rel = "client/" + AudioRoot + "/" + e.file;
                var row = RowFor(rows, rel);
                Assert.NotNull(row, rel + " missing provenance row");
                Assert.IsTrue(row!.cultural_entity,
                    rel + " BGM is a cultural-origin entity");
                Assert.NotNull(row.folklore_card,
                    rel + " BGM missing folklore card");
                Assert.IsTrue(
                    row.folklore_card!.source_tales != null
                        && row.folklore_card.source_tales.Count > 0,
                    rel + " folklore_card.source_tales empty");
                Assert.IsFalse(string.IsNullOrEmpty(
                        row.folklore_card.regional_variants),
                    rel + " folklore_card.regional_variants empty");
                Assert.IsTrue(
                    row.folklore_card.motifs_checked != null
                        && row.folklore_card.motifs_checked.Count > 0,
                    rel + " folklore_card.motifs_checked empty");
            }
        }
    }
}
