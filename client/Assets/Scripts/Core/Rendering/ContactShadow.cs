using UnityEngine;

namespace ThinhThan.Core.Rendering
{
    // Runtime soft contact shadow under an actor (ADR-0056): builds a unit
    // quad scaled to the profile's ellipse and draws it with the shared
    // ThinhThanContactShadow material — no per-actor material instance. All
    // allocation happens once here at actor load.
    public sealed class ContactShadow : MonoBehaviour
    {
        [SerializeField] private ActorSizeProfile _profile = ActorSizeProfile.Character;
        [SerializeField] private Material _shadowMaterial = null!; // ThinhThanContactShadow.mat assigned on the prefab
        [SerializeField] private int _sortingOrder = -1;

        public ActorSizeProfile Profile => _profile;

        private void Awake()
        {
            var spec = ContactShadowSpec.For(_profile);
            var child = new GameObject("ContactShadow");
            var childTransform = child.transform;
            childTransform.SetParent(transform, false);
            childTransform.localPosition = Vector3.zero;
            childTransform.localScale = new Vector3(spec.WidthMeters, spec.HeightMeters, 1f);
            var meshFilter = child.AddComponent<MeshFilter>();
            meshFilter.sharedMesh = BuildQuadMesh();
            var renderer = child.AddComponent<MeshRenderer>();
            renderer.sharedMaterial = _shadowMaterial;
            renderer.sortingOrder = _sortingOrder;
            renderer.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            renderer.receiveShadows = false;
        }

        private static Mesh BuildQuadMesh()
        {
            var mesh = new Mesh { name = "ContactShadowQuad" };
            mesh.vertices = new[]
            {
                new Vector3(-0.5f, -0.5f, 0f),
                new Vector3(0.5f, -0.5f, 0f),
                new Vector3(-0.5f, 0.5f, 0f),
                new Vector3(0.5f, 0.5f, 0f),
            };
            mesh.uv = new[]
            {
                new Vector2(0f, 0f),
                new Vector2(1f, 0f),
                new Vector2(0f, 1f),
                new Vector2(1f, 1f),
            };
            mesh.triangles = new[] { 0, 2, 1, 1, 2, 3 };
            mesh.bounds = new Bounds(Vector3.zero, new Vector3(1f, 1f, 0.01f));
            return mesh;
        }
    }
}
