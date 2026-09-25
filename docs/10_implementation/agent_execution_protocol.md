# Agent Execution Protocol
status: LOCKED

## Scope

The single operating manual for AI agents: claim, implement, review, merge, block, retry and evidence (ADR-0045, ADR-0050, ADR-0057, ADR-0058). It supplements `AGENTS.md` and does not restate product rules. Roles are defined in `README.md` § Roles. No human takes part except the repository owner for Owner Setup and `OPS-xxx` entries (`audit_gates.md`).

## 1. Separation of Duties

| Role | Does | Never |
|---|---|---|
| `spec-owner` (Contract Owner) | resolves `BLK-xxx`; changes protected specs/ADRs and task packets in spec-change PRs; grep-derived consumer list | implementation code in the same PR |
| `coordinator` | selects, claims and unclaims tasks; keeps concurrency ≤ the concurrency limit (5) | implementation or spec changes |
| implementer | one claimed task inside its `owned_paths`; tests; evidence | edit protected specs/ADRs/control content outside § Protected Paths of `audit_gates.md`; add unpinned dependencies; unrelated refactors |
| `reviewer` (Conformance Reviewer) | reviews every PR in its own session and OS account; posts `policy-review` through the App | review its own work; approve without executed checks |

Self-review is forbidden: the reviewer session is never the implementer session or one of its subagents.

## 2. Definition of Ready

A task is ready when all hold:

```text
[ ] status NOT_STARTED
[ ] every depends_on task is DONE on main
[ ] no open BLK/OPS entry names the task or its specs
[ ] specs:/adrs: listed; every ADR whose Consequences names those specs is read
[ ] owned_paths / forbidden_paths explicit; planned paths exist in repository_layout.md
[ ] contracts, errors and limits are defined; tests are named
[ ] toolchain/dependencies are pinned in technology_versions.md
[ ] recovery behavior is defined when durable state changes
```

If a box fails, the task stays `NOT_STARTED` or becomes `BLOCKED` (§6). Agents never "decide while implementing".

## 3. Claiming (coordinator)

1. Select the lowest topological index (`task_queue.md` § Topological Execution Order) among ready tasks; keep the number of `IN_PROGRESS` tasks ≤ 5 (ADR-0058: 20 concurrent hosted jobs, 2 verify jobs + 1 evidence job per PR).
2. Open a status-only claim PR on branch `claim/<yyyymmdd>-<n>` setting `status: IN_PROGRESS`, `claimed_by`, `branch: imp/IMP-XXX-<slug>`, `claimed_at` in the packet and the summary-row status. Status-only diffs take the Q0-only fast path (`audit_gates.md` § Protected Paths); the reviewer still posts `policy-review`.
3. After the claim merges, hand the task to exactly one implementer (one task per implementer, its own worktree/clone and isolated DB port, Unity cache and temp dirs).
4. A claim with no PR activity for 24 h is returned to `NOT_STARTED` by a new claim PR (clear claim fields).
5. Merge conflicts in `task_queue.md` status cells keep both edits.
6. Bootstrap exception: before `IMP-000` is on `main` no required check exists, so a claim PR cannot merge. `IMP-000` is therefore claimed inside its own PR (first commit sets its claim fields); every later task uses the claim PR above.

## 4. Implementation (implementer)

1. `git fetch && git merge origin/main` (never rebase); create `imp/IMP-XXX-<slug>` from the claim commit; open a **draft PR** immediately.
2. Read `AGENTS.md`, `technology_versions.md`, the packet's specs/ADRs and every ADR whose Consequences names them.
3. Grep `docs/` for every contract symbol you touch; list consumers in the Change Packet.
4. Write only inside `owned_paths`; never in `forbidden_paths`. No commented-out code, `panic("TODO")`, placeholder stubs, fake mocks, or unpinned packages.
5. Create every test named in `## Tests`; cover edge, failure, timeout, restart and boundary cases.
6. Verify locally: `pwsh -NoProfile -File scripts/verify.ps1` (Linux or Windows; Go commands use `go -C server ...`); no generated drift. On a machine without `pwsh` or the pinned Unity editor, `.devin/scripts/verify_delta.sh --full` reports the missing canonical step as `WARN` and defers it to CI, which is authoritative.

## 4a. Change Packet (PR body)

The PR body follows `.github/pull_request_template.md` (owned by `IMP-000`); title `<type>(IMP-XXX): <summary>`.

```text
Task ID / branch
Spec basis: specs, ADRs, and the merged spec-change PR if one preceded this task
Consumer search: query + matching files
Contract delta
Owned paths changed
Tests executed + results
Codegen drift result
Evidence: manifest path, ci_run_id, run_attempt, source_tree_hash
Cleanup verification
```

## 5. Evidence

Canonical schema: `../09_testing/test_and_release_evidence.md`; identity rules: ADR-0057.

- CI (`verify.yml`) checks out the PR head SHA, computes `source_tree_hash`, runs Q0-Q6 in the Linux and Windows jobs, and the `evidence manifest` job merges both reports and uploads the manifest as artifact `evidence`.
- The implementer runs `gh run download <run_id> -n evidence -D docs/10_implementation/evidence/IMP-XXX/` and commits it byte-for-byte. It never edits manifest content.
- Q6 re-verifies only manifests added in the PR: hash equals the head tree hash; `ci_run_id` + `run_attempt` exist, belong to `verify.yml` and concluded `success`.
- FAILED runs are never committed. Chat logs, local runs and screenshots are not evidence; screenshots may be attached as review artifacts referenced by the manifest.

## 5a. Merge Sequence (canonical; other files link here)

```text
1 claim merged (§3)
2 implement on draft PR (§4)
3 reviewer pass: verdict + checked specs as PR review comment
4 CI `Q0-Q6 verify (Linux)` and `Q0-Q6 verify (Windows)` green on head
5 evidence + DONE commit (status: DONE, summary row, evidence/IMP-XXX/manifest.json)
6 CI green again; reviewer re-posts `policy-review` for the new head SHA
7 gh pr ready <N>
8 gh pr merge <N> --auto --squash --delete-branch
```

- Two-phase gate tasks (`IMP-000`, `IMP-061`, `IMP-003`, `IMP-004`, `IMP-005`, `IMP-065`, `IMP-068`): steps 5–6 are skipped; the PR merges with the task `IN_PROGRESS`; a follow-up status PR (branch `imp/IMP-XXX-done`) sets `DONE`, waits for its own `verify.yml` run, downloads that run's `evidence` artifact into `evidence/IMP-XXX/`, pushes, and merges after CI and `policy-review` are green again (ADR-0068). The post-merge guard never produces task evidence.
- Any push after step 3 (including `git merge origin/main`) requires a new reviewer pass and new evidence.
- Never push to `main`, force-push, rebase, or merge manually.

## 5b. Failures and Retries

- CI red: fix inside scope and push. After 3 red runs with the same root cause, set the task `BLOCKED` with a `BLK-xxx` (contract) or `OPS-xxx` (environment) entry and stop.
- Infrastructure-classified failure (hosted runner unavailable, Unity licence activation, image pull, disk, network): `gh run rerun --failed` once, citing the log line; if it fails again, open `OPS-xxx`, label a GitHub issue `ops-blocked`, stop.
- Timeouts: whole job 120 min, Unity step 30 min; a timeout is an infrastructure failure.
- Flaky test: never retried into green or deleted; quarantine only through an ADR already on `main` (gate ratchet) and a `BLK-xxx` for the fix.
- Post-merge revert of your squash commit returns the task to `IN_PROGRESS`; restart at §4 on a new branch.
- `AUTO_MERGE_FROZEN=true`: stop all merges until the owner clears the `OPS-xxx` entry.

## 6. Blocked Decisions

1. Never guess (unpinned library, Redis, invented config or value).
2. Append a `BLK-xxx` entry to `known_blockers.md` (conflict, evidence file:line, owning spec, options), set the task `BLOCKED` with `blocked_by: BLK-xxx`, push, and stop.
3. The `spec-owner` resolves it in a spec-change PR (`spec/BLK-xxx-<slug>`): updates the owning spec/ADR and all consumers, adds the regression test to the unblocked packet's `## Tests`, moves the entry to Resolved, and returns the task to `NOT_STARTED`.
4. Behavior changes land in the spec first; the task PR cites the merged spec-change PR.
5. Every measurable requirement the `spec-owner` writes gets a requirement ID (`[A-Z]{2,6}-\d{3}`) in the spec's "Requirement IDs" table and is added to the owning packet's `## Acceptance` and `## Tests` in the same spec-change PR (Q0 requirement coverage).

## 7. Invariants

```text
explicit DoR before every claim; only the coordinator claims
one implementer = one task; no self-review
>2 numbered directories => complete consumer list + policy-review
protected specs change only in spec-owner spec-change PRs
no rebase, no force-push, no direct push; update by merge origin/main; squash into main
DONE only with ADR-0057 evidence from CI
3 red runs per root cause => BLOCKED
verify fail = not DONE
```
