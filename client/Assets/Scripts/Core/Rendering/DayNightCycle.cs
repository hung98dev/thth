using UnityEngine;

namespace ThinhThan.Core.Rendering
{
    // Day/night blend math for a map's Global Light2D (world_rules.md § Day
    // and Night): one world day is 120 real minutes = 80 minutes of day
    // followed by 40 minutes of night. The light colour and intensity
    // interpolate linearly across a 5-minute transition centred on each
    // boundary (dusk at t=80, dawn at t=0). Presentation-only; gameplay
    // reads no multipliers from it.
    public static class DayNightCycle
    {
        public const float WorldDayMinutes = 120f;
        public const float DayMinutes = 80f;
        public const float NightMinutes = 40f;
        public const float TransitionMinutes = 5f;

        // Returns the night blend factor in [0,1]: 0 during day, 1 during
        // night, a linear ramp across each 5-minute transition.
        public static float NightFactor(float worldMinutes)
        {
            var half = TransitionMinutes * 0.5f;
            var t = Mathf.Repeat(worldMinutes, WorldDayMinutes);
            if (t < half)
            {
                // dawn tail: t in [0, 2.5)
                return 1f - (t + half) / TransitionMinutes;
            }
            if (t < DayMinutes - half)
            {
                // day
                return 0f;
            }
            if (t < DayMinutes + half)
            {
                // dusk: t in [77.5, 82.5)
                return (t - (DayMinutes - half)) / TransitionMinutes;
            }
            if (t < WorldDayMinutes - half)
            {
                // night
                return 1f;
            }
            // dawn: t in [117.5, 120)
            return 1f - (t - (WorldDayMinutes - half)) / TransitionMinutes;
        }
    }
}
