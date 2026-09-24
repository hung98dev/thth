# Audit Gates A-D and Verify Gates Q0-Q6
status: LOCKED

## Scope

Owner Setup, bootstrap mode, integration barriers and protected paths (ADR-0045, ADR-0050, ADR-0057). Foundation tasks build the verifier and its fixtures; tasks that depend on `IMP-068` start only after `IMP-068` is `DONE`.

Passing local `go test` is not Gate D and is not task evidence.

## Owner Setup (one-time, before IMP-000)

The repository owner is the only human in the process. Before `IMP-000` the owner provisions, and afterwards only repairs `OPS-xxx` entries:

```text
repository         GitHub repo with docs/ pushed to main; forking disabled; allow auto-merge; delete branch on merge
runner             cloud Windows VM registered as self-hosted runner, labels [self-hosted, windows, thinhthan],
                   runner slots declared (default 2); GPU at DESKTOP_MIN level or better (Intel UHD 620 class);
                   installed: Unity 6000.6.1f1 + active licence (UNITY_EDITOR_PATH), Go 1.27.1, Git for Windows, gh, jq, gcloud
                   (no GPU => the client performance gate is an OPS-xxx blocker)
android devices    Google Cloud project on the free Firebase Spark plan with Test Lab enabled; one ANDROID_MIN-class and
                   one ANDROID_REC-class physical device model recorded here; service-account key stored as a runner
                   secret usable only by CI. Android performance = Unity game-loop tests via
                   `gcloud firebase test android run --type game-loop` in the scheduled `device-perf` workflow
                   (../04_architecture/client_performance.md § Measurement and Gates)
ruleset on main    PR required; required checks `Q0-Q6 verify (Windows)` (workflow verify.yml from main) and
                   `policy-review` (source = App thinhthan-policy-reviewer); branches must be up to date;
                   approvals 0; no bypass actors; force-push and deletion blocked
App                thinhthan-policy-reviewer installed on the repo (checks:write, pull_requests:write, contents:read);
                   private key readable only by the reviewer OS account, never by the runner service account
secrets/vars       REVIEWER_APP_ID, REVIEWER_APP_KEY (reviewer account only), GCP_TEST_LAB_SA_KEY (CI only), AUTO_MERGE_FROZEN=false,
                   GitHub token for guard workflow via actions/create-github-app-token
agent tokens       fine-grained, contents + pull requests write only; no administration scope
```

`IMP-068` only reads and evidences this setup (`gh api repos/{o}/{r}`, `.../rulesets/{id}`, App installation JSON committed to `evidence/IMP-068/`). Agents never change repository settings.

## Bootstrap Mode (until IMP-068 is DONE)

- `IMP-000` is the first pull request; no other PR merges before it.
- The required job runs the verifier from the PR head; the gate ratchet is treated as empty.
- A Q gate becomes required when its owner task is `DONE` on `main`: Q0/Q1/Q3-Go/Q4/Q6 after `IMP-000`; Q2 after `IMP-061`; Q5 after `IMP-005`, `IMP-003`, `IMP-004`; Unity EditMode after `IMP-000`; Unity PlayMode after `IMP-065`; client performance after its owning task. Before that the gate reports `SKIP(bootstrap)`, which is not a failure.
- A task may run before `IMP-068` iff `IMP-068` is not in its transitive `depends_on`.
- Two-phase gate tasks (`IMP-000`, `IMP-061`, `IMP-003`, `IMP-004`, `IMP-005`, `IMP-065`, `IMP-068`): the implementation PR merges with the task `IN_PROGRESS`; a follow-up status PR sets `DONE` citing the post-merge `main` run.

## Gate A — Contract Coherence

Met only when:

- no open `BLK-xxx` entry in `known_blockers.md` (`OPS-xxx` entries belong to Gate D);
- `AGENTS.md`, accepted ADRs, architecture/operations specs, and task packets describe one launch topology (one world, one process, one database);
- no production role split, Redis/Kafka/NATS authority, `global_leader_lease`, account item vault, or equipment durability exists;
- every launch constant/ID/message/formula/schema field has one canonical owner;
- every canonical task reference resolves to a packet;
- contract edits list the complete grep-derived `consumers_checked` set.

Open contract conflict means Gate A fails closed. An implementation task may not pick one side.

## Gate B — Task and Reproducibility Integrity

Met only when:

- Q0 validates the task DAG, paths, states, transitions, claim fields and evidence rules;
- requirement coverage: every requirement ID (pattern `[A-Z]{2,6}-\d{3}`, listed in a spec's "Requirement IDs" table) in `docs/00_context`..`docs/09_testing` is named in at least one packet's `## Acceptance` and the same packet's `## Tests`;
- every `depends_on` target exists and the graph is acyclic;
- `owned_paths` does not intersect `forbidden_paths`; overlapping ownership is ordered by `depends_on`;
- a task is `IN_PROGRESS` or `DONE` only if every dependency is `DONE`;
- implementer PRs change control files only as allowed (§ Protected Paths);
- exact toolchain, package and GitHub Action pins equal `../00_context/technology_versions.md`;
- `scripts/verify.ps1` / `scripts/codegen.ps1` invoke the Go verifier/codegen and leave no generated drift.

## Gate C — Executable Conformance

Every required Q gate executes; `SKIP(bootstrap)` is allowed only under Bootstrap Mode. Otherwise a tool missing on the runner is an `OPS-xxx` failure, never a silent skip.

- Q1 toolchain/dependency/action pins;
- Q2 protobuf Go/C# regenerated into a temp directory and byte-compared;
- Q3 Go unit + `-race` for `sim|edge|durable|global` + Unity EditMode; Unity PlayMode after `IMP-065`; desktop client performance tests on the runner GPU (`../04_architecture/client_performance.md`) after their owning task (Android device runs are the scheduled `device-perf` workflow, not Q3);
- Q4 architecture/import/ownership fences;
- Q5 migrations apply/down/apply on PostgreSQL 18.6 (Windows binaries started by `verify.ps1`, DSN in `THINHTHAN_TEST_PG_DSN`) and full content compile/activation;
- Q6 clean tree and evidence identity (ADR-0057);
- mutation fixtures prove every barrier fails closed.

An allowed non-bootstrap skip must be named by the owning test/evidence contract and recorded in `skipped_reasons`.

## Gate D — Trusted Integration

Met only when:

- Owner Setup is evidenced by `IMP-068`;
- `verify.yml` runs on `pull_request_target` from `main`, checks out the PR head SHA into a separate directory, builds the verifier from `main` and runs it on the head (trusted judge; a PR cannot change its own judge);
- all Actions are SHA-pinned per the technology matrix; job timeout 120 min, Unity step 30 min;
- `policy-review` is posted only by the App for the head SHA on every PR, after the reviewer session reviews `git diff origin/main...HEAD`; it is re-posted after every push; no workflow job has that name;
- the gate ratchet is derived on the base branch from the verifier gate list plus tests named in `DONE` packets; a decrease is accepted only when an ADR referencing it already exists on `main`;
- the post-merge guard runs full verify on every push to `main` (one concurrency group); a non-infrastructure failure makes the App open `revert/<sha>` for the first failing squash commit, which also sets affected dependents `BLOCKED` (`blocked_by: REVERT-<sha>`); a revert commit or infrastructure failure is never auto-reverted: `AUTO_MERGE_FROZEN=true` and an `OPS-xxx` entry is opened;
- evidence follows ADR-0057 (source-tree hash, CI artifact, API-verified `ci_run_id` + `run_attempt`);
- Android device performance is not a PR check: the scheduled `device-perf` workflow on `main` runs at most once per day (only when client code/assets changed) plus once for the launch candidate; an exhausted Test Lab quota reports `DEFERRED(quota)` and retries the next day, never blocks PRs and never opens `OPS-xxx`; the launch-candidate gate waits for a passing run.

## Protected Paths

PRs touching these need the protected-path checklist in the reviewer's `policy-review`:

```text
.github/  scripts/  .devin/**
server/cmd/verify/  server/internal/conformance/  server/internal/stackpin/  server/internal/conformance/architecture/
AGENTS.md  README.md
docs/** outside docs/10_implementation/        (specs, ADRs, templates — spec-owner only)
docs/10_implementation/*.md                    (control files)
```

Implementer PRs may change control content only as follows: their own packet `status`/`claimed_by`/`branch`/`claimed_at`/`blocked_by` fields and summary-row status cell; appending `known_blockers.md` entries; adding `evidence/<own ID>/`. Q0 rejects any other control-file change unless the PR author role is `spec-owner` or `coordinator` (claims only).

## Q0-Q6 Contract

| Gate | Owner | Mandatory result |
|---|---|---|
| Q0 Task/spec integrity | IMP-000, IMP-068 | DAG, links, states, transitions, claim fields, control-file diff rules, requirement-ID coverage, evidence schema |
| Q1 Version reproducibility | IMP-000 | exact native pins; no floating/unlisted dependency |
| Q2 Code generation drift | IMP-061 | pinned protoc generators; byte-identical Go/C# output |
| Q3 Test suites | subsystem task, IMP-068 | Go/race, Unity EditMode/PlayMode, client performance, deterministic fixtures |
| Q4 Architecture conformance | IMP-068 | import fences, one production main, generated boundaries, schema prohibitions |
| Q5 Data/content integrity | IMP-003, IMP-004, IMP-005, IMP-068 | migration rehearsal, schema drift, content compile/activation |
| Q6 Evidence/cleanliness | IMP-068 | clean generated state, evidence identity per ADR-0057 |

## Foundation Exit

`IMP-068` may become `DONE` only when Gates A-D and Q0-Q6 pass. If a later change breaks a gate, the post-merge guard reverts it.

## Invariants

```text
open BLK => Gate A fails;  open OPS => Gate D fails
bootstrap SKIP only before the gate's owner task is DONE
local green run != Gate D
PR judged by the verifier built from main (after IMP-068)
policy-review = App status on every PR, re-posted per push
gate ratchet only tightens without an ADR already on main
red main => automatic revert, except reverts/infra => freeze + OPS
CI = Windows only; Q0-Q6 fail-closed
```
