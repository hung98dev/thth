namespace ThinhThan.Core.Assets
{
    // Size-profile <-> canonical name and cell dimensions.
    // Names are the uppercase spellings of presentation_asset_manifest.md
    // §3 (CHARACTER, MONSTER_SMALL, ...).
    public static class SpriteSizeProfileExtensions
    {
        private static readonly string[] _names =
        {
            "CHARACTER",
            "MONSTER_SMALL",
            "MONSTER_MEDIUM",
            "MONSTER_ELITE",
            "BOSS_LARGE",
            "WORLD_BOSS",
            "SPIRIT_BEAST",
        };

        // Cell texture size in final texture pixels (2 x reference cell).
        private static readonly int[] _cellWidth = { 192, 128, 192, 320, 512, 640, 128 };
        private static readonly int[] _cellHeight = { 256, 128, 256, 384, 512, 640, 128 };

        public static string ToProfileName(this SpriteSizeProfile profile)
        {
            return _names[(int)profile];
        }

        public static bool TryParseProfileName(string name, out SpriteSizeProfile profile)
        {
            for (var i = 0; i < _names.Length; i++)
            {
                if (_names[i] == name)
                {
                    profile = (SpriteSizeProfile)i;
                    return true;
                }
            }
            profile = default;
            return false;
        }

        public static int CellTextureWidth(this SpriteSizeProfile profile)
        {
            return _cellWidth[(int)profile];
        }

        public static int CellTextureHeight(this SpriteSizeProfile profile)
        {
            return _cellHeight[(int)profile];
        }
    }
}
