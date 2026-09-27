package id

// contentGrantNamespace is CONTENT_GRANT_NAMESPACE_UUID
// (docs/00_context/technology_versions.md § Pinned Content System Constants).
// Immutable: changing it invalidates every previously issued grant key.
// Canonical text form: f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b.
var contentGrantNamespace = UUID{
	0xf7, 0xa3, 0xd2, 0xb1, 0x4e, 0x8c, 0x4a, 0x2f,
	0x9b, 0x3e, 0x6d, 0x1c, 0x5f, 0x8e, 0x7a, 0x2b,
}

// ContentGrantNamespaceUUID returns the pinned project content-grant namespace
// UUID. It is never the RFC 4122 DNS or URL namespace (ids.md).
func ContentGrantNamespaceUUID() UUID { return contentGrantNamespace }

// ContentGrantUUID derives the deterministic idempotency key of a one-time
// content grant (seasonal cosmetics, Atlas reward tiers, Guild Stone
// completions): UUID v5 over CONTENT_GRANT_NAMESPACE_UUID and the grant-scope
// string defined per grant type by the owning system spec, e.g.
// "season.<season_number>.<cosmetic_id>.<character_id>" (ids.md §
// Deterministic Content-Grant Idempotency Keys). The same scope always derives
// the same key across retries and restarts without persisting it first.
func ContentGrantUUID(grantScope string) UUID {
	return UUIDv5(contentGrantNamespace, grantScope)
}
