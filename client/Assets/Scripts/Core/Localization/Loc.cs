using System.Collections.Generic;
using UnityEngine.Localization;
using UnityEngine.Localization.Settings;
using UnityEngine.Localization.Tables;

namespace ThinhThan.Core.Localization
{
    /// <summary>
    /// Runtime accessor for the <c>Core</c> string table (client_localization.md).
    /// Keys are <c>loc.&lt;domain&gt;.&lt;stable_segments&gt;</c>. A missing key or
    /// value resolves to <c>[MISSING:&lt;key&gt;]</c> in development builds and is
    /// a release validation failure — see Tests/EditMode/LocalizationValidationTests.
    /// Smart String arguments are caller-supplied, already-computed values; this
    /// helper never computes gameplay state.
    /// </summary>
    public static class Loc
    {
        /// <summary>The dev-facing marker for an absent key or empty translation.</summary>
        public static string Missing(string key)
        {
            return LocKeys.MissingPrefix + key + LocKeys.MissingSuffix;
        }

        /// <summary>
        /// Synchronous lookup in the startup locale's string table. Call sites must
        /// run after LocalizationSettings initialization (it is synchronous when
        /// the startup table is preloaded; see LocalizationSettings asset).
        /// </summary>
        public static string Get(string key)
        {
            string value = LocalizationSettings.StringDatabase
                .GetLocalizedString(LocKeys.StringTableName, key);
            return string.IsNullOrEmpty(value) ? Missing(key) : value;
        }

        /// <summary>
        /// Smart String lookup with named, already-computed arguments
        /// (<c>{price}</c>, <c>{entity}</c>). The dictionary is passed as the sole
        /// argument list entry so Smart Format resolves names via DictionarySource.
        /// </summary>
        public static string Format(string key, Dictionary<string, object> args)
        {
            var localized = new LocalizedString(LocKeys.StringTableName, key);
            string value = localized.GetLocalizedString(args);
            return string.IsNullOrEmpty(value) ? Missing(key) : value;
        }

        /// <summary>
        /// Persisted locale change for the settings menu: writes the PlayerPrefs
        /// slot the LaunchLocaleSelector reads, then switches the active locale.
        /// Switching does not reload the table set — Unity Localization swaps the
        /// bound table in place, so this path performs no per-string reallocation.
        /// </summary>
        public static void SetLocale(string localeCode)
        {
            if (!LocKeys.IsSupportedLocale(localeCode))
            {
                return;
            }
            UnityEngine.PlayerPrefs.SetString(LocKeys.PlayerPrefsKey, localeCode);
            UnityEngine.PlayerPrefs.Save();
            Locale target = LocalizationSettings.AvailableLocales.GetLocale(
                new LocaleIdentifier(localeCode));
            if (target != null)
            {
                LocalizationSettings.SelectedLocale = target;
            }
        }
    }
}
