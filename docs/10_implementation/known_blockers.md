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

### `BLK-005` — `verify.yml` `client/Library` cache `restore-keys` prefix fallback restores a stale Library and breaks the Unity job (ugui CS86xx; deterministic editor self-SIGKILL at precompiled-dll registration)
opened_by: implementer/IMP-063   opened_at: 2026-09-27T03:30Z
evidence: `.github/workflows/verify.yml` — the `Cache Unity client/Library (pinned)` step falls back via `restore-keys` to a Library produced under a different project state (manifest/lock/compiler flags). On PR #17 (`imp/IMP-063-addressables`) the restored Library makes Unity recompile `Library/PackageCache/com.unity.ugui@af15df11c39e` under the project `Assets/csc.rsp` (`-nullable:enable`), producing CS86xx errors on attempt 1; the post-failure suppression step (~`Inject package nullable suppressions`, writes `csc.rsp -nullable:disable -warnaserror-` next to package `*.asmdef`) does not cover `com.unity.ugui` (built-in package). The same runs then die deterministically — every `-runTests` launch is `Killed` ~5 s in, at `Registered in <t> seconds` during `Registering precompiled unity dll's`, before user script compilation (`docker inspect`: `OOMKilled=false`, `ExitCode=137`; eBPF `signal_generate` trace: `sig=9` sent by `Unity Main Thre` with `grp=1` = editor sends SIGKILL to its own process group, ~14 GB free). Reproduced on: PR #17 runs https://github.com/hung98dev/thth/actions/runs/36287375718 (head `1af463d`) and https://github.com/hung98dev/thth/actions/runs/36290085829 (head `6d8da3e`, 5/5 EditMode kills); identical kills on a different branch — IMP-061's failed run https://github.com/hung98dev/thth/actions/runs/36285753712 (5/5 kills at the same log line), while IMP-061's green run https://github.com/hung98dev/thth/actions/runs/36287628798 passed EditMode 15/15 on attempt 1. Correlation hypothesis: the kills cluster on runs whose restored `ScriptAssemblies`/PackageCache state is inconsistent with the checkout (project content was verified compilable — a later attempt reached user-code compile errors like `CS0103`/`CS8604`); cannot be fixed inside IMP-063 `owned_paths` (`verify.yml` is not owned; the worktree diff itself is confined to `client/Assets/{AddressableAssetsData,Scripts/Core/Assets,Tests/EditMode/AddressablesValidation}/`).
owning spec / system: `.github/workflows/verify.yml` (Unity Library cache key + `restore-keys`, package nullable-suppression coverage), `docs/10_implementation/audit_gates.md` (Gate D), `docs/10_implementation/agent_execution_protocol.md` §4b (materialization loop consumes `unity-materialized-<os>` artifacts that these jobs must produce)
options:
  1. restore Library on exact key match only (miss = cold import, slower but correct) or fold the compiler-flag/asmdef fingerprint into the cache key so prefix restores cannot produce a mismatched `ScriptAssemblies`/`PackageCache` pair — fixes both symptom classes if the correlation holds;
  2. keep `restore-keys` but extend the suppression step to cover built-in packages (`com.unity.ugui` etc.) — fixes only the CS86xx symptom, leaves the self-SIGKILL unexplained;
  3. cache only `Library/PackageCache` (immutable by lock hash) and never `ScriptAssemblies`/project subdirs — most content-addressable portion kept, stale-state risk removed.
blocks: IMP-063
resolved_by: `spec/BLK-005-library-cache` (PR https://github.com/hung98dev/thth/pull/TBD)   resolved_at: 2026-09-27T05:02Z   resolution: option 1 — `unity-library` restores on exact key only on both jobs (`restore-keys` removed; `client/Assets/csc.rsp` folded into the content hash so a flag change also misses); §6 Keys + CI-001 + `.devin/scripts/cache-policy.md` tightened: content-derived caches forbid `restore-keys` while content-addressed stores (`go-build`) and pure-pin payloads (`unity-image`, `edb`) keep them; suppression coverage unchanged — `com.unity.ugui` ships `Runtime/UnityEngine.UI.asmdef`, so the asmdef-adjacent inject covers it once PackageCache is consistent (cold builds were historically green on ugui); regression test `TestLibraryCacheExactKeyOnly`; IMP-063 -> NOT_STARTED

### `BLK-003` — server/go.mod lacks the pinned protobuf module; IMP-000 owns the lockfiles IMP-061 must extend
opened_by: implementer/IMP-061   opened_at: 2026-09-26T17:20Z
evidence: `server/go.mod` on main @ d5a4ebf declares `module thinhthan` + `go 1.27.1` and nothing else; IMP-061's generated `server/internal/protocol/v1/*.pb.go` imports `google.golang.org/protobuf/reflect/protoreflect` + `runtime/protoimpl`, so `go build ./...` / `go vet ./...` fail with "no required module provides package google.golang.org/protobuf/...". The module is already pinned (`server/internal/stackpin/pins.go` `GoModulePins["google.golang.org/protobuf"] = v1.36.12`, `docs/00_context/technology_versions.md`) and IMP-000's own acceptance required go.mod to "use the pinned Go/direct-module versions", yet the lockfile pair `server/go.mod` + `server/go.sum` is listed under IMP-000 `owned_paths` (`task_queue.md` path ownership index) — an implementer PR adding the `require` is rejected by `Q0.control.diff` ("file server/go.mod outside IMP-061 owned_paths", `server/internal/conformance/gates/diff.go`). Reproduced: `pwsh -NoProfile -File scripts/codegen.ps1` then `go -C server build ./...` on main @ d5a4ebf → module-resolution failure; adding `require google.golang.org/protobuf v1.36.12` (exact `GoModulePins` pin) makes `go build ./...` + `go vet ./...` pass.
owning spec / system: `docs/10_implementation/task_queue.md` (IMP-000 owned_paths + acceptance), `docs/00_context/technology_versions.md`, `server/internal/stackpin/pins.go`, `server/internal/conformance/gates/diff.go`
options:
  1. spec/ PR adds `require google.golang.org/protobuf v1.36.12` to `server/go.mod` + the `go.sum` lines on `main` (IMP-000 follow-up under spec-owner scope) — smallest change; IMP-061 then needs no lockfile edit;
  2. amend the ownership index so `server/go.mod`/`server/go.sum` are multi-owned (IMP-000 + whichever task first needs a pinned module) — covers every later task that adds a dependency (pgx, websocket, otel, …) instead of fixing IMP-061 alone;
  3. move generated Go code behind a second module — violates the "one Go module `thinhthan`" invariant (`repository_layout.md`), do not use.
blocks: IMP-061
resolved_by: `spec/BLK-003-protobuf-require` (PR https://github.com/hung98dev/thth/pull/16)   resolved_at: 2026-09-26T22:50Z   resolution: options 1+2 — spec/ PR lands `require google.golang.org/protobuf v1.36.12` + `go.sum` (IMP-061 needs no lockfile edit) AND § Ownership Rules makes `server/go.mod`/`server/go.sum` co-ownable (a packet may list both to land its own pinned `require` lines), so no repeat BLK on later deps; IMP-061 -> NOT_STARTED

### `BLK-002` — `Q0.bootstrap.absent_paths` forbids owned feature paths unconditionally
opened_by: implementer/IMP-063   opened_at: 2026-09-26T17:16Z
evidence: `server/internal/conformance/gates/q0.go` `checkBootstrapAbsence` stats `client/Assets/AddressableAssetsData` (plus `proto/`, `server/migrations`, `server/cmd/server`, `client/Assets/Scenes`, `client/Assets/Prefabs`, `client/Assets/Art`, `client/Assets/Localization`, `client/Assets/Settings`) with no owner/task gating, so the check fails on the owning task's own PR and on every PR after the path lands — local run reproduced: `Q0.bootstrap.absent_paths | FAIL | client/Assets/AddressableAssetsData must not exist before its owning task` on a tree containing only IMP-063's owned directory. Blast radius: 19 packets own children of the listed roots (IMP-005, IMP-006, IMP-061, IMP-062, IMP-063, IMP-064, IMP-067, IMP-069, IMP-070, IMP-071, IMP-072, IMP-073, IMP-074, IMP-075, IMP-076, IMP-095, IMP-101, IMP-104, IMP-105); `blocks:` names only IMP-063 because `checkOpenBlockerGating` would fail this PR over IMP-061 already `IN_PROGRESS`.
owning spec / system: `docs/10_implementation/audit_gates.md` (Q0 contract), `docs/10_implementation/task_queue.md` (packet `owned_paths`), `docs/10_implementation/repository_layout.md` (Path Ownership Index), `server/internal/conformance/gates/q0.go` (`checkBootstrapAbsence`)
options:
  1. make `checkBootstrapAbsence` owner-aware — flag a listed path only when its owning packet is `NOT_STARTED` on both base and head (path exists <=> owner has run) — preserves the gate's intent; requires the verifier to parse the task queue, which it already does;
  2. scope the check to the PR diff — fail only when a PR adds a listed path its task does not own — weaker: never re-checks paths that already landed on main;
  3. remove the check — simplest, but loses the "absent until its owning task" invariant entirely.
blocks: IMP-063
resolved_by: `spec/BLK-002-owner-aware-bootstrap` (PR https://github.com/hung98dev/thth/pull/15)   resolved_at: 2026-09-26T22:50Z   resolution: option 1 — `checkBootstrapAbsence` is owner-aware (`bootstrapOwnerStarted`: a root/file unblocks when a packet owning it, a path under it, or a parent of it is `IN_PROGRESS`/`DONE`; unowned roots stay forbidden); `blocksLineRe` made case-insensitive; `verify -MergeReports` skips non-IMP branches; IMP-000 acceptance text now states the status semantics; IMP-063 -> NOT_STARTED, IMP-061 stays BLOCKED on BLK-003

### `BLK-004` — Q6.evidence.api.<task> fails forever after a squash-merged done-PR
opened_by: implementer/IMP-061   opened_at: 2026-09-26T18:55Z
evidence: `server/internal/conformance/gates/evidence.go` (`CheckRunIdentity`) required every committed `docs/10_implementation/evidence/*/manifest.json` to have `ci_run_id` whose GitHub-run `head_sha` is an ancestor of the PR HEAD (`git merge-base --is-ancestor`). IMP-000's manifest (squash-merged in `51dae43`) records run 36250265754 with head_sha `9213525474603ecd2f42cb9e596681010884e59f` — the tip of branch `imp/IMP-000-done`, never an ancestor of `main`. Reproduced on PR #12 verify: `FAIL Q6.evidence.api.IMP-000 — run head_sha 9213525... is not an ancestor of HEAD: exit status 128`. The squash-merge rule (ADR-0072) makes the check unsatisfiable by construction: every recorded head_sha ceases to be a main ancestor the moment its PR merges.
owning spec / system: `docs/10_implementation/audit_gates.md` § Q6, `server/internal/conformance/gates/evidence.go`, `server/internal/conformance/gates/q6.go`
options:
  1. accept the merge commit's sha instead of the run's head_sha — resolve the PR(s) containing the recorded sha via `/commits/{sha}/pulls` and require a merged `merge_commit_sha` to be an ancestor of HEAD;
  2. validate the run only (run exists, conclusion success, head_sha matches recorded tree) — drop ancestry entirely;
  3. scope `evidence.api` to the head packet's own manifest only.
blocks: IMP-061
resolved_by: `5d78a49` (PR https://github.com/hung98dev/thth/pull/5)   resolved_at: 2026-09-26T18:43Z   resolution: option 1 — `CheckRunIdentity` falls back to `mergedPRSHAs`/`landedViaMerge` (merge_commit_sha ancestry) when the run head_sha is not an ancestor

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
