using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using UnityEditor;
using UnityEngine;
using UnityEngine.Localization.Tables;

namespace ThinhThan.Core.Localization.Editor
{
    /// <summary>
    /// Editor-side ingestion + validation for the Core string table collection
    /// (client_localization.md, IMP-064). The committed TSV
    /// (<c>Assets/Localization/Tables/Core/Core.tsv</c>, columns key, vi-VN,
    /// en-US) is the authored ingest surface; the table assets under
    /// Assets/Localization/Tables/Core are its imported result. Both are
    /// committed so validation runs without a regeneration pass.
    /// </summary>
    public static class LocTableAuthoring
    {
        const string TsvPath = "Assets/Localization/Tables/Core/Core.tsv";
        const string SharedDataPath = "Assets/Localization/Tables/Core/Core Shared Table Data.asset";
        const string ViTablePath = "Assets/Localization/Tables/Core/Core_vi-VN.asset";
        const string EnTablePath = "Assets/Localization/Tables/Core/Core_en-US.asset";
        const string ReportPath = "loc-verification-report.json";

        struct Row
        {
            public string Key;
            public string Vi;
            public string En;
        }

        static List<Row> ReadTsv()
        {
            var rows = new List<Row>();
            foreach (string ln in File.ReadAllLines(TsvPath))
            {
                if (ln.Length == 0 || ln.StartsWith("#")) continue;
                string[] cells = ln.Split('\t');
                if (cells.Length != 3)
                {
                    throw new InvalidDataException("Core.tsv row needs 3 TSV cells: " + ln);
                }
                rows.Add(new Row { Key = cells[0], Vi = cells[1], En = cells[2] });
            }
            return rows;
        }

        /// <summary>Menu: rebuild both string tables from Core.tsv in place.</summary>
        [MenuItem("ThinhThan/Localization/Rebuild Core Tables From TSV")]
        public static void RebuildFromTsv()
        {
            List<Row> rows = ReadTsv();
            var shared = AssetDatabase.LoadAssetAtPath<SharedTableData>(SharedDataPath);
            var vi = AssetDatabase.LoadAssetAtPath<StringTable>(ViTablePath);
            var en = AssetDatabase.LoadAssetAtPath<StringTable>(EnTablePath);
            if (shared == null || vi == null || en == null)
            {
                throw new FileNotFoundException(
                    "Core table assets missing under Assets/Localization/Tables/Core/");
            }
            var seen = new HashSet<long>();
            var used = new HashSet<string>();
            foreach (Row r in rows)
            {
                if (!used.Add(r.Key))
                {
                    throw new InvalidDataException("Duplicate loc key in Core.tsv: " + r.Key);
                }
                SharedTableData.SharedTableEntry e = shared.GetEntry(r.Key);
                if (e == null) e = shared.AddKey(r.Key);
                seen.Add(e.Id);
                vi.AddEntry(e.Id, r.Vi);
                en.AddEntry(e.Id, r.En);
            }
            for (int i = shared.Entries.Count - 1; i >= 0; i--)
            {
                if (!seen.Contains(shared.Entries[i].Id))
                {
                    long id = shared.Entries[i].Id;
                    shared.RemoveKey(id);
                    vi.RemoveEntry(id);
                    en.RemoveEntry(id);
                }
            }
            EditorUtility.SetDirty(shared);
            EditorUtility.SetDirty(vi);
            EditorUtility.SetDirty(en);
            AssetDatabase.SaveAssets();
        }

        /// <summary>
        /// CI entry: compares committed table contents with the canonical key set
        /// derived from the content catalogs. Writes loc-verification-report.json;
        /// exits nonzero on any missing/extra key or empty translation.
        /// </summary>
        public static void VerifyAll()
        {
            var problems = new List<string>();
            var shared = AssetDatabase.LoadAssetAtPath<SharedTableData>(SharedDataPath);
            var vi = AssetDatabase.LoadAssetAtPath<StringTable>(ViTablePath);
            var en = AssetDatabase.LoadAssetAtPath<StringTable>(EnTablePath);
            var actual = new SortedSet<string>(StringComparer.Ordinal);
            if (shared != null)
            {
                foreach (SharedTableData.SharedTableEntry e in shared.Entries)
                {
                    actual.Add(e.Key);
                }
            }
            SortedSet<string> expected = LocCatalogExtractor.ExpectedKeys();
            foreach (string k in expected)
            {
                if (!actual.Contains(k)) problems.Add("missing-key:" + k);
            }
            foreach (string k in actual)
            {
                if (!expected.Contains(k)) problems.Add("unexpected-key:" + k);
            }
            if (shared != null && vi != null && en != null)
            {
                foreach (SharedTableData.SharedTableEntry e in shared.Entries)
                {
                    StringTableEntry v = vi.GetEntry(e.Id);
                    StringTableEntry n = en.GetEntry(e.Id);
                    if (v == null || string.IsNullOrEmpty(v.Value)) problems.Add("empty-vi:" + e.Key);
                    if (n == null || string.IsNullOrEmpty(n.Value)) problems.Add("empty-en:" + e.Key);
                }
            }
            else
            {
                problems.Add("tables-not-loadable");
            }
            var sb = new StringBuilder();
            sb.Append("{\"expected\":").Append(expected.Count);
            sb.Append(",\"actual\":").Append(actual.Count);
            sb.Append(",\"problems\":[");
            for (int i = 0; i < problems.Count; i++)
            {
                if (i > 0) sb.Append(',');
                sb.Append('"').Append(problems[i]).Append('"');
            }
            sb.Append("]}");
            File.WriteAllText(ReportPath, sb.ToString());
            if (problems.Count > 0)
            {
                throw new Exception("Loc verification failed: " + problems.Count + " problem(s), see " + ReportPath);
            }
            Debug.Log("Loc verification passed: " + expected.Count + " keys, bilingual.");
        }
    }
}
