using UnityEngine;

namespace ThinhThan.Core.Rendering
{
    // Authored day/night values for a map's Global Light2D (ADR-0056): the
    // driver lerps between these across the day/night transition. Each map
    // authors its own instance; Settings/Rendering/DefaultDayNight.asset is
    // the shared default.
    [CreateAssetMenu(fileName = "DayNight", menuName = "ThinhThan/Rendering/Day Night Profile")]
    public sealed class DayNightProfile : ScriptableObject
    {
        [SerializeField] private Color _dayColor = Color.white;
        [SerializeField] private float _dayIntensity = 1f;
        [SerializeField] private Color _nightColor = new Color(0.28f, 0.36f, 0.62f, 1f);
        [SerializeField] private float _nightIntensity = 0.35f;

        public Color DayColor => _dayColor;
        public float DayIntensity => _dayIntensity;
        public Color NightColor => _nightColor;
        public float NightIntensity => _nightIntensity;

        public void Evaluate(float nightFactor, out Color color, out float intensity)
        {
            color = Color.Lerp(_dayColor, _nightColor, nightFactor);
            intensity = Mathf.Lerp(_dayIntensity, _nightIntensity, nightFactor);
        }
    }
}
