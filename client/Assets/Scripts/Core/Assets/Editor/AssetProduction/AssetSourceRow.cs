using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // One register row = one packaged media file (section 6, extended by
    // ADR-0076 with folklore_card + cultural_entity). Strings are nullable:
    // a field that is semantically required but absent in JSON materializes
    // as null and produces a path-specific validator error rather than a
    // parse failure.
    [Serializable]
    public sealed class AssetSourceRow
    {
        public string? asset_key;
        public string? file_path;
        public string? content_id;
        public string? source_kind;
        public string? creator;
        public string? source_uri;
        public string? license_id;
        public string? license_uri;
        public string? acquired_at_utc;
        public string? source_sha256;
        public string? final_sha256;
        public string? changes;
        public string? attribution;
        public GenerationRecord? generation_record;
        public FolkloreCard? folklore_card;
        public bool cultural_entity;
        public List<InputProvenance>? inputs;
        public string? review_state;
    }
}
