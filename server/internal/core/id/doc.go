// Package id is the canonical owner of identity primitives for the
// Thỉnh Thần server: RFC 4122 UUID v4 generation and validation (crypto/rand),
// deterministic UUID v5 derivation for content-grant idempotency keys and
// server-job operation IDs, static content ID grammar, owner/epoch-scoped
// runtime entity IDs, operation payload fingerprints and content revision
// diagnostic context.
//
// Canonical contract: docs/06_data/ids.md and docs/06_data/config.md
// (ADR-0001, ADR-0043, ADR-0064, ADR-0065, ADR-0070). Every other package must
// use this owner; a second UUID/static-ID/entity-ID implementation is a
// defect.
package id
