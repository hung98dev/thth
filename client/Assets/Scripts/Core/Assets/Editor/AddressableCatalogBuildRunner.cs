using System;
using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEngine;

namespace ThinhThan.Core.Assets.Editor
{
    // CI entry point for the catalog build gate
    // (client_assets.md § Catalog Identity + Build Pipeline): provisions
    // the catalog, runs BuildPlayerContent, then writes
    // key_resolution_report.json and catalog_identity.json under
    // client/reports/addressables/ (gitignored, uploaded as CI
    // artifacts). Invoked via
    //   unity-editor -executeMethod
    //     ThinhThan.Core.Assets.Editor.AddressableCatalogBuildRunner.BuildCatalog
    public static class AddressableCatalogBuildRunner
    {
        public const string ReportsDir = "reports/addressables";

        public static void BuildCatalog()
        {
            try
            {
                AddressablesProvisioner.Provision();
                var settings = AddressablesProvisioner.LoadSettings();
                if (settings == null)
                {
                    throw new InvalidOperationException("AddressableAssetSettings.asset not found at " + AddressablesProvisioner.SettingsPath);
                }
                AddressablesProvisioner.InjectDefaultSettings(settings);
                var entries = CollectEntryReport(settings);
                WriteKeyResolutionReport(entries);
                AddressableAssetSettings.BuildPlayerContent(out var result);
                if (!string.IsNullOrEmpty(result.Error))
                {
                    throw new InvalidOperationException("Addressables build failed: " + result.Error);
                }
                WriteCatalogIdentity(settings, result);
                EditorApplication.Exit(0);
            }
            catch (Exception e)
            {
                Debug.LogError("AddressableCatalogBuildRunner failed: " + e);
                EditorApplication.Exit(1);
            }
        }

        private static List<string> CollectEntryReport(AddressableAssetSettings settings)
        {
            var lines = new List<string>();
            var keys = new SortedDictionary<string, string>(StringComparer.Ordinal);
            foreach (var group in settings.groups)
            {
                foreach (var entry in group.entries)
                {
                    var assetPath = AssetDatabase.GUIDToAssetPath(entry.guid);
                    keys[entry.address] = group.Name + "|" + assetPath;
                }
            }
            foreach (var kv in keys)
            {
                var parts = kv.Value.Split('|');
                lines.Add("{\"key\":\"" + Json(kv.Key) + "\",\"group\":\"" + Json(parts[0]) + "\",\"asset_path\":\"" + Json(parts[1]) + "\"}");
            }
            return lines;
        }

        private static void WriteKeyResolutionReport(List<string> entryLines)
        {
            var sb = new StringBuilder();
            sb.Append("{\"entries\":[");
            sb.Append(string.Join(",", entryLines));
            sb.Append("]}\n");
            WriteReport("key_resolution_report.json", sb.ToString());
        }

        private static void WriteCatalogIdentity(AddressableAssetSettings settings, AddressablesPlayerBuildResult result)
        {
            var keys = new List<string>();
            foreach (var group in settings.groups)
            {
                foreach (var entry in group.entries)
                {
                    keys.Add(group.Name + ":" + entry.address);
                }
            }
            keys.Sort(StringComparer.Ordinal);
            var contentHash = Sha256Hex(string.Join("\n", keys.ToArray()));

            var registryPaths = new List<string>();
            if (result.FileRegistry != null)
            {
                foreach (var p in result.FileRegistry.GetFilePaths())
                {
                    registryPaths.Add(p.Replace('\\', '/'));
                }
            }
            registryPaths.Sort(StringComparer.Ordinal);
            var buildId = Sha256Hex(string.Join("\n", registryPaths.ToArray()));

            var sb = new StringBuilder();
            sb.Append("{\n");
            sb.Append("  \"client_build\": \"" + Json(ClientBuild()) + "\",\n");
            sb.Append("  \"asset_catalog_revision\": \"" + Json(contentHash.Substring(0, 16)) + "\",\n");
            sb.Append("  \"addressables_build_id\": \"" + Json(buildId.Substring(0, 16)) + "\",\n");
            sb.Append("  \"platform\": \"" + Json(EditorUserBuildSettings.activeBuildTarget.ToString()) + "\",\n");
            sb.Append("  \"content_hash\": \"" + Json(contentHash) + "\",\n");
            sb.Append("  \"compatible_protocol_major\": " + ProtocolMajor() + ",\n");
            sb.Append("  \"compatible_content_schema_range\": \"" + Json(ContentSchemaRange()) + "\"\n");
            sb.Append("}\n");
            WriteReport("catalog_identity.json", sb.ToString());
        }

        private static string ClientBuild()
        {
            var sha = Environment.GetEnvironmentVariable("GITHUB_SHA");
            var version = PlayerSettings.bundleVersion;
            return string.IsNullOrEmpty(sha) ? version : version + "+" + sha;
        }

        private static int ProtocolMajor()
        {
            // proto/thinhthan/v<N> names the wire major.
            var protoDir = Path.GetFullPath(Path.Combine(Directory.GetCurrentDirectory(), "..", "proto", "thinhthan"));
            if (Directory.Exists(protoDir))
            {
                foreach (var dir in Directory.GetDirectories(protoDir))
                {
                    var name = Path.GetFileName(dir);
                    if (name != null && name.StartsWith("v", StringComparison.Ordinal) && int.TryParse(name.Substring(1), out var major))
                    {
                        return major;
                    }
                }
            }
            return 1;
        }

        private static string ContentSchemaRange()
        {
            // gameplay content schema compatibility range; default pins
            // the launch schema (min = max = 1) until a wider range is
            // declared by the owning content spec.
            var env = Environment.GetEnvironmentVariable("THINHTHAN_CONTENT_SCHEMA_RANGE");
            return string.IsNullOrEmpty(env) ? "1-1" : env;
        }

        private static void WriteReport(string name, string json)
        {
            var dir = Path.Combine(Directory.GetCurrentDirectory(), ReportsDir);
            Directory.CreateDirectory(dir);
            File.WriteAllText(Path.Combine(dir, name), json);
            Debug.Log("wrote " + Path.Combine(dir, name));
        }

        private static string Sha256Hex(string content)
        {
            using (var sha = SHA256.Create())
            {
                var bytes = sha.ComputeHash(Encoding.UTF8.GetBytes(content));
                var sb = new StringBuilder(bytes.Length * 2);
                foreach (var b in bytes)
                {
                    sb.Append(b.ToString("x2"));
                }
                return sb.ToString();
            }
        }

        private static string Json(string s)
        {
            return s.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
