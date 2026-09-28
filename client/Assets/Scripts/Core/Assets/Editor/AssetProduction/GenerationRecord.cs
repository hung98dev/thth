using System;
using System.Collections.Generic;

namespace ThinhThan.Core.Assets.Editor.AssetProduction
{
    // AI generation provenance (section 6): tool + version + terms at
    // creation time + prompt + every reference URI. Required iff
    // source_kind == AI_CREATED; a third-party reference must also appear
    // under inputs[] so it cannot be hidden.
    [Serializable]
    public sealed class GenerationRecord
    {
        public string? tool;
        public string? version;
        public string? terms_uri;
        public string? prompt;
        public List<string>? reference_uris;

        // JsonUtility materializes an absent object field as an empty
        // instance rather than null; IsEmpty distinguishes the two.
        public bool IsEmpty()
        {
            return string.IsNullOrEmpty(tool)
                && string.IsNullOrEmpty(version)
                && string.IsNullOrEmpty(terms_uri)
                && string.IsNullOrEmpty(prompt)
                && (reference_uris == null || reference_uris.Count == 0);
        }
    }
}
