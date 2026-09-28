using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using ThinhThan.Core.Assets;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Editor validator for the asset source register
    // (presentation_asset_manifest.md sections 5-6). Validate checks the
    // schema and per-row rules without touching disk; ValidateFiles also
    // compares each row against the file on disk; ValidateRelease adds
    // full media coverage and the APPROVED review state required for
    // release (fragments under fragments/ are merged first). Every failure
    // is a path-specific ProvenanceError.
    public static class AssetProvenanceValidator
    {
        public const string SourceKindAi = "AI_CREATED";
        public const string SourceKindFree = "FREE_LICENSED";
        public const string LicenseCc0 = "CC0-1.0";
        public const string LicenseCcBy = "CC-BY-4.0";
        public const string LicenseOfl = "OFL-1.1";
        public const string LicenseAiTool = "AI_TOOL_TERMS";
        public const string ReviewPending = "PENDING";
        public const string ReviewApproved = "APPROVED";
        public const string ReviewRejected = "REJECTED";

        private static readonly HashSet<string> FreeLicenses = new HashSet<string>
        {
            LicenseCc0,
            LicenseCcBy,
            LicenseOfl,
        };

        // Media extensions the register must cover (presentation_asset_manifest
        // §6: every packaged image/audio/font file gets exactly one row).
        private static readonly HashSet<string> MediaExtensions = new HashSet<string>
        {
            ".png", ".jpg", ".jpeg", ".webp", ".tga", ".psd",
            ".wav", ".ogg", ".mp3", ".aif", ".aiff",
            ".ttf", ".otf", ".ttc",
        };

        private static readonly HashSet<string> FontExtensions = new HashSet<string>
        {
            ".ttf", ".otf", ".ttc",
        };

        // Schema and per-row rules; no filesystem access.
        public static List<ProvenanceError> Validate(AssetSourceRegister register)
        {
            var errors = new List<ProvenanceError>();
            if (register.schema_version != 1)
            {
                errors.Add(new ProvenanceError(
                    "schema_version",
                    "expected 1, found " + register.schema_version));
            }
            var assets = register.assets ?? new List<AssetSourceRow>();
            var seen = new HashSet<string>();
            string? previousPath = null;
            foreach (var row in assets)
            {
                ValidateRow(row, seen, errors);
                if (previousPath != null && row.file_path != null
                    && string.CompareOrdinal(previousPath, row.file_path) >= 0)
                {
                    errors.Add(new ProvenanceError(
                        row.file_path,
                        "assets must be sorted by file_path ascending"));
                }
                if (row.file_path != null)
                {
                    previousPath = row.file_path;
                }
            }
            return errors;
        }

        // Foundation plus on-disk checks: the row's file must exist and its
        // sha256 must equal final_sha256.
        public static List<ProvenanceError> ValidateFiles(
            AssetSourceRegister register,
            string repoRoot)
        {
            var errors = Validate(register);
            foreach (var row in register.assets ?? new List<AssetSourceRow>())
            {
                if (row.file_path == null
                    || !AssetSourceRegisterIO.IsNormalizedRepoPath(row.file_path))
                {
                    continue;
                }
                var abs = AssetSourceRegisterIO.RepoToAbsolute(repoRoot, row.file_path);
                if (!File.Exists(abs))
                {
                    errors.Add(new ProvenanceError(row.file_path, "file does not exist"));
                    continue;
                }
                var actual = ComputeSha256(abs);
                if (row.final_sha256 != null && row.final_sha256 != actual)
                {
                    errors.Add(new ProvenanceError(
                        row.file_path,
                        "final_sha256 " + row.final_sha256 + " != disk " + actual));
                }
            }
            return errors;
        }

        // Release mode: every media file under assetsRoot has exactly one
        // row, and every row is APPROVED.
        public static List<ProvenanceError> ValidateRelease(
            AssetSourceRegister register,
            string repoRoot,
            string assetsRoot)
        {
            var merged = MergeFragments(register, repoRoot);
            var errors = ValidateFiles(merged, repoRoot);
            var rows = new HashSet<string>();
            foreach (var row in merged.assets ?? new List<AssetSourceRow>())
            {
                if (row.file_path != null)
                {
                    rows.Add(row.file_path);
                }
                if (row.review_state != null && row.review_state != ReviewApproved)
                {
                    errors.Add(new ProvenanceError(
                        row.file_path ?? "?",
                        "review_state " + row.review_state + " cannot ship (APPROVED required)"));
                }
            }
            foreach (var rel in EnumerateMediaFiles(assetsRoot))
            {
                var repoPath = AssetSourceRegisterIO.ToRepoRelative(
                    repoRoot,
                    Path.Combine(assetsRoot, rel));
                if (!rows.Contains(repoPath))
                {
                    errors.Add(new ProvenanceError(
                        repoPath,
                        "media file has no register row"));
                }
            }
            return errors;
        }

        // Merges fragment registers (IMP-071..075 write fragments; IMP-076
        // merges into the release register). Fragment files are every .json
        // under the fragments directory; rows are concatenated and re-sorted.
        public static AssetSourceRegister MergeFragments(
            AssetSourceRegister register,
            string repoRoot)
        {
            var merged = new AssetSourceRegister
            {
                schema_version = register.schema_version,
                assets = new List<AssetSourceRow>(register.assets ?? new List<AssetSourceRow>()),
            };
            var fragDir = Path.Combine(repoRoot, AssetSourceRegisterIO.FragmentsRepoDir);
            if (Directory.Exists(fragDir))
            {
                foreach (var file in Directory.GetFiles(fragDir, "*.json", SearchOption.AllDirectories))
                {
                    var frag = AssetSourceRegisterIO.Load(file);
                    if (frag?.assets != null)
                    {
                        merged.assets.AddRange(frag.assets);
                    }
                }
            }
            merged.assets.Sort(
                (a, b) => string.CompareOrdinal(a.file_path ?? "", b.file_path ?? ""));
            return merged;
        }

        // Repo-relative media files under assetsRoot, skipping Editor and
        // Tests path segments (tooling/test fixtures are not packaged media).
        public static List<string> EnumerateMediaFiles(string assetsRoot)
        {
            var result = new List<string>();
            if (!Directory.Exists(assetsRoot))
            {
                return result;
            }
            foreach (var file in Directory.GetFiles(assetsRoot, "*", SearchOption.AllDirectories))
            {
                var ext = Path.GetExtension(file).ToLowerInvariant();
                if (!MediaExtensions.Contains(ext))
                {
                    continue;
                }
                var rel = file.Substring(assetsRoot.Length).Replace('\\', '/').TrimStart('/');
                var skip = false;
                foreach (var seg in rel.Split('/'))
                {
                    if (seg == "Editor" || seg == "Tests")
                    {
                        skip = true;
                        break;
                    }
                }
                if (!skip)
                {
                    result.Add(rel);
                }
            }
            result.Sort(System.StringComparer.Ordinal);
            return result;
        }

        public static string ComputeSha256(string absPath)
        {
            using var sha = SHA256.Create();
            using var stream = File.OpenRead(absPath);
            var hash = sha.ComputeHash(stream);
            var sb = new System.Text.StringBuilder(64);
            foreach (var b in hash)
            {
                sb.Append(b.ToString("x2"));
            }
            return sb.ToString();
        }

        public static bool IsIsoUtc(string? s)
        {
            if (string.IsNullOrEmpty(s) || !s!.EndsWith("Z"))
            {
                return false;
            }
            var formats = new[]
            {
                "yyyy-MM-dd'T'HH:mm:ss'Z'",
                "yyyy-MM-dd'T'HH:mm:ss.FFFFFFF'Z'",
            };
            return System.DateTime.TryParseExact(
                s,
                formats,
                System.Globalization.CultureInfo.InvariantCulture,
                System.Globalization.DateTimeStyles.None,
                out _);
        }

        public static bool IsSha256(string? s)
        {
            if (s == null || s.Length != 64)
            {
                return false;
            }
            foreach (var c in s)
            {
                var digit = c >= '0' && c <= '9';
                var lower = c >= 'a' && c <= 'f';
                if (!digit && !lower)
                {
                    return false;
                }
            }
            return true;
        }

        public static bool IsHttpUri(string? s)
        {
            return !string.IsNullOrEmpty(s)
                && System.Uri.TryCreate(s, System.UriKind.Absolute, out var uri)
                && (uri.Scheme == "http" || uri.Scheme == "https");
        }

        // Heuristic for the section 6 rule "never write a search URL in
        // place of an origin URL": known search hosts, a /search path, or a
        // search query parameter mark the URI as a search page.
        public static bool IsSearchUri(string? s)
        {
            if (!IsHttpUri(s))
            {
                return false;
            }
            var uri = new System.Uri(s!);
            var host = uri.Host.ToLowerInvariant();
            var searchHosts = new[]
            {
                "google.", "www.google.", "bing.", "www.bing.",
                "duckduckgo.", "baidu.", "www.baidu.", "yahoo.",
                "search.yahoo.", "yandex.", "www.yandex.",
                "pinterest.", "www.pinterest.", "tineye.",
            };
            foreach (var h in searchHosts)
            {
                if (host.StartsWith(h))
                {
                    return true;
                }
            }
            var path = uri.AbsolutePath.ToLowerInvariant();
            if (path.Contains("/search"))
            {
                return true;
            }
            var query = uri.Query.ToLowerInvariant();
            return query.Contains("?q=") || query.Contains("&q=")
                || query.Contains("query=") || query.Contains("keyword=")
                || query.Contains("search=");
        }

        private static void ValidateRow(
            AssetSourceRow row,
            HashSet<string> seen,
            List<ProvenanceError> errors)
        {
            var path = row.file_path ?? "(missing file_path)";
            void Err(string message)
            {
                errors.Add(new ProvenanceError(path, message));
            }

            if (string.IsNullOrEmpty(row.file_path))
            {
                Err("file_path missing");
            }
            else if (!AssetSourceRegisterIO.IsNormalizedRepoPath(row.file_path!))
            {
                Err("file_path is not a normalized repo path");
            }
            else if (!seen.Add(row.file_path!))
            {
                Err("duplicate file_path");
            }

            if (string.IsNullOrEmpty(row.asset_key))
            {
                Err("asset_key missing");
            }
            else if (!AssetKey.IsValid(row.asset_key!))
            {
                Err("asset_key '" + row.asset_key + "' is not a valid Addressable key");
            }

            if (row.content_id != null && row.content_id.Length == 0)
            {
                Err("content_id must be null or a non-empty catalog id");
            }

            var isAi = row.source_kind == SourceKindAi;
            var isFree = row.source_kind == SourceKindFree;
            if (!isAi && !isFree)
            {
                Err("source_kind '" + row.source_kind + "' not in AI_CREATED|FREE_LICENSED");
            }

            if (string.IsNullOrEmpty(row.creator))
            {
                Err("creator missing");
            }

            if (isAi)
            {
                if (row.source_uri != null)
                {
                    Err("source_uri must be null for AI_CREATED");
                }
                if (row.license_id != LicenseAiTool)
                {
                    Err("license_id must be AI_TOOL_TERMS for AI_CREATED");
                }
            }
            else if (isFree)
            {
                if (!IsHttpUri(row.source_uri))
                {
                    Err("source_uri required and must be http(s) for FREE_LICENSED");
                }
                else if (IsSearchUri(row.source_uri))
                {
                    Err("source_uri is a search URL, not an origin page");
                }
                if (row.license_id == null || !FreeLicenses.Contains(row.license_id))
                {
                    Err("license_id must be CC0-1.0|CC-BY-4.0|OFL-1.1 for FREE_LICENSED");
                }
            }

            var knownLicense = row.license_id == LicenseCc0
                || row.license_id == LicenseCcBy
                || row.license_id == LicenseOfl
                || row.license_id == LicenseAiTool;
            if (!knownLicense)
            {
                Err("license_id '" + row.license_id
                    + "' not in CC0-1.0|CC-BY-4.0|OFL-1.1|AI_TOOL_TERMS");
            }
            if (row.license_id == LicenseOfl && row.file_path != null)
            {
                var ext = Path.GetExtension(row.file_path).ToLowerInvariant();
                if (!FontExtensions.Contains(ext))
                {
                    Err("OFL-1.1 is only valid for font files");
                }
            }
            if (!IsHttpUri(row.license_uri))
            {
                Err("license_uri missing or not http(s)");
            }
            if (!IsIsoUtc(row.acquired_at_utc))
            {
                Err("acquired_at_utc missing or not ISO 8601 UTC");
            }
            if (!IsSha256(row.source_sha256))
            {
                Err("source_sha256 missing or not 64 lowercase hex");
            }
            if (!IsSha256(row.final_sha256))
            {
                Err("final_sha256 missing or not 64 lowercase hex");
            }
            if (string.IsNullOrEmpty(row.changes))
            {
                Err("changes missing");
            }
            else if (row.changes == "none" && row.source_sha256 != row.final_sha256)
            {
                Err("changes is \"none\" but source_sha256 != final_sha256");
            }
            var needsAttribution = row.license_id == LicenseCcBy
                || row.license_id == LicenseOfl;
            if (needsAttribution && string.IsNullOrEmpty(row.attribution))
            {
                Err("attribution required for " + row.license_id);
            }

            var gen = row.generation_record;
            var genPresent = gen != null && !gen.IsEmpty();
            if (isAi && !genPresent)
            {
                Err("generation_record required for AI_CREATED");
            }
            else if (genPresent)
            {
                ValidateGenerationRecord(gen!, path, row.inputs, errors);
            }

            if (row.inputs != null)
            {
                for (var i = 0; i < row.inputs.Count; i++)
                {
                    ValidateInput(row.inputs[i], path, i, errors);
                }
            }

            // ART-010: entities of cultural origin must carry a
            // folklore_card; a card without cultural origin is allowed but
            // still has to satisfy the card shape.
            if (row.cultural_entity && row.folklore_card == null)
            {
                Err("folklore_card required for cultural entities (ART-010)");
            }
            if (row.folklore_card != null)
            {
                ValidateFolkloreCard(row.folklore_card, path, errors);
            }

            // ART-012: every upscale > 2x recorded in changes needs the
            // closing downscale-to-2x step (section 3.1); ops are the
            // semicolon-separated upscale:<factor>x / downscale_to_2x
            // tokens of the schema's changes convention.
            if (row.changes != null)
            {
                var sawOver2xUpscale = false;
                var sawDownscaleTo2x = false;
                foreach (var raw in row.changes.Split(';'))
                {
                    var op = raw.Trim();
                    if (op.StartsWith("upscale:"))
                    {
                        var f = op.Substring("upscale:".Length);
                        if (f.EndsWith("x"))
                        {
                            f = f.Substring(0, f.Length - 1);
                        }
                        if (float.TryParse(f, System.Globalization.NumberStyles.Float,
                                System.Globalization.CultureInfo.InvariantCulture, out var factor)
                            && factor > 2f)
                        {
                            sawOver2xUpscale = true;
                        }
                    }
                    else if (op == "downscale_to_2x")
                    {
                        sawDownscaleTo2x = true;
                    }
                }
                if (sawOver2xUpscale && !sawDownscaleTo2x)
                {
                    Err("changes records upscale > 2x without a closing downscale_to_2x step (ART-012)");
                }
            }

            if (row.review_state != ReviewPending
                && row.review_state != ReviewApproved
                && row.review_state != ReviewRejected)
            {
                Err("review_state '" + row.review_state + "' not in PENDING|APPROVED|REJECTED");
            }
        }

        private static void ValidateGenerationRecord(
            GenerationRecord gen,
            string path,
            List<InputProvenance>? inputs,
            List<ProvenanceError> errors)
        {
            void Err(string message)
            {
                errors.Add(new ProvenanceError(path, message));
            }
            if (string.IsNullOrEmpty(gen.tool))
            {
                Err("generation_record.tool missing");
            }
            if (string.IsNullOrEmpty(gen.version))
            {
                Err("generation_record.version missing");
            }
            if (string.IsNullOrEmpty(gen.model_id))
            {
                Err("generation_record.model_id missing (ART-012)");
            }
            if (gen.model_sha256 != null && !IsSha256(gen.model_sha256))
            {
                Err("generation_record.model_sha256 not 64 lowercase hex or null");
            }
            if (string.IsNullOrEmpty(gen.terms_uri))
            {
                Err("generation_record.terms_uri missing");
            }
            else if (!IsHttpUri(gen.terms_uri))
            {
                Err("generation_record.terms_uri not http(s)");
            }
            // ART-012: the terms snapshot hash names the stored copy at
            // client/Assets/Art/Provenance/terms/<fragment>/<sha256>.txt.
            if (!IsSha256(gen.terms_snapshot_sha256))
            {
                Err("generation_record.terms_snapshot_sha256 missing or not 64 lowercase hex (ART-012)");
            }
            if (string.IsNullOrEmpty(gen.prompt))
            {
                Err("generation_record.prompt missing");
            }
            if (!gen.seed.HasValue)
            {
                Err("generation_record.seed missing (ART-012)");
            }
            if (gen.parameters == null)
            {
                Err("generation_record.parameters missing (ART-012)");
            }
            if (gen.workflow_sha256 != null && !IsSha256(gen.workflow_sha256))
            {
                Err("generation_record.workflow_sha256 not 64 lowercase hex or null");
            }
            if (gen.style_pack_id != null && gen.style_pack_id.Length == 0)
            {
                Err("generation_record.style_pack_id must be null or a non-empty pack id");
            }
            if (gen.reference_uris != null)
            {
                var inputUris = new HashSet<string>();
                if (inputs != null)
                {
                    foreach (var input in inputs)
                    {
                        if (input.source_uri != null)
                        {
                            inputUris.Add(input.source_uri);
                        }
                    }
                }
                foreach (var refUri in gen.reference_uris)
                {
                    if (!inputUris.Contains(refUri))
                    {
                        Err("generation_record.reference_uris entry '" + refUri
                            + "' has no inputs[] row — a third-party input cannot be hidden");
                    }
                }
                if (gen.reference_sha256 == null
                    || gen.reference_sha256.Count != gen.reference_uris.Count)
                {
                    Err("generation_record.reference_sha256 must have one hash per reference_uris entry");
                }
                else
                {
                    for (var i = 0; i < gen.reference_sha256.Count; i++)
                    {
                        if (!IsSha256(gen.reference_sha256[i]))
                        {
                            Err("generation_record.reference_sha256[" + i
                                + "] not 64 lowercase hex");
                        }
                    }
                }
            }
        }

        private static void ValidateFolkloreCard(
            FolkloreCard card,
            string path,
            List<ProvenanceError> errors)
        {
            void Err(string message)
            {
                errors.Add(new ProvenanceError(path, message));
            }
            if (card.source_tales == null || card.source_tales.Count == 0)
            {
                Err("folklore_card.source_tales empty (ART-010)");
            }
            if (string.IsNullOrEmpty(card.regional_variants))
            {
                Err("folklore_card.regional_variants missing (ART-010)");
            }
            if (card.motifs_checked == null || card.motifs_checked.Count == 0)
            {
                Err("folklore_card.motifs_checked empty (ART-010)");
            }
        }

        private static void ValidateInput(
            InputProvenance input,
            string path,
            int index,
            List<ProvenanceError> errors)
        {
            var label = path + " inputs[" + index + "]";
            void Err(string message)
            {
                errors.Add(new ProvenanceError(label, message));
            }
            if (string.IsNullOrEmpty(input.creator))
            {
                Err("creator missing");
            }
            if (!IsHttpUri(input.source_uri))
            {
                Err("source_uri missing or not http(s)");
            }
            else if (IsSearchUri(input.source_uri))
            {
                Err("source_uri is a search URL, not an origin page");
            }
            if (input.license_id == null || !FreeLicenses.Contains(input.license_id))
            {
                Err("license_id must be a free license (CC0-1.0|CC-BY-4.0|OFL-1.1)");
            }
            if (!IsHttpUri(input.license_uri))
            {
                Err("license_uri missing or not http(s)");
            }
            if (!IsIsoUtc(input.acquired_at_utc))
            {
                Err("acquired_at_utc missing or not ISO 8601 UTC");
            }
            if (!IsSha256(input.sha256))
            {
                Err("sha256 missing or not 64 lowercase hex");
            }
        }
    }
}
