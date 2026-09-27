using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using UnityEditor;
using UnityEditor.AddressableAssets.Settings;
using UnityEngine;

namespace ThinhThan.Tests.EditMode.AddressablesValidation
{
    public class AddressablesValidationTests
    {
        private const string SettingsPath = "Assets/AddressableAssetsData/AddressableAssetSettings.asset";
        private const string PinnedSettingsGuid = "03d05df79b43898f254cff5dad608436";

        [OneTimeSetUp]
        public void ProvisionCatalog()
        {
            // Idempotent ensure: the catalog must be provisioned before
            // assertions regardless of editor startup ordering.
            AddressablesProvisioner.Provision();
        }

        private static AddressableAssetSettings LoadSettings()
        {
            var settings = AssetDatabase.LoadAssetAtPath<AddressableAssetSettings>(SettingsPath);
            Assert.NotNull(settings, "AddressableAssetSettings.asset must exist at " + SettingsPath);
            return settings;
        }

        private static Dictionary<string, (string Group, string Guid)> CollectEntries(AddressableAssetSettings settings)
        {
            var map = new Dictionary<string, (string, string)>();
            foreach (var group in settings.groups)
            {
                foreach (var entry in group.entries)
                {
                    map[entry.address] = (group.Name, entry.guid);
                }
            }
            return map;
        }

        [Test]
        public void TestCatalogAssetKeyResolution()
        {
            var settings = LoadSettings();
            var entries = CollectEntries(settings);
            var plan = AssetKeyPlanner.Build(ContentCatalogScanner.DefaultDocsRoot);
            Assert.Greater(plan.Count, 0, "catalog plan must produce entries");
            foreach (var row in plan)
            {
                Assert.IsTrue(AssetKey.IsValid(row.Key), "invalid key grammar: " + row.Key);
                Assert.IsTrue(entries.ContainsKey(row.Key), "missing addressable entry for key " + row.Key);
                var (group, guid) = entries[row.Key];
                var path = AssetDatabase.GUIDToAssetPath(guid);
                Assert.IsFalse(string.IsNullOrEmpty(path), "entry has no resolvable asset path: " + row.Key);
                Assert.IsTrue(group == row.Group, "entry " + row.Key + " in group " + group + " expected " + row.Group);
            }
        }

        [Test]
        public void TestAddressableGroupBudgets()
        {
            var settings = LoadSettings();
            foreach (var group in settings.groups)
            {
                long ram = 0;
                long compressed = 0;
                foreach (var entry in group.entries)
                {
                    var path = AssetDatabase.GUIDToAssetPath(entry.guid);
                    ram += AssetRamEstimator.EntryBytes(path);
                    var fullPath = System.IO.Path.Combine(System.IO.Directory.GetCurrentDirectory(), path);
                    if (System.IO.File.Exists(fullPath))
                    {
                        compressed += new System.IO.FileInfo(fullPath).Length;
                    }
                }
                var ramBudget = (long)AddressableGroups.RamBudgetMb(group.Name) * 1024 * 1024;
                var compressedBudget = (long)AddressableGroups.CompressedBudgetMb(group.Name) * 1024 * 1024;
                Assert.LessOrEqual(ram, ramBudget, "group " + group.Name + " RAM over budget");
                Assert.LessOrEqual(compressed, compressedBudget, "group " + group.Name + " compressed over budget");
            }
        }

        [Test]
        public void TestCanonicalSpriteImportProfiles()
        {
            Assert.AreEqual(100, SpriteImportRules.PpuFor(PresentationAssetClass.Actor));
            Assert.AreEqual(100, SpriteImportRules.PpuFor(PresentationAssetClass.VfxSoft));
            Assert.AreEqual(100, SpriteImportRules.PpuFor(PresentationAssetClass.Tile));
            Assert.AreEqual(200, SpriteImportRules.PpuFor(PresentationAssetClass.UiArt));
            Assert.AreEqual(200, SpriteImportRules.PpuFor(PresentationAssetClass.ItemIcon));
            Assert.AreEqual(200, SpriteImportRules.PpuFor(PresentationAssetClass.EquipmentIcon));
            Assert.AreEqual(200, SpriteImportRules.PpuFor(PresentationAssetClass.FontAtlas));
            Assert.IsTrue(SpriteImportRules.PivotBottomCenter(PresentationAssetClass.Actor));
            Assert.IsFalse(SpriteImportRules.PivotBottomCenter(PresentationAssetClass.UiArt));
            Assert.AreEqual(4, SpriteImportRules.MobileAstcBlock(PresentationAssetClass.Actor));
            Assert.AreEqual(6, SpriteImportRules.MobileAstcBlock(PresentationAssetClass.ParallaxFar));
            var iconPath = AssetKeyPlanner.PlaceholdersRoot + AddressableGroups.IconsShared + "/asset.effect.basic.burn.icon.png";
            var importer = AssetImporter.GetAtPath(iconPath) as TextureImporter;
            if (importer != null)
            {
                Assert.AreEqual(TextureImporterType.Sprite, importer.textureType);
                Assert.AreEqual(SpriteImportRules.UiPpu, (int)importer.spritePixelsPerUnit);
                Assert.AreEqual(FilterMode.Bilinear, importer.filterMode);
                Assert.IsFalse(importer.mipmapEnabled);
            }
        }

        [Test]
        public void TestPlayableSceneKeyCoverage()
        {
            var settings = LoadSettings();
            var entries = CollectEntries(settings);
            var sceneCount = 0;
            foreach (var req in ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot))
            {
                if (!req.Facets.Contains(AssetFacet.Scene))
                {
                    continue;
                }
                sceneCount++;
                var sceneKey = AssetKey.FromCatalogId(req.CatalogId, AssetFacet.Scene);
                Assert.IsTrue(entries.ContainsKey(sceneKey), "missing scene entry for " + sceneKey);
                var bgmKey = AssetKey.FromCatalogId(req.CatalogId, AssetFacet.Bgm);
                Assert.IsTrue(entries.ContainsKey(bgmKey), "missing bgm alias for " + bgmKey);
            }
            // 24 maps + 5 dungeons + 1 finale + 2 pvp + 1 guild_war.
            Assert.AreEqual(33, sceneCount, "playable scene count must match the release roster");
        }

        [Test]
        public void TestSettingsAssetMatchesBaselineGuid()
        {
            var guid = AssetDatabase.AssetPathToGUID(SettingsPath);
            Assert.AreEqual(PinnedSettingsGuid, guid, "AddressableAssetSettings.asset GUID drifted from the repository_layout.md baseline");
            Assert.AreEqual(PinnedGuidOf(SettingsPath), guid);
            // Baseline contract is the serialized m_configObjects slot
            // (repository_layout.md § ProjectSettings Baseline, same
            // convention as ProjectSettingsBaselineTests): the
            // com.unity.addressableassets key must reference the
            // path-derived GUID of the settings asset.
            var ebs = File.ReadAllText(Path.Combine("ProjectSettings", "EditorBuildSettings.asset"));
            var idx = ebs.IndexOf("com.unity.addressableassets", System.StringComparison.Ordinal);
            Assert.GreaterOrEqual(idx, 0, "m_configObjects entry missing: com.unity.addressableassets");
            var window = ebs.Substring(idx, System.Math.Min(300, ebs.Length - idx));
            Assert.IsTrue(window.Contains(guid),
                "com.unity.addressableassets must reference path-derived GUID " + guid);
        }

        private static string PinnedGuidOf(string path)
        {
            // path-derived baseline GUID rule (repository_layout.md):
            // first 32 hex digits of SHA-256 over the repo-relative path.
            var repoRelative = "client/" + path;
            using (var sha = SHA256.Create())
            {
                var bytes = sha.ComputeHash(Encoding.UTF8.GetBytes(repoRelative));
                var sb = new StringBuilder(32);
                for (var i = 0; i < 16; i++)
                {
                    sb.Append(bytes[i].ToString("x2"));
                }
                return sb.ToString();
            }
        }
    }
}
