# Known Implementation Blockers
status: LOCKED

## Scope

Live register of what stops tasks (structure locked; entries are live, ADR-0057):
- `BLK-xxx` — contract conflicts or gaps in specs/ADRs. Owner: `spec-owner` agent. Gate A.
- `OPS-xxx` — environment failures the agents cannot fix (runner offline, Unity licence, GPU missing, expired App key, disk full, ruleset drift). Owner: repository owner. Gate D. An exhausted Firebase Test Lab quota is `DEFERRED(quota)`, never an `OPS-xxx` entry (`../04_architecture/client_performance.md`).

Implementers only append entries and set the affected task `BLOCKED` with `blocked_by`. This file never chooses a product or architecture rule.

## Entry Format

```text
### `BLK-xxx` | `OPS-xxx` — <title>
opened_by: <agent/task>   opened_at: <UTC>
evidence: <file:line or CI run URL + log line>
owning spec / system: <path or component>
options: <2-3 options with one-line trade-offs>     (BLK only)
blocks: <IMP-IDs>
```

`OPS-xxx` entries are also filed as a GitHub issue labelled `ops-blocked`; agents stop and do not retry until the owner resolves it.

## Open Blockers

None. IDs start at `BLK-001` and `OPS-001`.

## Resolved Blockers

None.

## Resolution Rule

`BLK-xxx` (spec-owner, one spec-change PR `spec/BLK-xxx-<slug>`):
1. update the owning spec(s) and every consumer found by grep;
2. add or amend an ADR when architecture/data contracts change;
3. add the regression/conformance test that fails on the old contradiction to the unblocked packet's `## Tests`;
4. move the entry to Resolved and return blocked tasks to `NOT_STARTED`;
5. pass `policy-review`.

`OPS-xxx` (repository owner): fix the environment, close the issue, move the entry to Resolved; the coordinator returns the tasks to `NOT_STARTED` and clears `AUTO_MERGE_FROZEN` if it was set.

## Invariants

```text
open BLK => dependent task cannot be DONE; Gate A fails
open OPS => Gate D fails; agents stop
implementation never resolves a contract contradiction by guessing
removing a BLK requires a spec change plus a named regression test
```
