using UnityEngine;
using UnityEngine.Rendering.Universal;

namespace ThinhThan.Core.Rendering
{
    // Drives a map's Global Light2D colour and intensity from world time
    // (world_rules.md § Day and Night). The frame pipeline's presentation
    // phase calls ApplyWorldTime — this component registers no Unity frame
    // callback.
    public sealed class GlobalLightDayNightDriver : MonoBehaviour
    {
        [SerializeField] private Light2D _globalLight = null!; // Global Light2D assigned on the map scene
        [SerializeField] private DayNightProfile _profile = null!; // per-map authored day/night profile

        public void ApplyWorldTime(float worldMinutes)
        {
            _profile.Evaluate(DayNightCycle.NightFactor(worldMinutes), out var color, out var intensity);
            _globalLight.color = color;
            _globalLight.intensity = intensity;
        }
    }
}
