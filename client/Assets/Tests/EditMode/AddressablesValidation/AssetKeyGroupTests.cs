using System.Collections.Generic;
using NUnit.Framework;
using ThinhThan.Core.Assets;
using ThinhThan.Core.Assets.Editor;
using UnityEditor;
using UnityEditor.AddressableAssets.Settings;

namespace ThinhThan.Tests.EditMode.AddressablesValidation
{
    public class AssetKeyGroupTests
    {
        private const string SettingsPath = "Assets/AddressableAssetsData/AddressableAssetSettings.asset";

        [OneTimeSetUp]
        public void ProvisionCatalog()
        {
            AddressablesProvisioner.Provision();
        }

        private static AddressableAssetSettings LoadSettings()
        {
            return AssetDatabase.LoadAssetAtPath<AddressableAssetSettings>(SettingsPath);
        }

        [Test]
        public void TestKeyDerivationRule()
        {
            Assert.AreEqual("asset.monster.lang_da.dom_dom_ma.prefab", AssetKey.FromCatalogId("monster.lang_da.dom_dom_ma", AssetFacet.Prefab));
            Assert.AreEqual("asset.sfx.ui_confirm.clip", AssetKey.FromKind(AssetKeyKind.Sfx, "ui_confirm", AssetFacet.Clip));
            Assert.AreEqual("asset.bgm.zone.lang_da.bgm", AssetKey.FromKind(AssetKeyKind.Bgm, "zone.lang_da", AssetFacet.Bgm));

            Assert.IsTrue(AssetKey.IsValid("asset.item.eq.t1.dinh_lang.weapon.icon"));
            Assert.IsTrue(AssetKey.IsValid("asset.ui.boot_shell.sprite"));
            Assert.IsFalse(AssetKey.IsValid("asset.Monster.x.prefab"), "keys are lowercase only");
            Assert.IsFalse(AssetKey.IsValid("monster.lang_da.x.prefab"), "missing asset. prefix");
            Assert.IsFalse(AssetKey.IsValid("asset.item..icon"), "empty segment");
            Assert.IsFalse(AssetKey.IsValid("asset.item.abc"), "final segment is not a facet");
            Assert.IsFalse(AssetKey.IsValid("asset.item.abc.prefab.extra"), "no variant segment");
            Assert.IsFalse(AssetKey.IsValid(""), "empty key");

            Assert.IsTrue(AssetKey.TrySplit("asset.map.lang_da.dinh_lang.scene", out var id, out var facet, out var catalog));
            Assert.AreEqual("map.lang_da.dinh_lang", id);
            Assert.AreEqual(AssetFacet.Scene, facet);
            Assert.IsTrue(catalog);
            Assert.IsTrue(AssetKey.TrySplit("asset.bgm.zone.lang_da.bgm", out var id2, out var facet2, out var catalog2));
            Assert.AreEqual("bgm.zone.lang_da", id2);
            Assert.AreEqual(AssetFacet.Bgm, facet2);
            Assert.IsFalse(catalog2);
        }

        [Test]
        public void TestCanonicalGroupSetAndSingleMembership()
        {
            var settings = LoadSettings();
            var canonical = new HashSet<string>(AddressableGroups.CanonicalNames());
            Assert.AreEqual(25, canonical.Count);
            var seen = new HashSet<string>();
            var bossSpaces = BossSpaceMap();
            foreach (var group in settings.groups)
            {
                Assert.IsTrue(canonical.Contains(group.Name), "non-canonical group present: " + group.Name);
                foreach (var entry in group.entries)
                {
                    Assert.IsTrue(seen.Add(entry.address), "key in more than one group: " + entry.address);
                    var space = bossSpaces.TryGetValue(entry.address, out var s) ? s : null;
                    var want = KeyGroupRule.Assign(entry.address, space);
                    Assert.AreEqual(want, group.Name, "group mismatch for " + entry.address);
                }
            }
            Assert.AreEqual(canonical.Count, GroupCount(settings));
        }

        private static int GroupCount(AddressableAssetSettings settings)
        {
            var n = 0;
            foreach (var _ in settings.groups)
            {
                n++;
            }
            return n;
        }

        private static Dictionary<string, string> BossSpaceMap()
        {
            var map = new Dictionary<string, string>();
            foreach (var req in ContentCatalogScanner.Scan(ContentCatalogScanner.DefaultDocsRoot))
            {
                if (req.Kind == CatalogAssetKind.Boss && req.SpaceId.Length > 0)
                {
                    foreach (var facet in req.Facets)
                    {
                        map[AssetKey.FromCatalogId(req.CatalogId, facet)] = req.SpaceId;
                    }
                }
            }
            return map;
        }

        [Test]
        public void TestPresentationAliasSingleHop()
        {
            var settings = LoadSettings();
            var byAddress = new Dictionary<string, AddressableAssetEntry>();
            foreach (var group in settings.groups)
            {
                foreach (var entry in group.entries)
                {
                    byAddress[entry.address] = entry;
                }
            }
            var aliasCount = 0;
            foreach (var kv in byAddress)
            {
                var path = AssetDatabase.GUIDToAssetPath(kv.Value.guid);
                var alias = AssetDatabase.LoadAssetAtPath<PresentationAlias>(path);
                if (alias == null)
                {
                    continue;
                }
                aliasCount++;
                Assert.IsTrue(byAddress.ContainsKey(alias.TargetKey), "alias target missing: " + kv.Key + " -> " + alias.TargetKey);
                var targetPath = AssetDatabase.GUIDToAssetPath(byAddress[alias.TargetKey].guid);
                Assert.IsNull(
                    AssetDatabase.LoadAssetAtPath<PresentationAlias>(targetPath),
                    "alias to alias is forbidden: " + kv.Key + " -> " + alias.TargetKey);
            }
            Assert.AreEqual(33, aliasCount, "every playable scene carries a one-hop bgm alias");
        }

        [Test]
        public void TestDeterministicGroupRamBudgets()
        {
            var settings = LoadSettings();
            var totals = GroupRamTotals(settings);
            var totals2 = GroupRamTotals(settings);
            foreach (var kv in totals)
            {
                Assert.AreEqual(totals2[kv.Key], kv.Value, "RAM model must be deterministic for " + kv.Key);
                var budget = (long)AddressableGroups.RamBudgetMb(kv.Key) * 1024 * 1024;
                Assert.LessOrEqual(kv.Value, budget, "group " + kv.Key + " RAM over §1 budget");
            }
        }

        private static Dictionary<string, long> GroupRamTotals(AddressableAssetSettings settings)
        {
            var totals = new Dictionary<string, long>();
            foreach (var group in settings.groups)
            {
                long sum = 0;
                foreach (var entry in group.entries)
                {
                    sum += AssetRamEstimator.EntryBytes(AssetDatabase.GUIDToAssetPath(entry.guid));
                }
                totals[group.Name] = sum;
            }
            return totals;
        }

        [Test]
        public void TestResidentSteadyAndTransferPeak()
        {
            var settings = LoadSettings();
            var totals = GroupRamTotals(settings);
            long resident = 0;
            long largestTransient = 0;
            long baseCompressed = 0;
            foreach (var kv in totals)
            {
                if (AddressableGroups.IsResident(kv.Key))
                {
                    resident += kv.Value;
                }
                else if (kv.Value > largestTransient)
                {
                    largestTransient = kv.Value;
                }
            }
            foreach (var group in settings.groups)
            {
                if (!AddressableGroups.IsLocal(group.Name))
                {
                    continue;
                }
                foreach (var entry in group.entries)
                {
                    var path = System.IO.Path.Combine(System.IO.Directory.GetCurrentDirectory(), AssetDatabase.GUIDToAssetPath(entry.guid));
                    if (System.IO.File.Exists(path))
                    {
                        baseCompressed += new System.IO.FileInfo(path).Length;
                    }
                }
            }
            // §1: resident steady-state <= 450 MB.
            Assert.LessOrEqual(resident, 450L * 1024 * 1024, "resident steady-state over budget");
            // §1: transfer peak = resident + largest transient <= 570 MB.
            Assert.LessOrEqual(resident + largestTransient, 570L * 1024 * 1024, "transfer peak over budget");
            // §1: base install (bootstrap + shared) <= 82 MB compressed.
            Assert.LessOrEqual(baseCompressed, 82L * 1024 * 1024, "base install over budget");
        }

        [Test]
        public void TestMeshTypeRule()
        {
            Assert.IsTrue(SpriteImportRules.MeshTightExpected(256, 128, true), "long side 256 + margins -> Tight");
            Assert.IsTrue(SpriteImportRules.MeshTightExpected(640, 640, true));
            Assert.IsFalse(SpriteImportRules.MeshTightExpected(128, 128, true), "long side < 256 -> Full Rect");
            Assert.IsFalse(SpriteImportRules.MeshTightExpected(512, 512, false), "no transparent margins -> Full Rect");
            Assert.IsFalse(SpriteImportRules.MeshTightExpected(255, 255, true), "255 < 256 -> Full Rect");
        }

        [Test]
        public void TestDungeonZoneMappingMatchesActTable()
        {
            var act = ContentCatalogScanner.ScanActTable(ContentCatalogScanner.DefaultDocsRoot);
            Assert.AreEqual(5, act.Count, "encounter_catalog act table must map 5 dungeons");
            foreach (var (zone, dungeon) in act)
            {
                Assert.AreEqual(zone.Substring("zone.".Length), AddressableGroups.DungeonZone(dungeon));
            }
            Assert.AreEqual("nui_thieng", AddressableGroups.DungeonZone("instance.finale.than_trung"));
        }

        [Test]
        public void TestParallaxFarPpu50()
        {
            Assert.AreEqual(50, SpriteImportRules.PpuFor(PresentationAssetClass.ParallaxFar));
            Assert.AreEqual(100, SpriteImportRules.PpuFor(PresentationAssetClass.ParallaxNear));
        }
    }
}
