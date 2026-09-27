namespace ThinhThan.Core.Assets
{
    // Deterministic memory model of presentation_asset_manifest.md §1:
    // runtime RAM is computed from import settings (texture format x
    // size x mip count + mesh + decompressed audio), never measured
    // from a live process.
    public static class AssetRamModel
    {
        // Fixed structural cost charged per addressable entry whose
        // payload is not a measured texture/audio file (placeholder
        // ScriptableObjects, prefab/scene stubs, alias assets).
        public const long StructuralBytesPerEntry = 16 * 1024;

        // Streamed BGM never loads whole files into RAM: the manifest
        // allows a bounded streaming buffer per audio group.
        public const long BgmStreamingBufferBytes = 4L * 1024 * 1024;

        public static long TextureBytes(int width, int height, TextureBlockFormat format, bool mipmaps)
        {
            if (width <= 0 || height <= 0)
            {
                return 0;
            }
            long total = 0;
            var w = width;
            var h = height;
            while (true)
            {
                total += LevelBytes(w, h, format);
                if (!mipmaps || (w <= 1 && h <= 1))
                {
                    break;
                }
                w = w > 1 ? w / 2 : 1;
                h = h > 1 ? h / 2 : 1;
            }
            return total;
        }

        private static long LevelBytes(int w, int h, TextureBlockFormat format)
        {
            switch (format)
            {
                case TextureBlockFormat.Astc4x4:
                case TextureBlockFormat.Bc7:
                    return (long)CeilDiv(w, 4) * CeilDiv(h, 4) * 16;
                case TextureBlockFormat.Astc6x6:
                    return (long)CeilDiv(w, 6) * CeilDiv(h, 6) * 16;
                default:
                    return (long)w * h * 4;
            }
        }

        private static int CeilDiv(int v, int d)
        {
            return (v + d - 1) / d;
        }

        // A non-streamed clip decompresses to PCM16 on load.
        public static long AudioClipBytes(double seconds, int sampleRate, int channels)
        {
            if (seconds <= 0 || sampleRate <= 0 || channels <= 0)
            {
                return 0;
            }
            var frames = (long)(seconds * sampleRate);
            return frames * channels * 2;
        }
    }
}
