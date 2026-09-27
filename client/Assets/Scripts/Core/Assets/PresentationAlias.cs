using UnityEngine;

namespace ThinhThan.Core.Assets
{
    // One-hop shared-asset indirection of client_assets.md § Stable
    // Asset Keys: a catalog ID that reuses a shared asset keeps its own
    // key, addressed to a PresentationAlias whose TargetKey names the
    // shared key. An alias whose target resolves to another alias fails
    // validation.
    public class PresentationAlias : ScriptableObject
    {
        [SerializeField] private string _targetKey = "";

        public string TargetKey
        {
            get { return _targetKey; }
            set { _targetKey = value; }
        }
    }
}
