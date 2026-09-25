using System.IO;
using System.Security.Cryptography;
using System.Text;
using NUnit.Framework;

namespace ThinhThan.Tests.EditMode.AssemblyGraph
{
    // Asserts the ProjectSettings baseline of repository_layout.md §
    // ProjectSettings Baseline: pre-declared config-object slots use the
    // path-derived GUID (first 32 lowercase hex of SHA-256 over the
    // repository-relative path), the ServerGeometry tag exists, incremental
    // GC + Android Optimized Frame Pacing are on, Physics2D simulates from
    // script. Config-object entries are asserted only when the referenced
    // asset exists (its owning packet creates it).
    public class ProjectSettingsBaselineTests
    {
        private const string AddressablePath = "client/Assets/AddressableAssetsData/AddressableAssetSettings.asset";
        private const string LocalizationPath = "client/Assets/Localization/Settings/LocalizationSettings.asset";
        private const string UrpPath = "client/Assets/Settings/Rendering/ThinhThanURP.asset";

        private static string PathGuid(string repoRelativePath)
        {
            using (var sha = SHA256.Create())
            {
                var bytes = sha.ComputeHash(Encoding.UTF8.GetBytes(repoRelativePath));
                var sb = new StringBuilder(64);
                foreach (var b in bytes)
                {
                    sb.Append(b.ToString("x2"));
                }
                return sb.ToString().Substring(0, 32);
            }
        }

        private static string AssetText(string name)
        {
            var path = Path.Combine("ProjectSettings", name);
            Assert.IsTrue(File.Exists(path), "ProjectSettings/" + name + " missing — materialization must commit it");
            return File.ReadAllText(path);
        }

        private static void AssertConfigSlot(string assetText, string key, string targetPath)
        {
            if (!File.Exists(targetPath.Substring("client/".Length)))
            {
                return;
            }
            var guid = PathGuid(targetPath);
            var idx = assetText.IndexOf(key, System.StringComparison.Ordinal);
            Assert.GreaterOrEqual(idx, 0, "m_configObjects entry missing: " + key);
            var window = assetText.Substring(idx, System.Math.Min(300, assetText.Length - idx));
            Assert.IsTrue(window.Contains(guid),
                key + " must reference path-derived GUID " + guid + " (" + targetPath + ")");
        }

        [Test]
        public void TestEditorBuildSettingsConfigObjects()
        {
            var text = AssetText("EditorBuildSettings.asset");
            AssertConfigSlot(text, "com.unity.addressableassets", AddressablePath);
            AssertConfigSlot(text, "com.unity.localization.settings", LocalizationPath);
        }

        [Test]
        public void TestGraphicsSettingsUrpSlot()
        {
            if (!File.Exists(UrpPath.Substring("client/".Length)))
            {
                return;
            }
            var text = AssetText("GraphicsSettings.asset");
            var idx = text.IndexOf("m_CustomRenderPipeline", System.StringComparison.Ordinal);
            Assert.GreaterOrEqual(idx, 0, "GraphicsSettings m_CustomRenderPipeline missing");
            var window = text.Substring(idx, System.Math.Min(200, text.Length - idx));
            Assert.IsTrue(window.Contains(PathGuid(UrpPath)),
                "m_CustomRenderPipeline must reference path-derived GUID of " + UrpPath);
        }

        [Test]
        public void TestServerGeometryTag()
        {
            var text = AssetText("TagManager.asset");
            Assert.IsTrue(text.Contains("ServerGeometry"), "TagManager.asset must declare the ServerGeometry tag");
        }

        [Test]
        public void TestPlayerAndPhysics2DBaseline()
        {
            var player = AssetText("ProjectSettings.asset");
            Assert.IsTrue(player.Contains("gcIncremental: 1"), "incremental GC must be enabled");
            var lower = player.ToLowerInvariant();
            Assert.IsTrue(lower.Contains("optimizedframepacing: 1"),
                "Android Optimized Frame Pacing must be enabled");
            var physics = AssetText("Physics2DSettings.asset");
            Assert.IsTrue(physics.Contains("m_SimulationMode: 2"),
                "Physics2D simulationMode must be Script (2)");
        }
    }
}
