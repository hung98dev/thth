using System.IO;
using NUnit.Framework;

namespace ThinhThan.Tests.EditMode.AssemblyGraph
{
    // CODE-001: csc.rsp enforces -warnaserror+ and -nullable:enable; no
    // committed .cs may suppress warnings or linters.
    public class CompilerSettingsTests
    {
        [Test]
        public void TestCscRspWarnAsErrorNullable()
        {
            var path = Path.Combine("Assets", "csc.rsp");
            Assert.IsTrue(File.Exists(path), "Assets/csc.rsp missing");
            var text = File.ReadAllText(path);
            Assert.IsTrue(text.Contains("-warnaserror+"), "csc.rsp must set -warnaserror+");
            Assert.IsTrue(text.Contains("-nullable:enable"), "csc.rsp must set -nullable:enable");
        }

        [Test]
        public void TestZeroCompilerWarnings()
        {
            // Warnings already fail compilation via -warnaserror+; this test
            // additionally rejects every textual suppression escape hatch.
            foreach (var path in Directory.GetFiles("Assets", "*.cs", SearchOption.AllDirectories))
            {
                // Generated C# under Assets/Scripts/Protocol/ is exempt —
                // CODE-004 requires its generated #pragma/#nullable header.
                if (path.Replace('\\', '/').StartsWith("Assets/Scripts/Protocol/"))
                {
                    continue;
                }
                var text = File.ReadAllText(path);
                // Needles are concatenated so this file does not trip its own
                // suppression scan.
                Assert.IsFalse(text.Contains("#pragma " + "warning " + "disable"),
                    path + ": #pragma " + "warning " + "disable forbidden");
                Assert.IsFalse(text.Contains("//lint:" + "file-ignore"),
                    path + ": //lint:" + "file-ignore forbidden");
                Assert.IsFalse(text.Contains("// ReSharper " + "disable"),
                    path + ": ReSharper suppression forbidden");
            }
        }
    }
}
