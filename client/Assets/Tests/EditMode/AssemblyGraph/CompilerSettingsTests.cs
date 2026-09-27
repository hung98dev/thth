using System.IO;
using NUnit.Framework;

namespace ThinhThan.Tests.EditMode.AssemblyGraph
{
    // CODE-001: every first-party assembly is compiled by its own csc.rsp
    // beside the .asmdef carrying -warnaserror+ and -nullable:enable; a root
    // Assets/csc.rsp is forbidden because a global response file also applies
    // to Library/PackageCache package sources, which are not nullable-clean
    // (BLK-007). No committed .cs may suppress warnings or linters.
    public class CompilerSettingsTests
    {
        [Test]
        public void TestCscRspWarnAsErrorNullable()
        {
            Assert.IsFalse(File.Exists(Path.Combine("Assets", "csc.rsp")),
                "Assets/csc.rsp must not exist — flags are scoped per-asmdef");
            var asmdefs = Directory.GetFiles("Assets", "*.asmdef", SearchOption.AllDirectories);
            Assert.GreaterOrEqual(asmdefs.Length, 13, "mandatory asmdefs missing");
            foreach (var asmdef in asmdefs)
            {
                var path = Path.Combine(Path.GetDirectoryName(asmdef), "csc.rsp");
                Assert.IsTrue(File.Exists(path), path + ": asmdef-scoped csc.rsp missing");
                var text = File.ReadAllText(path);
                Assert.IsTrue(text.Contains("-warnaserror+"), path + " must set -warnaserror+");
                Assert.IsTrue(text.Contains("-nullable:enable"), path + " must set -nullable:enable");
                Assert.IsFalse(text.Contains("-nullable:disable"), path + " must not disable nullable");
            }
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
