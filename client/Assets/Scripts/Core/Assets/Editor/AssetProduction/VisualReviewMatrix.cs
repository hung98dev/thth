using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // The fixed review matrix of section 3.3: three reference resolutions,
    // day and night lighting, 100% and 200% zoom, over the review scenes in
    // Assets/Scenes/Review/.
    public static class VisualReviewMatrix
    {
        public static readonly (int Width, int Height)[] Resolutions =
        {
            (1280, 720),
            (1920, 1080),
            (2400, 1080),
        };

        public static readonly string[] Lightings = { "day", "night" };
        public static readonly int[] ZoomPercents = { 100, 200 };

        public static readonly string[] ScenePaths =
        {
            "Assets/Scenes/Review/ReviewEnvironment.unity",
            "Assets/Scenes/Review/ReviewActor.unity",
            "Assets/Scenes/Review/ReviewUI.unity",
        };

        // Environment-scene layer root object names (L0 foreground through
        // L4 far). The environment rule renders each layer isolated.
        public static readonly string[] EnvironmentLayers =
        {
            "layer_l0", "layer_l1", "layer_l2", "layer_l3", "layer_l4",
        };

        public static List<VisualReviewCombo> Combos()
        {
            var combos = new List<VisualReviewCombo>();
            foreach (var res in Resolutions)
            {
                foreach (var lighting in Lightings)
                {
                    foreach (var zoom in ZoomPercents)
                    {
                        combos.Add(new VisualReviewCombo
                        {
                            Width = res.Width,
                            Height = res.Height,
                            Lighting = lighting,
                            ZoomPercent = zoom,
                        });
                    }
                }
            }
            return combos;
        }
    }
}
