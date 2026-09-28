namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // One Visual Review capture combination (section 3.3): resolution,
    // lighting (day|night) and zoom percent.
    public struct VisualReviewCombo
    {
        public int Width;
        public int Height;
        public string Lighting;
        public int ZoomPercent;
    }
}
