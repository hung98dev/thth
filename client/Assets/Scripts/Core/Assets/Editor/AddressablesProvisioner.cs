using System;
using System.Collections.Generic;
using System.IO;
using System.Reflection;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEditor.AddressableAssets.Settings.GroupSchemas;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor
{
    // Deterministic provisioner for the Addressables catalog
    // (IMP-063): creates the canonical 25 groups with their schemas,
    // profiles, entries and placeholder backing files, idempotently, on
    // every editor load. The settings asset lives at the pinned path
    // declared in repository_layout.md; EditorBuildSettings already
    // references it by GUID, so the package never rewrites
    // ProjectSettings. The private s_DefaultSettingsObject slot is
    // injected so package build paths (BuildPlayerContent) resolve the
    // settings without a DefaultObject wrapper asset.
    public static class AddressablesProvisioner
    {
        public const string SettingsPath = "Assets/AddressableAssetsData/AddressableAssetSettings.asset";

        // Per-environment remote base URLs are placeholders: deployment
        // wiring is owned by a later ops task; the placeholder keeps the
        // Remote.LoadPath variable well formed.
        private static readonly (string Profile, string RemoteBase)[] _profiles =
        {
            ("dev", "https://assets.dev.example.com/thinhthan/"),
            ("staging", "https://assets.staging.example.com/thinhthan/"),
            ("production", "https://assets.example.com/thinhthan/"),
        };

        [InitializeOnLoadMethod]
        private static void AutoProvision()
        {
            // Provisioning inside the ReloadAssemblies window races the
            // AssetDatabase importer (batchmode self-SIGKILL during
            // "Registering precompiled unity dll's" on Linux). Defer to
            // the first editor update: still before -quit exits and
            // before the EditMode runner starts.
            EditorApplication.delayCall += ProvisionSafely;
        }

        private static void ProvisionSafely()
        {
            try
            {
                Provision();
            }
            catch (Exception e)
            {
                Debug.LogError("Addressables provisioning failed: " + e);
            }
        }

        public static void Provision()
        {
            var settings = EnsureSettings();
            if (settings == null)
            {
                return;
            }
            InjectDefaultSettings(settings);
            ValidateSettings(settings);
            var dirty = false;
            dirty |= EnsureCatalogFlags(settings);
            dirty |= EnsureProfiles(settings);
            dirty |= EnsureGroups(settings);
            dirty |= RehomePackageGroups(settings);
            dirty |= RestoreEditorBuildSettingsSlot(settings);
            dirty |= EnsureEntries(settings);
            if (dirty)
            {
                EditorUtility.SetDirty(settings);
                AssetDatabase.SaveAssets();
            }
        }

        public static AddressableAssetSettings? LoadSettings()
        {
            return AssetDatabase.LoadAssetAtPath<AddressableAssetSettings>(SettingsPath);
        }

        private static AddressableAssetSettings? EnsureSettings()
        {
            var settings = LoadSettings();
            if (settings != null)
            {
                return settings;
            }
            var dir = Path.GetDirectoryName(SettingsPath);
            if (dir != null && !AssetDatabase.IsValidFolder(dir))
            {
                AssetDatabase.CreateFolder("Assets", "AddressableAssetsData");
            }
            settings = ScriptableObject.CreateInstance<AddressableAssetSettings>();
            AssetDatabase.CreateAsset(settings, SettingsPath);
            return settings;
        }

        public static void InjectDefaultSettings(AddressableAssetSettings settings)
        {
            // AddressableAssetSettingsDefaultObject.Settings drives
            // BuildPlayerContent; the private static slot is set
            // directly so no DefaultObject asset has to exist.
            var field = typeof(AddressableAssetSettingsDefaultObject).GetField(
                "s_DefaultSettingsObject",
                BindingFlags.NonPublic | BindingFlags.Static);
            if (field == null)
            {
                throw new InvalidOperationException("s_DefaultSettingsObject not found on AddressableAssetSettingsDefaultObject");
            }
            if (!ReferenceEquals(field.GetValue(null), settings))
            {
                field.SetValue(null, settings);
            }
        }

        private static void ValidateSettings(AddressableAssetSettings settings)
        {
            // Validate() (private) materializes the default profile,
            // data builders and group templates the build path needs.
            var method = typeof(AddressableAssetSettings).GetMethod("Validate", BindingFlags.NonPublic | BindingFlags.Instance);
            if (method == null)
            {
                throw new InvalidOperationException("AddressableAssetSettings.Validate not found");
            }
            method.Invoke(settings, null);
        }

        private static bool EnsureCatalogFlags(AddressableAssetSettings settings)
        {
            var dirty = false;
            if (!settings.BuildRemoteCatalog)
            {
                settings.BuildRemoteCatalog = true;
                dirty = true;
            }
            if (!settings.EnableJsonCatalog)
            {
                settings.EnableJsonCatalog = true;
                dirty = true;
            }
            if (!settings.UniqueBundleIds)
            {
                settings.UniqueBundleIds = true;
                dirty = true;
            }
            return dirty;
        }

        private static bool EnsureProfiles(AddressableAssetSettings settings)
        {
            var dirty = false;
            var profileSettings = settings.profileSettings;
            var existing = new HashSet<string>(profileSettings.GetAllProfileNames());
            string devId = profileSettings.GetProfileId("dev");
            if (string.IsNullOrEmpty(devId))
            {
                devId = profileSettings.AddProfile("dev", null);
                dirty = true;
            }
            foreach (var (name, remoteBase) in _profiles)
            {
                var id = profileSettings.GetProfileId(name);
                if (string.IsNullOrEmpty(id))
                {
                    id = profileSettings.AddProfile(name, devId);
                    dirty = true;
                }
                dirty |= EnsureProfileVariables(profileSettings, id, remoteBase);
            }
            var activeId = profileSettings.GetProfileId("dev");
            if (settings.activeProfileId != activeId)
            {
                settings.activeProfileId = activeId;
                dirty = true;
            }
            return dirty;
        }

        private static bool EnsureProfileVariables(AddressableAssetProfileSettings profileSettings, string profileId, string remoteBase)
        {
            var dirty = false;
            dirty |= EnsureVariableValue(profileSettings, profileId, AddressableAssetSettings.kLocalBuildPath, AddressableAssetSettings.kLocalBuildPathValue);
            dirty |= EnsureVariableValue(profileSettings, profileId, AddressableAssetSettings.kLocalLoadPath, AddressableAssetSettings.kLocalLoadPathValue);
            dirty |= EnsureVariableValue(profileSettings, profileId, AddressableAssetSettings.kRemoteBuildPath, "ServerData/[BuildTarget]");
            dirty |= EnsureVariableValue(profileSettings, profileId, AddressableAssetSettings.kRemoteLoadPath, remoteBase + "[BuildTarget]");
            return dirty;
        }

        private static bool EnsureVariableValue(AddressableAssetProfileSettings profileSettings, string profileId, string name, string value)
        {
            var existing = profileSettings.GetValueByName(profileId, name);
            if (existing == value)
            {
                return false;
            }
            // CreateValue is a no-op when the variable already exists;
            // SetValue then applies the per-profile value.
            profileSettings.CreateValue(name, value);
            profileSettings.SetValue(profileId, name, value);
            return true;
        }

        private static bool EnsureGroups(AddressableAssetSettings settings)
        {
            var dirty = false;
            foreach (var name in AddressableGroups.CanonicalNames())
            {
                var group = settings.FindGroup(name);
                if (group == null)
                {
                    group = settings.CreateGroup(name, false, false, false, null, typeof(BundledAssetGroupSchema), typeof(ContentUpdateGroupSchema));
                    dirty = true;
                }
                dirty |= ConfigureGroup(settings, group, name);
            }
            return dirty;
        }

        private static bool ConfigureGroup(AddressableAssetSettings settings, AddressableAssetGroup group, string name)
        {
            var dirty = false;
            var bundled = group.GetSchema<BundledAssetGroupSchema>();
            if (bundled == null)
            {
                bundled = group.AddSchema<BundledAssetGroupSchema>();
                dirty = true;
            }
            var update = group.GetSchema<ContentUpdateGroupSchema>();
            if (update == null)
            {
                group.AddSchema<ContentUpdateGroupSchema>();
                dirty = true;
            }
            var wantLocal = AddressableGroups.IsLocal(name);
            var pathName = wantLocal ? AddressableAssetSettings.kLocalBuildPath : AddressableAssetSettings.kRemoteBuildPath;
            var loadName = wantLocal ? AddressableAssetSettings.kLocalLoadPath : AddressableAssetSettings.kRemoteLoadPath;
            if (bundled.BuildPath.GetName(settings) != pathName)
            {
                dirty |= bundled.BuildPath.SetVariableByName(settings, pathName);
            }
            if (bundled.LoadPath.GetName(settings) != loadName)
            {
                dirty |= bundled.LoadPath.SetVariableByName(settings, loadName);
            }
            if (bundled.Compression != BundledAssetGroupSchema.BundleCompressionMode.LZ4)
            {
                bundled.Compression = BundledAssetGroupSchema.BundleCompressionMode.LZ4;
                dirty = true;
            }
            var wantMode = AddressableGroups.IsPackSeparately(name)
                ? BundledAssetGroupSchema.BundlePackingMode.PackSeparately
                : BundledAssetGroupSchema.BundlePackingMode.PackTogether;
            if (bundled.BundleMode != wantMode)
            {
                bundled.BundleMode = wantMode;
                dirty = true;
            }
            if (!bundled.IncludeInBuild)
            {
                bundled.IncludeInBuild = true;
                dirty = true;
            }
            if (update != null && update.StaticContent)
            {
                update.StaticContent = false;
                dirty = true;
            }
            return dirty;
        }

        // BLK-011 / ADR-0074: com.unity.localization creates
        // `Localization-*` groups during OnPostprocessAllAssets, before
        // this provisioner runs. Entries are moved into the canonical
        // localization.* groups (labels/addresses preserved — the
        // package resolves by them) and the empty package groups are
        // removed, so committed settings contain canonical groups only.
        private static bool RehomePackageGroups(AddressableAssetSettings settings)
        {
            var dirty = false;
            var groups = new List<AddressableAssetGroup>(settings.groups);
            foreach (var group in groups)
            {
                if (group == null || AddressableGroups.IsCanonical(group.Name))
                {
                    continue;
                }
                if (!group.Name.StartsWith("Localization-", System.StringComparison.Ordinal))
                {
                    continue;
                }
                var fallback = group.Name == "Localization-Locales"
                    ? AddressableGroups.LocalizationLocales
                    : AddressableGroups.LocalizationShared;
                var entries = new List<AddressableAssetEntry>(group.entries);
                foreach (var entry in entries)
                {
                    var targetName = fallback;
                    if (group.Name.StartsWith("Localization-String-Tables-", System.StringComparison.Ordinal))
                    {
                        var localeKey = LocaleKeyOf(entry.address);
                        targetName = localeKey == null
                            ? AddressableGroups.LocalizationShared
                            : AddressableGroups.LocalizationStringsGroup(localeKey);
                    }
                    var target = settings.FindGroup(targetName);
                    if (target != null)
                    {
                        settings.MoveEntry(entry, target);
                        dirty = true;
                    }
                }
                if (group.entries.Count == 0)
                {
                    settings.RemoveGroup(group);
                    dirty = true;
                }
            }
            return dirty;
        }

        // String-table entry addresses end in _<locale code>
        // (e.g. Core_vi-VN); the canonical key lowercases it and maps
        // '-' to '_'.
        private static string? LocaleKeyOf(string address)
        {
            var idx = address.LastIndexOf('_');
            if (idx < 0 || idx == address.Length - 1)
            {
                return null;
            }
            var code = address.Substring(idx + 1).ToLowerInvariant().Replace('-', '_');
            return System.Array.IndexOf(AddressableGroups.LocaleKeys, code) >= 0 ? code : null;
        }

        // BLK-011 / ADR-0074: the localization postprocessor calls
        // GetSettings(true) before the injected slot exists, creating
        // DefaultObject.asset and repointing the EBS config object at it.
        // Converge back: the slot always references the canonical
        // settings asset (its pinned path-derived GUID) and the rogue
        // DefaultObject asset is deleted.
        private static bool RestoreEditorBuildSettingsSlot(AddressableAssetSettings settings)
        {
            var dirty = false;
            if (!EditorBuildSettings.TryGetConfigObject(
                    "com.unity.addressableassets", out AddressableAssetSettings current)
                || !ReferenceEquals(current, settings))
            {
                EditorBuildSettings.AddConfigObject("com.unity.addressableassets", settings, true);
                dirty = true;
            }
            if (AssetDatabase.LoadAssetAtPath<UnityEngine.Object>(
                    "Assets/AddressableAssetsData/DefaultObject.asset") != null)
            {
                AssetDatabase.DeleteAsset("Assets/AddressableAssetsData/DefaultObject.asset");
                dirty = true;
            }
            return dirty;
        }

        private static bool EnsureEntries(AddressableAssetSettings settings)
        {
            var dirty = false;
            var docsRoot = ContentCatalogScanner.DefaultDocsRoot;
            if (!Directory.Exists(docsRoot))
            {
                return false;
            }
            foreach (var plan in AssetKeyPlanner.Build(docsRoot))
            {
                if (string.IsNullOrEmpty(plan.Group))
                {
                    Debug.LogError("Addressables: unroutable key " + plan.Key);
                    continue;
                }
                var group = settings.FindGroup(plan.Group);
                if (group == null)
                {
                    continue;
                }
                if (plan.AliasTarget.Length > 0)
                {
                    PlaceholderFileFactory.EnsureAliasAsset(plan.PlaceholderPath, plan.AliasTarget);
                }
                else if (plan.Facet == AssetFacet.Icon)
                {
                    PlaceholderFileFactory.EnsureIcon(plan.PlaceholderPath);
                }
                else
                {
                    PlaceholderFileFactory.EnsurePlaceholderAsset(plan.PlaceholderPath, plan.CatalogId, plan.Facet, plan.AssetClass, plan.SizeProfile);
                }
                var guid = AssetDatabase.AssetPathToGUID(plan.PlaceholderPath);
                if (string.IsNullOrEmpty(guid))
                {
                    continue;
                }
                var entry = settings.FindAssetEntry(guid);
                if (entry == null)
                {
                    entry = settings.CreateOrMoveEntry(guid, group, false, false);
                    dirty = true;
                }
                if (entry.parentGroup != group)
                {
                    settings.MoveEntry(entry, group);
                    dirty = true;
                }
                if (entry.address != plan.Key)
                {
                    entry.SetAddress(plan.Key, false);
                    dirty = true;
                }
            }
            return dirty;
        }
    }
}
