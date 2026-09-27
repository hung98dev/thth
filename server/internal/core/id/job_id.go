package id

// serverJobNamespace is SERVER_JOB_NAMESPACE_UUID
// (docs/00_context/technology_versions.md § Pinned Content System Constants,
// ADR-0070). Immutable: never rotate. Canonical text form:
// 64d34c40-8657-462b-887f-5970db9eaa5f.
var serverJobNamespace = UUID{
	0x64, 0xd3, 0x4c, 0x40, 0x86, 0x57, 0x46, 0x2b,
	0x88, 0x7f, 0x59, 0x70, 0xdb, 0x9e, 0xaa, 0x5f,
}

// ServerJobNamespaceUUID returns the pinned server-job namespace UUID.
func ServerJobNamespaceUUID() UUID { return serverJobNamespace }

// ServerJobUUID is the operation_id of a server-initiated job / admin /
// webhook: deterministic UUID v5 over SERVER_JOB_NAMESPACE_UUID and
// "<operation_family>:<job_key>" (ids.md § Operation IDs, ADR-0070). job_key is
// the job's stable natural key — account_id (erasure), utc_date + scope
// (scheduled rollups/settlements), provider notification ID (webhooks),
// audit_event_id (admin) — so a retry after a crash recomputes the same ID and
// nothing is persisted first.
func ServerJobUUID(operationFamily, jobKey string) UUID {
	return UUIDv5(serverJobNamespace, operationFamily+":"+jobKey)
}
