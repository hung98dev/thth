using System;
using System.Globalization;
using UnityEngine;
using UnityEngine.Localization;
using UnityEngine.Localization.Settings;

namespace ThinhThan.Core.Localization
{
    /// <summary>
    /// Startup locale selection per client_localization.md:
    /// saved PlayerPrefs choice (when it names a supported locale) wins; otherwise
    /// an OS culture in the <c>en-*</c> family selects en-US and every other OS
    /// culture selects the project default vi-VN.
    /// </summary>
    [Serializable]
    public class LaunchLocaleSelector : IStartupLocaleSelector
    {
        public Locale GetStartupLocale(ILocalesProvider availableLocales)
        {
            string code = PlayerPrefs.GetString(LocKeys.PlayerPrefsKey, null);
            if (LocKeys.IsSupportedLocale(code))
            {
                return availableLocales.GetLocale(new LocaleIdentifier(code));
            }

            if (CultureInfo.CurrentUICulture.TwoLetterISOLanguageName == "en")
            {
                return availableLocales.GetLocale(new LocaleIdentifier(LocKeys.LocaleEnUs));
            }

            return availableLocales.GetLocale(new LocaleIdentifier(LocKeys.LocaleViVn));
        }
    }
}
