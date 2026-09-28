using System;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // Provenance of one third-party input used to create or composite an
    // asset (section 6 inputs[]). Every such input needs the full license
    // evidence chain just like a first-class row.
    [Serializable]
    public sealed class InputProvenance
    {
        public string? creator;
        public string? source_uri;
        public string? license_id;
        public string? license_uri;
        public string? acquired_at_utc;
        public string? sha256;
    }
}
