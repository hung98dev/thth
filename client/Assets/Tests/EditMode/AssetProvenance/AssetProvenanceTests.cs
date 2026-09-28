using System.Collections.Generic;
using System.IO;
using NUnit.Framework;
using ThinhThan.Core.Assets.Editor.AssetProduction;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.AssetProvenance
{
    // Provenance register validator tests (IMP-070). Fixtures build temp
    // repos under Path.GetTempPath; every test cleans up its directory so
    // no test-only imports or outputs survive (cleanup obligation).
    public class AssetProvenanceTests
    {
        private string _dir = "";

        [SetUp]
        public void SetUp()
        {
            _dir = Path.Combine(
                Path.GetTempPath(),
                "imp070_prov_" + System.Guid.NewGuid().ToString("N"));
            Directory.CreateDirectory(_dir);
        }

        [TearDown]
        public void TearDown()
        {
            if (Directory.Exists(_dir))
            {
                Directory.Delete(_dir, true);
            }
        }

        private string WriteMedia(string relPath, byte[] content)
        {
            var abs = Path.Combine(_dir, relPath);
            Directory.CreateDirectory(Path.GetDirectoryName(abs)!);
            File.WriteAllBytes(abs, content);
            return abs;
        }

        private static string Sha(byte[] content)
        {
            using var sha = System.Security.Cryptography.SHA256.Create();
            var h = sha.ComputeHash(content);
            var sb = new System.Text.StringBuilder(64);
            foreach (var b in h)
            {
                sb.Append(b.ToString("x2"));
            }
            return sb.ToString();
        }

        private static AssetSourceRow FreeRow(string path, string hash)
        {
            return new AssetSourceRow
            {
                asset_key = "asset.prop.example.world",
                file_path = path,
                content_id = null,
                source_kind = AssetProvenanceValidator.SourceKindFree,
                creator = "Example Author",
                source_uri = "https://example.com/art/example.png",
                license_id = AssetProvenanceValidator.LicenseCc0,
                license_uri = "https://creativecommons.org/publicdomain/zero/1.0/",
                acquired_at_utc = "2026-09-01T00:00:00Z",
                source_sha256 = hash,
                final_sha256 = hash,
                changes = "none",
                attribution = null,
                generation_record = null,
                inputs = new List<InputProvenance>(),
                review_state = AssetProvenanceValidator.ReviewApproved,
            };
        }

        private static AssetSourceRow AiRow(string path, string hash)
        {
            var row = FreeRow(path, hash);
            row.source_kind = AssetProvenanceValidator.SourceKindAi;
            row.source_uri = null;
            row.license_id = AssetProvenanceValidator.LicenseAiTool;
            row.license_uri = "https://example.com/tool-terms";
            row.generation_record = new GenerationRecord
            {
                tool = "example-diffusion",
                version = "1.2.3",
                terms_uri = "https://example.com/tool-terms",
                prompt = "painted-volume chibi sprite, top-front light",
                reference_uris = new List<string>(),
            };
            return row;
        }

        private static AssetSourceRegister RegisterOf(params AssetSourceRow[] rows)
        {
            return new AssetSourceRegister
            {
                schema_version = 1,
                assets = new List<AssetSourceRow>(rows),
            };
        }

        private static bool HasError(List<ProvenanceError> errors, string fragment)
        {
            foreach (var e in errors)
            {
                if (e.ToString().Contains(fragment))
                {
                    return true;
                }
            }
            return false;
        }

        [Test]
        public void TestEmptyRegisterPassesFoundation()
        {
            var errors = AssetProvenanceValidator.ValidateFiles(RegisterOf(), _dir);
            Assert.AreEqual(0, errors.Count, "clean empty register must pass foundation");
        }

        [Test]
        public void TestValidAiRow()
        {
            var rel = "client/Assets/Art/Actor/example.png";
            var content = new byte[] { 1, 2, 3, 4 };
            WriteMedia(rel, content);
            var errors = AssetProvenanceValidator.ValidateFiles(
                RegisterOf(AiRow(rel, Sha(content))), _dir);
            Assert.AreEqual(0, errors.Count);
        }

        [Test]
        public void TestValidCc0Row()
        {
            var rel = "client/Assets/Art/Prop/example.png";
            var content = new byte[] { 9, 8, 7 };
            WriteMedia(rel, content);
            var errors = AssetProvenanceValidator.ValidateFiles(
                RegisterOf(FreeRow(rel, Sha(content))), _dir);
            Assert.AreEqual(0, errors.Count);
        }

        [Test]
        public void TestValidCcByRow()
        {
            var rel = "client/Assets/Art/Prop/example_by.png";
            var content = new byte[] { 5, 5 };
            WriteMedia(rel, content);
            var row = FreeRow(rel, Sha(content));
            row.license_id = AssetProvenanceValidator.LicenseCcBy;
            row.license_uri = "https://creativecommons.org/licenses/by/4.0/";
            row.attribution = "Example Author, via example.com, CC-BY-4.0, cropped";
            var errors = AssetProvenanceValidator.ValidateFiles(RegisterOf(row), _dir);
            Assert.AreEqual(0, errors.Count);
        }

        [Test]
        public void TestValidOflFontRow()
        {
            var rel = "client/Assets/Art/Fonts/example_font.ttf";
            var content = new byte[] { 7, 7, 7 };
            WriteMedia(rel, content);
            var row = FreeRow(rel, Sha(content));
            row.license_id = AssetProvenanceValidator.LicenseOfl;
            row.license_uri = "https://openfontlicense.org/";
            row.attribution = "Example Font (c) Author, OFL-1.1 notice text";
            var errors = AssetProvenanceValidator.ValidateFiles(RegisterOf(row), _dir);
            Assert.AreEqual(0, errors.Count);
        }

        [Test]
        public void TestOflOnNonFontFails()
        {
            var rel = "client/Assets/Art/Prop/notafont.png";
            var content = new byte[] { 3 };
            WriteMedia(rel, content);
            var row = FreeRow(rel, Sha(content));
            row.license_id = AssetProvenanceValidator.LicenseOfl;
            row.attribution = "notice";
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "font"), "OFL-1.1 on a non-font file must fail");
        }

        [Test]
        public void TestMissingRowFailsReleaseMode()
        {
            var rel = "client/Assets/Art/Prop/unregistered.png";
            WriteMedia(rel, new byte[] { 1 });
            var assetsRoot = Path.Combine(_dir, "client", "Assets");
            var errors = AssetProvenanceValidator.ValidateRelease(RegisterOf(), _dir, assetsRoot);
            Assert.IsTrue(HasError(errors, "unregistered.png"),
                "a production file without a row must fail release mode");
        }

        [Test]
        public void TestOrphanRowFails()
        {
            var rel = "client/Assets/Art/Prop/missing.png";
            var errors = AssetProvenanceValidator.ValidateFiles(
                RegisterOf(FreeRow(rel, Sha(new byte[] { 0 }))), _dir);
            Assert.IsTrue(HasError(errors, "does not exist"),
                "a register row for a missing file must fail");
        }

        [Test]
        public void TestDuplicatePathFails()
        {
            var rel = "client/Assets/Art/Prop/dup.png";
            var errors = AssetProvenanceValidator.Validate(
                RegisterOf(FreeRow(rel, Sha(new byte[] { 1 })), FreeRow(rel, Sha(new byte[] { 2 }))));
            Assert.IsTrue(HasError(errors, "duplicate file_path"));
        }

        [Test]
        public void TestUnsortedRowsFail()
        {
            var a = FreeRow("client/Assets/Art/Prop/b.png", Sha(new byte[] { 1 }));
            var b = FreeRow("client/Assets/Art/Prop/a.png", Sha(new byte[] { 2 }));
            var errors = AssetProvenanceValidator.Validate(RegisterOf(a, b));
            Assert.IsTrue(HasError(errors, "sorted by file_path"));
        }

        [Test]
        public void TestInvalidAssetKeyFails()
        {
            var row = FreeRow("client/Assets/Art/Prop/x.png", Sha(new byte[] { 1 }));
            row.asset_key = "Not.A.Key!!";
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "asset_key"));
        }

        [Test]
        public void TestChangedFileHashFails()
        {
            var rel = "client/Assets/Art/Prop/changed.png";
            WriteMedia(rel, new byte[] { 9, 9, 9 });
            var row = FreeRow(rel, Sha(new byte[] { 0xDE }));
            row.changes = "none";
            var errors = AssetProvenanceValidator.ValidateFiles(RegisterOf(row), _dir);
            Assert.IsTrue(HasError(errors, "final_sha256"),
                "a disk hash mismatch must produce a path-specific error");
        }

        [Test]
        public void TestChangesNoneWithDifferentHashesFails()
        {
            var row = FreeRow("client/Assets/Art/Prop/x.png", Sha(new byte[] { 1 }));
            row.source_sha256 = Sha(new byte[] { 2 });
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "changes"));
        }

        [Test]
        public void TestDisallowedLicenseFails()
        {
            var row = FreeRow("client/Assets/Art/Prop/x.png", Sha(new byte[] { 1 }));
            row.license_id = "MIT";
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "license_id"));
        }

        [Test]
        public void TestPendingAndRejectedFailRelease()
        {
            var rel = "client/Assets/Art/Prop/pending.png";
            WriteMedia(rel, new byte[] { 4 });
            var assetsRoot = Path.Combine(_dir, "client", "Assets");
            var pending = FreeRow(rel, Sha(new byte[] { 4 }));
            pending.review_state = AssetProvenanceValidator.ReviewPending;
            var errors = AssetProvenanceValidator.ValidateRelease(
                RegisterOf(pending), _dir, assetsRoot);
            Assert.IsTrue(HasError(errors, "PENDING"));
            var rejected = FreeRow(rel, Sha(new byte[] { 4 }));
            rejected.review_state = AssetProvenanceValidator.ReviewRejected;
            errors = AssetProvenanceValidator.ValidateRelease(
                RegisterOf(rejected), _dir, assetsRoot);
            Assert.IsTrue(HasError(errors, "REJECTED"));
        }

        [Test]
        public void TestMissingAttributionFails()
        {
            var rel = "client/Assets/Art/Prop/by_noattr.png";
            var content = new byte[] { 6 };
            WriteMedia(rel, content);
            var row = FreeRow(rel, Sha(content));
            row.license_id = AssetProvenanceValidator.LicenseCcBy;
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "attribution"));
        }

        [Test]
        public void TestMissingFontNoticeFails()
        {
            var rel = "client/Assets/Art/Fonts/no_notice.ttf";
            var row = FreeRow(rel, Sha(new byte[] { 1 }));
            row.license_id = AssetProvenanceValidator.LicenseOfl;
            row.attribution = null;
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "attribution"));
        }

        [Test]
        public void TestAiRowRequiresGenerationRecord()
        {
            var row = AiRow("client/Assets/Art/Actor/x.png", Sha(new byte[] { 1 }));
            row.generation_record = null;
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "generation_record"));
        }

        [Test]
        public void TestAiRowForbidsSourceUri()
        {
            var row = AiRow("client/Assets/Art/Actor/x.png", Sha(new byte[] { 1 }));
            row.source_uri = "https://example.com/file.png";
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "source_uri"));
        }

        [Test]
        public void TestFreeRowRequiresSourceUri()
        {
            var row = FreeRow("client/Assets/Art/Prop/x.png", Sha(new byte[] { 1 }));
            row.source_uri = null;
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "source_uri"));
        }

        [Test]
        public void TestSearchUriRejected()
        {
            var row = FreeRow("client/Assets/Art/Prop/x.png", Sha(new byte[] { 1 }));
            row.source_uri = "https://www.google.com/search?q=some+sprite";
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "search URL"),
                "a search URL must not substitute for the origin URL");
        }

        [Test]
        public void TestGeneratedInputCannotBeHidden()
        {
            var row = AiRow("client/Assets/Art/Actor/x.png", Sha(new byte[] { 1 }));
            row.generation_record!.reference_uris =
                new List<string> { "https://example.com/ref.png" };
            var errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.IsTrue(HasError(errors, "cannot be hidden"),
                "reference URIs without inputs[] rows must fail");
            row.inputs = new List<InputProvenance>
            {
                new InputProvenance
                {
                    creator = "Ref Author",
                    source_uri = "https://example.com/ref.png",
                    license_id = AssetProvenanceValidator.LicenseCc0,
                    license_uri = "https://creativecommons.org/publicdomain/zero/1.0/",
                    acquired_at_utc = "2026-09-01T00:00:00Z",
                    sha256 = Sha(new byte[] { 42 }),
                },
            };
            errors = AssetProvenanceValidator.Validate(RegisterOf(row));
            Assert.AreEqual(0, errors.Count);
        }

        [Test]
        public void TestCreditsTextDeterministic()
        {
            var a = FreeRow("client/Assets/Art/Prop/b.png", Sha(new byte[] { 1 }));
            a.license_id = AssetProvenanceValidator.LicenseCcBy;
            a.attribution = "B Author";
            var c = FreeRow("client/Assets/Art/Fonts/f.ttf", Sha(new byte[] { 2 }));
            c.license_id = AssetProvenanceValidator.LicenseOfl;
            c.attribution = "F Font Notice";
            var b = FreeRow("client/Assets/Art/Prop/a.png", Sha(new byte[] { 3 }));
            b.license_id = AssetProvenanceValidator.LicenseCcBy;
            b.attribution = "A Author";
            var text1 = CreditsAuthoring.BuildCreditsText(RegisterOf(a, c, b));
            var text2 = CreditsAuthoring.BuildCreditsText(RegisterOf(b, a, c));
            Assert.AreEqual(text1, text2, "credits text must be deterministic");
            StringAssert.Contains("A Author", text1);
            StringAssert.Contains("F Font Notice", text1);
            Assert.Less(
                text1.IndexOf("A Author"), text1.IndexOf("B Author"),
                "CC-BY rows emit in file_path order");
        }

        [Test]
        public void TestRealRegisterLoadsAndValidates()
        {
            var repoRoot = Path.GetFullPath(Path.Combine(Application.dataPath, "..", ".."));
            var registerPath = Path.Combine(
                repoRoot, AssetSourceRegisterIO.RegisterRepoPath);
            var register = AssetSourceRegisterIO.Load(registerPath);
            Assert.NotNull(register, "committed register must parse");
            var errors = AssetProvenanceValidator.ValidateFiles(register!, repoRoot);
            Assert.AreEqual(0, errors.Count,
                "committed register must validate clean: "
                    + (errors.Count > 0 ? errors[0].ToString() : ""));
        }
    }
}
