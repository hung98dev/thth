using System.IO;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Loading and path helpers for the source register. Register JSON is
    // strict-typed through JsonUtility; structural problems (schema_version,
    // ordering, field rules) are reported by AssetProvenanceValidator as
    // path-specific errors instead of throwing here.
    public static class AssetSourceRegisterIO
    {
        public const string RegisterRepoPath =
            "client/Assets/Art/Provenance/asset_source_register.json";
        public const string FragmentsRepoDir =
            "client/Assets/Art/Provenance/fragments";

        public static AssetSourceRegister? Load(string absPath)
        {
            if (!File.Exists(absPath))
            {
                return null;
            }
            var json = File.ReadAllText(absPath);
            AssetSourceRegister? register;
            try
            {
                register = JsonUtility.FromJson<AssetSourceRegister>(json);
            }
            catch (System.ArgumentException)
            {
                return null;
            }
            return register;
        }

        public static AssetSourceRegister LoadOrEmpty(string absPath)
        {
            return Load(absPath) ?? new AssetSourceRegister();
        }

        // Normalized repo path: forward slashes, no leading/trailing slash,
        // no empty / "." / ".." segments, no backslash or drive colon.
        public static bool IsNormalizedRepoPath(string path)
        {
            if (string.IsNullOrEmpty(path)
                || path.IndexOf('\\') >= 0
                || path.IndexOf(':') >= 0
                || path.StartsWith("/")
                || path.EndsWith("/"))
            {
                return false;
            }
            foreach (var seg in path.Split('/'))
            {
                if (seg.Length == 0 || seg == "." || seg == "..")
                {
                    return false;
                }
            }
            return true;
        }

        public static string ToRepoRelative(string repoRoot, string absPath)
        {
            var fullRoot = Path.GetFullPath(repoRoot);
            var fullPath = Path.GetFullPath(absPath);
            var rel = fullPath.StartsWith(fullRoot)
                ? fullPath.Substring(fullRoot.Length)
                : fullPath;
            return rel.Replace('\\', '/').TrimStart('/');
        }

        public static string RepoToAbsolute(string repoRoot, string repoPath)
        {
            return Path.GetFullPath(Path.Combine(repoRoot, repoPath));
        }
    }
}
