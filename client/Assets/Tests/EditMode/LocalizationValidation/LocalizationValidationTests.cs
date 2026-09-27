using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using System.Text.RegularExpressions;
using NUnit.Framework;
using ThinhThan.Core.Localization;
using UnityEditor;
using UnityEngine.Localization.Tables;

namespace ThinhThan.Tests.EditMode.LocalizationValidation
{
    /// <summary>
    /// IMP-064 bilingual localization gate (client_localization.md):
    /// every launch content key must resolve in both vi-VN and en-US, and a
    /// missing or fallback-only key must fail validation rather than leak.
    /// </summary>
    public class LocalizationValidationTests
    {
        const string SharedDataPath = "Assets/Localization/Tables/Core/Core Shared Table Data.asset";
        const string ViTablePath = "Assets/Localization/Tables/Core/Core_vi-VN.asset";
        const string EnTablePath = "Assets/Localization/Tables/Core/Core_en-US.asset";
        const string SettingsAssetPath = "Assets/Localization/Settings/LocalizationSettings.asset";
        const string SettingsRepoPath = "client/Assets/Localization/Settings/LocalizationSettings.asset";

        static readonly Regex KeyShape = new Regex(
            "^loc\\.[a-z0-9_]+(\\.[a-z0-9_]+)+$", RegexOptions.CultureInvariant);
        static readonly Regex SmartArg = new Regex(
            "\\{([A-Za-z_][A-Za-z0-9_.]*)", RegexOptions.CultureInvariant);

        static SharedTableData Shared()
        {
            return AssetDatabase.LoadAssetAtPath<SharedTableData>(SharedDataPath);
        }

        static StringTable Table(string path)
        {
            return AssetDatabase.LoadAssetAtPath<StringTable>(path);
        }

        static string PathGuid(string repoRelativePath)
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

        [Test]
        public void TestBilingualKeyParity()
        {
            SharedTableData shared = Shared();
            Assert.NotNull(shared, "Shared table data missing at " + SharedDataPath);
            StringTable vi = Table(ViTablePath);
            StringTable en = Table(EnTablePath);
            Assert.NotNull(vi, "vi-VN table missing");
            Assert.NotNull(en, "en-US table missing");
            Assert.AreEqual(LocKeys.LocaleViVn, vi.LocaleIdentifier.Code);
            Assert.AreEqual(LocKeys.LocaleEnUs, en.LocaleIdentifier.Code);

            var seenKeys = new HashSet<string>();
            foreach (SharedTableData.SharedTableEntry e in shared.Entries)
            {
                StringAssert.StartsWith("loc.", e.Key);
                Assert.IsTrue(KeyShape.IsMatch(e.Key),
                    "Malformed loc key: " + e.Key);
                Assert.IsTrue(seenKeys.Add(e.Key), "Duplicate loc key: " + e.Key);

                StringTableEntry v = vi.GetEntry(e.Id);
                StringTableEntry n = en.GetEntry(e.Id);
                Assert.IsNotNull(v, "vi-VN missing entry: " + e.Key);
                Assert.IsNotNull(n, "en-US missing entry: " + e.Key);
                Assert.IsFalse(string.IsNullOrEmpty(v.Value), "vi-VN empty: " + e.Key);
                Assert.IsFalse(string.IsNullOrEmpty(n.Value), "en-US empty: " + e.Key);
                Assert.IsFalse(v.Value.Contains(LocKeys.MissingPrefix),
                    "vi-VN stores a fallback marker: " + e.Key);
                Assert.IsFalse(n.Value.Contains(LocKeys.MissingPrefix),
                    "en-US stores a fallback marker: " + e.Key);

                // Smart String argument parity: both locales must expose the
                // same placeholder names so a locale switch cannot lose a value.
                var argsVi = new SortedSet<string>();
                var argsEn = new SortedSet<string>();
                foreach (Match m in SmartArg.Matches(v.Value)) argsVi.Add(m.Groups[1].Value);
                foreach (Match m in SmartArg.Matches(n.Value)) argsEn.Add(m.Groups[1].Value);
                CollectionAssert.AreEquivalent(argsVi, argsEn,
                    "Smart arg drift at " + e.Key);
            }
            Assert.Greater(seenKeys.Count, 0, "Core table is empty");
        }

        [Test]
        public void TestNoMissingTranslations()
        {
            SortedSet<string> expected = LocCatalogExtractor.ExpectedKeys();
            Assert.Greater(expected.Count, 0, "Extractor produced no keys");

            SharedTableData shared = Shared();
            Assert.NotNull(shared, "Shared table data missing");
            var actual = new SortedSet<string>();
            foreach (SharedTableData.SharedTableEntry e in shared.Entries)
            {
                actual.Add(e.Key);
            }

            var missing = new List<string>();
            foreach (string k in expected)
            {
                if (!actual.Contains(k)) missing.Add(k);
            }
            var extra = new List<string>();
            foreach (string k in actual)
            {
                if (!expected.Contains(k)) extra.Add(k);
            }
            Assert.IsEmpty(missing,
                "Catalog keys absent from Core tables: " + string.Join(", ", missing));
            Assert.IsEmpty(extra,
                "Core keys with no catalog source: " + string.Join(", ", extra));
        }

        [Test]
        public void TestSettingsAssetMatchesBaselineGuid()
        {
            // repository_layout.md § ProjectSettings Baseline: the
            // LocalizationSettings config-object slot in EditorBuildSettings is
            // pre-declared with the path-derived GUID, so the committed asset's
            // .meta must carry exactly that GUID.
            var settings = AssetDatabase.LoadAssetAtPath<UnityEngine.Localization.Settings.LocalizationSettings>(SettingsAssetPath);
            Assert.NotNull(settings, "LocalizationSettings missing at " + SettingsAssetPath);
            string guid = AssetDatabase.AssetPathToGUID(SettingsAssetPath);
            Assert.AreEqual(PathGuid(SettingsRepoPath), guid,
                "LocalizationSettings.asset meta GUID does not match the pre-declared baseline GUID");
        }

        [Test]
        public void TestMissingMarkerContract()
        {
            Assert.AreEqual("[MISSING:loc.class.kim.name]",
                Loc.Missing("loc.class.kim.name"));
            Assert.IsTrue(LocKeys.IsSupportedLocale(LocKeys.LocaleViVn));
            Assert.IsTrue(LocKeys.IsSupportedLocale(LocKeys.LocaleEnUs));
            Assert.IsFalse(LocKeys.IsSupportedLocale("fr-FR"));
        }
    }
}
