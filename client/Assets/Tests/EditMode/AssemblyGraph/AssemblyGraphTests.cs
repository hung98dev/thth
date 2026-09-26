using System.Collections.Generic;
using System.IO;
using System.Text.RegularExpressions;
using NUnit.Framework;

namespace ThinhThan.Tests.EditMode.AssemblyGraph
{
    // Verifies the 13 mandatory assemblies of repository_layout.md §
    // Mandatory Assemblies: names, exact reference lists, acyclic graph,
    // Protocol purity (name references only, Google.Protobuf.dll
    // precompiled).
    public class AssemblyGraphTests
    {
        private static readonly string[] MandatoryNames =
        {
            "ThinhThan.Protocol",
            "ThinhThan.Core",
            "ThinhThan.Core.Assets",
            "ThinhThan.Core.Assets.Editor",
            "ThinhThan.Core.Localization",
            "ThinhThan.Core.Localization.Editor",
            "ThinhThan.Core.Geometry.Editor",
            "ThinhThan.Net",
            "ThinhThan.Systems",
            "ThinhThan.UI",
            "ThinhThan.App",
            "ThinhThan.Tests.EditMode",
            "ThinhThan.Tests.PlayMode",
        };

        // Exact References column of the mandatory table (name references
        // only; precompiled DLLs are asserted separately).
        private static readonly Dictionary<string, string[]> ExpectedRefs = new Dictionary<string, string[]>
        {
            ["ThinhThan.Protocol"] = new string[0],
            ["ThinhThan.Core"] = new[]
            {
                "Unity.InputSystem", "Unity.RenderPipelines.Core.Runtime",
                "Unity.RenderPipelines.Universal.Runtime",
            },
            ["ThinhThan.Core.Assets"] = new[]
            {
                "ThinhThan.Core", "Unity.Addressables", "Unity.ResourceManager",
            },
            ["ThinhThan.Core.Assets.Editor"] = new[]
            {
                "ThinhThan.Core", "ThinhThan.Core.Assets", "Unity.Addressables",
                "Unity.Addressables.Editor", "Unity.ResourceManager",
            },
            ["ThinhThan.Core.Localization"] = new[]
            {
                "ThinhThan.Core", "Unity.Localization", "Unity.Addressables",
                "Unity.ResourceManager",
            },
            ["ThinhThan.Core.Localization.Editor"] = new[]
            {
                "ThinhThan.Core", "ThinhThan.Core.Localization", "Unity.Localization",
                "Unity.Localization.Editor",
            },
            ["ThinhThan.Core.Geometry.Editor"] = new[] { "ThinhThan.Core" },
            ["ThinhThan.Net"] = new[] { "ThinhThan.Core", "ThinhThan.Protocol" },
            ["ThinhThan.Systems"] = new[]
            {
                "ThinhThan.Core", "ThinhThan.Core.Assets", "ThinhThan.Core.Localization",
                "ThinhThan.Net", "ThinhThan.Protocol", "Unity.InputSystem",
                "Unity.RenderPipelines.Core.Runtime", "Unity.RenderPipelines.Universal.Runtime",
                "Unity.2D.Animation.Runtime",
            },
            ["ThinhThan.UI"] = new[]
            {
                "ThinhThan.Core", "ThinhThan.Core.Assets", "ThinhThan.Core.Localization",
                "ThinhThan.Net", "ThinhThan.Protocol", "ThinhThan.Systems",
                "Unity.InputSystem", "UnityEngine.UI", "Unity.TextMeshPro",
            },
            ["ThinhThan.App"] = new[]
            {
                "ThinhThan.Core", "ThinhThan.Core.Assets", "ThinhThan.Core.Localization",
                "ThinhThan.Net", "ThinhThan.Protocol", "ThinhThan.Systems", "ThinhThan.UI",
                "Unity.InputSystem", "Unity.RenderPipelines.Universal.Runtime",
                "Unity.Addressables", "Unity.ResourceManager", "Unity.Localization",
            },
        };

        private static string[] FindAsmdefs()
        {
            return Directory.GetFiles("Assets", "*.asmdef", SearchOption.AllDirectories);
        }

        private static string JsonString(string json, string key)
        {
            var m = Regex.Match(json, "\"" + key + "\"\\s*:\\s*\"([^\"]+)\"");
            return m.Success ? m.Groups[1].Value : "";
        }

        private static string[] JsonArray(string json, string key)
        {
            var m = Regex.Match(json, "\"" + key + "\"\\s*:\\s*\\[([^\\]]*)\\]");
            var list = new List<string>();
            if (!m.Success)
            {
                return list.ToArray();
            }
            foreach (Match s in Regex.Matches(m.Groups[1].Value, "\"([^\"]*)\""))
            {
                list.Add(s.Groups[1].Value);
            }
            return list.ToArray();
        }

        private static bool JsonBool(string json, string key)
        {
            var m = Regex.Match(json, "\"" + key + "\"\\s*:\\s*(true|false)");
            return m.Success && m.Groups[1].Value == "true";
        }

        private static Dictionary<string, string[]> LoadGraph()
        {
            var graph = new Dictionary<string, string[]>();
            foreach (var path in FindAsmdefs())
            {
                var json = File.ReadAllText(path);
                var name = JsonString(json, "name");
                Assert.AreNotEqual("", name, path + ": asmdef name missing");
                Assert.IsFalse(JsonBool(json, "autoReferenced"), name + ": autoReferenced must be false");
                graph[name] = JsonArray(json, "references");
            }
            return graph;
        }

        [Test]
        public void TestMandatoryAssembliesDeclared()
        {
            var graph = LoadGraph();
            Assert.AreEqual(MandatoryNames.Length, graph.Count, "assembly count != 13");
            foreach (var name in MandatoryNames)
            {
                Assert.IsTrue(graph.ContainsKey(name), "mandatory assembly missing: " + name);
            }
        }

        [Test]
        public void TestReferenceGraphMatchesLayout()
        {
            var graph = LoadGraph();
            foreach (var kv in ExpectedRefs)
            {
                Assert.IsTrue(graph.ContainsKey(kv.Key), "asmdef missing: " + kv.Key);
                var got = new List<string>(graph[kv.Key]);
                var want = new List<string>(kv.Value);
                got.Sort();
                want.Sort();
                Assert.AreEqual(string.Join(",", want.ToArray()), string.Join(",", got.ToArray()),
                    kv.Key + " references differ from the layout table");
            }
        }

        [Test]
        public void TestAsmdefReferencesAcyclic()
        {
            var graph = LoadGraph();
            var project = new Dictionary<string, List<string>>();
            foreach (var kv in graph)
            {
                var refs = new List<string>();
                foreach (var r in kv.Value)
                {
                    if (r.StartsWith("ThinhThan."))
                    {
                        refs.Add(r);
                    }
                }
                project[kv.Key] = refs;
            }
            var visiting = new HashSet<string>();
            var done = new HashSet<string>();
            var stack = new List<string>();
            foreach (var name in project.Keys)
            {
                Assert.IsFalse(Cycle(name, project, visiting, done, stack),
                    "assembly reference cycle at " + name + ": " + string.Join(" -> ", stack.ToArray()));
            }
        }

        private static bool Cycle(string node, Dictionary<string, List<string>> graph,
            HashSet<string> visiting, HashSet<string> done, List<string> stack)
        {
            if (done.Contains(node))
            {
                return false;
            }
            if (visiting.Contains(node))
            {
                stack.Add(node);
                return true;
            }
            visiting.Add(node);
            stack.Add(node);
            if (graph.ContainsKey(node))
            {
                foreach (var next in graph[node])
                {
                    if (Cycle(next, graph, visiting, done, stack))
                    {
                        return true;
                    }
                }
            }
            stack.RemoveAt(stack.Count - 1);
            visiting.Remove(node);
            done.Add(node);
            return false;
        }

        [Test]
        public void TestProtocolReferencesNoProjectAssembly()
        {
            var graph = LoadGraph();
            Assert.IsTrue(graph.ContainsKey("ThinhThan.Protocol"), "ThinhThan.Protocol asmdef missing");
            foreach (var r in graph["ThinhThan.Protocol"])
            {
                Assert.IsFalse(r.StartsWith("ThinhThan."),
                    "Protocol must reference no project assembly, found " + r);
            }
            foreach (var path in FindAsmdefs())
            {
                var json = File.ReadAllText(path);
                var name = JsonString(json, "name");
                var pre = JsonArray(json, "precompiledReferences");
                foreach (var p in pre)
                {
                    if (name == "ThinhThan.Protocol")
                    {
                        Assert.AreEqual("Google.Protobuf.dll", p,
                            "Protocol may reference Google.Protobuf.dll only");
                    }
                }
            }
        }
    }
}
