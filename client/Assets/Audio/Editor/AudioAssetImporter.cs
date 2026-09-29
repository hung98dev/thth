using UnityEditor;
using UnityEngine;

namespace ThinhThan.Audio.Editor
{
    // IMP-075 audio import rules: stamps the canonical import profile
    // onto every .ogg under Assets/Audio/ so CI materialization produces
    // identical .meta files byte-for-byte. The declaration lives in the
    // committed .meta userData (asset_class=AUDIO plus audio_role=bgm or
    // audio_role=sfx), read back by the importer and by
    // AudioAssetCoverageTests:
    //   bgm -> Streaming + Vorbis + loadInBackground (manifest §1: BGM
    //          streams directly, never loads whole files into RAM);
    //   sfx -> DecompressOnLoad + Vorbis so EditMode tests and runtime
    //          one-shots can decode PCM without allocations.
    public sealed class AudioAssetImporter : AssetPostprocessor
    {
        private const string AudioRoot = "Assets/Audio/";

        private void OnPreprocessAudio()
        {
            if (!assetPath.StartsWith(AudioRoot, System.StringComparison.Ordinal)
                || !assetPath.EndsWith(".ogg", System.StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
            var importer = (AudioImporter)assetImporter;
            var userData = (importer.userData ?? string.Empty).Trim();
            if (userData.Length == 0)
            {
                return;
            }
            var isBgm = Declared(userData, "audio_role") == "bgm";

            importer.forceToMono = false;
            importer.normalize = true;
            importer.loadInBackground = isBgm;
            importer.ambisonic = false;

            var defaults = importer.defaultSampleSettings;
            defaults.loadType = isBgm
                ? AudioClipLoadType.Streaming
                : AudioClipLoadType.DecompressOnLoad;
            defaults.compressionFormat = AudioCompressionFormat.Vorbis;
            defaults.quality = isBgm ? 0.5f : 0.7f;
            defaults.sampleRateSetting = AudioSampleRateSetting.PreserveSampleRate;
            defaults.preloadAudioData = !isBgm;
            importer.defaultSampleSettings = defaults;
            importer.userData = userData;
        }

        private static string Declared(string userData, string key)
        {
            foreach (var raw in userData.Split(';'))
            {
                var token = raw.Trim();
                var eq = token.IndexOf('=');
                if (eq > 0 && token.Substring(0, eq).Trim() == key)
                {
                    return token.Substring(eq + 1).Trim();
                }
            }
            return "";
        }
    }
}
