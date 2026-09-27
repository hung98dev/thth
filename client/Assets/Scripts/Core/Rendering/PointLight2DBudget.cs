using System.Collections.Generic;
using UnityEngine;
using UnityEngine.Rendering.Universal;

namespace ThinhThan.Core.Rendering
{
    // Caps active point Light2D sources to the quality preset budget
    // (ADR-0056 amendment: LOW 4, MEDIUM 8, HIGH 16 — the preset alone
    // decides, not the platform). Lights register at map composition;
    // Enforce runs from the frame pipeline's presentation phase and keeps
    // the lights nearest the focus enabled. Stable ordering keeps ties
    // deterministic; no LINQ and no per-frame allocation after registration.
    public sealed class PointLight2DBudget : MonoBehaviour
    {
        public const int LowPresetBudget = 4;
        public const int MediumPresetBudget = 8;
        public const int HighPresetBudget = 16;

        [SerializeField] private QualityPreset _preset = QualityPreset.Medium;
        [SerializeField] private Transform _focus = null!; // optional override; defaults to this transform

        private readonly List<Light2D> _lights = new List<Light2D>(64);

        public QualityPreset Preset
        {
            get => _preset;
            set => _preset = value;
        }

        public int Count => _lights.Count;

        public int Budget => MaxActivePointLights(_preset);

        public static int MaxActivePointLights(QualityPreset preset)
        {
            switch (preset)
            {
                case QualityPreset.Low:
                    return LowPresetBudget;
                case QualityPreset.High:
                    return HighPresetBudget;
                case QualityPreset.Medium:
                default:
                    return MediumPresetBudget;
            }
        }

        public void Register(Light2D light)
        {
            if (light == null || _lights.Contains(light))
            {
                return;
            }
            _lights.Add(light);
        }

        public void Unregister(Light2D light)
        {
            _lights.Remove(light);
        }

        // Keeps at most Budget lights enabled: the ones nearest the focus
        // point (the map object position when no focus is assigned).
        public void Enforce()
        {
            Enforce(_focus != null ? _focus.position : transform.position);
        }

        public void Enforce(Vector3 focusPosition)
        {
            SortByDistance(focusPosition);
            var budget = Budget;
            for (var i = 0; i < _lights.Count; i++)
            {
                _lights[i].enabled = i < budget;
            }
        }

        private void SortByDistance(Vector3 focusPosition)
        {
            for (var i = 1; i < _lights.Count; i++)
            {
                var light = _lights[i];
                var key = (light.transform.position - focusPosition).sqrMagnitude;
                var j = i - 1;
                while (j >= 0 && (_lights[j].transform.position - focusPosition).sqrMagnitude > key)
                {
                    _lights[j + 1] = _lights[j];
                    j--;
                }
                _lights[j + 1] = light;
            }
        }
    }
}
