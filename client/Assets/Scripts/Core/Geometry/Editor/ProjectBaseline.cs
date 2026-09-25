using UnityEditor;
using UnityEngine;
using UnityEngine.Rendering;

namespace ThinhThan.Core.Geometry.Editor
{
    // Applies the ProjectSettings baseline of repository_layout.md §
    // ProjectSettings Baseline at every editor load so the materialized
    // assets committed by IMP-000 already carry the entries later packets
    // need (ADR-0068). Idempotent: only fills slots whose target asset
    // exists; missing targets are left for their owning packet.
    [InitializeOnLoad]
    public static class ProjectBaseline
    {
        private const string AddressableKey = "com.unity.addressableassets";
        private const string LocalizationKey = "com.unity.localization.settings";
        private const string AddressablePath = "client/Assets/AddressableAssetsData/AddressableAssetSettings.asset";
        private const string LocalizationPath = "client/Assets/Localization/Settings/LocalizationSettings.asset";
        private const string UrpPath = "client/Assets/Settings/Rendering/ThinhThanURP.asset";
        private const string ServerGeometryTag = "ServerGeometry";

        static ProjectBaseline()
        {
            EditorApplication.delayCall += Apply;
        }

        private static void Apply()
        {
            var changed = false;
            changed |= ApplyPlayer();
            changed |= ApplyPhysics2D();
            changed |= ApplyTag();
            changed |= ApplyConfigObjects();
            changed |= ApplyRenderPipeline();
            if (changed)
            {
                AssetDatabase.SaveAssets();
            }
        }

        private static bool ApplyPlayer()
        {
            var changed = false;
            if (!PlayerSettings.gcIncremental)
            {
                PlayerSettings.gcIncremental = true;
                changed = true;
            }
            if (!PlayerSettings.Android.optimizedFramePacing)
            {
                PlayerSettings.Android.optimizedFramePacing = true;
                changed = true;
            }
            return changed;
        }

        private static bool ApplyPhysics2D()
        {
            if (Physics2DSettings.simulationMode == SimulationMode2D.Script)
            {
                return false;
            }
            Physics2DSettings.simulationMode = SimulationMode2D.Script;
            return true;
        }

        private static bool ApplyTag()
        {
            var assets = AssetDatabase.LoadAllAssetsAtPath("ProjectSettings/TagManager.asset");
            if (assets == null || assets.Length == 0)
            {
                return false;
            }
            var so = new SerializedObject(assets[0]);
            var tags = so.FindProperty("tags");
            if (tags == null || !tags.isArray)
            {
                return false;
            }
            for (var i = 0; i < tags.arraySize; i++)
            {
                if (tags.GetArrayElementAtIndex(i).stringValue == ServerGeometryTag)
                {
                    return false;
                }
            }
            tags.InsertArrayElementAtIndex(tags.arraySize);
            tags.GetArrayElementAtIndex(tags.arraySize - 1).stringValue = ServerGeometryTag;
            so.ApplyModifiedPropertiesWithoutUndo();
            return true;
        }

        private static bool ApplyConfigObjects()
        {
            var changed = false;
            changed |= TryConfigObject(AddressableKey, AddressablePath);
            changed |= TryConfigObject(LocalizationKey, LocalizationPath);
            return changed;
        }

        private static bool TryConfigObject(string key, string path)
        {
            var target = AssetDatabase.LoadMainAssetAtPath(path);
            if (target == null)
            {
                return false;
            }
            var assets = AssetDatabase.LoadAllAssetsAtPath("ProjectSettings/EditorBuildSettings.asset");
            if (assets == null || assets.Length == 0)
            {
                return false;
            }
            var so = new SerializedObject(assets[0]);
            var objects = so.FindProperty("m_configObjects");
            if (objects == null)
            {
                return false;
            }
            var value = objects.FindPropertyRelative(key);
            if (value != null)
            {
                if (value.objectReferenceValue == target)
                {
                    return false;
                }
                value.objectReferenceValue = target;
            }
            else
            {
                objects.InsertArrayElementAtIndex(objects.arraySize);
                var element = objects.GetArrayElementAtIndex(objects.arraySize - 1);
                element.FindPropertyRelative("first").stringValue = key;
                element.FindPropertyRelative("second").objectReferenceValue = target;
            }
            so.ApplyModifiedPropertiesWithoutUndo();
            return true;
        }

        private static bool ApplyRenderPipeline()
        {
            var urp = AssetDatabase.LoadAssetAtPath<RenderPipelineAsset>(UrpPath);
            if (urp == null)
            {
                return false;
            }
            if (GraphicsSettings.defaultRenderPipeline == urp)
            {
                return false;
            }
            GraphicsSettings.defaultRenderPipeline = urp;
            return true;
        }
    }
}
