using System.IO;
using System.Security.Cryptography;
using System.Text;
using NUnit.Framework;
using ThinhThan.Core.Rendering;

namespace ThinhThan.Tests.EditMode.RenderingSetup
{
    // Asserts the IMP-101 rendering setup (ADR-0056, ADR-0071): the URP
    // pipeline asset at the path-derived baseline GUID
    // (repository_layout.md § ProjectSettings Baseline), the 2D renderer's
    // custom transparency sort axis, SRP Batcher, Sprite-Lit gameplay
    // materials, day/night interpolation, per-preset point Light2D budgets,
    // a contact shadow per actor profile and UI 200 PPU imports.
    public class RenderingSetupTests
    {
        private const string UrpAssetPath = "client/Assets/Settings/Rendering/ThinhThanURP.asset";
        private const string Renderer2DPath = "client/Assets/Settings/Rendering/ThinhThan2DRenderer.asset";
        private const string SpriteLitPath = "client/Assets/Settings/Rendering/ThinhThanSpriteLit.mat";
        private const string ContactShadowMaterialPath = "client/Assets/Settings/Rendering/ThinhThanContactShadow.mat";
        private const string ContactShadowShaderPath = "client/Assets/Settings/Rendering/ContactShadow.shader";
        private const string UrpAssetScriptGuid = "bf2edee5c58d82540a51f03df9d42094"; // UniversalRenderPipelineAsset
        private const string Renderer2DScriptGuid = "11145981673336645838492a2d98e247"; // Renderer2DData
        private const string SpriteLitShaderGuid = "e260cfa7296ee7642b167f1eb5be5023"; // URP Sprite-Lit-Default.shader

        private static string PathGuid(string repoRelativePath)
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

        private static string ReadAsset(string repoRelativePath)
        {
            var path = repoRelativePath.Substring("client/".Length);
            Assert.IsTrue(File.Exists(path), repoRelativePath + " missing");
            return File.ReadAllText(path);
        }

        private static string MetaGuid(string repoRelativePath)
        {
            var meta = ReadAsset(repoRelativePath + ".meta");
            var idx = meta.IndexOf("guid: ", System.StringComparison.Ordinal);
            Assert.GreaterOrEqual(idx, 0, repoRelativePath + ".meta has no guid");
            return meta.Substring(idx + "guid: ".Length, 32);
        }

        [Test]
        public void TestRendererAsset()
        {
            var urp = ReadAsset(UrpAssetPath);
            Assert.IsTrue(urp.Contains("guid: " + UrpAssetScriptGuid),
                "ThinhThanURP must be a UniversalRenderPipelineAsset");
            Assert.IsTrue(urp.Contains("m_RendererType: 2"),
                "pipeline RendererType must be _2DRenderer (2)");
            var rendererGuid = MetaGuid(Renderer2DPath);
            Assert.IsTrue(urp.Contains("guid: " + rendererGuid),
                "m_RendererDataList must reference the 2D renderer asset");
            var renderer = ReadAsset(Renderer2DPath);
            Assert.IsTrue(renderer.Contains("guid: " + Renderer2DScriptGuid),
                "ThinhThan2DRenderer must be a Renderer2DData asset");
        }

        [Test]
        public void TestSpriteLitMaterials()
        {
            var material = ReadAsset(SpriteLitPath);
            Assert.IsTrue(material.Contains("guid: " + SpriteLitShaderGuid),
                "gameplay sprite material must use the URP Sprite-Lit-Default shader");
            var renderer = ReadAsset(Renderer2DPath);
            Assert.IsTrue(renderer.Contains("m_DefaultMaterialType: 0"),
                "2D renderer default material type must be Lit (Sprite-Lit)");
        }

        [Test]
        public void TestDayNightInterpolation()
        {
            var half = DayNightCycle.TransitionMinutes * 0.5f;
            Assert.AreEqual(0f, DayNightCycle.NightFactor(40f), 0.0001f);
            Assert.AreEqual(0f, DayNightCycle.NightFactor(DayNightCycle.DayMinutes - half), 0.0001f);
            Assert.AreEqual(0.5f, DayNightCycle.NightFactor(DayNightCycle.DayMinutes), 0.0001f);
            Assert.AreEqual(1f, DayNightCycle.NightFactor(100f), 0.0001f);
            Assert.AreEqual(0.5f, DayNightCycle.NightFactor(0f), 0.0001f);
            Assert.AreEqual(0.5f, DayNightCycle.NightFactor(DayNightCycle.WorldDayMinutes), 0.0001f);
            var beforeDusk = DayNightCycle.NightFactor(78f);
            var atBoundary = DayNightCycle.NightFactor(80f);
            var afterDusk = DayNightCycle.NightFactor(82f);
            Assert.IsTrue(beforeDusk < atBoundary && atBoundary < afterDusk,
                "night factor must ramp up across the dusk transition");
            Assert.AreEqual(DayNightCycle.NightFactor(1f), DayNightCycle.NightFactor(121f), 0.0001f,
                "world time wraps at the world day boundary");
        }

        [Test]
        public void TestPresetLightBudget()
        {
            Assert.AreEqual(4, PointLight2DBudget.MaxActivePointLights(QualityPreset.Low));
            Assert.AreEqual(8, PointLight2DBudget.MaxActivePointLights(QualityPreset.Medium));
            Assert.AreEqual(16, PointLight2DBudget.MaxActivePointLights(QualityPreset.High));
        }

        [Test]
        public void TestContactShadowPerActorProfile()
        {
            var material = ReadAsset(ContactShadowMaterialPath);
            Assert.IsTrue(material.Contains("_BaseColor"),
                "contact shadow material must bind the procedural shadow shader");
            var shader = ReadAsset(ContactShadowShaderPath);
            Assert.IsTrue(shader.Contains("UnityPerMaterial"),
                "contact shadow shader must keep per-material state in UnityPerMaterial (SRP Batcher)");
            foreach (ActorSizeProfile profile in System.Enum.GetValues(typeof(ActorSizeProfile)))
            {
                var spec = ContactShadowSpec.For(profile);
                Assert.Greater(spec.WidthMeters, 0f, profile + " shadow width");
                Assert.Greater(spec.HeightMeters, 0f, profile + " shadow height");
                Assert.IsTrue(spec.Opacity > 0f && spec.Opacity <= 1f,
                    profile + " shadow opacity must be in (0,1]");
            }
        }

        [Test]
        public void TestUiImport200Ppu()
        {
            Assert.AreEqual(200, SpriteImportRules.UiPixelsPerUnit);
            Assert.AreEqual(100, SpriteImportRules.GameplayPixelsPerUnit);
            Assert.AreEqual(100, SpriteImportRules.CanvasReferencePixelsPerUnit);
            Assert.AreEqual(50, SpriteImportRules.ParallaxFarPixelsPerUnit);
            var uiDir = Path.Combine("Assets", "Art", "UI");
            if (Directory.Exists(uiDir))
            {
                foreach (var metaPath in Directory.GetFiles(uiDir, "*.png.meta", SearchOption.AllDirectories))
                {
                    var meta = File.ReadAllText(metaPath);
                    Assert.IsTrue(meta.Contains("spritePixelsToUnits: 200"),
                        metaPath + " must import at 200 PPU");
                }
            }
        }

        [Test]
        public void TestSortAxisAndSrpBatcher()
        {
            var renderer = ReadAsset(Renderer2DPath);
            Assert.IsTrue(renderer.Contains("m_TransparencySortMode: 3"),
                "2D renderer must use CustomAxis transparency sort mode (3)");
            Assert.IsTrue(renderer.Contains("m_TransparencySortAxis: {x: 0, y: 1, z: 0}"),
                "transparency sort axis must be (0,1,0)");
            var urp = ReadAsset(UrpAssetPath);
            Assert.IsTrue(urp.Contains("m_UseSRPBatcher: 1"),
                "SRP Batcher must be enabled on the pipeline asset");
        }

        [Test]
        public void TestPipelineAssetMatchesBaselineGuid()
        {
            var expected = PathGuid(UrpAssetPath);
            Assert.AreEqual(expected, MetaGuid(UrpAssetPath),
                "ThinhThanURP.asset.meta guid must be the path-derived baseline GUID");
            var graphics = File.ReadAllText(Path.Combine("ProjectSettings", "GraphicsSettings.asset"));
            var idx = graphics.IndexOf("m_CustomRenderPipeline", System.StringComparison.Ordinal);
            Assert.GreaterOrEqual(idx, 0, "GraphicsSettings m_CustomRenderPipeline missing");
            var window = graphics.Substring(idx, System.Math.Min(200, graphics.Length - idx));
            Assert.IsTrue(window.Contains(expected),
                "m_CustomRenderPipeline must reference the path-derived GUID of " + UrpAssetPath);
        }
    }
}
