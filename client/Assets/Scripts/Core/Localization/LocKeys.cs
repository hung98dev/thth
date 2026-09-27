namespace ThinhThan.Core.Localization
{
    /// <summary>
    /// Canonical constants for the bilingual localization contract
    /// (docs/04_architecture/client_localization.md, ADR-0015).
    /// Launch locales: vi-VN is the project default; en-US is the only
    /// supported alternate. PlayerPref <c>TT_LOCALE</c> stores the saved choice.
    /// </summary>
    public static class LocKeys
    {
        public const string StringTableName = "Core";
        public const string LocaleViVn = "vi-VN";
        public const string LocaleEnUs = "en-US";
        public const string PlayerPrefsKey = "TT_LOCALE";

        /// <summary>Developer-facing marker emitted for a missing key or value.</summary>
        public const string MissingPrefix = "[MISSING:";
        public const string MissingSuffix = "]";

        /// <summary>True when <paramref name="localeCode"/> is a supported launch locale.</summary>
        public static bool IsSupportedLocale(string localeCode)
        {
            return localeCode == LocaleViVn || localeCode == LocaleEnUs;
        }
    }
}
