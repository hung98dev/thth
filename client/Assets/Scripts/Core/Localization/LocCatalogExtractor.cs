using System;
using System.Collections.Generic;
using System.IO;
using System.Text.RegularExpressions;
using UnityEngine;

namespace ThinhThan.Core.Localization
{
    /// <summary>
    /// Derives the canonical localization key set from the content catalogs —
    /// the "expected" side of the missing-translation gate
    /// (client_localization.md: every launch content id must resolve in both
    /// launch locales). Key shape: <c>loc.&lt;domain&gt;.&lt;id tail&gt;.name</c>,
    /// <c>.title</c> for quest rows, <c>.desc</c> for lore-bearing atlas pages,
    /// and <c>loc.error.&lt;snake&gt;</c> for wire ErrorCode members.
    /// Editor/test code only — it parses the canonical markdown, never the
    /// generated tables.
    /// </summary>
    public static class LocCatalogExtractor
    {
        // Domains whose `## `<id>` heading` sections carry display strings.
        static readonly HashSet<string> SectionDomains = new HashSet<string>
        {
            "item", "soul", "cosmetic", "set", "formation", "resonance", "quest",
        };

        // Ids that share a display-entity prefix but name no player-facing
        // entity; `cosmetic.shared` is the Addressables group name reused by
        // presentation_asset_manifest.md, not a cosmetic.
        static readonly HashSet<string> ExcludedIds = new HashSet<string>
        {
            "cosmetic.shared",
        };

        static readonly Regex TableIdRow = new Regex(
            "^\\|\\s*`(PREFIX[a-z0-9_.]+)`\\s*\\|", RegexOptions.Compiled);
        static readonly Regex ClassRow = new Regex(
            "^\\|\\s*`(class\\.[a-z0-9_]+)`\\s*\\|\\s*([A-Z]+)\\s*\\|\\s*([^|]+?)\\s*\\|",
            RegexOptions.Compiled);
        static readonly Regex ZoneRow = new Regex(
            "^\\|\\s*[IVX]+\\s*\\|[^|]*\\|\\s*`(zone\\.[a-z0-9_]+)`\\s*\\|\\s*([^|]+?)\\s*\\|",
            RegexOptions.Compiled);
        static readonly Regex EncounterHead = new Regex(
            "^#{1,2}\\s*(Public\\s+Boss|Final\\s+Boss|Boss|Dungeon)\\s*—\\s*(.+?)\\s*$",
            RegexOptions.Compiled);
        static readonly Regex AnyRosterId = new Regex(
            "`((?:boss|dungeon|instance)\\.[a-z0-9_.]+)`", RegexOptions.Compiled);
        static readonly Regex SkillRow = new Regex(
            "^\\|\\s*`(skill\\.[a-z0-9_.]+)`\\s*\\|\\s*([^|`]+?)\\s*\\|\\s*Lv",
            RegexOptions.Compiled);
        static readonly Regex BeastRow = new Regex(
            "^\\|\\s*`(beast\\.[a-z0-9_]+\\.[a-z0-9_]+)`\\s*\\|\\s*([^|]+?)\\s*\\|\\s*([^|]+?)\\s*\\|",
            RegexOptions.Compiled);
        static readonly Regex BeastSkill = new Regex(
            "Passive\\s*\\d\\s*—\\s*(.+?)\\s*\\(`(beast\\.skill\\.[a-z0-9_.]+)`[,)]",
            RegexOptions.Compiled);
        static readonly Regex NpcName = new Regex(
            "`(npc\\.[a-z0-9_]+\\.[a-z0-9_]+)`\\s*—\\s*([^|]+?)\\s*(?:\\||$)",
            RegexOptions.Compiled);
        static readonly Regex NpcId = new Regex(
            "`(npc\\.[a-z0-9_]+\\.[a-z0-9_]+)`", RegexOptions.Compiled);
        static readonly Regex QuestHead = new Regex(
            "^#{2,4}\\s*`(quest\\.[a-z0-9_.]+)`\\s*—\\s*(.+?)\\s*$", RegexOptions.Compiled);
        static readonly Regex QuestRow = new Regex(
            "^\\|\\s*`(quest\\.[a-z0-9_.]+)`\\s*\\|\\s*([^|`]+?)\\s*\\|", RegexOptions.Compiled);
        static readonly Regex SectionNamed = new Regex(
            "^#{2,4}\\s*`([a-z][a-z0-9_]*(?:\\.[a-z0-9_]+)+)`\\s*—\\s*(.+?)\\s*$",
            RegexOptions.Compiled);
        static readonly Regex SectionCaps = new Regex(
            "^#{2,4}\\s*[A-Z_0-9][A-Z_0-9 ]*—\\s*`([a-z][a-z0-9_]*(?:\\.[a-z0-9_]+)+)`",
            RegexOptions.Compiled);
        static readonly Regex SectionBare = new Regex(
            "^#{2,4}\\s*`([a-z][a-z0-9_]*(?:\\.[a-z0-9_]+)+)`",
            RegexOptions.Compiled);
        static readonly Regex ItemCosmeticRow = new Regex(
            "^\\|\\s*`((?:item|cosmetic)\\.[a-z0-9_.]+)`\\s*\\|\\s*([^|`]+?)\\s*\\|",
            RegexOptions.Compiled);
        static readonly Regex AtlasPageRow = new Regex(
            "^\\|\\s*`(atlas\\.page\\.[a-z0-9_.]+)`\\s*\\|(.*)$", RegexOptions.Compiled);
        static readonly Regex SubjectCell = new Regex(
            "`((?:monster|soul|boss|item|relic|map|chest)\\.[a-z0-9_.]+)`",
            RegexOptions.Compiled);
        static readonly Regex CosmeticCell = new Regex(
            "`(cosmetic\\.[a-z0-9_.]+)`", RegexOptions.Compiled);
        static readonly Regex Diacritic = new Regex("[À-ỹĐđ]", RegexOptions.Compiled);
        static readonly Regex SeasonPage = new Regex(
            "atlas\\.page\\.season\\.(\\d+)\\.[a-z0-9_]+\\.([a-z0-9_]+)", RegexOptions.Compiled);
        static readonly Regex SweepId = new Regex(
            "`((?:cosmetic|item|chest)\\.[a-z0-9_]+(?:\\.[a-z0-9_]+)+)`",
            RegexOptions.Compiled);
        static readonly Regex ErrorEnum = new Regex(
            "^\\s*(ERROR_CODE_[A-Z_]+)\\s*=\\s*\\d+", RegexOptions.Compiled);

        /// <summary>Repo root is the ancestor of the Unity project that holds docs/.</summary>
        public static string RepoRoot()
        {
            var dir = new DirectoryInfo(Application.dataPath); // <repo>/client/Assets
            while (dir != null && !Directory.Exists(Path.Combine(dir.FullName, "docs", "07_content")))
            {
                dir = dir.Parent;
            }
            if (dir == null)
            {
                throw new DirectoryNotFoundException(
                    "docs/07_content not found above " + Application.dataPath);
            }
            return dir.FullName;
        }

        /// <summary>
        /// Collects roster ids keyed by domain. Domains: every catalog entity class
        /// that carries a player-facing display string. Atlas pages are collected
        /// separately via <paramref name="lorePages"/> (they emit .desc keys only
        /// when the row carries authored lore text).
        /// </summary>
        public static Dictionary<string, SortedSet<string>> CollectRoster(
            out SortedSet<string> lorePages)
        {
            string root = RepoRoot();
            string contentDir = Path.Combine(root, "docs", "07_content");
            string classesPath = Path.Combine(root, "docs", "01_gameplay", "classes.md");
            string protoPath = Path.Combine(root, "proto", "thinhthan", "v1", "common.proto");

            var ids = new Dictionary<string, SortedSet<string>>();
            var lore = new SortedSet<string>();

            void Add(string domain, string id)
            {
                if (ExcludedIds.Contains(id))
                {
                    return;
                }
                if (!ids.TryGetValue(domain, out SortedSet<string> set))
                {
                    set = new SortedSet<string>(StringComparer.Ordinal);
                    ids[domain] = set;
                }
                set.Add(id);
            }

            void AddTableIds(string file, string domain, string prefix)
            {
                Regex re = new Regex(
                    TableIdRow.ToString().Replace("PREFIX", prefix), RegexOptions.Compiled);
                foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, file)))
                {
                    Match m = re.Match(ln);
                    if (m.Success) Add(domain, m.Groups[1].Value);
                }
            }

            AddTableIds("monster_catalog.md", "monster", "monster\\.");
            AddTableIds("boss_catalog.md", "boss", "boss\\.");
            AddTableIds("world_route_catalog.md", "map", "map\\.");
            AddTableIds("dungeon_catalog.md", "dungeon", "dungeon\\.");
            Add("instance", "instance.finale.than_trung");

            foreach (string ln in File.ReadAllLines(classesPath))
            {
                Match m = ClassRow.Match(ln);
                if (m.Success) Add("class", m.Groups[1].Value);
            }

            bool encNameSeen = false;
            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "encounter_catalog.md")))
            {
                Match zm = ZoneRow.Match(ln);
                if (zm.Success) Add("zone", zm.Groups[1].Value);
                Match hm = EncounterHead.Match(ln);
                if (hm.Success) { encNameSeen = true; continue; }
                if (encNameSeen)
                {
                    Match im = AnyRosterId.Match(ln);
                    if (im.Success)
                    {
                        Add(im.Groups[1].Value.Split('.')[0], im.Groups[1].Value);
                        encNameSeen = false;
                    }
                }
            }
            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "class_skill_catalog.md")))
            {
                Match m = SkillRow.Match(ln);
                if (m.Success) Add("skill", m.Groups[1].Value);
            }

            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "spirit_beast_catalog.md")))
            {
                Match m = BeastRow.Match(ln);
                if (m.Success) Add("beast", m.Groups[1].Value);
                foreach (Match sm in BeastSkill.Matches(ln))
                {
                    Add("beast_skill", sm.Groups[2].Value);
                }
            }

            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "npc_shop_catalog.md")))
            {
                foreach (Match nm in NpcName.Matches(ln)) Add("npc", nm.Groups[1].Value);
                foreach (Match im in NpcId.Matches(ln)) Add("npc", im.Groups[1].Value);
            }

            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "quest_catalog.md")))
            {
                Match hm = QuestHead.Match(ln);
                if (hm.Success) Add("quest", hm.Groups[1].Value);
                Match rm = QuestRow.Match(ln);
                if (rm.Success && !rm.Groups[1].Value.Contains("<")) Add("quest", rm.Groups[1].Value);
            }
            Add("quest", "quest.event.spirit_surge.contribute");

            foreach (string file in Directory.GetFiles(contentDir, "*.md"))
            {
                foreach (string ln in File.ReadAllLines(file))
                {
                    Match m = SectionNamed.Match(ln);
                    if (m.Success)
                    {
                        string sect = m.Groups[1].Value;
                        string d = sect.Split('.')[0];
                        if (SectionDomains.Contains(d) || d == "boss" || d == "dungeon") Add(d, sect);
                        continue;
                    }
                    m = SectionCaps.Match(ln);
                    if (m.Success)
                    {
                        string sect = m.Groups[1].Value;
                        if (SectionDomains.Contains(sect.Split('.')[0])) Add(sect.Split('.')[0], sect);
                        continue;
                    }
                    m = SectionBare.Match(ln);
                    if (m.Success)
                    {
                        string sect = m.Groups[1].Value;
                        if (SectionDomains.Contains(sect.Split('.')[0])) Add(sect.Split('.')[0], sect);
                        continue;
                    }
                    Match rm = ItemCosmeticRow.Match(ln);
                    if (rm.Success && !rm.Groups[1].Value.Contains("<"))
                    {
                        Add(rm.Groups[1].Value.Split('.')[0], rm.Groups[1].Value);
                    }
                }
            }

            var atlasPages = new SortedSet<string>(StringComparer.Ordinal);
            foreach (string ln in File.ReadAllLines(Path.Combine(contentDir, "atlas_catalog.md")))
            {
                Match m = AtlasPageRow.Match(ln);
                if (!m.Success) continue;
                string pid = m.Groups[1].Value;
                atlasPages.Add(pid);
                var cells = new List<string>();
                foreach (string c in m.Groups[2].Value.Split('|'))
                {
                    string t = c.Trim();
                    if (t.Length > 0) cells.Add(t);
                }
                if (cells.Count > 0)
                {
                    Match sm = SubjectCell.Match(cells[0]);
                    if (sm.Success) Add(sm.Groups[1].Value.Split('.')[0], sm.Groups[1].Value);
                }
                for (int i = 1; i < cells.Count; i++)
                {
                    Match cm = CosmeticCell.Match(cells[i]);
                    if (cm.Success)
                    {
                        Add("cosmetic", cm.Groups[1].Value);
                        continue;
                    }
                    if (cells[i] != "—" && !cells[i].Contains("`") && Diacritic.IsMatch(cells[i]))
                    {
                        lore.Add(pid);
                    }
                }
            }

            // union sweep: every remaining concrete cosmetic/item/chest id
            foreach (string file in Directory.GetFiles(contentDir, "*.md"))
            {
                string text = File.ReadAllText(file);
                foreach (Match m in SweepId.Matches(text))
                {
                    string id = m.Groups[1].Value;
                    Add(id.Split('.')[0], id);
                }
            }

            // season title synthesis: atlas.page.season.<n>.<r>.<key> -> cosmetic.title.season.<n>.<key>
            foreach (string pid in atlasPages)
            {
                Match mm = SeasonPage.Match(pid);
                if (mm.Success)
                {
                    Add("cosmetic",
                        "cosmetic.title.season." + mm.Groups[1].Value + "." + mm.Groups[2].Value);
                }
            }

            foreach (string ln in File.ReadAllLines(protoPath))
            {
                Match m = ErrorEnum.Match(ln);
                if (m.Success && m.Groups[1].Value != "ERROR_CODE_UNSPECIFIED")
                {
                    Add("error", m.Groups[1].Value);
                }
            }

            lorePages = lore;
            return ids;
        }

        /// <summary>Entity id -> canonical loc key stem.</summary>
        public static string IdToKey(string id)
        {
            if (id.StartsWith("ERROR_CODE_", StringComparison.Ordinal))
            {
                return "loc.error." + id.Substring("ERROR_CODE_".Length).ToLowerInvariant();
            }
            int dot = id.IndexOf('.');
            string dom = id.Substring(0, dot);
            string tail = id.Substring(dot + 1);
            string suffix = dom == "atlas" ? "desc" : dom == "quest" ? "title" : "name";
            return "loc." + dom + "." + tail + "." + suffix;
        }

        /// <summary>The full expected key set for both launch locales.</summary>
        public static SortedSet<string> ExpectedKeys()
        {
            SortedSet<string> lorePages;
            Dictionary<string, SortedSet<string>> ids = CollectRoster(out lorePages);
            var keys = new SortedSet<string>(StringComparer.Ordinal);
            foreach (KeyValuePair<string, SortedSet<string>> kv in ids)
            {
                foreach (string id in kv.Value) keys.Add(IdToKey(id));
            }
            // lore-bearing atlas pages are tracked separately: only they emit .desc keys
            foreach (string pid in lorePages) keys.Add(IdToKey(pid));
            return keys;
        }
    }
}
