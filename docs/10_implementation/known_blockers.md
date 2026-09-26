# Known Implementation Blockers
status: LOCKED

## Scope

Live register of what stops tasks (structure locked; entries are live, ADR-0057):
- `BLK-xxx` — contract conflicts or gaps in specs/ADRs. Owner: `spec-owner` agent. Gate A.
- `OPS-xxx` — environment failures the agents cannot fix (hosted runner unavailable, Unity licence, expired App key or token, missing art tool, ruleset drift). Owner: repository owner. Gate D. An exhausted Firebase Test Lab quota is `DEFERRED(quota)`, never an `OPS-xxx` entry (`../04_architecture/client_performance.md`).

Implementers only append entries and set their task `BLOCKED` with `blocked_by`, in a status-only `block/IMP-XXX-<n>` PR that merges on the Q0-only fast path (`agent_execution_protocol.md` §6), so the entry is visible on `main`. This file never chooses a product or architecture rule.

## Entry Format

```text
### `BLK-xxx` | `OPS-xxx` — <title>
opened_by: <agent/task>   opened_at: <UTC>
evidence: <file:line or CI run URL + log line>
owning spec / system: <path or component>
options: <2-3 options with one-line trade-offs>     (BLK only)
blocks: <IMP-IDs> | ALL                              (OPS: ALL only for freeze, token, licence or ruleset failures)
issue: <ops-blocked issue URL>                       (OPS only)
```

`OPS-xxx` entries are also filed as a GitHub issue labelled `ops-blocked`. `blocks: ALL` stops every agent; a scoped entry stops only the listed tasks, which are not retried until the owner resolves it.

## Open Blockers

None. IDs start at `BLK-001` and `OPS-001`.

## Resolved Blockers

### `BLK-001` — URP materialization outputs not covered by IMP-000 owned_paths
opened_by: implementer/IMP-000   opened_at: 2026-09-26T00:04Z
evidence: `Q0.control.diff` FAIL on PR https://github.com/hung98dev/thth/pull/1, run https://github.com/hung98dev/thth/actions/runs/36199168601 job 108281973511 — `file client/Assets/DefaultVolumeProfile.asset outside IMP-000 owned_paths; file client/Assets/UniversalRenderPipelineGlobalSettings.asset outside IMP-000 owned_paths` (+ `.meta` companions)
owning spec / system: `docs/10_implementation/task_queue.md` (IMP-000 `owned_paths`, `generated_artifacts`), `docs/10_implementation/repository_layout.md` (Path Ownership Index)
options:
  1. add the 4 paths (`client/Assets/DefaultVolumeProfile.asset`, `.meta`, `client/Assets/UniversalRenderPipelineGlobalSettings.asset`, `.meta`) to IMP-000 `owned_paths` (+ layout index) — the editor materializes them during IMP-000's own CI and §4b requires committing them byte-for-byte, so IMP-000 is their de-facto owner;
  2. exempt editor-materialized files from `Q0.control.diff` — weakens the gate's ownership coverage;
  3. assign the 2 URP assets to a later packet (e.g. IMP-101 URP setup) — that packet cannot commit them until it runs, but IMP-000's CI must commit them now to close the materialization loop.
blocks: IMP-000
resolved_by: spec/BLK-001-urp-root-assets   resolved_at: 2026-09-26T00:23Z   resolution: option 1 — IMP-000 owns the 4 paths; `.meta` listed explicitly (Q0 flagged them literally); IMP-000 -> NOT_STARTED

## Resolution Rule

`BLK-xxx` (spec-owner, one spec-change PR `spec/BLK-xxx-<slug>`):
1. update the owning spec(s) and every consumer found by grep;
2. add or amend an ADR when architecture/data contracts change;
3. add the regression/conformance test that fails on the old contradiction to the unblocked packet's `## Tests`;
4. move the entry to Resolved and return blocked tasks to `NOT_STARTED`;
5. pass `policy-review`.

`OPS-xxx`: the repository owner fixes the environment and closes the `ops-blocked` issue (the owner never edits this file). The coordinator then lands a status-only `ops/OPS-xxx-resolved` PR that moves the entry to Resolved and returns the tasks it blocked to `NOT_STARTED`; the post-merge guard run of that merge clears `AUTO_MERGE_FROZEN` if it was set (`audit_gates.md` § Gate D, ADR-0072).

## Invariants

```text
open BLK => dependent task cannot be DONE; Gate A fails
open OPS => Gate D fails; blocks: ALL stops every agent, a scoped OPS stops only its listed tasks
implementation never resolves a contract contradiction by guessing
removing a BLK requires a spec change plus a named regression test
```
