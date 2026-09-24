# Backup and Recovery
status: LOCKED

## Scope
Defines launch backup scope, PostgreSQL recovery targets, restore validation, and disaster-recovery behavior.

## Durable Scope
Back up all canonical PostgreSQL durable gameplay/account state required to reconstruct:
- characters/progression,
- inventory/equipment/currency,
- quest/build/Soul state,
- guild/social durable state,
- Auction/trade escrow/proceeds,
- Reward Claims,
- cosmetics/entitlements,
- **WorldConsequence durable aggregate** (Di Tích world-consequence state; a partition must load this before accepting players — a silently-zeroed aggregate is an unsafe restore, not a degraded-mode restore),
- idempotency/audit records required for safe replay/recovery,
- active content revision metadata.

In-memory simulation state is not a backup target.

## PostgreSQL Strategy
Launch requires:
- continuous WAL archiving/PITR capability,
- at least one automated full/base backup daily,
- backups stored independently from the primary database failure domain,
- encryption in transit and at rest according to infrastructure policy.

## Targets
Launch disaster-recovery targets:
```text
RPO <= 5 minutes
RTO <= 60 minutes
```

RPO/RTO are measured restore objectives, not promises that every process resumes its exact in-memory combat state.

## Retention
Minimum:
```text
daily restore points: 14 days
weekly restore points: 8 weeks
monthly restore points: 6 months
```

Longer retention may be configured for compliance/business needs.

**Erasure ledger**: every executed account erasure appends `{account_id_hash, executed_at, operation_id}` to an append-only ledger object stored in the backup storage (outside the database, retained as long as the longest restore point). It holds no personal data beyond the salted hash and exists only to re-apply erasures after a restore. Restore points older than 6 months are deleted, so erased data leaves backups within 6 months.

## Restore Procedure
Recovery runbook:
1. stop/cordon writes,
2. choose verified restore point/PITR timestamp,
3. restore into isolated environment,
4. run database integrity/schema checks,
5. verify critical counts/foreign keys/idempotency tables,
6. reconcile Auction escrow/proceeds and Reward Claims,
6a. reconcile WorldConsequence aggregate: load from backup, verify schema, foreign keys, and internal consistency against the checkpoint manifest; an absent table or corrupted payload **must fail the restore**. A zero-row count is valid if and only if no world boss was defeated prior to the backup checkpoint (e.g. pre-defeat state or fresh launch); do not promote a recovered database where partitions would start against stale or missing world-consequence state,
6b. re-validate time-sensitive expiries: relic expiry timestamps and timed cosmetic expiry timestamps were stored against server wall time; after PITR, re-evaluate all `timestamptz` expiry fields against the restored-as-of timestamp (not the restore-execution time) and mark expired entries as expired; entries that fall in the recovery window must be resolved by policy, not silently carried forward as active,
6c. re-apply erasures: replay every entry of the erasure ledger executed after the restore point (deletion/anonymisation per `../06_data/data_model.md`) before promotion; a restore that would resurrect erased personal data **must not be promoted**,
7. verify active content/schema compatibility,
8. run smoke login/character/inventory/economy tests,
9. promote recovered database,
10. restart routing/simulation from canonical recovery rules.

Do not restore client snapshots as server truth.

## Backup Verification
A backup is not considered valid merely because upload succeeded.

Required:
- automated backup job success monitoring,
- checksum/integrity validation,
- scheduled restore drill at least monthly,
- documented measured restore duration,
- sampled application-level verification after restore.

## Point-in-Time Recovery Safety
PITR may replay committed DB transactions only; external side effects must be reconciled using operation/audit IDs.

After recovery, idempotency records must prevent clients/workers from duplicating already-restored value mutations.

## Incident Behavior
If PostgreSQL durability cannot be guaranteed:
- reject/fail closed for value-changing operations,
- do not continue issuing items/currency in an unpersisted memory-only mode,
- simulation may enter controlled drain/maintenance behavior.

## Invariants
- canonical durable state is recoverable from PostgreSQL backup/PITR.
- RPO <=5m, RTO <=60m target.
- backups are outside primary failure domain.
- monthly restore drill is mandatory.
- value mutation never falls back to non-durable success.
- WorldConsequence aggregate is in backup scope; an absent table or corrupted payload fails the restore (zero rows is valid only pre-defeat).
- after PITR, timestamptz expiries are re-validated against the restored-as-of time, not restore-execution time.
