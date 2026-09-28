using System.Collections.Generic;
using System.IO;
using System.Text;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Deterministic credits generation (section 6): every CC-BY-4.0
    // attribution and every OFL-1.1 font notice is produced from the
    // register, ordered by file_path, so the released credits text is
    // reproducible byte-for-byte from the register alone.
    public static class CreditsAuthoring
    {
        public const string CreditsOutputPath =
            "Library/AssetProvenance/asset_credits.txt";

        public static string BuildCreditsText(AssetSourceRegister register)
        {
            var rows = new List<AssetSourceRow>(register.assets ?? new List<AssetSourceRow>());
            rows.Sort((a, b) => string.CompareOrdinal(a.file_path ?? "", b.file_path ?? ""));
            var sb = new StringBuilder();
            sb.Append("Asset Credits\n");
            sb.Append("=============\n\n");
            sb.Append("CC-BY-4.0 Attributions\n");
            sb.Append("----------------------\n");
            var anyBy = false;
            foreach (var row in rows)
            {
                if (row.license_id != AssetProvenanceValidator.LicenseCcBy
                    || row.review_state != AssetProvenanceValidator.ReviewApproved)
                {
                    continue;
                }
                anyBy = true;
                sb.Append(row.attribution).Append(" (").Append(row.file_path).Append(")\n");
            }
            if (!anyBy)
            {
                sb.Append("(none)\n");
            }
            sb.Append('\n');
            sb.Append("OFL-1.1 Font Notices\n");
            sb.Append("--------------------\n");
            var anyOfl = false;
            foreach (var row in rows)
            {
                if (row.license_id != AssetProvenanceValidator.LicenseOfl
                    || row.review_state != AssetProvenanceValidator.ReviewApproved)
                {
                    continue;
                }
                anyOfl = true;
                sb.Append(row.attribution).Append(" (").Append(row.file_path).Append(")\n");
            }
            if (!anyOfl)
            {
                sb.Append("(none)\n");
            }
            return sb.ToString();
        }

        // Writes the deterministic credits text under Library/ (transient,
        // gitignored; packaged into player credits by a later task).
        public static string WriteCredits(AssetSourceRegister register, string projectRoot)
        {
            var outPath = Path.Combine(projectRoot, CreditsOutputPath);
            Directory.CreateDirectory(Path.GetDirectoryName(outPath)!);
            File.WriteAllText(outPath, BuildCreditsText(register));
            return outPath;
        }

        // CI entry point: validate the committed register and emit credits.
        // Returns 0 on success; on failure logs every path-specific error.
        public static int VerifyProvenanceAndWriteCredits(string repoRoot, string projectRoot)
        {
            var registerPath = Path.Combine(
                repoRoot, AssetSourceRegisterIO.RegisterRepoPath);
            var register = AssetSourceRegisterIO.LoadOrEmpty(registerPath);
            var errors = AssetProvenanceValidator.ValidateFiles(register, repoRoot);
            foreach (var e in errors)
            {
                Debug.LogError("provenance: " + e);
            }
            if (errors.Count > 0)
            {
                return 1;
            }
            var outPath = WriteCredits(register, projectRoot);
            Debug.Log("provenance: wrote credits to " + outPath);
            return 0;
        }
    }
}
