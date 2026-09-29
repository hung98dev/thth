using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // AI generation provenance (section 6, extended by ADR-0076): tool +
    // version + model identity + terms at creation time (URI plus the
    // stored snapshot hash) + prompt + reproducibility fields (seed,
    // parameters, workflow hash, style pack, reference hashes, C2PA flag).
    // Required iff source_kind == AI_CREATED; a third-party reference must
    // also appear under inputs[] so it cannot be hidden.
    [Serializable]
    public sealed class GenerationRecord
    {
        public string? tool;
        public string? version;
        public string? model_id;
        public string? model_sha256;
        public string? terms_uri;
        public string? terms_snapshot_sha256;
        public string? prompt;
        // -1 = absent: JsonUtility cannot bind Nullable<T>, so the field
        // uses a negative sentinel; valid seeds are non-negative.
        public long seed = -1;
        public string? parameters;
        public string? workflow_sha256;
        public string? style_pack_id;
        public List<string>? reference_uris;
        public List<string>? reference_sha256;
        public bool c2pa_present;

        // JsonUtility materializes an absent object field as an empty
        // instance rather than null; IsEmpty distinguishes the two.
        public bool IsEmpty()
        {
            return string.IsNullOrEmpty(tool)
                && string.IsNullOrEmpty(version)
                && string.IsNullOrEmpty(terms_uri)
                && string.IsNullOrEmpty(prompt)
                && seed < 0
                && (reference_uris == null || reference_uris.Count == 0);
        }
    }
}
