# Deployment
status: LOCKED

## Scope
Defines environments, backend artifact flow, PostgreSQL migration order, content activation, rollout, drain, and rollback.

## Environments
At minimum:
```text
local/dev
staging
production
```

Staging uses production-like protocol/content/database migration flow and can execute load/fault tests without production player value.

## Artifacts
Backend deployment ships one immutable artifact per release: the statically linked `thinhthan-server` binary (`CGO_ENABLED=0`, `GOOS=linux`, `GOARCH=amd64`, cross-built by the pinned Go toolchain on the Windows CI runner) plus the pinned Mozilla CA bundle (`../00_context/technology_versions.md`). Production runs it as one systemd service on one Linux host (`deploy/prod/thinhthan-server.service`); PostgreSQL 18.6 runs on its own host. No container runtime or Kubernetes is used in production (one process, ADR-0052).

Artifact identity includes:
- source commit,
- build version,
- canonical technology-version-matrix revision,
- exact Go/Unity/protobuf toolchain versions where applicable,
- resolved direct dependency/lockfile checksum state,
- protocol support range,
- schema migration compatibility,
- content schema compatibility.

Unity client builds are versioned separately and gated through protocol/versioning rules. Production build/bootstrap tooling must fail rather than silently using an editor/tool/dependency version that differs from `../00_context/technology_versions.md`.

## Deployment Order
Safe default order for compatible changes:
1. deploy backward-compatible PostgreSQL migration (expand),
2. deploy backend capable of old+new representation,
3. verify health,
4. publish and verify any required immutable Addressables catalog/bundles before they are referenced,
5. activate compatible static content revision if part of release,
6. release/gate Unity build and compatible asset-catalog revision,
7. after compatibility window, remove retired paths in a later deploy (contract).

Do not deploy destructive DB change before all running code stops depending on the old shape.

## Migrations
Database migrations:
- are versioned,
- are reviewed/tested against production-like volume,
- have explicit lock/runtime expectations,
- use expand/contract for online changes,
- do not silently rewrite large tables inside startup path,
- are run once by the deploy step before the new process starts, never by the server at startup.

## Backend Deploy (Maintenance Restart)
The deployable unit is the single `thinhthan-server` world process (ADR-0044, ADR-0052); there are no replicas. A deploy is a maintenance restart announced in advance:
- stop accepting new logins/character entries on the old build,
- drain in-flight intra-world transfers within their budgets,
- allow bounded active instance completion where policy permits,
- terminate, then start the new build; remaining owners recover/reconnect through canonical checkpoint/entry recovery.

Do not have two processes simultaneously own one partition to make rollout appear seamless.

## Content Activation
Static content revision activation is separate from binary deployment and follows atomic validation in `../06_data/config.md`.

A binary must declare which content schema versions it supports.

Invalid candidate content never partially activates.

## Health Gates
A new backend cohort must pass:
- startup/schema/content compatibility,
- PostgreSQL connectivity/pool health,
- protocol handshake,
- synthetic durable mutation/read,
- tick health of the world simulation,
- error/reconnect/latency thresholds.

Failed cohort stops rollout.

## Rollback
Backend rollback is allowed only while DB/content remain compatible with the older binary.

If a migration is not backward compatible, restore/forward-fix plan must be explicit before production deployment.

Content rollback activates the previous validated immutable revision when persistence compatibility permits; it does not mutate old IDs in place.

### Schema-Coupled Content Revisions
A content revision is tagged **`schema-coupled`** when it changes a formula whose output is persisted to character rows — specifically: EXP threshold tables, level cap, or any other value stored in durable character state rather than recomputed on load. Examples: the EXP x100 rescale that writes `current_exp` and character level to PostgreSQL.

**Binary rollback of a schema-coupled revision is not permitted.** A content rollback that reverts EXP thresholds or level caps cannot undo committed character rows; characters would be re-evaluated against old thresholds and could land above the level cap or in an invalid progression bucket. When a schema-coupled revision must be reverted:
- a compensating migration that re-normalizes affected character rows is required, or
- a forward-only fix plan (new revision that repairs the invariant without reverting) must be prepared and deployed instead of a binary rollback.

The schema-coupled tag is recorded in the content revision metadata and must be present before a release that carries formula-persisting changes may be activated.

## Secrets
Secrets/config are injected at runtime and are not baked into Unity builds, repository files, or the server binary artifact.

## Production Change Rule
Every production release records:
- backend build,
- Unity minimum/current build gate,
- DB schema version,
- active content revision,
- required Unity asset_catalog_revision / Addressables build identity when applicable,
- migration set,
- operator/deployment identity and timestamp.

## Invariants
- immutable deploy artifacts,
- expand -> code -> contract migration pattern,
- simulation ownership is drained, never duplicated,
- content activation is atomic,
- rollback compatibility is known before deploy,
- production secrets never ship in Unity client,
- schema-coupled content revisions cannot be binary-rolled-back without a migration or forward-only fix plan.
