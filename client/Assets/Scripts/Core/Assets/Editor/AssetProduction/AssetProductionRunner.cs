using System.Collections.Generic;
using System.IO;
using System.Text;
using UnityEditor;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Entry points for the asset-production validators: provenance register
    // (foundation and release modes), the cutout/volume quality gates, the
    // deterministic credits output, and the visual-review renders. Menu
    // items are for interactive use; the public static methods are the
    // -executeMethod entry points CI calls.
    public static class AssetProductionRunner
    {
        private const string ReportDirName = "Library/AssetProduction";

        [MenuItem("ThinhThan/Asset Production/Validate Provenance (Foundation)")]
        public static void MenuValidateFoundation()
        {
            var rc = ValidateFoundation();
            Debug.Log("provenance foundation " + (rc == 0 ? "PASS" : "FAIL"));
        }

        [MenuItem("ThinhThan/Asset Production/Validate Provenance (Release)")]
        public static void MenuValidateRelease()
        {
            var rc = ValidateRelease();
            Debug.Log("provenance release " + (rc == 0 ? "PASS" : "FAIL"));
        }

        [MenuItem("ThinhThan/Asset Production/Run Quality Gates")]
        public static void MenuRunGates()
        {
            var rc = RunGates(AssetsRoot());
            Debug.Log("quality gates " + (rc == 0 ? "PASS" : "FAIL"));
        }

        [MenuItem("ThinhThan/Asset Production/Render Visual Review")]
        public static void MenuRenderVisualReview()
        {
            VisualReviewRenderer.RenderAll(VisualReviewRenderer.OutputRoot());
        }

        // -executeMethod entry: foundation validation of the committed
        // register. Returns the process exit code semantics via
        // EditorApplication.Exit in menu paths; CI reads the log.
        public static int ValidateFoundation()
        {
            var repoRoot = RepoRoot();
            var registerPath = Path.Combine(repoRoot, AssetSourceRegisterIO.RegisterRepoPath);
            var register = AssetSourceRegisterIO.Load(registerPath);
            if (register == null)
            {
                Debug.LogError("provenance: cannot load " + AssetSourceRegisterIO.RegisterRepoPath);
                return 1;
            }
            var errors = AssetProvenanceValidator.ValidateFiles(register, repoRoot);
            foreach (var e in errors)
            {
                Debug.LogError("provenance: " + e);
            }
            Debug.Log("provenance: foundation validation " +
                (errors.Count == 0 ? "passed" : "failed with " + errors.Count + " errors"));
            return errors.Count == 0 ? 0 : 1;
        }

        // -executeMethod entry: release-mode validation (schema + disk
        // hashes + media coverage + APPROVED state).
        public static int ValidateRelease()
        {
            var repoRoot = RepoRoot();
            var registerPath = Path.Combine(repoRoot, AssetSourceRegisterIO.RegisterRepoPath);
            var register = AssetSourceRegisterIO.Load(registerPath);
            if (register == null)
            {
                Debug.LogError("provenance: cannot load " + AssetSourceRegisterIO.RegisterRepoPath);
                return 1;
            }
            var errors = AssetProvenanceValidator.ValidateRelease(register, repoRoot, AssetsRoot());
            foreach (var e in errors)
            {
                Debug.LogError("provenance: " + e);
            }
            return errors.Count == 0 ? 0 : 1;
        }

        // Runs the §3.2/§3.6 gates over every declared media file and writes
        // a per-file report to Library/AssetProduction/gate_report.json
        // (transient, gitignored). Files without declared asset_class in
        // importer userData are reported as violations.
        public static int RunGates(string assetsRoot)
        {
            var report = new StringBuilder();
            report.Append("{\"files\":[");
            var violations = 0;
            var first = true;
            var repoRoot = RepoRoot();
            foreach (var rel in AssetProvenanceValidator.EnumerateMediaFiles(assetsRoot))
            {
                var abs = Path.Combine(assetsRoot, rel);
                var fileViolations = GateFile(abs, repoRoot);
                violations += fileViolations.Count;
                if (!first)
                {
                    report.Append(',');
                }
                first = false;
                report.Append("{\"file\":\"").Append(EscapeJson(rel)).Append("\",\"violations\":[");
                for (var i = 0; i < fileViolations.Count; i++)
                {
                    if (i > 0)
                    {
                        report.Append(',');
                    }
                    var v = fileViolations[i];
                    report.Append("{\"rule\":\"").Append(v.Rule).Append("\",\"detail\":\"")
                        .Append(EscapeJson(v.Detail)).Append("\"}");
                }
                report.Append("]}");
            }
            report.Append("]}");
            var reportPath = Path.Combine(RepoRoot(), "client", ReportDirName, "gate_report.json");
            Directory.CreateDirectory(Path.GetDirectoryName(reportPath)!);
            File.WriteAllText(reportPath, report.ToString());
            Debug.Log("quality gates: " + violations + " violations, report " + reportPath);
            return violations == 0 ? 0 : 1;
        }

        private static List<GateViolation> GateFile(string absPng, string repoRoot)
        {
            var violations = new List<GateViolation>();
            if (!absPng.EndsWith(".png"))
            {
                return violations;
            }
            var repoPath = AssetSourceRegisterIO.ToRepoRelative(repoRoot, absPng);
            if (!repoPath.StartsWith("client/"))
            {
                return violations;
            }
            var importer = AssetImporter.GetAtPath(repoPath.Substring("client/".Length));
            var meta = ImportMetadata.Parse(
                importer != null ? importer.userData : null);
            if (meta.AssetClass == null)
            {
                violations.Add(new GateViolation(
                    "asset_class", "import metadata missing (declare asset_class in importer userData)"));
                return violations;
            }
            bool[]? translMask = null;
            if (meta.TranslucentSubject != null)
            {
                var maskPath = absPng.Substring(0, absPng.Length - 4) + ".translucent.png";
                var m = new Texture2D(2, 2, TextureFormat.RGBA32, false);
                if (File.Exists(maskPath) && ImageConversion.LoadImage(m, File.ReadAllBytes(maskPath)))
                {
                    var px = m.GetPixels32();
                    translMask = new bool[px.Length];
                    for (var i = 0; i < px.Length; i++)
                    {
                        translMask[i] = px[i].r > 127;
                    }
                }
                Object.DestroyImmediate(m);
            }
            var input = PresentationSizing.ToCutoutInput(
                null!, 0, 0, meta.AssetClass.Value, meta, translMask);
            violations.AddRange(CutoutQualityGate.ValidateFile(absPng, input));
            if (input.Width > 0)
            {
                var vinput = PresentationSizing.ToVolumeInput(
                    input.Pixels, input.Width, input.Height,
                    meta.AssetClass.Value, translMask);
                violations.AddRange(VolumeDepthGate.ValidatePixels(vinput));
            }
            return violations;
        }

        private static string RepoRoot()
        {
            return Path.GetFullPath(Path.Combine(Application.dataPath, "..", ".."));
        }

        private static string AssetsRoot()
        {
            return Path.GetFullPath(Application.dataPath);
        }

        private static string EscapeJson(string s)
        {
            return s.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
