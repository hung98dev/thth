namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // One deterministic finding produced by the cutout or volume gate.
    // Rule is the stable rule identifier (e.g. "fringe_hue"); X/Y carry the
    // offending texture pixel when one exists, else -1.
    public sealed class GateViolation
    {
        public readonly string Rule;
        public readonly int X;
        public readonly int Y;
        public readonly string Detail;

        public GateViolation(string rule, int x, int y, string detail)
        {
            Rule = rule;
            X = x;
            Y = y;
            Detail = detail;
        }

        public GateViolation(string rule, string detail)
            : this(rule, -1, -1, detail)
        {
        }
    }
}
