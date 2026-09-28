// Package rng is the canonical owner of deterministic gameplay randomness
// for the Thỉnh Thần server: a seeded PCG-64 stream (Go math/rand/v2) per
// typed RNG context, derived from stable authority inputs — content revision,
// context kind/subject ID and the server-owned seed. Every gameplay roll
// (combat, drop tables, enhancement, spawn pools, boss resample, fishing
// catch) must go through this owner; a second gameplay RNG implementation
// and math/rand (v1) are defects (engineering_conventions.md § 1.3,
// architecture_conformance.md § 3).
//
// Canonical contract: docs/04_architecture/concurrency.md § RNG Concurrency,
// docs/06_data/config.md § Probability, docs/09_testing/gameplay.md
// § Regression Seeds (ADR-0007, ADR-0053, ADR-0061). Authoritative RNG is
// never one unsynchronized global stream shared across partitions and is
// never supplied by the client: the server owns every roll outcome.
// UUID/secrets remain crypto/rand (docs/06_data/ids.md).
package rng
